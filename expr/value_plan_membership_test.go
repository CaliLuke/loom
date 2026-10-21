package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValuePlanAssociationLookupAllocations(t *testing.T) {
	plan, occurrence := wideMembershipPlan(t, 256)
	members := occurrence.Members()
	target := plan.Root().Members()[255].Node
	allocations := testing.AllocsPerRun(20, func() {
		_, err := plan.ForOccurrence(members[0].Occurrence, target)
		require.ErrorIs(t, err, ErrValuePlanAssociationNotFound)
	})
	require.Less(t, allocations, float64(10), "a missing association must not allocate a traversal of the target graph")
}

func TestValuePlanAssociationCannotEscapeSubtree(t *testing.T) {
	plan, occurrence := wideMembershipPlan(t, 2)
	members := occurrence.Members()
	targets := plan.Root().Members()
	child, err := plan.ForOccurrence(members[0].Occurrence, targets[0].Node)
	require.NoError(t, err)
	_, err = child.ForOccurrence(members[1].Occurrence, targets[1].Node)
	require.ErrorContains(t, err, "target node does not belong")
	require.NotErrorIs(t, err, ErrValuePlanAssociationNotFound)
	_, err = child.ForOccurrence(occurrence, plan.Root())
	require.ErrorContains(t, err, "target node does not belong")
	_, err = child.ForOccurrence(members[0].Occurrence, targets[0].Node)
	require.NoError(t, err)
}

func BenchmarkValuePlanAssociationLookup(b *testing.B) {
	for _, size := range []int{16, 256, 1024} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			plan, occurrence := wideMembershipPlan(b, size)
			source := occurrence.Members()[0].Occurrence
			target := plan.Root().Members()[size-1].Node
			b.ReportAllocs()
			for b.Loop() {
				_, err := plan.ForOccurrence(source, target)
				if err == nil {
					b.Fatal("expected missing association")
				}
			}
		})
	}
}

func wideMembershipPlan(t testing.TB, size int) (ValuePlan, ValueOccurrence) {
	t.Helper()
	object := make(Object, size)
	for i := range object {
		object[i] = &NamedAttributeExpr{Name: fmt.Sprintf("field%d", i), Attribute: &AttributeExpr{Type: String}}
	}
	attribute := &AttributeExpr{Type: &object}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: DupAtt(attribute), Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.NoError(t, err)
	return plan, occurrence
}
