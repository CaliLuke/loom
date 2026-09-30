package expr

import (
	"reflect"
	"sort"
	"strconv"

	"github.com/CaliLuke/loom/internal/jsonkey"
	"github.com/CaliLuke/loom/internal/naming"
)

func (r *valueResolver) array(node *valueOccurrenceNode, raw any, complete bool) ValueResult {
	input := reflect.ValueOf(raw)
	if !input.IsValid() || (input.Kind() != reflect.Array && input.Kind() != reflect.Slice) {
		return valueFailure(ValueInvalid, "type", "expected an array")
	}
	if !r.admission && !valueLengthAllowed(effectiveNodeValidation(node), input.Len()) {
		return valueFailure(ValueInvalid, "length", "array length violates constraints")
	}
	presence := ValuePresent
	if input.Kind() == reflect.Slice && input.IsNil() {
		presence = ValueNil
	}
	result := r.success(node, presence, ValueKindArray, raw)
	var children []ValueResult
	for index := range input.Len() {
		rawChild := input.Index(index).Interface()
		child := r.resolve(node.declaration.element, rawChild, complete, node.declaration.element, nil)
		if node.declaration.nonNullableElements && valueNullInput(rawChild) {
			child = valueFailure(ValueInvalid, "null", "array element does not admit null")
		}
		children = append(children, valueAtPath(child, strconv.Itoa(index)))
		result.value.node.elements = append(result.value.node.elements, child.value)
	}
	return valueCombine(result, children, r.admission)
}

func (r *valueResolver) object(node, owner *valueOccurrenceNode, raw any, complete bool) ValueResult {
	entries, ok := objectSourceEntries(node, raw)
	if !ok || entries == nil {
		return valueFailure(ValueInvalid, "type", "expected a non-nil object")
	}
	// Wire aliases belong to target representations, so distinct authored
	// members may share one. A supplied shared alias has no unique source owner;
	// reject it before assigning that entry to any member. Cross-namespace
	// authored-name/unique-alias overlaps retain the existing member rules.
	aliases := make(map[string]bool)
	for _, member := range node.declaration.members {
		if aliases[member.wire] {
			if _, supplied := entries[member.wire]; supplied {
				return valueAtPath(valueFailure(ValueAmbiguous, "alias", "wire alias identifies multiple source members"), member.wire)
			}
		}
		aliases[member.wire] = true
	}
	result := r.success(node, ValuePresent, ValueKindObject, raw)
	known := make(map[string]bool)
	var children []ValueResult
	for _, member := range node.declaration.members {
		known[member.name], known[member.wire] = true, true
		identity := ValueIdentity{context: r.context.identity, graph: r.occurrence.graph, index: member.id}
		child := r.objectMember(owner, member, entries, complete)
		result.value.node.fields = append(result.value.node.fields, ResolvedField{Member: identity, Name: member.name, Value: child.value})
		children = append(children, child)
	}
	var extraNames []string
	for name := range entries {
		if !known[name] {
			extraNames = append(extraNames, name)
		}
	}
	sort.Strings(extraNames)
	additional, specified := owner.attribute.Meta.Last("openapi:additionalProperties")
	if len(extraNames) > 0 && specified && additional == "false" {
		return valueFailure(ValueInvalid, "additional", "undeclared object members are not allowed")
	}
	for _, name := range extraNames {
		child := valueAtPath(r.raw(node, entries[name]), name)
		children = append(children, child)
		result.value.node.fields = append(result.value.node.fields, ResolvedField{Name: name, Value: child.value})
	}
	return valueCombine(result, children, r.admission)
}

