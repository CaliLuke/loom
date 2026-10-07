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
		items := Type("Items", ArrayOf(String))
		page := Type("Page", func() {
			Attribute("total", Int)
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
	"strings"
	"testing"

	client "example.com/arrayrpc/gen/jsonrpc/listing/client"
	server "example.com/arrayrpc/gen/jsonrpc/listing/server"
	svc "example.com/arrayrpc/gen/listing"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestJSONRPCArrayObject(t *testing.T) {
	echo := func(_ context.Context, value any) (any, error) {
		return value, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{Echo: echo}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
	host := httptest.NewServer(mux)
	defer host.Close()
	c := client.NewClient("http", strings.TrimPrefix(host.URL, "http://"), host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, wire := range []string{"[]", "[\"ok\"]", "{\"total\":0}"} {
		var payload svc.Choice
		require.NoError(t, json.Unmarshal([]byte(wire), &payload))
		result, err := c.Echo()(t.Context(), &payload)
		require.NoError(t, err)
		require.Equal(t, &payload, result)
	}
}
`
