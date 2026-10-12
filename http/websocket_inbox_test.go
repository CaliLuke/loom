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

type (
	lifecyclePairOptions struct {
		limits     loomhttp.WebSocketLimits
		context    context.Context
		cancel     context.CancelFunc
		configure  loomhttp.ConnConfigureFunc
		closeError error
	}
	lifecycleSocket struct {
		net.Conn
		closeError error
		gated      atomic.Bool
		entered    chan struct{}
		released   chan struct{}
		once       sync.Once
	}
)

func TestWebSocketInboxOverload(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limits loomhttp.WebSocketLimits
		frames []string
	}{
		{"byte budget", loomhttp.WebSocketLimits{MaxMessageBytes: 8, MaxBufferedBytes: 16}, []string{"123456", "789"}},
		{"empty message count", loomhttp.WebSocketLimits{MaxMessageBytes: 8, MaxBufferedMessages: 2}, []string{"", "", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{limits: tc.limits, context: ctx, cancel: cancel})
			for _, frame := range tc.frames {
				require.NoError(t, peer.WriteMessage(websocket.TextMessage, []byte(frame)))
			}
			require.NoError(t, peer.SetReadDeadline(time.Now().Add(2*time.Second)))
			_, _, err := peer.ReadMessage()
			require.True(t, websocket.IsCloseError(err, websocket.CloseTryAgainLater), "err = %v", err)
			select {
			case <-ctx.Done():
			case <-time.After(time.Second):
				t.Fatal("overload did not cancel lifetime")
			}
			for range 2 {
				_, err = stream.ReadMessage(context.Background())
				require.ErrorIs(t, err, loomhttp.ErrWebSocketBufferFull)
			}
			require.ErrorIs(t, stream.Conn().NetConn().SetDeadline(time.Now()), net.ErrClosed)
		})
	}
}

func TestWebSocketDecodeRetainsReservation(t *testing.T) {
	for _, failDecode := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "error"}[failDecode], func(t *testing.T) {
			stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{limits: loomhttp.WebSocketLimits{MaxMessageBytes: 4, MaxBufferedBytes: 12}})
			entered, release := make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			defer releaseOnce.Do(func() {
				close(release)
			})
			finished := make(chan error, 1)
			decodeErr := errors.New("decode failed")
			go func() {
				finished <- stream.DecodeMessage(context.Background(), func(data []byte) error {
					if string(data) != "1234" {
						return errors.New("wrong first message")
					}
					close(entered)
					<-release
					if failDecode {
						return decodeErr
					}
					return nil
				})
			}()
			require.NoError(t, peer.WriteMessage(websocket.TextMessage, []byte("1234")))
			select {
			case <-entered:
			case <-time.After(time.Second):
				t.Fatal("decode did not start")
			}
			require.NoError(t, peer.WriteMessage(websocket.TextMessage, []byte("5678")))
			require.NoError(t, peer.WriteMessage(websocket.TextMessage, []byte("9")))
			require.NoError(t, peer.SetReadDeadline(time.Now().Add(2*time.Second)))
			_, _, err := peer.ReadMessage()
			require.True(t, websocket.IsCloseError(err, websocket.CloseTryAgainLater), "err = %v", err)
			releaseOnce.Do(func() {
				close(release)
			})
			if failDecode {
				require.ErrorIs(t, <-finished, decodeErr)
			} else {
				require.NoError(t, <-finished)
			}
			_, err = stream.ReadMessage(context.Background())
			require.ErrorIs(t, err, loomhttp.ErrWebSocketBufferFull)
		})
	}
}

func TestWebSocketInboxReleasesConsumedReservation(t *testing.T) {
	for _, mode := range []string{"message", "decode success", "decode failure"} {
		t.Run(mode, func(t *testing.T) {
			stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{limits: loomhttp.WebSocketLimits{MaxMessageBytes: 4, MaxBufferedMessages: 1}})
			decodeErr := errors.New("decode failed")
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			for range 3 {
				require.NoError(t, peer.WriteMessage(websocket.TextMessage, []byte("1234")))
				if mode == "message" {
					data, err := stream.ReadMessage(ctx)
					require.NoError(t, err)
					require.Equal(t, "1234", string(data))
				} else {
					err := stream.DecodeMessage(ctx, func(data []byte) error {
						require.Equal(t, "1234", string(data))
						if mode == "decode failure" {
							return decodeErr
						}
						return nil
					})
					if mode == "decode failure" {
						require.ErrorIs(t, err, decodeErr)
					} else {
						require.NoError(t, err)
					}
				}
			}
		})
	}
}

