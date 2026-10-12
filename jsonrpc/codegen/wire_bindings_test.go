package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestJSONRPCWireBindingsGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.EffectiveWireBindingsDSL("jsonrpc"))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/binding", root)
	renderCodegenFiles(t, dir, ClientCLIFiles("example.com/binding/gen", CreateJSONRPCServices(root)))
	runGoJSONRPCTestCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "auth_test.go"), []byte(testdata.EffectiveWireBindingsHarness("jsonrpc")), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
}

func TestJSONRPCWireBindingScopeGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.EffectiveWireBindingScopeDSL("jsonrpc"))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/scope", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scope_test.go"), []byte(testdata.EffectiveWireBindingScopeHarness("jsonrpc")), 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e")
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}
