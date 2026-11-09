package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestNamedObjectExamplesGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.NamedObjectExamplesDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/addresses", root)
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
}
