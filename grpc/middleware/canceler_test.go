package middleware_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	grpcm "github.com/CaliLuke/loom/grpc/middleware"
)

type (
	testCancelerStream struct {
		grpc.ServerStream
	}

	cancelDuringContextStream struct {
		grpc.ServerStream

		cancelShutdown context.CancelFunc
		shutdownDone   <-chan struct{}
		once           sync.Once
	}
)

func (s *cancelDuringContextStream) Context() context.Context {
	s.once.Do(func() {
		s.cancelShutdown()
		<-s.shutdownDone
		time.Sleep(10 * time.Millisecond)
	})
	return context.Background()
}

func TestStreamCanceler(t *testing.T) {
	var (
		stream = &grpc.StreamServerInfo{
			FullMethod: "Test.Test",
		}
	)
	cases := []struct {
		name        string
		stream      grpc.ServerStream
		handlerFunc func(wg *sync.WaitGroup) grpc.StreamHandler
	}{
		{
			name:   "handler canceled",
			stream: grpcm.NewWrappedServerStream(context.Background(), &testCancelerStream{}),
			handlerFunc: func(wg *sync.WaitGroup) grpc.StreamHandler {
				return func(srv any, stream grpc.ServerStream) error {
					wg.Done()
					<-stream.Context().Done() // block until canceled
					return nil
				}
			},
		},
		{
			name:   "handler not canceled",
			stream: grpcm.NewWrappedServerStream(context.Background(), &testCancelerStream{}),
			handlerFunc: func(wg *sync.WaitGroup) grpc.StreamHandler {
				return func(srv any, stream grpc.ServerStream) error {
					wg.Done()
					// don't block, finish before being canceled
					return nil
				}
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			interceptor := grpcm.StreamCanceler(ctx)
			var wg sync.WaitGroup
			wg.Add(1)
			go func(t *testing.T) {
				assert.NoError(t, interceptor(nil, c.stream, stream, c.handlerFunc(&wg)))
			}(t)
			wg.Wait()
			cancel()
		})
	}
}

func TestStreamCancelerCancelsStreamAdmittedDuringShutdown(t *testing.T) {
	shutdownCtx, cancelShutdown := context.WithCancel(context.Background())
	interceptor := grpcm.StreamCanceler(shutdownCtx)
	stream := &grpc.StreamServerInfo{FullMethod: "Test.Test"}
	done := make(chan error, 1)

	go func() {
		done <- interceptor(nil, &cancelDuringContextStream{
			cancelShutdown: cancelShutdown,
			shutdownDone:   shutdownCtx.Done(),
		}, stream, func(srv any, stream grpc.ServerStream) error {
			<-stream.Context().Done()
			return nil
		})
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("stream admitted during shutdown was not canceled")
	}
}
