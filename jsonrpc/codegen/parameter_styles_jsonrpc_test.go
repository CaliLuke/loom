package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestParameterStylesGeneratedJSONRPC(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		valueRange := Type("ValueRange", func() {
			Attribute("gt", Int)
			Attribute("lt", Int)
			Required("gt", "lt")
		})
		Service("styles", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("inspect", func() {
				Payload(func() {
					Attribute("range", valueRange)
					Attribute("tags", ArrayOf(String))
					Required("range", "tags")
				})
				JSONRPC(func() {
					Header("range:X-Range", func() {
						Meta("openapi:explode", "true")
					})
					Cookie("tags", func() {
						Meta("openapi:explode", "false")
					})
				})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/rpcstyles", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "styles_test.go"), []byte(parameterStylesJSONRPCHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

const parameterStylesJSONRPCHarness = `package rpcstyles_test

import (
 "context"
 "encoding/json/v2"
 "net/http"
 "net/http/httptest"
 "strings"
 "testing"

 svc "example.com/rpcstyles/gen/styles"
 client "example.com/rpcstyles/gen/jsonrpc/styles/client"
 server "example.com/rpcstyles/gen/jsonrpc/styles/server"
 loomhttp "github.com/CaliLuke/loom/http"
 "github.com/stretchr/testify/require"
)

type wireDoer struct {
 t *testing.T
 base *http.Client
}

func (d wireDoer) Do(req *http.Request) (*http.Response, error) {
 require.Equal(d.t, "gt=1, lt=9", req.Header.Get("X-Range"))
 require.Equal(d.t, "tags=a,b", req.Header.Get("Cookie"))
 require.NotNil(d.t, req.Body, "metadata-only JSON-RPC calls still require an envelope")
 return d.base.Do(req)
}

func TestStyledMetadata(t *testing.T) {
 calls := 0
 var got *svc.InspectPayload
 endpoint := func(_ context.Context, raw any) (any, error) {
  calls++
  got = raw.(*svc.InspectPayload)
  return nil, nil
 }
 mux := loomhttp.NewMuxer()
 server.Mount(mux, server.New(&svc.Endpoints{Inspect:endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
 host := httptest.NewServer(mux)
 defer host.Close()
 c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), wireDoer{t:t, base:host.Client()}, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
 payload := &svc.InspectPayload{Range:&svc.ValueRange{Gt:1, Lt:9}, Tags:[]string{"a", "b"}}
 _, err := c.Inspect()(t.Context(), payload)
 require.NoError(t, err)
 require.Equal(t, 1, calls)
 require.Equal(t, payload, got)
 calls = 0
 request := httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"method\":\"inspect\",\"params\":{},\"id\":1}"))
 request.Header.Set("Content-Type", "application/json")
 request.Header.Set("X-Range", "gt=secret,lt")
 request.Header.Set("Cookie", "tags=a,b")
 response := httptest.NewRecorder()
 mux.ServeHTTP(response, request)
 require.Equal(t, http.StatusOK, response.Code)
 require.Zero(t, calls)
 var envelope map[string]any
 require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
 require.Contains(t, envelope, "error")
 rpcError := envelope["error"].(map[string]any)
 require.Equal(t, float64(-32602), rpcError["code"])
 require.NotContains(t, response.Body.String(), "secret")
}
`
