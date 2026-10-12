package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestObjectDefaultsTransport(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.ObjectDefaultsDSL("jsonrpc"))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/objectdefaults", root)
	runGoJSONRPCTestCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012214204-66a9dc9090a0")
	harness := strings.NewReplacer("TRANSPORT", "jsonrpc", "RPC", "true", "SERVER_ARGS", "nil").Replace(testdata.ObjectDefaultsHarness)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(harness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}
