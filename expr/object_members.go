package expr

import (
	"slices"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

// AdditionalAttributesFieldName is the Go field name of the map that carries
// the undeclared members of an object with typed AdditionalAttributes.
const AdditionalAttributesFieldName = "AdditionalAttributes"

// TypedObjectDeclaration returns the attribute that declares the typed
// additional-members policy of the object a refers to, following user types,
// or nil when the object has no typed policy.
func (a *AttributeExpr) TypedObjectDeclaration() *AttributeExpr {
	seen := make(map[*AttributeExpr]bool)
	for a != nil && !seen[a] {
		seen[a] = true
		if a.AdditionalAttributes != nil {
			return a
		}
		named, ok := a.Type.(UserType)
		if !ok {
			return nil
		}
		a = named.Attribute()
	}
	return nil
}

// SynthesizedMembers returns, in requirement order, the members that the typed
// object a refers to requires but does not declare. Each is typed by the
// object's AdditionalAttributes and occupies a physical field of its own, so
// generated code can require it like a declared member. It returns nil for
// objects without a typed policy.
func (a *AttributeExpr) SynthesizedMembers() []*NamedAttributeExpr {
	declaration := a.TypedObjectDeclaration()
	if declaration == nil {
		return nil
	}
	object := AsObject(declaration.Type)
	var members []*NamedAttributeExpr
	seen := make(map[string]bool)
	for _, name := range declaration.AllRequired() {
		if seen[name] || object != nil && object.Attribute(name) != nil {
			continue
		}
		seen[name] = true
		members = append(members, &NamedAttributeExpr{Name: name, Attribute: declaration.AdditionalAttributes})
	}
	return members
}

// PhysicalMembers returns the declared members of the object a refers to
// followed by its synthesized members.
func (a *AttributeExpr) PhysicalMembers() []*NamedAttributeExpr {
	var members []*NamedAttributeExpr
	if object := AsObject(a.Type); object != nil {
		members = append(members, *object...)
	}
	return append(members, a.SynthesizedMembers()...)
}

// AdditionalMembersCarrier returns the attribute of the map that carries the
// undeclared members of the typed object a refers to, keyed by member name, or
// nil for objects without a typed policy.
func (a *AttributeExpr) AdditionalMembersCarrier() *AttributeExpr {
	declaration := a.TypedObjectDeclaration()
	if declaration == nil {
		return nil
	}
	carrier := declaration.additionalCarrier
	if carrier == nil || AsMap(carrier.Type).ElemType != declaration.AdditionalAttributes {
		carrier = &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: declaration.AdditionalAttributes}}
		declaration.additionalCarrier = carrier
	}
	return carrier
}

// PhysicalWireNames returns the JSON names of the declared and synthesized
// members of the object a refers to, in member order, followed by the names of
// members a transport projection omitted. Members that are not encoded are
// left out. Undeclared members must not reuse these names.
func (a *AttributeExpr) PhysicalWireNames() []string {
	var names []string
	for _, member := range append(a.PhysicalMembers(), a.OmittedMembers()...) {
		if name := JSONFieldName(ElementName(member.Name), member.Attribute); name != "-" {
			names = append(names, name)
		}
	}
	return names
}

// OmittedMembers returns the members that a transport projection removed from
// the typed object a refers to, such as attributes bound to path parameters
// or headers. They are not required, and undeclared members cannot use their
// names, because the service type still declares them.
func (a *AttributeExpr) OmittedMembers() []*NamedAttributeExpr {
	seen := make(map[*AttributeExpr]bool)
	for a != nil && !seen[a] {
		seen[a] = true
		if a.AdditionalAttributes != nil {
			return slices.Clone(a.omittedMembers)
		}
		named, ok := a.Type.(UserType)
		if !ok {
			return nil
		}
		a = named.Attribute()
	}
	return nil
}

// omitMember records that a transport projection removed the declared member
// name from this typed object.
func (a *AttributeExpr) omitMember(name string) {
	if a.AdditionalAttributes == nil {
		return
	}
	if object := AsObject(a.Type); object != nil {
		if member := object.Attribute(name); member != nil {
			a.omittedMembers = append(slices.Clip(a.omittedMembers), &NamedAttributeExpr{Name: name, Attribute: member})
		}
	}
}

// FindPhysicalMember returns the key and attribute of the declared member with
// the given name, as FindAttribute does, or else of the synthesized member with
// that name. It returns an empty key and nil when neither exists.
func (a *AttributeExpr) FindPhysicalMember(name string) (string, *AttributeExpr) {
	if key, attribute := a.FindAttribute(name); attribute != nil {
		return key, attribute
	}
	for _, member := range a.SynthesizedMembers() {
		if member.Name == name {
			return member.Name, member.Attribute
		}
	}
	return "", nil
}

// containsTypedObject reports whether att or a value it contains is an object
// with typed AdditionalAttributes.
func containsTypedObject(att *AttributeExpr, seen map[*AttributeExpr]bool) bool {
	if att == nil || seen[att] {
		return false
	}
	seen[att] = true
	if att.TypedObjectDeclaration() != nil {
		return true
	}
	switch actual := att.Type.(type) {
	case UserType:
		return containsTypedObject(actual.Attribute(), seen)
	case *Object:
		for _, field := range *actual {
			if containsTypedObject(field.Attribute, seen) {
				return true
			}
		}
	case *Array:
		return containsTypedObject(actual.ElemType, seen)
	case *Map:
		return containsTypedObject(actual.KeyType, seen) || containsTypedObject(actual.ElemType, seen)
	case *Union:
		for _, branch := range actual.Values {
			if containsTypedObject(branch.Attribute, seen) {
				return true
			}
		}
	}
	return false
}

// validateTypedObject rejects declarations that conflict with the exact
// member matching of typed objects: authored case-insensitive member matching,
// a JSON tag on the AdditionalAttributes, which would give every undeclared
// member the same name, and a required undeclared member whose name is the wire name of a declared
// member, which would give two fields one JSON key.
func (a *AttributeExpr) validateTypedObject(ctx string, parent eval.Expression) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if a.AdditionalAttributes == nil {
		return verr
	}
	if object := AsObject(a.Type); object != nil {
		for _, field := range *object {
			if tag, ok := attributeJSONTag(field.Attribute); ok && slices.Contains(strings.Split(tag, ",")[1:], "case:ignore") {
				verr.Add(parent, "%sattribute %q cannot use JSON option case:ignore in an object with AdditionalAttributes", ctx, field.Name)
			}
		}
	}
	if _, tagged := attributeJSONTag(a.AdditionalAttributes); tagged {
		verr.Add(parent, "%sAdditionalAttributes cannot set a JSON tag: undeclared members are named by their keys", ctx)
	}
	declared := make(map[string]string)
	var members []*NamedAttributeExpr
	if object := AsObject(a.Type); object != nil {
		members = *object
	}
	for _, member := range members {
		if name := JSONFieldName(ElementName(member.Name), member.Attribute); name != "-" {
			declared[name] = member.Name
		}
	}
	for _, member := range a.SynthesizedMembers() {
		if owner, ok := declared[JSONFieldName(ElementName(member.Name), member.Attribute)]; ok {
			verr.Add(parent, "%srequired member %q is not declared but attribute %q already uses that JSON name", ctx, member.Name, owner)
		}
	}
	return verr
}
