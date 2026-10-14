package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	sse "github.com/tmaxmax/go-sse"
)

func TestExternalSSECompletion(t *testing.T) {
	terminal := errors.New("transport failure")
	for _, tc := range []struct {
		name       string
		count      int
		result     error
		wantError  error
		wrongCount bool
	}{
		{name: "completed then EOF", count: 2},
		{name: "completed then cancellation", count: 2, result: context.Canceled},
		{name: "completed at deadline", count: 2, result: context.DeadlineExceeded},
		{name: "premature EOF", count: 1, wrongCount: true},
		{name: "unexpected terminal failure", count: 2, result: terminal, wantError: terminal},
		{name: "extra event remains visible", count: 3, wrongCount: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			exited := make(chan struct{})
			events, err := collectExternalSSEEvents(context.Background(), 2, func(_ context.Context, receive func(sse.Event)) error {
				defer close(exited)
				for range tc.count {
					receive(sse.Event{Type: "message", Data: "value"})
				}
				return tc.result
			})
			require.Len(t, events, tc.count)
			select {
			case <-exited:
			default:
				t.Fatal("observer returned before connection completion")
			}
			switch {
			case tc.wantError != nil:
				require.ErrorIs(t, err, tc.wantError)
			case tc.wrongCount:
				require.ErrorContains(t, err, "expected 2 events")
			default:
				require.NoError(t, err)
			}
		})
	}
}

func TestExternalSSECompletionOwnsCancellation(t *testing.T) {
	exited := make(chan struct{})
	events, err := collectExternalSSEEvents(context.Background(), 1, func(ctx context.Context, receive func(sse.Event)) error {
		defer close(exited)
		receive(sse.Event{Type: "message", Data: "done"})
		<-ctx.Done()
		return ctx.Err()
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	select {
	case <-exited:
	default:
		t.Fatal("connection was not joined")
	}
}

func TestExternalSSECompletionRetainsIncompleteCancellation(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		ctx, cancel := context.WithCancelCause(context.Background())
		expected := context.Canceled
		if deadline {
			expected = context.DeadlineExceeded
		}
		events, err := collectExternalSSEEvents(ctx, 2, func(child context.Context, receive func(sse.Event)) error {
			receive(sse.Event{Type: "message", Data: "partial"})
			cancel(expected)
			<-child.Done()
			return context.Cause(child)
		})
		cancel(nil)
		require.Len(t, events, 1)
		require.ErrorContains(t, err, "expected 2 events")
		require.ErrorIs(t, err, expected)
	}
}
