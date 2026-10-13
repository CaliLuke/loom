package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func TestAnonymousCollectionWrapperContractIdentity(t *testing.T) {
	minimum := 2
	maximum := 4
	for _, tc := range []struct {
		name                                   string
		left, right, leftElement, rightElement *expr.ValidationExpr
		same                                   bool
	}{
		{name: "unconstrained sharing", same: true},
		{name: "equal length sharing", left: &expr.ValidationExpr{MinLength: &minimum}, right: &expr.ValidationExpr{MinLength: &minimum}, same: true},
		{name: "minimum differs", right: &expr.ValidationExpr{MinLength: &minimum}},
		{name: "maximum differs", right: &expr.ValidationExpr{MaxLength: &maximum}},
		{name: "element enum differs", rightElement: &expr.ValidationExpr{Values: []any{"one"}}},
		{name: "element pattern differs", rightElement: &expr.ValidationExpr{Pattern: "^one$"}},
		{name: "element format differs", rightElement: &expr.ValidationExpr{Format: expr.FormatEmail}},
		{name: "equal element clauses share", leftElement: &expr.ValidationExpr{Pattern: "^one$"}, rightElement: &expr.ValidationExpr{Pattern: "^one$"}, same: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left := wrapperContractAttribute(tc.left, tc.leftElement)
			right := wrapperContractAttribute(tc.right, tc.rightElement)
			sd := &ServiceData{Name: "Contract", Scope: codegen.NewNameScope()}
			wrapNestedCollection(left, "", sd)
			wrapNestedCollection(right, "", sd)
			leftType := left.Type.(expr.UserType)
			rightType := right.Type.(expr.UserType)
			if tc.same {
				require.Equal(t, leftType.ID(), rightType.ID())
			} else {
				require.NotEqual(t, leftType.ID(), rightType.ID(), "complete named validators cannot share an identity across different contents contracts")
			}
		})
	}
}

func wrapperContractAttribute(validation, element *expr.ValidationExpr) *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type:       &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String, Validation: element}},
		Validation: validation,
	}
}
