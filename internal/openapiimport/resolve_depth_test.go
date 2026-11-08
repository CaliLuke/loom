package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolverComponentDepth(t *testing.T) {
	chain := func(n int) [][]int {
		graph := make([][]int, n)
		for i := range n - 1 {
			graph[i] = []int{i + 1}
		}
		return graph
	}
	ring := chain(65)
	ring[64] = []int{0}
	groups := make([][]int, 128)
	for i := range 64 {
		groups[2*i] = []int{2*i + 1}
		groups[2*i+1] = []int{2 * i}
		if i < 63 {
			groups[2*i+1] = append(groups[2*i+1], 2*i+2)
		}
	}
	dense := make([][]int, 65)
	for i := range 12 {
		for j := range 12 {
			if i != j {
				dense[i] = append(dense[i], j)
			}
		}
	}
	for i := 12; i < len(dense); i++ {
		dense[0] = append(dense[0], i)
		dense[i] = []int{0}
	}
	for _, tc := range []struct {
		name  string
		graph [][]int
		fail  bool
	}{
		{"empty", nil, false},
		{"64 groups", chain(64), false},
		{"65 groups", chain(65), true},
		{"65 node recursive group", ring, false},
		{"64 recursive groups", groups, false},
		{"dense recursive group", dense, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolver := depthTestResolver(tc.graph)
			if tc.fail {
				require.ErrorIs(t, resolver.checkReferenceDepth(), errResolutionBudget)
			} else {
				require.NoError(t, resolver.checkReferenceDepth())
			}
		})
	}
}

func depthTestResolver(graph [][]int) *descriptionResolver {
	resolver := &descriptionResolver{
		result: &resolution{}, contexts: make(map[targetKey]map[string]bool),
	}
	key := func(vertex int) targetKey {
		return targetKey{doc: "models", pointer: fmt.Sprintf("/N%d", vertex)}
	}
	for i, neighbors := range graph {
		resolver.contexts[key(i)] = map[string]bool{"schemas": true}
		for j, neighbor := range neighbors {
			target := key(neighbor)
			resolver.result.edges = append(resolver.result.edges, &edge{
				at: targetKey{doc: "models", pointer: fmt.Sprintf("/N%d/properties/p%d/$ref", i, j)}, target: &target,
			})
		}
	}
	return resolver
}

func TestReferenceGraphDepthExhaustive(t *testing.T) {
	const n = 3
	for mask := range 1 << (n * n) {
		graph := make([][]int, n)
		for from := range n {
			for to := range n {
				if mask&(1<<(from*n+to)) != 0 {
					graph[from] = append(graph[from], to)
				}
			}
		}
		want := referenceDepthOracle(graph)
		require.Equal(t, want, referenceGraphDepth(graph), "graph %v", graph)
		reversed := make([][]int, n)
		for from, neighbors := range graph {
			for _, to := range neighbors {
				reversed[n-1-from] = append(reversed[n-1-from], n-1-to, n-1-to)
			}
		}
		require.Equal(t, want, referenceGraphDepth(reversed), "renumbered graph with duplicate edges %v", graph)
	}
}

// referenceDepthOracle uses transitive closure and bounded simple-path
// enumeration, independently of the production SCC and topological passes.
func referenceDepthOracle(graph [][]int) int {
	n := len(graph)
	reach := make([][]bool, n)
	for from, neighbors := range graph {
		reach[from] = make([]bool, n)
		reach[from][from] = true
		for _, to := range neighbors {
			reach[from][to] = true
		}
	}
	for via := range n {
		for from := range n {
			for to := range n {
				reach[from][to] = reach[from][to] || reach[from][via] && reach[via][to]
			}
		}
	}
	groups := make([]int, n)
	for from := range n {
		groups[from] = from
		for to := range from {
			if reach[from][to] && reach[to][from] {
				groups[from] = groups[to]
				break
			}
		}
	}
	best := 0
	var visit func(int, map[int]bool, map[int]int)
	visit = func(vertex int, seen map[int]bool, counts map[int]int) {
		if seen[vertex] {
			return
		}
		seen[vertex] = true
		counts[groups[vertex]]++
		best = max(best, len(counts))
		for _, next := range graph[vertex] {
			visit(next, seen, counts)
		}
		delete(seen, vertex)
		counts[groups[vertex]]--
		if counts[groups[vertex]] == 0 {
			delete(counts, groups[vertex])
		}
	}
	for vertex := range graph {
		visit(vertex, make(map[int]bool), make(map[int]int))
	}
	return best
}

func TestReferenceDepthContainment(t *testing.T) {
	graph := make([][]int, 65)
	for i := range 64 {
		graph[i] = []int{i + 1}
	}
	for _, tc := range []struct {
		name string
		at   targetKey
	}{
		{"other document", targetKey{doc: "other", pointer: "/N0/$ref"}},
		{"prefix is not ancestor", targetKey{doc: "models", pointer: "/N0suffix/$ref"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolver := depthTestResolver(graph)
			resolver.result.edges[0].at = tc.at
			require.NoError(t, resolver.checkReferenceDepth(), "unrelated reference must break the 65-group chain")
		})
	}
	short := make([][]int, 64)
	copy(short, graph[:63])
	resolver := depthTestResolver(short)
	resolver.contexts[targetKey{doc: "models", pointer: ""}] = map[string]bool{"schemas": true}
	require.NoError(t, resolver.checkReferenceDepth(), "depth counts a path, not total vertices")
	target := targetKey{doc: "models", pointer: "/N0"}
	resolver.result.edges = append(resolver.result.edges, &edge{
		at: targetKey{doc: "models", pointer: "/entry/$ref"}, target: &target,
	})
	require.ErrorIs(t, resolver.checkReferenceDepth(), errResolutionBudget, "document-root target owns its descendant references")
}
