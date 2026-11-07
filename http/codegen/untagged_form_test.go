package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestUntaggedFormGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.UntaggedFormDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/formchoices", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "form_test.go"), []byte(testdata.UntaggedFormHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "./...")
}
