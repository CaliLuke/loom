package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestCollectionReferenceMatchesDeclaration(t *testing.T) {
	var source strings.Builder
	source.WriteString("package carriers\nimport loom \"github.com/CaliLuke/loom/pkg\"\n")
	index := 0
	for _, nullable := range []bool{false, true} {
		for _, mapping := range []bool{false, true} {
			name := "array"
			if mapping {
				name = "map"
			}
			if nullable {
				name += " nullable"
			}
			t.Run(name, func(t *testing.T) {
				element := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}, Nullable: nullable}
				attribute := &expr.AttributeExpr{Type: &expr.Array{ElemType: element}}
				if mapping {
					attribute.Type = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: element}
				}
				scope := NewNameScope()
				definition, reference := scope.GoTypeDef(attribute, false, false), scope.GoTypeRef(attribute)
				require.Equal(t, definition, reference)
				fmt.Fprintf(&source, "func identity%d(value %s) %s { return value }\n", index, definition, reference)
				index++
				if nullable {
					require.Equal(t, scope.GoTypeDef(element, false, false), scope.GoTypeRef(element))
				}
			})
		}
	}

	dir := t.TempDir()
	module := fmt.Sprintf("module example.com/carrierrefs\n\ngo 1.27.3\n\nrequire github.com/CaliLuke/loom v0.0.0\nreplace github.com/CaliLuke/loom => %s\n", testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "carriers.go"), []byte(source.String()), 0o600))
	output, err := testingx.RunCmd(dir, "go", "test", "-mod=mod", "./...")
	require.NoError(t, err, output)
}
