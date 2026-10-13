package codegen_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	httptestdata "github.com/CaliLuke/loom/http/codegen/testdata"
	rpccodegen "github.com/CaliLuke/loom/jsonrpc/codegen"
	rpctestdata "github.com/CaliLuke/loom/jsonrpc/codegen/testdata"
)

func TestConstructorJSONRPCPackageInventory(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"ordinary", func() {
			httptestdata.ConstructorOwnership("ordinary", true, false)
		}},
		{"websocket", rpctestdata.ViewedStreamResultDSL},
		{"sse", rpctestdata.JSONRPCSSEObjectDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := rpccodegen.RunJSONRPCDSL(t, tc.design)
			data := rpccodegen.CreateJSONRPCServices(root)
			const genpkg = "example.com/inventory/gen"
			files := make([]*codegen.File, 0, len(data.Expressions.Services)*7)
			files = append(files, rpccodegen.PathFiles(data)...)
			files = append(files, rpccodegen.ClientTypeFiles(genpkg, data)...)
			files = append(files, rpccodegen.ClientFiles(genpkg, data)...)
			files = append(files, rpccodegen.ClientCLIFiles(genpkg, data)...)
			files = append(files, rpccodegen.ServerTypeFiles(genpkg, data)...)
			files = append(files, rpccodegen.ServerFiles(genpkg, data)...)
			files = append(files, rpccodegen.SSEServerFiles(genpkg, data)...)
			dir := t.TempDir()
			for _, file := range files {
				_, err := file.Render(dir)
				require.NoError(t, err)
			}
			for _, service := range data.Expressions.Services {
				sd := data.Get(service.Name())
				httpcodegen.CheckConstructorPackageInventoryForTest(t, sd, filepath.Join(dir, "gen", "jsonrpc", sd.Service.PathName))
			}
		})
	}
}
