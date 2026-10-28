package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testingx"
)

// TestConcurrencyAdmissionExample generates the exact runnable public example,
// then checks protected admission and its declared HTTP and gRPC error mappings.
// No generator behavior changes; this is a consumer of the existing hooks.
func TestConcurrencyAdmissionExample(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "design"), 0o700))
	design, err := os.ReadFile("../../docs/examples/concurrency/design.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "design/design.go"), design, 0o600))
	mod := fmt.Sprintf("module example.com/admission\n\ngo 1.27.3\n\nrequire github.com/CaliLuke/loom v1.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
	for _, args := range [][]string{
		{"get", "-tool", "github.com/CaliLuke/loom/cmd/loom"},
		{"tool", "loom", "gen", "example.com/admission/design"},
	} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, "%v\n%s", args, output)
	}
	for _, file := range []struct{ source, target string }{
		{"../../docs/examples/concurrency/main.go.txt", "main.go"},
		{"testdata/concurrency_admission_test.go.txt", "admission_test.go"},
	} {
		source, err := os.ReadFile(file.source)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, file.target), source, 0o600))
	}
	for _, args := range [][]string{
		{"mod", "tidy"},
		{"test", "-race", "./..."},
	} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, "%v\n%s", args, output)
	}
}
