package codegen

import (
	"github.com/CaliLuke/loom/expr"
)

// collectionMessageMeta marks the attribute of the message that wraps a
// named array or map, or an array or map that is an array element or a map
// value.
const collectionMessageMeta = "grpc:message:collection"

// wrapCollectionUserType makes the named array or map held by att, such as
// Type("Tags", ArrayOf(String)) or Type("Index", MapOf(String, Int)), the
// message that wraps the collection in its "field" attribute, as for an array
// or map payload or result, since a message field, an array element, a map
// value or a oneof branch cannot refer to an array or map by name. The message
// takes the name of the type. A named collection whose type is another named
// collection wraps the collection itself. The new user type has its own
// identifier, so it does not share the examples or copies of the service
// type.
func wrapCollectionUserType(att *expr.AttributeExpr) {
	ut, ok := att.Type.(expr.UserType)
	if !ok || !expr.IsArray(ut) && !expr.IsMap(ut) {
		return
	}
	att.Type = &expr.UserTypeExpr{
		TypeName:      ut.Name(),
		AttributeExpr: collectionMessageAttribute(ut),
		UID:           ut.ID() + "#message",
	}
}

// collectionMessageAttribute returns the attribute of the message that wraps
// the named array or map ut in its "field" attribute, marked with
// collectionMessageMeta. A named collection whose type is another named
// collection wraps the collection itself.
func collectionMessageAttribute(ut expr.UserType) *expr.AttributeExpr {
	collection := ut.Attribute()
	for {
		inner, ok := collection.Type.(expr.UserType)
		if !ok {
			break
		}
		collection = inner.Attribute()
	}
	message := wrapperAttribute(collection, false)
	message.Meta = expr.MetaExpr{collectionMessageMeta: []string{"true"}}
	return message
}

// isCollectionMessage reports whether ut is the message that
// wrapCollectionUserType generated for a named array or map, or the message
// that wraps an array or map that is an array element or a map value.
// Transform code converts such a message inline, as it converts the
// collection of the service type.
func isCollectionMessage(ut expr.UserType) bool {
	_, ok := ut.Attribute().Meta[collectionMessageMeta]
	return ok
}

// markCollectionMessage marks ut, the message that wraps an array or map
// that is an array element or a map value, with collectionMessageMeta.
func markCollectionMessage(ut expr.UserType) {
	att := ut.Attribute()
	if att.Meta == nil {
		att.Meta = expr.MetaExpr{}
	}
	att.Meta[collectionMessageMeta] = []string{"true"}
}

// wrapUnionBranch makes the union held by the union branch att the message
// that wraps its oneof, as for a union payload or result, since a oneof
// cannot hold another oneof. The message of a named union takes its name. The
// message of a union that is not named takes the name of the union unless a
// design type or a different union already uses it. The branch refers to a
// new user type with its own identifier, so the same named union used as a
// field elsewhere in the message remains a oneof of the message that holds
// it, including in copies of the message, which share user types by
// identifier.
func wrapUnionBranch(att *expr.AttributeExpr, sd *ServiceData) {
	union := expr.AsUnion(att.Type)
	if union == nil {
		return
	}
	ut, named := att.Type.(expr.UserType)
	if !named {
		name := sd.anonymousMessageName(messageScope{name: union.Name(), path: "oneof:" + union.Hash()})
		ut = &expr.UserTypeExpr{TypeName: name, AttributeExpr: &expr.AttributeExpr{Type: union}, UID: sd.Name + "#" + name}
	}
	att.Type = &expr.UserTypeExpr{
		TypeName:      ut.Name(),
		AttributeExpr: wrapperAttribute(ut.Attribute(), true),
		UID:           ut.ID() + "#branch",
	}
}

