package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	httptestdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestGeneratedAdditionalComposite(t *testing.T) {
	root := RunJSONRPCDSL(t, httptestdata.AdditionalCompositeJSONRPCDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/composite", root)
	harness, err := os.ReadFile("../../http/codegen/testdata/additional_composite_harness.go.txt")
	require.NoError(t, err)
	source := strings.ReplaceAll(string(harness), "TRANSPORT", "jsonrpc")
	source = strings.ReplaceAll(source, "SERVER_OPTIONS", "nil")
	source = strings.ReplaceAll(source, "IS_RPC", "true")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "composite_test.go"), []byte(source), 0600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "./...")
}
