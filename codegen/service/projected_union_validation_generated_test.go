package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestProjectedUnionValidationGenerated(t *testing.T) {
	for _, test := range []struct {
		name, shape      string
		required, viewed bool
	}{
		{"ordinary required", "number", true, false},
		{"view required", "number", true, true},
		{"view optional", "number", false, true},
		{"view default", "default", false, true},
		{"view strings", "strings", true, true},
		{"view nested", "nested", true, true},
		{"view collections", "collections", true, true},
		{"view array branch", "array", true, true},
		{"selected result branch", "selected", true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := codegen.RunDSL(t, testdata.ProjectedUnionValidationDSL(test.shape, test.required, test.viewed))
			services := NewServicesData(root)
			dir := t.TempDir()
			const module = "example.com/unionvalidation"
			for _, service := range root.Services {
				files := Files(module+"/gen", service, services, make(map[string][]string))
				if views := ViewsFile(module+"/gen", service, services); views != nil {
					files = append(files, views)
				}
				for _, file := range files {
					_, err := file.Render(dir)
					require.NoError(t, err)
				}
			}
			mod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "validation_test.go"), []byte(projectedUnionValidationRuntime(test.shape, test.required, test.viewed)), 0o600))
			for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-race", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}
