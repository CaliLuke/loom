package http

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"maps"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/CaliLuke/loom/internal/httpencoding"
	"github.com/CaliLuke/loom/internal/unionmatch"
)

type (
	// FormKeySet describes one field's complete captured physical key ownership.
	// Union fields include every alternative, independently of selection.
	FormKeySet struct {
		// Literals contains exact decoded form keys.
		Literals []string
		// Prefixes contains deepObject names, each claiming name[key].
		Prefixes []string
		// Residual claims keys not explicitly owned by another field.
		Residual bool
	}
	// FormProjection keeps logical schema values separate from physical keys.
	// Construct it through FormObjectJSON or FormMapJSON and keep it immutable
	// while evaluating branch membership and closure annotations.
	FormProjection struct {
		// Value contains declared logical properties and their exact JSON values.
		// Unknown physical fields are never inserted into this JSON object.
		Value jsontext.Value
		// Members contains the distinct decoded physical keys in the body.
		Members []string
		// Unclaimed contains physical keys outside all captured field claims.
		Unclaimed []string
		// FieldClaims maps logical properties to their present physical keys.
		// A union field retains keys owned by every alternative.
		FieldClaims map[string][]string
	}
	// FormField projects one declared form property into the logical root object.
	// Construct fresh views for each input and keep their raw carrier immutable.
	FormField struct {
		// Name is the effective form-root property name, not a nested JSON tag.
		Name string
		// Keys retains the full field namespace, including all union alternatives.
		Keys FormKeySet
		// View returns exact JSON and true for a present value, nil and true for
		// absence, or false for a codec/schema mismatch. Explicit null is nonnil
		// JSON. Errors identify invalid evaluator configuration or evaluator failure.
		View func() (jsontext.Value, bool, error)
	}
	// FormValueBranch supplies one schema-only projection of a form field.
	// Views may recursively call FormTextUnionValue; typed conversion is separate.
	FormValueBranch struct {
		// View has the same presence and mismatch contract as FormField.View.
		View func() (jsontext.Value, bool, error)
		// Member checks the exact projected JSON without Go carrier restrictions.
		Member func(jsontext.Value) (bool, error)
		// Evaluate reports evaluated object members after this schema matches.
		// Closed unions require it when the selected value is an object.
		Evaluate func(jsontext.Value) ([]string, error)
	}
)

// FormObjectJSON projects a present form body into one branch's logical object.
// Empty input produces {}. The media owner decides optional body absence before
// calling it. Logical values and physical ownership stay separate: an unknown
// key may equal an exploded property's logical name without replacing its value.
// Static descriptors are validated before any input-dependent callback runs.
// False denotes an input mismatch; errors denote invalid configuration or
// evaluator failure. Unknown values are ignored; invalid UTF-8 names reject.
// On a field mismatch, Value is nil but complete physical ownership is retained
// so an independent typed decoder can apply the same branch closure policy.
func FormObjectJSON(pairs RawPairs, fields []FormField) (FormProjection, bool, error) {
	if err := validateFormFields(fields); err != nil {
		return FormProjection{}, false, err
	}
	projection, valid := formPhysicalProjection(pairs, fields)
	if !valid {
		return FormProjection{}, false, nil
	}
	members := make(map[string]jsontext.Value, len(fields))
	for _, field := range fields {
		raw, matched, err := field.View()
		if err != nil || !matched {
			return projection, false, err
		}
		if raw == nil {
			continue
		}
		if !raw.IsValid() {
			return projection, false, nil
		}
		members[field.Name] = raw
	}
	raw, matched, err := formMembersJSON(members)
	projection.Value = raw
	return projection, matched, err
}

// FormJSONValue projects named JSON content without typed conversion.
// FieldValue uses the first occurrence; FieldItems combines every occurrence
// into an exact JSON array. Absence returns nil and true. Invalid JSON returns
// false. Null and other value predicates belong to the caller's schema.
func FormJSONValue(pairs RawPairs, name string, application FieldApplication) (jsontext.Value, bool, error) {
	if err := validateFieldApplication(application); err != nil {
		return nil, false, err
	}
	var values []string
	for _, pair := range pairs.pairs {
		if pair.name != name {
			continue
		}
		value, err := url.QueryUnescape(pair.value)
		if err != nil {
			return nil, false, nil //nolint:nilerr // Malformed wire values are schema mismatches, not matcher failures.
		}
		values = append(values, value)
		if application == FieldValue {
			break
		}
	}
	if len(values) == 0 {
		return nil, true, nil
	}
	raw, err := jsonFieldValue(values, application)
	if err != nil || !raw.IsValid() {
		return nil, false, nil //nolint:nilerr // Malformed wire values are schema mismatches, not matcher failures.
	}
	return raw, true, nil
}

