package jsonrpc

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

type (
	closeErrorOperation struct {
		name string
		run  func(context.Context, *WebSocketClientStream) error
	}

	// closeGatedConn lets a real write enter the socket after Close shuts it.
	closeGatedConn struct {
		net.Conn
		gated   atomic.Bool
		entered chan struct{}
		release chan struct{}
		once    sync.Once
	}
)

func TestWebSocketClientClosedOperations(t *testing.T) {
	for _, end := range []string{"stream", "connection", "cancel", "failure"} {
		for _, op := range closeErrorOperations() {
			t.Run(end+"/"+op.name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				srv := newScriptedServer(t)
				conn, peer := dialScripted(t, srv, nil)
				stream := openStream(t, ctx, conn, nil)
				switch end {
				case "stream":
					require.NoError(t, stream.Close())
				case "connection":
					require.NoError(t, conn.Close())
				case "cancel":
					cancel()
					<-stream.done
				case "failure":
					require.NoError(t, peer.Close())
					<-conn.Done()
				}
				err := op.run(t.Context(), stream)
				require.Error(t, err)
				if end == "stream" || end == "connection" {
					require.ErrorIs(t, err, ErrStreamClosed)
				} else {
					require.NotErrorIs(t, err, ErrStreamClosed)
					if end == "cancel" {
						require.ErrorIs(t, err, context.Canceled)
					}
				}
				require.NoError(t, stream.Close())
			})
		}
	}
}

func TestWebSocketClientCloseDuringReceive(t *testing.T) {
	for _, closeStream := range []bool{false, true} {
		name := "connection"
		if closeStream {
			name = "stream"
		}
		t.Run(name, func(t *testing.T) {
			srv := newScriptedServer(t)
			conn, _ := dialScripted(t, srv, nil)
			stream := openStream(t, t.Context(), conn, nil)
			require.NoError(t, stream.Send(t.Context(), "pending"))
			srv.next(t)
			result := make(chan error, 1)
			go func() {
				_, err := stream.Recv(t.Context())
				result <- err
			}()
			require.Eventually(t, func() bool {
				stream.mu.Lock()
				defer stream.mu.Unlock()
				return len(stream.queue) == 0
			}, 5*time.Second, time.Millisecond)
			if closeStream {
				require.NoError(t, stream.Close())
			} else {
				require.NoError(t, conn.Close())
			}
			select {
			case err := <-result:
				require.ErrorIs(t, err, ErrStreamClosed)
			case <-time.After(5 * time.Second):
				t.Fatal("closed receive did not return")
			}
			require.NoError(t, stream.Close())
		})
	}
}

func TestWebSocketClientCloseDuringWrite(t *testing.T) {
	for _, op := range closeErrorOperations()[:3] {
		t.Run(op.name, func(t *testing.T) {
			srv := newScriptedServer(t)
			var socket *closeGatedConn
			dialer := &websocket.Dialer{NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
				if err != nil {
					return nil, err
				}
				socket = &closeGatedConn{Conn: conn, entered: make(chan struct{}, 1), release: make(chan struct{})}
				return socket, nil
			}}
			ws, response, err := dialer.DialContext(t.Context(), srv.url, nil)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			conn := NewWebSocketClientConn(loomhttp.NewWebSocketStream(ws), nil)
			t.Cleanup(func() {
				require.NoError(t, conn.Close())
			})
			stream := openStream(t, t.Context(), conn, nil)
			socket.gated.Store(true)
			result := make(chan error, 1)
			go func() {
				result <- op.run(t.Context(), stream)
			}()
			select {
			case <-socket.entered:
			case <-time.After(5 * time.Second):
				t.Fatal("write did not reach socket")
			}
			require.NoError(t, conn.Close())
			select {
			case err := <-result:
				require.ErrorIs(t, err, ErrStreamClosed)
				require.ErrorIs(t, err, net.ErrClosed, "preserve the underlying write failure")
			case <-time.After(5 * time.Second):
				t.Fatal("closed write did not return")
			}
			require.NoError(t, stream.Close())
		})
	}
}

func TestWebSocketClientWriteFailureBeforeClose(t *testing.T) {
	srv := newScriptedServer(t)
	conn, _ := dialScripted(t, srv, nil)
	stream := openStream(t, t.Context(), conn, nil)
	failure := errors.New("encoding failed")
	require.NoError(t, conn.Close())
	err := stream.writeFailed(failure)
	require.ErrorIs(t, err, failure)
	require.NotErrorIs(t, err, ErrStreamClosed, "an independent encoding failure is not a closure")
	require.NoError(t, stream.Close())
}

func closeErrorOperations() []closeErrorOperation {
	return []closeErrorOperation{
		{"send", func(ctx context.Context, s *WebSocketClientStream) error {
			return s.Send(ctx, "value")
		}},
		{"notify", func(ctx context.Context, s *WebSocketClientStream) error {
			return s.Notify(ctx, "value")
		}},
		{"call", func(ctx context.Context, s *WebSocketClientStream) error {
			_, err := s.Call(ctx, "value")
			return err
		}},
		{"recv", func(ctx context.Context, s *WebSocketClientStream) error {
			_, err := s.Recv(ctx)
			return err
		}},
	}
}

func (c *closeGatedConn) Write(p []byte) (int, error) {
	if c.gated.Load() {
		c.entered <- struct{}{}
		<-c.release
	}
	return c.Conn.Write(p)
}

func (c *closeGatedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() {
		close(c.release)
	})
	return err
}
