package loom

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/loom/internal/exactnum"
)

type (
	// JSONShape is a generated, normalized JSON matching description. It has no
	// DSL references, schema resolution, coercions, defaults, or branch selection.
	// Callers must finish building the graph before sharing it and keep it immutable.
	JSONShape struct {
		// Kind is any, boolean, string, bytes, integer, number, object, array, map, union, or alias.
		Kind string
		// Nullable admits explicit JSON null at this occurrence.
		Nullable bool
		// Child is the array element, map value, or underlying alias shape.
		Child *JSONShape
		// Fields are the visible object members in declaration order.
		Fields []JSONShapeField
		// Branches are the nonempty alternatives of an untagged union; exactly one must match.
		Branches []*JSONShape
		// Closed rejects object members absent from Fields.
		Closed bool
		// UnevaluatedClosed rejects members not evaluated by the selected union
		// alternative, after unique matching. It applies to unions and their aliases.
		UnevaluatedClosed bool
		// Additional validates undeclared members. Nil means the schema did not
		// supply additionalProperties; an any shape represents explicit true.
		Additional *JSONShape
		// ForbiddenMembers are physical members removed by a transport projection.
		// An additional-member schema must never admit these reserved names.
		ForbiddenMembers []string
		// RequiredMembers requires exact object keys at this occurrence,
		// independently of whether its object shape is reached through an alias.
		RequiredMembers []string
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
		// MinLength and MaxLength bound strings, decoded bytes, arrays, or map entries.
		MinLength, MaxLength *int
		// Patterns are conjoined regular expressions.
		Patterns []string
		// Formats are conjoined Loom string formats.
		Formats []Format
		// Enums conjoins clauses whose members are already projected JSON values.
		Enums [][]jsontext.Value
	}
	// jsonShapeMatch retains annotations from the same successful traversal.
	jsonShapeMatch struct {
		matched   bool
		evaluated []string
	}
)

var jsonShapeBase64 = regexp.MustCompile(`\A(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?\z`)

// Match checks one complete JSON value without invoking application codecs.
// False is an ordinary contract mismatch; errors indicate invalid descriptions
// or failures of the exact predicate evaluator.
func (s *JSONShape) Match(wire jsontext.Value) (bool, error) {
	result, err := s.evaluate(wire)
	return result.matched, err
}

func (s *JSONShape) evaluate(wire jsontext.Value) (jsonShapeMatch, error) {
	if err := s.validate(make(map[*JSONShape]bool)); err != nil {
		return jsonShapeMatch{}, err
	}
	if !wire.IsValid() {
		return jsonShapeMatch{}, nil
	}
	return s.matchAnnotated(wire, 0)
}

func (s *JSONShape) validate(seen map[*JSONShape]bool) error {
	if s == nil {
		return fmt.Errorf("missing JSON matching description")
	}
	if seen[s] {
		return nil
	}
	seen[s] = true
	if s.UnevaluatedClosed {
		root := s
		aliases := make(map[*JSONShape]bool)
		for root != nil && root.Kind == "alias" && !aliases[root] {
			aliases[root] = true
			root = root.Child
		}
		if root == nil || root.Kind != "union" {
			return fmt.Errorf("unevaluated closure requires a JSON union or union alias")
		}
	}
	switch s.Kind {
	case "any", "boolean", "string", "bytes", "integer", "number":
	case "array", "map", "alias":
		if s.Kind == "alias" {
			if err := s.validateAlternatives(make(map[*JSONShape]bool)); err != nil {
				return err
			}
		}
		if err := s.Child.validate(seen); err != nil {
			return err
		}
	case "union":
		if len(s.Branches) == 0 {
			return fmt.Errorf("JSON matching union requires at least one branch")
		}
		if err := s.validateAlternatives(make(map[*JSONShape]bool)); err != nil {
			return err
		}
		for _, branch := range s.Branches {
			if err := branch.validate(seen); err != nil {
				return err
			}
		}
	case "object":
		if err := s.validateObject(seen); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported JSON matching kind %q", s.Kind)
	}
	return s.Rules.validate()
}

