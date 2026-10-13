package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/example"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
)

func TestHTTPModuleAssemblesDiagnostics(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		API("diagnosticassembly", func() {
		})
		auth := JWTSecurity("jwt")
		for _, name := range []string{"secured", "disabled", "ordinary"} {
			Service(name, func() {
				if name != "ordinary" {
					Security(auth)
				}
				Method("show", func() {
					if name == "disabled" {
						NoSecurity()
					}
					Payload(func() {
						if name == "secured" {
							Token("token", String)
						} else {
							Attribute("token", String)
						}
					})
					HTTP(func() {
						GET("/" + name)
						Header("token:X-Token")
					})
				})
			})
		}
	})
	const module = "example.com/diagnosticassembly"
	dir := t.TempDir()
	renderHTTPModule(t, dir, module, root)
	paths, err := filepath.Glob(filepath.Join(dir, "gen/http/*/diagnostics.go"))
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(dir, "gen/http/secured/diagnostics.go")}, paths)
	data := CreateHTTPServices(root)
	renderGeneratedFiles(t, dir, servicecodegen.ExampleServiceFiles(module+"/gen", root, data.ServicesData))
	renderGeneratedFiles(t, dir, servicecodegen.ExampleInterceptorsFiles(module+"/gen", root, data.ServicesData))
	renderGeneratedFiles(t, dir, example.ServerFiles(module+"/gen", root, data.ServicesData))
	renderGeneratedFiles(t, dir, example.CLIFiles(module+"/gen", root))
	renderGeneratedFiles(t, dir, ExampleServerFiles(module+"/gen", data))
	renderGeneratedFiles(t, dir, ExampleCLIFiles(module+"/gen", data))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "build", "./...")
	runGoCommand(t, dir, "vet", "./...")
}
