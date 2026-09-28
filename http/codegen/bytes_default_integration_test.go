package codegen

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/bytes_default_form_test.go.txt
var bytesDefaultFormHarness string

// TestBytesDefaultGeneratedForm checks the default boundary used by form bodies,
// whose source representation has native slices rather than JSON wrappers.
func TestBytesDefaultGeneratedForm(t *testing.T) {
	root := RunHTTPDSL(t, codegentestdata.BytesDefaultFormDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/bytesdefaults", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(bytesDefaultFormHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}