// FormTextValue projects a text field through its raw codec view and exact
// ParamBranchShape. Absent fields return nil, true, nil. Codec and scalar-domain
// mismatches return false. It never invokes Member, Evaluate, or typed Assign;
// schema validation belongs to the containing object or FormTextUnionValue.
func FormTextValue(branch TextBranch) (jsontext.Value, bool, error) {
	if branch.View == nil {
		return nil, false, fmt.Errorf("form text value requires a view")
	}
	if err := validateParamBranchShape(branch.Shape); err != nil {
		return nil, false, err
	}
	value, err := branch.View()
	if err != nil {
		_, err = textBranchError(err)
		return nil, false, err
	}
	raw, matched, err := ParamValueJSON(value, branch.Shape)
	if err != nil {
		return nil, false, err
	}
	if !value.Present {
		return nil, true, nil
	}
	return raw, matched, nil
}

// FormTextUnionValue selects exactly one schema-only field projection. Every
// view runs once per call. If every view is absent the field is omitted; invalid
// present input, zero matches, ambiguity, and closure failure return false.
// Nested views preserve their own exact-one accounting without flattening.
// Typed decoding is deliberately excluded and remains a TextUnionBranch concern.
func FormTextUnionValue(name string, branches []FormValueBranch, closed bool) (jsontext.Value, bool, error) {
	if len(branches) < 2 {
		return nil, false, fmt.Errorf("%s: form union requires at least two branches", name)
	}
	for index, branch := range branches {
		if branch.View == nil || branch.Member == nil {
			return nil, false, fmt.Errorf("%s: form union branch %d requires a view and membership evaluator", name, index)
		}
	}
	values := make([]jsontext.Value, len(branches))
	candidates := make([]unionmatch.Candidate, len(branches))
	absent := true
	for index, branch := range branches {
		raw, matched, err := branch.View()
		if err != nil {
			return nil, false, fmt.Errorf("%s view branch %d: %w", name, index, err)
		}
		if !matched || raw != nil {
			absent = false
		}
		if matched && raw != nil && raw.IsValid() {
			values[index] = raw
		}
		candidates[index].Schema = func() (bool, error) {
			if values[index] == nil {
				return false, nil
			}
			return branch.Member(values[index])
		}
	}
	if absent {
		return nil, true, nil
	}
	selected, err := unionmatch.SelectSchema(name, candidates)
	if err != nil {
		_, err = textUnionCandidateResult(err)
		return nil, false, err
	}
	raw := values[selected]
	if closed && raw.Kind() == '{' {
		if branches[selected].Evaluate == nil {
			return nil, false, fmt.Errorf("%s: selected object branch requires an annotation evaluator", name)
		}
		evaluated, err := branches[selected].Evaluate(raw)
		if err != nil {
			return nil, false, err
		}
		var members map[string]jsontext.Value
		if err := json.Unmarshal(raw, &members); err != nil {
			return nil, false, err
		}
		names := make([]string, 0, len(members))
		for name := range members {
			names = append(names, name)
		}
		if err := unionmatch.CheckEvaluated(name, names, evaluated); err != nil {
			_, err = textUnionCandidateResult(err)
			return nil, false, err
		}
	}
	return raw, true, nil
}

// FormMapJSON projects a present form body into a map using its captured value
// codec. value runs once per distinct decoded raw key, including the empty key;
// its captured RawPairs retain all repetitions for the codec to interpret.
// It may project text, JSON, collections or nested unions, but a key present in
// the body must produce a nonnil value. Empty bodies project {}. The callback
// uses FormField.View's mismatch/error contract. No typed Go conversion occurs.
// Physical members and claims are captured before callbacks and retained on a
// value mismatch, with nil Value, for independent typed-domain evaluation.
func FormMapJSON(pairs RawPairs, value func(string) (jsontext.Value, bool, error)) (FormProjection, bool, error) {
	if value == nil {
		return FormProjection{}, false, fmt.Errorf("form map requires a value projection")
	}
	projection := FormProjection{FieldClaims: make(map[string][]string)}
	for _, pair := range pairs.pairs {
		if !utf8.ValidString(pair.name) {
			return FormProjection{}, false, nil
		}
		projection.FieldClaims[pair.name] = []string{pair.name}
	}
	projection.Members = slices.Sorted(maps.Keys(projection.FieldClaims))
	members := make(map[string]jsontext.Value, len(projection.Members))
	for _, name := range projection.Members {
		raw, matched, err := value(name)
		if err != nil || !matched {
			return projection, false, err
		}
		if raw == nil || !raw.IsValid() {
			return projection, false, nil
		}
		members[name] = raw
	}
	raw, matched, err := formMembersJSON(members)
	projection.Value = raw
	return projection, matched, err
}

