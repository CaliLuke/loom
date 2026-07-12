package codegen_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

// TestHTTPObserver pins the observable contract generated HTTP handlers must
// honor: a Start event at request entry, exactly one terminal event before
// return, panic recovery through ReasonPanic with re-panic preserved, and
// response StatusCode/BytesWritten captured on the terminal event.
//
// The codegen tests under TestHandlerInit assert the literal generated text;
// this test exercises the runtime contract by composing the same building
// blocks (BeginHTTPRequest, obs.Fail, defer obs.End) the generator emits,
// against a real HTTP server.
func TestHTTPObserver(t *testing.T) {
	t.Run("success path emits start and finish", func(t *testing.T) {
		rec := newRecorder()
		srv := newGeneratedShapedServer(rec, func(ctx context.Context, w http.ResponseWriter, r *http.Request, obs *transport.RequestObserver) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		})
		defer srv.Close()
		resp, err := srv.Client().Get(srv.URL + "/foo")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		evs := rec.snapshot()
		require.Len(t, evs, 2)
		require.Equal(t, transport.EventKindRequestStart, evs[0].Kind)
		require.Equal(t, transport.EventKindRequestFinish, evs[1].Kind)
		require.Equal(t, transport.ReasonOK, evs[1].Reason)
		require.Equal(t, http.StatusOK, evs[1].StatusCode)
		require.EqualValues(t, 11, evs[1].BytesWritten)
	})

	t.Run("decode failure classifies terminal as request_decode_failed", func(t *testing.T) {
		rec := newRecorder()
		srv := newGeneratedShapedServer(rec, func(ctx context.Context, w http.ResponseWriter, r *http.Request, obs *transport.RequestObserver) {
			obs.Fail(transport.ReasonRequestDecodeFailed)
			w.WriteHeader(http.StatusBadRequest)
		})
		defer srv.Close()
		resp, err := srv.Client().Get(srv.URL + "/foo")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		evs := rec.snapshot()
		require.Equal(t, transport.EventKindRequestFailure, evs[1].Kind)
		require.Equal(t, transport.ReasonRequestDecodeFailed, evs[1].Reason)
		require.Equal(t, http.StatusBadRequest, evs[1].StatusCode)
	})

	t.Run("handler error classifies as handler_error", func(t *testing.T) {
		rec := newRecorder()
		srv := newGeneratedShapedServer(rec, func(ctx context.Context, w http.ResponseWriter, r *http.Request, obs *transport.RequestObserver) {
			obs.Fail(transport.ReasonHandlerError)
			http.Error(w, "boom", http.StatusInternalServerError)
		})
		defer srv.Close()
		resp, err := srv.Client().Get(srv.URL + "/foo")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		evs := rec.snapshot()
		require.Equal(t, transport.ReasonHandlerError, evs[1].Reason)
		require.Equal(t, http.StatusInternalServerError, evs[1].StatusCode)
	})

	t.Run("response write failure classifies as response_write_failed", func(t *testing.T) {
		rec := newRecorder()
		srv := newGeneratedShapedServer(rec, func(ctx context.Context, w http.ResponseWriter, r *http.Request, obs *transport.RequestObserver) {
			obs.Fail(transport.ReasonResponseWriteFailed)
			w.WriteHeader(http.StatusInternalServerError)
		})
		defer srv.Close()
		resp, err := srv.Client().Get(srv.URL + "/foo")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, transport.ReasonResponseWriteFailed, rec.snapshot()[1].Reason)
	})

	t.Run("panic re-propagates and emits ReasonPanic", func(t *testing.T) {
		rec := newRecorder()
		var caught any
		var caughtMu sync.Mutex
		handler := wrapGeneratedShaped(rec, func(ctx context.Context, w http.ResponseWriter, r *http.Request, obs *transport.RequestObserver) {
			panic(errors.New("kaboom"))
		})
		recoverer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				caughtMu.Lock()
				caught = recover()
				caughtMu.Unlock()
			}()
			handler.ServeHTTP(w, r)
		})
		srv := httptest.NewServer(recoverer)
		defer srv.Close()
		resp, err := srv.Client().Get(srv.URL + "/foo")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		caughtMu.Lock()
		require.NotNil(t, caught, "panic must propagate past obs.End")
		caughtMu.Unlock()
		evs := rec.snapshot()
		require.Equal(t, transport.EventKindRequestFailure, evs[1].Kind)
		require.Equal(t, transport.ReasonPanic, evs[1].Reason)
	})
}

type recorder struct {
	mu   sync.Mutex
	evts []transport.Event
}

func newRecorder() *recorder {
	return &recorder{}
}

func (r *recorder) ObserveEvent(_ context.Context, e transport.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.evts = append(r.evts, e)
}

func (r *recorder) snapshot() []transport.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]transport.Event, len(r.evts))
	copy(out, r.evts)
	return out
}

// wrapGeneratedShaped composes the same observer wiring the codegen emits in
// `serverHandlerInitSource`: BeginHTTPRequest, defer obs.End, and an inner
// handler that runs the test-specific logic against the request observer.
func wrapGeneratedShaped(rec transport.Observer, inner func(ctx context.Context, w http.ResponseWriter, r *http.Request, obs *transport.RequestObserver)) http.Handler {
	return transport.HTTPMiddleware(rec)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		obs, ww := transport.BeginHTTPRequest(ctx, w, "Service", "Method", r)
		defer obs.End()
		inner(ctx, ww, r, obs)
	}))
}

func newGeneratedShapedServer(rec transport.Observer, inner func(ctx context.Context, w http.ResponseWriter, r *http.Request, obs *transport.RequestObserver)) *httptest.Server {
	return httptest.NewServer(wrapGeneratedShaped(rec, inner))
}
