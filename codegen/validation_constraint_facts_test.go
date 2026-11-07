package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestValidationConstraintFactsPreserveLocalPredicates(t *testing.T) {
	base := &expr.UserTypeExpr{TypeName: "Text", UID: "Text", AttributeExpr: &expr.AttributeExpr{Type: expr.String, DefaultValue: "alpha", Validation: &expr.ValidationExpr{Pattern: "^a"}}}
	field := &expr.AttributeExpr{Type: base, Validation: &expr.ValidationExpr{Pattern: "z$"}}
	_, err := expr.EffectiveConstraintsFor(field)
	require.ErrorContains(t, err, "violates the effective contract")
	ctx := NewAttributeContext(false, false, true, "", NewNameScope())
	var code string
	require.NotPanics(t, func() {
		code = ValidationCode(field, nil, ctx, true, false, false, "value")
	})
	require.Contains(t, code, `"z$"`)
	rules := mergedValidation(field)
	require.Equal(t, []string{"z$", "^a"}, rules.Patterns())
}
