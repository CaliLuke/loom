package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

func TestSSEHandlerFailureClaimsBeforeRetainedWrite(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/events", nil)
	lifecycle := NewHandlerLifecycle(response, request, "events", "watch")
	defer lifecycle.End()
	stream := NewSSEStreamWriter(lifecycle.Writer(), lifecycle.Context(), loomtransport.TransportHTTP, StreamWritePolicy{})
	ready := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		<-ready
		finished <- stream.SendComment(lifecycle.Context(), "late")
	}()
	endpointErr := errors.New("endpoint failed")
	const prefix = `{"error":"endpoint"}`
	var retainedErr error
	var encoded, handled int
	func() {
		defer stream.End()
		lifecycle.HandlerFailed(endpointErr, stream.ClaimFailure(), func(_ context.Context, w http.ResponseWriter, err error) error {
			encoded++
			require.ErrorIs(t, err, endpointErr)
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusBadRequest)
			if _, writeErr := io.WriteString(w, prefix); writeErr != nil {
				return writeErr
			}
			close(ready)
			retainedErr = <-finished
			return nil
		}, func(context.Context, http.ResponseWriter, error) {
			handled++
		})
	}()
	require.Equal(t, 1, encoded)
	require.Zero(t, handled)
	require.Equal(t, http.StatusBadRequest, response.Code)
	require.Equal(t, "application/problem+json", response.Header().Get("Content-Type"))
	require.ErrorIs(t, retainedErr, ErrSSEStreamClosed)
	require.Equal(t, prefix, response.Body.String())
	require.False(t, stream.Started())
	require.False(t, response.Flushed)
	require.Empty(t, response.Header().Get("X-Accel-Buffering"))
	require.ErrorIs(t, stream.SendComment(lifecycle.Context(), "after End"), ErrSSEStreamClosed)
}

