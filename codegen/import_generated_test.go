package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestCustomImportPluginGeneratedModule compiles a plugin-style generated file
// whose imports and physical branch reference come from the public helpers.
func TestCustomImportPluginGeneratedModule(t *testing.T) {
	leaf := &expr.AttributeExpr{
		Type: expr.Int,
		Meta: expr.MetaExpr{"struct:field:type": {"constant.Kind", "go/constant"}},
	}
	root := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
		{Name: "kind", Attribute: leaf},
	}}}
	scope := NewNameScope()
	file := &File{
		Path: "gen/plugin/value.go",
		Sections: []Section{
			Header("Custom union branch", "plugin", GetMetaTypeImports(root)),
			NewJenniferSection("value", func(statement *jen.Statement) {
				statement.Type().Id("Value").Struct(jen.Id("Kind").Add(TypeRef(scope.GoTypeRef(leaf))))
			}),
		},
	}
	dir := t.TempDir()
	_, err := file.Render(dir)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/importplugin\n\ngo 1.27.0\n"), 0o600))
	output, err := testingx.RunCmd(dir, "go", "test", "./...")
	require.NoError(t, err, output)
}
