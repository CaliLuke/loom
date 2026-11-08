package codegen

// informationalContinueHarness extends the maintained informational module. The
// real net/http server, rather than an authored emitter, produces Continue.
const informationalContinueHarness = `package selection_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"strings"
	"testing"

	client "example.com/selection/gen/http/selection/client"
	server "example.com/selection/gen/http/selection/server"
	service "example.com/selection/gen/selection"
	loomhttp "github.com/CaliLuke/loom/http"
)

type expectDoer struct {
	client *http.Client
}

func (d expectDoer) Do(req *http.Request) (*http.Response, error) {
	req.Body = io.NopCloser(strings.NewReader("request"))
	req.ContentLength = int64(len("request"))
	req.Header.Set("Expect", "100-continue")
	return d.client.Do(req)
}

func TestDeclaredContinueNativeWire(t *testing.T) {
	handler := server.NewReadHandler(func(context.Context, any) (any, error) {
		return &service.SelectionResult{Message: "message", Trace: "final"}, nil
	}, loomhttp.NewMuxer(), loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if _, err := io.Copy(io.Discard, req.Body); err != nil {
			return
		}
		handler.ServeHTTP(w, req)
	}))
	defer srv.Close()
	abort := errors.New("continue observer aborted")
	for _, tc := range []struct {
		name          string
		observe       bool
		class         bool
		callbackError error
	}{
		{name: "typed exact", observe: true},
		{name: "nil exact shadows class", class: true},
		{name: "all nil"},
		{name: "callback abort", observe: true, callbackError: abort},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exact, classes, native := 0, 0, 0
			ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
				Got1xxResponse: func(code int, _ textproto.MIMEHeader) error {
					if code == 100 {
						native++
					}
					return nil
				},
			})
			ctx = client.WithReadContinue(ctx, nil)
			if tc.observe {
				ctx = client.WithReadContinue(ctx, func(_ context.Context, value *service.Continued, decodeErr error) error {
					exact++
					if decodeErr != nil || value == nil || value.Status != 100 || value.Trace != nil {
						t.Errorf("automatic metadata=%#v error=%v", value, decodeErr)
					}
					return tc.callbackError
				})
			}
			if tc.class {
				ctx = client.WithReadProgress(ctx, func(context.Context, *service.Progress, error) error {
					classes++
					return nil
				})
			}
			c := client.NewClient("http", strings.TrimPrefix(srv.URL, "http://"), expectDoer{srv.Client()}, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			result, err := c.Read()(ctx, nil)
			if tc.callbackError != nil {
				if !errors.Is(err, tc.callbackError) || result != nil {
					t.Errorf("abort result=%#v error=%v", result, err)
				}
			} else if err != nil || result == nil {
				t.Errorf("result=%#v error=%v", result, err)
			}
			want := 0
			if tc.observe {
				want = 1
			}
			if exact != want || classes != 0 || native != 1 {
				t.Errorf("callbacks exact=%d class=%d native=%d", exact, classes, native)
			}
		})
	}
}

type continueWriter struct {
	header http.Header
	writes int
}

func (w *continueWriter) Header() http.Header {
	return w.header
}
func (w *continueWriter) WriteHeader(int) {
	w.writes++
}
func (w *continueWriter) Write(b []byte) (int, error) {
	w.writes++
	return len(b), nil
}

func TestDeclaredContinueCannotEmit(t *testing.T) {
	for _, send := range []func(context.Context) error{
		func(ctx context.Context) error {
			trace := "must not publish"
			return server.SendReadContinue(ctx, &service.Continued{Status: 100, Trace: &trace})
		},
		func(ctx context.Context) error {
			return server.SendBodyContinue(ctx, &service.Marker{})
		},
	} {
		w := &continueWriter{header: http.Header{"X-Pending": {"keep"}}}
		lifecycle := loomhttp.NewHandlerLifecycle(w, httptest.NewRequest(http.MethodGet, "/read", nil), "selection", "read")
		err := send(lifecycle.Context())
		lifecycle.End()
		var violation *loomhttp.StatusViolationError
		if !errors.As(err, &violation) {
			t.Errorf("send error=%T %v", err, err)
		}
		if w.writes != 0 || len(w.header) != 1 || w.header.Get("X-Pending") != "keep" {
			t.Errorf("rejected Send published: writes=%d headers=%v", w.writes, w.header)
		}
	}
}
`
