package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	d "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestSharedDeclarationSpecimens(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{Service}, nil
	}
	for _, shape := range []string{"collision", "shared-union", "normal"} {
		t.Run(shape, func(t *testing.T) {
			codegen.RunDSL(t, sharedDeclarationDesign(shape))
			dir := t.TempDir()
			if root := os.Getenv("LOOM_DECLARATION_OUTPUT"); root != "" {
				dir = filepath.Join(root, shape)
				require.NoError(t, os.MkdirAll(dir, 0750))
			}
			module := fmt.Sprintf("module example.com/merge\n\ngo 1.27.2\n\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", loomModuleSource(t))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600))
			_, err := Generate(dir, "gen", "github.com/CaliLuke/loom/codegen/generator", false)
			require.NoError(t, err)
			if os.Getenv("LOOM_DECLARATION_GENERATE_ONLY") != "" {
				return
			}
			output, err := testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "test", "./...")
			require.NoError(t, err, output)
		})
	}
}

func sharedDeclarationDesign(shape string) func() {
	return func() {
		d.API("merge", func() {
		})
		first := d.Type("FooBAR", func() {
			d.Meta("struct:pkg:path", "types/common")
			d.Attribute("first", d.String)
		})
		secondName := "FooBar"
		if shape == "normal" {
			secondName = "Other"
		}
		second := d.Type(secondName, func() {
			d.Meta("struct:pkg:path", "types/common")
			d.Attribute("second", d.Int)
		})
		if shape == "shared-union" {
			union := d.Type("Choice", func() {
				d.Meta("struct:pkg:path", "types/common")
				d.OneOf("Choice", func() {
					d.Attribute("first", first)
					d.Attribute("second", second)
				})
			})
			first = union
			second = union
		}
		d.Service("alpha", func() {
			d.Method("read", func() {
				d.Result(first)
			})
		})
		d.Service("beta", func() {
			d.Method("read", func() {
				d.Result(second)
			})
		})
	}
}
