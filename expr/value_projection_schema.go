package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"regexp"
	"strings"

	"github.com/CaliLuke/loom/internal/exactnum"
	loom "github.com/CaliLuke/loom/pkg"
)

var valueBase64Language = regexp.MustCompile(`\A(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?\z`)

// The body and enum gate are deliberately separate: returning from a body
// branch must never bypass whole-array/object/map/union enum constraints.
func schemaJSON(plan *valuePlanNode, wire jsontext.Value) (bool, error) {
	if plan == nil {
		return false, fmt.Errorf("missing JSON schema plan")
	}
	matched, err := schemaJSONBody(plan, wire)
	if err != nil || !matched {
		return matched, err
	}
	matched, err = schemaLocalRules(plan, wire)
	if err != nil || !matched {
		return matched, err
	}
	if !plan.hasEnum {
		return true, nil
	}
	for clauseIndex, values := range plan.enumClauses {
		projection := plan.enumProjection(clauseIndex)
		matched := false
		for _, member := range values {
			observed, failure := observeJSON(projection, ValueRoleEnum, member)
			if failure != nil {
				continue
			}
			expected, failure := constructJSON(projection, observed)
			if failure == nil && loom.JSONValueEqual(wire, expected) {
				matched = true
				break
			}
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func schemaJSONBody(plan *valuePlanNode, wire jsontext.Value) (bool, error) {
	if !wire.IsValid() {
		return false, nil
	}
	if wire.Kind() == 'n' && plan.nullable {
		return true, nil
	}
	if plan.alias != nil {
		return schemaJSON(plan.alias, wire)
	}
	switch plan.kind {
	case AnyKind:
		return true, nil
	case ArrayKind:
		return schemaJSONArray(plan, wire)
	case ObjectKind:
		return schemaJSONObject(plan, wire)
	case MapKind:
		return schemaJSONMap(plan, wire)
	case UnionKind:
		return schemaJSONUnion(plan, wire)
	case StringKind, BytesKind:
		if wire.Kind() != '"' {
			return false, nil
		}
	case BooleanKind:
		return wire.Kind() == 't' || wire.Kind() == 'f', nil
	case IntKind, Int32Kind, Int64Kind, UIntKind, UInt32Kind, UInt64Kind, Float32Kind, Float64Kind:
		if wire.Kind() != '0' {
			return false, nil
		}
		number, err := exactnum.ParseJSON(string(wire))
		if err != nil {
			return false, err
		}
		if plan.kind >= IntKind && plan.kind <= UInt64Kind && !number.IsInt() {
			return false, nil
		}
	default:
		return false, fmt.Errorf("unsupported JSON schema kind %d", plan.kind)
	}
	return true, nil
}

func schemaJSONObject(plan *valuePlanNode, wire jsontext.Value) (bool, error) {
	var entries map[string]jsontext.Value
	if wire.Kind() != '{' {
		return false, nil
	}
	if err := json.Unmarshal(wire, &entries); err != nil {
		return false, err
	}
	known := make(map[string]bool)
	for _, member := range plan.members {
		if !member.visible {
			continue
		}
		known[member.wire] = true
		child, present := entries[member.wire]
		if !present {
			if member.required {
				return false, nil
			}
			continue
		}
		matched, err := schemaJSON(member.node, child)
		if err != nil || !matched {
			return matched, err
		}
	}
	if !plan.schemaUnknown {
		for name := range entries {
			if !known[name] {
				return false, nil
			}
		}
	}
	return true, nil
}

func schemaJSONUnion(plan *valuePlanNode, wire jsontext.Value) (bool, error) {
	if plan.untagged {
		matches := 0
		for _, branch := range plan.branches {
			matched, err := schemaJSON(branch.node, wire)
			if err != nil {
				return false, err
			}
			if matched {
				matches++
			}
		}
		return matches == 1, nil
	}
	var envelope map[string]jsontext.Value
	if wire.Kind() != '{' {
		return false, nil
	}
	if err := json.Unmarshal(wire, &envelope); err != nil {
		return false, err
	}
	var tag string
	if envelope[plan.typeKey].Kind() != '"' {
		return false, nil
	}
	if err := json.Unmarshal(envelope[plan.typeKey], &tag); err != nil {
		return false, err
	}
	if !plan.schemaUnknown && len(envelope) != 2 {
		return false, nil
	}
	for _, branch := range plan.branches {
		if branch.tag == tag {
			return schemaJSON(branch.node, envelope[plan.valueKey])
		}
	}
	return false, nil
}

func schemaLocalRules(plan *valuePlanNode, wire jsontext.Value) (bool, error) {
	rules := plan.validation
	if wire.Kind() == 'n' {
		return true, nil
	}
	effective := plan
	for effective.alias != nil {
		effective = effective.alias
	}
	if effective.kind == BytesKind {
		var text string
		if err := json.Unmarshal(wire, &text); err != nil {
			return false, err
		}
		if !valueBase64Language.MatchString(text) {
			return false, nil
		}
		padding := len(text) - len(strings.TrimRight(text, "="))
		return valueLengthAllowed(rules, len(text)/4*3-padding), nil
	}
	if effective.kind == ArrayKind {
		var items []jsontext.Value
		if err := json.Unmarshal(wire, &items); err != nil {
			return false, err
		}
		return valueLengthAllowed(rules, len(items)), nil
	}
	if effective.kind == MapKind {
		var entries map[string]jsontext.Value
		if err := json.Unmarshal(wire, &entries); err != nil {
			return false, err
		}
		return valueLengthAllowed(rules, len(entries)), nil
	}
	if rules == nil {
		return true, nil
	}
	if wire.Kind() == '0' {
		number, err := exactnum.ParseJSON(string(wire))
		if err != nil {
			return false, err
		}
		return schemaNumberRules(rules, number), nil
	}
	if wire.Kind() == '"' {
		var text string
		if err := json.Unmarshal(wire, &text); err != nil {
			return false, err
		}
		return valueValidationRules(rules, ResolvedValue{node: &resolvedValueNode{presence: ValuePresent, kind: ValueKindScalar, scalar: text}}), nil
	}
	return true, nil
}

func schemaNumberRules(rules *ValidationExpr, number NumericBound) bool {
	for _, bound := range []struct {
		value            *NumericBound
		lower, exclusive bool
	}{
		{rules.Minimum, true, false}, {rules.ExclusiveMinimum, true, true},
		{rules.Maximum, false, false}, {rules.ExclusiveMaximum, false, true},
	} {
		if bound.value == nil {
			continue
		}
		order := number.Cmp(*bound.value)
		if bound.lower && (order < 0 || order == 0 && bound.exclusive) || !bound.lower && (order > 0 || order == 0 && bound.exclusive) {
			return false
		}
	}
	return true
}

func schemaJSONArray(plan *valuePlanNode, wire jsontext.Value) (bool, error) {
	var items []jsontext.Value
	if wire.Kind() != '[' {
		return false, nil
	}
	if err := json.Unmarshal(wire, &items); err != nil {
		return false, err
	}
	for _, child := range items {
		if plan.nonNullableElements && child.Kind() == 'n' {
			return false, nil
		}
		matched, err := schemaJSON(plan.element, child)
		if err != nil || !matched {
			return matched, err
		}
	}
	return true, nil
}

func schemaJSONMap(plan *valuePlanNode, wire jsontext.Value) (bool, error) {
	var entries map[string]jsontext.Value
	if wire.Kind() != '{' {
		return false, nil
	}
	if err := json.Unmarshal(wire, &entries); err != nil {
		return false, err
	}
	for _, child := range entries {
		// Current map schemas constrain values, not object-member spelling.
		// Runtime key parsing is an independent decoder obligation.
		matched, err := schemaJSON(plan.element, child)
		if err != nil || !matched {
			return matched, err
		}
	}
	return true, nil
}