func objectSourceEntries(node *valueOccurrenceNode, raw any) (map[string]any, bool) {
	entries, ok := stringMapExample(raw)
	if !ok {
		return nil, false
	}
	value := reflect.ValueOf(raw)
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() || valueHasCustomCodec(value.Type()) {
			return entries, true
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return entries, true
	}
	for _, member := range node.declaration.members {
		if _, authored := entries[member.name]; authored {
			continue
		}
		if _, wire := entries[member.wire]; wire {
			continue
		}
		goName := naming.GoifyAttribute(member.name, member.node.attribute.Meta, true)
		entry, found := entries[goName]
		if !found {
			continue
		}
		entries[member.name] = entry
		if goName != member.name && goName != member.wire {
			delete(entries, goName)
		}
	}
	return entries, true
}

func (r *valueResolver) mapping(node *valueOccurrenceNode, raw any, complete bool) ValueResult {
	input := reflect.ValueOf(raw)
	if !input.IsValid() || input.Kind() != reflect.Map {
		return valueFailure(ValueInvalid, "type", "expected a map")
	}
	if !r.admission && !valueLengthAllowed(effectiveNodeValidation(node), input.Len()) {
		return valueFailure(ValueInvalid, "length", "map length violates constraints")
	}
	presence := ValuePresent
	if input.IsNil() {
		presence = ValueNil
	}
	result := r.success(node, presence, ValueKindMap, raw)
	entries := valueSortedMapEntries(input)
	var keyResults []ValueResult
	for _, entry := range entries {
		keyResults = append(keyResults, r.resolve(node.declaration.key, entry.key.Interface(), true, node.declaration.key, nil))
	}
	checked := valueCombine(result, keyResults, r.admission)
	if valueErrorPriority(checked) != 99 &&
		!(checked.outcome == ValueUnsupported && !checked.checkableFailure && checked.value.node != nil) {
		return checked
	}
	seen := make(map[string]bool)
	children := append([]ValueResult(nil), keyResults...)
	for index, entry := range entries {
		keyValue := keyResults[index].value
		if keyValue.Kind() == ValueKindAny {
			keyValue = keyValue.node.payload
		}
		opaqueKey := keyValue.node != nil && keyValue.node.opaque
		if keyValue.Kind() != ValueKindScalar && !opaqueKey {
			return valueFailure(ValueInvalid, "key", "map key must be a builtin scalar")
		}
		name := valueKeyOrder(entry.key)
		if !opaqueKey {
			var valid bool
			name, valid = jsonkey.Name(reflect.ValueOf(keyValue.node.scalar))
			if !valid || seen[name] {
				return valueFailure(ValueInvalid, "key", "unsupported or colliding map key spelling")
			}
			seen[name] = true
		}
		child := valueAtPath(r.resolve(node.declaration.element, entry.value.Interface(), complete, node.declaration.element, nil), name)
		children = append(children, child)
		result.value.node.entries = append(result.value.node.entries, ResolvedEntry{Key: keyValue, Value: child.value})
	}
	return valueCombine(result, children, r.admission)
}

func (r *valueResolver) raw(node *valueOccurrenceNode, raw any) ValueResult {
	if valueNullInput(raw) {
		return r.success(node, ValueNull, 0, raw)
	}
	input := reflect.ValueOf(raw)
	if valueHasCustomCodec(input.Type()) {
		if r.admission {
			return r.opaque(node, raw, "custom value requires codec materialization")
		}
		return valueFailure(ValueUnsupported, "custom", "custom value requires codec materialization")
	}
	leave, entered := r.enter(raw)
	if !entered {
		return valueFailure(ValueInvalid, "cyclic", "cyclic value")
	}
	defer leave()
	if valueIsBytes(input) {
		presence := ValuePresent
		if input.IsNil() {
			presence = ValueNil
		}
		result := r.success(node, presence, ValueKindScalar, raw)
		result.value.node.scalar = raw
		return result
	}
	switch input.Kind() {
	case reflect.Bool, reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		if input.Kind() == reflect.Float32 || input.Kind() == reflect.Float64 {
			if _, finite := numericExampleRat(raw); !finite {
				return valueFailure(ValueInvalid, "number", "non-finite numbers are not JSON values")
			}
		}
		result := r.success(node, ValuePresent, ValueKindScalar, raw)
		result.value.node.scalar = raw
		return result
	case reflect.Array, reflect.Slice:
		presence := ValuePresent
		if input.Kind() == reflect.Slice && input.IsNil() {
			presence = ValueNil
		}
		result := r.success(node, presence, ValueKindArray, raw)
		var children []ValueResult
		for index := range input.Len() {
			child := valueAtPath(r.raw(node, input.Index(index).Interface()), strconv.Itoa(index))
			children = append(children, child)
			result.value.node.elements = append(result.value.node.elements, child.value)
		}
		return valueCombine(result, children, r.admission)
	case reflect.Map:
		return r.rawMap(node, raw, input)
	default:
		return valueFailure(ValueUnsupported, "type", "value is outside finite builtin data")
	}
}

