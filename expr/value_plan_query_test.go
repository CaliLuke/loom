package expr

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValuePlanOccurrenceForTarget(t *testing.T) {
	plan, root := wideMembershipPlan(t, 3)
	members, targets := root.Members(), plan.Root().Members()
	for index, member := range members {
		found, err := plan.OccurrenceForTarget(root, targets[index].Node)
		require.NoError(t, err)
		require.Equal(t, member.Occurrence.ID(), found.ID())
		projection, err := plan.ForOccurrence(found, targets[index].Node)
		require.NoError(t, err)
		require.Equal(t, targets[index].Node, projection.Root())
	}
	other, foreign := wideMembershipPlan(t, 1)
	sameContext := foreign
	sameContext.context = root.context
	child, err := plan.ForOccurrence(members[0].Occurrence, targets[0].Node)
	require.NoError(t, err)
	for _, test := range []struct {
		name    string
		plan    ValuePlan
		root    ValueOccurrence
		target  ValuePlanNode
		missing bool
	}{
		{"zero plan", ValuePlan{}, root, targets[0].Node, false},
		{"zero root", plan, ValueOccurrence{}, targets[0].Node, false},
		{"foreign root", plan, foreign, targets[0].Node, false},
		{"foreign capture", plan, sameContext, targets[0].Node, false},
		{"absent target", plan, root, ValuePlanNode{}, false},
		{"foreign target", plan, root, other.Root(), false},
		{"subplan sibling", child, root, targets[1].Node, false},
		{"subplan ancestor", child, root, plan.Root(), false},
		{"unreachable owner", plan, members[1].Occurrence, targets[0].Node, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.plan.OccurrenceForTarget(test.root, test.target)
			require.Error(t, err)
			if test.missing {
				require.ErrorIs(t, err, ErrValuePlanAssociationNotFound)
			} else {
				require.NotErrorIs(t, err, ErrValuePlanAssociationNotFound)
			}
		})
	}
}

