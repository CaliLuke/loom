package debug

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/CaliLuke/loom/clue/log"
)

type optionTestStream struct {
	grpc.ServerStream
	ctx context.Context
}

func TestDebugTransportOptionsStayInRequest(t *testing.T) {
	previous := debugLogs.Load()
	t.Cleanup(func() {
		debugLogs.Store(previous)
	})
	for _, tc := range []struct {
		name   string
		invoke func(*testing.T, context.Context) context.Context
	}{
		{"HTTP", invokeOptionHTTP},
		{"unary gRPC", invokeOptionUnary},
		{"streaming gRPC", invokeOptionStream},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := log.Context(context.Background(), log.WithOutputs(log.Output{Writer: io.Discard, Format: log.FormatText}))
			debugLogs.Store(true)
			first := tc.invoke(t, root)
			require.True(t, log.DebugEnabled(first))
			debugLogs.Store(false)
			second := tc.invoke(t, root)
			require.False(t, log.DebugEnabled(second))
			require.True(t, log.DebugEnabled(first), "later request must not disable an existing request or stream")
			require.False(t, log.DebugEnabled(root), "request option must not mutate root configuration")
		})
	}
}

// Context returns the stream request context.
func (s *optionTestStream) Context() context.Context {
	return s.ctx
}

func invokeOptionHTTP(t *testing.T, root context.Context) context.Context {
	t.Helper()
	var captured context.Context
	handler := log.HTTP(root, log.WithDisableRequestLogging())(HTTP()(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		captured = r.Context()
	})))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	return captured
}

func invokeOptionUnary(t *testing.T, root context.Context) context.Context {
	t.Helper()
	var captured context.Context
	info := &grpc.UnaryServerInfo{FullMethod: "/Service/Method"}
	_, err := log.UnaryServerInterceptor(root, log.WithDisableCallLogging())(context.Background(), nil, info, func(ctx context.Context, request any) (any, error) {
		return UnaryServerInterceptor()(ctx, request, info, func(ctx context.Context, _ any) (any, error) {
			captured = ctx
			return nil, nil
		})
	})
	require.NoError(t, err)
	return captured
}

func invokeOptionStream(t *testing.T, root context.Context) context.Context {
	t.Helper()
	var captured context.Context
	info := &grpc.StreamServerInfo{FullMethod: "/Service/Method"}
	stream := &optionTestStream{ctx: context.Background()}
	err := log.StreamServerInterceptor(root, log.WithDisableCallLogging())(nil, stream, info, func(service any, stream grpc.ServerStream) error {
		return StreamServerInterceptor()(service, stream, info, func(_ any, stream grpc.ServerStream) error {
			captured = stream.Context()
			return nil
		})
	})
	require.NoError(t, err)
	return captured
}
