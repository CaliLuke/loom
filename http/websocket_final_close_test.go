package http_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

// A data frame during final close must not make the sole reader exit before
// observing peer termination. The subsequent ping/pong proves reader progress.
func TestWebSocketFinalJSONCloseWaitsForPeer(t *testing.T) {
	for _, abrupt := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "abrupt"}[abrupt], func(t *testing.T) {
			stream, peer, socket := newLifecyclePair(t, lifecyclePairOptions{})
			closed := make(chan error, 1)
			go func() {
				closed <- stream.CloseWithFinalJSON(nil)
			}()
			var final any
			require.NoError(t, peer.ReadJSON(&final))
			require.Nil(t, final)
			pong := make(chan struct{}, 1)
			peer.SetPongHandler(func(string) error {
				pong <- struct{}{}
				return nil
			})
			require.NoError(t, peer.WriteJSON("late response"))
			require.NoError(t, peer.WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)))
			readDone := make(chan error, 1)
			go func() {
				_, _, err := peer.ReadMessage()
				readDone <- err
			}()
			select {
			case <-pong:
			case err := <-closed:
				t.Fatalf("final close returned before peer termination: %v", err)
			case <-time.After(2 * time.Second):
				t.Fatal("sole reader did not process ping after late data")
			}
			if abrupt {
				require.NoError(t, peer.Close())
			} else {
				require.NoError(t, peer.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
			}
			select {
			case err := <-closed:
				require.NoError(t, err)
			case <-time.After(2 * time.Second):
				t.Fatal("final close did not finish after peer termination")
			}
			require.Error(t, <-readDone)
			require.ErrorIs(t, socket.SetDeadline(time.Now()), net.ErrClosed)
			require.NoError(t, stream.Close())
		})
	}
}

func TestWebSocketFinalJSONCloseBoundsPeerWait(t *testing.T) {
	stream, peer, socket := newLifecyclePair(t, lifecyclePairOptions{})
	closed := make(chan error, 1)
	go func() {
		closed <- stream.CloseWithFinalJSON(nil)
	}()
	var final any
	require.NoError(t, peer.ReadJSON(&final))
	// The peer remains open without reading or replying. The final write was
	// successful, so expiration of this best-effort wait adds no Close error.
	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("stalled peer prevented bounded final close")
	}
	require.ErrorIs(t, socket.SetDeadline(time.Now()), net.ErrClosed)
}

func TestWebSocketSimultaneousFinalJSONClose(t *testing.T) {
	stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{})
	config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{}, nil)
	require.NoError(t, err)
	other := loomhttp.NewWebSocketStream(config)
	require.NoError(t, other.SetConn(context.Background(), peer, nil))
	closed := make(chan error, 2)
	start := make(chan struct{})
	for _, endpoint := range []*loomhttp.WebSocketStream{stream, other} {
		go func() {
			<-start
			closed <- endpoint.CloseWithFinalJSON(nil)
		}()
	}
	close(start)
	for range 2 {
		select {
		case err := <-closed:
			require.NoError(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("simultaneous final close deadlocked")
		}
	}
}

// The first service decode is paused while the pump admits the final null.
// A ping after that write establishes FIFO admission without scheduler sleeps.
func TestWebSocketFinalJSONPreservesQueuedInput(t *testing.T) {
	client, peer, socket := newLifecyclePair(t, lifecyclePairOptions{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pumped := make(chan struct{}, 1)
	config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
		conn.SetPingHandler(func(string) error {
			pumped <- struct{}{}
			return nil
		})
		return conn
	})
	require.NoError(t, err)
	server := loomhttp.NewWebSocketStream(config)
	require.NoError(t, server.SetConn(ctx, peer, cancel))
	entered, resume := make(chan struct{}), make(chan struct{})
	defer close(resume)
	finished := make(chan error, 1)
	go func() {
		finished <- server.RunHandler(func() error {
			if err := server.DecodeMessage(ctx, func(data []byte) error {
				if string(data) != "\"first\"\n" {
					return fmt.Errorf("unexpected payload %q", data)
				}
				close(entered)
				<-resume
				return nil
			}); err != nil {
				return err
			}
			var final *string
			if err := server.ReadJSON(ctx, &final); err != nil {
				return err
			}
			if final != nil {
				return fmt.Errorf("expected final null, got %q", *final)
			}
			return nil
		}, nil)
	}()
	require.NoError(t, client.WriteJSON(context.Background(), "first"))
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("service decode did not start")
	}
	socket.gated.Store(true)
	closed := make(chan error, 1)
	go func() {
		closed <- client.CloseWithFinalJSON(nil)
	}()
	select {
	case <-socket.entered:
	case <-time.After(time.Second):
		t.Fatal("final JSON write did not start")
	}
	socket.gated.Store(false)
	socket.once.Do(func() {
		close(socket.released)
	})
	// WriteControl waits behind the in-progress final JSON write. The server
	// sees this ping only after its sole reader has admitted the final null.
	require.NoError(t, client.Conn().WriteControl(websocket.PingMessage, nil, time.Now().Add(time.Second)))
	select {
	case <-pumped:
	case <-ctx.Done():
		t.Fatal("final close canceled the service before queued input was consumed")
	case <-time.After(time.Second):
		t.Fatal("server did not admit the final JSON frame")
	}
	require.NoError(t, ctx.Err())
	resume <- struct{}{}
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("service did not consume queued final null")
	}
	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("client did not observe service completion")
	}
}
