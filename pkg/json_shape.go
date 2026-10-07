package loom

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"unicode/utf8"
)

type (
	// JSONShape is a generated, normalized JSON matching description. It has no
	// DSL references, schema resolution, coercions, defaults, or branch selection.
	// Callers must finish building the graph before sharing it and keep it immutable.
	JSONShape struct {
		// Kind is any, boolean, string, bytes, integer, number, object, array, or alias.
		Kind string
		// Nullable admits explicit JSON null at this occurrence.
		Nullable bool
		// Child is the array element or underlying alias shape.
		Child *JSONShape
		// Fields are the visible object members in declaration order.
		Fields []JSONShapeField
		// Closed rejects object members absent from Fields.
		Closed bool
		// NonNullableElements rejects null array elements independently of Child.
		NonNullableElements bool
		// Rules holds the occurrence's effective, conjoined predicates.
		Rules JSONShapeRules
	}
	// JSONShapeField records exact member spelling and required membership.
	JSONShapeField struct {
		// Name is the case-sensitive JSON member name.
		Name string
		// Required checks membership before any typed defaulting.
		Required bool
		// Shape is the member's matching description.
		Shape *JSONShape
	}
	// JSONShapeRules contains already-lowered predicates, not authored DSL policy.
	JSONShapeRules struct {
		// Minimum and Maximum are decimal inclusive bounds; empty means absent.
		Minimum, Maximum string
		// ExclusiveMinimum and ExclusiveMaximum are decimal exclusive bounds.
		ExclusiveMinimum, ExclusiveMaximum string
		// MinLength and MaxLength bound strings, decoded bytes, or arrays.
		MinLength, MaxLength *int
		// Patterns are conjoined regular expressions.
		Patterns []string
		// Formats are conjoined Loom string formats.
		Formats []Format
		// Enums conjoins clauses whose members are already projected JSON values.
		Enums [][]jsontext.Value
	}
)

var jsonShapeBase64 = regexp.MustCompile(`\A(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?\z`)

// Match checks one complete JSON value without invoking application codecs.
// False is an ordinary contract mismatch; errors indicate invalid descriptions
// or values beyond the exact predicate evaluator's capacity.
func (s *JSONShape) Match(wire jsontext.Value) (bool, error) {
	if err := s.validate(make(map[*JSONShape]bool)); err != nil {
		return false, err
	}
	if !wire.IsValid() {
		return false, nil
	}
	return s.match(wire, 0)
}

func (s *JSONShape) validate(seen map[*JSONShape]bool) error {
	if s == nil {
		return fmt.Errorf("missing JSON matching description")
	}
	if seen[s] {
		return nil
	}
	seen[s] = true
	switch s.Kind {
	case "any", "boolean", "string", "bytes", "integer", "number":
	case "array", "alias":
		if s.Kind == "alias" {
			aliases := make(map[*JSONShape]bool)
			for node := s; node != nil && node.Kind == "alias"; node = node.Child {
				if aliases[node] {
					return fmt.Errorf("cyclic JSON matching alias")
				}
				aliases[node] = true
			}
		}
		if err := s.Child.validate(seen); err != nil {
			return err
		}
	case "object":
		names := make(map[string]bool)
		for _, field := range s.Fields {
			if names[field.Name] {
				return fmt.Errorf("duplicate JSON matching member %q", field.Name)
			}
			names[field.Name] = true
			if err := field.Shape.validate(seen); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported JSON matching kind %q", s.Kind)
	}
	return s.Rules.validate()
}

func (r JSONShapeRules) validate() error {
	for _, bound := range []string{r.Minimum, r.Maximum, r.ExclusiveMinimum, r.ExclusiveMaximum} {
		if bound != "" {
			if _, valid := new(big.Rat).SetString(bound); !valid {
				return fmt.Errorf("invalid JSON numeric bound %q", bound)
			}
		}
	}
	for _, pattern := range r.Patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return err
		}
	}
	for _, format := range r.Formats {
		switch format {
		case FormatDate, FormatDateTime, FormatUUID, FormatEmail, FormatHostname,
			FormatIPv4, FormatIPv6, FormatIP, FormatURI, FormatURIReference,
			FormatMAC, FormatCIDR, FormatRegexp, FormatJSON, FormatRFC1123:
		default:
			return fmt.Errorf("unsupported JSON matching format %q", format)
		}
	}
	for _, clause := range r.Enums {
		for _, member := range clause {
			if !member.IsValid() {
				return fmt.Errorf("invalid JSON matching enum")
			}
		}
	}
	return nil
}

