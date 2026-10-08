package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/codegen/service"
)

func TestExampleImportNameReservations(t *testing.T) {
	specs := []*codegen.ImportSpec{
		nil,
		{},
		{Path: "example.com/sideeffect", Name: "_"},
		{Path: "example.com/dsl", Name: "."},
		{Path: "gopkg.in/yaml.v3"},
		{Path: "example.com/pkg", Name: "custom"},
	}
	scope := codegen.NewNameScope()
	reserveExampleImportNames(scope, specs)
	for _, name := range []string{"yaml", "custom"} {
		require.Equal(t, name+"2", scope.Unique(name))
	}
	for _, name := range []string{"", "_", "."} {
		require.Equal(t, name, scope.Unique(name))
	}
}

func TestTransportVariableReservesVersionedImport(t *testing.T) {
	data := &ServiceData{Service: &service.Data{
		PkgName: "service", ViewsPkg: "views",
		UserTypeImports: []*codegen.ImportSpec{{Path: "gopkg.in/yaml.v3"}},
	}}
	scope := newRequestVarScope(data)
	name, _ := scope.allocate("yaml", nil)
	require.Equal(t, "yaml2", name)
}

func TestCLIReservesVersionedImport(t *testing.T) {
	command := &cli.CommandData{Name: "sample", VarName: "sample", PkgName: "yaml"}
	specs := []*codegen.ImportSpec{{Path: "gopkg.in/yaml.v3"}}
	allocated := allocateHTTPCommandIdentifiers([]*commandData{{CommandData: command}}, specs, httpClientCLITransport())
	require.Equal(t, "yaml2", allocated[0].PkgName)
	require.Equal(t, "yaml", command.PkgName)
}
