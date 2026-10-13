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

func TestStreamObservationReadiness(t *testing.T) {
	for _, tc := range []struct {
		name string
		open bool
		fail bool
	}{
		{name: "opened stream", open: true},
		{name: "failed opening flush", open: true, fail: true},
		{name: "never opened control"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var events []loomtransport.Event
			ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, e loomtransport.Event) {
				events = append(events, e)
			}))
			request := httptest.NewRequest(http.MethodGet, "/ticks", nil).WithContext(ctx)
			request.Pattern = "GET /ticks"
			var writer http.ResponseWriter = httptest.NewRecorder()
			if tc.fail {
				writer = &failingFlushWriter{header: make(http.Header), err: errors.New("flush failed")}
			}
			lifecycle := NewHandlerLifecycle(writer, request, "clock", "tick")
			stream := NewSSEStreamWriter(lifecycle.Writer(), lifecycle.Context(), loomtransport.TransportHTTP, StreamWritePolicy{})
			if tc.open {
				err := stream.Open(lifecycle.Context())
				if tc.fail {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}
			require.NoError(t, stream.Close())
			lifecycle.End()
			opened, closed, failed := 0, 0, 0
			for _, event := range events {
				switch event.Kind {
				case loomtransport.EventKindStreamOpen:
					opened++
				case loomtransport.EventKindStreamClose:
					closed++
				case loomtransport.EventKindStreamFailure:
					failed++
				default:
					continue
				}
				if event.Service != "clock" || event.Method != "tick" || event.Route != "GET /ticks" || event.HTTPMethod != http.MethodGet {
					t.Errorf("stream event lost request identity: %+v", event)
				}
			}
			if !tc.open {
				require.Zero(t, opened)
				require.Zero(t, closed)
				require.Zero(t, failed)
				return
			}
			if opened != 1 {
				t.Errorf("opened=%d, want1", opened)
			}
			if !tc.fail && closed != 1 {
				t.Errorf("closed=%d, want1", closed)
			}
			if tc.fail {
				require.Equal(t, 1, failed)
			}
		})
	}
}

// TestSSEObservationCallbackOwnership checks recoverable failures and callback
// execution outside the writer's lock without blocking a test goroutine.
func TestSSEObservationCallbackOwnership(t *testing.T) {
	var stream *SSEStreamWriter
	var events []loomtransport.Event
	callbackLocked := false
	ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
		events = append(events, event)
		if event.Kind == loomtransport.EventKindStreamFailure {
			if stream.lock.TryLock() {
				stream.lock.Unlock()
			} else {
				callbackLocked = true
			}
		}
	}))
	request := httptest.NewRequest(http.MethodGet, "/ticks", nil).WithContext(ctx)
	lifecycle := NewHandlerLifecycle(httptest.NewRecorder(), request, "clock", "tick")
	stream = NewSSEStreamWriter(lifecycle.Writer(), lifecycle.Context(), loomtransport.TransportHTTP, StreamWritePolicy{})
	failure := errors.New("encoding failed before writing")
	require.ErrorIs(t, stream.WriteEvent(ctx, func(io.Writer) error {
		return failure
	}), failure)
	require.NoError(t, stream.SendComment(ctx, "healthy"))
	require.NoError(t, stream.Close())
	lifecycle.End()
	require.False(t, callbackLocked, "observer callbacks may reenter Started/Close")
	kinds := make([]loomtransport.EventKind, 0, len(events))
	for _, event := range events {
		kinds = append(kinds, event.Kind)
	}
	require.Equal(t, []loomtransport.EventKind{
		loomtransport.EventKindRequestStart,
		loomtransport.EventKindStreamOpen,
		loomtransport.EventKindStreamFailure,
		loomtransport.EventKindStreamClose,
		loomtransport.EventKindRequestFinish,
	}, kinds)
}

func TestSSETerminalPreparationOwnership(t *testing.T) {
	ctx := context.Background()
	writer := httptest.NewRecorder()
	stream := NewSSEStreamWriter(writer, ctx, loomtransport.TransportHTTP, StreamWritePolicy{})
	failure := errors.New("encoding failed")
	require.ErrorIs(t, stream.Complete(ctx, func() (SSEMessage, error) {
		return SSEMessage{}, failure
	}), failure)
	require.False(t, stream.Started())
	require.NoError(t, stream.SendComment(ctx, "still-open"))
	require.NoError(t, stream.Complete(ctx, func() (SSEMessage, error) {
		return SSEMessage{Data: "final"}, nil
	}))
	require.ErrorIs(t, stream.Complete(ctx, func() (SSEMessage, error) {
		t.Error("duplicate final prepared")
		return SSEMessage{}, nil
	}), ErrSSEStreamClosed)
	require.ErrorIs(t, stream.SendComment(ctx, "late"), ErrSSEStreamClosed)
	require.Equal(t, ": still-open\n\ndata: final\n\n", writer.Body.String())
}

func TestSSEHandlerEndPanicPrecedence(t *testing.T) {
	var kinds []loomtransport.EventKind
	ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
		kinds = append(kinds, event.Kind)
		if event.Kind == loomtransport.EventKindStreamClose {
			panic("close observer")
		}
	}))
	stream := NewSSEStreamWriter(httptest.NewRecorder(), ctx, loomtransport.TransportHTTP, StreamWritePolicy{})
	require.PanicsWithValue(t, "handler", func() {
		defer stream.End()
		require.NoError(t, stream.Open(ctx))
		panic("handler")
	})
	require.Equal(t, []loomtransport.EventKind{loomtransport.EventKindStreamOpen, loomtransport.EventKindStreamClose}, kinds)
	require.NoError(t, stream.Close())
}

// TestSSEConcurrentTerminalOwnership forces final preparation to hold admission
// while an ordinary send and a second final operation compete for the writer.
func TestSSEConcurrentTerminalOwnership(t *testing.T) {
	ctx := context.Background()
	rec := httptest.NewRecorder()
	stream := NewSSEStreamWriter(rec, ctx, loomtransport.TransportHTTP, StreamWritePolicy{})
	prepared := make(chan struct{})
	release := make(chan struct{})
	first, second, ordinary := make(chan error, 1), make(chan error, 1), make(chan error, 1)
	go func() {
		first <- stream.Complete(ctx, func() (SSEMessage, error) {
			close(prepared)
			<-release
			return SSEMessage{Data: "final"}, nil
		})
	}()
	<-prepared
	go func() {
		second <- stream.Complete(ctx, func() (SSEMessage, error) {
			return SSEMessage{Data: "duplicate"}, nil
		})
	}()
	go func() {
		ordinary <- stream.SendComment(ctx, "late")
	}()
	close(release)
	require.NoError(t, <-first)
	require.ErrorIs(t, <-second, ErrSSEStreamClosed)
	require.ErrorIs(t, <-ordinary, ErrSSEStreamClosed)
	require.Equal(t, "data: final\n\n", rec.Body.String())
}
