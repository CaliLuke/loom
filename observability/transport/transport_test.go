package transport_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	mu     sync.Mutex
	events []transport.Event
}

func (r *recordingObserver) ObserveEvent(_ context.Context, e transport.Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recordingObserver) snapshot() []transport.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]transport.Event, len(r.events))
	copy(out, r.events)
	return out
}

func TestObserverNoOpWhenAbsent(t *testing.T) {
	t.Parallel()
	transport.Observe(context.Background(), transport.Event{Kind: transport.EventKindRequestStart})
	require.Nil(t, transport.ObserverFromContext(context.Background()))
}

func TestObserverFuncReceivesEvents(t *testing.T) {
	t.Parallel()
	var got []transport.Event
	fn := transport.ObserverFunc(func(_ context.Context, e transport.Event) {
		got = append(got, e)
	})
	ctx := transport.WithObserver(context.Background(), fn)
	transport.Observe(ctx, transport.Event{Kind: transport.EventKindRequestStart, Transport: transport.TransportHTTP})
	transport.Observe(ctx, transport.Event{Kind: transport.EventKindRequestFinish, Transport: transport.TransportHTTP, Reason: transport.ReasonOK})
	require.Len(t, got, 2)
	require.Equal(t, transport.EventKindRequestStart, got[0].Kind)
	require.Equal(t, transport.ReasonOK, got[1].Reason)
}

func TestObserverFuncNilIsNoOp(t *testing.T) {
	t.Parallel()
	var fn transport.ObserverFunc
	fn.ObserveEvent(context.Background(), transport.Event{Kind: transport.EventKindRequestStart})
}

func TestObserverFromContextReturnsInjectedObserver(t *testing.T) {
	t.Parallel()
	rec := &recordingObserver{}
	ctx := transport.WithObserver(context.Background(), rec)
	got := transport.ObserverFromContext(ctx)
	require.Same(t, rec, got)
	transport.Observe(ctx, transport.Event{Kind: transport.EventKindRequestStart, Reason: transport.ReasonOK})
	events := rec.snapshot()
	require.Len(t, events, 1)
	require.Equal(t, transport.ReasonOK, events[0].Reason)
}

func TestHTTPMiddlewareInjectsObserverIntoRequestContext(t *testing.T) {
	t.Parallel()
	rec := &recordingObserver{}
	var seen transport.Observer
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = transport.ObserverFromContext(r.Context())
		transport.Observe(r.Context(), transport.Event{Kind: transport.EventKindRequestFinish, Reason: transport.ReasonOK})
		w.WriteHeader(http.StatusNoContent)
	})
	mw := transport.HTTPMiddleware(rec)
	srv := httptest.NewServer(mw(handler))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Same(t, rec, seen)
	events := rec.snapshot()
	require.Len(t, events, 1)
	require.Equal(t, transport.EventKindRequestFinish, events[0].Kind)
}

func TestHTTPMiddlewareNilObserverPassesThrough(t *testing.T) {
	t.Parallel()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Nil(t, transport.ObserverFromContext(r.Context()))
		w.WriteHeader(http.StatusOK)
	})
	mw := transport.HTTPMiddleware(nil)
	srv := httptest.NewServer(mw(handler))
	defer srv.Close()
	resp, err := srv.Client().Get(srv.URL)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
