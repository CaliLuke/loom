package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCResponseCollection(t *testing.T) {
	root := RunJSONRPCDSL(t, jsonrpcHeaderCollectionDSL)
	dir := t.TempDir()
	if output := os.Getenv("LOOM_HEADER_OUTPUT"); output != "" {
		dir = filepath.Join(output, t.Name())
		require.NoError(t, os.MkdirAll(dir, 0750))
	}
	renderJSONRPCModule(t, dir, "example.com/rpcheaders", root)
	harness, err := os.ReadFile("testdata/header_collection.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "header_test.go"), harness, 0600))
	if os.Getenv("LOOM_HEADER_GENERATE_ONLY") != "" {
		return
	}
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}

func jsonrpcHeaderCollectionDSL() {
	API("headers", func() {
		JSONRPC(func() {
		})
	})
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
				Attribute("tags", ArrayOf(String), func() {
					Default([]string{"fallback"})
				})
				Attribute("numbers", ArrayOf(Int), func() {
					Default([]int{7})
				})
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
}
