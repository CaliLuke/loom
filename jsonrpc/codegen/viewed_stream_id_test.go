package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

const viewedStreamIDHarness = `package idviews_test

import (
	"context"
	"encoding/json/v2"
	svc "example.com/idviews/gen/files"
	client "example.com/idviews/gen/jsonrpc/files/client"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIDBeforeViewValidation(t *testing.T) {
	for _, view := range []string{"default", "tiny"} {
		for _, explicit := range []bool{false, true} {
			name := view + "/response-id"
			if explicit {
				name = view + "/body-id"
			}
			t.Run(name, func(t *testing.T) {
				ids := make(chan string, 1)
				upgrader := websocket.Upgrader{}
				hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					conn, err := upgrader.Upgrade(w, r, nil)
					if err != nil {
						t.Error(err)
						return
					}
					defer func() {
						if err := conn.Close(); err != nil {
							t.Error(err)
						}
					}()
					var req struct{ ID string }
					if err := conn.ReadJSON(&req); err != nil {
						t.Error(err)
						return
					}
					ids <- req.ID
					body := map[string]any{}
					if view == "default" {
						body["title"] = "title"
					}
					if explicit {
						body["id"] = "body-id"
					}
					message := map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": body, "loom_view": view}
					data, err := json.Marshal(message)
					if err != nil {
						t.Error(err)
						return
					}
					if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
						t.Error(err)
					}
				}))
				defer hs.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				c := client.NewClient("ws", strings.TrimPrefix(hs.URL, "http://"), nil, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
				defer func() {
					require.NoError(t, c.Close())
				}()
				raw, err := c.Talk()(ctx, nil)
				require.NoError(t, err)
				stream := raw.(*client.TalkClientStream)
				require.NoError(t, stream.SendWithContext(ctx, &svc.Input{ID: "application-id"}))
				result, err := stream.RecvWithContext(ctx)
				require.NoError(t, err)
				wantID := <-ids
				if explicit {
					wantID = "body-id"
				}
				require.Equal(t, wantID, result.ID)
				require.NoError(t, stream.Close())
			})
		}
	}
}
`

// TestViewedStreamResponseID keeps the WebSocket response-ID fallback ahead of
// view validation when the ID field is absent from the result body.
func TestViewedStreamResponseID(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.ViewedStreamResponseIDDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/idviews", root)
	services := CreateJSONRPCServices(root)
	for _, svc := range root.Services {
		if views := servicecodegen.ViewsFile("example.com/idviews/gen", svc, services.ServicesData); views != nil {
			renderCodegenFiles(t, dir, []*cg.File{views})
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "id_test.go"), []byte(viewedStreamIDHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}
