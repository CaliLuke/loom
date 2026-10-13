package http

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	responseObservationBody struct {
		data     string
		readErr  error
		closeErr error
		reads    int
		closes   int
		sizes    []int
	}
	responseObservationDoer struct {
		body io.ReadCloser
	}
)

func TestDebugResponseObservationIsPassive(t *testing.T) {
	body := &responseObservationBody{data: "payload", readErr: io.EOF}
	debug := NewDebugDoer(responseObservationDoer{body: body})
	response, err := debug.Do(httptest.NewRequest("GET", "http://example.test", nil))
	require.NoError(t, err)
	require.Zero(t, body.reads, "Do must not consume the response")
	var snapshot bytes.Buffer
	debug.Fprint(&snapshot)
	require.Zero(t, body.reads, "Fprint must not consume the response")
	buf := make([]byte, 32)
	n, err := response.Body.Read(buf)
	require.Equal(t, "payload", string(buf[:n]))
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, []int{32}, body.sizes)
	require.NoError(t, response.Body.Close())
	require.Equal(t, 1, body.closes)
}

func TestDebugResponseObservationPreservesFailures(t *testing.T) {
	for _, data := range []string{"", "partial"} {
		t.Run(data, func(t *testing.T) {
			readErr := errors.New("original read failure")
			closeErr := errors.New("original close failure")
			body := &responseObservationBody{data: data, readErr: readErr, closeErr: closeErr}
			response, err := NewDebugDoer(responseObservationDoer{body: body}).Do(httptest.NewRequest("GET", "http://example.test", nil))
			require.NoError(t, err)
			buf := make([]byte, 32)
			n, err := response.Body.Read(buf)
			require.Equal(t, data, string(buf[:n]))
			require.Same(t, readErr, err)
			require.Same(t, closeErr, response.Body.Close())
			require.Same(t, closeErr, response.Body.Close())
			require.Equal(t, 2, body.closes)
		})
	}
}

func TestDebugResponseObservationRetainsCloseAfterReadFailure(t *testing.T) {
	closeErr := errors.New("original close failure")
	body := &responseObservationBody{readErr: errors.New("read failure"), closeErr: closeErr}
	response, err := NewDebugDoer(responseObservationDoer{body: body}).Do(httptest.NewRequest("GET", "http://example.test", nil))
	require.NoError(t, err)
	require.Same(t, closeErr, response.Body.Close())
	require.Equal(t, 1, body.closes)
}

func TestDebugResponseObservationEmptyIdentity(t *testing.T) {
	for _, tc := range []struct {
		name string
		body io.ReadCloser
	}{{"nil", nil}, {"NoBody", http.NoBody}} {
		t.Run(tc.name, func(t *testing.T) {
			debug := NewDebugDoer(responseObservationDoer{body: tc.body})
			require.NotPanics(t, func() {
				response, err := debug.Do(httptest.NewRequest("GET", "http://example.test", nil))
				require.NoError(t, err)
				if response.Body != nil {
					require.NoError(t, response.Body.Close())
				}
				require.Equal(t, tc.body, response.Body)
				var snapshot bytes.Buffer
				debug.Fprint(&snapshot)
			})
		})
	}
}

// Do returns the configured response without consuming its body.
func (d responseObservationDoer) Do(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: d.body}, nil
}

// Read records the requested size and returns the configured bytes and error.
func (b *responseObservationBody) Read(p []byte) (int, error) {
	b.reads++
	b.sizes = append(b.sizes, len(p))
	return copy(p, b.data), b.readErr
}

// Close records every call and returns the configured source error.
func (b *responseObservationBody) Close() error {
	b.closes++
	return b.closeErr
}
