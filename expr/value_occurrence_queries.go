package expr

type (
	// ValueMember is a copied descriptor of a declared semantic object member.
	ValueMember struct {
		// ID identifies the member edge within its occurrence graph.
		ID ValueIdentity
		// Name is the full authored member spelling.
		Name string
		// WireName is the expression's JSON alias, before a target policy is applied.
		WireName string
		// Occurrence carries this member's effective constraints and presence.
		Occurrence ValueOccurrence
	}
	// ValueBranch is a copied descriptor of a declared union alternative.
	ValueBranch struct {
		// ID identifies the branch edge independently of its payload shape.
		ID ValueIdentity
		// Name is the authored alternative name.
		Name string
		// Tag is the declared discriminator value.
		Tag string
		// Occurrence carries the branch's effective contract.
		Occurrence ValueOccurrence
	}
)

// Members returns ordered declared members. Named declarations are transparent;
// mutating the returned slice cannot change the captured occurrence.
func (o ValueOccurrence) Members() []ValueMember {
	if o.node == nil {
		return nil
	}
	node := valueUnderlyingOccurrence(o.node)
	members := make([]ValueMember, len(node.declaration.members))
	for i, member := range node.declaration.members {
		members[i] = ValueMember{ID: ValueIdentity{context: o.context, graph: o.graph, index: member.id}, Name: member.name, WireName: member.wire, Occurrence: o.child(member.node)}
	}
	return members
}

// Branches returns ordered union alternatives, transparently following named declarations.
func (o ValueOccurrence) Branches() []ValueBranch {
	if o.node == nil {
		return nil
	}
	node := valueUnderlyingOccurrence(o.node)
	branches := make([]ValueBranch, len(node.declaration.branches))
	for i, branch := range node.declaration.branches {
		branches[i] = ValueBranch{ID: ValueIdentity{context: o.context, graph: o.graph, index: branch.id}, Name: branch.name, Tag: branch.tag, Occurrence: o.child(branch.node)}
	}
	return branches
}

// Underlying returns the immediate underlying occurrence of a named declaration.
// The zero occurrence means that the receiver is not a named declaration.
func (o ValueOccurrence) Underlying() ValueOccurrence {
	if o.node == nil {
		return ValueOccurrence{}
	}
	return o.child(o.node.declaration.alias)
}

// Element returns the array element or map value occurrence, following named declarations.
func (o ValueOccurrence) Element() ValueOccurrence {
	if o.node == nil {
		return ValueOccurrence{}
	}
	return o.child(valueUnderlyingOccurrence(o.node).declaration.element)
}

// Key returns the map key occurrence, following named declarations.
func (o ValueOccurrence) Key() ValueOccurrence {
	if o.node == nil {
		return ValueOccurrence{}
	}
	return o.child(valueUnderlyingOccurrence(o.node).declaration.key)
}

func (o ValueOccurrence) child(node *valueOccurrenceNode) ValueOccurrence {
	if node == nil {
		return ValueOccurrence{}
	}
	return ValueOccurrence{context: o.context, graph: o.graph, node: node}
}
