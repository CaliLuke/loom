package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
)

func TestCustomDefaultConversionJSONRPC(t *testing.T) {
	root := RunJSONRPCDSL(t, func() {
		API("customdefault", func() {
			JSONRPC(func() {})
		})
		Service("customstream", func() {
			JSONRPC(func() {
				GET("/stream")
			})
			Method("stream", func() {
				StreamingPayload(func() {
					Attribute("token", String, func() {
						Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
						Default("invalid")
					})
				})
				Result(String)
				JSONRPC(func() {})
			})
		})
		Service("customdefault", func() {
			JSONRPC(func() {
				POST("/rpc")
			})
			for _, name := range []string{"good", "bad"} {
				fallback := "550e8400-e29b-41d4-a716-446655440000"
				if name == "bad" {
					fallback = "invalid"
				}
				Method(name, func() {
					fields := func() {
						Attribute("token", String, func() {
							Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
							Default(fallback)
						})
					}
					Payload(fields)
					Result(fields)
					JSONRPC(func() {})
				})
			}
		})
	})
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/customdefault", root)
	services := CreateJSONRPCServices(root)
	for _, file := range ClientCLIFiles("example.com/customdefault/gen", services) {
		if header := file.HeaderTemplate(); header != nil {
			service.AddServiceDataMetaTypeImports(header, services.Get("customdefault").Service)
		}
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	source, err := os.ReadFile("testdata/custom_default_conversion.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "conversion_test.go"), source, 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-race", "-count=1", "./...")
}
