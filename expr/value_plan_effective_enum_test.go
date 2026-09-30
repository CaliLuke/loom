package expr

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValuePlanKeepsInheritedEnumExcludedByLaterPredicate(t *testing.T) {
	base := namedScalar("PlanEnumBase", String, &ValidationExpr{Values: []any{"ab", "ax"}})
	derived := namedScalar("PlanEnumDerived", base, &ValidationExpr{Pattern: "b$"})
	attribute := &AttributeExpr{Type: derived}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)

	for _, use := range []ValuePlanUse{ValuePlanDocumentation, ValuePlanRuntime} {
		plan, planErr := context.NewValuePlan(occurrence, ValuePlanRequest{
			Target: attribute,
			Codec:  ValueCodecJSON,
			Use:    use,
		})
		require.NoError(t, planErr)
		require.Len(t, plan.root.enumClauses, 1)
		require.Len(t, plan.root.enumClauses[0], 2)
		require.True(t, schemaJSON(plan.root, jsontext.Value(`"ab"`)))
		require.False(t, schemaJSON(plan.root, jsontext.Value(`"ax"`)))
		excluded := context.ResolveDeclaredShape(occurrence,
			context.SupplyValue(ValueInput{Raw: "ax", Origin: "excluded inherited enum member"}))
		require.Equal(t, ValueResolved, excluded.Outcome())
		require.Equal(t, ProjectionUnrepresentable, context.ProjectJSON(excluded, plan).Outcome())
		if use == ValuePlanRuntime {
			_, accepted := decodeJSON(plan.root, jsontext.Value(`"ax"`))
			require.False(t, accepted)
		}
	}
}

func TestValuePlanRejectsMalformedEnumCarrier(t *testing.T) {
	context := NewValueContext()
	source := &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	target := DupAtt(source)
	target.Validation = &ValidationExpr{EnumClauses: [][]any{{[]any{1}}}}

	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: target,
		Codec:  ValueCodecJSON,
		Use:    ValuePlanDocumentation,
	})
	require.ErrorContains(t, err, "declared type")
}