func (s *JSONShape) validateObject(seen map[*JSONShape]bool) error {
	if s.Additional != nil {
		if s.Closed {
			return fmt.Errorf("closed JSON object cannot supply an additional matching shape")
		}
		if err := s.Additional.validate(seen); err != nil {
			return err
		}
	}
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
	return nil
}

// validateAlternatives rejects cycles that revisit the same wire value. Object
// fields and collection values consume input and may contain recursive unions.
func (s *JSONShape) validateAlternatives(active map[*JSONShape]bool) error {
	if s == nil || s.Kind != "alias" && s.Kind != "union" {
		return nil
	}
	if active[s] {
		return fmt.Errorf("cyclic JSON matching alternatives")
	}
	active[s] = true
	defer delete(active, s)
	if s.Kind == "alias" {
		return s.Child.validateAlternatives(active)
	}
	for _, branch := range s.Branches {
		if err := branch.validateAlternatives(active); err != nil {
			return err
		}
	}
	return nil
}

func (r JSONShapeRules) validate() error {
	for _, bound := range []string{r.Minimum, r.Maximum, r.ExclusiveMinimum, r.ExclusiveMaximum} {
		if bound != "" {
			if _, err := exactnum.ParseJSON(bound); err != nil {
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
		case FormatBase64, FormatDate, FormatDateTime, FormatUUID, FormatEmail, FormatHostname,
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
	result, err := s.matchAnnotated(wire, depth)
	return result.matched, err
}

func (s *JSONShape) matchAnnotated(wire jsontext.Value, depth int) (jsonShapeMatch, error) {
	if s == nil || depth > 10000 {
		return jsonShapeMatch{}, fmt.Errorf("invalid or cyclic JSON matching description")
	}
	if wire.Kind() == 'n' && s.Nullable {
		return jsonShapeMatch{matched: true}, nil
	}
	if matched, err := s.requiredMembers(wire); err != nil || !matched {
		return jsonShapeMatch{}, err
	}
	length := -1
	result := jsonShapeMatch{matched: true}
	switch s.Kind {
	case "alias":
		child, err := s.Child.matchAnnotated(wire, depth+1)
		if err != nil || !child.matched {
			return jsonShapeMatch{}, err
		}
		return s.finishMatch(wire, s.length(wire), child)
	case "any":
	case "boolean":
		if wire.Kind() != 't' && wire.Kind() != 'f' {
			return jsonShapeMatch{}, nil
		}
	case "string", "bytes":
		if wire.Kind() != '"' {
			return jsonShapeMatch{}, nil
		}
		length = s.length(wire)
		if length < 0 {
			return jsonShapeMatch{}, nil
		}
	case "integer", "number":
		if matched, err := s.matchNumericKind(wire); err != nil || !matched {
			return jsonShapeMatch{}, err
		}
	case "array":
		matched, err := s.array(wire, depth)
		return jsonShapeMatch{matched: matched}, err
	case "map":
		return s.mapping(wire, depth)
	case "union":
		return s.union(wire, depth)
	case "object":
		var err error
		result, err = s.object(wire, depth)
		if err != nil || !result.matched {
			return jsonShapeMatch{}, err
		}
	default:
		return jsonShapeMatch{}, fmt.Errorf("unsupported JSON matching kind %q", s.Kind)
	}
	return s.finishMatch(wire, length, result)
}

func (s *JSONShape) matchNumericKind(wire jsontext.Value) (bool, error) {
	if wire.Kind() != '0' {
		return false, nil
	}
	number, err := exactnum.ParseJSON(string(wire))
	if err != nil {
		return false, err
	}
	return s.Kind != "integer" || number.IsInt(), nil
}

func (s *JSONShape) finishMatch(wire jsontext.Value, length int, result jsonShapeMatch) (jsonShapeMatch, error) {
	matched, err := s.rules(wire, length)
	if err != nil || !matched {
		return jsonShapeMatch{}, err
	}
	if matched, err = s.matchClosure(wire, result.evaluated); err != nil || !matched {
		return jsonShapeMatch{}, err
	}
	return result, nil
}

func (s *JSONShape) union(wire jsontext.Value, depth int) (jsonShapeMatch, error) {
	matches := 0
	var selected jsonShapeMatch
	for _, branch := range s.Branches {
		result, err := branch.matchAnnotated(wire, depth+1)
		if err != nil {
			return jsonShapeMatch{}, err
		}
		if result.matched {
			matches++
			selected = result
		}
	}
	if matches != 1 {
		return jsonShapeMatch{}, nil
	}
	return s.finishMatch(wire, -1, selected)
}

func (s *JSONShape) requiredMembers(wire jsontext.Value) (bool, error) {
	if len(s.RequiredMembers) == 0 {
		return true, nil
	}
	if wire.Kind() != '{' {
		return false, nil
	}
	var members map[string]jsontext.Value
	if err := json.Unmarshal(wire, &members); err != nil {
		return false, err
	}
	for _, name := range s.RequiredMembers {
		if _, present := members[name]; !present {
			return false, nil
		}
	}
	return true, nil
}

func (s *JSONShape) object(wire jsontext.Value, depth int) (jsonShapeMatch, error) {
	if wire.Kind() != '{' {
		return jsonShapeMatch{}, nil
	}
	var members map[string]jsontext.Value
	if err := json.Unmarshal(wire, &members); err != nil {
		return jsonShapeMatch{}, err
	}
	for _, name := range s.ForbiddenMembers {
		if _, present := members[name]; present {
			return jsonShapeMatch{}, nil
		}
	}
	result := jsonShapeMatch{matched: true}
	for _, field := range s.Fields {
		child, present := members[field.Name]
		if !present {
			if field.Required {
				return jsonShapeMatch{}, nil
			}
			continue
		}
		matched, err := field.Shape.match(child, depth+1)
		if err != nil || !matched {
			return jsonShapeMatch{}, err
		}
		result.evaluated = append(result.evaluated, field.Name)
		delete(members, field.Name)
	}
	if s.Additional != nil {
		for name, child := range members {
			matched, err := s.Additional.match(child, depth+1)
			if err != nil || !matched {
				return jsonShapeMatch{}, err
			}
			result.evaluated = append(result.evaluated, name)
		}
	}
	if s.Closed && len(members) != 0 {
		return jsonShapeMatch{}, nil
	}
	return result, nil
}

func (s *JSONShape) mapping(wire jsontext.Value, depth int) (jsonShapeMatch, error) {
	if wire.Kind() != '{' {
		return jsonShapeMatch{}, nil
	}
	var members map[string]jsontext.Value
	if err := json.Unmarshal(wire, &members); err != nil {
		return jsonShapeMatch{}, err
	}
	result := jsonShapeMatch{matched: true}
	for name, child := range members {
		matched, err := s.Child.match(child, depth+1)
		if err != nil || !matched {
			return jsonShapeMatch{}, err
		}
		result.evaluated = append(result.evaluated, name)
	}
	return s.finishMatch(wire, len(members), result)
}

func (s *JSONShape) length(wire jsontext.Value) int {
	underlying := s
	for underlying != nil && underlying.Kind == "alias" {
		underlying = underlying.Child
	}
	if underlying == nil {
		return -1
	}
	if underlying.Kind == "map" && wire.Kind() == '{' {
		var members map[string]jsontext.Value
		if json.Unmarshal(wire, &members) == nil {
			return len(members)
		}
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
	number, err := exactnum.ParseJSON(string(wire))
	if err != nil {
		return false, err
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
		limit, err := exactnum.ParseJSON(bound.value)
		if err != nil {
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
