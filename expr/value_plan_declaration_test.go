package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValuePlanCapturesDeclarationAncestry(t *testing.T) {
	named := &UserTypeExpr{TypeName: "Original", AttributeExpr: &AttributeExpr{}}
	named.Type = &Object{{Name: "next", Attribute: &AttributeExpr{Type: named}}, {Name: "data", Attribute: &AttributeExpr{Type: Bytes}}}
	original := &AttributeExpr{Type: named}
	copied := cloneExplicitHTTPBody(original, "TransportBody", "Body", "service#TransportBody")
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(copied)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: copied, Codec: ValueCodecJSON, Use: ValuePlanSchema})
	require.NoError(t, err)
	require.Equal(t, "Original", plan.Root().TargetDeclarationID())
	require.Equal(t, "service#TransportBody", plan.Root().Attribute().Type.(UserType).ID())
	named.TypeName = "LaterMutation"
	require.Equal(t, "Original", plan.Root().TargetDeclarationID(), "capture owns identity instead of querying mutable ancestry")
	independent := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Other", AttributeExpr: &AttributeExpr{Type: Bytes}}}
	other, err := context.NewOccurrence(independent)
	require.NoError(t, err)
	otherPlan, err := context.NewValuePlan(other, ValuePlanRequest{Target: independent, Codec: ValueCodecJSON, Use: ValuePlanSchema})
	require.NoError(t, err)
	require.Equal(t, "Other", otherPlan.Root().TargetDeclarationID())
}

func TestValuePlanDeclarationThroughStreamingWrapper(t *testing.T) {
	for _, test := range []struct {
		aliases    int
		identities []string
	}{
		{0, []string{"Base", "Base", ""}},
		{1, []string{"Alias0", "Alias0", "Base", ""}},
		{2, []string{"Alias1", "Alias1", "Alias0", "Base", ""}},
	} {
		t.Run(fmt.Sprint(test.aliases), func(t *testing.T) {
			var typ DataType = &UserTypeExpr{TypeName: "Base", AttributeExpr: &AttributeExpr{Type: &Object{{Name: "data", Attribute: &AttributeExpr{Type: Bytes}}}}}
			for index := range test.aliases {
				typ = &UserTypeExpr{TypeName: fmt.Sprintf("Alias%d", index), AttributeExpr: &AttributeExpr{Type: typ}}
			}
			source := &AttributeExpr{Type: typ}
			target := httpStreamingBody(&HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: "send", Stream: ClientStreamKind, StreamingPayload: source}, Service: &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "transport"}}})
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(source)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanSchema})
			require.NoError(t, err)
			node := plan.Root()
			for _, expected := range test.identities {
				require.True(t, node.Valid())
				require.Equal(t, expected, node.TargetDeclarationID(), "target authority follows the authored declaration sequence, not the independently advancing source cursor")
				node = node.Underlying()
			}
			require.False(t, node.Valid())
		})
	}
}

func TestValuePlanDeclarationAuthorityPositions(t *testing.T) {
	for _, shape := range []string{"field", "array", "map", "tagged", "untagged"} {
		t.Run(shape, func(t *testing.T) {
			first := &UserTypeExpr{TypeName: "First", AttributeExpr: &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}}}
			second := &UserTypeExpr{TypeName: "Second", AttributeExpr: &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}}}
			root := &UserTypeExpr{TypeName: "Root", AttributeExpr: &AttributeExpr{}}
			root.Type = &Object{
				{Name: "first", Attribute: declarationPosition(shape, first)},
				{Name: "second", Attribute: declarationPosition(shape, second)},
				{Name: "again", Attribute: declarationPosition(shape, first)},
				{Name: "next", Attribute: &AttributeExpr{Type: root}},
			}
			source := &AttributeExpr{Type: root}
			target := httpStreamingBody(&HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: "watch", Stream: ClientStreamKind, StreamingPayload: source}, Service: &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "transport"}}})
			var previous ValuePlanNode
			for range 2 {
				context := NewValueContext()
				occurrence, err := context.NewOccurrence(source)
				require.NoError(t, err)
				for _, codec := range []ValueCodec{ValueCodecJSON, ValueCodecText} {
					plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: codec, Use: ValuePlanSchema})
					require.NoError(t, err)
					require.NotEqual(t, previous, plan.Root(), "each captured plan owns its nodes")
					previous = plan.Root()
					body := plan.Root().Underlying().Underlying()
					fields := body.Members()
					require.Len(t, fields, 4)
					for index, expected := range []string{"First", "Second", "First"} {
						node := fields[index].Node
						switch shape {
						case "array", "map":
							node = node.Element()
						case "tagged", "untagged":
							node = node.Branches()[0].Node
						}
						require.Equal(t, expected, node.TargetDeclarationID(), "distinct authored equal-shaped declarations must not inherit their enclosing root's authority")
						require.Equal(t, codec, node.Codec())
						copied := node.Attribute()
						copied.Type.(UserType).Rename("CallerMutation")
						require.Equal(t, expected, node.TargetDeclarationID())
					}
					require.Equal(t, "Root", fields[3].Node.TargetDeclarationID(), "recursive reuse retains declaration authority")
				}
			}
		})
	}
}

func declarationPosition(shape string, typ UserType) *AttributeExpr {
	child := &AttributeExpr{Type: typ}
	switch shape {
	case "array":
		return &AttributeExpr{Type: &Array{ElemType: child}}
	case "map":
		return &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: child}}
	case "tagged", "untagged":
		return &AttributeExpr{Type: &Union{TypeName: "Choice", Untagged: shape == "untagged", Values: []*NamedAttributeExpr{{Name: "value", Attribute: child}}}}
	default:
		return child
	}
}

