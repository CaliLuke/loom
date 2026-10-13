package transport_test

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/observability/transport"
)

type readAheadCaptureWriter struct {
	*captureFastWriter
}

func (w *readAheadCaptureWriter) ReadFrom(r io.Reader) (int64, error) {
	w.reads++
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}

func TestCaptureResponseBodyPaths(t *testing.T) {
	for _, method := range []string{"write", "string", "reader"} {
		for _, size := range []int{0, 3, 4, 5, 100000} {
			t.Run(fmt.Sprintf("%s/%d", method, size), func(t *testing.T) {
				original := &captureFastWriter{ResponseRecorder: httptest.NewRecorder()}
				w, capture := transport.CaptureResponse(original, transport.WithResponseBodyCapture(4))
				text := strings.Repeat("x", size)
				switch method {
				case "write":
					n, err := w.Write([]byte(text))
					require.NoError(t, err)
					require.Equal(t, size, n)
				case "string":
					n, err := io.WriteString(w, text)
					require.NoError(t, err)
					require.Equal(t, size, n)
					require.Equal(t, 1, original.strings)
				case "reader":
					n, err := w.(io.ReaderFrom).ReadFrom(strings.NewReader(text))
					require.NoError(t, err)
					require.EqualValues(t, size, n)
					require.Equal(t, 1, original.reads)
				}
				require.Equal(t, text, original.Body.String())
				body, omitted := capture.Body()
				require.Equal(t, size > 4, omitted)
				if omitted {
					require.Nil(t, body)
				} else {
					require.Equal(t, text, string(body))
					if len(body) > 0 {
						body[0] = 'z'
						again, _ := capture.Body()
						require.Equal(t, text, string(again), "body access is detached")
					}
				}
			})
		}
	}
}

func TestCaptureResponseBodyReadAhead(t *testing.T) {
	for _, limit := range []int{2, 3, 4} {
		original := &readAheadCaptureWriter{&captureFastWriter{ResponseRecorder: httptest.NewRecorder(), failure: errors.New("partial")}}
		w, capture := transport.CaptureResponse(original, transport.WithResponseBodyCapture(limit))
		n, err := w.(io.ReaderFrom).ReadFrom(strings.NewReader("abcdef"))
		require.EqualValues(t, 3, n)
		require.ErrorIs(t, err, original.failure)
		require.Equal(t, 1, original.reads)
		require.Equal(t, "abc", original.Body.String())
		body, omitted := capture.Body()
		require.Equal(t, limit < 3, omitted)
		if limit >= 3 {
			require.Equal(t, "abc", string(body), "reader lookahead must not cause omission or capture rejected bytes")
		}
	}
}

func TestCaptureResponseBodyAcrossWrites(t *testing.T) {
	original := &captureFastWriter{ResponseRecorder: httptest.NewRecorder()}
	w, capture := transport.CaptureResponse(original, transport.WithResponseBodyCapture(4))
	n, err := w.Write([]byte("a"))
	require.NoError(t, err)
	require.Equal(t, 1, n)
	n, err = io.WriteString(w, "b")
	require.NoError(t, err)
	require.Equal(t, 1, n)
	count, err := w.(io.ReaderFrom).ReadFrom(strings.NewReader("cd"))
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	body, omitted := capture.Body()
	require.False(t, omitted)
	require.Equal(t, "abcd", string(body))
	n, err = w.Write([]byte("e"))
	require.NoError(t, err)
	require.Equal(t, 1, n)
	body, omitted = capture.Body()
	require.True(t, omitted)
	require.Nil(t, body)
	count, err = w.(io.ReaderFrom).ReadFrom(strings.NewReader("fg"))
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	body, omitted = capture.Body()
	require.True(t, omitted)
	require.Nil(t, body)
	require.Equal(t, "abcdefg", original.Body.String())
	require.Equal(t, http.StatusOK, capture.StatusCode())
	require.EqualValues(t, 7, capture.BytesWritten())
}

func TestCaptureResponseBodyDisabled(t *testing.T) {
	original := httptest.NewRecorder()
	w, capture := transport.CaptureResponse(original)
	n, err := w.Write([]byte("body"))
	require.NoError(t, err)
	require.Equal(t, 4, n)
	body, omitted := capture.Body()
	require.Nil(t, body)
	require.False(t, omitted)
	require.Panics(t, func() {
		transport.WithResponseBodyCapture(0)
	})
}

func TestCaptureResponseBodyNested(t *testing.T) {
	original := &captureFastWriter{ResponseRecorder: httptest.NewRecorder()}
	inner, a := transport.CaptureResponse(original, transport.WithResponseBodyCapture(4))
	w, b := transport.CaptureResponse(inner, transport.WithResponseBodyCapture(4))
	n, err := w.(io.ReaderFrom).ReadFrom(strings.NewReader("body"))
	require.NoError(t, err)
	require.EqualValues(t, 4, n)
	require.Equal(t, 1, original.reads)
	for _, capture := range []*transport.ResponseCapture{a, b} {
		body, omitted := capture.Body()
		require.False(t, omitted)
		require.Equal(t, "body", string(body))
		require.EqualValues(t, 4, capture.BytesWritten())
	}
}

func TestCaptureResponseBodyConcurrentObservation(t *testing.T) {
	w, capture := transport.CaptureResponse(httptest.NewRecorder(), transport.WithResponseBodyCapture(16))
	done := make(chan error, 1)
	go func() {
		for range 32 {
			if _, err := io.WriteString(w, "x"); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	for {
		body, omitted := capture.Body()
		require.LessOrEqual(t, len(body), 16)
		if omitted {
			require.Nil(t, body)
		}
		select {
		case err := <-done:
			require.NoError(t, err)
			return
		default:
		}
	}
}