func TestSSEStreamWriterClaimFailure(t *testing.T) {
	failure := errors.New("prior transport failure")
	for _, tc := range []struct {
		name                         string
		prepare                      func(*testing.T, *SSEStreamWriter)
		opened, failed, flushFailure bool
	}{
		{name: "fresh"},
		{name: "suppressed", prepare: func(t *testing.T, s *SSEStreamWriter) {
			require.NoError(t, s.Suppress())
		}},
		{name: "cleanup", prepare: func(_ *testing.T, s *SSEStreamWriter) {
			s.End()
		}},
		{name: "opened", opened: true, prepare: func(t *testing.T, s *SSEStreamWriter) {
			require.NoError(t, s.Open(t.Context()))
		}},
		{name: "successful close", opened: true, prepare: func(t *testing.T, s *SSEStreamWriter) {
			require.NoError(t, s.Close())
		}},
		{name: "empty completion", opened: true, prepare: func(t *testing.T, s *SSEStreamWriter) {
			require.NoError(t, s.Complete(t.Context(), nil))
		}},
		{name: "completed", opened: true, prepare: func(t *testing.T, s *SSEStreamWriter) {
			require.NoError(t, s.Complete(t.Context(), func() (SSEMessage, error) {
				return SSEMessage{Data: "done"}, nil
			}))
		}},
		{name: "preparation failure", prepare: func(t *testing.T, s *SSEStreamWriter) {
			require.ErrorIs(t, s.Complete(t.Context(), func() (SSEMessage, error) {
				return SSEMessage{}, failure
			}), failure)
		}},
		{name: "untouched callback failure", opened: true, prepare: func(t *testing.T, s *SSEStreamWriter) {
			require.ErrorIs(t, s.WriteEvent(t.Context(), func(io.Writer) error {
				return failure
			}), failure)
		}},
		{name: "touched write failure", opened: true, failed: true, prepare: func(t *testing.T, s *SSEStreamWriter) {
			require.ErrorIs(t, s.WriteEvent(t.Context(), func(w io.Writer) error {
				if _, err := io.WriteString(w, "partial"); err != nil {
					return err
				}
				return failure
			}), failure)
		}},
		{name: "flush failure", opened: true, failed: true, flushFailure: true, prepare: func(t *testing.T, s *SSEStreamWriter) {
			require.ErrorIs(t, s.Open(t.Context()), failure)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			var w http.ResponseWriter = rec
			if tc.flushFailure {
				w = &failingFlushWriter{header: make(http.Header), err: failure}
			}
			s := NewSSEStreamWriter(w, t.Context(), loomtransport.TransportHTTP, StreamWritePolicy{})
			if tc.prepare != nil {
				tc.prepare(t, s)
			}
			headers, body, flushed := w.Header().Clone(), rec.Body.String(), rec.Flushed
			headerCalls, flushes := 0, 0
			if tc.flushFailure {
				failed := w.(*failingFlushWriter)
				headerCalls, flushes = failed.writeHeaderCalls, failed.flushes
			}
			cause, opening := s.failErr, s.openErr
			require.Equal(t, tc.opened, s.ClaimFailure())
			require.Equal(t, tc.opened, s.ClaimFailure())
			require.Equal(t, tc.opened, s.Started())
			require.Equal(t, cause, s.failErr)
			require.Equal(t, opening, s.openErr)
			require.Equal(t, headers, w.Header())
			if tc.flushFailure {
				failed := w.(*failingFlushWriter)
				require.Equal(t, headerCalls, failed.writeHeaderCalls)
				require.Equal(t, flushes, failed.flushes)
			} else {
				require.Equal(t, body, rec.Body.String())
				require.Equal(t, flushed, rec.Flushed)
			}
			err := s.SendComment(t.Context(), "refused")
			require.ErrorIs(t, err, ErrSSEStreamClosed)
			if tc.failed {
				require.ErrorIs(t, err, failure)
			} else {
				require.NotErrorIs(t, err, failure)
			}
			called := false
			require.ErrorIs(t, s.Complete(t.Context(), func() (SSEMessage, error) {
				called = true
				return SSEMessage{Data: "refused"}, nil
			}), ErrSSEStreamClosed)
			require.False(t, called)
			require.ErrorIs(t, s.Open(t.Context()), ErrSSEStreamClosed)
			require.Equal(t, headers, w.Header())
			if tc.flushFailure {
				failed := w.(*failingFlushWriter)
				require.Equal(t, headerCalls, failed.writeHeaderCalls)
				require.Equal(t, flushes, failed.flushes)
			} else {
				require.Equal(t, body, rec.Body.String())
				require.Equal(t, flushed, rec.Flushed)
			}
		})
	}
}

func TestSSEStreamWriterClaimFailurePreservesContextPrecedence(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	s := NewSSEStreamWriter(httptest.NewRecorder(), ctx, loomtransport.TransportHTTP, StreamWritePolicy{})
	cancel()
	require.False(t, s.ClaimFailure())
	require.True(t, s.closed)
	require.Nil(t, s.failErr)
	require.ErrorIs(t, s.SendComment(t.Context(), "valid"), context.Canceled)
	require.ErrorIs(t, s.SendComment(t.Context(), "invalid\ncomment"), ErrInvalidSSEComment)
}