func TestWebSocketReaderPreservesConfiguredCloseHandler(t *testing.T) {
	called := make(chan int, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, peer, _ := newLifecyclePair(t, lifecyclePairOptions{context: ctx, cancel: cancel,
		configure: func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
			conn.SetCloseHandler(func(code int, _ string) error {
				called <- code
				return nil
			})
			return conn
		},
	})
	require.NoError(t, peer.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
	select {
	case code := <-called:
		require.Equal(t, websocket.CloseNormalClosure, code)
	case <-time.After(time.Second):
		t.Fatal("configured close handler was not called")
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("configured handler prevented lifetime cancellation")
	}
}

func TestWebSocketInboxNaturalClosePreservesFIFO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{context: ctx, cancel: cancel})
	for _, frame := range []string{"one", "two", "three"} {
		require.NoError(t, peer.WriteMessage(websocket.TextMessage, []byte(frame)))
	}
	require.NoError(t, peer.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("peer close did not cancel lifetime")
	}
	for _, frame := range []string{"one", "two", "three"} {
		data, err := stream.ReadMessage(context.Background())
		require.NoError(t, err)
		require.Equal(t, frame, string(data))
	}
	_, first := stream.ReadMessage(context.Background())
	require.True(t, websocket.IsCloseError(first, websocket.CloseNormalClosure))
	require.NoError(t, stream.Close())
	_, repeated := stream.ReadMessage(context.Background())
	require.Same(t, first, repeated)
}

func TestWebSocketIdleLifetimeCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{context: ctx, cancel: cancel})
	cancel()
	require.NoError(t, peer.SetReadDeadline(time.Now().Add(time.Second)))
	_, _, err := peer.ReadMessage()
	require.Error(t, err)
	_, err = stream.ReadMessage(context.Background())
	require.ErrorIs(t, err, loomhttp.ErrWebSocketStreamClosed)
}

func TestWebSocketFinalJSONCloseBoundsWriterWait(t *testing.T) {
	stream, _, socket := newLifecyclePair(t, lifecyclePairOptions{})
	socket.gated.Store(true)
	written := make(chan error, 1)
	go func() {
		written <- stream.WriteJSON(context.Background(), "blocked")
	}()
	select {
	case <-socket.entered:
	case <-time.After(time.Second):
		t.Fatal("writer did not enter socket")
	}
	closed := make(chan error, 1)
	go func() {
		closed <- stream.CloseWithFinalJSON(nil)
	}()
	select {
	case err := <-closed:
		require.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("final JSON close waited behind application writer")
	}
	require.Error(t, <-written)
	require.ErrorIs(t, socket.SetDeadline(time.Now()), net.ErrClosed)
}

func newLifecyclePair(t *testing.T, opts lifecyclePairOptions) (*loomhttp.WebSocketStream, *websocket.Conn, *lifecycleSocket) {
	t.Helper()
	peers := make(chan *websocket.Conn, 1)
	release := make(chan struct{})
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		peers <- conn
		<-release
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("peer close: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		close(release)
	})
	var socket *lifecycleSocket
	dialer := websocket.Dialer{NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		socket = &lifecycleSocket{Conn: conn, closeError: opts.closeError, entered: make(chan struct{}, 1), released: make(chan struct{})}
		return socket, nil
	}}
	conn, response, err := dialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	config, err := loomhttp.NewWebSocketConfig(opts.limits, opts.configure)
	require.NoError(t, err)
	stream := loomhttp.NewWebSocketStream(config)
	ctx := opts.context
	if ctx == nil {
		ctx = context.Background()
	}
	require.NoError(t, stream.SetConn(ctx, conn, opts.cancel))
	t.Cleanup(func() {
		if err := stream.Close(); err != nil && opts.closeError == nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("stream cleanup: %v", err)
		}
	})
	return stream, <-peers, socket
}

func (c *lifecycleSocket) Write(data []byte) (int, error) {
	if c.gated.Load() {
		select {
		case c.entered <- struct{}{}:
		default:
		}
		<-c.released
	}
	return c.Conn.Write(data)
}

func (c *lifecycleSocket) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() {
		close(c.released)
	})
	return errors.Join(err, c.closeError)
}
