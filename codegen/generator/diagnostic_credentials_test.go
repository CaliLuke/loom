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

func TestDiagnosticCredentialExampleWiring(t *testing.T) {
	for _, tc := range []struct {
		name       string
		design     func()
		credential bool
	}{
		{"credentials", testdata.DiagnosticCredentialsDSL, true},
		{"controls", testdata.DiagnosticControlsDSL, false},
		{"body policies", testdata.DiagnosticBodyPoliciesDSL, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codegen.RunDSL(t, tc.design)
			dir := t.TempDir()
			mod := fmt.Sprintf("module example.com/diagnostics\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261013051947-ee4552dc94ee\n\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0600))
			_, err := Generate(dir, "gen", "github.com/CaliLuke/loom/codegen/generator", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", "github.com/CaliLuke/loom/codegen/generator", false)
			require.NoError(t, err)
			if tc.credential {
				for _, transport := range []struct {
					name, service string
				}{{"http", "web"}, {"jsonrpc", "rpc"}} {
					source, err := os.ReadFile(filepath.Join(dir, "gen", transport.name, transport.service, "diagnostics.go"))
					require.NoError(t, err)
					require.Contains(t, string(source), `"permit"`)
					cli, err := os.ReadFile(filepath.Join(dir, "cmd", "diagnostics-cli", transport.name+".go"))
					require.NoError(t, err)
					require.Contains(t, string(cli), ".Fields()")
				}
			} else if tc.name == "controls" {
				for _, service := range []string{"ordinary", "disabled"} {
					require.NoFileExists(t, filepath.Join(dir, "gen", "http", service, "diagnostics.go"))
				}
			}
			if tc.name != "controls" {
				cases := diagnosticMixedWireCases
				if tc.name == "body policies" {
					cases = diagnosticBodyWireCases
				}
				require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), []byte(diagnosticWireHarness), 0600))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "cases_test.go"), []byte(cases), 0600))
			}

			output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "build", "./...")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "test", "-race", "./...")
			require.NoError(t, err, output)
		})
	}
}
