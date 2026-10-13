package transport

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStreamObserverLifecycle(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(map[bool]string{false: "reentrant close", true: "panic releases delivery"}[panics], func(t *testing.T) {
			var stream *StreamObserver
			var kinds []EventKind
			ctx := WithObserver(context.Background(), ObserverFunc(func(_ context.Context, event Event) {
				kinds = append(kinds, event.Kind)
				if event.Kind == EventKindStreamOpen {
					stream.RecordClose()
					stream.Drain()
					if panics {
						panic("observer")
					}
				}
			}))
			stream = NewStreamObserver(ctx, TransportHTTP)
			stream.RecordClose()
			stream.Drain()
			require.Empty(t, kinds)
			stream.RecordOpen()
			if panics {
				require.PanicsWithValue(t, "observer", stream.Drain)
			} else {
				stream.Drain()
			}
			stream.RecordOpen()
			stream.RecordClose()
			stream.Drain()
			require.Equal(t, []EventKind{EventKindStreamOpen, EventKindStreamClose}, kinds)
		})
	}
}

func TestStreamObserverRetainedIdentity(t *testing.T) {
	var events []Event
	ctx := WithObserver(context.Background(), ObserverFunc(func(_ context.Context, event Event) {
		events = append(events, event)
	}))
	request := BeginRequest(ctx, TransportJSONRPC, "clock", "")
	request.SetJSONRPC("first", "1", 0, false)
	first := NewStreamObserver(WithRequestObserver(ctx, request), TransportJSONRPC)
	second := first.JSONRPC("second", "", true)
	request.End()
	first.Failure(ReasonStreamWriteFailed)
	second.Suppressed()
	require.Equal(t, "first", events[len(events)-2].Method)
	require.Equal(t, "1", events[len(events)-2].JSONRPCID)
	require.Equal(t, "second", events[len(events)-1].Method)
	require.True(t, events[len(events)-1].Notification)
	require.Equal(t, "clock", events[len(events)-1].Service)
}

func TestStreamObserverBoundedLifecycle(t *testing.T) {
	opened, release := make(chan struct{}), make(chan struct{})
	var mu sync.Mutex
	var events []EventKind
	ctx := WithObserver(context.Background(), ObserverFunc(func(_ context.Context, event Event) {
		mu.Lock()
		events = append(events, event.Kind)
		mu.Unlock()
		if event.Kind == EventKindStreamOpen {
			close(opened)
			<-release
		}
	}))
	stream := NewStreamObserver(ctx, TransportHTTP)
	stream.RecordOpen()
	done := make(chan struct{})
	go func() {
		defer close(done)
		stream.Drain()
	}()
	<-opened
	stream.RecordClose()
	stream.Drain()
	for range 10 {
		stream.Failure(ReasonStreamWriteFailed)
	}
	mu.Lock()
	require.Len(t, events, 11, "failures are synchronous, not queued behind opening")
	mu.Unlock()
	close(release)
	<-done
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, EventKindStreamOpen, events[0])
	require.Equal(t, EventKindStreamClose, events[len(events)-1])
	require.Len(t, events, 12)
}
