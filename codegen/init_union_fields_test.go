package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestInitUnionFieldUsesDeclaredPresence(t *testing.T) {
	union := &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
		{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "Number", Attribute: &expr.AttributeExpr{Type: expr.Int}},
	}}
	named := &expr.UserTypeExpr{TypeName: "NamedChoice", AttributeExpr: &expr.AttributeExpr{Type: union}}
	for _, pointer := range []bool{false, true} {
		plan, err := InitStructFields([]*InitArgData{{Name: "input", Type: named, FieldName: "Value", FieldType: named, FieldPointer: pointer}}, "result", "source", "target", NewNameScope())
		require.NoError(t, err)
		if pointer {
			require.Contains(t, plan.Code, "result.Value = &u")
		} else {
			require.Contains(t, plan.Code, "result.Value = u")
			require.NotContains(t, plan.Code, "result.Value != nil")
		}
	}
}
