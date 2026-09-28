package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestExternalTypeNamesIgnoreServiceReservations(t *testing.T) {
	for _, test := range []struct {
		name   string
		typeOf expr.DataType
	}{
		{"empty object", &expr.Object{}},
		{"object", &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}},
		{"primitive", expr.String},
		{"array", &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
		{"map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Int}}},
		{"union", &expr.Union{TypeName: "Choice"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
				TypeName: "Moved",
				AttributeExpr: &expr.AttributeExpr{
					Type: test.typeOf,
					Meta: expr.MetaExpr{"struct:pkg:path": {"types"}},
				},
			}}
			for _, reserved := range []bool{false, true} {
				scope := NewNameScope()
				if reserved {
					scope.Unique("Moved")
				}
				require.Equal(t, "Moved", scope.GoValueTypeName(attribute), "external declarations use their own package namespace")
				require.Equal(t, "types.Moved", scope.GoFullTypeName(attribute, "types"))
				require.Equal(t, "Moved", scope.GoTypeName(attribute), "repeated references retain the declaration name")
				wantNext := "Moved2"
				if reserved {
					wantNext = "Moved3"
				}
				require.Equal(t, wantNext, scope.Unique("Moved"), "retain existing reservations for generated helper names")
			}
		})
	}
}
