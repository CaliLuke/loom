package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

func TestBasicCredentialValidationGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, servicedata.BasicCredentialConstraintsDSL("jsonrpc"))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/basic", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "basic_test.go"), []byte(servicedata.BasicCredentialConstraintsHarness("jsonrpc")), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-run", "TestBasic", "./...")
}
