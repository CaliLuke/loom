package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestInitStructFieldsTargetLayout(t *testing.T) {
	for _, pointer := range []bool{false, true} {
		name := "native value fields"
		if pointer {
			name = "viewed pointer fields"
		}
		t.Run(name, func(t *testing.T) {
			source := &expr.AttributeExpr{Type: &expr.UserTypeExpr{
				TypeName: "WireRecord", UID: "wire-record",
				AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
					&expr.NamedAttributeExpr{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
					&expr.NamedAttributeExpr{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int}},
				}, Validation: &expr.ValidationExpr{Required: []string{"name", "count"}}},
			}}
			target := expr.DupAtt(source)
			ut := target.Type.(*expr.UserTypeExpr)
			ut.TypeName, ut.UID = "Record", "record"
			scope := NewNameScope()
			sourceContext := NewAttributeContext(true, false, false, "wire", scope)
			targetContext := NewAttributeContext(pointer, false, !pointer, "native", scope)
			plan, err := InitStructFields([]*InitArgData{{
				Name: "range_", Type: source.Type, Attribute: source, SourceContext: sourceContext,
				FieldName: "Range", FieldType: target.Type, FieldAttribute: target, TargetContext: targetContext,
			}}, "v", "wire", "native", scope)
			require.NoError(t, err)
			for _, field := range []string{"Name", "Count"} {
				if pointer {
					require.NotContains(t, plan.Code, "*range_."+field, "viewed destination fields retain pointers")
				} else {
					require.Contains(t, plan.Code, "*range_."+field, "ordinary required fields retain native values")
				}
			}
		})
	}
}
