package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueBindingAuthoritySurvivesCopyAndCapture(t *testing.T) {
	source := &AttributeExpr{Type: String}
	plain := DupAtt(source)
	require.NotNil(t, plain.valueSourceOrigin)
	require.False(t, plain.valueSourceBound, "ordinary normalized ancestry is not explicit binding")
	bound := &AttributeExpr{Type: String}
	bindValueSource(bound, source)
	for depth := range 3 {
		bound = DupAtt(bound)
		require.True(t, bound.valueSourceBound)
		occurrence, err := NewValueContext().NewOccurrence(bound)
		require.NoError(t, err)
		require.True(t, occurrence.node.sourceBound, "copy depth %d", depth)
		require.True(t, occurrence.node.attribute.valueSourceBound)
	}
	bound.constraintLocal = &AttributeExpr{Type: String}
	occurrence, err := NewValueContext().NewOccurrence(bound)
	require.NoError(t, err)
	require.True(t, occurrence.node.attribute.valueSourceBound, "local constraint snapshots do not replace occurrence binding")
	require.False(t, source.valueSourceBound)
}

func TestValueBindingAuthoritySurvivesStreamingWrapper(t *testing.T) {
	source := aliasBodySource("Source")
	bound := aliasBodyDeclaration(1)
	bindValueSource(bound, source)
	endpoint := &HTTPEndpointExpr{
		MethodExpr: &MethodExpr{Name: "stream", Stream: ClientStreamKind, StreamingPayload: bound},
		Service:    &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "bindings"}},
	}
	wrapper := httpStreamingBody(endpoint)
	require.True(t, wrapper.valueSourceBound)
	occurrence, err := NewValueContext().NewOccurrence(wrapper)
	require.NoError(t, err)
	require.True(t, occurrence.node.sourceBound)
}

func TestValueBindingAuthorityMemoIsolation(t *testing.T) {
	for _, firstBound := range []bool{false, true} {
		t.Run(fmt.Sprint(firstBound), func(t *testing.T) {
			context := NewValueContext()
			source, err := context.NewOccurrence(&AttributeExpr{Type: String})
			require.NoError(t, err)
			target, err := context.NewOccurrence(foreignAliasBody())
			require.NoError(t, err)
			builder := valuePlanBuilder{context: context, request: ValuePlanRequest{Use: ValuePlanSchema}, nodes: make(map[valuePlanKey]*valuePlanNode)}
			for _, binding := range []bool{firstBound, !firstBound} {
				_, err := builder.nodeAssociated(source.node, target.node, ValueCodecJSON, true, binding)
				if binding {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, "alias has no matching source ancestry")
				}
			}
		})
	}
}

func TestValueBindingAuthorityStopsAtStructuralChildren(t *testing.T) {
	source := aliasBodySource("Source")
	target := DupAtt(source)
	bindValueSource(target, source)
	AsObject(target.Type).Attribute("message").Type = foreignAliasBody().Type
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.ErrorContains(t, err, "alias has no matching source ancestry", "root authority must not authorize an unbound foreign child")
}

func TestValueBindingAuthorityRejectsCyclicSource(t *testing.T) {
	cyclic := &valueOccurrenceNode{origin: &AttributeExpr{Type: String}, declaration: &valueDeclarationNode{}}
	cyclic.declaration.alias = cyclic
	target := &valueOccurrenceNode{origin: &AttributeExpr{Type: String}, declaration: &valueDeclarationNode{alias: cyclic}}
	_, _, err := valuePlanAliasSource(cyclic, target, true)
	require.ErrorContains(t, err, "no matching source ancestry")
}

func TestValueBindingAuthorityRetainsAliasConstraints(t *testing.T) {
	source := &AttributeExpr{Type: String}
	target := foreignAliasBody()
	minimum := 5
	target.Type.(UserType).Attribute().Validation = &ValidationExpr{MinLength: &minimum}
	bindValueSource(target, source)
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	for _, test := range []struct {
		value string
		want  ProjectionOutcome
	}{
		{"short", ProjectionEmitted},
		{"bad", ProjectionUnrepresentable},
	} {
		t.Run(test.value, func(t *testing.T) {
			resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: test.value}), ValueRoleExample)
			require.Equal(t, ValueResolved, resolved.Outcome())
			require.Equal(t, test.want, context.ProjectJSON(resolved, plan).Outcome())
		})
	}
}

func foreignAliasBody() *AttributeExpr {
	return &AttributeExpr{Type: &UserTypeExpr{TypeName: "Unrelated", AttributeExpr: &AttributeExpr{
		Type: &UserTypeExpr{TypeName: "Foreign", AttributeExpr: &AttributeExpr{Type: String}},
	}}}
}
