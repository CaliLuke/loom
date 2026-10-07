package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestUntaggedArraysGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedArraysDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/arrayunion", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union_test.go"), []byte(testdata.UntaggedArraysHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
