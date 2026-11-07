package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestUnionContainerDefaultRepresentation(t *testing.T) {
	choice := &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "number", Attribute: &expr.AttributeExpr{Type: expr.Int}},
	}}
	union := &expr.AttributeExpr{Type: choice}
	mapping := &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: union}}
	array := &expr.AttributeExpr{Type: &expr.Array{ElemType: union}}
	object := &expr.UserTypeExpr{TypeName: "Envelope", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "choice", Attribute: union},
		{Name: "label", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}, Validation: &expr.ValidationExpr{Required: []string{"label"}}}}
	cases := []struct {
		name      string
		attribute *expr.AttributeExpr
		raw       any
		want      string
	}{
		{"map", mapping, map[string]any{"a": "hello", "b": 2}, `map[string]svc.Choice{"a": svc.NewChoiceText("hello"), "b": svc.NewChoiceNumber(2)}`},
		{"array", array, []any{"hello", 2}, `[]svc.Choice{svc.NewChoiceText("hello"), svc.NewChoiceNumber(2)}`},
		{"empty map", mapping, map[string]any{}, `map[string]svc.Choice{}`},
		{"empty array", array, []any{}, `[]svc.Choice{}`},
		{"object optional union", &expr.AttributeExpr{Type: object}, map[string]any{"choice": "hello", "label": "kept"}, `&svc.Envelope{Choice: func() *svc.Choice { v := svc.NewChoiceText("hello"); return &v }(), Label: "kept"}`},
		{"object omitted union", &expr.AttributeExpr{Type: object}, map[string]any{"label": "kept"}, `&svc.Envelope{Label: "kept"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			context := NewAttributeContext(false, false, true, "svc", NewNameScope())
			plan, err := context.DefaultValue(tc.attribute, tc.raw)
			require.NoError(t, err)
			require.False(t, plan.ErrorAware)
			require.Equal(t, tc.want, plan.Expression)
		})
	}
}

func TestUnionContainerDefaultRejectsInvalidValue(t *testing.T) {
	for _, name := range []string{"invalid", "ambiguous"} {
		t.Run(name, func(t *testing.T) {
			alternatives := []*expr.NamedAttributeExpr{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}
			raw := any(true)
			if name == "ambiguous" {
				alternatives = append(alternatives, &expr.NamedAttributeExpr{Name: "other", Attribute: &expr.AttributeExpr{Type: expr.String}})
				raw = "hello"
			}
			choice := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Untagged: true, Values: alternatives}}
			target := &expr.AttributeExpr{Type: &expr.Array{ElemType: choice}}
			context := NewAttributeContext(false, false, true, "svc", NewNameScope())
			_, err := context.DefaultValue(target, []any{raw})
			require.ErrorContains(t, err, "cannot resolve value")
		})
	}
}

func TestUnionDefaultDeclarationAndBranchStorage(t *testing.T) {
	choice := &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "number", Attribute: &expr.AttributeExpr{Type: expr.Int}},
	}}
	required := &expr.UserTypeExpr{TypeName: "Envelope", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "choice", Attribute: &expr.AttributeExpr{Type: choice}},
	}, Validation: &expr.ValidationExpr{Required: []string{"choice"}}}}
	outer := &expr.Union{TypeName: "Outer", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "nested", Attribute: &expr.AttributeExpr{Type: choice}},
		{Name: "flag", Attribute: &expr.AttributeExpr{Type: expr.Boolean}},
	}}
	leaf := &expr.UserTypeExpr{TypeName: "Leaf", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int, DefaultValue: 3}},
	}, Validation: &expr.ValidationExpr{Required: []string{"name"}}}}
	objectUnion := &expr.Union{TypeName: "ObjectChoice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "leaf", Attribute: &expr.AttributeExpr{Type: leaf}},
		{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}
	anyChoice := &expr.Union{TypeName: "AnyChoice", Values: []*expr.NamedAttributeExpr{
		{Name: "Kind", Attribute: &expr.AttributeExpr{Type: expr.Any}},
	}}
	wrapper := &expr.UserTypeExpr{TypeName: "NamedChoice", AttributeExpr: &expr.AttributeExpr{Type: choice}}
	located := &expr.UserTypeExpr{TypeName: "LocatedChoice", AttributeExpr: &expr.AttributeExpr{Type: choice, Meta: expr.MetaExpr{"struct:pkg:path": {"types/choices"}}}}
	array := func(attribute *expr.AttributeExpr) *expr.AttributeExpr {
		return &expr.AttributeExpr{Type: &expr.Array{ElemType: attribute}}
	}
	cases := []struct {
		name      string
		attribute *expr.AttributeExpr
		raw       any
		want      string
	}{
		{"required field", &expr.AttributeExpr{Type: required}, map[string]any{"choice": 2}, `&svc.Envelope{Choice: svc.NewChoiceNumber(2)}`},
		{"nested union branch", array(&expr.AttributeExpr{Type: outer}), []any{"hello"}, `[]svc.Outer{svc.NewOuterNested(func() *svc.Choice { v := svc.NewChoiceText("hello"); return &v }())}`},
		{"object branch nested default", array(&expr.AttributeExpr{Type: objectUnion}), []any{map[string]any{"name": "Ada"}}, `[]svc.ObjectChoice{svc.NewObjectChoiceLeaf(&svc.Leaf{Name: "Ada", Count: 3})}`},
		{"native Any alias branch", array(&expr.AttributeExpr{Type: anyChoice}), []any{map[string]any{"x": 1}}, `[]svc.AnyChoice{svc.NewAnyChoiceKind(loom.JSONValue("{\"x\":1}"))}`},
		{"named wrapper", array(&expr.AttributeExpr{Type: wrapper}), []any{2}, `[]svc.NamedChoice{svc.NewNamedChoiceNumber(2)}`},
		{"located wrapper", array(&expr.AttributeExpr{Type: located}), []any{2}, `[]choices.LocatedChoice{choices.NewLocatedChoiceNumber(2)}`},
		{"outer nullable union", array(&expr.AttributeExpr{Type: choice, Nullable: true}), []any{nil, "hello"}, `[]loom.Nullable[svc.Choice]{loom.NullValue[svc.Choice](), loom.NullableValue[svc.Choice](svc.NewChoiceText("hello"))}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			context := NewAttributeContext(false, false, true, "svc", NewNameScope())
			plan, err := context.DefaultValue(tc.attribute, tc.raw)
			require.NoError(t, err)
			require.Equal(t, tc.want, plan.Expression)
		})
	}
}

func TestUnionDefaultUsesAllocatedSanitizedConstructor(t *testing.T) {
	choice := &expr.Union{TypeName: "Choice", Untagged: true, Values: []*expr.NamedAttributeExpr{
		{Name: "x_y", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"second"}}}},
		{Name: "x-y", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"first"}}}},
	}}
	scope := NewNameScope()
	scope.Unique("Choice")
	scope.GoValueTypeName(&expr.AttributeExpr{Type: choice})
	context := NewAttributeContext(false, false, true, "svc", scope)
	attribute := &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: choice}}}
	plan, err := context.DefaultValue(attribute, []any{"second", "first"})
	require.NoError(t, err)
	require.Equal(t, `[]svc.Choice2{svc.NewChoice2XY2("second"), svc.NewChoice2XY("first")}`, plan.Expression)
}
