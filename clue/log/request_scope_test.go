package log

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type requestScopeKey struct {
}

func TestRequestLoggerScope(t *testing.T) {
	for _, protocol := range []string{"HTTP", "unary", "streaming"} {
		for _, ids := range []bool{false, true} {
			for _, logging := range []bool{false, true} {
				for _, rootState := range []string{"empty", "buffered", "flushed"} {
					t.Run(fmt.Sprintf("%s/ids=%t/logging=%t/root=%s", protocol, ids, logging, rootState), func(t *testing.T) {
						var output bytes.Buffer
						root := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
						root = With(root, KV{K: "root_field", V: "retained"})
						if rootState != "empty" {
							Info(root, KV{K: "msg", V: "root entry"})
						}
						if rootState == "flushed" {
							FlushAndDisableBuffering(root)
							output.Reset()
						}
						var first, second context.Context
						invokeRequestScope(t, protocol, root, context.Background(), ids, logging, func(ctx context.Context) {
							first = ctx
							require.Equal(t, "request value", ctx.Value(requestScopeKey{}))
							Info(ctx, KV{K: "msg", V: "first entry"})
						})
						require.NotContains(t, output.String(), "first entry", "root flush state must not disable request buffering")
						invokeRequestScope(t, protocol, root, context.Background(), ids, logging, func(ctx context.Context) {
							second = ctx
							Info(ctx, KV{K: "msg", V: "second entry"})
							Error(ctx, nil, KV{K: "msg", V: "second failure"})
						})
						require.NotSame(t, first.Value(ctxLogger), second.Value(ctxLogger))
						require.NotSame(t, root.Value(ctxLogger), second.Value(ctxLogger))
						require.NotContains(t, output.String(), "first entry")
						require.NotContains(t, output.String(), "root entry")
						require.Contains(t, output.String(), "second entry")
						require.Contains(t, output.String(), "second failure")
						require.Contains(t, output.String(), "root_field=retained")
						if ids {
							require.Contains(t, output.String(), RequestIDKey+"=")
						} else {
							require.NotContains(t, output.String(), RequestIDKey+"=")
						}
						output.Reset()
						FlushAndDisableBuffering(first)
						require.Contains(t, output.String(), "first entry")
						require.NotContains(t, output.String(), "second entry")
						require.NotContains(t, output.String(), "root entry")
					})
				}
			}
		}
	}
}

func TestRequestScopePreservesBufferPolicyAndContext(t *testing.T) {
	for _, protocol := range []string{"HTTP", "unary", "streaming"} {
		for _, tc := range []struct {
			name   string
			option LogOption
		}{
			{"debug", WithDebug()},
			{"request policy", WithDisableBuffering(func(ctx context.Context) bool {
				return ctx.Value(requestScopeKey{}) == "request value"
			})},
		} {
			t.Run(protocol+"/"+tc.name, func(t *testing.T) {
				var output bytes.Buffer
				root := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}), tc.option)
				deadline := time.Now().Add(time.Minute)
				request, cancel := context.WithDeadline(context.Background(), deadline)
				cancel()
				invokeRequestScope(t, protocol, root, request, false, false, func(ctx context.Context) {
					require.ErrorIs(t, ctx.Err(), context.Canceled)
					got, ok := ctx.Deadline()
					require.True(t, ok)
					require.Equal(t, deadline, got)
					Info(ctx, KV{K: "msg", V: "immediate"})
				})
				require.Contains(t, output.String(), "msg=immediate")
			})
		}
	}
}

func invokeRequestScope(t *testing.T, protocol string, root, request context.Context, ids, logging bool, handler func(context.Context)) {
	t.Helper()
	ctx := context.WithValue(request, requestScopeKey{}, "request value")
	switch protocol {
	case "HTTP":
		var opts []HTTPLogOption
		if !ids {
			opts = append(opts, WithDisableRequestID())
		}
		if !logging {
			opts = append(opts, WithDisableRequestLogging())
		}
		next := HTTP(root, opts...)(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
			handler(req.Context())
		}))
		next.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(ctx, "GET", "/scope", nil))
	case "unary", "streaming":
		var opts []GRPCLogOption
		if !ids {
			opts = append(opts, WithDisableCallID())
		}
		if !logging {
			opts = append(opts, WithDisableCallLogging())
		}
		if protocol == "unary" {
			_, err := UnaryServerInterceptor(root, opts...)(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/service/scope"}, func(ctx context.Context, _ any) (any, error) {
				handler(ctx)
				return nil, nil
			})
			require.NoError(t, err)
		} else {
			err := StreamServerInterceptor(root, opts...)(nil, &fakeServerStream{ctx: ctx}, &grpc.StreamServerInfo{FullMethod: "/service/scope"}, func(_ any, stream grpc.ServerStream) error {
				handler(stream.Context())
				return nil
			})
			require.NoError(t, err)
		}
	}
}
