package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueAliasOwnership(t *testing.T) {
	for _, reversed := range []bool{false, true} {
		object := &Object{
			{Name: "body:shared", Attribute: &AttributeExpr{Type: String}},
			{Name: "header:shared", Attribute: &AttributeExpr{Type: String}},
		}
		if reversed {
			(*object)[0], (*object)[1] = (*object)[1], (*object)[0]
		}
		attribute := &AttributeExpr{Type: object, Validation: &ValidationExpr{Required: []string{"body:shared", "header:shared"}}}
		context := NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err, "distinct source members may belong to separate transport representations")
		members := occurrence.Members()
		require.Len(t, members, 2)
		require.NotEqual(t, members[0].ID, members[1].ID)
		for _, tc := range []struct {
			name    string
			raw     map[string]any
			outcome ValueOutcome
		}{
			{"authored keys", map[string]any{"body:shared": "body", "header:shared": "header"}, ValueResolved},
			{"ambiguous alias", map[string]any{"shared": "same"}, ValueAmbiguous},
			{"alias with authored keys", map[string]any{"shared": "same", "body:shared": "body", "header:shared": "header"}, ValueAmbiguous},
		} {
			for _, role := range []ValueRole{ValueRoleExample, ValueRoleEnum, ValueRoleDefault} {
				result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.raw}), role)
				require.Equal(t, tc.outcome, result.Outcome(), "%s reversed=%t role=%v: %v", tc.name, reversed, role, result.Diagnostics())
				value, present := result.Value()
				if tc.outcome != ValueResolved {
					require.False(t, present, "ambiguous aliases must not broadcast a value to multiple members")
					continue
				}
				require.True(t, present)
				for _, field := range value.Fields() {
					scalar, ok := field.Value.Scalar()
					require.True(t, ok)
					require.Equal(t, tc.raw[field.Name], scalar)
				}
			}
		}
		_, err = context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(attribute), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
		require.ErrorContains(t, err, "duplicate emitted member")
		target := DupAtt(attribute)
		fields := AsObject(target.Type)
		*fields = (*fields)[:1]
		plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
		require.NoError(t, err, "a representation selecting one member owns an unambiguous wire name")
		require.Len(t, plan.root.members, 1)
		require.Equal(t, "shared", plan.root.members[0].wire)
	}
}

func TestValueOccurrenceStillRejectsDuplicateAuthoredNames(t *testing.T) {
	_, err := NewValueContext().NewOccurrence(&AttributeExpr{Type: &Object{
		{Name: "same", Attribute: &AttributeExpr{Type: String}},
		{Name: "same", Attribute: &AttributeExpr{Type: String}},
	}})
	require.ErrorContains(t, err, "duplicate member")
}

func TestValueUniqueAliasCrossNamespaceOverlap(t *testing.T) {
	attribute := &AttributeExpr{Type: &Object{
		{Name: "left:shared", Attribute: &AttributeExpr{Type: String}},
		{Name: "shared", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": {"other"}}}},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"shared": "value"}}), ValueRoleExample)
	require.Equal(t, ValueResolved, result.Outcome(), "a unique wire alias overlapping an authored name retains existing semantics")
	value, present := result.Value()
	require.True(t, present)
	for _, field := range value.Fields() {
		scalar, ok := field.Value.Scalar()
		require.True(t, ok)
		require.Equal(t, "value", scalar)
	}
}

func TestValueAliasRuntimeVisibility(t *testing.T) {
	attribute := &AttributeExpr{Type: &Object{
		{Name: "body:shared", Attribute: &AttributeExpr{Type: String}},
		{Name: "header:shared", Attribute: &AttributeExpr{Type: String}},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	target := DupAtt(attribute)
	policies := make([]ValueFieldPolicy, 0, 2)
	for _, field := range *AsObject(target.Type) {
		policies = append(policies, ValueFieldPolicy{Parent: target, Target: field.Attribute, Name: field.Name, WireName: "shared", Visible: true, Presence: ValueFieldRetain})
	}
	request := ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanRuntime, Fields: policies, Containers: []ValueContainerPolicy{{Target: target}}}
	_, err = context.NewValuePlan(occurrence, request)
	require.ErrorContains(t, err, "duplicate emitted member")
	policies[1].Visible = false
	plan, err := context.NewValuePlan(occurrence, request)
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"body:shared": "body", "header:shared": "header"}}), ValueRoleExample)
	projected := context.ProjectJSON(result, plan)
	wire, emitted := projected.JSON()
	require.True(t, emitted, "%v", projected.Diagnostics())
	require.JSONEq(t, `{"shared":"body"}`, string(wire))
}
