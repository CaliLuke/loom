package grpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	grpcnative "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// These probes establish the native ownership constraints used by #726's model.
// They deliberately bypass generated adapters and the proposed Loom owner.
func TestNativeBlockedServerReceiveLifetime(t *testing.T) {
	for _, terminal := range []string{"handler-return", "client-cancel"} {
		t.Run(terminal, func(t *testing.T) {
			entered := make(chan struct{})
			derivedCanceled := make(chan struct{})
			received := make(chan error, 1)
			returnHandler := make(chan struct{})
			_, cancel := nativeContextProbe(t, func(_ any, server grpcnative.ServerStream) error {
				derived, stop := context.WithCancel(server.Context())
				go func() {
					close(entered)
					received <- server.RecvMsg(new(emptypb.Empty))
				}()
				stop()
				<-derived.Done()
				close(derivedCanceled)
				select {
				case <-returnHandler:
				case <-server.Context().Done():
				}
				return nil
			})
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("native handler was not dispatched")
			}
			<-derivedCanceled
			select {
			case err := <-received:
				t.Fatalf("derived cancellation unexpectedly released native receive: %v", err)
			case <-time.After(50 * time.Millisecond):
			}
			if terminal == "handler-return" {
				close(returnHandler)
			} else {
				cancel()
			}
			select {
			case err := <-received:
				require.Error(t, err)
			case <-time.After(time.Second):
				t.Fatal("native receive survived its transport termination")
			}
		})
	}
}

func TestNativeClientReceiveOpeningContext(t *testing.T) {
	entered := make(chan struct{})
	stream, cancel := nativeContextProbe(t, func(_ any, server grpcnative.ServerStream) error {
		close(entered)
		<-server.Context().Done()
		return server.Context().Err()
	})
	<-entered
	received := make(chan error, 1)
	go func() {
		received <- stream.RecvMsg(new(emptypb.Empty))
	}()
	cancel()
	select {
	case err := <-received:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("opening context cancellation did not release client receive")
	}
}

func TestNativeBlockedServerSendLifetime(t *testing.T) {
	started := make(chan int, 32)
	sent := make(chan error, 1)
	returnHandler := make(chan struct{})
	_, cancel := nativeContextProbe(t, func(_ any, server grpcnative.ServerStream) error {
		go func() {
			message := wrapperspb.Bytes(make([]byte, 1<<20))
			for index := range 32 {
				started <- index
				if err := server.SendMsg(message); err != nil {
					sent <- err
					return
				}
			}
			sent <- nil
		}()
		select {
		case <-returnHandler:
		case <-server.Context().Done():
		}
		return nil
	})
	defer cancel()
	// Native SendMsg may queue its first message before flow control blocks the
	// following write. The client consumes no messages and windows stay bounded.
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("native send did not start")
	}
	select {
	case err := <-sent:
		t.Fatalf("flow-controlled stream unexpectedly completed before handler return: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(returnHandler)
	select {
	case err := <-sent:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("native send survived handler return")
	}
}

func TestNativeServerRecoveryAffinity(t *testing.T) {
	recovered := make(chan any, 1)
	stream, cancel := nativeContextProbe(t, func(any, grpcnative.ServerStream) error {
		panic("service panic")
	}, grpcnative.StreamInterceptor(func(service any, stream grpcnative.ServerStream, _ *grpcnative.StreamServerInfo, next grpcnative.StreamHandler) (err error) {
		defer func() {
			if value := recover(); value != nil {
				recovered <- value
				err = status.Error(codes.Internal, "recovered service panic")
			}
		}()
		return next(service, stream)
	}))
	defer cancel()
	err := stream.RecvMsg(new(emptypb.Empty))
	require.Equal(t, codes.Internal, status.Code(err))
	select {
	case value := <-recovered:
		require.Equal(t, "service panic", value)
	case <-time.After(time.Second):
		t.Fatal("outer stream recovery did not observe the service panic")
	}
}

func TestNativeOperationWorkerPreservesHandlerRecovery(t *testing.T) {
	nativeDone := make(chan error, 1)
	recovered := make(chan any, 1)
	stream, cancel := nativeContextProbe(t, func(_ any, server grpcnative.ServerStream) error {
		go func() {
			nativeDone <- server.RecvMsg(new(emptypb.Empty))
		}()
		ctx, stop := context.WithTimeout(server.Context(), 50*time.Millisecond)
		defer stop()
		select {
		case <-ctx.Done():
			// Application execution stays on the original handler stack even
			// after the contextual operation returns before native I/O completes.
			panic("service panic after operation cancellation")
		case err := <-nativeDone:
			return err
		}
	}, grpcnative.StreamInterceptor(func(service any, stream grpcnative.ServerStream, _ *grpcnative.StreamServerInfo, next grpcnative.StreamHandler) (err error) {
		defer func() {
			if value := recover(); value != nil {
				recovered <- value
				err = status.Error(codes.Internal, "recovered service panic")
			}
		}()
		return next(service, stream)
	}))
	defer cancel()
	err := stream.RecvMsg(new(emptypb.Empty))
	require.Equal(t, codes.Internal, status.Code(err))
	select {
	case value := <-recovered:
		require.Equal(t, "service panic after operation cancellation", value)
	case <-time.After(time.Second):
		t.Fatal("service panic escaped outer recovery")
	}
	select {
	case err := <-nativeDone:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("tracked native receive survived handler return")
	}
}

func nativeContextProbe(t *testing.T, handler grpcnative.StreamHandler, options ...grpcnative.ServerOption) (grpcnative.ClientStream, context.CancelFunc) {
	t.Helper()
	listener := bufconn.Listen(1 << 20)
	options = append(options, grpcnative.StaticStreamWindowSize(65535), grpcnative.StaticConnWindowSize(65535))
	server := grpcnative.NewServer(options...)
	server.RegisterService(&grpcnative.ServiceDesc{
		ServiceName: "contextprobe.Service",
		HandlerType: (*any)(nil),
		Streams: []grpcnative.StreamDesc{{
			StreamName: "Exchange", Handler: handler, ServerStreams: true, ClientStreams: true,
		}},
	}, nil)
	served := make(chan error, 1)
	go func() {
		served <- server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		require.NoError(t, <-served)
	})
	conn, err := grpcnative.NewClient("passthrough:///contextprobe", grpcnative.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpcnative.WithTransportCredentials(insecure.NewCredentials()), grpcnative.WithStaticStreamWindowSize(65535), grpcnative.WithStaticConnWindowSize(65535))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	t.Cleanup(cancel)
	stream, err := conn.NewStream(ctx, &grpcnative.StreamDesc{ServerStreams: true, ClientStreams: true}, "/contextprobe.Service/Exchange")
	require.NoError(t, err)
	return stream, cancel
}
