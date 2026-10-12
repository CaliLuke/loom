package codegen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestTypedCollectionDefaultsGenerated(t *testing.T) {
	root := RunJSONRPCDSL(t, testdata.TypedCollectionDefaultsDSL("jsonrpc"))
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, "example.com/defaults", root)
	serviceData := servicecodegen.NewServicesData(root)
	for _, svc := range root.Services {
		require.NoError(t, servicecodegen.SetUserTypeImports("example.com/defaults/gen", serviceData.Get(svc.Name)))
	}
	transportData := httpcodegen.NewServicesData(serviceData, &root.API.JSONRPC.HTTPExpr)
	for _, file := range ClientCLIFiles("example.com/defaults/gen", transportData) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	module, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	module = []byte(strings.ReplaceAll(string(module), "require github.com/CaliLuke/loom v1.0.0", "require github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012201303-8c7d8d711b9e"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), module, 0o600))
	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "build", "./...")
	runGoJSONRPCTestCommand(t, dir, "vet", "./...")
}
