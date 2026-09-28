package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnionBranchCyclesMatchReachability(t *testing.T) {
	// Exhaust all directed graphs on three nodes, including shared DAG tails,
	// self edges, mutual cycles, and cycles unreachable from the selected root.
	for edges := range 1 << 9 {
		t.Run(fmt.Sprint(edges), func(t *testing.T) {
			unions := []*Union{{TypeName: "A"}, {TypeName: "B"}, {TypeName: "C"}}
			var reach [3][3]bool
			for from := range 3 {
				for to := range 3 {
					if edges&(1<<(from*3+to)) != 0 {
						reach[from][to] = true
						unions[from].Values = append(unions[from].Values, &NamedAttributeExpr{
							Name: unions[to].Name(), Attribute: &AttributeExpr{Type: unions[to]},
						})
					}
				}
			}
			for via := range 3 {
				for from := range 3 {
					for to := range 3 {
						reach[from][to] = reach[from][to] || reach[from][via] && reach[via][to]
					}
				}
			}
			for root := range 3 {
				cycle := false
				for node := range 3 {
					cycle = cycle || (node == root || reach[root][node]) && reach[node][node]
				}
				require.Equal(t, cycle, hasInlineUnionCycle(unions[root]), "root %d", root)
			}
		})
	}
}

func TestUnionBranchCycleStopsAtObjectFields(t *testing.T) {
	union := &Union{TypeName: "Tree"}
	branch := &UserTypeExpr{TypeName: "Node", AttributeExpr: &AttributeExpr{
		Type: &Object{{Name: "next", Attribute: &AttributeExpr{Type: union}}},
	}}
	union.Values = []*NamedAttributeExpr{{Name: "node", Attribute: &AttributeExpr{Type: branch}}}
	require.False(t, hasInlineUnionCycle(union))
}
