package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestParameterStylesGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ParameterStylesDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/parameterstyles", root)
	for _, name := range []string{"client", "foreign"} {
		content, err := os.ReadFile(filepath.Join("testdata", "parameter_styles", name+".go.txt"))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name+"_test.go"), content, 0o600))
	}
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
