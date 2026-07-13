package log

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type partialFailingBody struct {
	sent bool
}

func (b *partialFailingBody) Read(p []byte) (int, error) {
	if b.sent {
		return 0, errors.New("read failed")
	}
	b.sent = true
	return copy(p, "partial"), nil
}

func (*partialFailingBody) Close() error {
	return nil
}

// pusherRecorder is a response writer that records http.Pusher calls.
type pusherRecorder struct {
	http.ResponseWriter
	target string
}

func (p *pusherRecorder) Push(target string, opts *http.PushOptions) error {
	p.target = target
	return nil
}

// hijackerRecorder is a response writer that records http.Hijacker calls.
type hijackerRecorder struct {
	http.ResponseWriter
	hijacked bool
}

func (h *hijackerRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, nil
}

// noFlushResponseWriter implements only http.ResponseWriter.
type noFlushResponseWriter struct {
	header http.Header
}

func (w *noFlushResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *noFlushResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (w *noFlushResponseWriter) WriteHeader(int) {}

func TestHTTPMiddleware(t *testing.T) {
	stubShortID(t, "req-123")
	stubTimeSince(t, 42*time.Millisecond)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
	})

	cases := []struct {
		name        string
		opts        []HTTPLogOption
		handler     http.Handler
		contains    []string
		notContains []string
		empty       bool
	}{
		{
			name:    "logs start and end with request id",
			handler: okHandler,
			contains: []string{
				"msg=start",
				"msg=end",
				"http.method=GET",
				`http.url="http://example.com/path?q=1"`,
				"http.status=202",
				"http.time_ms=42",
				"http.bytes=2",
				"request_id=req-123",
			},
		},
		{
			name:        "disable request id",
			opts:        []HTTPLogOption{WithDisableRequestID()},
			handler:     okHandler,
			contains:    []string{"msg=start", "msg=end"},
			notContains: []string{"request_id="},
		},
		{
			name: "disable request logging still enriches context",
			opts: []HTTPLogOption{WithDisableRequestLogging()},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Print(r.Context(), KV{K: "inner", V: "handler"})
			}),
			contains:    []string{"inner=handler", "request_id=req-123"},
			notContains: []string{"msg=start", "msg=end"},
		},
		{
			name:    "path filter skips logging",
			opts:    []HTTPLogOption{WithPathFilter(regexp.MustCompile(`^/path$`))},
			handler: okHandler,
			empty:   true,
		},
		{
			name: "nil option is ignored",
			opts: []HTTPLogOption{nil},
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			}),
			contains: []string{"msg=start", "msg=end"},
		},
		{
			name: "status is zero when handler does not write header",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			}),
			contains: []string{"http.status=0"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logCtx := newTestContext(&buf)
			middleware := HTTP(logCtx, tc.opts...)

			req := httptest.NewRequest(http.MethodGet, "http://example.com/path?q=1", nil)
			rec := httptest.NewRecorder()
			middleware(tc.handler).ServeHTTP(rec, req)

			if tc.empty {
				require.Empty(t, buf.String())
				return
			}
			for _, want := range tc.contains {
				assert.Contains(t, buf.String(), want)
			}
			for _, unwanted := range tc.notContains {
				assert.NotContains(t, buf.String(), unwanted)
			}
		})
	}
}

func TestHTTPMiddlewareCustomLogFunc(t *testing.T) {
	stubShortID(t, "req-123")
	var buf bytes.Buffer
	logCtx := newTestContext(&buf, WithDebug())
	middleware := HTTP(logCtx, WithRequestLogFunc(Debug))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
	})).ServeHTTP(httptest.NewRecorder(), req)

	require.Contains(t, buf.String(), "level=debug")
	require.Contains(t, buf.String(), "msg=start")
	require.Contains(t, buf.String(), "msg=end")
}

func TestHTTPPanicsWithoutLogger(t *testing.T) {
	require.Panics(t, func() {
		HTTP(context.Background())
	})
}

func TestEndpoint(t *testing.T) {
	var buf bytes.Buffer
	ctx := newTestContext(&buf)
	ctx = context.WithValue(ctx, loom.ServiceKey, "calc")
	ctx = context.WithValue(ctx, loom.MethodKey, "add")

	ep := Endpoint(func(ctx context.Context, req any) (any, error) {
		Print(ctx, KV{K: "inner", V: "call"})
		return "resp", nil
	})
	res, err := ep(ctx, nil)

	require.NoError(t, err)
	require.Equal(t, "resp", res)
	require.Contains(t, buf.String(), "loom.service=calc")
	require.Contains(t, buf.String(), "loom.method=add")
	require.Contains(t, buf.String(), "inner=call")
}

