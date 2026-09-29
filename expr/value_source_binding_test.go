package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueSourceBindingSurvivesControlledWrappers(t *testing.T) {
	sourceType := &UserTypeExpr{TypeName: "Payload", AttributeExpr: &AttributeExpr{Type: &Object{{Name: "count", Attribute: &AttributeExpr{Type: Int}}}}}
	source := &AttributeExpr{Type: sourceType}
	declaredType := &UserTypeExpr{TypeName: "AuthoredBody", AttributeExpr: DupAtt(sourceType.AttributeExpr)}
	declared := &AttributeExpr{Type: declaredType}
	bound := DupAtt(declared)
	bindValueSource(bound, source)
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"count": 7}}), ValueRoleExample)
	require.Equal(t, ValueResolved, resolved.Outcome())
	for depth := range 3 {
		require.Same(t, declared, valueAttributeOrigin(bound))
		require.Same(t, source, valueSemanticOrigin(bound))
		plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: bound, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
		require.NoError(t, err)
		require.Equal(t, "AuthoredBody", plan.Root().TargetDeclarationID())
		projected := context.ProjectJSON(resolved, plan)
		require.Equal(t, ProjectionEmitted, projected.Outcome(), "depth %d: %v", depth, projected.Diagnostics())
		wire, present := projected.JSON()
		require.True(t, present)
		require.Equal(t, `{"count":7}`, string(wire))
		bound = httpStreamingBody(&HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: "stream", Stream: ClientStreamKind, StreamingPayload: bound}, Service: &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "bindings"}}})
	}
}

func TestValueSourceBindingCyclesStayInvalidAfterCopy(t *testing.T) {
	source := &AttributeExpr{Type: String}
	other := &AttributeExpr{Type: String}
	source.valueSourceOrigin = other
	other.valueSourceOrigin = source
	require.Nil(t, valueSemanticOrigin(source))
	copied := DupAtt(source)
	require.Nil(t, valueSemanticOrigin(copied), "copying may not turn invalid semantic ancestry into declaration ancestry")
	require.False(t, valueSameAncestry(copied, source), "two cyclic paths are not a valid shared origin")
	stable := &AttributeExpr{Type: String}
	bindValueSource(stable, stable)
	require.Same(t, stable, valueSemanticOrigin(stable), "identity binding must not create a cycle")
}
