package expr

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPreparedSchemaCompactNumericMembership(t *testing.T) {
	for _, test := range []struct {
		name string
		kind DataType
		wire string
		want bool
	}{
		{"huge integer is integral", Int64, "1e1000001", true},
		{"tiny decimal is fractional", Int64, "1e-1000001", false},
		{"tiny decimal is a number", Float64, "1e-1000001", true},
		{"ordinary number", Float64, "1", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			attribute := &AttributeExpr{Type: test.kind}
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
			require.NoError(t, err)
			matched, err := plan.MatchJSONSchema([]byte(test.wire))
			require.NoError(t, err)
			require.Equal(t, test.want, matched, "schema number membership is independent of finite target decoding")
		})
	}
}

func TestPreparedSchemaEvaluatorErrorsDoNotSelectBranches(t *testing.T) {
	valid := valuePlanBranch{node: &valuePlanNode{kind: Float64Kind, codec: ValueCodecJSON}}
	broken := valuePlanBranch{node: &valuePlanNode{kind: Kind(255), codec: ValueCodecJSON}}
	for _, branches := range [][]valuePlanBranch{{broken, valid}, {valid, broken}, {valid, valid, broken}} {
		plan := ValuePlan{root: &valuePlanNode{kind: UnionKind, codec: ValueCodecJSON, untagged: true, branches: branches}}
		matched, err := plan.MatchJSONSchema([]byte("1"))
		require.ErrorContains(t, err, "unsupported JSON schema kind")
		require.False(t, matched)
	}
}

func TestCustomProjectionRetainsSchemaEvaluatorError(t *testing.T) {
	context := NewValueContext()
	attribute := &AttributeExpr{Type: Any}
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	var calls atomic.Int32
	raw := &projectionCustom{calls: &calls, wire: "1"}
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	// Deliberately corrupt an internal plan after admission to exercise the
	// evaluator-failure channel; valid authored plans cannot create this state.
	plan.root.kind = Kind(255)
	result := context.ProjectJSON(resolved, plan)
	require.Equal(t, ProjectionInvalidPlan, result.Outcome())
	require.ErrorContains(t, result.Err(), "unsupported JSON schema kind")
	_, emitted := result.JSON()
	require.False(t, emitted)
}

func TestBinaryBoundaryArtifactsRequireSchemaMembership(t *testing.T) {
	for _, tc := range []struct {
		name       string
		primitive  Primitive
		value      any
		validation *ValidationExpr
	}{
		{"float64 tenth", Float64, 0.1, &ValidationExpr{Minimum: boundForTest(0.1)}},
		{"float32 tenth", Float32, float32(0.1), &ValidationExpr{Minimum: boundForTest(float32(0.1))}},
		{"float64 fixture", Float64, 100.1, &ValidationExpr{Maximum: boundForTest(100.1)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attribute := &AttributeExpr{Type: tc.primitive, Validation: tc.validation}
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
			require.NoError(t, err)
			for _, role := range []ValueRole{ValueRoleExample, ValueRoleDefault, ValueRoleEnum} {
				resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.value}), role)
				require.Equal(t, ValueResolved, resolved.Outcome())
				projection := context.ProjectJSON(resolved, plan)
				require.Equal(t, ProjectionUnrepresentable, projection.Outcome())
				_, emitted := projection.JSON()
				require.False(t, emitted)
			}
		})
	}
}
