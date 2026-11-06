package expr

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// additionalMembersPresent shares physical-member presence policy between the
// schema and carrier domains without sharing either domain's value predicates.
func additionalMembersPresent(plan *valuePlanNode, wire jsontext.Value) bool {
	if len(plan.additionalRequired) == 0 && len(plan.additionalForbidden) == 0 || wire.Kind() == 'n' && IsNullable(plan.attribute) {
		return true
	}
	var entries map[string]jsontext.Value
	if wire.Kind() != '{' || json.Unmarshal(wire, &entries) != nil {
		return false
	}
	for _, name := range plan.additionalRequired {
		if _, present := entries[name]; !present {
			return false
		}
	}
	for _, name := range plan.additionalForbidden {
		if _, present := entries[name]; present {
			return false
		}
	}
	return true
}
