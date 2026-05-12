package transport_test

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

func TestCaptureResponseWriterRecordsStatusAndBytes(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	c := transport.NewCaptureResponseWriter(rec)
	c.WriteHeader(http.StatusCreated)
	n, err := c.Write([]byte("hello"))
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
	c := transport.NewCaptureResponseWriter(rec)
	_, err := c.Write([]byte("abc"))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, c.StatusCode())
	require.EqualValues(t, 3, c.BytesWritten())
}

func TestCaptureResponseWriterFirstStatusWins(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	c := transport.NewCaptureResponseWriter(rec)
	c.WriteHeader(http.StatusAccepted)
	c.WriteHeader(http.StatusInternalServerError)
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
	c := transport.NewCaptureResponseWriter(h)
	_, _, err := http.NewResponseController(c).Hijack()
	require.ErrorIs(t, err, http.ErrNotSupported)
	require.True(t, h.hijacked)
}
