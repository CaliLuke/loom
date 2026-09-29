package expr

import "fmt"

type (
	valuePlanKey struct {
		source *valueOccurrenceNode
		target *valueOccurrenceNode
	}
	valuePlanBuilder struct {
		request    ValuePlanRequest
		context    *ValueContext
		source     ValueOccurrence
		nodes      map[valuePlanKey]*valuePlanNode
		used       []bool
		containers []bool
		next       uint64
	}
)

// NewValuePlan captures a finalized target and maps it to a supplied semantic
// root using structural copy ancestry. It rejects foreign contexts, unrelated
// targets, ambiguous mappings and incomplete runtime field policies. It never
// interns by type hash or takes ownership of a mutable caller attribute.
func (c *ValueContext) NewValuePlan(source ValueOccurrence, request ValuePlanRequest) (ValuePlan, error) {
	if c == nil || source.context != c.identity || source.node == nil {
		return ValuePlan{}, fmt.Errorf("value plan requires its source context")
	}
	if request.Codec < ValueCodecJSON || request.Codec > ValueCodecCustom ||
		(request.Use != ValuePlanRuntime && request.Use != ValuePlanDocumentation) {
		return ValuePlan{}, fmt.Errorf("value plan requires an explicit codec and use")
	}
	selected := source.node
	var selection []uint64
	for _, name := range request.Selection {
		selected = valueUnderlyingOccurrence(selected)
		var match *valueOccurrenceMember
		for i := range selected.declaration.members {
			member := &selected.declaration.members[i]
			if member.name == name {
				match = member
				break
			}
		}
		if match == nil {
			return ValuePlan{}, fmt.Errorf("value plan selection has no authored member %q", name)
		}
		selection = append(selection, match.id)
		selected = match.node
	}
	if !valueSameAncestry(selected.origin, request.Target) {
		return ValuePlan{}, fmt.Errorf("value plan target does not descend from the selected occurrence")
	}
	target, err := c.NewOccurrence(request.Target)
	if err != nil {
		return ValuePlan{}, fmt.Errorf("value plan target: %w", err)
	}
	builder := valuePlanBuilder{request: request, context: c, source: source, nodes: make(map[valuePlanKey]*valuePlanNode), used: make([]bool, len(request.Fields)), containers: make([]bool, len(request.Containers))}
	root, err := builder.node(selected, target.node)
	if err != nil {
		return ValuePlan{}, err
	}
	for i, used := range builder.used {
		if !used {
			return ValuePlan{}, fmt.Errorf("value plan field policy %d does not identify a target member", i)
		}
	}
	for i, used := range builder.containers {
		if !used {
			return ValuePlan{}, fmt.Errorf("value plan container policy %d does not identify a target container", i)
		}
	}
	return ValuePlan{context: c.identity, source: source, root: root, selection: selection}, nil
}

func valueSameAncestry(left, right *AttributeExpr) bool {
	return left != nil && right != nil && valueAttributeOrigin(left) == valueAttributeOrigin(right)
}

func valueUnderlyingOccurrence(node *valueOccurrenceNode) *valueOccurrenceNode {
	for node.declaration.alias != nil {
		node = node.declaration.alias
	}
	return node
}

