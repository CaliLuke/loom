package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type failingCloseValue struct {
	err error
}

func TestWebSocketCloseAfterPeerAcknowledgement(t *testing.T) {
	for _, finalJSON := range []bool{false, true} {
		for _, cleanupFailure := range []bool{false, true} {
			name := "normal"
			var cleanupErr error
			if cleanupFailure {
				name = "physical-cleanup-error"
				cleanupErr = errors.New("physical cleanup failed")
			}
			if finalJSON {
				name += "/final-json"
			} else {
				name += "/close-control"
			}
			t.Run(name, func(t *testing.T) {
				acknowledged, release := make(chan struct{}), make(chan struct{})
				var once sync.Once
				unblock := func() {
					once.Do(func() {
						close(release)
					})
				}
				defer unblock()
				stream, peer, _ := newLifecyclePair(t, lifecyclePairOptions{
					closeError: cleanupErr,
					configure: func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
						handler := conn.CloseHandler()
						conn.SetCloseHandler(func(code int, text string) error {
							err := handler(code, text)
							close(acknowledged)
							<-release
							return err
						})
						return conn
					},
				})
				require.NoError(t, peer.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second)))
				select {
				case <-acknowledged:
				case <-time.After(2 * time.Second):
					t.Fatal("reader did not acknowledge peer close")
				}
				// The peer-close handler has emitted a close frame, but the owner
				// has not yet received NextReader's terminal result.
				require.ErrorIs(t, stream.WriteJSON(context.Background(), "late"), websocket.ErrCloseSent)
				closed := make(chan error, 1)
				go func() {
					if finalJSON {
						closed <- stream.CloseWithFinalJSON(nil)
					} else {
						closed <- stream.CloseWithStatus(websocket.CloseNormalClosure, "")
					}
				}()
				select {
				case err := <-closed:
					if cleanupFailure {
						require.ErrorIs(t, err, cleanupErr)
						require.NotErrorIs(t, err, websocket.ErrCloseSent)
					} else {
						require.NoError(t, err)
					}
				case <-time.After(2 * time.Second):
					t.Fatal("cleanup waited for a blocked peer-close handler")
				}
				unblock()
				if cleanupFailure {
					require.ErrorIs(t, stream.Close(), cleanupErr)
				} else {
					require.NoError(t, stream.Close())
				}
			})
		}
	}
}

func TestWebSocketFinalJSONPreservesWrappedEncodingError(t *testing.T) {
	encodingErr := errors.New("application encoding failed")
	cleanupErr := errors.New("independent physical cleanup failed")
	stream, _, _ := newLifecyclePair(t, lifecyclePairOptions{closeError: cleanupErr})
	err := stream.CloseWithFinalJSON(failingCloseValue{errors.Join(websocket.ErrCloseSent, encodingErr)})
	require.ErrorIs(t, err, encodingErr)
	require.ErrorIs(t, err, cleanupErr)
	require.ErrorIs(t, err, websocket.ErrCloseSent)
	var marshalErr *json.MarshalerError
	require.ErrorAs(t, err, &marshalErr)
	require.ErrorIs(t, stream.Close(), encodingErr)
	require.ErrorIs(t, stream.Close(), cleanupErr)
}

func (v failingCloseValue) MarshalJSON() ([]byte, error) {
	return nil, v.err
}
