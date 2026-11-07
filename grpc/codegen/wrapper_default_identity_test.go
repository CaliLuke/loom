package codegen

import (
	"testing"

	shared "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestCollectionWrapperDefaultDeclarationIdentity(t *testing.T) {
	for _, same := range []bool{false, true} {
		left := wrapperContractAttribute(nil, nil)
		left.DefaultValue = []string{"left"}
		right := expr.DupAtt(left)
		if !same {
			right.DefaultValue = []string{"right"}
		}
		before := expr.DupAtt(left)
		sd := &ServiceData{Name: "DefaultIdentity", Scope: shared.NewNameScope()}
		wrapNestedCollection(left, "", sd)
		wrapNestedCollection(right, "", sd)
		if same {
			require.Equal(t, left.Type.(expr.UserType).ID(), right.Type.(expr.UserType).ID())
		} else {
			require.NotEqual(t, left.Type.(expr.UserType).ID(), right.Type.(expr.UserType).ID())
		}
		left.DefaultValue.([]string)[0] = "mutated after registration"
		wrapNestedCollection(before, "", sd)
		require.Equal(t, left.Type.(expr.UserType).ID(), before.Type.(expr.UserType).ID(), "registration owns its default identity snapshot")
	}
}

func TestPhysicalWrapperRequiredPredicateFacts(t *testing.T) {
	source := &expr.AttributeExpr{Type: expr.String, DefaultValue: "default"}
	wrapper := wrapperAttribute(source)
	facts, err := expr.EffectiveConstraintFactsFor(wrapper)
	require.NoError(t, err)
	require.Equal(t, []string{"field"}, facts.Validation().Lowered().Required)
	require.False(t, facts.HasDefault(), "synthetic envelope does not author the contents default")
}
