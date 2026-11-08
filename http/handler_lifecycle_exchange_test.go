package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
)

type (
	exchangeReader struct {
		read   func([]byte) (int, error)
		closed bool
	}
	exchangeObserver struct {
		observe func(loomtransport.Event)
	}
)

func TestHandlerLifecycleBeginsFinalBeforeHeaderClone(t *testing.T) {
	w := &exchangeTestWriter{header: http.Header{"X-Original": {"keep"}}}
	l := NewHandlerLifecycle(w, httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
	var checked bool
	w.onHeader = func() {
		w.onHeader = nil
		checked = true
		require.ErrorIs(t, l.Exchange().Send(103, nil), ErrInformationalClosed)
		require.False(t, l.responseCommitted())
	}
	require.True(t, l.EncodeResponse(func(_ context.Context, w http.ResponseWriter) error {
		w.WriteHeader(http.StatusCreated)
		return nil
	}, nil, nil))
	require.True(t, checked)
	require.Equal(t, []int{http.StatusCreated}, w.codes)
}

func TestHandlerLifecycleExchangePrecommitFallback(t *testing.T) {
	response := httptest.NewRecorder()
	l := NewHandlerLifecycle(response, httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
	response.Header().Set("X-Original", "keep")
	first, second := errors.New("result encoding"), errors.New("error encoding")
	calls := 0
	require.False(t, l.EncodeResponse(func(_ context.Context, w http.ResponseWriter) error {
		require.ErrorIs(t, l.Exchange().Send(103, nil), ErrInformationalClosed)
		require.False(t, l.responseCommitted())
		w.Header().Set("X-Result", "discard")
		return first
	}, func(_ context.Context, w http.ResponseWriter, err error) error {
		calls++
		require.ErrorIs(t, l.Exchange().Send(103, nil), ErrInformationalClosed)
		require.False(t, l.responseCommitted())
		require.Empty(t, w.Header().Get("X-Result"))
		if calls == 1 {
			require.ErrorIs(t, err, first)
			w.Header().Set("X-Error", "discard")
			return second
		}
		require.ErrorIs(t, err, second)
		require.Empty(t, w.Header().Get("X-Error"))
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}, nil))
	require.Equal(t, 2, calls)
	require.Equal(t, http.StatusInternalServerError, response.Code)
	require.Equal(t, "keep", response.Header().Get("X-Original"))
}

func TestHandlerLifecycleExchangeFinalEntries(t *testing.T) {
	for _, name := range []string{"encode-error", "handler-failure", "committed-handler-failure", "raw-peek", "file", "end", "panic-end"} {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			var l *HandlerLifecycle
			var events []loomtransport.Event
			observer := &exchangeObserver{observe: func(e loomtransport.Event) {
				events = append(events, e)
				if l != nil {
					require.ErrorIs(t, l.Exchange().Send(103, nil), ErrInformationalClosed,
						"observer must run after admission closes and outside its lock")
				}
			}}
			request = request.WithContext(loomtransport.WithObserver(request.Context(), observer))
			l = NewHandlerLifecycle(response, request, "svc", "method")
			require.Same(t, l.observer, loomtransport.RequestObserverFromContext(l.Context()))
			checked := false
			check := func() {
				checked = true
				require.ErrorIs(t, l.Exchange().Send(103, nil), ErrInformationalClosed)
			}
			encodeError := func(_ context.Context, w http.ResponseWriter, _ error) error {
				check()
				w.WriteHeader(http.StatusBadRequest)
				return nil
			}
			switch name {
			case "encode-error":
				l.DecodeFailed(errors.New("decode"), encodeError, nil)
			case "handler-failure":
				l.HandlerFailed(errors.New("endpoint"), false, encodeError, nil)
			case "committed-handler-failure":
				l.Writer().WriteHeader(http.StatusCreated)
				l.HandlerFailed(errors.New("late"), true, encodeError, func(context.Context, http.ResponseWriter, error) {
					check()
				})
			case "raw-peek":
				body := &exchangeReader{read: func([]byte) (int, error) {
					check()
					require.False(t, l.responseCommitted())
					return 0, errors.New("peek")
				}}
				l.WriteRawBody(body, nil, encodeError, nil)
				require.True(t, body.closed)
			case "file":
				l.ServeFile(request, &FileResponse{Content: strings.NewReader("file")}, "text/plain",
					func(context.Context, http.ResponseWriter) error {
						check()
						require.False(t, l.responseCommitted())
						return nil
					}, encodeError, nil)
			case "end":
				l.End()
				check()
				require.False(t, l.responseCommitted(), "End is admission cleanup, not publication")
			case "panic-end":
				require.PanicsWithValue(t, "original", func() {
					defer l.End()
					panic("original")
				})
				check()
				require.False(t, l.responseCommitted())
			}
			require.True(t, checked)
			l.End()
			require.Len(t, events, 2, "preserve one request observer and one terminal observation")
		})
	}
}

