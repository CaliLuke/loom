package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestInitStructFieldsNominalScope checks casts and nested conversions against
// declaration identities, independently of constructor-local variable names.
func TestInitStructFieldsNominalScope(t *testing.T) {
	for _, shape := range []string{"scalar", "pointer", "array", "map"} {
		for _, located := range []bool{false, true} {
			t.Run(shape+map[bool]string{false: "/service", true: "/located"}[located], func(t *testing.T) {
				scope := NewNameScopeWithPackageNames(map[string]string{"types/shared": "shared2"})
				scope.Unique("Client")
				alias := &expr.UserTypeExpr{
					TypeName:      "Client",
					UID:           "constructor-client",
					AttributeExpr: &expr.AttributeExpr{Type: expr.String},
				}
				pkg := "labels"
				if located {
					alias.Meta = expr.MetaExpr{"struct:pkg:path": []string{"types/shared"}}
					pkg = "shared2"
				}
				att := &expr.AttributeExpr{Type: alias}
				name := "Client2"
				if located {
					name = "Client"
				}
				require.Equal(t, name, scope.GoTypeName(att))
				want := scope.GoFullTypeRef(att, pkg)
				require.Equal(t, pkg+"."+name, want)
				var source expr.DataType = expr.String
				var target expr.DataType = alias
				switch shape {
				case "array":
					source = &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}
					target = &expr.Array{ElemType: att}
				case "map":
					source = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}
					target = &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: att}
				}
				args := []*InitArgData{{
					Name: "value", Type: source, FieldName: "Value", FieldType: target,
					Pointer: shape == "pointer", FieldPointer: shape == "pointer",
				}}
				for _, local := range []string{"v", "Client2"} {
					plan, err := InitStructFields(args, local, "", "labels", scope)
					require.NoError(t, err)
					require.Contains(t, plan.Code, want)
					require.Equal(t, want, scope.GoFullTypeRef(att, pkg))
				}
				require.Equal(t, "v", scope.Unique("v"), "constructor locals must not reserve package declarations")
			})
		}
	}
}
