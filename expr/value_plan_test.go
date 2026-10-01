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
	selectedMember := occurrence.Members()[0].Occurrence
	memberPlan, err := selected.ForOccurrence(selectedMember, selected.Root())
	require.NoError(t, err)
	require.Same(t, selected.root, memberPlan.root)
	require.Empty(t, memberPlan.selection)
	require.NotSame(t, selected.association, memberPlan.association)
	rootPlan, err := selected.ForOccurrence(occurrence, selected.Root())
	require.NoError(t, err)
	require.Equal(t, selected.selection, rootPlan.selection)
	require.Same(t, selected.association, rootPlan.association)
	repeated, err := selected.ForOccurrence(selectedMember, selected.Root())
	require.NoError(t, err)
	require.Same(t, memberPlan.association, repeated.association)
}

func TestValuePlanForOccurrenceRequiresExactSourceAndTarget(t *testing.T) {
	child := &AttributeExpr{Type: String}
	source := &AttributeExpr{Type: &Object{{Name: "value", Attribute: child}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.NoError(t, err)
	foreign, err := NewValueContext().NewOccurrence(source)
	require.NoError(t, err)
	_, err = plan.ForOccurrence(foreign, plan.Root())
	require.ErrorContains(t, err, "does not belong")

	missing := occurrence
	missing.node = &valueOccurrenceNode{}
	_, err = plan.ForOccurrence(missing, plan.Root())
	require.ErrorIs(t, err, ErrValuePlanAssociationNotFound)
	_, err = plan.ForOccurrence(occurrence, ValuePlanNode{})
	require.ErrorContains(t, err, "target node is absent")

	member := occurrence.Members()[0].Occurrence
	association := plan.associations[member.node][0]
	secondTarget := &valuePlanAssociation{
		id: association.id + 100, source: member.node, root: plan.root,
	}
	plan.associations[member.node] = append(plan.associations[member.node], secondTarget)
	memberPlan, err := plan.ForOccurrence(member, ValuePlanNode{association.root})
	require.NoError(t, err)
	require.Same(t, association, memberPlan.association)
	rootTarget, err := plan.ForOccurrence(member, plan.Root())
	require.NoError(t, err)
	require.Same(t, secondTarget, rootTarget.association,
		"the exact target disambiguates representations sharing one source")
	_, err = plan.ForOccurrence(occurrence, ValuePlanNode{association.root})
	require.ErrorIs(t, err, ErrValuePlanAssociationNotFound, "a target owned by another source must fail")

	other, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.NoError(t, err)
	_, err = plan.ForOccurrence(occurrence, other.Root())
	require.ErrorContains(t, err, "does not belong", "a node from another plan must fail")
	require.NotErrorIs(t, err, ErrValuePlanAssociationNotFound)

	plan.associations[member.node] = append(plan.associations[member.node], &valuePlanAssociation{
		id: association.id + 200, source: member.node, root: association.root,
	})
	_, err = plan.ForOccurrence(member, ValuePlanNode{association.root})
	require.ErrorContains(t, err, "ambiguous")
}

func TestValuePlanForOccurrencePreservesAliasOwners(t *testing.T) {
	minimum := 2
	base := &UserTypeExpr{TypeName: "Base", AttributeExpr: &AttributeExpr{Type: String,
		UserExamples: []*ExampleExpr{{Value: "x"}}}}
	derived := &UserTypeExpr{TypeName: "Derived", AttributeExpr: &AttributeExpr{Type: base,
		Validation: &ValidationExpr{MinLength: &minimum}}}
	attribute := &AttributeExpr{Type: derived}
	context := NewValueContext()
	outer, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	derivedOccurrence := outer.Underlying()
	baseOccurrence := derivedOccurrence.Underlying()
	plan, err := context.NewValuePlan(outer, ValuePlanRequest{
		Target: DupAtt(attribute), Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.NoError(t, err)
	outerPlan, err := plan.ForOccurrence(outer, plan.Root())
	require.NoError(t, err)
	derivedPlan, err := plan.ForOccurrence(derivedOccurrence, plan.Root().Underlying())
	require.NoError(t, err)
	basePlan, err := plan.ForOccurrence(baseOccurrence, plan.Root().Underlying().Underlying())
	require.NoError(t, err)
	require.Same(t, plan.root, outerPlan.root)
	require.Same(t, plan.root.alias, derivedPlan.root)
	require.Same(t, plan.root.alias.alias, basePlan.root)
	require.NotSame(t, outerPlan.association, derivedPlan.association)
	require.NotSame(t, derivedPlan.association, basePlan.association)

	selection := context.SelectExample(outer, ExamplePolicy{Reachable: true})
	source, present := selection.Source()
	require.True(t, present)
	require.Equal(t, ValueInvalid, context.Resolve(outer, source, ValueRoleExample).Outcome())
	require.Equal(t, ValueInvalid, context.Resolve(derivedOccurrence, source, ValueRoleExample).Outcome())
	require.Equal(t, ValueResolved, context.Resolve(baseOccurrence, source, ValueRoleExample).Outcome())
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