func (b *valuePlanBuilder) node(source, target *valueOccurrenceNode) (*valuePlanNode, error) {
	if target.declaration.alias == nil {
		source = valueUnderlyingOccurrence(source)
	}
	key := valuePlanKey{source: source, target: target}
	if prior := b.nodes[key]; prior != nil {
		return prior, nil
	}
	b.next++
	decl := target.declaration
	node := &valuePlanNode{
		id: b.next, source: source, attribute: target.attribute, kind: decl.kind,
		codec: b.request.Codec, documentary: b.request.Use == ValuePlanDocumentation,
		nullable: target.attribute.Nullable, nonNullableElements: decl.nonNullableElements,
		untagged: decl.untagged, typeKey: decl.typeKey, valueKey: decl.valueKey,
		schemaUnknown: target.attribute.Meta["openapi:additionalProperties"] == nil,
	}
	if values := target.attribute.Meta["openapi:additionalProperties"]; len(values) > 0 {
		node.schemaUnknown = values[len(values)-1] != "false"
	}
	b.nodes[key] = node
	if err := b.enums(node, source, target); err != nil {
		return nil, err
	}
	if decl.alias != nil {
		child := source
		if source.declaration.alias != nil {
			child = source.declaration.alias
		}
		var err error
		node.alias, err = b.node(child, decl.alias)
		return node, err
	}
	if source.declaration.kind != decl.kind {
		return nil, fmt.Errorf("value plan changes source kind %v to %v", source.declaration.kind, decl.kind)
	}
	if decl.kind == ObjectKind || decl.kind == UnionKind {
		policy, err := b.containerPolicy(target)
		if err != nil {
			return nil, err
		}
		node.runtimeUnknown = !policy.RejectUnknownMembers
		node.preserveAdditional = policy.PreserveAdditional
	}
	var err error
	switch decl.kind {
	case ObjectKind:
		err = b.object(node, source, target)
	case ArrayKind, MapKind:
		node.element, err = b.node(source.declaration.element, decl.element)
		if err == nil && decl.kind == MapKind {
			node.key, err = b.node(source.declaration.key, decl.key)
		}
	case UnionKind:
		err = b.union(node, source, target)
	}
	return node, err
}

func (b *valuePlanBuilder) object(node *valuePlanNode, source, target *valueOccurrenceNode) error {
	wires := make(map[string]bool)
	for _, field := range target.declaration.members {
		var matched *valueOccurrenceMember
		for i := range source.declaration.members {
			candidate := &source.declaration.members[i]
			if !valueSameAncestry(candidate.node.origin, field.node.origin) {
				continue
			}
			if matched != nil {
				return fmt.Errorf("value plan has ambiguous member ancestry %q", field.name)
			}
			matched = candidate
		}
		if matched == nil {
			return fmt.Errorf("value plan has unrelated member %q", field.name)
		}
		policy, err := b.fieldPolicy(target, field)
		if err != nil {
			return err
		}
		if policy.Visible && wires[policy.WireName] {
			return fmt.Errorf("value plan has duplicate emitted member %q", policy.WireName)
		}
		if policy.Visible {
			wires[policy.WireName] = true
		}
		child, err := b.node(matched.node, field.node)
		if err != nil {
			return err
		}
		node.members = append(node.members, valuePlanMember{
			source: matched.id, wire: policy.WireName, required: policy.Required,
			visible: policy.Visible, presence: policy.Presence, implicitDefault: policy.ImplicitDefault, node: child,
		})
	}
	return nil
}

