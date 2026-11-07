package openapiimport

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// evaluateOneOfClosureSchema is an oracle for the bounded profile vocabulary,
// not a general JSON Schema validator. It reads source/export JSON directly,
// counts original alternatives, and reports annotations only from passing
// alternatives. The independently written value tables prevent two malformed
// exports from establishing their own expected answer.
func evaluateOneOfClosureSchema(t *testing.T, document, schema map[string]any, value any) (bool, map[string]bool) {
	t.Helper()
	allowed := map[string]bool{"$ref": true, "type": true, "properties": true, "required": true, "additionalProperties": true, "oneOf": true, "anyOf": true, "allOf": true, "unevaluatedProperties": true, "enum": true, "const": true, "title": true, "description": true, "example": true, "examples": true, "format": true, "default": true, "items": true, "minItems": true, "readOnly": true, "writeOnly": true, "deprecated": true}
	for keyword := range schema {
		require.True(t, allowed[keyword] || strings.HasPrefix(keyword, "x-"), "closure oracle does not evaluate keyword %q", keyword)
	}
	evaluated := make(map[string]bool)
	merge := func(members map[string]bool) {
		for name := range members {
			evaluated[name] = true
		}
	}
	if schema["$ref"] != nil {
		valid, members := evaluateOneOfClosureSchema(t, document, oneOfSpecReferenceTarget(t, document, schema["$ref"].(string)), value)
		if !valid {
			return false, nil
		}
		merge(members)
	}
	if typ, ok := schema["type"].(string); ok && !oneOfClosureTypeMatches(typ, value) {
		return false, nil
	}
	if array, ok := value.([]any); ok {
		if minimum, present := schema["minItems"].(float64); present && float64(len(array)) < minimum {
			return false, nil
		}
		if items, present := schema["items"].(map[string]any); present {
			for _, item := range array {
				valid, _ := evaluateOneOfClosureSchema(t, document, items, item)
				if !valid {
					return false, nil
				}
			}
		}
	}
	if choices, ok := schema["enum"].([]any); ok {
		matches := false
		for _, choice := range choices {
			matches = matches || reflect.DeepEqual(choice, value)
		}
		if !matches {
			return false, nil
		}
	}
	if constant, exists := schema["const"]; exists && !reflect.DeepEqual(constant, value) {
		return false, nil
	}
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		branches, exists := schema[keyword].([]any)
		if !exists {
			continue
		}
		matches := 0
		selected := make(map[string]bool)
		for _, branch := range branches {
			valid, members := evaluateOneOfClosureSchema(t, document, branch.(map[string]any), value)
			if valid {
				matches++
				for name := range members {
					selected[name] = true
				}
			}
		}
		if keyword == "oneOf" && matches != 1 || keyword == "anyOf" && matches == 0 || keyword == "allOf" && matches != len(branches) {
			return false, nil
		}
		merge(selected)
	}
	object, isObject := value.(map[string]any)
	if !isObject {
		return true, evaluated
	}
	properties, _ := schema["properties"].(map[string]any)
	if required, ok := schema["required"].([]any); ok {
		for _, name := range required {
			if _, present := object[name.(string)]; !present {
				return false, nil
			}
		}
	}
	for name, member := range object {
		if property, declared := properties[name]; declared {
			valid, _ := evaluateOneOfClosureSchema(t, document, property.(map[string]any), member)
			if !valid {
				return false, nil
			}
			evaluated[name] = true
			continue
		}
		additional, present := schema["additionalProperties"]
		if !present {
			continue
		}
		switch rule := additional.(type) {
		case bool:
			if !rule {
				return false, nil
			}
		case map[string]any:
			valid, _ := evaluateOneOfClosureSchema(t, document, rule, member)
			if !valid {
				return false, nil
			}
		default:
			t.Fatalf("unsupported additionalProperties oracle input: %T", additional)
		}
		evaluated[name] = true
	}
	if closure, present := schema["unevaluatedProperties"]; present {
		require.Equal(t, false, closure, "profile closure is explicitly false")
		for name := range object {
			if !evaluated[name] {
				return false, nil
			}
		}
	}
	return true, evaluated
}

func oneOfClosureTypeMatches(typ string, value any) bool {
	switch typ {
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		number, ok := value.(float64)
		return ok && math.Trunc(number) == number
	case "number":
		_, ok := value.(float64)
		return ok
	case "null":
		return value == nil
	default:
		return false
	}
}
