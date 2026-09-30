package expr

type (
	valueSelectedInput struct {
		occurrence ValueIdentity
		branch     ValueIdentity
		payload    any
	}

	valueSynthesisGraph struct {
		occurrence   ValueOccurrence
		nodes        map[*valueOccurrenceNode]*AttributeExpr
		declarations map[*valueDeclarationNode]DataType
	}

	// The adapter changes only union choice storage. All sampling, constraints,
	// collection construction and recursion limits remain in the existing owner.
	valueSynthesisUnion struct {
		*Union
		occurrence ValueIdentity
		branches   []ValueIdentity
	}
)

// Synthesize attempts an example only for an absent, reachable selection.
// Repeated calls for that occurrence return the same retained choice. Authored
// failures are never replaced, and LegacyValue uses this exact sampling attempt.
func (c *ValueContext) Synthesize(selection ExampleSelection, generator *ExampleGenerator) ValueResult {
	if c == nil || selection.occurrence.context != c.identity || selection.occurrence.node == nil {
		return valueFailure(ValueInvalid, "ownership", "selection requires the same value context")
	}
	if selection.state == ExampleSelected {
		return valueFailure(ValueInvalid, "synthesis", "authored sources are ineligible for synthesis")
	}
	if selection.state != ExampleAbsent || generator == nil || generator.Randomizer == nil {
		return ValueResult{outcome: ValueSuppressed, role: ValueRoleExample}
	}
	c.synthesisMu.Lock()
	defer c.synthesisMu.Unlock()
	if prior, found := c.synthesized[selection.occurrence.node]; found {
		return prior
	}
	graph := valueSynthesisGraph{occurrence: selection.occurrence,
		nodes: make(map[*valueOccurrenceNode]*AttributeExpr), declarations: make(map[*valueDeclarationNode]DataType)}
	attribute := graph.attribute(selection.occurrence.node)
	effectiveConstraints := make(map[*AttributeExpr]*EffectiveConstraints, len(graph.nodes))
	for node, prepared := range graph.nodes {
		effectiveConstraints[prepared] = node.constraints
	}
	// Raw memo entries from a different occurrence cannot own this occurrence's
	// branch choices or constraints. The existing recursion memo still operates
	// within this one synthesis graph; random draws use the original stream.
	random := &ExampleGenerator{Randomizer: generator.Randomizer, effectiveConstraints: effectiveConstraints}
	raw := attribute.Example(random)
	if raw == nil {
		result := ValueResult{outcome: ValueSuppressed, role: ValueRoleExample, synthesized: true}
		c.synthesized[selection.occurrence.node] = result
		return result
	}
	source := c.SupplyValue(ValueInput{Raw: raw, Origin: "synthesized example"})
	result := c.Resolve(selection.occurrence, source, ValueRoleExample)
	snapshot := valueSnapshotState{eraseSelections: true}
	legacy := snapshot.snapshot(raw)
	result.legacy, result.synthesized = &legacy, true
	c.synthesized[selection.occurrence.node] = result
	return result
}

func (g *valueSynthesisGraph) attribute(node *valueOccurrenceNode) *AttributeExpr {
	if prior := g.nodes[node]; prior != nil {
		return prior
	}
	copy := *node.attribute
	copy.Validation = node.constraints.Validation().Lowered()
	if copy.Validation.HasRequiredOnly() && len(copy.Validation.Required) == 0 {
		copy.Validation = nil
	}
	g.nodes[node] = &copy
	if node.declaration.kind == UnionKind {
		union := *node.declaration.typ.(*Union)
		union.Values = nil
		occurrence := g.occurrence
		occurrence.node = node
		adapter := &valueSynthesisUnion{Union: &union, occurrence: occurrence.ID()}
		for _, branch := range node.declaration.branches {
			union.Values = append(union.Values, &NamedAttributeExpr{Name: branch.name, Attribute: g.attribute(branch.node)})
			adapter.branches = append(adapter.branches, ValueIdentity{context: occurrence.context, graph: occurrence.graph, index: branch.id})
		}
		copy.Type = adapter
	} else {
		copy.Type = g.datatype(node.declaration)
	}
	return &copy
}

func (g *valueSynthesisGraph) datatype(declaration *valueDeclarationNode) DataType {
	if prior := g.declarations[declaration]; prior != nil {
		return prior
	}
	switch actual := declaration.typ.(type) {
	case UserType:
		copy := actual.Dup(nil)
		g.declarations[declaration] = copy
		copy.SetAttribute(g.attribute(declaration.alias))
		return copy
	case *Object:
		copy := &Object{}
		g.declarations[declaration] = copy
		for _, member := range declaration.members {
			*copy = append(*copy, &NamedAttributeExpr{Name: member.name, Attribute: g.attribute(member.node)})
		}
		return copy
	case *Array:
		copy := &Array{NonNullableElems: actual.NonNullableElems}
		g.declarations[declaration] = copy
		copy.ElemType = g.attribute(declaration.element)
		return copy
	case *Map:
		copy := &Map{}
		g.declarations[declaration] = copy
		copy.KeyType, copy.ElemType = g.attribute(declaration.key), g.attribute(declaration.element)
		return copy
	default:
		return actual
	}
}

func (u *valueSynthesisUnion) Example(random *ExampleGenerator) any {
	if len(u.Values) == 0 {
		return nil
	}
	index := random.Int() % len(u.Values)
	return valueSelectedInput{occurrence: u.occurrence, branch: u.branches[index], payload: u.Values[index].Attribute.Example(random)}
}