func (b *valuePlanBuilder) fieldPolicy(parent *valueOccurrenceNode, field valueOccurrenceMember) (ValueFieldPolicy, error) {
	var matched *ValueFieldPolicy
	for i := range b.request.Fields {
		policy := &b.request.Fields[i]
		if policy.Parent != parent.origin || policy.Target != field.node.origin || policy.Name != field.name {
			continue
		}
		if matched != nil {
			return ValueFieldPolicy{}, fmt.Errorf("value plan has duplicate field policy %q", field.name)
		}
		matched, b.used[i] = policy, true
	}
	if matched == nil {
		if b.request.Use == ValuePlanRuntime {
			return ValueFieldPolicy{}, fmt.Errorf("value plan has no runtime field policy %q", field.name)
		}
		// Documentation observes the expression's names and presence; it makes
		// no claim about generated field tags or an actual runtime decoder.
		return ValueFieldPolicy{WireName: field.wire, Visible: field.wire != "-", Required: valueMemberRequired(parent, field), Presence: ValueFieldRetain}, nil
	}
	if matched.Presence < ValueFieldRetain || matched.Presence > ValueFieldImplicitDefault {
		return ValueFieldPolicy{}, fmt.Errorf("value plan has invalid field presence %q", field.name)
	}
	kind := valueUnderlyingOccurrence(field.node).declaration.kind
	if (kind >= IntKind && kind <= Float64Kind) && matched.NumericKind != kind {
		return ValueFieldPolicy{}, fmt.Errorf("value plan has incompatible numeric precision %q", field.name)
	}
	policy := *matched
	if policy.Presence == ValueFieldImplicitDefault {
		primitive, ok := valueUnderlyingOccurrence(field.node).declaration.typ.(Primitive)
		if !ok || primitive == Any || policy.ImplicitDefault == nil || !primitive.IsCompatible(policy.ImplicitDefault) {
			return ValueFieldPolicy{}, fmt.Errorf("value plan has no compatible scalar implicit default %q", field.name)
		}
		snapshot := snapshotValueSource(policy.ImplicitDefault)
		if snapshot.err != nil {
			return ValueFieldPolicy{}, fmt.Errorf("value plan implicit default %q: %w", field.name, snapshot.err)
		}
		policy.ImplicitDefault = snapshot.raw
	} else if policy.ImplicitDefault != nil {
		return ValueFieldPolicy{}, fmt.Errorf("value plan has an implicit default without its presence policy %q", field.name)
	}
	return policy, nil
}

func (b *valuePlanBuilder) containerPolicy(target *valueOccurrenceNode) (ValueContainerPolicy, error) {
	var matched *ValueContainerPolicy
	for i := range b.request.Containers {
		policy := &b.request.Containers[i]
		if policy.Target != target.origin {
			continue
		}
		if matched != nil {
			return ValueContainerPolicy{}, fmt.Errorf("value plan has duplicate container policy")
		}
		matched, b.containers[i] = policy, true
	}
	if matched == nil {
		if b.request.Use == ValuePlanRuntime {
			return ValueContainerPolicy{}, fmt.Errorf("value plan has no runtime container policy")
		}
		return ValueContainerPolicy{PreserveAdditional: true}, nil
	}
	return *matched, nil
}

func (b *valuePlanBuilder) enums(node *valuePlanNode, source, target *valueOccurrenceNode) error {
	node.hasEnum = target.attribute.Validation != nil && target.attribute.Validation.Values != nil
	for _, snapshot := range target.enumValues {
		if snapshot.err != nil {
			return fmt.Errorf("value plan enum source: %w", snapshot.err)
		}
		occurrence := ValueOccurrence{context: b.source.context, graph: b.source.graph, node: source}
		input := b.context.SupplyValue(ValueInput{Raw: snapshot.raw, ExplicitNull: snapshot.raw == nil, Origin: "target enum"})
		result := b.context.Resolve(occurrence, input, ValueRoleEnum)
		if result.Outcome() != ValueResolved {
			return fmt.Errorf("value plan enum does not resolve against its semantic occurrence: %v", result.Diagnostics())
		}
		value, _ := result.Value()
		node.enumValues = append(node.enumValues, value)
	}
	return nil
}

func (b *valuePlanBuilder) union(node *valuePlanNode, source, target *valueOccurrenceNode) error {
	for _, branch := range target.declaration.branches {
		var matched *valueOccurrenceBranch
		for i := range source.declaration.branches {
			candidate := &source.declaration.branches[i]
			if candidate.name == branch.name && valueSameAncestry(candidate.node.origin, branch.node.origin) {
				if matched != nil {
					return fmt.Errorf("value plan has ambiguous branch %q", branch.name)
				}
				matched = candidate
			}
		}
		if matched == nil {
			return fmt.Errorf("value plan has unrelated branch %q", branch.name)
		}
		child, childErr := b.node(matched.node, branch.node)
		if childErr != nil {
			return childErr
		}
		node.branches = append(node.branches, valuePlanBranch{source: matched.id, tag: branch.tag, node: child})
	}
	return nil
}
