package codegen

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

func TestSelectedStreamProjectionReadiness(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		Service("events", func() {
			Method("watch", func() {
				StreamingPayload(String)
				Result(func() {
					Attribute("value", String)
					Attribute("internal", String)
					Required("value", "internal")
				})
				HTTP(func() {
					GET("/rpc")
					Response(StatusOK, func() {
						Body("value")
					})
				})
			})
		})
	})
	dir := t.TempDir()
	if base := os.Getenv("LOOM_PROJECTION_ARTIFACTS"); base != "" {
		dir = filepath.Join(base, "http-websocket")
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	renderHTTPModule(t, dir, "example.com/selectedhttpstream", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "response_test.go"), []byte(selectedWebSocketProjectionHarness), 0o600))
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const selectedWebSocketProjectionHarness = `package selectedstream_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	events "example.com/selectedhttpstream/gen/events"
	server "example.com/selectedhttpstream/gen/http/events/server"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/gorilla/websocket"
)

type implementation struct{}

func (implementation) Watch(ctx context.Context, stream events.WatchServerStream) error {
	value, err := stream.RecvWithContext(ctx)
	if err != nil {
		return err
	}
	return stream.SendAndCloseWithContext(ctx, &events.WatchResult{Value: value, Internal: "private-sentinel"})
}
func TestSelectedWebSocketResponse(t *testing.T) {
	mux := loomhttp.NewMuxer()
	svc := implementation{}
	server.Mount(mux, server.New(events.NewEndpoints(svc), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil, &websocket.Upgrader{}, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	for _, value := range []string{"hello", ""} {
		t.Run(value, func(t *testing.T) {
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
			if err := conn.WriteMessage(websocket.TextMessage, []byte("\""+value+"\"")); err != nil {
				t.Fatal(err)
			}
			_, data, err := conn.ReadMessage()
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(string(data)) != "\""+value+"\"" || strings.Contains(string(data), "private-sentinel") {
				t.Errorf("selected response=%s", data)
			}
		})
	}
}
`
