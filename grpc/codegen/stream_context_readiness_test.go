package codegen

import (
	"go/format"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
)

func TestGRPCMessageContextGenerated(t *testing.T) {
	root := RunGRPCDSL(t, grpcMessageContextDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_GRPC_CONTEXT_ARTIFACTS"); output != "" {
		dir = output
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	renderGRPCModule(t, dir, "example.com/messagecontext", root, resolveGRPCLoomSource(t))
	harness, err := format.Source([]byte(grpcMessageContextHarness))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "context_test.go"), harness, 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "-race", "-count=1", "-v", ".")
}

func grpcMessageContextDSL() {
	message := dsl.Type("Message", func() {
		dsl.Field(1, "text", dsl.String)
		dsl.Field(2, "data", dsl.Bytes)
		dsl.Required("text")
	})
	dsl.Service("messages", func() {
		dsl.Method("chat", func() {
			dsl.StreamingPayload(message)
			dsl.StreamingResult(message)
			dsl.GRPC(func() {
			})
		})
		dsl.Method("upload", func() {
			dsl.StreamingPayload(message)
			dsl.Result(message)
			dsl.GRPC(func() {
			})
		})
	})
}

const grpcMessageContextHarness = `package messagecontext_test

import (
	"context"
	"errors"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"example.com/messagecontext/gen/grpc/messages/client"
	pb "example.com/messagecontext/gen/grpc/messages/pb"
	"example.com/messagecontext/gen/grpc/messages/server"
	"example.com/messagecontext/gen/messages"
)

type (
	service struct {
		recovered chan any
		chat      func(context.Context, messages.ChatServerStream) error
		upload    func(context.Context, messages.UploadServerStream) error
	}
	countedClientStream struct {
		grpc.ClientStream
		writes *atomic.Int32
	}
	countedServerStream struct {
		grpc.ServerStream
		writes *atomic.Int32
	}
)

func (s *service) Chat(ctx context.Context, stream messages.ChatServerStream) error {
	return s.chat(ctx, stream)
}
func (s *service) Upload(ctx context.Context, stream messages.UploadServerStream) error {
	return s.upload(ctx, stream)
}
func (s *countedServerStream) SendMsg(value any) error {
	s.writes.Add(1)
	return s.ServerStream.SendMsg(value)
}
func (s *countedClientStream) SendMsg(value any) error {
	s.writes.Add(1)
	return s.ClientStream.SendMsg(value)
}
func start(t *testing.T, svc *service) (*client.Client, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	writes := new(atomic.Int32)
	clientWrites := new(atomic.Int32)
	listener := bufconn.Listen(1 << 20)
	srv := grpc.NewServer(grpc.StreamInterceptor(func(s any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, next grpc.StreamHandler) (err error) {
		defer func() {
			if value := recover(); value != nil {
				if svc.recovered == nil {
					panic(value)
				}
				svc.recovered <- value
				err = status.Error(codes.Internal, "recovered service panic")
			}
		}()
		return next(s, &countedServerStream{ServerStream: stream, writes: writes})
	}))
	pb.RegisterMessagesServer(srv, server.New(messages.NewEndpoints(svc), nil))
	served := make(chan error, 1)
	go func() {
		served <- srv.Serve(listener)
	}()
	t.Cleanup(func() {
		srv.Stop()
		require.NoError(t, <-served)
	})
	conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithStreamInterceptor(func(ctx context.Context, description *grpc.StreamDesc, connection *grpc.ClientConn, method string, next grpc.Streamer, options ...grpc.CallOption) (grpc.ClientStream, error) {
		stream, err := next(ctx, description, connection, method, options...)
		if err != nil {
			return nil, err
		}
		return &countedClientStream{ClientStream: stream, writes: clientWrites}, nil
	}))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, conn.Close())
	})
	return client.NewClient(conn), writes, clientWrites
}

func canceled() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func TestAlreadyCanceledServerSend(t *testing.T) {
	for _, final := range []bool{false, true} {
		t.Run(map[bool]string{false: "Send", true: "SendAndClose"}[final], func(t *testing.T) {
			result := make(chan error, 1)
			svc := &service{
				chat: func(_ context.Context, stream messages.ChatServerStream) error {
					err := stream.SendWithContext(canceled(), &messages.Message{Text: "must-not-send"})
					result <- err
					return err
				},
				upload: func(_ context.Context, stream messages.UploadServerStream) error {
					err := stream.SendAndCloseWithContext(canceled(), &messages.Message{Text: "must-not-send"})
					result <- err
					return err
				},
			}
			c, writes, _ := start(t, svc)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if final {
				_, err := c.Upload()(ctx, nil)
				require.NoError(t, err)
			} else {
				_, err := c.Chat()(ctx, nil)
				require.NoError(t, err)
			}
			select {
			case err := <-result:
				if !errors.Is(err, context.Canceled) {
					t.Errorf("operation error=%v, want context.Canceled", err)
				}
			case <-ctx.Done():
				t.Fatal("service did not run")
			}
			if got := writes.Load(); got != 0 {
				t.Errorf("pre-canceled operation performed %d native writes", got)
			}
		})
	}
}

func TestAlreadyCanceledClientSend(t *testing.T) {
	received := make(chan *messages.Message, 1)
	c, _, writes := start(t, &service{chat: func(ctx context.Context, stream messages.ChatServerStream) error {
		value, err := stream.RecvWithContext(ctx)
		if err == nil {
			received <- value
		}
		return err
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	value, err := c.Chat()(ctx, nil)
	require.NoError(t, err)
	stream := value.(*client.ChatClientStream)
	err = stream.SendWithContext(canceled(), &messages.Message{Text: "must-not-send"})
	if got := writes.Load(); got != 0 {
		t.Errorf("pre-canceled operation performed %d native writes", got)
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("operation error=%v, want context.Canceled", err)
		select {
		case got := <-received:
			t.Errorf("pre-canceled send reached service: %+v", got)
		case <-ctx.Done():
		}
	}
}

func TestBlockedReceiveContext(t *testing.T) {
	for _, side := range []string{"server", "client", "client-final"} {
		t.Run(side, func(t *testing.T) {
			result := make(chan error, 1)
			started := make(chan struct{})
			svc := &service{
				chat: func(ctx context.Context, stream messages.ChatServerStream) error {
					if side != "server" {
						close(started)
						<-ctx.Done()
						return ctx.Err()
					}
					operation, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
					defer cancel()
					close(started)
					_, err := stream.RecvWithContext(operation)
					result <- err
					return err
				},
				upload: func(ctx context.Context, _ messages.UploadServerStream) error {
					close(started)
					<-ctx.Done()
					return ctx.Err()
				},
			}
			c, _, _ := start(t, svc)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			var value any
			var err error
			if side == "client-final" {
				value, err = c.Upload()(ctx, nil)
			} else {
				value, err = c.Chat()(ctx, nil)
			}
			require.NoError(t, err)
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("service did not start")
			}
			if side != "server" {
				go func() {
					operation, stop := context.WithTimeout(ctx, 100*time.Millisecond)
					defer stop()
					var err error
					if side == "client-final" {
						_, err = value.(*client.UploadClientStream).CloseAndRecvWithContext(operation)
					} else {
						_, err = value.(*client.ChatClientStream).RecvWithContext(operation)
					}
					result <- err
				}()
			}
			select {
			case err := <-result:
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("operation error=%v, want context.DeadlineExceeded", err)
				}
			case <-time.After(600 * time.Millisecond):
				t.Error("100ms operation deadline did not release blocked receive within 600ms")
				cancel()
				select {
				case <-result:
				case <-time.After(time.Second):
					t.Fatal("request cancellation did not release test worker")
				}
			}
		})
	}
}

func TestDuplexHalfCloseControl(t *testing.T) {
	c, _, _ := start(t, &service{chat: func(ctx context.Context, stream messages.ChatServerStream) error {
		value, err := stream.RecvWithContext(ctx)
		if err != nil {
			return err
		}
		if _, err := stream.RecvWithContext(ctx); !errors.Is(err, io.EOF) {
			return err
		}
		return stream.SendWithContext(ctx, value)
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	value, err := c.Chat()(ctx, nil)
	require.NoError(t, err)
	stream := value.(*client.ChatClientStream)
	require.NoError(t, stream.SendWithContext(ctx, &messages.Message{Text: "after-half-close"}))
	require.NoError(t, stream.Close())
	got, err := stream.RecvWithContext(ctx)
	require.NoError(t, err)
	require.Equal(t, "after-half-close", got.Text)
	_, err = stream.RecvWithContext(ctx)
	require.ErrorIs(t, err, io.EOF)
}

func TestCompletedOperationContextDoesNotCancelNext(t *testing.T) {
	c, _, _ := start(t, &service{chat: func(ctx context.Context, stream messages.ChatServerStream) error {
		first, stop := context.WithCancel(ctx)
		defer stop()
		value, err := stream.RecvWithContext(first)
		if err != nil {
			return err
		}
		stop()
		if err := stream.SendWithContext(ctx, value); err != nil {
			return err
		}
		value, err = stream.RecvWithContext(ctx)
		if err != nil {
			return err
		}
		return stream.SendWithContext(ctx, value)
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	value, err := c.Chat()(ctx, nil)
	require.NoError(t, err)
	stream := value.(*client.ChatClientStream)
	first, stop := context.WithCancel(ctx)
	defer stop()
	require.NoError(t, stream.SendWithContext(first, &messages.Message{Text: "first"}))
	stop()
	require.NoError(t, stream.SendWithContext(ctx, &messages.Message{Text: "second"}))
	for _, want := range []string{"first", "second"} {
		got, err := stream.RecvWithContext(ctx)
		require.NoError(t, err)
		require.Equal(t, want, got.Text)
	}
	_, err = stream.RecvWithContext(ctx)
	require.ErrorIs(t, err, io.EOF)
}

func TestBlockedServerSendContextAndNilServiceReturn(t *testing.T) {
	result := make(chan error, 1)
	c, _, _ := start(t, &service{chat: func(ctx context.Context, stream messages.ChatServerStream) error {
		operation, stop := context.WithTimeout(ctx, 100*time.Millisecond)
		defer stop()
		message := &messages.Message{Text: "flow-control", Data: make([]byte, 1<<20)}
		for range 32 {
			if err := stream.SendWithContext(operation, message); err != nil {
				result <- err
				return nil
			}
		}
		result <- nil
		return nil
	}})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	value, err := c.Chat()(ctx, nil)
	require.NoError(t, err)
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(600 * time.Millisecond):
		t.Fatal("flow-controlled send ignored operation deadline")
	}
	stream := value.(*client.ChatClientStream)
	for {
		_, err = stream.RecvWithContext(ctx)
		if err != nil {
			break
		}
	}
	require.Equal(t, codes.DeadlineExceeded, status.Code(err), "nil service return must not turn aborted RPC into clean EOF")
}

func TestServiceRecoveryPreserved(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-operation", true: "after-cancel"}[after], func(t *testing.T) {
			recovered := make(chan any, 1)
			c, _, _ := start(t, &service{recovered: recovered, chat: func(ctx context.Context, stream messages.ChatServerStream) error {
				if after {
					operation, stop := context.WithTimeout(ctx, 50*time.Millisecond)
					defer stop()
					_, err := stream.RecvWithContext(operation)
					if !errors.Is(err, context.DeadlineExceeded) {
						return err
					}
				}
				panic("service panic")
			}})
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			value, err := c.Chat()(ctx, nil)
			require.NoError(t, err)
			_, err = value.(*client.ChatClientStream).RecvWithContext(ctx)
			require.Equal(t, codes.Internal, status.Code(err))
			select {
			case value := <-recovered:
				require.Equal(t, "service panic", value)
			case <-ctx.Done():
				t.Fatal("service panic did not reach outer recovery")
			}
		})
	}
}
`
