package http_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

func TestWebSocketRunHandlerCompletion(t *testing.T) {
	for _, failed := range []bool{false, true} {
		name, wantCode := "success", websocket.CloseNormalClosure
		if failed {
			name, wantCode = "error", websocket.CloseInternalServerErr
		}
		t.Run(name, func(t *testing.T) {
			stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{})
			var handlerErr error
			if failed {
				handlerErr = errors.New("private service failure")
			}
			err := stream.RunHandler(func() error {
				return handlerErr
			}, nil)
			if failed {
				require.Same(t, handlerErr, err)
			} else {
				require.NoError(t, err)
			}
			_, _, err = peer.ReadMessage()
			var closeErr *websocket.CloseError
			require.ErrorAs(t, err, &closeErr)
			require.Equal(t, wantCode, closeErr.Code)
			require.NotContains(t, closeErr.Text, "private")
		})
	}
}

func TestWebSocketRunHandlerCleanupError(t *testing.T) {
	for _, reporter := range []bool{false, true} {
		name := "joined"
		if reporter {
			name = "reported"
		}
		t.Run(name, func(t *testing.T) {
			cleanupErr := errors.New("socket cleanup failed")
			handlerErr := errors.New("endpoint failed")
			stream, _, _ := newLifecyclePair(t, lifecyclePairOptions{closeError: cleanupErr})
			var report func(error)
			var reported error
			if reporter {
				report = func(err error) {
					reported = err
				}
			}
			err := stream.RunHandler(func() error {
				return handlerErr
			}, report)
			require.ErrorIs(t, err, handlerErr)
			if reporter {
				require.Same(t, handlerErr, err)
				require.ErrorIs(t, reported, cleanupErr)
			} else {
				require.ErrorIs(t, err, cleanupErr)
			}
		})
	}
}

func TestWebSocketRunHandlerPanicPrecedence(t *testing.T) {
	for _, reportPanics := range []bool{false, true} {
		name := "report succeeds"
		if reportPanics {
			name = "report panics"
		}
		t.Run(name, func(t *testing.T) {
			original := errors.New("private original panic")
			cleanupErr := errors.New("physical close failure")
			stream, peer, socket := newLifecyclePair(t, lifecyclePairOptions{closeError: cleanupErr})
			reported := false
			require.PanicsWithValue(t, original, func() {
				err := stream.RunHandler(func() error {
					panic(original)
				}, func(err error) {
					require.ErrorIs(t, err, cleanupErr)
					reported = true
					if reportPanics {
						panic("reporter panic")
					}
				})
				t.Errorf("handler unexpectedly returned %v", err)
			})
			require.True(t, reported)
			require.ErrorIs(t, socket.SetDeadline(time.Now()), net.ErrClosed)
			_, _, err := peer.ReadMessage()
			var closeErr *websocket.CloseError
			require.ErrorAs(t, err, &closeErr)
			require.Equal(t, websocket.CloseInternalServerErr, closeErr.Code)
			require.Equal(t, "handler failed", closeErr.Text)
		})
	}
}

func TestWebSocketCloseStatusValidation(t *testing.T) {
	for _, tc := range []struct {
		code   int
		reason string
	}{
		{1005, ""}, {1006, ""}, {1015, ""}, {2999, ""}, {5000, ""}, {1000, strings.Repeat("x", 124)}, {1000, string([]byte{0xff})},
	} {
		stream, _, socket := newLifecyclePair(t, lifecyclePairOptions{})
		require.Error(t, stream.CloseWithStatus(tc.code, tc.reason))
		require.ErrorIs(t, socket.SetDeadline(time.Now()), net.ErrClosed)
	}
}

func TestWebSocketAttachmentCancellation(t *testing.T) {
	for _, action := range []string{"already canceled", "cancel during hook", "close inside hook"} {
		t.Run(action, func(t *testing.T) {
			conn, cleanup := newIdleWebSocketConn(t)
			defer cleanup()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var stream *loomhttp.WebSocketStream
			called := false
			config, err := loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{}, func(c *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
				called = true
				if action == "close inside hook" {
					require.NoError(t, stream.Close())
				} else {
					cancel()
				}
				return c
			})
			require.NoError(t, err)
			stream = loomhttp.NewWebSocketStream(config)
			if action == "already canceled" {
				cancel()
			}
			err = stream.SetConn(ctx, conn, cancel)
			require.Error(t, err)
			require.Equal(t, action != "already canceled", called)
			require.ErrorIs(t, conn.NetConn().SetDeadline(time.Now()), net.ErrClosed)
			require.NoError(t, stream.Close())
		})
	}
}
