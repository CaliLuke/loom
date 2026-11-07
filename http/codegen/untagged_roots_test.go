package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestUntaggedRootsGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedRootsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/rootunion", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union_test.go"), []byte(testdata.UntaggedRootsHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "closure_profiles_test.go"), []byte(testdata.UntaggedClosureProfilesHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