func (s *JSONShape) match(wire jsontext.Value, depth int) (bool, error) {
	if s == nil || depth > 10000 {
		return false, fmt.Errorf("invalid or cyclic JSON matching description")
	}
	if wire.Kind() == 'n' && s.Nullable {
		return s.rules(wire, -1)
	}
	length := -1
	switch s.Kind {
	case "alias":
		matched, err := s.Child.match(wire, depth+1)
		if err != nil || !matched {
			return matched, err
		}
		return s.rules(wire, s.length(wire))
	case "any":
	case "boolean":
		if wire.Kind() != 't' && wire.Kind() != 'f' {
			return false, nil
		}
	case "string", "bytes":
		if wire.Kind() != '"' {
			return false, nil
		}
		length = s.length(wire)
		if length < 0 {
			return false, nil
		}
	case "integer", "number":
		if wire.Kind() != '0' {
			return false, nil
		}
		number, valid := new(big.Rat).SetString(string(wire))
		if !valid {
			return false, fmt.Errorf("JSON number exceeds exact matching capacity")
		}
		if s.Kind == "integer" && !number.IsInt() {
			return false, nil
		}
	case "array":
		return s.array(wire, depth)
	case "object":
		matched, err := s.object(wire, depth)
		if err != nil || !matched {
			return matched, err
		}
	default:
		return false, fmt.Errorf("unsupported JSON matching kind %q", s.Kind)
	}
	return s.rules(wire, length)
}

func (s *JSONShape) object(wire jsontext.Value, depth int) (bool, error) {
	if wire.Kind() != '{' {
		return false, nil
	}
	var members map[string]jsontext.Value
	if err := json.Unmarshal(wire, &members); err != nil {
		return false, err
	}
	for _, field := range s.Fields {
		child, present := members[field.Name]
		if !present {
			if field.Required {
				return false, nil
			}
			continue
		}
		matched, err := field.Shape.match(child, depth+1)
		if err != nil || !matched {
			return matched, err
		}
		delete(members, field.Name)
	}
	return !s.Closed || len(members) == 0, nil
}

func (s *JSONShape) length(wire jsontext.Value) int {
	underlying := s
	for underlying != nil && underlying.Kind == "alias" {
		underlying = underlying.Child
	}
	if underlying == nil {
		return -1
	}
	if wire.Kind() == '[' {
		var values []jsontext.Value
		if json.Unmarshal(wire, &values) == nil {
			return len(values)
		}
	}
	if wire.Kind() == '"' {
		var value string
		if json.Unmarshal(wire, &value) != nil {
			return -1
		}
		if underlying.Kind == "bytes" {
			if !jsonShapeBase64.MatchString(value) {
				return -1
			}
			return len(value)/4*3 - (len(value) - len(strings.TrimRight(value, "=")))
		}
		return utf8.RuneCountInString(value)
	}
	return -1
}

func (s *JSONShape) rules(wire jsontext.Value, length int) (bool, error) {
	r := s.Rules
	if wire.Kind() != 'n' {
		if length >= 0 && (r.MinLength != nil && length < *r.MinLength || r.MaxLength != nil && length > *r.MaxLength) {
			return false, nil
		}
		if wire.Kind() == '0' {
			matched, err := r.number(wire)
			if err != nil || !matched {
				return matched, err
			}
		}
		if wire.Kind() == '"' {
			var value string
			if err := json.Unmarshal(wire, &value); err != nil {
				return false, err
			}
			for _, pattern := range r.Patterns {
				re, err := regexp.Compile(pattern)
				if err != nil {
					return false, err
				}
				if !re.MatchString(value) {
					return false, nil
				}
			}
			if !r.matchesFormats(value) {
				return false, nil
			}
		}
	}
	for _, clause := range r.Enums {
		matched := false
		for _, expected := range clause {
			matched = matched || JSONValueEqual(wire, expected)
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func (r JSONShapeRules) number(wire jsontext.Value) (bool, error) {
	number, valid := new(big.Rat).SetString(string(wire))
	if !valid {
		return false, fmt.Errorf("JSON number exceeds exact matching capacity")
	}
	for _, bound := range []struct {
		value            string
		lower, exclusive bool
	}{
		{r.Minimum, true, false}, {r.Maximum, false, false},
		{r.ExclusiveMinimum, true, true}, {r.ExclusiveMaximum, false, true},
	} {
		if bound.value == "" {
			continue
		}
		limit, valid := new(big.Rat).SetString(bound.value)
		if !valid {
			return false, fmt.Errorf("invalid JSON numeric bound %q", bound.value)
		}
		comparison := number.Cmp(limit)
		if bound.lower && (comparison < 0 || comparison == 0 && bound.exclusive) || !bound.lower && (comparison > 0 || comparison == 0 && bound.exclusive) {
			return false, nil
		}
	}
	return true, nil
}

func (s *JSONShape) array(wire jsontext.Value, depth int) (bool, error) {
	if wire.Kind() != '[' {
		return false, nil
	}
	var items []jsontext.Value
	if err := json.Unmarshal(wire, &items); err != nil {
		return false, err
	}
	length := len(items)
	for _, item := range items {
		if s.NonNullableElements && item.Kind() == 'n' {
			return false, nil
		}
		matched, err := s.Child.match(item, depth+1)
		if err != nil || !matched {
			return matched, err
		}
	}
	return s.rules(wire, length)
}

func (r JSONShapeRules) matchesFormats(value string) bool {
	for _, format := range r.Formats {
		if ValidateFormat("value", value, format) != nil {
			return false
		}
	}
	return true
}
