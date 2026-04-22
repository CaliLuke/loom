package expr

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

type (
	// AttributeExpr defines an object field with optional description,
	// default value and validations.
	AttributeExpr struct {
		// DSLFunc contains the DSL used to initialize the expression.
		eval.DSLFunc
		// Attribute type
		Type DataType
		// Base types if any
		Bases []DataType
		// Attribute reference types if any
		References []DataType
		// Optional description
		Description string
		// Docs points to external documentation
		Docs *DocsExpr
		// Optional validations
		Validation *ValidationExpr
		// Meta is a list of key/value pairs
		Meta MetaExpr
		// Optional member default value
		DefaultValue any
		// UserExample set in DSL or computed in Finalize
		UserExamples []*ExampleExpr
		// finalized is true if the attribute has been finalized - only
		// applies if attribute type is an object
		finalized bool
	}

	// ExampleExpr represents an example.
	ExampleExpr struct {
		// Summary is the example short summary.
		Summary string
		// Description is an optional long description.
		Description string
		// Meta holds design-time metadata attached to the example.
		Meta MetaExpr
		// Value is the example value.
		Value any
	}

	// Val is the type used to provide the value of examples for attributes that are
	// objects.
	Val map[string]any

	// CompositeExpr defines a generic composite expression that contains an
	// attribute.  This makes it possible for plugins to use attributes in
	// their own data structures.
	CompositeExpr interface {
		// Attribute returns the composite expression embedded attribute.
		Attribute() *AttributeExpr
	}

	// ValidationExpr contains validation rules for an attribute.
	ValidationExpr struct {
		// Values represents an enum validation as described at
		// http://json-schema.org/latest/json-schema-validation.html#anchor76.
		Values []any
		// Format represents a format validation as described at
		// http://json-schema.org/latest/json-schema-validation.html#anchor104.
		Format ValidationFormat
		// PatternValidationExpr represents a pattern validation as
		// described at
		// http://json-schema.org/latest/json-schema-validation.html#anchor33
		Pattern string
		// ExclusiveMinimum represents an exclusiveMinimum value validation as described
		// at
		// http://json-schema.org/draft/2019-09/json-schema-validation.html#rfc.section.6.2.5.
		ExclusiveMinimum *float64
		// Minimum represents an minimum value validation as described
		// at
		// http://json-schema.org/latest/json-schema-validation.html#anchor21.
		Minimum *float64
		// Maximum represents a maximum value validation as described at
		// http://json-schema.org/latest/json-schema-validation.html#anchor17.
		Maximum *float64
		// ExclusiveMaximum represents an exclusiveMaximum value validation as described
		// at
		// http://json-schema.org/draft/2019-09/json-schema-validation.html#rfc.section.6.2.3.
		ExclusiveMaximum *float64
		// MinLength represents an minimum length validation as
		// described at
		// http://json-schema.org/latest/json-schema-validation.html#anchor29.
		MinLength *int
		// MaxLength represents an maximum length validation as
		// described at
		// http://json-schema.org/latest/json-schema-validation.html#anchor26.
		MaxLength *int
		// Required list the required fields of object attributes as
		// described at
		// http://json-schema.org/latest/json-schema-validation.html#anchor61.
		Required []string
	}

	// ValidationFormat is the type used to enumerate the possible string
	// formats.
	ValidationFormat string

	// CookieSameSiteValue is the type used to enumerate the possible cookie
	// SameSite values.
	CookieSameSiteValue string
)