func TestClientRoundTrip(t *testing.T) {
	stubTimeSince(t, 42*time.Millisecond)

	cases := []struct {
		name     string
		opts     []HTTPClientLogOption
		resp     *http.Response
		rtErr    error
		wantErr  bool
		contains []string
	}{
		{
			name: "success logs info",
			resp: &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Body:       http.NoBody,
			},
			contains: []string{
				"level=info",
				`msg="finished client HTTP request"`,
				"http.method=GET",
				`http.status="200 OK"`,
				"http.time_ms=42",
			},
		},
		{
			name: "error status logs error",
			resp: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Body:       http.NoBody,
			},
			contains: []string{
				"level=error",
				`err="500 Internal Server Error"`,
				`http.status="500 Internal Server Error"`,
			},
		},
		{
			name: "error status logs body when configured",
			opts: []HTTPClientLogOption{WithLogBodyOnError()},
			resp: &http.Response{
				StatusCode: http.StatusBadRequest,
				Status:     "400 Bad Request",
				Body:       io.NopCloser(bytes.NewBufferString("oops")),
			},
			contains: []string{
				"level=error",
				"http.body=oops",
			},
		},
		{
			name: "custom error status matches",
			opts: []HTTPClientLogOption{WithErrorStatus(http.StatusTeapot)},
			resp: &http.Response{
				StatusCode: http.StatusTeapot,
				Status:     "418 I'm a teapot",
				Body:       http.NoBody,
			},
			contains: []string{"level=error"},
		},
		{
			name: "custom error status treats other statuses as success",
			opts: []HTTPClientLogOption{WithErrorStatus(http.StatusTeapot)},
			resp: &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     "500 Internal Server Error",
				Body:       http.NoBody,
			},
			contains: []string{"level=info"},
		},
		{
			name:    "transport error is logged and returned",
			rtErr:   errors.New("connection refused"),
			wantErr: true,
			contains: []string{
				"level=error",
				`err="connection refused"`,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := newTestContext(&buf)
			rt := Client(roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return tc.resp, tc.rtErr
			}), tc.opts...)

			req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)
			resp, err := rt.RoundTrip(req.WithContext(ctx))

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.resp.StatusCode, resp.StatusCode)
			}
			if resp != nil && resp.Body != nil {
				require.NoError(t, resp.Body.Close())
			}
			for _, want := range tc.contains {
				assert.Contains(t, buf.String(), want)
			}
		})
	}
}

func TestClientLogBodyOnErrorKeepsBodyReadable(t *testing.T) {
	stubTimeSince(t, time.Millisecond)
	var buf bytes.Buffer
	ctx := newTestContext(&buf)
	rt := Client(roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Body:       io.NopCloser(bytes.NewBufferString("oops")),
		}, nil
	}), WithLogBodyOnError())

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)
	resp, err := rt.RoundTrip(req.WithContext(ctx))
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "oops", string(body))
	require.NoError(t, resp.Body.Close())
}

func TestResponseCaptureWriteAndHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseCapture{ResponseWriter: rec}

	rw.WriteHeader(http.StatusCreated)
	n, err := rw.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)

	assert.Equal(t, http.StatusCreated, rw.StatusCode)
	assert.Equal(t, 5, rw.ContentLength)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
}

func TestResponseCaptureImplicitStatusOK(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseCapture{ResponseWriter: rec}

	n, err := rw.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, http.StatusOK, rw.StatusCode)

	rw = &responseCapture{ResponseWriter: httptest.NewRecorder()}
	rw.Flush()
	require.Equal(t, http.StatusOK, rw.StatusCode)
}

func TestResponseCaptureFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseCapture{ResponseWriter: rec}
	rw.Flush()
	require.True(t, rec.Flushed)

	// Flushing a non-flusher writer must not panic.
	require.NotPanics(t, func() {
		(&responseCapture{ResponseWriter: &noFlushResponseWriter{}}).Flush()
	})
}

func TestClientLogBodyOnErrorReadFailure(t *testing.T) {
	stubTimeSince(t, time.Millisecond)
	var buf bytes.Buffer
	ctx := newTestContext(&buf)
	rt := Client(roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Status:     "400 Bad Request",
			Body:       &partialFailingBody{},
		}, nil
	}), WithLogBodyOnError())

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api", nil)
	resp, err := rt.RoundTrip(req.WithContext(ctx))

	require.NoError(t, err)
	require.NotNil(t, resp)
	body, readErr := io.ReadAll(resp.Body)
	require.Equal(t, "partial", string(body))
	require.EqualError(t, readErr, "read failed")
	require.NoError(t, resp.Body.Close())
	require.Contains(t, buf.String(), `err="read failed"`)
}

func TestResponseCapturePush(t *testing.T) {
	rw := &responseCapture{ResponseWriter: httptest.NewRecorder()}
	err := rw.Push("/asset", nil)
	require.EqualError(t, err, "push not supported")

	p := &pusherRecorder{ResponseWriter: httptest.NewRecorder()}
	rw = &responseCapture{ResponseWriter: p}
	require.NoError(t, rw.Push("/asset", nil))
	require.Equal(t, "/asset", p.target)
}

func TestResponseCaptureHijack(t *testing.T) {
	h := &hijackerRecorder{ResponseWriter: httptest.NewRecorder()}
	rw := &responseCapture{ResponseWriter: h}
	conn, brw, err := rw.Hijack()
	require.NoError(t, err)
	require.Nil(t, conn)
	require.Nil(t, brw)
	require.True(t, h.hijacked)
}

func TestResponseCaptureHijackUnsupported(t *testing.T) {
	rw := &responseCapture{ResponseWriter: &noFlushResponseWriter{}}
	conn, brw, err := rw.Hijack()
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support hijacking")
	require.Nil(t, conn)
	require.Nil(t, brw)
}

func TestFrom(t *testing.T) {
	cases := []struct {
		name       string
		forwarded  string
		remoteAddr string
		want       string
	}{
		{"x-forwarded-for wins", "10.0.0.1", "192.168.1.1:1234", "10.0.0.1"},
		{"remote addr with port", "", "192.168.1.1:1234", "192.168.1.1"},
		{"remote addr without port", "", "192.168.1.1", "192.168.1.1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.forwarded != "" {
				req.Header.Set("X-Forwarded-For", tc.forwarded)
			}
			require.Equal(t, tc.want, from(req))
		})
	}
}
