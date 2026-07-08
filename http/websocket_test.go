package http_test

import (
	"context"
	"errors"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

func TestWebSocketStreamCloseUnblocksRead(t *testing.T) {
	stream, cleanup := newIdleWebSocketStream(t)
	defer cleanup()

	errc := make(chan error, 1)
	go func() {
		var msg map[string]string
		errc <- stream.ReadJSON(context.Background(), &msg)
	}()

	require.Eventually(t, func() bool {
		return stream.Close() == nil
	}, time.Second, 10*time.Millisecond)

	select {
	case err := <-errc:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("ReadJSON did not unblock after Close")
	}
}

func TestWebSocketStreamReadHonorsContextCancellation(t *testing.T) {
	stream, cleanup := newIdleWebSocketStream(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		var msg map[string]string
		errc <- stream.ReadJSON(ctx, &msg)
	}()
	cancel()

	select {
	case err := <-errc:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("ReadJSON did not unblock after context cancellation")
	}
}

func TestWebSocketStreamCloseIsIdempotent(t *testing.T) {
	stream, cleanup := newIdleWebSocketStream(t)
	defer cleanup()

	require.NoError(t, stream.Close())
	require.NoError(t, stream.Close())
}

func TestWebSocketStreamNilConnection(t *testing.T) {
	stream := loomhttp.NewWebSocketStream(nil)
	require.NoError(t, stream.Close())

	var msg map[string]string
	err := stream.ReadJSON(context.Background(), &msg)
	require.True(t, errors.Is(err, loomhttp.ErrWebSocketStreamClosed))
}

func newIdleWebSocketStream(t *testing.T) (*loomhttp.WebSocketStream, func()) {
	t.Helper()

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer func() {
			require.NoError(t, conn.Close())
		}()
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}
	return loomhttp.NewWebSocketStream(conn), server.Close
}
