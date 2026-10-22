package expr

import (
	"fmt"
	"sync"
)

type (
	valuePlanQueryIndex struct {
		once    sync.Once
		sources map[*valuePlanNode][]*valueOccurrenceNode
	}
	valueOccurrenceParentIndex struct {
		once    sync.Once
		parents map[*valueOccurrenceNode][]*valueOccurrenceNode
	}
)

// OccurrenceForTarget returns the unique captured semantic owner of target
// reachable from root, including root itself. It rejects foreign handles,
// targets outside this plan's reachable graph and ambiguous associations.
// Missing associations wrap ErrValuePlanAssociationNotFound. This query never
// resolves examples, materializes values or infers ownership from type names.
func (p ValuePlan) OccurrenceForTarget(root ValueOccurrence, target ValuePlanNode) (ValueOccurrence, error) {
	if p.context == nil || p.root == nil || p.queryIndex == nil || root.context != p.context || root.graph != p.source.graph || root.node == nil {
		return ValueOccurrence{}, fmt.Errorf("value plan occurrence does not belong to the plan source")
	}
	if target.node == nil {
		return ValueOccurrence{}, fmt.Errorf("value plan target node is absent")
	}
	if !p.containsNode(target.node) {
		return ValueOccurrence{}, fmt.Errorf("value plan target node does not belong to the plan")
	}
	p.queryIndex.once.Do(func() {
		p.queryIndex.sources = make(map[*valuePlanNode][]*valueOccurrenceNode)
		for _, source := range p.source.graph.nodes {
			for _, association := range p.associations[source] {
				p.queryIndex.sources[association.root] = append(p.queryIndex.sources[association.root], source)
			}
		}
	})
	var found *valueOccurrenceNode
	for _, source := range p.queryIndex.sources[target.node] {
		if source == found || !root.graph.descendsFrom(source, root.node) {
			continue
		}
		occurrence := ValueOccurrence{context: root.context, graph: root.graph, node: source}
		if _, err := p.ForOccurrence(occurrence, target); err != nil {
			return ValueOccurrence{}, err
		}
		if found != nil {
			return ValueOccurrence{}, fmt.Errorf("value plan target has ambiguous semantic owners")
		}
		found = source
	}
	if found == nil {
		return ValueOccurrence{}, fmt.Errorf("%w in the semantic region", ErrValuePlanAssociationNotFound)
	}
	return ValueOccurrence{context: root.context, graph: root.graph, node: found}, nil
}

// MetaLast returns the last captured value of key, or false when absent. It
// exposes no mutable metadata collections and does not copy the target graph.
func (n ValuePlanNode) MetaLast(key string) (string, bool) {
	if n.node == nil {
		return "", false
	}
	return n.node.attribute.Meta.Last(key)
}

// IsInlineHTTPBody reports whether this captured target is the synthetic named
// wrapper around an inline HTTP body. The zero handle returns false.
func (n ValuePlanNode) IsInlineHTTPBody() bool {
	if n.node == nil {
		return false
	}
	typ, ok := n.node.attribute.Type.(*UserTypeExpr)
	if !ok {
		return false
	}
	_, ok = typ.Attribute().Meta[inlineHTTPBodyMetaKey]
	return ok
}

func (g *valueOccurrenceGraph) descendsFrom(node, root *valueOccurrenceNode) bool {
	if node == root {
		return true
	}
	g.parentIndex.once.Do(func() {
		parents := make(map[*valueOccurrenceNode][]*valueOccurrenceNode)
		for _, parent := range g.nodes {
			decl := parent.declaration
			add := func(child *valueOccurrenceNode) {
				if child != nil {
					parents[child] = append(parents[child], parent)
				}
			}
			add(decl.alias)
			add(decl.element)
			add(decl.key)
			for _, member := range decl.members {
				add(member.node)
			}
			for _, branch := range decl.branches {
				add(branch.node)
			}
		}
		g.parentIndex.parents = parents
	})
	visited := make(map[*valueOccurrenceNode]bool)
	queue := []*valueOccurrenceNode{node}
	for len(queue) > 0 {
		current := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		if visited[current] {
			continue
		}
		visited[current] = true
		for _, parent := range g.parentIndex.parents[current] {
			if parent == root {
				return true
			}
			queue = append(queue, parent)
		}
	}
	return false
}
