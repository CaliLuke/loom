package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestAnyDefaultsGeneratedHTTP(t *testing.T) {
	root := RunHTTPDSL(t, testdata.HTTPAnyDefaultsDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/anydefaults", root)
	harness := strings.NewReplacer("TRANSPORT", "http", "RPC", "false", "SERVER_ARGS", "nil, nil").Replace(testdata.AnyDefaultsHarness)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(harness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "./...")
}
