package middleware_test

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	httpm "github.com/CaliLuke/loom/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	// plainResponseWriter hides the optional Flusher/Hijacker/Pusher
	// interfaces of the wrapped writer.
	plainResponseWriter struct {
		wrapped http.ResponseWriter
	}

	// hijackableWriter is a ResponseWriter that supports hijacking.
	hijackableWriter struct {
		http.ResponseWriter
		conn net.Conn
		rw   *bufio.ReadWriter
		err  error
	}

	// pushableWriter is a ResponseWriter that supports HTTP/2 push.
	pushableWriter struct {
		http.ResponseWriter
		target string
		opts   *http.PushOptions
		err    error
	}
)

func (w *plainResponseWriter) Header() http.Header {
	return w.wrapped.Header()
}

func (w *plainResponseWriter) Write(b []byte) (int, error) {
	return w.wrapped.Write(b)
}

func (w *plainResponseWriter) WriteHeader(code int) {
	w.wrapped.WriteHeader(code)
}

func (w *hijackableWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, w.rw, w.err
}

func (w *pushableWriter) Push(target string, opts *http.PushOptions) error {
	w.target = target
	w.opts = opts
	return w.err
}

func TestCaptureResponseWriteHeader(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		wantStatus int
	}{
		{name: "created", status: http.StatusCreated, wantStatus: http.StatusCreated},
		{name: "server error", status: http.StatusInternalServerError, wantStatus: http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			cw := httpm.CaptureResponse(rec)

			cw.WriteHeader(c.status)

			assert.Equal(t, c.wantStatus, cw.StatusCode)
			assert.Equal(t, c.wantStatus, rec.Code, "status not forwarded to underlying writer")
		})
	}
}

func TestCaptureResponseWrite(t *testing.T) {
	cases := []struct {
		name    string
		writes  []string
		wantLen int
	}{
		{name: "single write", writes: []string{"hello"}, wantLen: 5},
		{name: "multiple writes accumulate", writes: []string{"hello", ", ", "world"}, wantLen: 12},
		{name: "empty write", writes: []string{""}, wantLen: 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			cw := httpm.CaptureResponse(rec)

			var total bytes.Buffer
			for _, w := range c.writes {
				n, err := cw.Write([]byte(w))
				require.NoError(t, err)
				assert.Equal(t, len(w), n)
				total.WriteString(w)
			}

			assert.Equal(t, c.wantLen, cw.ContentLength)
			assert.Equal(t, http.StatusOK, cw.StatusCode)
			assert.Equal(t, total.String(), rec.Body.String(), "body not forwarded to underlying writer")
		})
	}
}

func TestCaptureResponseFlush(t *testing.T) {
	t.Run("forwards to flusher", func(t *testing.T) {
		rec := httptest.NewRecorder()
		cw := httpm.CaptureResponse(rec)

		cw.Flush()

		assert.True(t, rec.Flushed, "flush not forwarded to underlying writer")
		assert.Equal(t, http.StatusOK, cw.StatusCode)
	})

	t.Run("degrades without flusher", func(t *testing.T) {
		rec := httptest.NewRecorder()
		cw := httpm.CaptureResponse(&plainResponseWriter{wrapped: rec})

		require.NotPanics(t, func() {
			cw.Flush()
		})
		assert.False(t, rec.Flushed)
	})
}

func TestCaptureResponseHijack(t *testing.T) {
	t.Run("forwards to hijacker", func(t *testing.T) {
		client, server := net.Pipe()
		defer func() {
			require.NoError(t, client.Close())
			require.NoError(t, server.Close())
		}()
		rw := bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server))
		hw := &hijackableWriter{
			ResponseWriter: httptest.NewRecorder(),
			conn:           server,
			rw:             rw,
		}
		cw := httpm.CaptureResponse(hw)

		conn, gotRW, err := cw.Hijack()

		require.NoError(t, err)
		assert.Same(t, server, conn)
		assert.Same(t, rw, gotRW)
	})

	t.Run("forwards hijacker error", func(t *testing.T) {
		wantErr := errors.New("hijack failed")
		hw := &hijackableWriter{
			ResponseWriter: httptest.NewRecorder(),
			err:            wantErr,
		}
		cw := httpm.CaptureResponse(hw)

		conn, rw, err := cw.Hijack()

		require.ErrorIs(t, err, wantErr)
		assert.Nil(t, conn)
		assert.Nil(t, rw)
	})

	t.Run("errors without hijacker", func(t *testing.T) {
		// httptest.ResponseRecorder does not implement http.Hijacker.
		cw := httpm.CaptureResponse(httptest.NewRecorder())

		conn, rw, err := cw.Hijack()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not support hijacking")
		assert.Nil(t, conn)
		assert.Nil(t, rw)
	})
}

func TestCaptureResponsePush(t *testing.T) {
	t.Run("forwards to pusher", func(t *testing.T) {
		opts := &http.PushOptions{Method: "GET"}
		pw := &pushableWriter{ResponseWriter: httptest.NewRecorder()}
		cw := httpm.CaptureResponse(pw)

		err := cw.Push("/style.css", opts)

		require.NoError(t, err)
		assert.Equal(t, "/style.css", pw.target)
		assert.Same(t, opts, pw.opts)
	})

	t.Run("forwards pusher error", func(t *testing.T) {
		wantErr := errors.New("push failed")
		pw := &pushableWriter{ResponseWriter: httptest.NewRecorder(), err: wantErr}
		cw := httpm.CaptureResponse(pw)

		err := cw.Push("/style.css", nil)

		require.ErrorIs(t, err, wantErr)
	})

	t.Run("errors without pusher", func(t *testing.T) {
		// httptest.ResponseRecorder does not implement http.Pusher.
		cw := httpm.CaptureResponse(httptest.NewRecorder())

		err := cw.Push("/style.css", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "push not supported")
	})
}
