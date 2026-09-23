package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/CaliLuke/loom/dsl"
)

// nonASCIIServiceDSL declares a JSON-RPC service whose name contains a
// non-ASCII letter and that streams server-sent events.
var nonASCIIServiceDSL = func() {
	API("nonascii", func() {
		JSONRPC(func() {})
	})
	Service("Café", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("añadir", func() {
			Payload(func() {
				ID("id", String)
				Attribute("nombre", String)
			})
			Result(func() {
				ID("id", String)
				Attribute("total", Int)
			})
			JSONRPC(func() {})
		})
		Method("flüss", func() {
			Payload(func() {
				ID("id", String)
			})
			StreamingResult(func() {
				ID("id", String)
				Attribute("wert", String)
			})
			JSONRPC(func() {
				ServerSentEvents()
			})
		})
	})
}

// TestNonASCIIServiceGeneratedModuleBuilds checks that the generated
// packages of a JSON-RPC service with a non-ASCII name, including the SSE
// stream files, have valid ASCII Go import paths and compile.
func TestNonASCIIServiceGeneratedModuleBuilds(t *testing.T) {
	root := RunJSONRPCDSL(t, nonASCIIServiceDSL)
	files := SSEServerFiles("example.com/nonascii/gen", CreateJSONRPCServices(root))
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	assert.Contains(t, paths, filepath.Join("gen", "jsonrpc", "cafu00e9", "server", "stream.go"))

	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/nonascii", root)
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}
