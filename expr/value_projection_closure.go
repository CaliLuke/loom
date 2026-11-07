package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/CaliLuke/loom/internal/unionmatch"
	loom "github.com/CaliLuke/loom/pkg"
)

// schemaUnionClosure uses the lowered annotation owner after the prepared
// schema has matched. Tagged envelopes evaluate their two envelope members.
func schemaUnionClosure(plan *valuePlanNode, wire jsontext.Value) (bool, error) {
	root := plan
	for root.alias != nil {
		root = root.alias
	}
	if !root.untagged {
		var envelope map[string]jsontext.Value
		if err := json.Unmarshal(wire, &envelope); err != nil {
			return false, err
		}
		return len(envelope) == 2, nil
	}
	shape, err := lowerJSONShape(plan, make(map[*valuePlanNode]*loom.JSONShape))
	if err != nil {
		return false, err
	}
	return shape.Match(wire)
}

// decodedUnionClosure follows the decoder's retained branch identity. It must
// not reselect using schema counts: narrower Go carriers have their own domain.
func decodedUnionClosure(plan *valuePlanNode, wire jsontext.Value, decoded ResolvedValue) bool {
	if !plan.unionClosed || wire.Kind() != '{' {
		return true
	}
	var members map[string]jsontext.Value
	if json.Unmarshal(wire, &members) != nil {
		return false
	}
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	evaluated := decodedEvaluatedMembers(plan, decoded, names)
	return unionmatch.CheckEvaluated("JSON union", names, evaluated) == nil
}

func decodedEvaluatedMembers(plan *valuePlanNode, decoded ResolvedValue, members []string) []string {
	if plan.alias != nil {
		return decodedEvaluatedMembers(plan.alias, decoded, members)
	}
	switch plan.kind {
	case UnionKind:
		if !plan.untagged {
			return []string{plan.typeKey, plan.valueKey}
		}
		for _, branch := range plan.branches {
			if branch.source == decoded.node.branch.index {
				return decodedEvaluatedMembers(branch.node, decoded.node.payload, members)
			}
		}
	case MapKind:
		return members
	case ObjectKind:
		if !plan.schemaUnknown || plan.additional != nil || plan.attribute.ObjectOpenness() == ObjectOpen {
			return members
		}
		names := make([]string, 0, len(plan.members))
		for _, field := range plan.members {
			if field.visible {
				names = append(names, field.wire)
			}
		}
		return names
	}
	return nil
}
