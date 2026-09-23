package jsonrpc

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

type (
	// webSocketReceiveOutcome records what ReceiveWebSocketRequest did on the
	// server side of one test connection.
	webSocketReceiveOutcome struct {
		errs       []error
		errorCodes []Code
		dispatched []string
	}

	// webSocketReceiveCase describes one ReceiveWebSocketRequest scenario.
	webSocketReceiveCase struct {
		name string
		// setup runs on the upgraded server connection before the first Recv.
		setup func(conn *websocket.Conn, cancel context.CancelFunc)
		// client drives the client side of the connection.
		client func(t *testing.T, conn *websocket.Conn)
		// recvs is the maximum number of Recv calls.
		recvs int
		// keepReading keeps calling Recv after an error instead of stopping
		// at the first one.
		keepReading bool
		// check asserts the terminal error of the Recv loop.
		check      func(t *testing.T, err error)
		wantCodes  []Code
		wantMethod []string
	}
)

func TestReceiveWebSocketRequestClassifiesReadFailures(t *testing.T) {
	validRequest := `{"jsonrpc":"2.0","method":"echo","id":1}`
	tests := []webSocketReceiveCase{
		{
			name:   "normal closure ends the stream cleanly",
			client: closeWith(websocket.CloseNormalClosure),
			recvs:  1,
			check: func(t *testing.T, err error) {
				require.ErrorIs(t, err, io.EOF)
			},
		},
		{
			name:   "going away is returned as a close error",
			client: closeWith(websocket.CloseGoingAway),
			recvs:  1,
			check: func(t *testing.T, err error) {
				require.True(t, websocket.IsCloseError(err, websocket.CloseGoingAway), "err = %v", err)
			},
		},
		{
			name: "abrupt TCP close is returned as an abnormal closure",
			client: func(t *testing.T, conn *websocket.Conn) {
				require.NoError(t, conn.NetConn().Close())
			},
			recvs: 1,
			check: func(t *testing.T, err error) {
				require.True(t, websocket.IsCloseError(err, websocket.CloseAbnormalClosure), "err = %v", err)
			},
		},
		{
			name: "read deadline expiry is returned as a timeout",
			setup: func(conn *websocket.Conn, _ context.CancelFunc) {
				if err := conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
					panic(err)
				}
			},
			recvs: 1,
			check: func(t *testing.T, err error) {
				var netErr net.Error
				require.ErrorAs(t, err, &netErr)
				require.True(t, netErr.Timeout())
			},
		},
		{
			name: "context cancellation is returned as-is",
			setup: func(_ *websocket.Conn, cancel context.CancelFunc) {
				time.AfterFunc(50*time.Millisecond, cancel)
			},
			recvs: 1,
			check: func(t *testing.T, err error) {
				require.ErrorIs(t, err, context.Canceled)
			},
		},
		{
			name:       "malformed JSON sends a parse error and keeps reading",
			client:     sendFrames(`{"jsonrpc":`, validRequest),
			recvs:      2,
			check:      requireNoReceiveError,
			wantCodes:  []Code{ParseError},
			wantMethod: []string{"echo"},
		},
		{
			name:       "empty message sends a parse error",
			client:     sendFrames("", validRequest),
			recvs:      2,
			check:      requireNoReceiveError,
			wantCodes:  []Code{ParseError},
			wantMethod: []string{"echo"},
		},
		{
			name:       "whitespace-only message sends a parse error",
			client:     sendFrames(" \n\t ", validRequest),
			recvs:      2,
			check:      requireNoReceiveError,
			wantCodes:  []Code{ParseError},
			wantMethod: []string{"echo"},
		},
		{
			name:       "truncated message sends a parse error",
			client:     sendFrames(`{"jsonrpc":"2.0","method":"echo"`, validRequest),
			recvs:      2,
			check:      requireNoReceiveError,
			wantCodes:  []Code{ParseError},
			wantMethod: []string{"echo"},
		},
		{
			name:       "trailing garbage sends a parse error",
			client:     sendFrames(validRequest+" x", validRequest),
			recvs:      2,
			check:      requireNoReceiveError,
			wantCodes:  []Code{ParseError},
			wantMethod: []string{"echo"},
		},
		{
			name:       "trailing JSON value sends a parse error",
			client:     sendFrames(validRequest+"{}", validRequest),
			recvs:      2,
			check:      requireNoReceiveError,
			wantCodes:  []Code{ParseError},
			wantMethod: []string{"echo"},
		},
		{
			name:       "duplicate key sends a parse error",
			client:     sendFrames(`{"jsonrpc":"2.0","method":"echo","method":"echo","id":1}`, validRequest),
			recvs:      2,
			check:      requireNoReceiveError,
			wantCodes:  []Code{ParseError},
			wantMethod: []string{"echo"},
		},
		{
			name:   "valid JSON that is not an object is an invalid request",
			client: sendFrames(`[1]`),
			recvs:  1,
			check:  requireNoReceiveError,
		},
		{
			name:       "valid request is dispatched",
			client:     sendFrames(validRequest),
			recvs:      1,
			check:      requireNoReceiveError,
			wantMethod: []string{"echo"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outcome := runWebSocketReceive(t, test)

			require.NotEmpty(t, outcome.errs)
			test.check(t, outcome.errs[len(outcome.errs)-1])
			require.Equal(t, test.wantCodes, outcome.errorCodes)
			require.Equal(t, test.wantMethod, outcome.dispatched)
		})
	}
}

