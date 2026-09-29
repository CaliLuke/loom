package expr

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"slices"
	"sort"

	"github.com/CaliLuke/loom/internal/jsonkey"
	loom "github.com/CaliLuke/loom/pkg"
)

type valueJSONDomain uint8

const (
	valueJSONValueDomain valueJSONDomain = iota
	valueJSONKeyDomain
)

// decodeJSON is independent of source matching and schema branch counts. It
// decodes the same canonical wire using the target's runtime scalar categories.
func decodeJSON(plan *valuePlanNode, wire jsontext.Value) (ResolvedValue, bool) {
	decoded, valid := decodeJSONBody(plan, wire)
	if !valid || !decodedJSONAllowed(plan, decoded, valueJSONValueDomain) {
		return ResolvedValue{}, false
	}
	return decoded, true
}

// All decoder entry paths apply the owning occurrence's constraints and enums
// after their body, including aliases and object-member key decoding.
func decodedJSONAllowed(plan *valuePlanNode, decoded ResolvedValue, domain valueJSONDomain) bool {
	if !valueLocalRules(plan.attribute, decoded) {
		return false
	}
	if !plan.hasEnum {
		return true
	}
	for _, member := range plan.enumValues {
		if domain == valueJSONKeyDomain {
			observed, valid := observeJSONKey(plan, member)
			if valid && projectionEquivalent(observed, decoded, true) {
				return true
			}
			continue
		}
		observed, failure := observeJSON(plan, ValueRoleEnum, member)
		if failure == nil && projectionEquivalent(observed, decoded, true) {
			return true
		}
	}
	return false
}

func decodeJSONBody(plan *valuePlanNode, wire jsontext.Value) (ResolvedValue, bool) {
	if !wire.IsValid() {
		return ResolvedValue{}, false
	}
	if wire.Kind() == 'n' && plan.nullable {
		return ResolvedValue{node: &resolvedValueNode{presence: ValueNull}}, true
	}
	if plan.alias != nil {
		return decodeJSON(plan.alias, wire)
	}
	result := ResolvedValue{node: &resolvedValueNode{presence: ValuePresent, occurrence: ValueOccurrence{node: plan.source}}}
	switch plan.kind {
	case AnyKind:
		result.node.kind, result.node.json = ValueKindAny, append(jsontext.Value(nil), wire...)
		return result, true
	case ArrayKind:
		var items []jsontext.Value
		if wire.Kind() != '[' || json.Unmarshal(wire, &items) != nil {
			return ResolvedValue{}, false
		}
		result.node.kind = ValueKindArray
		for _, item := range items {
			child, valid := decodeJSON(plan.element, item)
			if !valid || plan.nonNullableElements && item.Kind() == 'n' {
				return ResolvedValue{}, false
			}
			result.node.elements = append(result.node.elements, child)
		}
		return result, true
	case ObjectKind:
		return decodeJSONObject(plan, wire, result)
	case MapKind:
		return decodeJSONMap(plan, wire, result)
	case UnionKind:
		return decodeJSONUnion(plan, wire, result)
	default:
		scalar, valid := decodeJSONScalar(plan.kind, wire)
		result.node.kind, result.node.scalar = ValueKindScalar, scalar
		return result, valid
	}
}

func decodeJSONObject(plan *valuePlanNode, wire jsontext.Value, result ResolvedValue) (ResolvedValue, bool) {
	var entries map[string]jsontext.Value
	if wire.Kind() != '{' || json.Unmarshal(wire, &entries) != nil {
		return ResolvedValue{}, false
	}
	result.node.kind = ValueKindObject
	known := make(map[string]bool)
	for _, member := range plan.members {
		if !member.visible {
			continue
		}
		known[member.wire] = true
		input, present := entries[member.wire]
		var child ResolvedValue
		switch {
		case present:
			var valid bool
			child, valid = decodeJSON(member.node, input)
			if !valid {
				return ResolvedValue{}, false
			}
		case member.required:
			return ResolvedValue{}, false
		case member.presence == ValueFieldImplicitDefault:
			child = ResolvedValue{node: &resolvedValueNode{presence: ValuePresent, kind: ValueKindScalar, scalar: member.implicitDefault}}
		}
		result.node.fields = append(result.node.fields, ResolvedField{Member: ValueIdentity{index: member.source}, Name: member.wire, Value: child})
	}
	var extras []string
	for name := range entries {
		if !known[name] {
			extras = append(extras, name)
		}
	}
	if len(extras) > 0 && !plan.runtimeUnknown {
		return ResolvedValue{}, false
	}
	sort.Strings(extras)
	if plan.preserveAdditional {
		for _, name := range extras {
			result.node.fields = append(result.node.fields, ResolvedField{Name: name, Value: ResolvedValue{node: &resolvedValueNode{presence: ValuePresent, json: entries[name]}}})
		}
	}
	return result, true
}

