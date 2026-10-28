package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCUntaggedArrayObject(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		API("arrayrpc", func() {
			JSONRPC(func() {

			})
		})
		first := Type("Person", func() {
			Attribute("name", String)
			Required("name")
		})
		second := Type("Group", func() {
			Attribute("group", String)
			Required("group")
		})
		entry := Type("Entry", OneOf(first, second), Untagged)
		items := Type("Items", ArrayOf(String))
		page := Type("Page", func() {
			Attribute("total", Int)
			Attribute("entries", ArrayOf(entry))
			Attribute("labels", MapOf(String, String), func() {
				MinLength(1)
				Elem(func() {
					MinLength(2)
				})
			})
			Attribute("status", String, func() {
				Nullable()
				Enum("open", "closed")
			})
			Required("total")
		})
		choice := Type("Choice", OneOf(items, page), func() {
			Untagged()
		})
		Service("listing", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("echo", func() {
				Payload(choice)
				Result(choice)
				JSONRPC(func() {

				})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/arrayrpc", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "arrays_test.go"), []byte(jsonRPCUntaggedArrayHarness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}

const jsonRPCUntaggedArrayHarness = `package arrayrpc_test

import (
	"context"
	"encoding/json/v2"
	"net/http/httptest"
 "net/http"
 "io"
 "sync/atomic"
	"strings"
	"testing"

	client "example.com/arrayrpc/gen/jsonrpc/listing/client"
	server "example.com/arrayrpc/gen/jsonrpc/listing/server"
	svc "example.com/arrayrpc/gen/listing"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestJSONRPCArrayObject(t *testing.T) {
 var calls atomic.Int32
	echo := func(_ context.Context, value any) (any, error) {
 calls.Add(1)
		return value, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{Echo: echo}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, wire := range []string{"[]", "[\"ok\"]", "{\"total\":0}", "{\"total\":0,\"status\":null}", "{\"total\":0,\"status\":\"open\"}", "{\"total\":0,\"labels\":{\"a\":\"ok\"}}", "{\"total\":0,\"entries\":[]}", "{\"total\":2,\"entries\":[{\"name\":\"person\"},{\"group\":\"team\"}]}"} {
		var payload svc.Choice
		require.NoError(t, json.Unmarshal([]byte(wire), &payload))
		result, err := c.Echo()(t.Context(), &payload)
		require.NoError(t, err)
		require.Equal(t, &payload, result)
	}
 for _, wire := range []string{"{\"total\":0,\"labels\":{}}", "{\"total\":0,\"labels\":{\"a\":\"x\"}}", "{\"total\":0,\"labels\":{\"a\":null}}"} {
  var value svc.Choice
  require.Error(t, json.Unmarshal([]byte(wire), &value))
 }

 for _, wire := range []string{"{\"total\":0,\"entries\":[{}]}", "{\"total\":0,\"entries\":[{\"name\":\"person\",\"group\":\"team\"}]}", "{\"total\":0,\"entries\":[null]}"} {
  before := calls.Load()
  body := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"echo\",\"params\":" + wire + "}"
  response, err := host.Client().Post(host.URL+"/rpc", "application/json", strings.NewReader(body))
  require.NoError(t, err)
  raw, err := io.ReadAll(response.Body)
  require.NoError(t, err)
  require.NoError(t, response.Body.Close())
  require.Equal(t, http.StatusOK, response.StatusCode, string(raw))
  var envelope map[string]any
  require.NoError(t, json.Unmarshal(raw, &envelope))
  problem, ok := envelope["error"].(map[string]any)
  require.True(t, ok, string(raw))
  require.Equal(t, float64(-32602), problem["code"])
  require.Equal(t, "invalid request body", problem["message"])
  require.Equal(t, before, calls.Load())
 }

	var invalid svc.Choice
	require.Error(t, json.Unmarshal([]byte("{\"total\":0,\"status\":\"other\"}"), &invalid))
}
`
