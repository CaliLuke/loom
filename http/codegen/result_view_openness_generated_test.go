package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResultViewOpennessGenerated(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"open-api", testdata.ResultViewOpennessDSL},
		{"closed-api", testdata.ClosedAPIResultViewOpennessDSL},
		{"declaration-selected", testdata.ResultDeclarationViewOpennessDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/viewopenness", root)
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "build", "./...")
		})
	}
}
