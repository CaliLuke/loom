package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestUnionValidationUsesPhysicalBranchCarrier(t *testing.T) {
	minimum := 1
	leaf := &expr.UserTypeExpr{TypeName: "Leaf", UID: "leaf", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{MinLength: &minimum}}}},
	}}
	empty := &expr.UserTypeExpr{TypeName: "Empty", UID: "empty", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{},
	}}
	nested := &expr.UserTypeExpr{TypeName: "Nested", UID: "nested", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Union{TypeName: "Nested", Values: []*expr.NamedAttributeExpr{{Name: "leaf", Attribute: &expr.AttributeExpr{Type: leaf}}}},
	}}
	list := &expr.UserTypeExpr{TypeName: "List", UID: "list", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}, Validation: &expr.ValidationExpr{MinLength: &minimum},
	}}
	dict := &expr.UserTypeExpr{TypeName: "Dict", UID: "dict", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}, Validation: &expr.ValidationExpr{MinLength: &minimum},
	}}
	for _, test := range []struct {
		name      string
		attribute *expr.AttributeExpr
		pointer   bool
		call      string
	}{
		{"unconstrained_object", &expr.AttributeExpr{Type: empty}, true, ""},
		{"object", &expr.AttributeExpr{Type: leaf}, true, "ValidateLeaf("},
		{"nested", &expr.AttributeExpr{Type: nested}, true, "switch string(actual.Kind())"},
		{"nullable", &expr.AttributeExpr{Type: nested, Nullable: true}, false, "if !actual.Present()"},
		{"list", &expr.AttributeExpr{Type: list}, false, "ValidateList(actual,"},
		{"map", &expr.AttributeExpr{Type: dict}, false, "ValidateDict(actual,"},
		{"primitive", &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{MinLength: &minimum}}, false, "utf8.RuneCountInString(actual)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			union := &expr.Union{TypeName: "Outer", Values: []*expr.NamedAttributeExpr{{Name: test.name, Attribute: test.attribute}}}
			for _, mode := range []struct {
				name            string
				pointer         bool
				jsonPresence    bool
				validationValue bool
			}{
				{name: "service"},
				{name: "transport", pointer: true},
				{name: "json", jsonPresence: true},
				{name: "extracted", pointer: true, validationValue: true},
			} {
				t.Run(mode.name, func(t *testing.T) {
					ctx := NewAttributeContext(mode.pointer, false, false, "", NewNameScope())
					ctx.JSONPresence = mode.jsonPresence
					ctx.validationValue = mode.validationValue
					cases := renderUnionSumValidationCases(union, nil, ctx, false, "body")
					require.Len(t, cases, 1)
					require.Equal(t, test.pointer, cases[0].RequiresValue)
					if test.name == "list" || test.name == "map" {
						require.NotContains(t, cases[0].Validation, "if actual != nil", "nil collections still need value predicates")
					}
					if test.call == "" {
						require.Empty(t, cases[0].Validation)
						require.NotContains(t, renderUnionSumValidation("branch", cases), "break", "presence-only cases have no invocation to skip")
					} else {
						require.Contains(t, cases[0].Validation, test.call)
					}
				})
			}
		})
	}
}
