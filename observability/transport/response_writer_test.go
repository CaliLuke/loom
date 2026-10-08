package transport_test

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/observability/transport"
)

func TestCaptureResponseWriterRecordsStatusAndBytes(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	w, c := transport.CaptureResponse(rec)
	w.WriteHeader(http.StatusCreated)
	n, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, http.StatusCreated, c.StatusCode())
	require.EqualValues(t, 5, c.BytesWritten())
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "hello", rec.Body.String())
}

func TestCaptureResponseWriterImplicitOK(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	w, c := transport.CaptureResponse(rec)
	_, err := w.Write([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, c.StatusCode())
	require.EqualValues(t, 3, c.BytesWritten())
}

func TestCaptureResponseWriterFirstStatusWins(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	w, c := transport.CaptureResponse(rec)
	w.WriteHeader(http.StatusAccepted)
	w.WriteHeader(http.StatusInternalServerError)
	require.Equal(t, http.StatusAccepted, c.StatusCode())
}

type hijackableRecorder struct {
	*httptest.ResponseRecorder
	hijacked bool
}

func (h *hijackableRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, http.ErrNotSupported
}

func TestCaptureResponseWriterForwardsHijack(t *testing.T) {
	t.Parallel()
	h := &hijackableRecorder{ResponseRecorder: httptest.NewRecorder()}
	w, _ := transport.CaptureResponse(h)
	hijacker, ok := any(w).(http.Hijacker)
	require.True(t, ok)
	_, _, err := hijacker.Hijack()
	require.ErrorIs(t, err, http.ErrNotSupported)
	require.True(t, h.hijacked)
}

func TestCaptureResponseWriterInformationalStatus(t *testing.T) {
	w, c := transport.CaptureResponse(httptest.NewRecorder())
	w.WriteHeader(http.StatusEarlyHints)
	w.WriteHeader(http.StatusOK)
	require.Equal(t, http.StatusOK, c.StatusCode())
}

func TestCaptureResponseWriterCapabilities(t *testing.T) {
	w, _ := transport.CaptureResponse(httptest.NewRecorder())
	_, hijacker := any(w).(http.Hijacker)
	require.False(t, hijacker)
	_, flusher := any(w).(http.Flusher)
	require.True(t, flusher)
}