const (
	// FormatDate describes RFC3339 date values.
	FormatDate ValidationFormat = "date"

	// FormatDateTime describes RFC3339 date time values.
	FormatDateTime ValidationFormat = "date-time"

	// FormatUUID describes RFC4122 UUID values.
	FormatUUID = "uuid"

	// FormatEmail describes RFC5322 email addresses.
	FormatEmail = "email"

	// FormatHostname describes RFC1035 Internet hostnames.
	FormatHostname = "hostname"

	// FormatIPv4 describes RFC2373 IPv4 address values.
	FormatIPv4 = "ipv4"

	// FormatIPv6 describes RFC2373 IPv6 address values.
	FormatIPv6 = "ipv6"

	// FormatIP describes RFC2373 IPv4 or IPv6 address values.
	FormatIP = "ip"

	// FormatURI describes RFC3986 URI values.
	FormatURI = "uri"

	// FormatMAC describes IEEE 802 MAC-48, EUI-48 or EUI-64 MAC address values.
	FormatMAC = "mac"

	// FormatCIDR describes RFC4632 and RFC4291 CIDR notation IP address values.
	FormatCIDR = "cidr"

	// FormatRegexp describes regular expression syntax accepted by RE2.
	FormatRegexp = "regexp"

	// FormatJSON describes JSON text.
	FormatJSON = "json"

	// FormatRFC1123 describes RFC1123 date time values.
	FormatRFC1123 = "rfc1123"
)

const (
	CookieSameSiteStrict  CookieSameSiteValue = "strict"
	CookieSameSiteLax     CookieSameSiteValue = "lax"
	CookieSameSiteNone    CookieSameSiteValue = "none"
	CookieSameSiteDefault CookieSameSiteValue = "default"
)

// validated keeps track of validated attributes to handle cyclical definitions.
var validated = make(map[*AttributeExpr]bool)

// TaggedAttribute returns the name of the child attribute of a with the given
// tag if a is an object.
func TaggedAttribute(a *AttributeExpr, tag string) string {
	obj := AsObject(a.Type)
	if obj == nil {
		return ""
	}
	for _, at := range *obj {
		if _, ok := at.Attribute.Meta[tag]; ok {
			return at.Name
		}
	}
	for _, b := range a.Bases {
		at := &AttributeExpr{Type: b}
		if ut, ok := b.(UserType); ok {
			at = ut.Attribute()
		}
		if n := TaggedAttribute(at, tag); n != "" {
			return n
		}
	}
	return ""
}

// walkAttribute iterates over the given attribute, its bases and references
// (if any). It calls the given function giving each attribute as it iterates.
// It stops if the given attribute is not an object type or there is no more
// attribute to iterate over or if the iterator function returned an error. It
// is generally used in implementing the Validator interface since attribute
// bases and references are only merged during Finalize. It is not a recursive
// implementation.
// Note: keep this function private as it does not walk through all types.
// External packages should use the codegen.Walk function instead.
func walkAttribute(att *AttributeExpr, it func(name string, a *AttributeExpr) error) error {
	switch dt := att.Type.(type) {
	case UserType:
		if err := walkAttribute(dt.Attribute(), it); err != nil {
			return err
		}
	case *Object:
		for _, nat := range *dt {
			if err := it(nat.Name, nat.Attribute); err != nil {
				return err
			}
		}
	}
	for _, b := range att.Bases {
		if err := walkAttribute(&AttributeExpr{Type: b}, it); err != nil {
			return err
		}
	}
	for _, r := range att.References {
		if err := walkAttribute(&AttributeExpr{Type: r}, it); err != nil {
			return err
		}
	}
	return nil
}

// EvalName returns the name used by the DSL evaluation.
func (a *AttributeExpr) EvalName() string {
	if a != nil {
		if n, ok := a.Meta["openapi:typename"]; ok && len(n) > 0 {
			return fmt.Sprintf("type %#v", n[0])
		}
	}
	return "attribute"
}

