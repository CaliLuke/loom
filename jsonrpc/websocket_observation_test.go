package jsonrpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	loomhttp "github.com/CaliLuke/loom/http"
	loomtransport "github.com/CaliLuke/loom/observability/transport"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestWebSocketObservationOperationIdentity(t *testing.T) {
	var mu sync.Mutex
	var events []loomtransport.Event
	ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	}))
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		r.Pattern = "GET /rpc"
		ServeWebSocket(w, r.WithContext(ctx), WebSocketHandlerSpec{
			Service:  "clock",
			Upgrader: &websocket.Upgrader{},
			Run: func(ctx context.Context, _ context.CancelFunc, _ *http.Request, _ http.ResponseWriter, stream *loomhttp.WebSocketStream) error {
				var bindings []*loomtransport.StreamObserver
				for range 2 {
					if err := ReceiveWebSocketRequest(ctx, stream, func(string) bool {
						return true
					}, func(ctx context.Context, _ *RawRequest) error {
						bindings = append(bindings, loomtransport.StreamObserverFromContext(ctx, loomtransport.TransportJSONRPC))
						return nil
					}, func(context.Context, any, Code, string, any) error {
						t.Error("unexpected protocol error")
						return nil
					}); err != nil {
						return err
					}
				}
				// Both dispatch calls returned. A retained first wrapper must not
				// acquire the second operation's identity or its caller's context.
				for _, binding := range bindings {
					if err := CompleteStream(loomtransport.WithStreamObserver(context.Background(), binding), false, nil, nil, func(*Response) error {
						t.Error("notification response was sent")
						return nil
					}); err != nil {
						return err
					}
				}
				return stream.WriteJSON(ctx, MakeNotification("still-open", "ok"))
			},
			HandleFailure: func(_ context.Context, _ http.ResponseWriter, err error) {
				t.Errorf("handler failure: %v", err)
			},
		})
	}))
	defer server.Close()
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	defer func() {
		require.NoError(t, conn.Close())
	}()
	for _, method := range []string{"first", "second"} {
		require.NoError(t, conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "method": method}))
	}
	var notification map[string]any
	require.NoError(t, conn.ReadJSON(&notification))
	require.Equal(t, "still-open", notification["method"])
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("handler did not finish")
	}
	mu.Lock()
	defer mu.Unlock()
	var physical []loomtransport.EventKind
	var logical []string
	for _, event := range events {
		require.Equal(t, "clock", event.Service)
		require.Equal(t, "GET /rpc", event.Route)
		if event.Reason == loomtransport.ReasonStreamFinalResponseSuppressed {
			logical = append(logical, event.Method)
			require.True(t, event.Notification)
			continue
		}
		if event.Kind == loomtransport.EventKindStreamOpen || event.Kind == loomtransport.EventKindStreamClose {
			physical = append(physical, event.Kind)
			require.Empty(t, event.Method, "physical identity is connection-level")
			require.Empty(t, event.JSONRPCID)
		}
	}
	require.Equal(t, []string{"first", "second"}, logical)
	require.Equal(t, []loomtransport.EventKind{loomtransport.EventKindStreamOpen, loomtransport.EventKindStreamClose}, physical)
}

// TestWebSocketOpeningObserverPanicReleasesConnection checks the eager
// attachment path before the generated service handler has been entered.
func TestWebSocketOpeningObserverPanicReleasesConnection(t *testing.T) {
	panicked := make(chan any, 1)
	ctx := loomtransport.WithObserver(context.Background(), loomtransport.ObserverFunc(func(_ context.Context, event loomtransport.Event) {
		if event.Kind == loomtransport.EventKindStreamOpen {
			panic("opening observer")
		}
	}))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			panicked <- recover()
		}()
		ServeWebSocket(w, r.WithContext(ctx), WebSocketHandlerSpec{
			Service:  "clock",
			Upgrader: &websocket.Upgrader{},
			Run: func(context.Context, context.CancelFunc, *http.Request, http.ResponseWriter, *loomhttp.WebSocketStream) error {
				t.Error("service ran after opening observer panic")
				return nil
			},
		})
	}))
	defer server.Close()
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	defer func() {
		require.NoError(t, conn.Close())
	}()
	require.Equal(t, "opening observer", <-panicked)
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
	_, _, err = conn.ReadMessage()
	require.True(t, websocket.IsCloseError(err, websocket.CloseInternalServerErr), "native cleanup: %v", err)
}
