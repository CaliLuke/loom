package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestTypedCollectionDefaultsGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.TypedCollectionDefaultsDSL("http"))
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/defaults", root)
	module, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	module = []byte(strings.ReplaceAll(string(module), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), module, 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
}