// Schema checks logical membership and the branch's captured object closure.
// It invokes member before input-dependent closure rejection so evaluator
// configuration failures cannot be hidden by an unknown physical key. Outer
// union closure belongs to MatchUntaggedText after original branch counting.
func (p FormProjection) Schema(member func(jsontext.Value) (bool, error), closed bool) (bool, error) {
	if member == nil {
		return false, fmt.Errorf("form projection requires a membership evaluator")
	}
	matched, err := member(p.Value)
	if err != nil || !matched {
		return false, err
	}
	return !closed || len(p.Unclaimed) == 0, nil
}

// Annotations maps evaluated logical properties to physical form key claims.
// evaluateUnknown is the captured explicit-open policy (additionalProperties:
// true), not the implicit openness default. Only explicit open schemas evaluate
// unclaimed fields. Both returned sets contain physical decoded member names.
func (p FormProjection) Annotations(evaluate func(jsontext.Value) ([]string, error), evaluateUnknown bool) (TextUnionAnnotations, error) {
	if evaluate == nil {
		return TextUnionAnnotations{}, fmt.Errorf("form projection requires an annotation evaluator")
	}
	logical, err := evaluate(p.Value)
	if err != nil {
		return TextUnionAnnotations{}, err
	}
	evaluated := make(map[string]bool)
	for _, name := range logical {
		keys, exists := p.FieldClaims[name]
		if !exists {
			return TextUnionAnnotations{}, fmt.Errorf("evaluated form property %q has no captured key claim", name)
		}
		for _, key := range keys {
			evaluated[key] = true
		}
	}
	if evaluateUnknown {
		for _, key := range p.Unclaimed {
			evaluated[key] = true
		}
	}
	return TextUnionAnnotations{Members: slices.Clone(p.Members), Evaluated: slices.Sorted(maps.Keys(evaluated))}, nil
}

func validateFormFields(fields []FormField) error {
	names := make(map[string]bool, len(fields))
	bindings := make([]httpencoding.Binding, 0, len(fields))
	for _, field := range fields {
		if field.View == nil || names[field.Name] {
			return fmt.Errorf("form property %q has a missing view or duplicate name", field.Name)
		}
		if !utf8.ValidString(field.Name) {
			return fmt.Errorf("form property name is not valid UTF-8")
		}
		for _, name := range field.Keys.Literals {
			if !utf8.ValidString(name) {
				return fmt.Errorf("form property %q has an invalid UTF-8 key claim", field.Name)
			}
		}
		for _, prefix := range field.Keys.Prefixes {
			if !utf8.ValidString(prefix) || strings.ContainsAny(prefix, "[]") {
				return fmt.Errorf("form property %q has an invalid deepObject prefix", field.Name)
			}
		}
		names[field.Name] = true
		bindings = append(bindings, httpencoding.Binding{Name: field.Name, Keys: httpencoding.KeySet{
			Literals: field.Keys.Literals, Prefixes: field.Keys.Prefixes, Residual: field.Keys.Residual,
		}})
	}
	return httpencoding.CheckNamespaces(httpencoding.FormField, bindings)
}

func formPhysicalProjection(pairs RawPairs, fields []FormField) (FormProjection, bool) {
	projection := FormProjection{FieldClaims: make(map[string][]string, len(fields))}
	residual := -1
	for index, field := range fields {
		projection.FieldClaims[field.Name] = nil
		if field.Keys.Residual {
			residual = index
		}
	}
	members := make(map[string]bool)
	for _, pair := range pairs.pairs {
		if !utf8.ValidString(pair.name) {
			return FormProjection{}, false
		}
		if members[pair.name] {
			continue
		}
		members[pair.name] = true
		owner := residual
		for index, field := range fields {
			if paramSiblingOwns(ParamNamespace{Literals: field.Keys.Literals, Prefixes: field.Keys.Prefixes}, pair.name) {
				owner = index
				break
			}
		}
		if owner < 0 {
			projection.Unclaimed = append(projection.Unclaimed, pair.name)
		} else {
			name := fields[owner].Name
			projection.FieldClaims[name] = append(projection.FieldClaims[name], pair.name)
		}
	}
	projection.Members = slices.Sorted(maps.Keys(members))
	slices.Sort(projection.Unclaimed)
	for _, keys := range projection.FieldClaims {
		slices.Sort(keys)
	}
	return projection, true
}

func formMembersJSON(members map[string]jsontext.Value) (jsontext.Value, bool, error) {
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	slices.Sort(names)
	out := []byte{'{'}
	for index, name := range names {
		key, err := json.Marshal(name)
		if err != nil {
			return nil, false, err
		}
		if index > 0 {
			out = append(out, ',')
		}
		out = append(out, key...)
		out = append(out, ':')
		out = append(out, members[name]...)
	}
	return append(out, '}'), true, nil
}
