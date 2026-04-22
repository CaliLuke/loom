package expr

import (
	"slices"
	"strings"
)

// AllRequired returns the list of all required fields from the underlying
// object. This method recurses if the type is itself an attribute (i.e. a
// UserType, this happens with the Reference DSL for example).
func (a *AttributeExpr) AllRequired() []string {
	if u, ok := a.Type.(UserType); ok {
		return u.Attribute().AllRequired()
	}
	if a.Validation != nil {
		return a.Validation.Required
	}
	return nil
}

// IsRequired returns true if the given string matches the name of a required
// attribute, false otherwise. This method only applies to attributes of type
// Object.
func (a *AttributeExpr) IsRequired(attName string) bool {
	return slices.Contains(a.AllRequired(), attName)
}

// IsRequiredNoDefault returns true if the given string matches the name of a
// required attribute and the attribute has no default value, false otherwise.
// This method only applies to attributes of type Object.
func (a *AttributeExpr) IsRequiredNoDefault(attName string) bool {
	for _, name := range a.AllRequired() {
		if name == attName {
			return a.GetDefault(name) == nil
		}
	}
	return false
}

// IsPrimitivePointer returns true if the field generated for the given
// attribute should be a pointer to a primitive type. The receiver attribute must
// be an object.
//
// If useDefault is true and the attribute has a default value then
// IsPrimitivePointer returns false. This makes it possible to differentiate
// between request types where attributes with default values should not be
// generated using a pointer value and response types where they should.
//
//	DefaultValue UseDefault Pointer (assuming all other conditions are true)
//	Yes          True       False
//	Yes          False      True
//	No           True       True
//	No           False      True
func (a *AttributeExpr) IsPrimitivePointer(attName string, useDefault bool) bool {
	o := AsObject(a.Type)
	if o == nil {
		panic("checking pointer field on non-object") // bug
	}
	att := o.Attribute(attName)
	if att == nil {
		return false
	}
	if !IsPrimitive(att.Type) {
		return false
	}
	t := unalias(att.Type)
	return t.Kind() != BytesKind && t.Kind() != AnyKind &&
		!a.IsRequired(attName) && (!a.HasDefaultValue(attName) || !useDefault)
}

func unalias(dt DataType) DataType {
	if ut, ok := dt.(UserType); ok {
		if _, ok := ut.Attribute().Type.(Primitive); ok {
			return ut.Attribute().Type
		}
		return unalias(ut.Attribute().Type)
	}
	return dt
}

// HasTag returns true if the attribute is an object that has an attribute with
// the given tag.
func (a *AttributeExpr) HasTag(tag string) bool {
	if a == nil {
		return false
	}
	obj := AsObject(a.Type)
	if obj == nil {
		return false
	}
	for _, at := range *obj {
		if _, ok := at.Attribute.Meta[tag]; ok {
			return true
		}
	}
	return false
}

// HasTagPrefix returns true if the attribute is an object that has an attribute with
// the given tag prefix.
func (a *AttributeExpr) HasTagPrefix(prefix string) bool {
	if a == nil {
		return false
	}
	obj := AsObject(a.Type)
	if obj == nil {
		return false
	}
	for _, at := range *obj {
		for k := range at.Attribute.Meta {
			if strings.HasPrefix(k, prefix) {
				return true
			}
		}
	}
	return false
}

// FieldTag returns the field tag if the attribute is a field.
func (a *AttributeExpr) FieldTag() (tag string, found bool) {
	if a == nil {
		return
	}
	return a.Meta.Last("rpc:tag")
}

// HasDefaultValue returns true if the attribute with the given name has a
// default value.
func (a *AttributeExpr) HasDefaultValue(attName string) bool {
	return a.GetDefault(attName) != nil
}

// GetDefault gets the default value for the child attribute with the given
// name. It returns nil if the child attribute with the given name does not
// exist or if the child attribute does not have a default value.
func (a *AttributeExpr) GetDefault(attName string) any {
	if o := AsObject(a.Type); o != nil {
		att := o.Attribute(attName)
		if att.DefaultValue != nil {
			return att.DefaultValue
		}
		if ut, ok := att.Type.(UserType); ok && !IsObject(ut) {
			return ut.Attribute().DefaultValue
		}
	}
	return nil
}

// SetDefault sets the default for the attribute. It also converts HashVal
// and ArrayVal to map and slice respectively.
func (a *AttributeExpr) SetDefault(def any) {
	switch actual := def.(type) {
	case MapVal:
		a.DefaultValue = actual.ToMap()
	case ArrayVal:
		a.DefaultValue = actual.ToSlice()
	default:
		a.DefaultValue = actual
	}
}