func decodeJSONMap(plan *valuePlanNode, wire jsontext.Value, result ResolvedValue) (ResolvedValue, bool) {
	var entries map[string]jsontext.Value
	if wire.Kind() != '{' || json.Unmarshal(wire, &entries) != nil {
		return ResolvedValue{}, false
	}
	result.node.kind = ValueKindMap
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	seen := make(map[any]bool)
	for _, name := range names {
		key, valid := decodeJSONKey(plan.key, name)
		if !valid {
			return ResolvedValue{}, false
		}
		_, valid = jsonkey.Name(reflect.ValueOf(key.node.scalar))
		if !valid || seen[key.node.scalar] {
			return ResolvedValue{}, false
		}
		seen[key.node.scalar] = true
		child, valid := decodeJSON(plan.element, entries[name])
		if !valid {
			return ResolvedValue{}, false
		}
		result.node.entries = append(result.node.entries, ResolvedEntry{Key: key, Value: child})
	}
	return result, true
}

func decodeJSONUnion(plan *valuePlanNode, wire jsontext.Value, result ResolvedValue) (ResolvedValue, bool) {
	result.node.kind = ValueKindUnion
	if plan.untagged {
		matches := 0
		for _, branch := range plan.branches {
			child, valid := decodeJSON(branch.node, wire)
			if valid {
				matches++
				result.node.branch, result.node.payload = ValueIdentity{index: branch.source}, child
			}
		}
		return result, matches == 1
	}
	var envelope map[string]jsontext.Value
	if wire.Kind() != '{' || json.Unmarshal(wire, &envelope) != nil {
		return ResolvedValue{}, false
	}
	var tag string
	if json.Unmarshal(envelope[plan.typeKey], &tag) != nil || !plan.runtimeUnknown && len(envelope) != 2 {
		return ResolvedValue{}, false
	}
	for _, branch := range plan.branches {
		if branch.tag == tag {
			child, valid := decodeJSON(branch.node, envelope[plan.valueKey])
			result.node.branch, result.node.payload = ValueIdentity{index: branch.source}, child
			return result, valid
		}
	}
	return ResolvedValue{}, false
}

func decodeJSONScalar(kind Kind, wire jsontext.Value) (any, bool) {
	typ := valueJSONScalarType(kind)
	if typ == nil {
		return nil, false
	}
	pointer := reflect.New(typ)
	if wire.Kind() == 'n' || json.Unmarshal(wire, pointer.Interface(), loom.JSONOptions()) != nil {
		return nil, false
	}
	return pointer.Elem().Interface(), true
}

// valueJSONScalarType retains the actual generated carrier's width and sign.
// Scalar values and map keys must use the same physical type owner.
func valueJSONScalarType(kind Kind) reflect.Type {
	switch kind {
	case BooleanKind:
		return reflect.TypeFor[bool]()
	case StringKind:
		return reflect.TypeFor[string]()
	case BytesKind:
		return reflect.TypeFor[[]byte]()
	case IntKind:
		return reflect.TypeFor[int]()
	case Int32Kind:
		return reflect.TypeFor[int32]()
	case Int64Kind:
		return reflect.TypeFor[int64]()
	case UIntKind:
		return reflect.TypeFor[uint]()
	case UInt32Kind:
		return reflect.TypeFor[uint32]()
	case UInt64Kind:
		return reflect.TypeFor[uint64]()
	case Float32Kind:
		return reflect.TypeFor[float32]()
	case Float64Kind:
		return reflect.TypeFor[float64]()
	default:
		return nil
	}
}

