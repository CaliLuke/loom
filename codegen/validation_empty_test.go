package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestValidationEligibilityWithEmptyRules(t *testing.T) {
	for _, pointer := range []bool{false, true} {
		ctx := NewAttributeContext(pointer, false, false, "", NewNameScope())
		field := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{}}
		attribute := &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: field}}}
		user := &expr.UserTypeExpr{TypeName: "Holder", AttributeExpr: attribute}
		require.False(t, hasValidations(ctx, user), "empty lowered rules cannot require a validator")
		attribute.Validation = &expr.ValidationExpr{Required: []string{"value"}}
		require.Equal(t, pointer, hasValidations(ctx, user), "required checks depend on the physical field layout")
	}
}
