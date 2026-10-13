package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestAccessLogArtifacts(t *testing.T) {
	output := os.Getenv("LOOM_ACCESS_LOG_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_ACCESS_LOG_OUTPUT for isolated output comparison")
	}
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"hosts", testdata.AccessLogCredentialsDSL},
		{"existing", testdata.DiagnosticCredentialsDSL},
		{"controls", testdata.DiagnosticControlsDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			renderAccessLogExample(t, filepath.Join(output, tc.name), tc.design)
		})
	}
}

func TestAccessLogArtifactBuilds(t *testing.T) {
	output := os.Getenv("LOOM_ACCESS_LOG_OUTPUT")
	if output == "" {
		t.Skip("set LOOM_ACCESS_LOG_OUTPUT to compile comparison modules")
	}
	for _, name := range []string{"hosts", "existing", "controls"} {
		t.Run(name, func(t *testing.T) {
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
				logs, err := testingx.RunCmd(filepath.Join(output, name), "go", args...)
				require.NoError(t, err, logs)
			}
		})
	}
}

func renderAccessLogExample(t *testing.T, dir string, design func()) {
	t.Helper()
	codegen.RunDSL(t, design)
	require.NoError(t, os.MkdirAll(dir, 0o700))
	mod := fmt.Sprintf("module example.com/diagnostics\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013051947-ee4552dc94ee\n\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
	for _, command := range []string{"gen", "example"} {
		_, err := Generate(dir, command, "github.com/CaliLuke/loom/codegen/generator", false)
		require.NoError(t, err)
	}
}
