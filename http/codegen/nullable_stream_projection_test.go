package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNullableSelectedStreamProjection(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		selectedWebSocketProjectionDSL(true)
	})
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/selectedhttpstream", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), []byte(nullableSelectedWebSocketHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const nullableSelectedWebSocketHarness = `package selectedstream_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	events "example.com/selectedhttpstream/gen/events"
	server "example.com/selectedhttpstream/gen/http/events/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type nullableImplementation struct {
	sent     chan error
	detached bool
}

func (i nullableImplementation) Watch(ctx context.Context, stream events.WatchServerStream) error {
	value, err := stream.RecvWithContext(ctx)
	if err != nil {
		return err
	}
	private := "private-sentinel"
	result := loom.NullableValue(events.WatchResult{Value: value, Internal: &private})
	switch value {
	case "null":
		result = loom.NullValue[events.WatchResult]()
	case "absent":
		result = loom.Nullable[events.WatchResult]{}
	}
	if i.detached {
		ctx = context.WithoutCancel(ctx)
	}
	err = stream.SendAndCloseWithContext(ctx, result)
	i.sent <- err
	return err
}

func TestNullableSelectedWebSocketFrame(t *testing.T) {
	for _, detached := range []bool{false, true} {
		t.Run(fmt.Sprintf("detached=%v", detached), func(t *testing.T) {
			testNullableSelectedWebSocketFrame(t, detached)
		})
	}
}

func testNullableSelectedWebSocketFrame(t *testing.T, detached bool) {
	sent := make(chan error, 1)
	mux := loomhttp.NewMuxer()
	// Recv upgrades before the application has its result. The projection check
	// governs the application frame, not that already completed handshake.
	handler := func(context.Context, http.ResponseWriter, error) {
		// The stream already owns and closes its upgraded connection.
	}
	server.Mount(mux, server.New(events.NewEndpoints(nullableImplementation{sent: sent, detached: detached}), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, handler, nil, &websocket.Upgrader{}, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	for _, value := range []string{"hello", "", "null", "absent"} {
		t.Run(value, func(t *testing.T) {
			conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(host.URL, "http")+"/rpc", nil)
			if response != nil && response.Body != nil {
				require.NoError(t, response.Body.Close())
			}
			require.NoError(t, err)
			defer func() {
				require.NoError(t, conn.Close())
			}()
			require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
			require.NoError(t, conn.WriteJSON(value))
			_, data, readErr := conn.ReadMessage()
			var sendErr error
			select {
			case sendErr = <-sent:
			case <-time.After(5 * time.Second):
				t.Fatal("service did not report send result")
			}
			if value == "null" || value == "absent" {
				require.Error(t, sendErr)
				if !detached {
					require.ErrorIs(t, sendErr, context.Canceled)
				} else if value == "absent" {
					require.ErrorIs(t, sendErr, loom.ErrAbsentNullable)
				} else {
					require.ErrorContains(t, sendErr, "null response cannot supply selected body")
				}
				require.Error(t, readErr)
				require.Empty(t, data, "invalid parent must not publish an application frame")
				return
			}
			require.NoError(t, sendErr)
			require.NoError(t, readErr)
			require.Equal(t, "\""+value+"\"", strings.TrimSpace(string(data)))
			require.NotContains(t, string(data), "private-sentinel")
		})
	}
}
`
