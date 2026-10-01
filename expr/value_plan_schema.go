package expr

import (
	"errors"
	"fmt"
)

var (
	// ErrValuePlanAssociationNotFound reports that a valid occurrence and target
	// node have no captured association in this plan. Callers that search exact
	// captured ancestry may continue only for this error.
	ErrValuePlanAssociationNotFound = errors.New("value plan association not found")
)

type (
	// ValuePlanNode is an immutable representation node. Handles are comparable
	// within a captured plan and preserve recursive graph identity. The zero handle
	// denotes an absent child; allocation identity is not a public schema name.
	ValuePlanNode struct {
		node *valuePlanNode
	}

	// ValuePlanMember describes an actual target member, including invisible members.
	ValuePlanMember struct {
		// Name is the full authored target name.
		Name string
		// WireName is the actual emitted name.
		WireName string
		// Visible reports whether the representation emits this member.
		Visible bool
		// Required reports the representation's requiredness.
		Required bool
		// Presence describes field-local omission behavior.
		Presence ValueFieldPresence
		// Node is the immutable child representation.
		Node ValuePlanNode
	}

	// ValuePlanBranch describes a target union alternative without reselecting it.
	ValuePlanBranch struct {
		// Tag is the actual wire discriminator value.
		Tag string
		// Node is the immutable alternative representation.
		Node ValuePlanNode
	}
)

// Root returns the target root after the plan's semantic member selection.
func (p ValuePlan) Root() ValuePlanNode {
	return ValuePlanNode{p.root}
}

// ForOccurrence returns the already-captured representation associated with an
// exact semantic occurrence and target node. It never rebuilds a plan or infers
// ownership from copied attributes. Foreign, missing and ambiguous pairs are
// rejected.
func (p ValuePlan) ForOccurrence(source ValueOccurrence, target ValuePlanNode) (ValuePlan, error) {
	if p.context == nil || p.root == nil || source.context != p.context || source.graph != p.source.graph || source.node == nil {
		return ValuePlan{}, fmt.Errorf("value plan occurrence does not belong to the plan source")
	}
	if target.node == nil {
		return ValuePlan{}, fmt.Errorf("value plan target node is absent")
	}
	if !valuePlanContainsNode(p.root, target.node, make(map[*valuePlanNode]bool)) {
		return ValuePlan{}, fmt.Errorf("value plan target node does not belong to the plan")
	}
	if p.association != nil && source.node == p.association.source && target.node == p.association.root {
		return ValuePlan{context: p.context, source: source, root: p.association.root,
			selection: append([]uint64(nil), p.association.selection...), association: p.association,
			associations: p.associations}, nil
	}
	var matched []*valuePlanAssociation
	for _, association := range p.associations[source.node] {
		if association.root == target.node {
			matched = append(matched, association)
		}
	}
	if len(matched) == 0 {
		return ValuePlan{}, fmt.Errorf("%w for the occurrence and target", ErrValuePlanAssociationNotFound)
	}
	if len(matched) != 1 {
		return ValuePlan{}, fmt.Errorf("value plan has ambiguous representations for the occurrence and target")
	}
	association := matched[0]
	return ValuePlan{context: p.context, source: source, root: association.root,
		selection: append([]uint64(nil), association.selection...), association: association,
		associations: p.associations}, nil
}

func valuePlanContainsNode(root, target *valuePlanNode, visited map[*valuePlanNode]bool) bool {
	if root == nil || visited[root] {
		return false
	}
	if root == target {
		return true
	}
	visited[root] = true
	if valuePlanContainsNode(root.alias, target, visited) ||
		valuePlanContainsNode(root.element, target, visited) ||
		valuePlanContainsNode(root.key, target, visited) {
		return true
	}
	for _, member := range root.members {
		if valuePlanContainsNode(member.node, target, visited) {
			return true
		}
	}
	for _, branch := range root.branches {
		if valuePlanContainsNode(branch.node, target, visited) {
			return true
		}
	}
	return false
}

// Valid reports whether the handle denotes a captured target node.
func (n ValuePlanNode) Valid() bool {
	return n.node != nil
}

// Codec returns the actual representation owner, or zero for an absent node.
func (n ValuePlanNode) Codec() ValueCodec {
	if n.node == nil {
		return 0
	}
	return n.node.codec
}

// Attribute returns an owned copy of the finalized target declaration details.
// Editing it cannot alter the plan. Child identity must be read through the
// node queries, not inferred from pointers in this copy.
func (n ValuePlanNode) Attribute() *AttributeExpr {
	if n.node == nil {
		return nil
	}
	builder := &valueOccurrenceBuilder{
		graph: &valueOccurrenceGraph{}, types: make(map[DataType]*valueDeclarationNode),
		capturedSources: true,
	}
	copy, err := builder.occurrence(n.node.attribute)
	if err != nil {
		panic(fmt.Errorf("copying a validated immutable value plan: %w", err))
	}
	return copy.attribute
}

// Members returns an owned list of target object members.
func (n ValuePlanNode) Members() []ValuePlanMember {
	if n.node == nil {
		return nil
	}
	result := make([]ValuePlanMember, 0, len(n.node.members))
	for _, member := range n.node.members {
		result = append(result, ValuePlanMember{Name: member.name, WireName: member.wire,
			Visible: member.visible, Required: member.required, Presence: member.presence,
			Node: ValuePlanNode{member.node}})
	}
	return result
}

// Branches returns an owned list of target union alternatives.
func (n ValuePlanNode) Branches() []ValuePlanBranch {
	if n.node == nil {
		return nil
	}
	result := make([]ValuePlanBranch, 0, len(n.node.branches))
	for _, branch := range n.node.branches {
		result = append(result, ValuePlanBranch{Tag: branch.tag, Node: ValuePlanNode{branch.node}})
	}
	return result
}

// Underlying returns the child of a named declaration, or an absent handle.
func (n ValuePlanNode) Underlying() ValuePlanNode {
	if n.node == nil {
		return ValuePlanNode{}
	}
	return ValuePlanNode{n.node.alias}
}

// UnderlyingReusesSource reports whether the named-declaration edge introduces
// only a target representation wrapper. Its child matches the current source
// ancestry without consuming an authored alias. Structural target children may
// subsequently descend to the source's underlying shape for member identities.
// The role belongs to this edge, not to the potentially shared child handle.
func (n ValuePlanNode) UnderlyingReusesSource() bool {
	if n.node == nil || n.node.alias == nil {
		return false
	}
	return n.node.aliasReusesSource
}

// Element returns the array or map value representation, or an absent handle.
func (n ValuePlanNode) Element() ValuePlanNode {
	if n.node == nil {
		return ValuePlanNode{}
	}
	return ValuePlanNode{n.node.element}
}

// Key returns the map key representation, or an absent handle.
func (n ValuePlanNode) Key() ValuePlanNode {
	if n.node == nil {
		return ValuePlanNode{}
	}
	return ValuePlanNode{n.node.key}
}

// TargetDeclarationID returns the captured authored declaration identity of this
// target node. Controlled transport copies retain that identity when their Go
// names change. It is independent of the semantic source cursor, which may move
// through aliases at a different depth. An unnamed target returns an empty string.
func (n ValuePlanNode) TargetDeclarationID() string {
	if n.node == nil {
		return ""
	}
	return n.node.targetDeclarationID
}
