package http_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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

// TestWebSocketStreamCloseAfterContextCancellation checks that a stream closed
// by a canceled read reports that close, not a second close of the
// connection, to every later or concurrent Close. Generated JSON-RPC clients
// rely on this when a stream and the client share one stream wrapper.
func TestWebSocketStreamCloseAfterContextCancellation(t *testing.T) {
	cases := []struct {
		name    string
		closers int
		// waitRead makes Close run only after the canceled read returned.
		waitRead bool
	}{
		{name: "after the canceled read", closers: 2, waitRead: true},
		{name: "concurrent with the canceled read", closers: 8, waitRead: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream, cleanup := newIdleWebSocketStream(t)
			defer cleanup()

			// The deadline leaves the read time to block on the connection
			// before it is canceled, so the cancellation closes the stream.
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			readc := make(chan error, 1)
			go func() {
				readc <- stream.ReadJSON(ctx, new(map[string]string))
			}()
			if tc.waitRead {
				require.ErrorIs(t, <-readc, context.DeadlineExceeded)
				require.ErrorIs(t, stream.Conn().Close(), net.ErrClosed, "the canceled read must close the connection")
			} else {
				<-ctx.Done()
			}
			errc := make(chan error, tc.closers)
			for range tc.closers {
				go func() {
					errc <- stream.Close()
				}()
			}
			for range tc.closers {
				require.NoError(t, <-errc)
			}
			if !tc.waitRead {
				require.ErrorIs(t, <-readc, context.DeadlineExceeded)
			}
		})
	}
}

func TestWebSocketStreamNilConnection(t *testing.T) {
	stream := loomhttp.NewWebSocketStream(nil)
	require.NoError(t, stream.Close())
	require.NoError(t, stream.WriteClose("not upgraded"))

	var msg map[string]string
	err := stream.ReadJSON(context.Background(), &msg)
	require.True(t, errors.Is(err, loomhttp.ErrWebSocketStreamClosed))
}

func TestWebSocketStreamCloseIsTerminal(t *testing.T) {
	cases := []struct {
		name        string
		attachFirst bool
	}{
		{name: "close before upgrade", attachFirst: false},
		{name: "close after upgrade", attachFirst: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stream := loomhttp.NewWebSocketStream(nil)
			if tc.attachFirst {
				first, cleanup := newIdleWebSocketConn(t)
				defer cleanup()
				stream.SetConn(first)
			}
			require.NoError(t, stream.Close())

			late, cleanup := newIdleWebSocketConn(t)
			defer cleanup()
			stream.SetConn(late)

			require.Same(t, late, stream.Conn(), "SetConn after Close must still expose the upgraded connection")
			require.ErrorIs(t, late.ReadJSON(new(map[string]string)), net.ErrClosed, "SetConn after Close must close the incoming connection")
			require.ErrorIs(t, stream.ReadJSON(context.Background(), new(map[string]string)), loomhttp.ErrWebSocketStreamClosed)
			require.ErrorIs(t, stream.WriteJSON(context.Background(), map[string]string{"k": "v"}), loomhttp.ErrWebSocketStreamClosed)
			require.NoError(t, stream.Close())
		})
	}
}

