package http_test

import (
	"bytes"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

type messageLimitResult struct {
	length int
	err    error
	closed bool
}

func TestWebSocketMessageLimit(t *testing.T) {
	cases := []struct {
		name        string
		limit       int64
		size        int
		compress    bool
		tooLarge    bool
		hookLimit   int64
		priorLimit  int64
		messageType int
		fragmented  bool
	}{
		{name: "exact configured limit", limit: 128, size: 128},
		{name: "fragmented binary exact", limit: 128, size: 128, fragmented: true},
		{name: "fragmented text exceeds", limit: 128, size: 129, fragmented: true, messageType: websocket.TextMessage, tooLarge: true},
		{name: "exact compressed limit", limit: 128, size: 128, compress: true},
		{name: "text configured limit exceeded", limit: 128, size: 129, messageType: websocket.TextMessage, tooLarge: true},
		{name: "hook lowers framing limit", limit: 128, hookLimit: 64, size: 65, tooLarge: true},
		{name: "hook cannot raise expanded limit", limit: 128, hookLimit: 512, size: 129, tooLarge: true},
		{name: "explicit config replaces prior smaller framing", limit: 128, priorLimit: 64, size: 128},
		{name: "configured limit exceeded", limit: 128, size: 129, tooLarge: true},
		{name: "compressed expansion", limit: 128, size: 4096, compress: true, tooLarge: true},
		// Compressible data keeps the default expanded-byte-limit case a small
		// wire frame under coverage while still exceeding the configured default.
		{name: "default limit exceeded", size: loomhttp.DefaultMaxRequestBodyBytes + 1, compress: true, tooLarge: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := make(chan messageLimitResult, 1)
			release := make(chan struct{})
			done := make(chan struct{})
			upgrader := websocket.Upgrader{EnableCompression: tc.compress}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer close(done)
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Errorf("upgrade: %v", err)
					return
				}
				var config loomhttp.WebSocketConfig
				if tc.limit > 0 {
					config, err = loomhttp.NewWebSocketConfig(loomhttp.WebSocketLimits{MaxMessageBytes: tc.limit}, func(conn *websocket.Conn, _ context.CancelFunc) *websocket.Conn {
						if tc.hookLimit > 0 {
							conn.SetReadLimit(tc.hookLimit)
						}
						return conn
					})
					if err != nil {
						t.Errorf("config: %v", err)
						return
					}
				}
				if tc.priorLimit > 0 {
					conn.SetReadLimit(tc.priorLimit)
				}
				stream := loomhttp.NewWebSocketStream(config)
				if err := stream.SetConn(context.Background(), conn, nil); err != nil {
					t.Errorf("attach: %v", err)
					return
				}
				defer func() {
					if err := stream.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
						t.Errorf("close server: %v", err)
					}
				}()
				data, readErr := stream.ReadMessage(context.Background())
				if tc.tooLarge {
					_, repeated := stream.ReadMessage(context.Background())
					if !errors.Is(repeated, websocket.ErrReadLimit) {
						t.Errorf("repeated read: %v", repeated)
					}
				}
				closed := errors.Is(conn.NetConn().SetDeadline(time.Now().Add(time.Second)), net.ErrClosed)
				result <- messageLimitResult{length: len(data), err: readErr, closed: closed}
				<-release
			}))
			defer server.Close()
			defer func() {
				close(release)
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("server handler did not finish")
				}
			}()
			dialer := websocket.Dialer{EnableCompression: tc.compress}
			if tc.fragmented {
				dialer.WriteBufferSize = 32
			}
			conn, resp, err := dialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			defer func() {
				if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
					t.Errorf("close client: %v", err)
				}
			}()
			require.NoError(t, conn.SetWriteDeadline(time.Now().Add(5*time.Second)))
			messageType := tc.messageType
			if messageType == 0 {
				messageType = websocket.BinaryMessage
			}
			var writeErr error
			data := bytes.Repeat([]byte("x"), tc.size)
			if tc.fragmented {
				writer, err := conn.NextWriter(messageType)
				require.NoError(t, err)
				for offset := 0; offset < len(data); offset += 16 {
					end := min(offset+16, len(data))
					if _, err := writer.Write(data[offset:end]); err != nil {
						writeErr = err
						break
					}
				}
				writeErr = errors.Join(writeErr, writer.Close())
			} else {
				writeErr = conn.WriteMessage(messageType, data)
			}
			if !tc.tooLarge {
				require.NoError(t, writeErr)
			}
			select {
			case got := <-result:
				if tc.tooLarge {
					require.ErrorIs(t, got.err, websocket.ErrReadLimit)
					require.Zero(t, got.length)
					require.True(t, got.closed, "oversized message must release the socket")
					require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
					_, _, err := conn.ReadMessage()
					require.True(t, websocket.IsCloseError(err, websocket.CloseMessageTooBig), "peer close: %v", err)
				} else {
					require.NoError(t, got.err)
					require.Equal(t, tc.size, got.length)
					require.False(t, got.closed)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("read did not complete")
			}
		})
	}
}
