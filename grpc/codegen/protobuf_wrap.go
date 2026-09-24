package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// arrayMessageMeta marks the attribute of the message that wraps a named
// array.
const arrayMessageMeta = "grpc:message:array"

// wrapArrayUserType makes the named array held by att, such as
// Type("Tags", ArrayOf(String)), the message that wraps the array in its
// "field" attribute, as for an array payload or result, since a message field
// or a oneof branch cannot refer to an array by name. The message takes the
// name of the type. A named array whose type is another named array wraps the
// array itself. The new user type has its own identifier, so it does not
// share the examples or copies of the service type.
func wrapArrayUserType(att *expr.AttributeExpr) {
	ut, ok := att.Type.(expr.UserType)
	if !ok || !expr.IsArray(ut) {
		return
	}
	att.Type = &expr.UserTypeExpr{
		TypeName:      ut.Name(),
		AttributeExpr: arrayMessageAttribute(ut),
		UID:           ut.ID() + "#message",
	}
}

// arrayMessageAttribute returns the attribute of the message that wraps the
// named array ut in its "field" attribute, marked with arrayMessageMeta. A
// named array whose type is another named array wraps the array itself.
func arrayMessageAttribute(ut expr.UserType) *expr.AttributeExpr {
	array := ut.Attribute()
	for {
		inner, ok := array.Type.(expr.UserType)
		if !ok {
			break
		}
		array = inner.Attribute()
	}
	message := wrapperAttribute(array, false)
	message.Meta = expr.MetaExpr{arrayMessageMeta: []string{"true"}}
	return message
}

// isArrayMessage reports whether ut is the message that wrapArrayUserType
// generated for a named array. Transform code converts a field that holds
// such a message inline, as it converts the array of the service type.
func isArrayMessage(ut expr.UserType) bool {
	_, ok := ut.Attribute().Meta[arrayMessageMeta]
	return ok
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
func wrapAttr(att *expr.AttributeExpr, tname string, req bool, sd *ServiceData) {
	switch dt := att.Type.(type) {
	case expr.UserType:
		// Don't change the original user type. Create a copy and wrap that.
		ut := expr.Dup(dt).(expr.UserType)
		ut.SetAttribute(wrapperAttribute(ut.Attribute(), req))
		att.Type = ut
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
	name, _ := protoBufUnionNames("field", union)
	return name
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
func wrapperGoFieldName(att *expr.AttributeExpr, ctx *codegen.AttributeContext) string {
	if nat := unionWrapperField(att); nat != nil {
		return ctx.Scope.Field(nat.Attribute, nat.Name, true)
	}
	return "Field"
}
