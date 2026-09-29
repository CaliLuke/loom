package expr

import "fmt"

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