// wrapAttr makes the attribute type a user type by wrapping the given
// attribute into an attribute named "field", or for a union into the
// attribute that unionWrapperFieldName names.
//
// A named type, such as a named primitive or a named union, is wrapped in a
// new user type with its name and its own identifier, whose field holds the
// value of the named type. The named type itself is left unchanged, so the
// other references to it, such as a field of the same named primitive or the
// field of a recursive branch type of the named union, still refer to the
// named type, in the message, in the copies of the message and in examples,
// which share user types by identifier.
func wrapAttr(att *expr.AttributeExpr, tname string, req bool, sd *ServiceData) {
	switch dt := att.Type.(type) {
	case expr.UserType:
		att.Type = &expr.UserTypeExpr{
			TypeName:      dt.Name(),
			AttributeExpr: wrapperAttribute(dt.Attribute(), req),
			UID:           dt.ID() + "#message",
		}
	default:
		att.Type = &expr.UserTypeExpr{
			TypeName:      tname,
			AttributeExpr: wrapperAttribute(att, req),
			UID:           sd.Name + "#" + tname,
		}
	}
	// Validation is moved to wrapped attribute.
	att.Validation = nil
}

// wrapperAttribute returns the object attribute of the message that wraps the
// value of attr: a single attribute named "field", or for a union the
// attribute that unionWrapperFieldName names, numbered 1 and carrying the
// validations of attr. req makes the wrapped attribute required.
func wrapperAttribute(attr *expr.AttributeExpr, req bool) *expr.AttributeExpr {
	name := "field"
	if union := expr.AsUnion(attr.Type); union != nil {
		name = unionWrapperFieldName(union)
	}
	res := &expr.AttributeExpr{
		Type: &expr.Object{
			&expr.NamedAttributeExpr{
				Name: name,
				Attribute: &expr.AttributeExpr{
					Type:       attr.Type,
					Meta:       expr.MetaExpr{"rpc:tag": []string{"1"}},
					Validation: attr.Validation,
				},
			},
		},
	}
	if req {
		res.Validation = &expr.ValidationExpr{
			Required: []string{name},
		}
	}
	return res
}

// unwrapAttr returns the attribute under the attribute name "field", or the
// union that wrapAttr wrapped. Otherwise it returns the given attribute.
func unwrapAttr(att *expr.AttributeExpr) *expr.AttributeExpr {
	if a := att.Find("field"); a != nil {
		return a
	}
	if nat := unionWrapperField(att); nat != nil {
		return nat.Attribute
	}
	return att
}

// unionWrapperFieldName returns the name of the attribute of the message that
// wraps union, which is also the name of its oneof: "field" followed by
// "_oneof" as many times as needed to differ from the branch names.
func unionWrapperFieldName(union *expr.Union) string {
	names := newProtoMessageNames(&expr.Object{{Name: "field", Attribute: &expr.AttributeExpr{Type: union}}})
	return names.field("field")
}

// unionWrapperField returns the single attribute of att when att is a
// message that wrapAttr made to wrap a union, or nil otherwise.
func unionWrapperField(att *expr.AttributeExpr) *expr.NamedAttributeExpr {
	obj := expr.AsObject(att.Type)
	if obj == nil || len(*obj) != 1 {
		return nil
	}
	nat := (*obj)[0]
	union := expr.AsUnion(nat.Attribute.Type)
	if union == nil || nat.Name != unionWrapperFieldName(union) {
		return nil
	}
	return nat
}

// wrapperGoFieldName returns the name of the Go field that holds the wrapped
// value in the protocol buffer Go type generated for the wrapper message att:
// the oneof of a wrapped union or "Field".
func wrapperGoFieldName(att *expr.AttributeExpr) string {
	if nat := unionWrapperField(att); nat != nil {
		return newProtoMessageNames(expr.AsObject(att.Type)).goField(nat.Name)
	}
	return "Field"
}

// wrappedUnionTransformAttrs returns ta when att is not a message that
// wrapAttr made to wrap a union. Otherwise it returns a copy of ta whose
// oneofFields are the names of the Go fields of the oneof fields of the
// message.
func wrappedUnionTransformAttrs(att *expr.AttributeExpr, ta *transformAttrs) *transformAttrs {
	nat := unionWrapperField(att)
	if nat == nil {
		return ta
	}
	ta = dupTransformAttrs(ta)
	ta.oneofFields = newProtoMessageNames(expr.AsObject(att.Type)).goBranches(nat.Name)
	return ta
}
