package expr

import (
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

// validateEnumDefault makes sure that the attribute default value is one of the
// enum values.
func (a *AttributeExpr) validateEnumDefault(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if a.DefaultValue != nil && a.Validation != nil && a.Validation.Values != nil {
		var found bool
		for _, value := range a.Validation.Values {
			if reflect.DeepEqual(value, a.DefaultValue) {
				found = true
				break
			}
		}
		if !found {
			verr.Add(
				parent,
				"%sdefault value %#v is not one of the accepted values: %#v",
				ctx,
				a.DefaultValue,
				a.Validation.Values,
			)
		}
	}
	return verr
}

// validateExamples makes sure that the attribute example values are compatible
// with the attribute type.
func (a *AttributeExpr) validateExamples(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	for _, ex := range a.UserExamples {
		if ex.ExplicitNull {
			if !AllowsNull(a) {
				verr.Add(parent, "%sexample value null is incompatible with non-nullable type %s", ctx, a.Type.Name())
			}
			continue
		}
		if !exampleValueCompatible(a, ex.Value) { // DSL ensures a top-level ex.Value is not nil
			verr.Add(parent, "%sexample value %#v is incompatible with type %s", ctx, ex.Value, a.Type.Name())
		}
	}
	return verr
}

func exampleValueCompatible(attribute *AttributeExpr, value any) bool {
	if attribute == nil || attribute.Type == nil {
		return false
	}
	if value == nil {
		return AllowsNull(attribute)
	}
	if userType, ok := attribute.Type.(UserType); ok {
		return exampleValueCompatible(userType.Attribute(), value)
	}
	switch actual := attribute.Type.(type) {
	case *Array:
		kind := reflect.TypeOf(value).Kind()
		if kind != reflect.Array && kind != reflect.Slice {
			return false
		}
		items := reflect.ValueOf(value)
		for index := 0; index < items.Len(); index++ {
			if !exampleValueCompatible(actual.ElemType, items.Index(index).Interface()) {
				return false
			}
		}
		return true
	case *Map:
		if reflect.TypeOf(value).Kind() != reflect.Map {
			return false
		}
		mapping := reflect.ValueOf(value)
		for _, key := range mapping.MapKeys() {
			if !exampleValueCompatible(actual.KeyType, key.Interface()) ||
				!exampleValueCompatible(actual.ElemType, mapping.MapIndex(key).Interface()) {
				return false
			}
		}
		return true
	case *Union:
		for _, variant := range actual.Values {
			if exampleValueCompatible(variant.Attribute, value) {
				return true
			}
		}
		return false
	default:
		return attribute.Type.IsCompatible(value)
	}
}

func (a *AttributeExpr) inheritRecursive(parent *AttributeExpr, seen map[*AttributeExpr]struct{}) {
	if !a.shouldInherit(parent) {
		return
	}
	for _, nat := range *AsObject(a.Type) {
		if patt := AsObject(parent.Type).Attribute(nat.Name); patt != nil {
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
					parent := AsObject(patt.Type).Attribute(nat.Name)
					if parent != nil {
						child.inheritValidations(parent)
						child.inheritRecursive(parent, seen)
					}
				}
			}
		}
	}
}

func (a *AttributeExpr) inheritValidations(parent *AttributeExpr) {
	if parent.Validation == nil {
		return
	}
	if a.Validation == nil {
		a.Validation = &ValidationExpr{}
	}
	a.Validation.AddRequired(parent.Validation.Required...)
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
	if v.Pattern != "" {
		if _, err := regexp.Compile(v.Pattern); err != nil {
			verr.Add(parent, "%sinvalid pattern %q: %s", ctx, v.Pattern, err)
		}
	}
	return verr
}

// Merge merges other into v.
func (v *ValidationExpr) Merge(other *ValidationExpr) {
	if v.Values == nil {
		v.Values = other.Values
	}
	if v.Format == "" {
		v.Format = other.Format
	}
	if v.Pattern == "" {
		v.Pattern = other.Pattern
	}
	if v.ExclusiveMinimum == nil || (other.ExclusiveMinimum != nil && *v.ExclusiveMinimum > *other.ExclusiveMinimum) {
		v.ExclusiveMinimum = other.ExclusiveMinimum
	}
	if v.Minimum == nil || (other.Minimum != nil && *v.Minimum > *other.Minimum) {
		v.Minimum = other.Minimum
	}
	if v.ExclusiveMaximum == nil || (other.ExclusiveMaximum != nil && *v.ExclusiveMaximum > *other.ExclusiveMaximum) {
		v.ExclusiveMaximum = other.ExclusiveMaximum
	}
	if v.Maximum == nil || (other.Maximum != nil && *v.Maximum < *other.Maximum) {
		v.Maximum = other.Maximum
	}
	if v.MinLength == nil || (other.MinLength != nil && *v.MinLength > *other.MinLength) {
		v.MinLength = other.MinLength
	}
	if v.MaxLength == nil || (other.MaxLength != nil && *v.MaxLength < *other.MaxLength) {
		v.MaxLength = other.MaxLength
	}
	v.AddRequired(other.Required...)
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
	if len(v.Values) > 0 {
		return false
	}
	if v.Format != "" || v.Pattern != "" {
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
		Format:           v.Format,
		Pattern:          v.Pattern,
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
	if len(v.Values) > 0 {
		fmt.Printf("%s%s- enum: %s\n", prefix, indent, fmt.Sprintf("%v", v.Values))
	}
	if v.Format != "" {
		fmt.Printf("%s%s- format: %s\n", prefix, indent, v.Format)
	}
	if v.Pattern != "" {
		fmt.Printf("%s%s- pattern: %s\n", prefix, indent, v.Pattern)
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
