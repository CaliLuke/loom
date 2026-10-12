package expr

// compositionGraph separates named declaration discovery and inheritance edges
// from recursive member edges. Named references stay live until shapes finish.
type compositionGraph struct {
	attributes   []*AttributeExpr
	seen         map[*AttributeExpr]bool
	keep         map[copyTypeKey]struct{}
	declarations map[*AttributeExpr]UserType
}

// compositionCopy binds structural recursion to its enclosing declaration.
// A nil owner identifies a constraint snapshot that must be fully detached.
type compositionCopy struct {
	attribute *AttributeExpr
	owner     UserType
}

func newCompositionGraph(roots []unionBranchRoot) *compositionGraph {
	graph := &compositionGraph{seen: make(map[*AttributeExpr]bool), keep: make(map[copyTypeKey]struct{}), declarations: make(map[*AttributeExpr]UserType)}
	for _, root := range roots {
		graph.attribute(root.attribute, root.path)
	}
	return graph
}

func (g *compositionGraph) attribute(attribute *AttributeExpr, path string) {
	if attribute == nil || g.seen[attribute] {
		return
	}
	g.seen[attribute] = true
	if err := attribute.resolveTypeRef(); err != nil {
		compositionError(attribute, "%s: %v", path, err)
		return
	}
	attribute.compositionName = path
	g.attributes = append(g.attributes, attribute)
	g.typ(attribute.Type, path)
	for _, base := range attribute.Bases {
		g.typ(base, base.Name())
	}
	for _, reference := range attribute.References {
		g.typ(reference, reference.Name())
	}
}

func (g *compositionGraph) typ(typ DataType, path string) {
	switch actual := typ.(type) {
	case UserType:
		g.keep[typeCopyKey(actual)] = struct{}{}
		g.declarations[actual.Attribute()] = actual
		g.attribute(actual.Attribute(), actual.Name())
	case *Object:
		for _, field := range *actual {
			g.attribute(field.Attribute, path+"."+field.Name)
		}
	case *Array:
		g.attribute(actual.ElemType, path+"[]")
	case *Map:
		g.attribute(actual.KeyType, path+"[key]")
		g.attribute(actual.ElemType, path+"[value]")
	case *Union:
		for _, branch := range actual.Values {
			g.attribute(branch.Attribute, path+"."+branch.Name)
		}
	}
}

func (g *compositionGraph) validateInheritance() {
	state := make(map[*AttributeExpr]uint8)
	var visit func(*AttributeExpr)
	visit = func(attribute *AttributeExpr) {
		if state[attribute] == 1 {
			compositionError(attribute, "%s: recursive Extend inheritance", attribute.compositionName)
			return
		}
		if state[attribute] == 2 {
			return
		}
		state[attribute] = 1
		if named, ok := attribute.Type.(UserType); ok {
			visit(named.Attribute())
		}
		for _, base := range attribute.Bases {
			if named, ok := base.(UserType); ok {
				visit(named.Attribute())
			}
		}
		state[attribute] = 2
	}
	for _, attribute := range g.attributes {
		visit(attribute)
	}
}

// copyAttribute owns local and inline data while preserving named declarations
// whose effective shape may still be under construction.
func (c *designComposer) copyAttribute(attribute *AttributeExpr, owner UserType) *AttributeExpr {
	d := newDupper()
	d.keep = c.keep
	copy := d.DupAttribute(attribute)
	for _, kept := range d.kept {
		c.pending = append(c.pending, compositionCopy{attribute: kept, owner: owner})
	}
	return copy
}

func (c *designComposer) copyType(typ DataType) DataType {
	d := newDupper()
	d.keep = c.keep
	copy := d.DupType(typ)
	for _, kept := range d.kept {
		c.pending = append(c.pending, compositionCopy{attribute: kept, owner: c.owner})
	}
	return copy
}

// detachNamedCopies snapshots only complete shapes. Compute all replacements
// before publishing any, so later snapshots cannot depend on publication order.
// Structural back references bind to their enclosing named owner; constraint
// snapshots have no binding and retain a fully detached named graph.
func (c *designComposer) detachNamedCopies() {
	types := make([]DataType, len(c.pending))
	for i, copy := range c.pending {
		d := newDupper()
		if copy.owner != nil {
			d.uts[typeCopyKey(copy.owner)] = copy.owner
		}
		types[i] = d.DupType(copy.attribute.Type)
	}
	for i, copy := range c.pending {
		copy.attribute.Type = types[i]
	}
}
