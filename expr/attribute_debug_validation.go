package expr

import (
	"cmp"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

// Debug dumps the attribute to STDOUT in a Loom developer friendly way.
func (a *AttributeExpr) Debug(prefix string) { a.debug(prefix, make(map[*AttributeExpr]int), 0) }

func (a *AttributeExpr) debug(prefix string, seen map[*AttributeExpr]int, indent int) {
	tab := "    "
	tabs := strings.Repeat(tab, indent)
	prefix = tabs + prefix
	if shouldStopDebugRecursion(a, seen, prefix) {
		return
	}
	debugAttributeHeader(a, prefix)
	tabs = debugAttributeShape(a, seen, indent, tab, tabs)
	debugResultTypeViews(a, tabs, tab)
	debugDefaultValue(a, tabs, tab)
	debugUserExamples(a, tabs, tab)
	debugAttributeMeta(a, tabs, tab)
	debugValidation(a, tabs, tab)
	debugNamedExprs("bases", a.Bases, tabs, tab)
	debugNamedExprs("references", a.References, tabs, tab)
}

func shouldStopDebugRecursion(a *AttributeExpr, seen map[*AttributeExpr]int, prefix string) bool {
	if !IsObject(a.Type) {
		return false
	}
	if c, ok := seen[a]; ok && c > 1 {
		fmt.Printf("%s: ...\n", prefix)
		return true
	}
	seen[a]++
	return false
}

func debugAttributeHeader(a *AttributeExpr, prefix string) {
	name := a.Type.Name()
	if desc := a.Description; desc != "" {
		fmt.Printf("%s: %s (%s) <%T>\n", prefix, name, desc, a.Type)
		return
	}
	fmt.Printf("%s: %s <%T>\n", prefix, name, a.Type)
}

func debugAttributeShape(a *AttributeExpr, seen map[*AttributeExpr]int, indent int, tab, tabs string) string {
	if ut, ok := a.Type.(UserType); ok {
		ut.Attribute().debug("att", seen, indent+1)
		return strings.Repeat(tab, indent+1)
	}
	switch {
	case IsObject(a.Type):
		for _, nat := range *AsObject(a.Type) {
			nat.Attribute.debug("- "+nat.Name, seen, indent+1)
		}
	case IsArray(a.Type):
		AsArray(a.Type).ElemType.debug("elem", seen, indent+1)
	case IsMap(a.Type):
		debugMapAttribute(AsMap(a.Type), seen, indent+1)
	case IsUnion(a.Type):
		for _, nat := range AsUnion(a.Type).Values {
			nat.Attribute.debug("* "+nat.Name, seen, indent+1)
		}
	}
	return tabs
}

func debugMapAttribute(m *Map, seen map[*AttributeExpr]int, indent int) {
	m.KeyType.debug("key", seen, indent)
	m.ElemType.debug("elem", seen, indent)
}

func debugResultTypeViews(a *AttributeExpr, tabs, tab string) {
	rt, ok := a.Type.(*ResultTypeExpr)
	if !ok {
		return
	}
	fmt.Printf("%s%sviews\n", tabs, tab)
	for _, v := range rt.Views {
		fmt.Printf("%s%s- %s: %v\n", tabs+tab, tab, v.Name, debugViewKeys(v))
	}
}

func debugViewKeys(v *ViewExpr) []string {
	nats := *AsObject(v.Type)
	keys := make([]string, len(nats))
	for i, n := range nats {
		keys[i] = n.Name
	}
	return keys
}

func debugDefaultValue(a *AttributeExpr, tabs, tab string) {
	if a.DefaultValue == nil {
		return
	}
	fmt.Printf("%s%sdefault\n", tabs, tab)
	fmt.Printf("%s%s%#v\n", tabs+tab, tab, a.DefaultValue)
}

func debugUserExamples(a *AttributeExpr, tabs, tab string) {
	if len(a.UserExamples) == 0 {
		return
	}
	fmt.Printf("%s%sexamples\n", tabs, tab)
	for _, ex := range a.UserExamples {
		fmt.Printf("%s%s- %s: %#v\n", tabs+tab, tab, ex.Summary, ex.Value)
	}
}

func debugAttributeMeta(a *AttributeExpr, tabs, tab string) {
	if len(a.Meta) == 0 {
		return
	}
	fmt.Printf("%s%smeta\n", tabs, tab)
	for k, v := range a.Meta {
		fmt.Printf("%s%s- %s: %s\n", tabs+tab, tab, k, strings.Join(v, ", "))
	}
}

func debugValidation(a *AttributeExpr, tabs, tab string) {
	if a.Validation == nil {
		return
	}
	a.Validation.Debug("", tabs+tab, tab)
}

func debugNamedExprs(label string, values []DataType, tabs, tab string) {
	if len(values) == 0 {
		return
	}
	fmt.Printf("%s%s%s\n", tabs, tab, label)
	for _, value := range values {
		fmt.Printf("%s%s- %s\n", tabs+tab, tab, value.Name())
	}
}

// validateEffectiveConstraints validates authored enums and defaults against
// the complete finalized named-type contract.
func (a *AttributeExpr) validateEffectiveConstraints(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if !hasConstraintValueAncestry(a) {
		return verr
	}
	if _, err := EffectiveConstraintsFor(a); err != nil {
		verr.Add(parent, "%s%s", ctx, err)
	}
	return verr
}

func hasConstraintValueAncestry(attribute *AttributeExpr) bool {
	seen := make(map[UserType]bool)
	for attribute != nil {
		if attribute.DefaultValue != nil || (attribute.Validation != nil &&
			(attribute.Validation.Values != nil || len(attribute.Validation.EnumClauses) > 0)) {
			return true
		}
		named, ok := attribute.Type.(UserType)
		if !ok || seen[named] {
			return false
		}
		seen[named] = true
		attribute = named.Attribute()
	}
	return false
}

// validateExamples makes sure that the attribute example values are compatible
// with the attribute type.
func (a *AttributeExpr) validateExamples(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	var collisionContext *ValueContext
	var collisionOccurrence ValueOccurrence
	var collisionErr error
	if len(a.UserExamples) > 0 {
		collisionContext = NewValueContext()
		collisionOccurrence, collisionErr = collisionContext.newCollisionOccurrence(a)
	}
	for _, ex := range a.UserExamples {
		if ex.ExplicitNull {
			if !AllowsNull(a) {
				verr.Add(parent, "%sexample value null is incompatible with non-nullable type %s", ctx, a.Type.Name())
			}
			continue
		}
		compatibility := valueCollisionUnknown
		var collisions []valueMapKeyCollision
		if collisionContext != nil && collisionErr == nil {
			result := collisionContext.mapKeyCollisionResult(collisionOccurrence,
				collisionContext.SupplyValue(ValueInput{Raw: ex.Value}))
			compatibility = result.compatibility
			collisions = append(collisions, result.evidence.collisions...)
			slices.SortFunc(collisions, compareValueMapKeyCollisions)
		}
		if len(collisions) > 0 {
			collision := collisions[0]
			location := strings.Join(collision.path, ".")
			if location == "" {
				location = "the example root"
			}
			if len(collision.branchNames) > 0 {
				location += fmt.Sprintf(" in union branch %q", strings.Join(collision.branchNames, "/"))
			}
			verr.Add(parent, "%sexample map keys collide as JSON object member %q at %s", ctx, collision.name, location)
			continue
		}
		if collisionErr != nil {
			continue
		}
		if collisionErr == nil && compatibility == valueCollisionIncompatible {
			verr.Add(parent, "%sexample value %#v is incompatible with type %s", ctx, ex.Value, a.Type.Name())
		}
	}
	return verr
}

func (a *AttributeExpr) inheritRecursive(parent *AttributeExpr, seen map[*AttributeExpr]struct{}) {
	if !a.shouldInherit(parent) {
		return
	}
	for _, nat := range *AsObject(a.Type) {
		if _, patt := objectAttribute(AsObject(parent.Type), nat.Name); patt != nil {
			att := nat.Attribute
			att.Nullable = att.Nullable || patt.Nullable
			if att.Description == "" {
				att.Description = patt.Description
			}
			att.inheritValidations(patt)
			if att.DefaultValue == nil {
				att.DefaultValue = patt.DefaultValue
			}
			if att.Type == nil {
				att.Type = patt.Type
			} else if att.shouldInherit(patt) {
				if _, ok := seen[att]; ok {
					continue
				}
				seen[att] = struct{}{}
				for _, nat := range *AsObject(att.Type) {
					child := nat.Attribute
					if _, parent := objectAttribute(AsObject(patt.Type), nat.Name); parent != nil {
						child.inheritValidations(parent)
						child.inheritRecursive(parent, seen)
					}
				}
			}
		}
	}
}

func (a *AttributeExpr) inheritValidations(parent *AttributeExpr) {
	required := parent.AllRequired()
	if len(required) == 0 {
		return
	}
	obj := AsObject(a.Type)
	for _, name := range required {
		if obj != nil {
			key, _ := objectAttribute(obj, name)
			if key == "" {
				continue
			}
			name = key
		}
		if a.Validation == nil {
			a.Validation = &ValidationExpr{}
		}
		a.Validation.AddRequired(name)
	}
}

func (a *AttributeExpr) shouldInherit(parent *AttributeExpr) bool {
	return a != nil && AsObject(a.Type) != nil &&
		parent != nil && AsObject(parent.Type) != nil
}

// EvalName returns the name used by the DSL evaluation.
func (a *ExampleExpr) EvalName() string {
	return `example "` + a.Summary + `"`
}

// Validate validates the validation expression.
func (v *ValidationExpr) Validate(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	hasMin, hasMax := v.Minimum != nil, v.Maximum != nil
	hasExclusiveMin, hasExclusiveMax := v.ExclusiveMinimum != nil, v.ExclusiveMaximum != nil
	if hasMin && hasExclusiveMin {
		verr.Add(parent, "%sboth minimum and exclusive minimum are defined", ctx)
	}
	if hasMax && hasExclusiveMax {
		verr.Add(parent, "%sboth maximum and exclusive maximum are defined", ctx)
	}
	if hasMin && hasMax && *v.Minimum > *v.Maximum {
		verr.Add(parent, "%sminimum is greater than maximum", ctx)
	}
	if hasMin && hasExclusiveMax && *v.Minimum >= *v.ExclusiveMaximum {
		verr.Add(parent, "%sminimum is greater than or equal to exclusive maximum", ctx)
	}
	if hasExclusiveMin && hasExclusiveMax && *v.ExclusiveMinimum > *v.ExclusiveMaximum {
		verr.Add(parent, "%sexclusive minimum is greater than exclusive maximum", ctx)
	}
	if hasExclusiveMin && hasMax && *v.ExclusiveMinimum >= *v.Maximum {
		verr.Add(parent, "%sexclusive minimum is greater than or equal to maximum", ctx)
	}
	if v.MinLength != nil && v.MaxLength != nil && *v.MinLength > *v.MaxLength {
		verr.Add(parent, "%smin length is greater than max length", ctx)
	}
	for _, pattern := range v.Patterns() {
		if _, err := regexp.Compile(pattern); err != nil {
			verr.Add(parent, "%sinvalid pattern %q: %s", ctx, pattern, err)
		}
	}
	for _, format := range v.Formats() {
		if !isSupportedValidationFormat(format) {
			verr.Add(parent, "%sunsupported format %q", ctx, format)
		}
	}
	return verr
}

// Merge merges other into v so that v enforces the constraints of both: the
// result accepts a value only if both v and other accept it. Numeric and length
// bounds keep the tighter value (the larger lower bound and the smaller upper
// bound), inclusive and exclusive bounds are kept independently, pattern and
// format clauses are conjoined with exact duplicates removed, and required
// fields are unioned. When v has no enum predicate, other's authored enum is
// adopted. Otherwise incoming authored values are added as a conjoined clause,
// followed by the additional clauses. Existing receiver clauses retain their
// order and authorship distinction. Merge never mutates other.
func (v *ValidationExpr) Merge(other *ValidationExpr) {
	if other.Values != nil {
		if v.Values == nil && len(v.EnumClauses) == 0 {
			v.Values = copyEnumValues(other.Values)
		} else {
			v.mergeEnumClauses([][]any{other.Values})
		}
	}
	v.mergeEnumClauses(other.EnumClauses)
	v.mergeFormats(other.Formats())
	v.mergePatterns(other.Patterns())
	v.ExclusiveMinimum = tighterBound(v.ExclusiveMinimum, other.ExclusiveMinimum, true)
	v.Minimum = tighterBound(v.Minimum, other.Minimum, true)
	v.ExclusiveMaximum = tighterBound(v.ExclusiveMaximum, other.ExclusiveMaximum, false)
	v.Maximum = tighterBound(v.Maximum, other.Maximum, false)
	v.MinLength = tighterBound(v.MinLength, other.MinLength, true)
	v.MaxLength = tighterBound(v.MaxLength, other.MaxLength, false)
	v.AddRequired(other.Required...)
}

func copyEnumValues(values []any) []any {
	if values == nil {
		return nil
	}
	copy := make([]any, len(values))
	for index, value := range values {
		copy[index] = copyValueRaw(value)
	}
	return copy
}

// Enums returns every enum membership predicate in authored-then-clause order.
// The returned outer and inner slices are detached. Exact duplicate predicates
// are returned once. An explicit empty predicate is preserved.
func (v *ValidationExpr) Enums() [][]any {
	if v == nil {
		return nil
	}
	capacity := len(v.EnumClauses)
	if v.Values != nil {
		capacity++
	}
	result := make([][]any, 0, capacity)
	if v.Values != nil {
		result = append(result, slices.Clone(v.Values))
	}
	for _, clause := range v.EnumClauses {
		duplicate := slices.ContainsFunc(result, func(existing []any) bool {
			return reflect.DeepEqual(existing, clause)
		})
		if !duplicate {
			result = append(result, slices.Clone(clause))
		}
	}
	return result
}

// Patterns returns the conjoined pattern validations in current-then-clause
// order with exact duplicates removed.
func (v *ValidationExpr) Patterns() []string {
	if v == nil {
		return nil
	}
	patterns := make([]string, 0, 1+len(v.PatternClauses))
	if v.Pattern != "" {
		patterns = append(patterns, v.Pattern)
	}
	for _, pattern := range v.PatternClauses {
		if pattern != "" && !slices.Contains(patterns, pattern) {
			patterns = append(patterns, pattern)
		}
	}
	return patterns
}

// Formats returns the conjoined format validations in current-then-clause
// order with exact duplicates removed.
func (v *ValidationExpr) Formats() []ValidationFormat {
	if v == nil {
		return nil
	}
	formats := make([]ValidationFormat, 0, 1+len(v.FormatClauses))
	if v.Format != "" {
		formats = append(formats, v.Format)
	}
	for _, format := range v.FormatClauses {
		if format != "" && !slices.Contains(formats, format) {
			formats = append(formats, format)
		}
	}
	return formats
}

func (v *ValidationExpr) mergePatterns(patterns []string) {
	for _, pattern := range patterns {
		if pattern == "" || slices.Contains(v.Patterns(), pattern) {
			continue
		}
		if len(v.Patterns()) == 0 {
			v.Pattern = pattern
		} else {
			v.PatternClauses = append(v.PatternClauses, pattern)
		}
	}
}

func (v *ValidationExpr) mergeFormats(formats []ValidationFormat) {
	for _, format := range formats {
		if format == "" || slices.Contains(v.Formats(), format) {
			continue
		}
		if len(v.Formats()) == 0 {
			v.Format = format
		} else {
			v.FormatClauses = append(v.FormatClauses, format)
		}
	}
}

func (v *ValidationExpr) mergeEnumClauses(clauses [][]any) {
	for _, clause := range clauses {
		if slices.ContainsFunc(v.Enums(), func(existing []any) bool {
			return reflect.DeepEqual(existing, clause)
		}) {
			continue
		}
		v.EnumClauses = append(v.EnumClauses, copyEnumValues(clause))
	}
}

func cloneEnumClauses(clauses [][]any) [][]any {
	if clauses == nil {
		return nil
	}
	result := make([][]any, len(clauses))
	for index, clause := range clauses {
		result[index] = copyEnumValues(clause)
	}
	return result
}

// AddRequired merges the required fields into v.
func (v *ValidationExpr) AddRequired(required ...string) {
	for _, r := range required {
		found := slices.Contains(v.Required, r)
		if !found {
			v.Required = append(v.Required, r)
		}
	}
}

// RemoveRequired removes the given field from the list of required fields.
func (v *ValidationExpr) RemoveRequired(required string) {
	for i, r := range v.Required {
		if required == r {
			v.Required = append(v.Required[:i], v.Required[i+1:]...)
			break
		}
	}
}

// HasRequiredOnly returns true if the validation only has the Required field
// with a non-zero value.
func (v *ValidationExpr) HasRequiredOnly() bool {
	if len(v.Enums()) > 0 {
		return false
	}
	if len(v.Formats()) > 0 || len(v.Patterns()) > 0 {
		return false
	}
	if (v.ExclusiveMinimum != nil) ||
		(v.Minimum != nil) ||
		(v.ExclusiveMaximum != nil) ||
		(v.Maximum != nil) ||
		(v.MinLength != nil) ||
		(v.MaxLength != nil) {
		return false
	}
	return true
}

// Dup makes a shallow dup of the validation.
func (v *ValidationExpr) Dup() *ValidationExpr {
	var req []string
	if len(v.Required) > 0 {
		req = make([]string, len(v.Required))
		copy(req, v.Required)
	}
	return &ValidationExpr{
		Values:           v.Values,
		EnumClauses:      cloneEnumClauses(v.EnumClauses),
		Format:           v.Format,
		Pattern:          v.Pattern,
		PatternClauses:   slices.Clone(v.PatternClauses),
		FormatClauses:    slices.Clone(v.FormatClauses),
		ExclusiveMinimum: v.ExclusiveMinimum,
		Minimum:          v.Minimum,
		ExclusiveMaximum: v.ExclusiveMaximum,
		Maximum:          v.Maximum,
		MinLength:        v.MinLength,
		MaxLength:        v.MaxLength,
		Required:         req,
	}
}

// Debug dumps the validation to STDOUT in a Loom developer friendly way.
func (v *ValidationExpr) Debug(title, prefix, indent string) {
	if v.HasRequiredOnly() && len(v.Required) == 0 {
		return
	}
	fmt.Printf("%s%svalidations\n", prefix, title)
	for _, values := range v.Enums() {
		fmt.Printf("%s%s- enum: %s\n", prefix, indent, fmt.Sprintf("%v", values))
	}
	for _, format := range v.Formats() {
		fmt.Printf("%s%s- format: %s\n", prefix, indent, format)
	}
	for _, pattern := range v.Patterns() {
		fmt.Printf("%s%s- pattern: %s\n", prefix, indent, pattern)
	}
	if v.ExclusiveMinimum != nil {
		fmt.Printf("%s%s- exclMin: %v\n", prefix, indent, *v.ExclusiveMinimum)
	}
	if v.Minimum != nil {
		fmt.Printf("%s%s- min: %v\n", prefix, indent, *v.Minimum)
	}
	if v.ExclusiveMaximum != nil {
		fmt.Printf("%s%s- exclMax: %v\n", prefix, indent, *v.ExclusiveMaximum)
	}
	if v.Maximum != nil {
		fmt.Printf("%s%s- max: %v\n", prefix, indent, *v.Maximum)
	}
	if v.MinLength != nil {
		fmt.Printf("%s%s- minLength: %v\n", prefix, indent, *v.MinLength)
	}
	if v.MaxLength != nil {
		fmt.Printf("%s%s- maxLength: %v\n", prefix, indent, *v.MaxLength)
	}
	if len(v.Required) > 0 {
		fmt.Printf("%s%s- required: %v\n", prefix, indent, v.Required)
	}
}

// IsSupportedValidationFormat checks if the validation format is supported by Loom.
func (*AttributeExpr) IsSupportedValidationFormat(vf ValidationFormat) bool {
	return isSupportedValidationFormat(vf)
}

func isSupportedValidationFormat(vf ValidationFormat) bool {
	switch vf {
	case FormatDate:
		return true
	case FormatDateTime:
		return true
	case FormatUUID:
		return true
	case FormatEmail:
		return true
	case FormatHostname:
		return true
	case FormatIPv4:
		return true
	case FormatIPv6:
		return true
	case FormatIP:
		return true
	case FormatURI:
		return true
	case FormatURIReference:
		return true
	case FormatMAC:
		return true
	case FormatCIDR:
		return true
	case FormatRegexp:
		return true
	case FormatJSON:
		return true
	case FormatRFC1123:
		return true
	}
	return false
}

// tighterBound returns whichever of a and b is the more restrictive bound. A
// nil bound is unconstrained, so the other bound wins. When lower is true the
// bounds are lower bounds and the larger value is tighter; otherwise they are
// upper bounds and the smaller value is tighter.
func tighterBound[T cmp.Ordered](a, b *T, lower bool) *T {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if (lower && *b > *a) || (!lower && *b < *a) {
		return b
	}
	return a
}
