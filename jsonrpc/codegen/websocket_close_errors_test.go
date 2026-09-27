package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCWebSocketServerStreamingCloseGeneratedModule(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		API("closeerrors", func() {
			JSONRPC(func() {})
		})
		Service("watcher", func() {
			JSONRPC(func() {
				GET("/ws")
			})
			Method("watch", func() {
				StreamingResult(String)
				JSONRPC(func() {})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/closeerrors", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "close_test.go"), []byte(webSocketCloseErrorsHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=3", "./...")
}

const webSocketCloseErrorsHarness = `package closeerrors_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	client "example.com/closeerrors/gen/jsonrpc/watcher/client"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/jsonrpc"
)

func TestServerStreamingClose(t *testing.T) {
	for _, target := range []string{"stream", "client"} {
		t.Run(target, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			requested := make(chan struct{}, 1)
			upgrader := websocket.Upgrader{}
			hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ws, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Errorf("upgrade: %v", err)
					return
				}
				defer func() {
					if err := ws.Close(); err != nil {
						t.Errorf("close peer: %v", err)
					}
				}()
				for {
					if _, _, err := ws.ReadMessage(); err != nil {
						return
					}
					requested <- struct{}{}
				}
			}))
			defer hs.Close()
			c := client.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), hs.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
			defer func() {
				if err := c.Close(); err != nil {
					t.Errorf("close client: %v", err)
				}
			}()
			raw, err := c.Watch()(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			stream := raw.(*client.WatchClientStream)
			result := make(chan error, 1)
			go func() {
				_, err := stream.RecvWithContext(ctx)
				result <- err
			}()
			select {
			case <-requested:
			case <-ctx.Done():
				t.Fatal("receive never sent its request")
			}
			if target == "stream" {
				err = stream.Close()
			} else {
				err = c.Close()
			}
			if err != nil {
				t.Fatal(err)
			}
			select {
			case err := <-result:
				if !errors.Is(err, jsonrpc.ErrStreamClosed) {
					t.Errorf("blocked receive: got %v, want ErrStreamClosed", err)
				}
			case <-ctx.Done():
				t.Fatal("receive did not return after closure")
			}
			if _, err := stream.RecvWithContext(ctx); !errors.Is(err, jsonrpc.ErrStreamClosed) {
				t.Errorf("receive after closure: got %v, want ErrStreamClosed", err)
			}
			if err := stream.Close(); err != nil {
				t.Error(err)
			}
		})
	}
}
`
