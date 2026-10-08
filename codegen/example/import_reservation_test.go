package example

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
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
	scope := serverMainImportScope(specs)
	for _, name := range []string{"yaml", "custom"} {
		require.Equal(t, name+"2", scope.Unique(name))
	}
	for _, name := range []string{"", "_", "."} {
		require.Equal(t, name, scope.Unique(name))
	}
}
