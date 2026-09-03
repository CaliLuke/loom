package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	cg "github.com/CaliLuke/loom/codegen"
	d "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/loomsource"
)

func TestSSESharedTypePackageNestedNullableGeneratedModuleCompiles(t *testing.T) {
	t.Cleanup(func() {
		generatorLoader = generators
	})
	generatorLoader = func(string) ([]genfunc, error) {
		return []genfunc{Service, Transport}, nil
	}

	tests := []struct {
		name       string
		localFirst bool
	}{
		{name: "shared path first"},
		{name: "service-local path first", localFirst: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			repoRoot, err := loomsource.RepositoryRoot(".")
			require.NoError(t, err)
			goMod := fmt.Sprintf(`module example.com/ssharedtypes

go 1.27.0

require github.com/CaliLuke/loom v1.0.0

replace github.com/CaliLuke/loom => %s
`, repoRoot)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

			root := cg.RunDSL(t, sseSharedTypePackageDSL(test.localFirst))
			wantFirst := "stream"
			if test.localFirst {
				wantFirst = "show"
			}
			require.Equal(t, wantFirst, root.Services[0].Methods[0].Name)
			_, err = Generate(dir, "gen", false)
			require.NoError(t, err)
			for _, name := range []string{"event.go", "cell.go", "cell_content.go"} {
				require.FileExists(t, filepath.Join(dir, cg.Gendir, "types", name))
			}

			runGeneratedModuleGo(t, dir, "mod", "tidy")
			runGeneratedModuleGo(t, dir, "test", "./...")
		})
	}
}

func sseSharedTypePackageDSL(localFirst bool) func() {
	return func() {
		d.API("sharedtypes", func() {})

		var CellContent = d.Type("CellContent", func() {
			d.Attribute("summary", d.String)
		})
		var LocalCell = d.Type("LocalCell", func() {
			d.Attribute("content", CellContent, func() {
				d.Nullable()
			})
		})
		var Cell = d.Type("Cell", func() {
			d.Attribute("content", CellContent, func() {
				d.Nullable()
			})
		})
		var Event = d.Type("Event", func() {
			d.Meta("struct:pkg:path", "types")
			d.Attribute("cell", Cell)
		})

		d.Service("tabular", func() {
			local := func() {
				d.Method("show", func() {
					d.Result(LocalCell)
					d.HTTP(func() {
						d.GET("/cell")
					})
				})
			}
			stream := func() {
				d.Method("stream", func() {
					d.StreamingResult(Event)
					d.HTTP(func() {
						d.GET("/stream")
						d.ServerSentEvents(func() {
							d.SSEEventData("cell")
						})
					})
				})
			}
			if localFirst {
				local()
				stream()
				return
			}
			stream()
			local()
		})
	}
}

func runGeneratedModuleGo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "go %v failed:\n%s", args, output)
}
