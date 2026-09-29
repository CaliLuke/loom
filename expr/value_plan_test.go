package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValuePlanMapsOwnedCopies(t *testing.T) {
	context := NewValueContext()
	source := &AttributeExpr{Type: &Object{
		{Name: "value", Attribute: &AttributeExpr{Type: Float32}},
		{Name: "header", Attribute: &AttributeExpr{Type: String}},
	}}
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	target := DupAtt(source)
	object := AsObject(target.Type)
	*object = (*object)[:1]
	(*object)[0].Attribute.Meta = MetaExpr{"struct:tag:json": {"wire"}}
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	require.Len(t, plan.root.members, 1)
	require.Equal(t, occurrence.node.declaration.members[0].id, plan.root.members[0].source)
	require.Equal(t, "wire", plan.root.members[0].wire)
	require.Equal(t, Float32Kind, plan.root.members[0].node.kind)
	(*object)[0].Attribute.Meta["struct:tag:json"][0] = "changed"
	require.Equal(t, "wire", plan.root.members[0].wire)
}

func TestValuePlanRejectsUnrelatedAndAmbiguousTargets(t *testing.T) {
	context := NewValueContext()
	source := &AttributeExpr{Type: String}
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{Target: &AttributeExpr{Type: String}, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.Error(t, err)
	_, err = NewValueContext().NewValuePlan(occurrence, ValuePlanRequest{Target: source, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.Error(t, err)
}

func TestValuePlanSelectedMemberAndRecursiveDeclaration(t *testing.T) {
	context := NewValueContext()
	list := &UserTypeExpr{TypeName: "List", AttributeExpr: &AttributeExpr{}}
	list.Type = &Object{
		{Name: "value", Attribute: &AttributeExpr{Type: Float64}},
		{Name: "next", Attribute: &AttributeExpr{Type: list, Nullable: true}},
	}
	source := &AttributeExpr{Type: list}
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	require.NotNil(t, plan.root.alias.members[1].node.alias)
	member := AsObject(list.Type).Attribute("value")
	selected, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(member), Selection: []string{"value"}, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	require.Equal(t, Float64Kind, selected.root.kind)
	require.Len(t, selected.selection, 1)
}

func TestValuePlanRuntimePolicies(t *testing.T) {
	context := NewValueContext()
	source := &AttributeExpr{Type: &Object{{Name: "number", Attribute: &AttributeExpr{Type: Float32}}}}
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	target := DupAtt(source)
	member := AsObject(target.Type).Attribute("number")
	field := ValueFieldPolicy{Parent: target, Target: member, Name: "number", WireName: "n", Visible: true, Required: true, Presence: ValueFieldRetain, NumericKind: Float32Kind}
	container := ValueContainerPolicy{Target: target, RejectUnknownMembers: true, PreserveAdditional: false}
	for _, test := range []struct {
		name       string
		fields     []ValueFieldPolicy
		containers []ValueContainerPolicy
		valid      bool
	}{
		{name: "missing all policies"},
		{name: "missing container", fields: []ValueFieldPolicy{field}},
		{name: "missing field", containers: []ValueContainerPolicy{container}},
		{name: "duplicate field", fields: []ValueFieldPolicy{field, field}, containers: []ValueContainerPolicy{container}},
		{name: "duplicate container", fields: []ValueFieldPolicy{field}, containers: []ValueContainerPolicy{container, container}},
		{name: "complete", fields: []ValueFieldPolicy{field}, containers: []ValueContainerPolicy{container}, valid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanRuntime, Fields: test.fields, Containers: test.containers})
			if !test.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.True(t, plan.root.schemaUnknown)
			require.False(t, plan.root.runtimeUnknown)
			require.False(t, plan.root.preserveAdditional)
			require.Equal(t, Float32Kind, plan.root.members[0].node.kind)
			require.True(t, plan.root.members[0].required)
		})
	}
	field.NumericKind = Float64Kind
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanRuntime, Fields: []ValueFieldPolicy{field}, Containers: []ValueContainerPolicy{container}})
	require.ErrorContains(t, err, "precision")
}

func TestValuePlanRejectsAmbiguousLineage(t *testing.T) {
	shared := &AttributeExpr{Type: String}
	source := &AttributeExpr{Type: &Object{{Name: "a", Attribute: shared}, {Name: "b", Attribute: shared}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.ErrorContains(t, err, "ambiguous")
}
