package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestAnyDefaultsGeneratedJSONRPC(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCAnyDefaultsDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/anydefaults", root)
	harness := strings.NewReplacer("TRANSPORT", "jsonrpc", "RPC", "true", "SERVER_ARGS", "nil").Replace(testdata.AnyDefaultsHarness)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(harness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}
