package http_test

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
)

// TestWebSocketStreamTerminalReadReleasesSocket covers the runtime owner of
// #712: recording a terminal read error must also release the local socket.
func TestWebSocketStreamTerminalReadReleasesSocket(t *testing.T) {
	for _, kind := range []string{"normal", "abrupt", "deadline"} {
		t.Run(kind, func(t *testing.T) {
			release := make(chan struct{})
			upgrader := websocket.Upgrader{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Errorf("upgrade: %v", err)
					return
				}
				defer func() {
					if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
						t.Errorf("server close: %v", err)
					}
				}()
				switch kind {
				case "normal":
					err = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
					if err != nil {
						t.Errorf("close control: %v", err)
					}
				case "deadline":
					<-release
				}
			}))
			defer server.Close()
			defer close(release)
			conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			if kind == "deadline" {
				require.NoError(t, conn.SetReadDeadline(time.Now().Add(20*time.Millisecond)))
			}
			stream := newConfiguredWebSocketStream(t, conn)
			defer func() {
				require.NoError(t, stream.Close())
			}()

			_, first := stream.ReadMessage(context.Background())
			require.Error(t, first)
			_, second := stream.ReadMessage(context.Background())
			require.ErrorIs(t, second, first, "terminal cause must survive resource release")
			if err := conn.NetConn().SetDeadline(time.Now()); !errors.Is(err, net.ErrClosed) {
				t.Errorf("terminal %s read left local socket open: deadline error %v", kind, err)
			}
		})
	}
}
