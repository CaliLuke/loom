package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

const finalResultHarness = `package finalresult_test

import (
	"context"
	"errors"
	"io"
	"testing"

	svc "example.com/finalresult/gen/final"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type (
	contextKey     struct{}
	implementation struct {
		contextual bool
	}
	serverStream struct {
		sent        *svc.Message
		view        string
		sendErr     error
		recvErr     error
		calls       int
		sentContext context.Context
	}
	audit struct {
		t           *testing.T
		client      bool
		calls       int
		unaryCalls  int
		recvCalls   int
		stop        error
		wantContext string
	}
	clientStream struct {
		err   error
		calls int
	}
)

func TestServer(t *testing.T) {
	for _, contextual := range []bool{false, true} {
		for _, outcome := range []string{"success", "interceptor-error", "send-error", "receive-error"} {
			t.Run(outcome, func(t *testing.T) {
				requestCtx := context.WithValue(context.Background(), contextKey{}, "request")
				stream := &serverStream{}
				intercept := &audit{t: t, wantContext: "request"}
				if contextual {
					intercept.wantContext = "message"
				}
				sentinel := errors.New(outcome)
				switch outcome {
				case "interceptor-error":
					intercept.stop = sentinel
				case "send-error":
					stream.sendErr = sentinel
				case "receive-error":
					stream.recvErr = sentinel
				}
				endpoint := svc.NewEndpoints(&implementation{contextual: contextual}, intercept).Upload
				result, err := endpoint(requestCtx, &svc.UploadEndpointInput{Payload: &svc.UploadPayload{}, Stream: stream})
				require.Nil(t, result)
				require.Equal(t, MIXED_CALLS, intercept.unaryCalls)
				require.Equal(t, MIXED_CALLS, intercept.recvCalls)
				if outcome == "success" {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, sentinel)
				}
				if outcome == "receive-error" {
					require.Zero(t, intercept.calls)
					require.Zero(t, stream.calls)
					return
				}
				require.Equal(t, 1, intercept.calls)
				require.Equal(t, "EXPECTED_VIEW", stream.view)
				if outcome == "interceptor-error" {
					require.Zero(t, stream.calls)
					return
				}
				require.Equal(t, "server", *stream.sent.ID)
				require.Equal(t, 1, stream.calls)
				require.Equal(t, intercept.wantContext, stream.sentContext.Value(contextKey{}))
			})
		}
	}
}

func TestClient(t *testing.T) {
	for _, contextual := range []bool{false, true} {
		for _, transportErr := range []error{nil, io.ErrUnexpectedEOF} {
			intercept := &audit{t: t, client: true, wantContext: "request"}
			if contextual {
				intercept.wantContext = "message"
			}
			transport := &clientStream{err: transportErr}
			endpoint := svc.WrapUploadClientEndpoint(func(context.Context, any) (any, error) {
				return transport, nil
			}, intercept)
			requestCtx := context.WithValue(context.Background(), contextKey{}, "request")
			raw, err := endpoint(requestCtx, &svc.UploadPayload{})
			require.NoError(t, err)
			stream := raw.(svc.UploadClientStream)
			var result *svc.Message
			if contextual {
				result, err = stream.CloseAndRecvWithContext(context.WithValue(requestCtx, contextKey{}, "message"))
			} else {
				result, err = stream.CloseAndRecv()
			}
			require.Equal(t, 1, transport.calls)
			require.Equal(t, MIXED_CALLS, intercept.unaryCalls)
			if transportErr != nil {
				require.ErrorIs(t, err, transportErr)
				require.Nil(t, result)
				require.Zero(t, intercept.calls)
			} else {
				require.NoError(t, err)
				require.Equal(t, "client", *result.ID)
				require.Equal(t, 1, intercept.calls)
			}
		}
	}
}

func (s *implementation) Upload(ctx context.Context, p *svc.UploadPayload, stream svc.UploadServerStream) error {
	if s.contextual {
		ctx = context.WithValue(ctx, contextKey{}, "message")
		if _, err := stream.RecvWithContext(ctx); err != nil {
			return err
		}
	} else {
		if _, err := stream.Recv(); err != nil {
			return err
		}
	}
	VIEW_SELECTION
	id := "original"
	result := &svc.Message{ID: &id}
	if s.contextual {
		return stream.SendAndCloseWithContext(ctx, result)
	}
	return stream.SendAndClose(result)
}

func (s *serverStream) SetView(v string) {
	s.view = v
}
func (s *serverStream) SendAndClose(v *svc.Message) error {
	return s.SendAndCloseWithContext(context.Background(), v)
}
func (s *serverStream) SendAndCloseWithContext(ctx context.Context, v *svc.Message) error {
	s.calls++
	s.sent = v
	s.sentContext = ctx
	return s.sendErr
}
func (s *serverStream) Recv() (*svc.UploadStreamingPayload, error) {
	return s.RecvWithContext(context.Background())
}
func (s *serverStream) RecvWithContext(context.Context) (*svc.UploadStreamingPayload, error) {
	if s.recvErr != nil {
		return nil, s.recvErr
	}
	v := "chunk"
	return &svc.UploadStreamingPayload{Chunk: &v}, nil
}

func (a *audit) Audit(ctx context.Context, info *svc.AuditInfo, next loom.Endpoint) (any, error) {
	if info.CallType() == loom.InterceptorUnary {
		a.unaryCalls++
	}
	if !a.client && info.CallType() == loom.InterceptorStreamingRecv {
		a.recvCalls++
	}
	if !a.client && info.CallType() == loom.InterceptorStreamingSend {
		a.calls++
		require.Equal(a.t, a.wantContext, ctx.Value(contextKey{}))
		require.Equal(a.t, "original", info.ServerStreamingResult().ID())
		info.ServerStreamingResult().SetID("server")
		if a.stop != nil {
			return nil, a.stop
		}
		return next(ctx, info.RawPayload())
	}
	res, err := next(ctx, info.RawPayload())
	if a.client && info.CallType() == loom.InterceptorStreamingRecv && err == nil {
		a.calls++
		require.Equal(a.t, a.wantContext, ctx.Value(contextKey{}))
		require.Equal(a.t, "original", info.ClientStreamingResult(res).ID())
		info.ClientStreamingResult(res).SetID("client")
	}
	return res, err
}

func (s *clientStream) Send(v *svc.UploadStreamingPayload) error {
	return nil
}
func (s *clientStream) SendWithContext(context.Context, *svc.UploadStreamingPayload) error {
	return nil
}
func (s *clientStream) CloseAndRecv() (*svc.Message, error) {
	return s.CloseAndRecvWithContext(context.Background())
}
func (s *clientStream) CloseAndRecvWithContext(context.Context) (*svc.Message, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	v := "original"
	return &svc.Message{ID: &v}, nil
}
`