// Finalize merges base and reference type attributes and finalizes the Type
// attribute.
func (a *AttributeExpr) Finalize() {
	if a.finalized {
		return // Avoid infinite recursion.
	}
	a.finalized = true
	if err := a.resolveTypeRef(); err != nil {
		eval.ReportError(err.Error())
		return
	}
	if ut, ok := a.Type.(UserType); ok {
		ut.Finalize()
	}
	switch {
	case IsObject(a.Type):
		for _, ref := range a.References {
			ru, ok := ref.(UserType)
			if !ok {
				continue
			}
			a.Inherit(ru.Attribute())
		}
		for _, base := range a.Bases {
			ru, ok := base.(UserType)
			if !ok {
				continue
			}
			a.Merge(ru.Attribute())
		}

		// Now that we've merged the bases, we can clear them.  This
		// avoids issues where the bases are dupped and modifications
		// made to the originals are not reflected in the dups.
		a.Bases = nil

		for _, nat := range *AsObject(a.Type) {
			nat.Attribute.Finalize()
		}
	case IsUnion(a.Type):
		union := AsUnion(a.Type)
		for _, nat := range union.Values {
			nat.Attribute.Finalize()
		}
		normalizeDerivedUnion(union)
	case IsArray(a.Type):
		AsArray(a.Type).ElemType.Finalize()
	case IsMap(a.Type):
		m := AsMap(a.Type)
		m.ElemType.Finalize()
		m.KeyType.Finalize()
	}
}

func (a *AttributeExpr) prepareTypeRefs(seen map[*AttributeExpr]struct{}) {
	if a == nil {
		return
	}
	if _, ok := seen[a]; ok {
		return
	}
	seen[a] = struct{}{}

	if err := a.resolveTypeRef(); err != nil {
		eval.ReportError(err.Error())
		return
	}
	switch {
	case IsObject(a.Type):
		for _, nat := range *AsObject(a.Type) {
			nat.Attribute.prepareTypeRefs(seen)
		}
	case IsUnion(a.Type):
		for _, nat := range AsUnion(a.Type).Values {
			nat.Attribute.prepareTypeRefs(seen)
		}
	case IsArray(a.Type):
		AsArray(a.Type).ElemType.prepareTypeRefs(seen)
	case IsMap(a.Type):
		m := AsMap(a.Type)
		m.ElemType.prepareTypeRefs(seen)
		m.KeyType.prepareTypeRefs(seen)
	}
}

func (a *AttributeExpr) resolveTypeRef() error {
	ut, ok := a.Type.(*UserTypeExpr)
	if !ok || !isTypeRef(ut) {
		return nil
	}
	resolved := Root.UserType(ut.TypeName)
	if resolved == nil {
		return fmt.Errorf("unknown type reference %q", ut.TypeName)
	}
	a.Type = resolved
	return nil
}

func isTypeRef(ut *UserTypeExpr) bool {
	return ut != nil && strings.HasPrefix(ut.UID, "$type-ref:")
}

func normalizeDerivedUnion(union *Union) {
	if !hasDerivedUnionVariantNames(union) {
		return
	}
	types := make([]DataType, len(union.Values))
	for i, nat := range union.Values {
		types[i] = nat.Attribute.Type
	}
	names := DerivedUnionVariantNames(types)
	for i, nat := range union.Values {
		nat.Name = names[i]
	}
	if !union.ExplicitTypeName {
		union.TypeName = DerivedUnionTypeName(names)
	}
}

func hasDerivedUnionVariantNames(union *Union) bool {
	if union == nil || len(union.Values) == 0 {
		return false
	}
	for _, nat := range union.Values {
		if nat == nil || nat.Attribute == nil {
			return false
		}
		if _, ok := nat.Attribute.Meta.Last("oneof:variant:derived"); !ok {
			return false
		}
	}
	return true
}

// Merge merges other's attributes into a overriding attributes of a with
// attributes of other with identical names.
//
// This only applies to attributes of type Object and Merge panics if the
// argument or the target is not of type Object.
func (a *AttributeExpr) Merge(other *AttributeExpr) {
	if other == nil {
		return
	}
	left := AsObject(a.Type)
	right := AsObject(other.Type)
	if left == nil || right == nil {
		panic("cannot merge non object attributes") // bug
	}
	if a.Type == Empty && len(*right) > 0 {
		a.Type = &Object{}
		left = AsObject(a.Type)
	}
	if other.Validation != nil {
		if a.Validation == nil {
			a.Validation = other.Validation.Dup()
		} else {
			a.Validation.Merge(other.Validation)
		}
	}
	for _, nat := range *right {
		left.Set(nat.Name, nat.Attribute)
	}
}

