package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNamedOccurrenceValidationUsesDeclarationLayout(t *testing.T) {
	declaration := &expr.UserTypeExpr{TypeName: "Base", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}}},
	}}
	occurrence := &expr.AttributeExpr{Type: declaration, Validation: &expr.ValidationExpr{Required: []string{"name"}}}
	for _, jsonPresence := range []bool{false, true} {
		context := NewAttributeContext(true, false, true, "", NewNameScope())
		context.JSONPresence = !jsonPresence
		context.JSONPresenceTypes = map[string]bool{declaration.Hash(): jsonPresence}
		code := validateNamedOccurrence(context, occurrence, declaration, "value", "", "value", true)
		if jsonPresence {
			require.Contains(t, code, "!value.Name.Present()")
			require.NotContains(t, code, "value.Name == nil")
		} else {
			require.Contains(t, code, "value.Name == nil")
			require.NotContains(t, code, "value.Name.Present()")
		}
	}
}

func TestNamedOccurrenceValidatesRequiredAdditionalKeys(t *testing.T) {
	declaration := &expr.UserTypeExpr{TypeName: "Extras", AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{}, AdditionalAttributes: &expr.AttributeExpr{Type: expr.Int},
		Validation: &expr.ValidationExpr{Required: []string{"base"}},
	}}
	occurrence := &expr.AttributeExpr{Type: declaration, Validation: &expr.ValidationExpr{Required: []string{"a:b"}}}
	context := NewAttributeContext(false, false, true, "", NewNameScope())
	code := validateNamedOccurrence(context, occurrence, declaration, "value", "", "value", true)
	require.Contains(t, code, `value.AdditionalAttributes["a:b"]`)
	require.NotContains(t, code, `value.AdditionalAttributes["base"]`, "intrinsic requirements stay in the named validator")
	require.Contains(t, code, "ValidateExtras(value")
}