func decodeJSONKey(plan *valuePlanNode, text string) (ResolvedValue, bool) {
	result, valid := decodeJSONKeyBody(plan, text)
	if !valid || !decodedJSONAllowed(plan, result, valueJSONKeyDomain) {
		return ResolvedValue{}, false
	}
	return result, true
}

// Map keys observe canonical object-member names and the native key carrier.
// Any key enums therefore observe string names, not ordinary Any JSON snapshots.
// Typed keys retain the actual decoded key width and numerical semantics.
func observeJSONKey(plan *valuePlanNode, member ResolvedValue) (ResolvedValue, bool) {
	if member.Kind() == ValueKindAny {
		member = member.node.payload
	}
	scalar, present := member.Scalar()
	if !present {
		return ResolvedValue{}, false
	}
	name, valid := jsonkey.Name(reflect.ValueOf(scalar))
	if !valid {
		return ResolvedValue{}, false
	}
	for plan.alias != nil {
		plan = plan.alias
	}
	return decodeJSONKeyBody(plan, name)
}

func decodeJSONKeyBody(plan *valuePlanNode, text string) (ResolvedValue, bool) {
	if plan.alias != nil {
		return decodeJSONKey(plan.alias, text)
	}
	typ := valueJSONScalarType(plan.kind)
	if plan.kind == AnyKind {
		typ = reflect.TypeFor[any]()
	}
	if typ == nil || !typ.Comparable() {
		return ResolvedValue{}, false
	}
	// Decode one object member through the same typed map and Loom options as
	// generated carriers. strconv alone admits spellings that JSON rejects.
	wire, err := json.Marshal(map[string]jsontext.Value{text: jsontext.Value("null")})
	if err != nil {
		return ResolvedValue{}, false
	}
	pointer := reflect.New(reflect.MapOf(typ, reflect.TypeFor[jsontext.Value]()))
	if json.Unmarshal(wire, pointer.Interface(), loom.JSONOptions()) != nil {
		return ResolvedValue{}, false
	}
	entries := pointer.Elem().MapRange()
	if !entries.Next() {
		return ResolvedValue{}, false
	}
	scalar := entries.Key().Interface()
	result := ResolvedValue{node: &resolvedValueNode{presence: ValuePresent, kind: ValueKindScalar, scalar: scalar}}
	return result, true
}

func projectionEqual(left, right ResolvedValue) bool {
	return projectionEquivalent(left, right, false)
}

func projectionEquivalent(left, right ResolvedValue, enumeration bool) bool {
	if left.Presence() != right.Presence() {
		return false
	}
	if left.Presence() == ValueAbsent || left.Presence() == ValueNull {
		return true
	}
	if len(left.node.json) > 0 || len(right.node.json) > 0 {
		if enumeration {
			return loom.JSONValueEqual(left.node.json, right.node.json)
		}
		return bytes.Equal(left.node.json, right.node.json)
	}
	if left.Kind() != right.Kind() {
		return false
	}
	switch left.Kind() {
	case ValueKindScalar:
		return (enumeration || projectionScalarCategory(left.node.scalar) == projectionScalarCategory(right.node.scalar)) && exampleValuesEqual(left.node.scalar, right.node.scalar)
	case ValueKindArray:
		return slices.EqualFunc(left.node.elements, right.node.elements, func(a, b ResolvedValue) bool {
			return projectionEquivalent(a, b, enumeration)
		})
	case ValueKindObject:
		return valueUnorderedEqual(left.node.fields, right.node.fields, func(a, b ResolvedField) bool {
			return a.Member.index == b.Member.index && a.Name == b.Name && projectionEquivalent(a.Value, b.Value, enumeration)
		})
	case ValueKindMap:
		return valueUnorderedEqual(left.node.entries, right.node.entries, func(a, b ResolvedEntry) bool {
			return resolvedKeyEqual(a.Key, b.Key) && projectionEquivalent(a.Value, b.Value, enumeration)
		})
	case ValueKindUnion:
		return left.node.branch.index == right.node.branch.index && left.node.occurrence.node == right.node.occurrence.node && projectionEquivalent(left.node.payload, right.node.payload, enumeration)
	}
	return true
}
