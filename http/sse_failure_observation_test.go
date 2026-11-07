package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	loomtransport "github.com/CaliLuke/loom/observability/transport"
	"github.com/stretchr/testify/require"
)

func TestSSEFailureClaimDefersObservation(t *testing.T) {
	for _, opened := range []bool{false, true} {
		t.Run(map[bool]string{false: "unopened", true: "opened"}[opened], func(t *testing.T) {
			var stream *SSEStreamWriter
			var order []string
			var closed int
			ctx := loomtransport.WithObserver(t.Context(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
				if event.Kind == loomtransport.EventKindStreamClose {
					closed++
					order = append(order, "close")
					// Observation remains outside the writer lock and may reenter it.
					require.Equal(t, opened, stream.Started())
					require.NoError(t, stream.Close())
				}
				if event.Kind == loomtransport.EventKindRequestFailure {
					order = append(order, "request failure")
				}
			}))
			request := httptest.NewRequest(http.MethodGet, "/events", nil).WithContext(ctx)
			lifecycle := NewHandlerLifecycle(httptest.NewRecorder(), request, "events", "watch")
			stream = NewSSEStreamWriter(lifecycle.Writer(), lifecycle.Context(), loomtransport.TransportHTTP, StreamWritePolicy{})
			if opened {
				require.NoError(t, stream.Open(ctx))
			}
			claimed := stream.ClaimFailure()
			require.Empty(t, order, "claim must not dispatch observer callbacks")
			require.Zero(t, closed)
			func() {
				defer stream.End()
				lifecycle.HandlerFailed(errors.New("endpoint"), claimed, func(context.Context, http.ResponseWriter, error) error {
					order = append(order, "encode")
					return nil
				}, func(context.Context, http.ResponseWriter, error) {
					order = append(order, "handler")
				})
			}()
			lifecycle.End()
			if opened {
				require.Equal(t, []string{"handler", "close", "request failure"}, order)
				require.Equal(t, 1, closed)
			} else {
				require.Equal(t, []string{"encode", "request failure"}, order)
				require.Zero(t, closed)
			}
			require.NoError(t, stream.Close())
			if opened {
				require.Equal(t, 1, closed)
			}
		})
	}
}

func TestSSEFailureClaimPreservesEndPanic(t *testing.T) {
	ctx := loomtransport.WithObserver(t.Context(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
		if event.Kind == loomtransport.EventKindStreamClose {
			panic("observer")
		}
	}))
	stream := NewSSEStreamWriter(httptest.NewRecorder(), ctx, loomtransport.TransportHTTP, StreamWritePolicy{})
	require.NoError(t, stream.Open(ctx))
	require.NotPanics(t, func() {
		require.True(t, stream.ClaimFailure())
	})
	require.PanicsWithValue(t, "handler", func() {
		defer stream.End()
		panic("handler")
	})
}
