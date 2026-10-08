package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
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
	scope := exampleServerImportScope(specs)
	for _, name := range []string{"yaml", "custom"} {
		require.Equal(t, name+"2", scope.Unique(name))
	}
	for _, name := range []string{"", "_", "."} {
		require.Equal(t, name, scope.Unique(name))
	}
}

func TestCLIReservesVersionedImport(t *testing.T) {
	command := &cli.CommandData{Name: "sample", VarName: "sample", PkgName: "yaml"}
	specs := []*codegen.ImportSpec{{Path: "gopkg.in/yaml.v3"}}
	allocated := allocateGRPCCommandIdentifiers([]*cli.CommandData{command}, specs)
	require.Equal(t, "yaml2", allocated[0].PkgName)
	require.Equal(t, "yaml", command.PkgName)
}
