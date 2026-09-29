package expr

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/CaliLuke/loom/internal/jsonkey"
	loom "github.com/CaliLuke/loom/pkg"
)

func (r *valueResolver) scalar(node *valueOccurrenceNode, raw any) ValueResult {
	// Admission precedes normalization: reflect.Kind and numeric equivalence
	// erase concrete host distinctions retained by the authored DSL contract.
	primitive, ok := node.declaration.typ.(Primitive)
	if !ok || !primitive.IsCompatible(raw) {
		return valueFailure(ValueInvalid, "type", "value has an incompatible scalar type")
	}
	input := reflect.ValueOf(raw)
	if !input.IsValid() {
		return valueFailure(ValueInvalid, "type", "scalar does not admit null")
	}
	var normalized any
	switch node.declaration.kind {
	case BooleanKind:
		if input.Kind() == reflect.Bool {
			normalized = input.Bool()
		}
	case StringKind:
		if input.Kind() == reflect.String {
			normalized = input.String()
		}
	case BytesKind:
		if input.Kind() == reflect.String {
			normalized = []byte(input.String())
		} else if valueIsBytes(input) {
			bytes := make([]byte, input.Len())
			reflect.Copy(reflect.ValueOf(bytes), input)
			normalized = bytes
		}
	case IntKind, Int32Kind, Int64Kind, UIntKind, UInt32Kind, UInt64Kind:
		switch input.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			normalized = raw
		}
	case Float32Kind, Float64Kind:
		if _, finite := numericExampleRat(raw); finite {
			bits := 64
			if node.declaration.kind == Float32Kind {
				bits = 32
			}
			// Match declared literal normalization, not binary widening. Raw Any
			// bypasses this conversion and keeps concrete host precision.
			number, err := strconv.ParseFloat(fmt.Sprint(raw), bits)
			if err == nil {
				if bits == 32 {
					normalized = float32(number)
				} else {
					normalized = number
				}
			}
		}
	}
	if normalized == nil {
		return valueFailure(ValueInvalid, "type", "value has an incompatible scalar kind")
	}
	result := r.success(node, ValuePresent, ValueKindScalar, normalized)
	result.value.node.scalar = normalized
	return result
}

func valueLengthAllowed(rules *ValidationExpr, length int) bool {
	return rules == nil || ((rules.MinLength == nil || length >= *rules.MinLength) && (rules.MaxLength == nil || length <= *rules.MaxLength))
}

func valueLocalRules(attribute *AttributeExpr, value ResolvedValue) bool {
	rules := attribute.Validation
	if rules == nil || value.Presence() == ValueNull {
		return true
	}
	if rules.MinLength != nil || rules.MaxLength != nil {
		length, applicable := resolvedLength(value)
		if applicable && ((rules.MinLength != nil && length < *rules.MinLength) || (rules.MaxLength != nil && length > *rules.MaxLength)) {
			return false
		}
	}
	if value.Kind() != ValueKindScalar {
		return true
	}
	raw := value.node.scalar
	if !checkMinMaxValue(attribute, raw) {
		return false
	}
	text, textValue := raw.(string)
	if rules.Pattern != "" && textValue {
		pattern, err := regexp.Compile(rules.Pattern)
		if err != nil || !pattern.MatchString(text) {
			return false
		}
	}
	if rules.Format != "" && textValue {
		return loom.ValidateFormat("value", text, loom.Format(rules.Format)) == nil
	}
	return true
}

func resolvedLength(value ResolvedValue) (int, bool) {
	switch value.Kind() {
	case ValueKindArray:
		return len(value.node.elements), true
	case ValueKindMap:
		return len(value.node.entries), true
	case ValueKindScalar:
		switch scalar := value.node.scalar.(type) {
		case string:
			return utf8.RuneCountInString(scalar), true
		case []byte:
			return len(scalar), true
		}
	}
	return 0, false
}

func resolvedEnumEqual(left, right ResolvedValue) bool {
	if left.Presence() != right.Presence() {
		if !((left.Presence() == ValueNil || left.Presence() == ValuePresent) && (right.Presence() == ValueNil || right.Presence() == ValuePresent)) {
			return false
		}
	}
	if left.Kind() != right.Kind() {
		return false
	}
	if left.node == nil || right.node == nil || left.Presence() == ValueAbsent || left.Presence() == ValueNull {
		return left.Presence() == right.Presence()
	}
	switch left.Kind() {
	case ValueKindScalar:
		return exampleValuesEqual(left.node.scalar, right.node.scalar)
	case ValueKindAny:
		// Both payloads were structurally validated before the host equality
		// precheck. Concrete map and dynamic child types remain in the snapshot.
		return exampleValuesEqual(left.node.raw, right.node.raw)
	case ValueKindArray:
		return slices.EqualFunc(left.node.elements, right.node.elements, resolvedEnumEqual)
	case ValueKindObject:
		return valueUnorderedEqual(left.node.fields, right.node.fields, resolvedFieldEqual)
	case ValueKindMap:
		return valueUnorderedEqual(left.node.entries, right.node.entries, func(entry, other ResolvedEntry) bool {
			return resolvedKeyEqual(entry.Key, other.Key) && resolvedEnumEqual(entry.Value, other.Value)
		})
	case ValueKindUnion:
		return left.node.branch == right.node.branch && left.OccurrenceID() == right.OccurrenceID() && resolvedEnumEqual(left.node.payload, right.node.payload)
	}
	return true
}

func resolvedFieldEqual(field, other ResolvedField) bool {
	if field.Member != other.Member || field.Name != other.Name {
		return false
	}
	if field.Member == (ValueIdentity{}) && field.Value.node != nil && other.Value.node != nil {
		return exampleValuesEqual(field.Value.node.raw, other.Value.node.raw)
	}
	return resolvedEnumEqual(field.Value, other.Value)
}

func resolvedKeyEqual(left, right ResolvedValue) bool {
	leftName, leftValid := jsonkey.Name(reflect.ValueOf(left.node.scalar))
	rightName, rightValid := jsonkey.Name(reflect.ValueOf(right.node.scalar))
	return leftValid && rightValid && leftName == rightName
}

// Callers establish unique semantic identities or key spellings first.
func valueUnorderedEqual[T any](left, right []T, equal func(T, T) bool) bool {
	if len(left) != len(right) {
		return false
	}
	for _, entry := range left {
		if !slices.ContainsFunc(right, func(other T) bool {
			return equal(entry, other)
		}) {
			return false
		}
	}
	return true
}
