package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

type trackedReadCloser struct {
	io.Reader
	closeErr error
	closed   bool
}

func (r *trackedReadCloser) Close() error {
	r.closed = true
	return r.closeErr
}

type failingWriterToBody struct {
	err    error
	wrote  int64
	closed bool
}

type trackedReadSeekCloser struct {
	*strings.Reader
	closeErr error
	closed   bool
}

func (r *trackedReadSeekCloser) Close() error {
	r.closed = true
	return r.closeErr
}

func (b *failingWriterToBody) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (b *failingWriterToBody) WriteTo(w io.Writer) (int64, error) {
	if b.wrote > 0 {
		_, writeErr := w.Write([]byte("x"))
		if writeErr != nil {
			return 0, writeErr
		}
	}
	return b.wrote, b.err
}

func (b *failingWriterToBody) Close() error {
	b.closed = true
	return nil
}

func TestHandlerLifecycleWritesAndClosesRawBody(t *testing.T) {
	body := &trackedReadCloser{Reader: strings.NewReader("payload")}
	recorder := &unaryEventRecorder{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/raw", nil)
	request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
	lifecycle := NewHandlerLifecycle(response, request, "raw", "download")

	lifecycle.WriteRawBody(
		body,
		func(_ context.Context, w http.ResponseWriter) error {
			w.WriteHeader(http.StatusCreated)
			return nil
		},
		func(context.Context, http.ResponseWriter, error) error {
			t.Error("unexpected error encoder call")
			return nil
		},
		nil,
	)
	lifecycle.End()

	require.True(t, body.closed)
	require.Equal(t, http.StatusCreated, response.Code)
	require.Equal(t, "payload", response.Body.String())
	require.Len(t, recorder.events, 2)
	require.Equal(t, loomtransport.ReasonOK, recorder.events[1].Reason)
}

func TestHandlerLifecycleRoutesRawBodyFailures(t *testing.T) {
	errWrite := errors.New("write")
	body := &failingWriterToBody{err: errWrite}
	recorder := &unaryEventRecorder{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/raw", nil)
	request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
	lifecycle := NewHandlerLifecycle(response, request, "raw", "download")
	var encodedErr, handledErr error

	lifecycle.WriteRawBody(
		body,
		func(context.Context, http.ResponseWriter) error {
			return nil
		},
		func(_ context.Context, w http.ResponseWriter, err error) error {
			encodedErr = err
			w.WriteHeader(http.StatusBadGateway)
			return nil
		},
		func(_ context.Context, _ http.ResponseWriter, err error) {
			handledErr = err
		},
	)
	lifecycle.End()

	require.True(t, body.closed)
	require.ErrorIs(t, encodedErr, errWrite)
	require.NoError(t, handledErr)
	require.Equal(t, http.StatusBadGateway, response.Code)
	require.Len(t, recorder.events, 2)
	require.Equal(t, loomtransport.ReasonResponseWriteFailed, recorder.events[1].Reason)
}

func TestHandlerLifecycleDoesNotEncodeCommittedZeroByteRawFailure(t *testing.T) {
	errWrite := errors.New("write")
	body := &failingWriterToBody{err: errWrite}
	recorder := &unaryEventRecorder{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/raw", nil)
	request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
	var encoded bool

	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		lifecycle := NewHandlerLifecycle(response, request, "raw", "download")
		defer lifecycle.End()
		lifecycle.WriteRawBody(
			body,
			func(_ context.Context, w http.ResponseWriter) error {
				w.WriteHeader(http.StatusCreated)
				return nil
			},
			func(context.Context, http.ResponseWriter, error) error {
				encoded = true
				return nil
			},
			nil,
		)
	})

	require.True(t, body.closed)
	require.False(t, encoded)
	require.Equal(t, http.StatusCreated, response.Code)
	require.Len(t, recorder.events, 2)
	require.Equal(t, loomtransport.ReasonPanic, recorder.events[1].Reason)
}

func TestHandlerLifecycleAbortsLateRawBodyFailure(t *testing.T) {
	errWrite := errors.New("write")
	body := &failingWriterToBody{err: errWrite, wrote: 1}
	recorder := &unaryEventRecorder{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/raw", nil)
	request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))

	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		lifecycle := NewHandlerLifecycle(response, request, "raw", "download")
		defer lifecycle.End()
		lifecycle.WriteRawBody(
			body,
			func(context.Context, http.ResponseWriter) error {
				return nil
			},
			func(context.Context, http.ResponseWriter, error) error {
				t.Error("late write failure must not be encoded")
				return nil
			},
			nil,
		)
	})

	require.True(t, body.closed)
	require.Equal(t, "x", response.Body.String())
	require.Len(t, recorder.events, 2)
	require.Equal(t, loomtransport.ReasonPanic, recorder.events[1].Reason)
}