func TestWebSocketStreamReadMessage(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer func() {
			if closeErr := conn.Close(); closeErr != nil {
				t.Errorf("close: %v", closeErr)
			}
		}()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"k":"v"}`)); err != nil {
			t.Errorf("write: %v", err)
			return
		}
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		if err := conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second)); err != nil {
			t.Errorf("write close: %v", err)
		}
	}))
	defer server.Close()

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}
	stream := loomhttp.NewWebSocketStream(conn)
	defer func() {
		require.NoError(t, stream.Close())
	}()

	data, err := stream.ReadMessage(context.Background())
	require.NoError(t, err)
	require.JSONEq(t, `{"k":"v"}`, string(data))

	for range 3 {
		_, err = stream.ReadMessage(context.Background())
		require.True(t, websocket.IsCloseError(err, websocket.CloseNormalClosure), "err = %v", err)
	}
}

func TestWebSocketStreamFailedReadIsTerminal(t *testing.T) {
	conn, cleanup := newIdleWebSocketConn(t)
	defer cleanup()
	stream := loomhttp.NewWebSocketStream(conn)
	defer func() {
		require.NoError(t, stream.Close())
	}()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(20*time.Millisecond)))

	// gorilla/websocket panics after 1000 reads of a failed connection, so
	// the stream must stop reading the connection after the first failure.
	for range 1100 {
		_, err := stream.ReadMessage(context.Background())
		var timeout interface{ Timeout() bool }
		require.ErrorAs(t, err, &timeout)
		require.True(t, timeout.Timeout())
	}
	var timeout interface{ Timeout() bool }
	require.ErrorAs(t, stream.ReadJSON(context.Background(), new(map[string]string)), &timeout)
	require.True(t, timeout.Timeout())
}

func TestWebSocketStreamFailedReadJSONIsTerminal(t *testing.T) {
	conn, cleanup := newIdleWebSocketConn(t)
	defer cleanup()
	stream := loomhttp.NewWebSocketStream(conn)
	defer func() {
		require.NoError(t, stream.Close())
	}()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(20*time.Millisecond)))

	// gorilla/websocket panics after 1000 reads of a failed connection, so
	// ReadJSON must stop reading the connection after the first failure.
	for range 1100 {
		err := stream.ReadJSON(context.Background(), new(map[string]string))
		var timeout interface{ Timeout() bool }
		require.ErrorAs(t, err, &timeout)
		require.True(t, timeout.Timeout())
	}
}

type webSocketDecodeTarget struct {
	K string `json:"k"`
}

func TestWebSocketStreamReadJSONDecodeErrorIsNotTerminal(t *testing.T) {
	// Every case reads one frame from the same stream, in order, so the
	// valid and close cases also prove earlier decode errors are not terminal.
	tests := []struct {
		name  string
		frame string
		check func(t *testing.T, err error, got webSocketDecodeTarget)
	}{
		{name: "malformed", frame: `{"k":x}`, check: requireDecodeError(false)},
		{name: "truncated", frame: `{"k":`, check: requireDecodeError(true)},
		{name: "empty", frame: "", check: requireDecodeError(true)},
		{name: "whitespace only", frame: " \n\t ", check: requireDecodeError(true)},
		{name: "trailing data", frame: `{"k":"v"} x`, check: requireDecodeError(false)},
		{name: "duplicate key", frame: `{"k":"a","k":"b"}`, check: requireDecodeError(false)},
		{name: "invalid UTF-8", frame: "{\"k\":\"\xff\"}", check: requireDecodeError(false)},
		{name: "wrong-case field is not matched", frame: `{"K":"v"}`, check: func(t *testing.T, err error, got webSocketDecodeTarget) {
			require.NoError(t, err)
			require.Empty(t, got.K)
		}},
		{name: "valid", frame: `{"k":"v"}`, check: func(t *testing.T, err error, got webSocketDecodeTarget) {
			require.NoError(t, err)
			require.Equal(t, "v", got.K)
		}},
	}
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer func() {
			if closeErr := conn.Close(); closeErr != nil {
				t.Errorf("close: %v", closeErr)
			}
		}()
		for _, test := range tests {
			if err := conn.WriteMessage(websocket.TextMessage, []byte(test.frame)); err != nil {
				t.Errorf("write: %v", err)
				return
			}
		}
		closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
		if err := conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second)); err != nil {
			t.Errorf("write close: %v", err)
		}
	}))
	defer server.Close()

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}
	stream := loomhttp.NewWebSocketStream(conn)
	defer func() {
		require.NoError(t, stream.Close())
	}()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got webSocketDecodeTarget
			err := stream.ReadJSON(context.Background(), &got)
			test.check(t, err, got)
		})
	}
	t.Run("normal closure", func(t *testing.T) {
		err := stream.ReadJSON(context.Background(), new(webSocketDecodeTarget))
		require.True(t, websocket.IsCloseError(err, websocket.CloseNormalClosure), "err = %v", err)
	})
}

// requireDecodeError asserts a non-close decode error and whether it wraps
// io.ErrUnexpectedEOF.
func requireDecodeError(unexpectedEOF bool) func(*testing.T, error, webSocketDecodeTarget) {
	return func(t *testing.T, err error, _ webSocketDecodeTarget) {
		t.Helper()
		require.Error(t, err)
		var closeErr *websocket.CloseError
		require.False(t, errors.As(err, &closeErr), "decode error must not be a close error: %v", err)
		require.Equal(t, unexpectedEOF, errors.Is(err, io.ErrUnexpectedEOF), "err = %v", err)
	}
}

func TestWebSocketStreamBoundsBlockedWrite(t *testing.T) {
	policy, err := loomhttp.NewStreamWritePolicy(25 * time.Millisecond)
	require.NoError(t, err)
	result := make(chan error, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		conn, upgradeErr := upgrader.Upgrade(w, r, nil)
		if upgradeErr != nil {
			result <- upgradeErr
			return
		}
		tcpConn, ok := conn.NetConn().(*net.TCPConn)
		if !ok {
			result <- fmt.Errorf("unexpected WebSocket connection type %T", conn.NetConn())
			return
		}
		if bufferErr := tcpConn.SetWriteBuffer(1024); bufferErr != nil {
			result <- bufferErr
			return
		}
		stream := loomhttp.NewWebSocketStream(conn, policy)
		result <- stream.WriteJSON(context.Background(), bytes.Repeat([]byte("x"), 1<<20))
	}))
	defer server.Close()

	conn, resp, err := websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], nil)
	require.NoError(t, err)
	if resp != nil && resp.Body != nil {
		require.NoError(t, resp.Body.Close())
	}
	defer func() {
		require.NoError(t, conn.Close())
	}()

	select {
	case err := <-result:
		require.Error(t, err)
		var timeout interface{ Timeout() bool }
		require.ErrorAs(t, err, &timeout)
		require.True(t, timeout.Timeout())
	case <-time.After(5 * time.Second):
		t.Fatal("WriteJSON did not honor the write policy")
	}
}

func newIdleWebSocketStream(t *testing.T) (*loomhttp.WebSocketStream, func()) {
	t.Helper()

	conn, cleanup := newIdleWebSocketConn(t)
	return loomhttp.NewWebSocketStream(conn), cleanup
}

func newIdleWebSocketConn(t *testing.T) (*websocket.Conn, func()) {
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
	return conn, server.Close
}
