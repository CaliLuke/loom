package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestRequiredAdditionalMemberDefaultAndValidation(t *testing.T) {
	attribute := &expr.AttributeExpr{
		Type:                 &expr.Object{},
		AdditionalAttributes: &expr.AttributeExpr{Type: expr.Int},
		Validation:           &expr.ValidationExpr{Required: []string{"a:b", "a,b"}},
	}
	context := NewAttributeContext(false, false, true, "", NewNameScope())
	value, err := context.DefaultValue(attribute, map[string]any{"a:b": 0, "a,b": 1})
	require.NoError(t, err)
	require.Contains(t, value.Expression, `AdditionalAttributes: map[string]int{"a,b": 1, "a:b": 0}`)
	_, err = context.DefaultValue(attribute, map[string]any{"a,b": 1})
	require.Error(t, err)
	validation := AttributeValidationCode(attribute, nil, context, true, false, "body", "body")
	require.Contains(t, validation, `if _, ok := body.AdditionalAttributes["a:b"]; !ok`)
	require.Contains(t, validation, `if _, ok := body.AdditionalAttributes["a,b"]; !ok`)
}
