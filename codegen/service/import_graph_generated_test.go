package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestImportGraphGeneratedModule uses the UnionMethodDSL branch shape with a
// custom integer alias and a recursive service field to cover import traversal.
func TestImportGraphGeneratedModule(t *testing.T) {
	for _, custom := range []bool{false, true} {
		name := map[bool]string{false: "native", true: "custom"}[custom]
		t.Run(name, func(t *testing.T) {
			const module = "example.com/importgraph"
			root := codegen.RunDSL(t, func() {
				node := dsl.Type("Node", func() {
					dsl.Attribute("next", "Node")
					dsl.OneOf("values", func() {
						dsl.Attribute("int", dsl.Int, func() {
							if custom {
								dsl.Meta("struct:field:type", "constant.Kind", "go/constant")
							}
						})
						dsl.Attribute("string", dsl.String)
						dsl.Attribute("boolean", dsl.Boolean)
						dsl.Attribute("bytes", dsl.Bytes)
					})
				})
				dsl.Service("graph", func() {
					dsl.Method("echo", func() {
						dsl.Payload(node)
						dsl.Result(node)
					})
				})
			})
			services := NewServicesData(root)
			dir := t.TempDir()
			for _, service := range root.Services {
				for _, file := range Files(module+"/gen", service, services, make(map[string][]string)) {
					_, err := file.Render(dir)
					require.NoError(t, err)
				}
			}
			mod := fmt.Sprintf("module %s\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
			for _, args := range [][]string{{"mod", "tidy"}, {"vet", "./..."}, {"build", "./..."}} {
				output, err := testingx.RunCmd(dir, "go", args...)
				require.NoError(t, err, output)
			}
		})
	}
}