func TestSSEHandlerFailureEncoderFallback(t *testing.T) {
	for _, fallbackFails := range []bool{false, true} {
		t.Run(map[bool]string{false: "fallback commits", true: "fallback fails"}[fallbackFails], func(t *testing.T) {
			rec := httptest.NewRecorder()
			lifecycle := NewHandlerLifecycle(rec, httptest.NewRequest(http.MethodGet, "/events", nil), "events", "watch")
			defer lifecycle.End()
			s := NewSSEStreamWriter(lifecycle.Writer(), lifecycle.Context(), loomtransport.TransportHTTP, StreamWritePolicy{})
			defer s.End()
			endpointErr, encodingErr, fallbackErr := errors.New("endpoint"), errors.New("encoding"), errors.New("fallback")
			encoded, handled := 0, 0
			lifecycle.HandlerFailed(endpointErr, s.ClaimFailure(), func(_ context.Context, w http.ResponseWriter, err error) error {
				encoded++
				require.ErrorIs(t, s.SendComment(t.Context(), "late"), ErrSSEStreamClosed)
				if encoded == 1 {
					require.ErrorIs(t, err, endpointErr)
					w.Header().Set("X-Transient", "discard")
					return encodingErr
				}
				require.ErrorIs(t, err, encodingErr)
				require.Empty(t, w.Header().Get("X-Transient"))
				if fallbackFails {
					return fallbackErr
				}
				w.WriteHeader(http.StatusInternalServerError)
				_, writeErr := io.WriteString(w, "fallback")
				return writeErr
			}, func(_ context.Context, _ http.ResponseWriter, err error) {
				handled++
				require.ErrorIs(t, err, fallbackErr)
			})
			require.Equal(t, 2, encoded)
			if fallbackFails {
				require.Equal(t, 1, handled)
				require.Empty(t, rec.Body.String())
			} else {
				require.Zero(t, handled)
				require.Equal(t, http.StatusInternalServerError, rec.Code)
				require.Equal(t, "fallback", rec.Body.String())
			}
			require.False(t, s.Started())
			require.False(t, rec.Flushed)
		})
	}
}

func TestSSEHandlerFailureWaitsForAdmittedWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	lifecycle := NewHandlerLifecycle(rec, httptest.NewRequest(http.MethodGet, "/events", nil), "events", "watch")
	defer lifecycle.End()
	s := NewSSEStreamWriter(lifecycle.Writer(), lifecycle.Context(), loomtransport.TransportHTTP, StreamWritePolicy{})
	defer s.End()
	held, release := make(chan struct{}), make(chan struct{})
	written := make(chan error, 1)
	go func() {
		written <- s.WriteEvent(t.Context(), func(w io.Writer) error {
			close(held)
			<-release
			_, err := io.WriteString(w, "data: admitted\n\n")
			return err
		})
	}()
	<-held
	claim := make(chan bool, 1)
	go func() {
		claim <- s.ClaimFailure()
	}()
	close(release)
	require.NoError(t, <-written)
	started := <-claim
	require.True(t, started)
	encoded, handled := 0, 0
	lifecycle.HandlerFailed(errors.New("endpoint"), started, func(context.Context, http.ResponseWriter, error) error {
		encoded++
		return nil
	}, func(context.Context, http.ResponseWriter, error) {
		handled++
	})
	require.Zero(t, encoded)
	require.Equal(t, 1, handled)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "data: admitted\n\n", rec.Body.String())
	require.ErrorIs(t, s.SendComment(t.Context(), "late"), ErrSSEStreamClosed)
}

// Post-frame cancellation with supported deadline control leaves no terminal I/O
// cause; failure admission must still preserve the already-started response.
func TestSSEFailureClaimAfterPostFrameCancellation(t *testing.T) {
	writer := &deadlineResponseWriter{header: make(http.Header)}
	s := NewSSEStreamWriter(writer, t.Context(), loomtransport.TransportHTTP, StreamWritePolicy{})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	require.ErrorIs(t, s.WriteEvent(ctx, func(w io.Writer) error {
		if _, err := io.WriteString(w, "data: buffered\n\n"); err != nil {
			return err
		}
		cancel()
		return nil
	}), context.Canceled)
	require.False(t, s.closed)
	require.Nil(t, s.failErr)
	require.True(t, s.ClaimFailure())
	require.True(t, s.ClaimFailure())
	require.Nil(t, s.failErr)
	require.ErrorIs(t, s.SendComment(t.Context(), "late"), ErrSSEStreamClosed)
	require.Equal(t, "data: buffered\n\n", writer.body.String())
	require.Zero(t, writer.flushes)
}
