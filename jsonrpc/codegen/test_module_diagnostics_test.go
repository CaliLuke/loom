package codegen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/example"
	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func TestJSONRPCModuleAssemblesDiagnostics(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		name := "jsonrpc"
		if mixed {
			name = "mixed"
		}
		t.Run(name, func(t *testing.T) {
			root := RunJSONRPCDSL(t, func() {
				API("diagnosticassembly", func() {
					JSONRPC(func() {
					})
				})
				auth := JWTSecurity("jwt")
				for _, service := range []string{"secured", "disabled", "ordinary"} {
					Service(service, func() {
						if service != "ordinary" {
							Security(auth)
						}
						JSONRPC(func() {
							POST("/rpc/" + service)
						})
						Method("call", func() {
							if service == "disabled" {
								NoSecurity()
							}
							Payload(func() {
								ID("id", String)
								if service == "secured" {
									Token("token", String)
								} else {
									Attribute("token", String)
								}
							})
							JSONRPC(func() {
								Header("token:X-Token")
							})
						})
						if mixed {
							Method("show", func() {
								if service == "disabled" {
									NoSecurity()
								}
								Payload(func() {
									if service == "secured" {
										Token("token", String)
									} else {
										Attribute("token", String)
									}
								})
								HTTP(func() {
									GET("/" + service)
									Header("token:X-Token")
								})
							})
						}
					})
				}
			})
			const module = "example.com/diagnosticassembly"
			dir := t.TempDir()
			if mixed {
				renderMixedHTTPJSONRPCModule(t, dir, module, root)
			} else {
				renderJSONRPCModule(t, dir, module, root)
			}
			paths, err := filepath.Glob(filepath.Join(dir, "gen/*/*/diagnostics.go"))
			require.NoError(t, err)
			want := []string{filepath.Join(dir, "gen/jsonrpc/secured/diagnostics.go")}
			if mixed {
				want = append(want, filepath.Join(dir, "gen/http/secured/diagnostics.go"))
			}
			require.ElementsMatch(t, want, paths)
			data := CreateJSONRPCServices(root)
			renderCodegenFiles(t, dir, servicecodegen.ExampleServiceFiles(module+"/gen", root, data.ServicesData))
			renderCodegenFiles(t, dir, servicecodegen.ExampleInterceptorsFiles(module+"/gen", root, data.ServicesData))
			renderCodegenFiles(t, dir, example.ServerFiles(module+"/gen", root, data.ServicesData))
			renderCodegenFiles(t, dir, example.CLIFiles(module+"/gen", root))
			httpData := httpcodegen.NewServicesData(data.ServicesData, root.API.HTTP)
			renderCodegenFiles(t, dir, ClientCLIFiles(module+"/gen", data))
			renderCodegenFiles(t, dir, ExampleServerFiles(module+"/gen", data, httpcodegen.ExampleServerFiles(module+"/gen", httpData)))
			renderCodegenFiles(t, dir, ExampleCLIFiles(module+"/gen", data))
			if mixed {
				renderCodegenFiles(t, dir, httpcodegen.ClientCLIFiles(module+"/gen", httpData))
				renderCodegenFiles(t, dir, httpcodegen.ExampleCLIFiles(module+"/gen", httpData))
			}
			runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
			runGoJSONRPCTestCommand(t, dir, "build", "./...")
			runGoJSONRPCTestCommand(t, dir, "vet", "./...")
		})
	}
}
