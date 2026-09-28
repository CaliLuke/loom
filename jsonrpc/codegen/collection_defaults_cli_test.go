package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestCollectionBodyDefaultsGeneratedCLI(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.JSONRPCCollectionBodyDefaultsDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/collectiondefaults", root)
	for _, file := range ClientCLIFiles("example.com/collectiondefaults/gen", CreateJSONRPCServices(root)) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	harness := strings.ReplaceAll(testdata.CollectionBodyDefaultsCLIHarness, "TRANSPORT", "jsonrpc")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "defaults_test.go"), []byte(harness), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}
