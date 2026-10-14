package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestJSONRPCRequestContentLength(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		API("lengths", func() {
			JSONRPC(func() {
			})
		})
		Service("lengths", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			Method("send", func() {
				Payload(func() {
					ID("id", String)
					Attribute("length", Int64)
					Attribute("value", String)
					Required("id", "value")
				})
				Result(Empty)
				JSONRPC(func() {
					Header("length:Content-Length")
				})
			})
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/rpclengths", root)
	harness, err := os.ReadFile("testdata/request_content_length_test.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "lengths_test.go"), harness, 0600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}
