package service

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestDestinationEmissionGenerated(t *testing.T) {
	for _, tc := range []struct {
		name    string
		design  func()
		reverse bool
	}{
		{"shared-custom-forward", destinationRawUnionDSL(false), false},
		{"shared-custom-reverse", destinationRawUnionDSL(false), true},
		{"shared-service-forward", destinationRawUnionDSL(true), false},
		{"shared-service-reverse", destinationRawUnionDSL(true), true},
		{"distinct-nominals", destinationNominalDSL, false},
		{"authored-gen-path", destinationGenPathDSL, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, tc.design)
			services := NewServicesData(root)
			const module = "example.com/destination"
			directory := t.TempDir()
			packages := make(map[string][]string)
			paths := make(map[string]bool)
			order := slices.Clone(root.Services)
			if tc.reverse {
				slices.Reverse(order)
			}
			for _, service := range order {
				require.NoError(t, SetUserTypeImports(module+"/gen", services.Get(service.Name)))
				for _, file := range Files(module+"/gen", service, services, packages) {
					require.False(t, paths[file.Path], "fixture files must not overwrite earlier declarations: %s", file.Path)
					paths[file.Path] = true
					_, err := file.Render(directory)
					require.NoError(t, err)
				}
			}
			mod := fmt.Sprintf("module %s\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
			require.NoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte(mod), 0o600))
			for _, args := range [][]string{{"mod", "tidy"}, {"test", "./..."}} {
				output, err := testingx.RunCmd(directory, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}