func TestValuePlanOccurrenceForTargetSelectionAndAmbiguity(t *testing.T) {
	attribute := &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}}
	context := NewValueContext()
	root, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(root, ValuePlanRequest{Target: DupAtt(AsObject(attribute.Type).Attribute("value")), Selection: []string{"value"}, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	_, err = plan.OccurrenceForTarget(root, plan.Root())
	require.ErrorContains(t, err, "ambiguous")
	member := root.Members()[0].Occurrence
	found, err := plan.OccurrenceForTarget(member, plan.Root())
	require.NoError(t, err)
	require.Equal(t, member.ID(), found.ID())
}

func TestValuePlanOccurrenceForTargetConcurrentPublication(t *testing.T) {
	plan, root := wideMembershipPlan(t, 64)
	targets, members := plan.Root().Members(), root.Members()
	failures := make(chan error, len(targets))
	var group sync.WaitGroup
	for index := range targets {
		group.Go(func() {
			found, err := plan.OccurrenceForTarget(root, targets[index].Node)
			if err != nil {
				failures <- err
				return
			}
			if found.ID() != members[index].Occurrence.ID() {
				failures <- fmt.Errorf("wrong owner for %d", index)
			}
		})
	}
	group.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
}

func TestValuePlanOccurrenceForTargetAllocations(t *testing.T) {
	plan, root := wideMembershipPlan(t, 256)
	target := plan.Root().Members()[255].Node
	allocations := testing.AllocsPerRun(20, func() {
		_, err := plan.OccurrenceForTarget(root, target)
		require.NoError(t, err)
	})
	require.Less(t, allocations, float64(10), "an exact owner query must not allocate a traversal of every sibling")
}

func TestValuePlanNodeMetadataQueriesAreCaptured(t *testing.T) {
	attribute := &AttributeExpr{Type: String, Meta: MetaExpr{"openapi:example": {"true", "false"}}}
	context := NewValueContext()
	root, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(root, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	attribute.Meta["openapi:example"][1] = "changed"
	value, present := plan.Root().MetaLast("openapi:example")
	require.True(t, present)
	require.Equal(t, "false", value)
	value, present = plan.Root().MetaLast("missing")
	require.False(t, present)
	require.Empty(t, value)
	_, present = (ValuePlanNode{}).MetaLast("missing")
	require.False(t, present)
	require.False(t, (ValuePlanNode{}).IsInlineHTTPBody())
	require.False(t, plan.Root().IsInlineHTTPBody())
}

func BenchmarkValuePlanOccurrenceForTarget(b *testing.B) {
	for _, size := range []int{16, 256, 1024} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			plan, root := wideMembershipPlan(b, size)
			target := plan.Root().Members()[size-1].Node
			_, err := plan.OccurrenceForTarget(root, target)
			require.NoError(b, err)
			b.ReportAllocs()
			for b.Loop() {
				_, err := plan.OccurrenceForTarget(root, target)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestValuePlanOccurrenceForTargetMatchesExhaustiveRecursiveSearch(t *testing.T) {
	list := &UserTypeExpr{TypeName: "List", AttributeExpr: &AttributeExpr{}}
	list.Type = &Object{
		{Name: "value", Attribute: &AttributeExpr{Type: String}},
		{Name: "next", Attribute: &AttributeExpr{Type: list, Nullable: true}},
		{Name: "array", Attribute: &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: list}}}},
		{Name: "choice", Attribute: &AttributeExpr{Type: &Union{TypeName: "Choice", Values: []*NamedAttributeExpr{{Name: "text", Attribute: &AttributeExpr{Type: String}}, {Name: "number", Attribute: &AttributeExpr{Type: Int}}}}}},
		{Name: "map", Attribute: &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: list}}}},
	}
	attribute := &AttributeExpr{Type: list}
	context := NewValueContext()
	root, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(root, ValuePlanRequest{Target: DupAtt(attribute), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	for _, source := range root.graph.nodes {
		region := ValueOccurrence{context: root.context, graph: root.graph, node: source}
		for target := range plan.nodes {
			expected, expectedErr := exhaustiveTargetOwner(plan, region, ValuePlanNode{target})
			actual, actualErr := plan.OccurrenceForTarget(region, ValuePlanNode{target})
			if expectedErr != nil {
				require.Error(t, actualErr)
				require.Equal(t, errors.Is(expectedErr, ErrValuePlanAssociationNotFound), errors.Is(actualErr, ErrValuePlanAssociationNotFound))
			} else {
				require.NoError(t, actualErr)
				require.Equal(t, expected.ID(), actual.ID())
			}
		}
	}
}

func TestValuePlanOccurrenceForTargetPairPrecedence(t *testing.T) {
	for _, preferred := range []bool{false, true} {
		t.Run(fmt.Sprint(preferred), func(t *testing.T) {
			plan, root := wideMembershipPlan(t, 1)
			member, target := root.Members()[0].Occurrence, plan.Root().Members()[0].Node
			if preferred {
				var err error
				plan, err = plan.ForOccurrence(member, target)
				require.NoError(t, err)
			}
			association := *plan.associations[member.node][0]
			association.id += 100
			plan.associations[member.node] = append(plan.associations[member.node], &association)
			found, err := plan.OccurrenceForTarget(member, target)
			if preferred {
				require.NoError(t, err)
				require.Equal(t, member.ID(), found.ID())
			} else {
				require.ErrorContains(t, err, "ambiguous")
			}
		})
	}
}

func TestValuePlanOccurrenceForTargetExcludesConstraintParents(t *testing.T) {
	plan, root := wideMembershipPlan(t, 2)
	members, targets := root.Members(), plan.Root().Members()
	// Constraint-parent links belong to validation provenance, not structural
	// traversal. Both nodes already belong to the same capture.
	members[0].Occurrence.node.constraintParents = []*valueOccurrenceNode{members[1].Occurrence.node}
	_, err := plan.OccurrenceForTarget(members[0].Occurrence, targets[1].Node)
	require.ErrorIs(t, err, ErrValuePlanAssociationNotFound)
}

func TestValuePlanNodeInlineBodyQueryIsCaptured(t *testing.T) {
	body := &UserTypeExpr{TypeName: "InlineBody", AttributeExpr: &AttributeExpr{
		Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}},
		Meta: MetaExpr{inlineHTTPBodyMetaKey: {"true"}},
	}}
	attribute := &AttributeExpr{Type: body}
	context := NewValueContext()
	root, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(root, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	delete(body.Meta, inlineHTTPBodyMetaKey)
	require.True(t, plan.Root().IsInlineHTTPBody())
	require.False(t, plan.Root().Underlying().IsInlineHTTPBody())
}

func exhaustiveTargetOwner(plan ValuePlan, root ValueOccurrence, target ValuePlanNode) (ValueOccurrence, error) {
	visited := make(map[ValueIdentity]bool)
	queue := []ValueOccurrence{root}
	var found ValueOccurrence
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.ID() == (ValueIdentity{}) || visited[current.ID()] {
			continue
		}
		visited[current.ID()] = true
		_, err := plan.ForOccurrence(current, target)
		if err == nil {
			if found.ID() != (ValueIdentity{}) {
				return ValueOccurrence{}, fmt.Errorf("ambiguous")
			}
			found = current
		} else if !errors.Is(err, ErrValuePlanAssociationNotFound) {
			return ValueOccurrence{}, err
		}
		queue = append(queue, current.Underlying(), current.Element(), current.Key())
		for _, member := range current.Members() {
			queue = append(queue, member.Occurrence)
		}
		for _, branch := range current.Branches() {
			queue = append(queue, branch.Occurrence)
		}
	}
	if found.ID() == (ValueIdentity{}) {
		return ValueOccurrence{}, ErrValuePlanAssociationNotFound
	}
	return found, nil
}

func BenchmarkValuePlanExhaustiveOwnerSearch(b *testing.B) {
	for _, size := range []int{16, 256, 1024} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			plan, root := wideMembershipPlan(b, size)
			target := plan.Root().Members()[size-1].Node
			b.ReportAllocs()
			for b.Loop() {
				_, err := exhaustiveTargetOwner(plan, root, target)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
