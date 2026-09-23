package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/CaliLuke/loom/dsl"
)

// nonASCIIServiceDSL declares an HTTP service whose name contains a
// non-ASCII letter and that streams server-sent events.
var nonASCIIServiceDSL = func() {
	Service("Café", func() {
		Method("añadir", func() {
			Payload(func() {
				Attribute("nombre", String)
			})
			Result(String)
			HTTP(func() {
				POST("/añadir")
			})
		})
		Method("flüss", func() {
			StreamingResult(func() {
				Attribute("wert", String)
			})
			HTTP(func() {
				GET("/flüss")
				ServerSentEvents()
			})
		})
	})
}

// TestNonASCIIServiceGeneratedModuleBuilds checks that the generated
// packages of a service with a non-ASCII name, including the SSE files, have
// valid ASCII Go import paths and compile.
func TestNonASCIIServiceGeneratedModuleBuilds(t *testing.T) {
	root := RunHTTPDSL(t, nonASCIIServiceDSL)
	files := ServerFiles("example.com/nonascii/gen", CreateHTTPServices(root))
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.Path)
	}
	assert.Contains(t, paths, filepath.Join("gen", "http", "cafu00e9", "server", "sse.go"))

	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/nonascii", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
}
