package jsonrpc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

// TestServeWebSocketDisconnectCancelsDispatch reproduces #722 without a
// generated adapter: synchronous dispatch must not stop lifetime observation.
func TestServeWebSocketDisconnectCancelsDispatch(t *testing.T) {
	for _, name := range []string{"normal", "abrupt", "overload"} {
		t.Run(name, func(t *testing.T) {
			started := make(chan struct{})
			ended := make(chan error, 1)
			escape := make(chan struct{})
			config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxBufferedMessages: 1}, nil)
			require.NoError(t, err)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ServeWebSocket(w, r, WebSocketHandlerSpec{
					Upgrader: &websocket.Upgrader{},
					Config:   config,
					Run: func(ctx context.Context, _ context.CancelFunc, _ *http.Request, _ http.ResponseWriter, stream *loomhttp.WebSocketStream) error {
						return ReceiveWebSocketRequest(ctx, stream,
							func(method string) bool {
								return method == "wait"
							},
							func(ctx context.Context, _ *RawRequest) error {
								close(started)
								select {
								case <-ctx.Done():
									ended <- ctx.Err()
								case <-escape:
									ended <- nil
								}
								return nil
							},
							func(context.Context, any, Code, string, any) error {
								t.Error("unexpected protocol error")
								return nil
							})
					},
					HandleFailure: func(_ context.Context, _ http.ResponseWriter, err error) {
						t.Errorf("handler failure: %v", err)
					},
				})
			}))
			defer server.Close()
			defer close(escape)
			conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			defer func() {
				if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
					t.Errorf("client close: %v", err)
				}
			}()
			require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(`{"jsonrpc":"2.0","method":"wait","id":1}`)))
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("dispatch did not start")
			}
			switch name {
			case "overload":
				for range 2 {
					require.NoError(t, conn.WriteMessage(websocket.TextMessage, nil))
				}
			case "normal":
				require.NoError(t, conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
			default:
				require.NoError(t, conn.Close())
			}
			select {
			case err := <-ended:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(time.Second):
				t.Error("peer disconnect did not cancel the synchronous service dispatch")
			}
		})
	}
}
