package expr

import "fmt"

// checkKnownSchemaAnnotation preserves the boundary between an unknown codec
// leaf and a proven builtin failure beside it. It uses the same observer,
// constructor and schema evaluator as ProjectJSON; no predicate is reimplemented.
// True means some subtree remains owned by an external codec.
func checkKnownSchemaAnnotation(plan *valuePlanNode, value ResolvedValue, role ValueRole) (bool, error) {
	if value.Presence() == ValueAbsent {
		return false, nil
	}
	if value.node.opaque || plan.codec != ValueCodecJSON {
		return true, nil
	}
	if plan.alias != nil {
		return checkKnownSchemaAnnotation(plan.alias, value, role)
	}
	opaque, err := checkKnownSchemaChildren(plan, value, role)
	if err != nil || opaque {
		return opaque, err
	}
	observed, failure := observeJSON(plan, role, value)
	if failure != nil {
		return false, fmt.Errorf("required JSON schema value is unrepresentable: %s", failure.message)
	}
	wire, failure := constructJSON(plan, observed)
	if failure != nil {
		return false, fmt.Errorf("required JSON schema value is unrepresentable: %s", failure.message)
	}
	matched, err := schemaJSON(plan, wire)
	if err != nil {
		return false, err
	}
	if !matched {
		return false, fmt.Errorf("required JSON schema value is unrepresentable: canonical JSON does not satisfy the target schema")
	}
	return false, nil
}

func checkKnownSchemaChildren(plan *valuePlanNode, value ResolvedValue, role ValueRole) (bool, error) {
	opaque := false
	check := func(child *valuePlanNode, member ResolvedValue, path string) error {
		unknown, err := checkKnownSchemaAnnotation(child, member, role)
		opaque = opaque || unknown
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return nil
	}
	switch plan.kind {
	case AnyKind:
		// Any has no declared member predicates. A retained opaque payload is
		// an external codec boundary, not evidence about typed siblings.
		return schemaValueHasOpaque(value), nil
	case ObjectKind:
		for _, member := range plan.members {
			if member.visible {
				if err := check(member.node, projectionMember(value, member.source), member.wire); err != nil {
					return false, err
				}
			}
		}
	case ArrayKind:
		for index, member := range value.node.elements {
			if err := check(plan.element, member, fmt.Sprintf("element %d", index)); err != nil {
				return false, err
			}
		}
	case MapKind:
		for index, entry := range value.node.entries {
			opaque = opaque || schemaValueHasOpaque(entry.Key)
			if err := check(plan.element, entry.Value, fmt.Sprintf("entry %d", index)); err != nil {
				return false, err
			}
		}
	case UnionKind:
		for _, branch := range plan.branches {
			if branch.source == value.node.branch.index {
				return checkKnownSchemaAnnotation(branch.node, value.node.payload, role)
			}
		}
		return false, fmt.Errorf("schema annotation has no retained union branch")
	}
	return opaque, nil
}

func schemaValueHasOpaque(value ResolvedValue) bool {
	if value.node == nil {
		return false
	}
	if value.node.opaque || schemaValueHasOpaque(value.node.payload) {
		return true
	}
	for _, element := range value.node.elements {
		if schemaValueHasOpaque(element) {
			return true
		}
	}
	for _, entry := range value.node.entries {
		if schemaValueHasOpaque(entry.Key) || schemaValueHasOpaque(entry.Value) {
			return true
		}
	}
	for _, field := range value.node.fields {
		if schemaValueHasOpaque(field.Value) {
			return true
		}
	}
	return false
}
