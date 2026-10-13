package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestErrorDeclarationScopeReuseCompiles(t *testing.T) {
	dir := t.TempDir()
	if output := os.Getenv("LOOM_ERROR_DECLARATION_OUTPUT"); output != "" {
		dir = filepath.Join(output, "scopes")
		require.NoError(t, os.MkdirAll(dir, 0o750))
	}
	renderDesignModule(t, dir, "example.com/errordeclarations", errorDeclarationScopeDSL)
	specs, err := httpcodegen.OpenAPIFiles(codegen.RunDSL(t, errorDeclarationScopeDSL))
	require.NoError(t, err)
	for _, spec := range specs {
		_, err := spec.Render(dir)
		require.NoError(t, err)
	}
	if os.Getenv("LOOM_ERROR_DECLARATION_GENERATE_ONLY") != "" {
		return
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}

func errorDeclarationScopeDSL() {
	dsl.API("errors", func() {
		dsl.JSONRPC(func() {
		})
		dsl.Error("boom", func() {
			dsl.ProblemTitle("API failure")
		})
	})
	for _, transport := range []string{"web", "rpc", "proto"} {
		dsl.Service(transport, func() {
			dsl.Error("boom", func() {
				dsl.ProblemTitle("Service failure")
			})
			if transport == "rpc" {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
			}
			for _, name := range []string{"inherited", "overridden"} {
				dsl.Method(name, func() {
					dsl.Result(dsl.String)
					if name == "overridden" {
						dsl.Error("boom", func() {
							dsl.ProblemTitle("Method failure")
						})
					}
					switch transport {
					case "web":
						dsl.HTTP(func() {
							dsl.GET("/" + name)
							dsl.Response("boom", dsl.StatusBadRequest)
						})
					case "rpc":
						dsl.JSONRPC(func() {
							dsl.Response("boom", -32001)
						})
					case "proto":
						dsl.GRPC(func() {
							dsl.Response("boom", dsl.CodeInvalidArgument)
						})
					}
				})
			}
		})
	}
}