func (r *valueResolver) rawMap(node *valueOccurrenceNode, raw any, input reflect.Value) ValueResult {
	presence := ValuePresent
	if input.IsNil() {
		presence = ValueNil
	}
	result := r.success(node, presence, ValueKindMap, raw)
	seen := make(map[string]bool)
	var children []ValueResult
	for _, entry := range valueSortedMapEntries(input) {
		keyResult := r.raw(node, entry.key.Interface())
		opaqueKey := r.admission && keyResult.outcome == ValueUnsupported &&
			!keyResult.checkableFailure && keyResult.value.node != nil && keyResult.value.node.opaque
		if !opaqueKey && (keyResult.outcome != ValueResolved || keyResult.value.Kind() != ValueKindScalar) {
			return valueFailure(ValueInvalid, "key", "map key must be a builtin scalar")
		}
		name := valueKeyOrder(entry.key)
		if !opaqueKey {
			var valid bool
			name, valid = jsonkey.Name(entry.key)
			if !valid || seen[name] {
				return valueFailure(ValueInvalid, "key", "unsupported or colliding map key spelling")
			}
			seen[name] = true
		}
		child := valueAtPath(r.raw(node, entry.value.Interface()), name)
		children = append(children, keyResult, child)
		result.value.node.entries = append(result.value.node.entries, ResolvedEntry{Key: keyResult.value, Value: child.value})
	}
	return valueCombine(result, children, r.admission)
}

func valueIsBytes(value reflect.Value) bool {
	return value.IsValid() && value.Kind() == reflect.Slice && value.Type().Elem() == reflect.TypeFor[byte]()
}

func (r *valueResolver) objectMember(owner *valueOccurrenceNode, member valueOccurrenceMember, entries map[string]any, complete bool) ValueResult {
	identity := ValueIdentity{context: r.context.identity, graph: r.occurrence.graph, index: member.id}
	var supplied []ValueResult
	for _, name := range []string{member.name, member.wire} {
		if value, found := entries[name]; found {
			supplied = append(supplied, valueAtPath(r.resolve(member.node, value, complete, member.node, nil), name))
		}
		if member.name == member.wire {
			break
		}
	}
	// Every spelling is checked before the winning wire spelling is chosen.
	checked := valueCombine(r.success(member.node, ValueAbsent, 0, nil), supplied, r.admission)
	if valueErrorPriority(checked) != 99 &&
		!(checked.outcome == ValueUnsupported && !checked.checkableFailure && checked.value.node != nil) {
		return checked
	}
	value, present := entries[member.wire]
	if !present {
		value, present = entries[member.name]
	}
	child := r.success(member.node, ValueAbsent, 0, nil)
	if present {
		child = valueAtPath(r.resolve(member.node, value, complete, member.node, nil), member.name)
		for index, path := range child.missing {
			child.missing[index] = append([]ValueIdentity{identity}, path...)
		}
	} else if !r.shapeNodes[owner] && valueMemberRequired(owner, member) {
		if complete {
			child = valueAtPath(valueFailure(ValueInvalid, "required", "required member is absent"), member.name)
		} else {
			child.outcome, child.missing = ValueIncomplete, [][]ValueIdentity{{identity}}
		}
	}
	return child
}