func TestValuePlanStreamingSourceSequence(t *testing.T) {
	for _, aliases := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(aliases), func(t *testing.T) {
			base := &UserTypeExpr{TypeName: "SequenceBase", AttributeExpr: &AttributeExpr{Type: &Object{{Name: "count", Attribute: &AttributeExpr{Type: Int}}}}}
			var typ DataType = base
			origins := []*AttributeExpr{base.AttributeExpr}
			for index := range aliases {
				alias := &UserTypeExpr{TypeName: fmt.Sprintf("SequenceAlias%d", index), AttributeExpr: &AttributeExpr{Type: typ}}
				origins = append([]*AttributeExpr{alias.AttributeExpr}, origins...)
				typ = alias
			}
			source := &AttributeExpr{Type: typ}
			expected := append([]*AttributeExpr{source, source}, origins...)
			target := httpStreamingBody(&HTTPEndpointExpr{MethodExpr: &MethodExpr{Name: "send", Stream: ClientStreamKind, StreamingPayload: source}, Service: &HTTPServiceExpr{ServiceExpr: &ServiceExpr{Name: "transport"}}})
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(source)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanSchema})
			require.NoError(t, err)
			node := plan.Root()
			for index, origin := range expected {
				require.True(t, node.Valid())
				require.Same(t, origin, node.node.source.origin, "source step %d must follow original authored edge, not each target wrapper", index)
				require.Equal(t, index == 0, node.UnderlyingReusesSource(), "only the inserted outer edge reuses its source")
				node = node.Underlying()
			}
			require.False(t, node.Valid())
		})
	}
}

func TestValuePlanControlledStructuralUnwrap(t *testing.T) {
	named := &UserTypeExpr{TypeName: "InlineBody", AttributeExpr: &AttributeExpr{Type: &Object{{Name: "count", Attribute: &AttributeExpr{Type: Int}}}, Meta: MetaExpr{inlineHTTPBodyMetaKey: {}}}}
	source := &AttributeExpr{Type: named}
	target := UnwrapInlineHTTPBody(DupAtt(source))
	require.IsType(t, &Object{}, target.Type)
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	require.Same(t, named.AttributeExpr, plan.root.source.origin)
	require.Equal(t, occurrence.node.declaration.alias.declaration.members[0].id, plan.root.members[0].source)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"count": 3}}), ValueRoleExample)
	projected := context.ProjectJSON(result, plan)
	require.Equal(t, ProjectionEmitted, projected.Outcome())
}

func TestValuePlanAliasEdgeRolesDoNotBelongToSharedChild(t *testing.T) {
	leaf := &AttributeExpr{Type: &Object{}}
	outer := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Outer", AttributeExpr: leaf}}
	sourceLeaf := &valueOccurrenceNode{origin: leaf, attribute: leaf, declaration: &valueDeclarationNode{kind: ObjectKind}}
	sourceOuter := &valueOccurrenceNode{origin: outer, attribute: outer, declaration: &valueDeclarationNode{alias: sourceLeaf}}
	targetLeaf := &valueOccurrenceNode{origin: leaf, attribute: leaf, declaration: &valueDeclarationNode{kind: ObjectKind}}
	authored := &valueOccurrenceNode{origin: outer, attribute: outer, declaration: &valueDeclarationNode{alias: targetLeaf}}
	wrapper := &valueOccurrenceNode{origin: leaf, attribute: leaf, declaration: &valueDeclarationNode{alias: targetLeaf}}
	for _, reverse := range []bool{false, true} {
		t.Run(fmt.Sprint(reverse), func(t *testing.T) {
			builder := valuePlanBuilder{context: NewValueContext(), request: ValuePlanRequest{Use: ValuePlanSchema}, nodes: make(map[valuePlanKey]*valuePlanNode)}
			inputs := []struct {
				source, target *valueOccurrenceNode
				reuses         bool
			}{{sourceOuter, authored, false}, {sourceLeaf, wrapper, true}}
			if reverse {
				inputs[0], inputs[1] = inputs[1], inputs[0]
			}
			var shared *valuePlanNode
			for _, input := range inputs {
				parent, err := builder.node(input.source, input.target, ValueCodecJSON)
				require.NoError(t, err)
				require.Equal(t, input.reuses, (ValuePlanNode{parent}).UnderlyingReusesSource())
				if shared != nil {
					require.Same(t, shared, parent.alias)
				}
				shared = parent.alias
			}
		})
	}
}

func TestValuePlanAliasPairingRejectsForeignAndCyclicAncestry(t *testing.T) {
	source := &AttributeExpr{Type: &UserTypeExpr{TypeName: "Original", AttributeExpr: &AttributeExpr{Type: String}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	target := DupAtt(source)
	target.Type = &UserTypeExpr{TypeName: "Unrelated", AttributeExpr: &AttributeExpr{Type: &UserTypeExpr{TypeName: "Foreign", AttributeExpr: &AttributeExpr{Type: String}}}}
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanSchema})
	require.ErrorContains(t, err, "alias has no matching source ancestry")
	cyclic := &valueOccurrenceNode{origin: source, declaration: &valueDeclarationNode{}}
	cyclic.declaration.alias = cyclic
	_, _, err = valuePlanAliasSource(cyclic, &valueOccurrenceNode{origin: &AttributeExpr{Type: String}, declaration: &valueDeclarationNode{alias: cyclic}})
	require.ErrorContains(t, err, "no matching source ancestry")
	_, err = valuePlanStructuralSource(cyclic)
	require.ErrorContains(t, err, "cyclic alias chain")
}
