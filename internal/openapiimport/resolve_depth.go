package openapiimport

import "strings"

type referenceDepthFrame struct {
	vertex int
	next   int
}

// checkReferenceDepth counts strongly connected groups on a dependency path.
// Members of a recursive group have one depth, regardless of the number of
// simple paths inside it. Pointer ancestors supply containment edges without
// rescanning every reference for every target.
func (r *descriptionResolver) checkReferenceDepth() error {
	indices := make(map[targetKey]int, len(r.contexts))
	for key := range r.contexts {
		indices[key] = len(indices)
	}
	graph := make([][]int, len(indices))
	for _, reference := range r.result.edges {
		if reference.target == nil {
			continue
		}
		to := indices[*reference.target]
		pointer := reference.at.pointer
		for {
			separator := strings.LastIndexByte(pointer, '/')
			if separator < 0 {
				break
			}
			pointer = pointer[:separator]
			if from, exists := indices[targetKey{doc: reference.at.doc, pointer: pointer}]; exists {
				graph[from] = append(graph[from], to)
			}
		}
	}
	if referenceGraphDepth(graph) > maxResolutionDepth {
		return errResolutionBudget
	}
	return nil
}

// referenceGraphDepth runs iterative Kosaraju followed by a topological depth
// pass. Every vertex and edge is processed a constant number of times; neither
// recursion nor simple-path enumeration is used.
func referenceGraphDepth(graph [][]int) int {
	order, reverse := referenceFinishOrder(graph)
	components := make([]int, len(graph))
	for i := range components {
		components[i] = -1
	}
	count := 0
	for i := len(order) - 1; i >= 0; i-- {
		root := order[i]
		if components[root] >= 0 {
			continue
		}
		components[root] = count
		stack := []int{root}
		for len(stack) > 0 {
			vertex := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, neighbor := range reverse[vertex] {
				if components[neighbor] < 0 {
					components[neighbor] = count
					stack = append(stack, neighbor)
				}
			}
		}
		count++
	}
	return referenceComponentDepth(graph, components, count)
}

func referenceFinishOrder(graph [][]int) ([]int, [][]int) {
	reverse := make([][]int, len(graph))
	for from, neighbors := range graph {
		for _, to := range neighbors {
			reverse[to] = append(reverse[to], from)
		}
	}
	seen := make([]bool, len(graph))
	order := make([]int, 0, len(graph))
	for root := range graph {
		if seen[root] {
			continue
		}
		seen[root] = true
		stack := []referenceDepthFrame{{vertex: root}}
		for len(stack) > 0 {
			frame := &stack[len(stack)-1]
			if frame.next == len(graph[frame.vertex]) {
				order = append(order, frame.vertex)
				stack = stack[:len(stack)-1]
				continue
			}
			neighbor := graph[frame.vertex][frame.next]
			frame.next++
			if !seen[neighbor] {
				seen[neighbor] = true
				stack = append(stack, referenceDepthFrame{vertex: neighbor})
			}
		}
	}
	return order, reverse
}

func referenceComponentDepth(graph [][]int, components []int, count int) int {
	outgoing := make([][]int, count)
	incoming := make([]int, count)
	for from, neighbors := range graph {
		for _, to := range neighbors {
			source, target := components[from], components[to]
			if source != target {
				outgoing[source] = append(outgoing[source], target)
				incoming[target]++
			}
		}
	}
	depths := make([]int, count)
	queue := make([]int, 0, count)
	for component, degree := range incoming {
		if degree == 0 {
			queue = append(queue, component)
			depths[component] = 1
		}
	}
	depth := 0
	for i := 0; i < len(queue); i++ {
		component := queue[i]
		depth = max(depth, depths[component])
		for _, next := range outgoing[component] {
			depths[next] = max(depths[next], depths[component]+1)
			incoming[next]--
			if incoming[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	return depth
}
