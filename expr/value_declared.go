package expr

import (
	"reflect"

	"github.com/CaliLuke/loom/internal/jsonkey"
)

// DeclaredJSONValue returns the retained semantic graph as a JSON-shaped host
// value without encoding it. Scalars retain their declared Go precision,
// Bytes remain []byte, objects use finalized wire names, map keys use their JSON
// member spelling, and unions retain their selected tagged or untagged shape.
// Explicit null returns nil with true. Complete and incomplete results preserve
// their distinct Outcome; invalid and absent results return false. An
// unsupported result succeeds only when it retains a structurally known value
// with opaque codec leaves and no independently checkable failure. Built-in
// values are detached, opaque custom leaves remain borrowed, and no codec is
// invoked. ProjectJSON remains the API for checked target wire emission.
func (r ValueResult) DeclaredJSONValue() (any, bool) {
	value, ok := declaredResultValue(r)
	if !ok {
		return nil, false
	}
	return declaredResolvedJSONValue(value)
}

// DeclaredJSONValue returns the retained declared value selected by a captured
// plan. It checks context and semantic ownership, then follows the plan's
// retained member identities without resolving or synthesizing again. Bytes
// remain []byte and scalar precision is preserved; no target codec, field
// renaming, presence filtering or wire validation is applied. Explicit null
// returns nil with true; absent, invalid, suppressed, foreign or schema-only
// inputs return false. Opaque leaves retain the same eligibility and borrowing
// rules as ValueResult.DeclaredJSONValue. Use ProjectJSON for checked target
// wire output.
func (c *ValueContext) DeclaredJSONValue(result ValueResult, plan ValuePlan) (any, bool) {
	if c == nil || plan.context != c.identity || plan.root == nil || plan.root.schemaOnly {
		return nil, false
	}
	if result.source == nil || result.source.context != c.identity ||
		result.occurrence.graph != plan.source.graph || result.occurrence.node != plan.source.node {
		return nil, false
	}
	value, ok := declaredResultValue(result)
	if !ok || value.node.context != c.identity || value.node.occurrence.graph != plan.source.graph {
		return nil, false
	}
	for _, selected := range plan.selection {
		value = projectionMember(value, selected)
	}
	return declaredResolvedJSONValue(value)
}

func declaredResultValue(result ValueResult) (ResolvedValue, bool) {
	if result.outcome != ValueResolved && result.outcome != ValueIncomplete &&
		!(result.outcome == ValueUnsupported && result.value.node != nil && !result.checkableFailure) {
		return ResolvedValue{}, false
	}
	return result.value, true
}

func declaredResolvedJSONValue(value ResolvedValue) (any, bool) {
	if value.node == nil || value.node.presence == ValueAbsent {
		return nil, false
	}
	if value.node.presence == ValueNull {
		return nil, true
	}
	if value.node.opaque {
		return copyValueRaw(value.node.raw), true
	}
	switch value.node.kind {
	case ValueKindScalar:
		return copyValueRaw(value.node.scalar), true
	case ValueKindArray:
		return declaredArrayJSONValue(value)
	case ValueKindObject:
		return declaredObjectJSONValue(value)
	case ValueKindMap:
		return declaredMapJSONValue(value)
	case ValueKindUnion:
		return declaredUnionJSONValue(value)
	case ValueKindAny:
		return copyValueRaw(value.node.raw), true
	default:
		return nil, false
	}
}

func declaredArrayJSONValue(value ResolvedValue) (any, bool) {
	if value.node.presence == ValueNil {
		return []any{}, true
	}
	result := make([]any, len(value.node.elements))
	for index, element := range value.node.elements {
		resolved, ok := declaredResolvedJSONValue(element)
		if !ok {
			return nil, false
		}
		result[index] = resolved
	}
	return result, true
}

func declaredObjectJSONValue(value ResolvedValue) (any, bool) {
	result := make(map[string]any, len(value.node.fields))
	for _, field := range value.node.fields {
		if field.Value.Presence() == ValueAbsent {
			continue
		}
		resolved, ok := declaredResolvedJSONValue(field.Value)
		if !ok {
			return nil, false
		}
		name := field.Name
		if member, found := declaredMember(value, field.Member); found {
			name = member.wire
		}
		if name != "-" {
			result[name] = resolved
		}
	}
	return result, true
}

func declaredMapJSONValue(value ResolvedValue) (any, bool) {
	if value.node.presence == ValueNil {
		return map[string]any{}, true
	}
	result := make(map[string]any, len(value.node.entries))
	for _, entry := range value.node.entries {
		key, ok := declaredResolvedJSONValue(entry.Key)
		if !ok || key == nil {
			return nil, false
		}
		name, valid := jsonkey.Name(reflect.ValueOf(key))
		if !valid {
			return nil, false
		}
		if _, exists := result[name]; exists {
			return nil, false
		}
		resolved, ok := declaredResolvedJSONValue(entry.Value)
		if !ok {
			return nil, false
		}
		result[name] = resolved
	}
	return result, true
}

func declaredUnionJSONValue(value ResolvedValue) (any, bool) {
	payload, ok := declaredResolvedJSONValue(value.node.payload)
	if !ok {
		return nil, false
	}
	branch, found := declaredBranch(value, value.node.branch)
	if !found {
		return nil, false
	}
	declaration := value.node.occurrence.node.declaration
	if declaration.untagged {
		return payload, true
	}
	return map[string]any{declaration.typeKey: branch.tag, declaration.valueKey: payload}, true
}

func declaredMember(value ResolvedValue, identity ValueIdentity) (valueOccurrenceMember, bool) {
	if value.node == nil || identity.index == 0 {
		return valueOccurrenceMember{}, false
	}
	for _, member := range value.node.occurrence.node.declaration.members {
		if member.id == identity.index {
			return member, true
		}
	}
	return valueOccurrenceMember{}, false
}

func declaredBranch(value ResolvedValue, identity ValueIdentity) (valueOccurrenceBranch, bool) {
	if value.node == nil || identity.index == 0 {
		return valueOccurrenceBranch{}, false
	}
	for _, branch := range value.node.occurrence.node.declaration.branches {
		if branch.id == identity.index {
			return branch, true
		}
	}
	return valueOccurrenceBranch{}, false
}
