package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAdmittedDefaultContainerMembers(t *testing.T) {
	leaf := &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 9}},
	}, Validation: &expr.ValidationExpr{Required: []string{"name"}}}}
	choice := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "leaf", Attribute: &expr.AttributeExpr{Type: leaf}},
		{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}}
	for _, tc := range []struct {
		name string
		typ  expr.DataType
		raw  any
	}{
		{"named array", &expr.Array{ElemType: choice}, []any{map[string]any{"name": "Ada"}, "kept"}},
		{"named map", &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: choice}, map[string]any{"first": map[string]any{"name": "Ada"}, "second": "kept"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			container := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Selected", AttributeExpr: &expr.AttributeExpr{Type: tc.typ}}}
			context := NewAttributeContext(true, false, false, "", NewNameScope())
			ordinary, err := context.DefaultValue(container, tc.raw)
			require.NoError(t, err)
			require.NotContains(t, ordinary.Expression, "Count:")
			admitted, err := context.DefaultValue(container, tc.raw, DefaultValueOptions{MaterializeMembers: true})
			require.NoError(t, err)
			require.Contains(t, admitted.Expression, "Selected{")
			require.Contains(t, admitted.Expression, "NewChoiceLeaf(&Leaf{")
			require.Contains(t, admitted.Expression, `NewChoiceText("kept")`)
			require.Contains(t, admitted.Expression, "Count:")
			require.Contains(t, admitted.Expression, "int(9)")
			require.True(t, context.Pointer, "materialization timing must not alter source layout")
			require.False(t, context.UseDefault)
		})
	}
}

func TestAdmittedDefaultPreservesNestedSuppliedValue(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 9}}}}
	context := NewAttributeContext(true, false, false, "", NewNameScope())
	literal, err := context.DefaultValue(attribute, map[string]any{"count": 0}, DefaultValueOptions{MaterializeMembers: true})
	require.NoError(t, err)
	require.Contains(t, literal.Expression, "int(0)")
	require.NotContains(t, literal.Expression, "int(9)")
}

func TestAdmittedDefaultRejectsInfiniteMembers(t *testing.T) {
	node := &expr.UserTypeExpr{TypeName: "Node", AttributeExpr: &expr.AttributeExpr{}}
	node.Type = &expr.Object{{Name: "next", Attribute: &expr.AttributeExpr{Type: node, DefaultValue: map[string]any{}}}}
	context := NewAttributeContext(true, false, false, "", NewNameScope())
	attribute := &expr.AttributeExpr{Type: node}
	_, err := context.DefaultValue(attribute, map[string]any{})
	require.NoError(t, err, "ordinary deferred source layout leaves omitted members absent")
	_, err = context.DefaultValue(attribute, map[string]any{}, DefaultValueOptions{MaterializeMembers: true})
	require.ErrorContains(t, err, "nested defaults require an infinite object value")
}
