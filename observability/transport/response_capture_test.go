package transport_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/observability/transport"
)

type (
	basicWriter       struct{ http.ResponseWriter }
	captureFastWriter struct {
		*httptest.ResponseRecorder
		strings, reads              int
		readDeadline, writeDeadline time.Time
		duplex                      bool
		failure                     error
	}
)

func (w *captureFastWriter) Write(b []byte) (int, error) {
	if w.failure != nil {
		b = b[:len(b)/2]
	}
	n, err := w.ResponseRecorder.Write(b)
	return n, errors.Join(err, w.failure)
}
func (w *captureFastWriter) WriteString(s string) (int, error) {
	w.strings++
	if w.failure != nil {
		s = s[:len(s)/2]
	}
	n, err := w.ResponseRecorder.WriteString(s)
	return n, errors.Join(err, w.failure)
}
func (w *captureFastWriter) ReadFrom(r io.Reader) (int64, error) {
	w.reads++
	if w.failure != nil {
		r = io.LimitReader(r, 2)
	}
	n, err := io.Copy(w.ResponseRecorder, r)
	return n, errors.Join(err, w.failure)
}
func (w *captureFastWriter) FlushError() error {
	w.Flush()
	return w.failure
}
func (w *captureFastWriter) SetReadDeadline(d time.Time) error {
	w.readDeadline = d
	return w.failure
}
func (w *captureFastWriter) SetWriteDeadline(d time.Time) error {
	w.writeDeadline = d
	return w.failure
}
func (w *captureFastWriter) EnableFullDuplex() error {
	w.duplex = true
	return w.failure
}
func (w *captureFastWriter) Push(string, *http.PushOptions) error {
	return w.failure
}

func TestCaptureResponseStatus(t *testing.T) {
	for _, tc := range []struct {
		name   string
		write  func(http.ResponseWriter)
		status int
	}{
		{"uncommitted", func(http.ResponseWriter) {
		}, 0},
		{"informational", func(w http.ResponseWriter) {
			w.WriteHeader(103)
		}, 0},
		{"final", func(w http.ResponseWriter) {
			w.WriteHeader(103)
			w.WriteHeader(201)
			w.WriteHeader(500)
		}, 201},
		{"upgrade", func(w http.ResponseWriter) {
			w.WriteHeader(101)
			w.WriteHeader(500)
		}, 101},
		{"flush", func(w http.ResponseWriter) {
			w.WriteHeader(103)
			w.(http.Flusher).Flush()
			w.WriteHeader(500)
		}, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, capture := transport.CaptureResponse(httptest.NewRecorder())
			tc.write(w)
			require.Equal(t, tc.status, capture.StatusCode())
		})
	}
}

func TestCaptureResponseCapabilities(t *testing.T) {
	w, _ := transport.CaptureResponse(basicWriter{httptest.NewRecorder()})
	_, flusher := w.(http.Flusher)
	_, hijacker := w.(http.Hijacker)
	_, pusher := w.(http.Pusher)
	_, reader := w.(io.ReaderFrom)
	_, writer := w.(io.StringWriter)
	require.False(t, flusher)
	require.False(t, hijacker)
	require.False(t, pusher)
	require.False(t, reader)
	require.False(t, writer)
	require.ErrorIs(t, http.NewResponseController(w).Flush(), http.ErrNotSupported)

	failure := errors.New("writer failure")
	original := &captureFastWriter{ResponseRecorder: httptest.NewRecorder(), failure: failure}
	w, capture := transport.CaptureResponse(original)
	controller := http.NewResponseController(w)
	deadline := time.Now()
	require.ErrorIs(t, controller.SetReadDeadline(deadline), failure)
	require.ErrorIs(t, controller.SetWriteDeadline(deadline), failure)
	require.Equal(t, deadline, original.readDeadline)
	require.Equal(t, deadline, original.writeDeadline)
	require.ErrorIs(t, controller.EnableFullDuplex(), failure)
	require.True(t, original.duplex)
	require.ErrorIs(t, w.(http.Pusher).Push("/asset", nil), failure)
	require.Zero(t, capture.StatusCode())
	require.ErrorIs(t, controller.Flush(), failure)
	require.True(t, original.Flushed)
	require.Equal(t, 200, capture.StatusCode())
	require.Same(t, original, w.(interface{ Unwrap() http.ResponseWriter }).Unwrap())
}

func TestCaptureResponseFastPaths(t *testing.T) {
	for _, failed := range []bool{false, true} {
		for _, text := range []string{"", "body"} {
			for _, method := range []string{"write", "string", "reader"} {
				t.Run(method+"/"+text+"/"+map[bool]string{true: "error", false: "success"}[failed], func(t *testing.T) {
					original := &captureFastWriter{ResponseRecorder: httptest.NewRecorder()}
					if failed {
						original.failure = errors.New("write failed")
					}
					// Nest captures as when logging surrounds transport observation.
					inner, a := transport.CaptureResponse(original)
					w, b := transport.CaptureResponse(inner)
					var n int64
					var err error
					switch method {
					case "write":
						var count int
						count, err = w.Write([]byte(text))
						n = int64(count)
					case "string":
						var count int
						count, err = io.WriteString(w, text)
						n = int64(count)
						require.Equal(t, 1, original.strings)
					case "reader":
						n, err = w.(io.ReaderFrom).ReadFrom(strings.NewReader(text))
						require.Equal(t, 1, original.reads)
					}
					if failed {
						require.ErrorIs(t, err, original.failure)
					} else {
						require.NoError(t, err)
					}
					want := len(text)
					if failed {
						want /= 2
					}
					require.EqualValues(t, want, n)
					for _, capture := range []*transport.ResponseCapture{a, b} {
						require.Equal(t, n, capture.BytesWritten())
						if method == "reader" && text == "" {
							require.Zero(t, capture.StatusCode())
						} else {
							require.Equal(t, 200, capture.StatusCode())
						}
					}
				})
			}
		}
	}
}
