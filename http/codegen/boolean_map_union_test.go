package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestBooleanMapUnionsGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.BooleanMapUnionDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/unionkeys", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union_keys_test.go"), []byte(testdata.BooleanMapUnionHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}
