package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectRetainsObjectOpenness(t *testing.T) {
	for _, policy := range []ObjectOpenness{ObjectOpennessUnset, ObjectOpen, ObjectClosed} {
		for _, selected := range []string{DefaultView, "tiny"} {
			for _, shape := range []string{"root", "nested", "collection"} {
				t.Run(fmt.Sprintf("policy=%d/view=%s/%s", policy, selected, shape), func(t *testing.T) {
					source := nullableProjectionResult(false)
					switch policy {
					case ObjectOpen:
						source.Meta = MetaExpr{objectOpennessMeta: {"true"}}
					case ObjectClosed:
						source.CloseObject()
					}
					root := source
					viewName := selected
					switch shape {
					case "nested":
						root = resultType("child", source, view(DefaultView, "child:"+selected, source))
						viewName = DefaultView
					case "collection":
						root = collection(source)
					}
					projected, err := Project(root, viewName)
					require.NoError(t, err)
					switch shape {
					case "nested":
						projected = projected.Find("child").Type.(*ResultTypeExpr)
					case "collection":
						projected = AsArray(projected.Type).ElemType.Type.(*ResultTypeExpr)
					}
					require.Equal(t, policy, projected.ObjectOpenness())
					require.Equal(t, selected == DefaultView, projected.Find("extra") != nil)
					require.NotNil(t, projected.Find("text"))
					projected.CloseObject()
					require.Equal(t, policy, source.ObjectOpenness(), "projected policy must not alias the source")
				})
			}
		}
	}
}
