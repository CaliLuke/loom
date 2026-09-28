package service

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

func TestExampleServiceFrameworkImportAliases(t *testing.T) {
	for _, name := range []string{"context", "io", "log", "loom", "security", "ordinary"} {
		t.Run(name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.API("stubs", func() {})
				dsl.Service(name, func() {
					dsl.Method("send", func() {
						dsl.Payload(func() {
							dsl.Attribute("name", dsl.String)
						})
						dsl.StreamingResult(dsl.String)
					})
				})
			})
			services := NewServicesData(root)
			neutral := services.Get(name)
			files := ExampleServiceFiles("example.com/stubs/gen", root, services)
			require.Len(t, files, 1)
			header := codegen.HeaderDataForSection(files[0].HeaderSection())
			var alias string
			for _, spec := range header.Imports {
				if spec.Path == "example.com/stubs/gen/"+name {
					alias = spec.Name
				}
			}
			if name == "ordinary" {
				require.Equal(t, name, alias)
			} else {
				require.Equal(t, name+"2", alias)
			}
			require.Equal(t, name, neutral.PkgName)
			dir := t.TempDir()
			_, err := files[0].Render(dir)
			require.NoError(t, err)
			got := stubSignatures(t, filepath.Join(dir, files[0].Path), alias)
			require.Equal(t, "(context.Context, *SendPayload, SendServerStream) (error)", got["Send"])
		})
	}
}