func TestHandlerLifecycleReportsRawBodyCloseFailure(t *testing.T) {
	errClose := errors.New("close")
	body := &trackedReadCloser{Reader: strings.NewReader("payload"), closeErr: errClose}
	recorder := &unaryEventRecorder{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/raw", nil)
	request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
	lifecycle := NewHandlerLifecycle(response, request, "raw", "download")
	var handledErr error

	lifecycle.WriteRawBody(
		body,
		func(context.Context, http.ResponseWriter) error {
			return nil
		},
		func(context.Context, http.ResponseWriter, error) error {
			return nil
		},
		func(_ context.Context, _ http.ResponseWriter, err error) {
			handledErr = err
		},
	)
	lifecycle.End()

	require.ErrorIs(t, handledErr, errClose)
	require.Len(t, recorder.events, 2)
	require.Equal(t, loomtransport.ReasonResponseWriteFailed, recorder.events[1].Reason)
}

func TestHandlerLifecycleServesAndClosesFile(t *testing.T) {
	content := &trackedReadSeekCloser{Reader: strings.NewReader("file")}
	recorder := &unaryEventRecorder{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
	lifecycle := NewHandlerLifecycle(response, request, "files", "download")

	lifecycle.ServeFile(
		request,
		&FileResponse{Name: "sample.bin", Content: content},
		"application/octet-stream",
		func(_ context.Context, w http.ResponseWriter) error {
			w.Header().Set("ETag", "example")
			return nil
		},
		func(context.Context, http.ResponseWriter, error) error {
			t.Error("unexpected error encoder call")
			return nil
		},
		nil,
	)
	lifecycle.End()

	require.True(t, content.closed)
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "file", response.Body.String())
	require.Equal(t, "application/octet-stream", response.Header().Get("Content-Type"))
	require.Equal(t, "example", response.Header().Get("ETag"))
	require.Len(t, recorder.events, 2)
	require.Equal(t, loomtransport.ReasonOK, recorder.events[1].Reason)
}

func TestHandlerLifecycleRejectsNilFileContent(t *testing.T) {
	recorder := &unaryEventRecorder{}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/file", nil)
	request = request.WithContext(loomtransport.WithObserver(request.Context(), recorder))
	lifecycle := NewHandlerLifecycle(response, request, "files", "download")
	var encodedErr error

	lifecycle.ServeFile(
		request,
		&FileResponse{},
		"",
		func(context.Context, http.ResponseWriter) error {
			t.Error("unexpected response encoder call")
			return nil
		},
		func(_ context.Context, w http.ResponseWriter, err error) error {
			encodedErr = err
			w.WriteHeader(http.StatusInternalServerError)
			return nil
		},
		nil,
	)
	lifecycle.End()

	require.EqualError(t, encodedErr, "files.download returned nil file response content")
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Len(t, recorder.events, 2)
	require.Equal(t, loomtransport.ReasonHandlerError, recorder.events[1].Reason)
}
