package tests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	sse "github.com/tmaxmax/go-sse"
)

// collectExternalSSE adapts the external client to the shared completion owner.
func collectExternalSSE(request *http.Request, expected int) ([]sse.Event, error) {
	return collectExternalSSEEvents(request.Context(), expected, func(ctx context.Context, receive func(sse.Event)) error {
		connection := sse.NewConnection(request.WithContext(ctx))
		unsubscribe := connection.SubscribeToAll(receive)
		defer unsubscribe()
		return connection.Connect()
	})
}

// collectExternalSSEEvents gives the observer sole ownership of cancellation and
// connection completion. Callbacks publish events before signaling; whichever
// signal wakes the observer, it joins the connection and examines the complete
// event snapshot. Connect must stop on context cancellation and finish callbacks
// before returning, as the external client's Connect contract requires.
func collectExternalSSEEvents(parent context.Context, expected int, connect func(context.Context, func(sse.Event)) error) ([]sse.Event, error) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	var (
		mu     sync.Mutex
		events []sse.Event
	)
	received := make(chan struct{})
	ended := make(chan error, 1)
	receive := func(event sse.Event) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
		if len(events) == expected {
			close(received)
		}
	}
	go func() {
		ended <- connect(ctx, receive)
	}()
	var (
		connectErr error
		joined     bool
	)
	select {
	case <-received:
	case connectErr = <-ended:
		joined = true
	case <-ctx.Done():
	}
	cancel()
	if !joined {
		connectErr = <-ended
	}
	mu.Lock()
	snapshot := append([]sse.Event(nil), events...)
	mu.Unlock()
	if len(snapshot) != expected {
		cause := errors.Join(connectErr, context.Cause(parent))
		if cause != nil {
			return snapshot, fmt.Errorf("expected %d events, received %d: %w", expected, len(snapshot), cause)
		}
		return snapshot, fmt.Errorf("expected %d events, received %d before connection closed", expected, len(snapshot))
	}
	if connectErr != nil && !errors.Is(connectErr, context.Canceled) && !errors.Is(connectErr, context.DeadlineExceeded) {
		return snapshot, fmt.Errorf("external SSE connection: %w", connectErr)
	}
	return snapshot, nil
}
