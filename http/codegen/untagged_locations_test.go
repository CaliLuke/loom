package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestUntaggedLocationsGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedLocationsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/textunion", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "union_test.go"), []byte(testdata.UntaggedLocationsHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
