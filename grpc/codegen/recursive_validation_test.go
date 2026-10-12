package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestRecursiveValidationGenerated(t *testing.T) {
	root := RunGRPCDSL(t, testdata.RecursiveValidationDSL)
	dir := t.TempDir()
	renderGRPCModule(t, dir, "example.com/recursivegrpc", root, resolveGRPCLoomSource(t))
	mod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	mod = []byte(strings.ReplaceAll(string(mod), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), mod, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "recursive_test.go"), []byte(recursiveValidationRuntime), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "build", "./...")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "-race", "-count=1", ".")
}
