package middleware_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	loomhttp "github.com/CaliLuke/loom/http"
	httpm "github.com/CaliLuke/loom/http/middleware"
	"github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

type (
	debugBasicWriter struct {
		http.ResponseWriter
	}
	debugResponseWriter struct {
		header         http.Header
		body           bytes.Buffer
		status         int
		flushes        int
		deadline       time.Time
		duplex         bool
		failure        error
		partial        bool
		strings, reads int
	}
)

func (w *debugResponseWriter) Header() http.Header {
	return w.header
}
func (w *debugResponseWriter) WriteHeader(code int) {
	if w.status == 0 && (code == 101 || code >= 200) {
		w.status = code
	}
}
func (w *debugResponseWriter) Write(p []byte) (int, error) {
	w.WriteHeader(200)
	if w.partial {
		p = p[:len(p)/2]
	}
	n, err := w.body.Write(p)
	return n, errors.Join(err, w.failure)
}
func (w *debugResponseWriter) WriteString(s string) (int, error) {
	w.strings++
	return w.Write([]byte(s))
}
func (w *debugResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	w.reads++
	return io.Copy(struct{ io.Writer }{w}, r)
}
func (w *debugResponseWriter) Flush() {
	w.flushes++
	w.WriteHeader(200)
}
func (w *debugResponseWriter) FlushError() error {
	w.Flush()
	return w.failure
}
func (w *debugResponseWriter) SetWriteDeadline(d time.Time) error {
	w.deadline = d
	return w.failure
}
func (w *debugResponseWriter) EnableFullDuplex() error {
	w.duplex = true
	return w.failure
}
func (w *debugResponseWriter) Push(string, *http.PushOptions) error {
	return w.failure
}

func TestDebugResponseController(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[failed], func(t *testing.T) {
			original := &debugResponseWriter{header: make(http.Header)}
			if failed {
				original.failure = errors.New("transport failure")
			}
			var output bytes.Buffer
			httpm.Debug(loomhttp.NewMuxer(), &output)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				c := http.NewResponseController(w)
				deadline := time.Unix(123, 0)
				require.ErrorIs(t, c.SetWriteDeadline(deadline), original.failure)
				require.Equal(t, deadline, original.deadline)
				require.ErrorIs(t, c.EnableFullDuplex(), original.failure)
				require.True(t, original.duplex)
				require.ErrorIs(t, c.Flush(), original.failure)
				require.Equal(t, 1, original.flushes)
			})).ServeHTTP(original, httptest.NewRequest("GET", "/", nil))
		})
	}
}

func TestDebugResponseCapabilities(t *testing.T) {
	for _, capable := range []bool{false, true} {
		t.Run(map[bool]string{false: "basic", true: "capable"}[capable], func(t *testing.T) {
			var original http.ResponseWriter = &debugResponseWriter{header: make(http.Header)}
			if !capable {
				original = debugBasicWriter{original}
			}
			httpm.Debug(loomhttp.NewMuxer(), io.Discard)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, flush := w.(http.Flusher)
				require.Equal(t, capable, flush, "Flusher")
				_, push := w.(http.Pusher)
				require.Equal(t, capable, push, "Pusher")
				_, str := w.(io.StringWriter)
				require.Equal(t, capable, str, "StringWriter")
				_, read := w.(io.ReaderFrom)
				require.Equal(t, capable, read, "ReaderFrom")
				_, hijack := w.(http.Hijacker)
				require.False(t, hijack, "must not fabricate Hijacker")
			})).ServeHTTP(original, httptest.NewRequest("GET", "/", nil))
		})
	}
}

func TestDebugResponseStatus(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers []int
		write   bool
		want    int
	}{
		{"implicit", nil, true, 200}, {"first-final", []int{201, 500}, false, 201},
		{"informational", []int{103, 201}, false, 201}, {"upgrade", []int{101, 500}, false, 101},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := &debugResponseWriter{header: make(http.Header)}
			var output bytes.Buffer
			httpm.Debug(loomhttp.NewMuxer(), &output)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				for _, status := range tc.headers {
					w.WriteHeader(status)
				}
				if tc.write {
					_, err := w.Write([]byte("body"))
					require.NoError(t, err)
				}
			})).ServeHTTP(original, httptest.NewRequest("GET", "/", nil))
			require.Equal(t, tc.want, original.status)
			require.Contains(t, output.String(), "] "+http.StatusText(tc.want)+"\n")
		})
	}
}