func TestReceiveWebSocketRequestFailedReadIsTerminal(t *testing.T) {
	tests := []struct {
		name  string
		setup func(conn *websocket.Conn, cancel context.CancelFunc)
		// client drives the client side of the connection.
		client func(t *testing.T, conn *websocket.Conn)
		check  func(t *testing.T, err error)
	}{
		{
			name: "read deadline expiry",
			setup: func(conn *websocket.Conn, _ context.CancelFunc) {
				if err := conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
					panic(err)
				}
			},
			check: func(t *testing.T, err error) {
				var netErr net.Error
				require.ErrorAs(t, err, &netErr)
				require.True(t, netErr.Timeout())
			},
		},
		{
			name:   "normal closure",
			client: closeWith(websocket.CloseNormalClosure),
			check: func(t *testing.T, err error) {
				require.ErrorIs(t, err, io.EOF)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// gorilla/websocket panics after 1000 reads of a failed
			// connection; loop past that bound to prove Recv stays terminal.
			outcome := runWebSocketReceive(t, webSocketReceiveCase{
				setup:       test.setup,
				client:      test.client,
				recvs:       1100,
				keepReading: true,
			})

			require.Len(t, outcome.errs, 1100)
			for _, err := range outcome.errs {
				test.check(t, err)
			}
			require.Empty(t, outcome.errorCodes)
			require.Empty(t, outcome.dispatched)
		})
	}
}

func requireNoReceiveError(t *testing.T, err error) {
	t.Helper()
	require.NoError(t, err)
}

func closeWith(code int) func(*testing.T, *websocket.Conn) {
	return func(t *testing.T, conn *websocket.Conn) {
		t.Helper()
		message := websocket.FormatCloseMessage(code, "")
		require.NoError(t, conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second)))
	}
}

func sendFrames(frames ...string) func(*testing.T, *websocket.Conn) {
	return func(t *testing.T, conn *websocket.Conn) {
		t.Helper()
		for _, frame := range frames {
			require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(frame)))
		}
	}
}

// runWebSocketReceive runs ReceiveWebSocketRequest against a real WebSocket
// connection and returns what the server side observed.
func runWebSocketReceive(t *testing.T, test webSocketReceiveCase) webSocketReceiveOutcome {
	t.Helper()

	outcomes := make(chan webSocketReceiveOutcome, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		if test.setup != nil {
			test.setup(conn, cancel)
		}
		stream := loomhttp.NewWebSocketStream(conn)
		defer func() {
			if closeErr := stream.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
				t.Errorf("close: %v", closeErr)
			}
		}()

		var outcome webSocketReceiveOutcome
		var mu sync.Mutex
		matches := func(method string) bool {
			return method == "echo"
		}
		dispatch := func(_ context.Context, request *RawRequest) error {
			mu.Lock()
			defer mu.Unlock()
			outcome.dispatched = append(outcome.dispatched, request.Method)
			return nil
		}
		sendError := func(_ context.Context, _ any, code Code, _ string, _ any) error {
			mu.Lock()
			defer mu.Unlock()
			outcome.errorCodes = append(outcome.errorCodes, code)
			return nil
		}
		for range test.recvs {
			err := ReceiveWebSocketRequest(ctx, stream, matches, dispatch, sendError)
			outcome.errs = append(outcome.errs, err)
			if err != nil && !test.keepReading {
				break
			}
		}
		outcomes <- outcome
	}))
	defer server.Close()

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			t.Errorf("client close: %v", closeErr)
		}
	}()
	if test.client != nil {
		test.client(t, conn)
	}

	select {
	case outcome := <-outcomes:
		return outcome
	case <-time.After(5 * time.Second):
		t.Fatal("ReceiveWebSocketRequest did not finish")
		return webSocketReceiveOutcome{}
	}
}
