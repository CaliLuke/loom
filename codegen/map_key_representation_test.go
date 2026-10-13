package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestMapKeyPhysicalRepresentations(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  *expr.AttributeExpr
		want string
	}{
		{"bytes", &expr.AttributeExpr{Type: expr.Bytes}, "map[[]byte]string"},
		{"array", &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}, "map[[]string]string"},
		{"any", &expr.AttributeExpr{Type: expr.Any}, "map[any]string"},
		{"custom-comparable", &expr.AttributeExpr{Type: expr.Bytes, Meta: expr.MetaExpr{"struct:field:type": {"[16]byte"}}}, "map[[16]byte]string"},
		{"custom-opaque", &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"custom.Key"}}}, "map[custom.Key]string"},
		{"custom-slice", &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"[]byte"}}}, "map[[]byte]string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mapping := &expr.AttributeExpr{Type: &expr.Map{KeyType: tc.key, ElemType: &expr.AttributeExpr{Type: expr.String}}}
			scope := NewNameScope()
			require.Equal(t, tc.want, scope.GoFullTypeName(mapping, ""))
			require.Equal(t, tc.want, scope.GoTypeDef(mapping, false, false))
		})
	}
}