// Inherit merges the properties of existing target type attributes with the
// argument's. The algorithm is recursive so that child attributes are also
// merged.
func (a *AttributeExpr) Inherit(parent *AttributeExpr) {
	if !a.shouldInherit(parent) {
		return
	}
	pobj := AsObject(parent.Type)
	if a.Type == Empty && len(*pobj) > 0 {
		a.Type = &Object{}
	}
	a.inheritValidations(parent)
	a.inheritRecursive(parent, make(map[*AttributeExpr]struct{}))
}

// Find finds a child attribute with the given name in the attribute and
// its bases and references. If the parent attribute is not an object, it
// returns nil.
func (a *AttributeExpr) Find(name string) *AttributeExpr {
	findAttrFn := func(typ DataType) *AttributeExpr {
		switch t := typ.(type) {
		case UserType:
			return t.Attribute().Find(name)
		case *Object:
			if att := t.Attribute(name); att != nil {
				return att
			}
		}
		return nil
	}

	if att := findAttrFn(a.Type); att != nil {
		return att
	}
	for _, b := range a.Bases {
		if att := findAttrFn(b); att != nil {
			return att
		}
	}
	for _, ref := range a.References {
		if att := findAttrFn(ref); att != nil {
			return att
		}
	}
	return nil
}

// Delete removes an attribute with the given name. It does nothing if the
// attribute expression is not an object type.
func (a *AttributeExpr) Delete(name string) {
	switch t := a.Type.(type) {
	case UserType:
		t.Attribute().Delete(name)
	case *Object:
		AsObject(t).Delete(name)
		if a.Validation != nil {
			a.Validation.RemoveRequired(name)
		}
		for _, ex := range a.UserExamples {
			if m, ok := ex.Value.(map[string]any); ok {
				delete(m, name)
			}
		}
	}
}

// AddMeta adds values to the meta field of the attribute.
func (a *AttributeExpr) AddMeta(name string, vals ...string) {
	if a.Meta == nil {
		a.Meta = make(MetaExpr)
	}
	a.Meta[name] = append(a.Meta[name], vals...)
}

// DeleteMeta removes the metadata entry with the given name.
func (a *AttributeExpr) DeleteMeta(name string) {
	delete(a.Meta, name)
}

// ExtractUserExamples return the examples defined in the design directly on the
// attribute or on its type.
func (a *AttributeExpr) ExtractUserExamples() []*ExampleExpr {
	return a.extractUserExamples(make(map[string]struct{}))
}

func (a *AttributeExpr) extractUserExamples(seen map[string]struct{}) []*ExampleExpr {
	if a == nil {
		return nil
	}
	if len(a.UserExamples) > 0 {
		return a.UserExamples
	}
	for _, ref := range a.References {
		if examples := extractUserExamplesFromType(ref, seen); len(examples) > 0 {
			return examples
		}
	}
	for _, base := range a.Bases {
		if examples := extractUserExamplesFromType(base, seen); len(examples) > 0 {
			return examples
		}
	}
	ut, ok := a.Type.(UserType)
	if !ok {
		return nil
	}
	return extractUserExamplesFromType(ut, seen)
}

func extractUserExamplesFromType(dt DataType, seen map[string]struct{}) []*ExampleExpr {
	ut, ok := dt.(UserType)
	if !ok {
		return nil
	}
	id := ut.ID()
	if _, ok := seen[id]; ok {
		return nil
	}
	seen[id] = struct{}{}
	return ut.Attribute().extractUserExamples(seen)
}
