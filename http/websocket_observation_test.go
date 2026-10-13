package http_test

import (
	"context"
	"sync"
	"testing"
	"time"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestWebSocketObservationLifecycle(t *testing.T) {
	for _, name := range []string{"normal", "late attachment", "rejected configuration", "client isolation", "reentrant close", "observer panic", "handler panic"} {
		t.Run(name, func(t *testing.T) {
			conn, cleanup := newIdleWebSocketConn(t)
			defer cleanup()
			var mu sync.Mutex
			var events []loomtransport.Event
			var stream *loomhttp.WebSocketStream
			ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
				mu.Lock()
				events = append(events, event)
				mu.Unlock()
				if event.Kind == loomtransport.EventKindStreamClose && name == "handler panic" {
					panic("close observer")
				}
				if event.Kind == loomtransport.EventKindStreamOpen {
					switch name {
					case "reentrant close":
						require.NoError(t, stream.Close())
					case "observer panic":
						panic("open observer")
					}
				}
			}))
			observer := loomtransport.BeginRequest(ctx, loomtransport.TransportHTTP, "chat", "listen")
			ctx = loomtransport.WithRequestObserver(ctx, observer)
			var configure loomhttp.ConnConfigureFunc
			if name == "rejected configuration" {
				configure = func(*websocket.Conn, context.CancelFunc) *websocket.Conn {
					return nil
				}
			}
			config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{}, configure)
			require.NoError(t, err)
			if name == "client isolation" {
				stream = loomhttp.NewWebSocketStream(config)
			} else {
				stream = loomhttp.NewServerWebSocketStream(ctx, loomtransport.TransportHTTP, config)
			}
			if name == "late attachment" {
				require.NoError(t, stream.Close())
			}
			if name == "observer panic" {
				require.PanicsWithValue(t, "open observer", func() {
					if err := stream.RunHandler(func() error {
						return stream.SetConn(ctx, conn, nil)
					}, nil); err != nil {
						t.Errorf("handler: %v", err)
					}
				})
			} else {
				err = stream.SetConn(ctx, conn, nil)
				if name == "late attachment" || name == "rejected configuration" {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}
			if name == "handler panic" {
				require.PanicsWithValue(t, "handler", func() {
					if err := stream.RunHandler(func() error {
						panic("handler")
					}, nil); err != nil {
						t.Errorf("handler: %v", err)
					}
				})
			}
			require.NoError(t, stream.Close())
			require.NoError(t, stream.Close())
			observer.End()
			mu.Lock()
			defer mu.Unlock()
			var kinds []loomtransport.EventKind
			for _, event := range events {
				if event.Kind != loomtransport.EventKindStreamOpen && event.Kind != loomtransport.EventKindStreamClose {
					continue
				}
				kinds = append(kinds, event.Kind)
				require.Equal(t, "chat", event.Service)
				require.Equal(t, "listen", event.Method)
			}
			if name == "client isolation" {
				require.Empty(t, kinds)
			} else {
				require.Equal(t, []loomtransport.EventKind{loomtransport.EventKindStreamOpen, loomtransport.EventKindStreamClose}, kinds)
			}
		})
	}
}

// TestWebSocketObservationBackgroundRelease checks that cancellation releases
// the socket without invoking observer code on its background callback stack.
func TestWebSocketObservationBackgroundRelease(t *testing.T) {
	var mu sync.Mutex
	var kinds []loomtransport.EventKind
	ctx, cancel := context.WithCancel(loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
		mu.Lock()
		kinds = append(kinds, event.Kind)
		mu.Unlock()
	})))
	defer cancel()
	stream, _, socket := newLifecyclePair(t, lifecyclePairOptions{context: ctx, cancel: cancel, serverObservation: true})
	cancel()
	select {
	case <-socket.released:
	case <-time.After(3 * time.Second):
		t.Fatal("physical socket not released")
	}
	mu.Lock()
	require.Equal(t, []loomtransport.EventKind{loomtransport.EventKindStreamOpen}, kinds)
	mu.Unlock()
	require.NoError(t, stream.Close())
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []loomtransport.EventKind{loomtransport.EventKindStreamOpen, loomtransport.EventKindStreamClose}, kinds)
}
