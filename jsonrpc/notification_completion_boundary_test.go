package jsonrpc

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestNotificationCompletionPreservesTransportFailures prevents request-error
// suppression from swallowing dispatch I/O or protocol-error response failures.
func TestNotificationCompletionPreservesTransportFailures(t *testing.T) {
	for _, tc := range []struct {
		name  string
		frame string
		match bool
	}{
		{"notification-dispatch-io", `{"jsonrpc":"2.0","method":"call"}`, true},
		{"request-dispatch-io", `{"jsonrpc":"2.0","method":"call","id":"one"}`, true},
		{"unknown-error-write", `{"jsonrpc":"2.0","method":"missing","id":"one"}`, false},
		{"parse-error-write", `{"jsonrpc":`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := errors.New("transport failed")
			result := make(chan error, 1)
			upgrader := websocket.Upgrader{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					result <- err
					return
				}
				stream := newRuntimeWebSocketStream(t, conn)
				defer func() {
					if err := stream.Close(); err != nil {
						t.Errorf("close stream: %v", err)
					}
				}()
				result <- ReceiveWebSocketRequest(r.Context(), stream,
					func(string) bool {
						return tc.match
					},
					func(context.Context, *RawRequest) error {
						return want
					},
					func(context.Context, any, Code, string, any) error {
						return want
					},
				)
			}))
			defer server.Close()
			conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			defer func() {
				require.NoError(t, conn.Close())
			}()
			require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(tc.frame)))
			select {
			case err := <-result:
				require.ErrorIs(t, err, want)
			case <-time.After(3 * time.Second):
				t.Fatal("transport failure did not return")
			}
		})
	}
}
