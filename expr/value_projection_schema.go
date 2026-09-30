package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	loom "github.com/CaliLuke/loom/pkg"
)

var valueBase64Language = regexp.MustCompile(`\A(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?\z`)

// The body and enum gate are deliberately separate: returning from a body
// branch must never bypass whole-array/object/map/union enum constraints.
func schemaJSON(plan *valuePlanNode, wire jsontext.Value) bool {
	if !schemaJSONBody(plan, wire) || !schemaLocalRules(plan, wire) {
		return false
	}
	if !plan.hasEnum {
		return true
	}
	for _, values := range plan.enumClauses {
		matched := false
		for _, member := range values {
			observed, failure := observeJSON(plan, ValueRoleEnum, member)
			if failure != nil {
				continue
			}
			expected, failure := constructJSON(plan, observed)
			if failure == nil && loom.JSONValueEqual(wire, expected) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func schemaJSONBody(plan *valuePlanNode, wire jsontext.Value) bool {
	if !wire.IsValid() {
		return false
	}
	if wire.Kind() == 'n' && plan.nullable {
		return true
	}
	if plan.alias != nil {
		return schemaJSON(plan.alias, wire)
	}
	switch plan.kind {
	case AnyKind:
		return true
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
			return false
		}
	case BooleanKind:
		return wire.Kind() == 't' || wire.Kind() == 'f'
	default:
		if wire.Kind() != '0' {
			return false
		}
		number, valid := new(big.Rat).SetString(string(wire))
		if !valid || (plan.kind >= IntKind && plan.kind <= UInt64Kind && !number.IsInt()) {
			return false
		}
	}
	return true
}

func schemaJSONObject(plan *valuePlanNode, wire jsontext.Value) bool {
	var entries map[string]jsontext.Value
	if wire.Kind() != '{' || json.Unmarshal(wire, &entries) != nil {
		return false
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
				return false
			}
			continue
		}
		if !schemaJSON(member.node, child) {
			return false
		}
	}
	if !plan.schemaUnknown {
		for name := range entries {
			if !known[name] {
				return false
			}
		}
	}
	return true
}

func schemaJSONUnion(plan *valuePlanNode, wire jsontext.Value) bool {
	if plan.untagged {
		matches := 0
		for _, branch := range plan.branches {
			if schemaJSON(branch.node, wire) {
				matches++
			}
		}
		return matches == 1
	}
	var envelope map[string]jsontext.Value
	if wire.Kind() != '{' || json.Unmarshal(wire, &envelope) != nil {
		return false
	}
	var tag string
	if json.Unmarshal(envelope[plan.typeKey], &tag) != nil {
		return false
	}
	if !plan.schemaUnknown && len(envelope) != 2 {
		return false
	}
	for _, branch := range plan.branches {
		if branch.tag == tag {
			return schemaJSON(branch.node, envelope[plan.valueKey])
		}
	}
	return false
}

func schemaLocalRules(plan *valuePlanNode, wire jsontext.Value) bool {
	rules := plan.validation
	if wire.Kind() == 'n' {
		return true
	}
	effective := plan
	for effective.alias != nil {
		effective = effective.alias
	}
	if effective.kind == BytesKind {
		var text string
		if json.Unmarshal(wire, &text) != nil || !valueBase64Language.MatchString(text) {
			return false
		}
		padding := len(text) - len(strings.TrimRight(text, "="))
		return valueLengthAllowed(rules, len(text)/4*3-padding)
	}
	if effective.kind == ArrayKind {
		var items []jsontext.Value
		return json.Unmarshal(wire, &items) == nil && valueLengthAllowed(rules, len(items))
	}
	if effective.kind == MapKind {
		var entries map[string]jsontext.Value
		return json.Unmarshal(wire, &entries) == nil && valueLengthAllowed(rules, len(entries))
	}
	if rules == nil {
		return true
	}
	if wire.Kind() == '0' {
		number, valid := new(big.Rat).SetString(string(wire))
		return valid && schemaNumberRules(rules, number)
	}
	if wire.Kind() == '"' {
		var text string
		if json.Unmarshal(wire, &text) != nil {
			return false
		}
		return valueValidationRules(rules, ResolvedValue{node: &resolvedValueNode{presence: ValuePresent, kind: ValueKindScalar, scalar: text}})
	}
	return true
}

func schemaNumberRules(rules *ValidationExpr, number *big.Rat) bool {
	for _, bound := range []struct {
		value     *float64
		lower     bool
		exclusive bool
	}{
		{rules.Minimum, true, false}, {rules.ExclusiveMinimum, true, true},
		{rules.Maximum, false, false}, {rules.ExclusiveMaximum, false, true},
	} {
		if bound.value == nil {
			continue
		}
		limit, valid := new(big.Rat).SetString(strconv.FormatFloat(*bound.value, 'g', -1, 64))
		if !valid {
			return false
		}
		comparison := number.Cmp(limit)
		if bound.lower && (comparison < 0 || comparison == 0 && bound.exclusive) || !bound.lower && (comparison > 0 || comparison == 0 && bound.exclusive) {
			return false
		}
	}
	return true
}

func schemaJSONArray(plan *valuePlanNode, wire jsontext.Value) bool {
	var items []jsontext.Value
	if wire.Kind() != '[' || json.Unmarshal(wire, &items) != nil {
		return false
	}
	for _, child := range items {
		if plan.nonNullableElements && child.Kind() == 'n' || !schemaJSON(plan.element, child) {
			return false
		}
	}
	return true
}

func schemaJSONMap(plan *valuePlanNode, wire jsontext.Value) bool {
	var entries map[string]jsontext.Value
	if wire.Kind() != '{' || json.Unmarshal(wire, &entries) != nil {
		return false
	}
	for _, child := range entries {
		// Current map schemas constrain values, not object-member spelling.
		// Runtime key parsing is an independent decoder obligation.
		if !schemaJSON(plan.element, child) {
			return false
		}
	}
	return true
}
