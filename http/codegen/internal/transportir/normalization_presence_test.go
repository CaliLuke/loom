package transportir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestNormalizeCollectionElementNullability preserves nullability inherited
// from named scalar elements when normalization removes their names.
func TestNormalizeCollectionElementNullability(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		for _, container := range []string{"array", "map"} {
			name := container + "/non-null"
			if nullable {
				name = container + "/nullable"
			}
			t.Run(name, func(t *testing.T) {
				alias := &expr.UserTypeExpr{TypeName: "Element", AttributeExpr: &expr.AttributeExpr{
					Type: expr.String, Nullable: nullable,
				}}
				element := &expr.AttributeExpr{Type: alias}
				attribute := &expr.AttributeExpr{Type: &expr.Array{ElemType: element}}
				if container == "map" {
					attribute.Type = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: element}
				}
				normalized := normalizeHTTPAttribute(attribute)
				var actual *expr.AttributeExpr
				if container == "array" {
					actual = expr.AsArray(normalized.Type).ElemType
				} else {
					actual = expr.AsMap(normalized.Type).ElemType
				}
				require.Equal(t, expr.String, actual.Type)
				require.Equal(t, nullable, expr.IsNullable(actual))
				require.Same(t, alias, element.Type)
				require.False(t, element.Nullable)
			})
		}
	}
}