func TestHandlerLifecycleExchangeStagedInformationalFailure(t *testing.T) {
	for _, returnedError := range []bool{false, true} {
		name := "nil-encoder-error"
		if returnedError {
			name = "encoder-error-also-retained"
		}
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			response.Header().Set("X-Original", "keep")
			l := NewHandlerLifecycle(response, httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
			body := &trackedReadCloser{Reader: strings.NewReader("must-not-write")}
			ownErr := errors.New("encoder")
			calls := 0
			l.WriteRawBody(body, func(_ context.Context, w http.ResponseWriter) error {
				w.Header().Set("X-Staged", "must-not-leak")
				w.WriteHeader(http.StatusEarlyHints)
				w.WriteHeader(http.StatusCreated)
				if returnedError {
					return ownErr
				}
				return nil
			}, func(_ context.Context, w http.ResponseWriter, err error) error {
				calls++
				require.ErrorIs(t, err, ErrInformationalClosed)
				if returnedError {
					require.ErrorIs(t, err, ownErr)
				}
				require.False(t, l.responseCommitted())
				require.Empty(t, w.Header().Get("X-Staged"))
				w.WriteHeader(http.StatusInternalServerError)
				return nil
			}, nil)
			require.Equal(t, 1, calls)
			require.True(t, body.closed)
			require.Equal(t, http.StatusInternalServerError, response.Code)
			require.Empty(t, response.Body.String())
			require.Empty(t, response.Header().Get("X-Staged"))
			require.Equal(t, "keep", response.Header().Get("X-Original"))
		})
	}
}

func TestHandlerLifecycleStatusViolationRoutesToErrorEncoder(t *testing.T) {
	l := NewHandlerLifecycle(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
	key := StatusSelector{Code: http.StatusCreated}
	encoded := false
	require.False(t, l.EncodeResponse(func(context.Context, http.ResponseWriter) error {
		return CheckFinalStatus([]StatusSelector{key}, key, http.StatusOK, false)
	}, func(_ context.Context, w http.ResponseWriter, err error) error {
		encoded = true
		var violation *StatusViolationError
		require.ErrorAs(t, err, &violation)
		require.Equal(t, http.StatusOK, violation.Code)
		require.False(t, l.responseCommitted())
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}, nil))
	require.True(t, encoded)
}

func TestHandlerLifecycleErrorOnlyNilResultIsViolation(t *testing.T) {
	l := NewHandlerLifecycle(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil), "svc", "method")
	encoded := false
	require.False(t, l.EncodeResponse(func(context.Context, http.ResponseWriter) error {
		return &StatusViolationError{}
	}, func(_ context.Context, w http.ResponseWriter, err error) error {
		encoded = true
		var violation *StatusViolationError
		require.ErrorAs(t, err, &violation)
		require.Zero(t, violation.Code)
		require.False(t, l.responseCommitted())
		w.WriteHeader(http.StatusInternalServerError)
		return nil
	}, nil))
	require.True(t, encoded, "runtime routes a simulated generated error-only violation")
}

func (r *exchangeReader) Read(b []byte) (int, error) {
	return r.read(b)
}

func (r *exchangeReader) Close() error {
	r.closed = true
	return nil
}

func (o *exchangeObserver) ObserveEvent(_ context.Context, e loomtransport.Event) {
	o.observe(e)
}