// TestFinalResultInterceptorRuntime exercises both ends of generated client streams.
func TestFinalResultInterceptorRuntime(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		viewed, mixed, fixed bool
	}{{"plain", false, false, false}, {"viewed", true, false, false}, {"mixed", true, true, false}, {"fixed", true, false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				Interceptor("audit", func() {
					ReadStreamingResult(func() {
						Attribute("id")
					})
					WriteStreamingResult(func() {
						Attribute("id")
					})
					if tc.mixed {
						ReadPayload(func() {
							Attribute("initial")
						})
						ReadStreamingPayload(func() {
							Attribute("chunk")
						})
					}
				})
				fields := func() {
					Attribute("id", String)
					Attribute("name", String)
				}
				var result any
				if tc.viewed {
					result = ResultType("application/vnd.finalmessage", "Message", func() {
						fields()
						View("default", func() {
							Attribute("id")
							Attribute("name")
						})
						View("tiny", func() {
							Attribute("id")
						})
					})
				} else {
					result = Type("Message", fields)
				}
				Service("final", func() {
					ServerInterceptor("audit")
					ClientInterceptor("audit")
					Method("upload", func() {
						Payload(func() {
							Attribute("initial", String)
						})
						StreamingPayload(func() {
							Attribute("chunk", String)
						})
						if tc.fixed {
							Result(result, func() {
								View("tiny")
							})
						} else {
							Result(result)
						}
						HTTP(func() {
							GET("/upload")
							Header("initial")
						})
					})
				})
			})
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/finalresult", root)
			harness := strings.ReplaceAll(finalResultHarness, "MIXED_CALLS", "0")
			if tc.mixed {
				harness = strings.ReplaceAll(finalResultHarness, "MIXED_CALLS", "1")
			}
			if tc.viewed && !tc.fixed {
				harness = strings.ReplaceAll(harness, "VIEW_SELECTION", "stream.SetView(\"tiny\")")
				harness = strings.ReplaceAll(harness, "EXPECTED_VIEW", "tiny")
			} else {
				harness = strings.ReplaceAll(harness, "VIEW_SELECTION", "")
				harness = strings.ReplaceAll(harness, "EXPECTED_VIEW", "")
			}
			require.NoError(t, os.WriteFile(filepath.Join(dir, "contract_test.go"), []byte(harness), 0600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "vet", "./...")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
		})
	}
}
