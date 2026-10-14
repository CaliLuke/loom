package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

const jsonrpcHeaderPhysicalHarness = `package rpcheaderrepresentation_test

import (
	"context"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	loomhttp "github.com/CaliLuke/loom/http"
	svc "example.com/rpcheaderrepresentation/gen/headers"
	client "example.com/rpcheaderrepresentation/gen/jsonrpc/headers/client"
	server "example.com/rpcheaderrepresentation/gen/jsonrpc/headers/server"
)

func TestNamedMetadataRoundTrip(t *testing.T) {
	for _, value := range []*svc.MetadataResult{
		{},
		{Tags: []svc.Label{}, Numbers: []svc.Number{}},
		{Tags: []svc.Label{"a", "b"}, Numbers: []svc.Number{-1, 2}},
	} {
		mux := loomhttp.NewMuxer()
		endpoints := &svc.Endpoints{Metadata: func(context.Context, any) (any, error) {
			return value, nil
		}}
		server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
		host := httptest.NewServer(mux)
		t.Cleanup(host.Close)
		u, err := url.Parse(host.URL)
		require.NoError(t, err)
		c := client.NewClient(u.Scheme, u.Host, host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
		result, err := c.Metadata()(context.Background(), &svc.MetadataPayload{ID: "request"})
		require.NoError(t, err)
		require.Equal(t, value, result)
	}
}
`

// TestJSONRPCResponseCollectionPhysicalTypes checks the body response encoder's
// named string and numeric element carriers through actual JSON-RPC generation.
func TestJSONRPCResponseCollectionPhysicalTypes(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		API("headers", func() {
			JSONRPC(func() {
			})
		})
		label := Type("Label", String)
		number := Type("Number", Int64)
		Service("headers", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("metadata", func() {
				Payload(func() {
					ID("id", String)
					Required("id")
				})
				Result(func() {
					Attribute("tags", ArrayOf(label))
					Attribute("numbers", ArrayOf(number))
				})
				JSONRPC(func() {
					Response(func() {
						Body(func() {
							Attribute("tags")
							Attribute("numbers")
						})
					})
				})
			})
		})
	})
	dir := t.TempDir()
	if output := os.Getenv("LOOM_HEADER_REPRESENTATION_OUTPUT"); output != "" {
		dir = filepath.Join(output, t.Name())
		require.NoError(t, os.MkdirAll(dir, 0750))
	}
	renderJSONRPCModule(t, dir, "example.com/rpcheaderrepresentation", root)
	if os.Getenv("LOOM_HEADER_REPRESENTATION_GENERATE_ONLY") != "" {
		return
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "header_test.go"), []byte(jsonrpcHeaderPhysicalHarness), 0600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}
