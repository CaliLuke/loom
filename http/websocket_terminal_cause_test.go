package http_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

type admittedReadContext struct {
	context.Context
	entered chan struct{}
	once    sync.Once
}

func (c *admittedReadContext) Err() error {
	err := c.Context.Err()
	c.once.Do(func() {
		close(c.entered)
	})
	return err
}

func TestWebSocketStreamLocalCloseOwnsTerminalCause(t *testing.T) {
	for _, action := range []string{"cancel active read", "close active read"} {
		t.Run(action, func(t *testing.T) {
			stream, _, _ := newLifecyclePair(t, lifecyclePairOptions{limits: loomhttp.WebSocketLimits{MaxMessageBytes: 128}})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			active := &admittedReadContext{Context: ctx, entered: make(chan struct{})}
			result := make(chan error, 1)
			go func() {
				_, err := stream.ReadMessage(active)
				result <- err
			}()
			select {
			case <-active.entered:
			case <-time.After(time.Second):
				t.Fatal("read did not pass context preflight")
			}
			switch action {
			case "cancel active read":
				cancel()
			case "close active read":
				require.NoError(t, stream.Close())
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
			for range 3 {
				_, err := stream.ReadMessage(context.Background())
				require.ErrorIs(t, err, loomhttp.ErrWebSocketStreamClosed)
			}
		})
	}
}

func TestWebSocketStreamAlreadyCanceledReadPreservesConnection(t *testing.T) {
	stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{limits: loomhttp.WebSocketLimits{MaxMessageBytes: 128}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := stream.ReadMessage(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, peer.WriteMessage(websocket.TextMessage, []byte("valid")))
	data, err := stream.ReadMessage(context.Background())
	require.NoError(t, err)
	require.Equal(t, "valid", string(data))
}

func TestWebSocketStreamOverflowOwnsTerminalCause(t *testing.T) {
	stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{limits: loomhttp.WebSocketLimits{MaxMessageBytes: 128}})
	require.NoError(t, peer.WriteMessage(websocket.BinaryMessage, []byte(strings.Repeat("x", 129))))
	_, err := stream.ReadMessage(context.Background())
	require.ErrorIs(t, err, websocket.ErrReadLimit)
	require.NoError(t, stream.Close())
	_, err = stream.ReadMessage(context.Background())
	require.ErrorIs(t, err, websocket.ErrReadLimit)
}
