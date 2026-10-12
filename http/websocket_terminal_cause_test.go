package http_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

type readObservedConn struct {
	net.Conn
	armed   atomic.Bool
	once    sync.Once
	started chan struct{}
}

func TestWebSocketStreamLocalCloseOwnsTerminalCause(t *testing.T) {
	for _, action := range []string{"cancel active read", "close active read", "close after failed read"} {
		t.Run(action, func(t *testing.T) {
			stream, started, _ := newObservedWebSocketStream(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := stream.ReadMessage(ctx)
				result <- err
			}()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("read did not reach the socket")
			}
			switch action {
			case "cancel active read":
				cancel()
			case "close active read":
				require.NoError(t, stream.Close())
			case "close after failed read":
				require.NoError(t, stream.Conn().SetReadDeadline(time.Now()))
			}
			select {
			case err := <-result:
				if action == "cancel active read" {
					require.ErrorIs(t, err, context.Canceled)
				} else {
					require.Error(t, err)
				}
			case <-time.After(time.Second):
				t.Fatal("read did not unblock")
			}
			if action == "close after failed read" {
				require.NoError(t, stream.Close())
			}
			for range 3 {
				_, err := stream.ReadMessage(context.Background())
				require.ErrorIs(t, err, loomhttp.ErrWebSocketStreamClosed)
			}
		})
	}
}

func TestWebSocketStreamAlreadyCanceledReadPreservesConnection(t *testing.T) {
	stream, started, peer := newObservedWebSocketStream(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := stream.ReadMessage(ctx)
	require.ErrorIs(t, err, context.Canceled)
	select {
	case <-started:
		t.Fatal("already canceled read reached the socket")
	default:
	}
	require.NoError(t, peer.WriteMessage(websocket.TextMessage, []byte("valid")))
	data, err := stream.ReadMessage(context.Background())
	require.NoError(t, err)
	require.Equal(t, "valid", string(data))
}

func TestWebSocketStreamOverflowOwnsTerminalCause(t *testing.T) {
	stream, _, peer := newObservedWebSocketStream(t)
	require.NoError(t, peer.WriteMessage(websocket.BinaryMessage, []byte(strings.Repeat("x", 129))))
	_, err := stream.ReadMessage(context.Background())
	require.ErrorIs(t, err, websocket.ErrReadLimit)
	require.NoError(t, stream.Close())
	_, err = stream.ReadMessage(context.Background())
	require.ErrorIs(t, err, websocket.ErrReadLimit)
}

func newObservedWebSocketStream(t *testing.T) (*loomhttp.WebSocketStream, <-chan struct{}, *websocket.Conn) {
	t.Helper()
	peers := make(chan *websocket.Conn, 1)
	release := make(chan struct{})
	done := make(chan struct{})
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(done)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		peers <- conn
		<-release
		if err := conn.Close(); err != nil {
			t.Errorf("close peer: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	var observed *readObservedConn
	dialer := websocket.Dialer{NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		observed = &readObservedConn{Conn: conn, started: make(chan struct{})}
		return observed, nil
	}}
	conn, response, err := dialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	config, err := loomhttp.NewWebSocketConfig(128, nil)
	require.NoError(t, err)
	stream := loomhttp.NewWebSocketStream(config)
	require.NoError(t, stream.SetConn(conn, nil))
	t.Cleanup(func() {
		if err := stream.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close stream: %v", err)
		}
		close(release)
		<-done
	})
	observed.armed.Store(true)
	return stream, observed.started, <-peers
}

func (c *readObservedConn) Read(p []byte) (int, error) {
	if c.armed.Load() {
		c.once.Do(func() {
			close(c.started)
		})
	}
	return c.Conn.Read(p)
}