func TestDebugResponseAcceptedBytes(t *testing.T) {
	for _, method := range []string{"write", "string", "reader"} {
		t.Run(method, func(t *testing.T) {
			failure := errors.New("partial write")
			original := &debugResponseWriter{header: make(http.Header), partial: true, failure: failure}
			var output bytes.Buffer
			httpm.Debug(loomhttp.NewMuxer(), &output)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				var n int64
				var err error
				switch method {
				case "write":
					var count int
					count, err = w.Write([]byte("abcdef"))
					n = int64(count)
				case "string":
					var count int
					count, err = io.WriteString(w, "abcdef")
					n = int64(count)
				case "reader":
					// An explicit commit makes the subsequent opaque copy metadata observable.
					w.WriteHeader(http.StatusOK)
					n, err = io.Copy(w, struct{ io.Reader }{strings.NewReader("abcdef")})
				}
				require.EqualValues(t, 3, n)
				require.ErrorIs(t, err, failure)
			})).ServeHTTP(original, httptest.NewRequest("GET", "/", nil))
			require.Equal(t, "abc", original.body.String())
			require.Contains(t, output.String(), "] abc\n")
			require.NotContains(t, output.String(), "abcdef")
			if method == "string" {
				require.Equal(t, 1, original.strings)
			}
			if method == "reader" {
				require.Equal(t, 1, original.reads)
			}
		})
	}
}

func TestDebugResponseSSEWithBodyPolicy(t *testing.T) {
	original := &debugResponseWriter{header: make(http.Header)}
	policy, err := loomhttp.NewRequestBodyPolicy(16)
	require.NoError(t, err)
	var output bytes.Buffer
	handler := httpm.Debug(loomhttp.NewMuxer(), &output)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "request", string(body))
		stream := loomhttp.NewSSEStreamWriter(w, r.Context(), transport.TransportHTTP, loomhttp.StreamWritePolicy{})
		require.NoError(t, stream.Open(r.Context()))
		require.NoError(t, stream.SendComment(r.Context(), "heartbeat"))
		require.Equal(t, 2, original.flushes, "must flush before handler returns")
		require.Equal(t, ": heartbeat\n\n", original.body.String())
	}))
	policy.Handler(handler).ServeHTTP(original, httptest.NewRequest("POST", "/events", strings.NewReader("request")))
}

func TestDebugResponseCaptureLimit(t *testing.T) {
	const limit = 64 << 10
	for _, size := range []int{limit - 1, limit, limit + 1, limit * 4} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			body := strings.Repeat("x", size)
			var output bytes.Buffer
			original := httptest.NewRecorder()
			httpm.Debug(loomhttp.NewMuxer(), &output)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				// Multiple writes exercise retention across events as well as one large body.
				for offset := 0; offset < len(body); {
					end := min(offset+8192, len(body))
					n, err := io.WriteString(w, body[offset:end])
					require.NoError(t, err)
					require.Equal(t, end-offset, n)
					offset = end
				}
			})).ServeHTTP(original, httptest.NewRequest("GET", "/", nil))
			require.Equal(t, body, original.Body.String(), "capture must not limit the wire")
			if size > limit {
				require.Contains(t, output.String(), "[body omitted after 65536 bytes]")
				require.NotContains(t, output.String(), strings.Repeat("x", 128))
			} else {
				require.Contains(t, output.String(), body)
				require.NotContains(t, output.String(), "body omitted")
			}
		})
	}
}

func TestDebugResponseSSETerminalFailure(t *testing.T) {
	failure := errors.New("SSE write failed")
	original := &debugResponseWriter{header: make(http.Header), failure: failure}
	var output bytes.Buffer
	httpm.Debug(loomhttp.NewMuxer(), &output)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := loomhttp.NewSSEStreamWriter(w, r.Context(), transport.TransportHTTP, loomhttp.StreamWritePolicy{})
		require.ErrorIs(t, stream.SendComment(r.Context(), "first"), failure)
		require.ErrorIs(t, stream.SendComment(r.Context(), "second"), loomhttp.ErrSSEStreamClosed)
		require.Equal(t, ": first\n\n", original.body.String())
	})).ServeHTTP(original, httptest.NewRequest("GET", "/events", nil))
	require.Contains(t, output.String(), ": first")
	require.NotContains(t, output.String(), ": second")
}

func TestDebugResponseSSEFlushVisibleBeforeReturn(t *testing.T) {
	release := make(chan struct{})
	result := make(chan error, 1)
	server := httptest.NewServer(httpm.Debug(loomhttp.NewMuxer(), io.Discard)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := loomhttp.NewSSEStreamWriter(w, r.Context(), transport.TransportHTTP, loomhttp.StreamWritePolicy{})
		err := stream.SendComment(r.Context(), "visible")
		result <- err
		if err != nil {
			return
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})))
	defer server.Close()
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	response, err := server.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, response.Body.Close())
	}()
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, ": visible\n", line)
	require.NoError(t, <-result)
}
