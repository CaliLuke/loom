package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNullableSelectedJSONRPCStream(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		selectedJSONRPCStreamProjectionDSL("websocket", true)
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/selectedstream", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), []byte(nullableSelectedJSONRPCStreamHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const nullableSelectedJSONRPCStreamHarness = `package selectedstream_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	events "example.com/selectedstream/gen/events"
	server "example.com/selectedstream/gen/jsonrpc/events/server"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

type implementation struct {
	sent chan error
}

func (implementation) Watch(_ context.Context, value string) (loom.Nullable[events.WatchResult], error) {
	return loom.NullableValue(events.WatchResult{Value: value, Internal: new("private-sentinel")}), nil
}
func (i implementation) HandleStream(ctx context.Context, stream events.Stream) error {
	for _, invalid := range []loom.Nullable[events.WatchResult]{loom.NullValue[events.WatchResult](), {}} {
		i.sent <- stream.SendWatchNotification(ctx, invalid)
	}
	if err := stream.SendWatchNotification(ctx, loom.NullableValue(events.WatchResult{Value: "", Internal: new("private-sentinel")})); err != nil {
		return err
	}
	for {
		if err := stream.Recv(ctx); err != nil {
			return err
		}
	}
}
func TestSelectedWebSocketResponse(t *testing.T) {
	mux := loomhttp.NewMuxer()
	sent := make(chan error, 2)
	svc := implementation{sent: sent}
	server.Mount(mux, server.New(svc.HandleStream, events.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, &websocket.Upgrader{}, loomhttp.WebSocketConfig{}))
	host := httptest.NewServer(mux)
	defer host.Close()
	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(host.URL, "http")+"/rpc", nil)
	if response != nil && response.Body != nil {
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	for index := range 2 {
		select {
		case sendErr := <-sent:
			if index == 0 {
				require.ErrorContains(t, sendErr, "null response cannot supply selected body")
			} else {
				require.ErrorIs(t, sendErr, loom.ErrAbsentNullable)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("missing invalid notification result")
		}
	}
	// The first published frame must be the concrete empty notification.
	_, notification, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notification), "\"params\":\"\"") || !strings.Contains(string(notification), "\"method\":\"watch\"") || strings.Contains(string(notification), "\"id\"") || strings.Contains(string(notification), "private-sentinel") {
		t.Errorf("selected notification=%s", notification)
	}
	for _, value := range []string{"hello", ""} {
		request := "{\"jsonrpc\":\"2.0\",\"id\":\"request\",\"method\":\"watch\",\"params\":\"" + value + "\"}"
		if err := conn.WriteMessage(websocket.TextMessage, []byte(request)); err != nil {
			t.Fatal(err)
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "\"result\":\""+value+"\"") || !strings.Contains(string(data), "\"id\":\"request\"") || strings.Contains(string(data), "private-sentinel") {
			t.Errorf("selected response=%s", data)
		}
	}
}
`
