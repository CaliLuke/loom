package valuecontract

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// Wrappers change target shape without changing its semantic value or policies.
// The expected outcome/wire comes from the independent Lean input translation,
// never from the unwrapped Go plan or from the plan's own selected source IDs.
func checkProductionProjectionWrappers(t *testing.T, context *expr.ValueContext, occurrence expr.ValueOccurrence, resolved expr.ValueResult, tc productionProjectionCase, use expr.ValuePlanUse, expected productionProjected) {
	t.Helper()
	for _, depth := range []int{1, 2} {
		t.Run(fmt.Sprintf("target wrappers %d", depth), func(t *testing.T) {
			request := productionProjectionPlan(tc, use)
			for index := range depth {
				child := request.Target
				wrapper := expr.DupAtt(child)
				wrapper.Type = &expr.UserTypeExpr{TypeName: fmt.Sprintf("ValueContractWrapper%d", index), AttributeExpr: child}
				request.Target = wrapper
			}
			plan, err := context.NewValuePlan(occurrence, request)
			require.NoError(t, err)
			actual := productionProjectionOutput(t, context.ProjectJSON(resolved, plan))
			require.Equal(t, expected, actual, "controlled wrappers preserve branch identity, presence and target constraints")
		})
	}
}

func productionProjectionOutput(t *testing.T, projected expr.ProjectionResult) productionProjected {
	t.Helper()
	output := productionProjected{outcome: map[expr.ProjectionOutcome]string{
		expr.ProjectionEmitted:         "emitted",
		expr.ProjectionIncomplete:      "incomplete",
		expr.ProjectionUnrepresentable: "unrepresentable",
		expr.ProjectionUnsupported:     "unsupported",
		expr.ProjectionInvalidPlan:     "invalidPlan",
	}[projected.Outcome()]}
	require.NotEmpty(t, output.outcome)
	if wire, present := projected.JSON(); present {
		output.wire = productionCanonical(t, wire)
	}
	return output
}
