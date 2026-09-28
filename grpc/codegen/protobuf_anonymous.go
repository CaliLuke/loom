package codegen

import (
	"strconv"

	"github.com/CaliLuke/loom/expr"
)

type (
	// messageScope identifies a position in a protocol buffer message.
	messageScope struct {
		// name prefixes the names of the messages generated for anonymous
		// objects at the position.
		name string
		// path is the unambiguous path of the position from the user type
		// that defines the enclosing message.
		path string
	}
)

// anonymousMessageMeta marks the attribute of a message generated for an
// anonymous object.
const anonymousMessageMeta = "grpc:message:anonymous"

// makeProtoBufMessageField applies makeProtoBufMessageR to the attribute of
// a field of the message at scope. An anonymous object field becomes a
// message named after the scope and the field name.
func makeProtoBufMessageField(nat *expr.NamedAttributeExpr, tname *string, sd *ServiceData, seen map[string]expr.UserType, scope messageScope) {
	scope = messageScope{name: scope.name + "_" + nat.Name, path: scope.path + "/" + strconv.Quote(nat.Name)}
	nameAnonymousMessage(nat.Attribute, scope, sd)
	makeProtoBufMessageR(nat.Attribute, tname, sd, seen, scope)
}

// nameAnonymousMessage turns an anonymous object attribute into a user type
// named after scope. Protocol buffer fields cannot have an anonymous message
// type, so the user type provides the message that the field refers to.
func nameAnonymousMessage(att *expr.AttributeExpr, scope messageScope, sd *ServiceData) {
	if !isAnonymousObject(att.Type) {
		return
	}
	tname := sd.anonymousMessageName(scope)
	message := expr.DupAtt(att)
	if message.Meta == nil {
		message.Meta = expr.MetaExpr{}
	}
	message.Meta[anonymousMessageMeta] = []string{"true"}
	att.Type = &expr.UserTypeExpr{
		TypeName:      tname,
		AttributeExpr: message,
		UID:           sd.Name + "#" + tname,
	}
}

// isAnonymousMessage reports whether ut is the message generated for an
// anonymous object by nameAnonymousMessage. Transform code converts such a
// message inline, as the service type of the object is an unnamed struct.
func isAnonymousMessage(ut expr.UserType) bool {
	_, ok := ut.Attribute().Meta[anonymousMessageMeta]
	return ok
}

// isDesignUserTypeName reports whether the design defines a user type or a
// result type with the given name.
func isDesignUserTypeName(name string) bool {
	return expr.Root != nil && expr.Root.UserType(name) != nil
}

// isAnonymousObject reports whether dt is an object type that is not a user
// type.
func isAnonymousObject(dt expr.DataType) bool {
	_, ok := dt.(*expr.Object)
	return ok
}

// anonymousMessageName returns the user type name of the message generated
// for the anonymous object at scope. Protocol buffer messages are identified
// by user type name, so two positions whose names coincide, such as the field
// "outer_inner" and the field "inner" of the field "outer", receive distinct
// names, and so does a position whose name is the name of a design user type
// or an allocated endpoint message.
// The same position always receives the same name.
func (sd *ServiceData) anonymousMessageName(scope messageScope) string {
	name := scope.name
	for i := 2; ; i++ {
		if !isDesignUserTypeName(name) && sd.reserveMessageName(name, scope.path) {
			return name
		}
		name = scope.name + strconv.Itoa(i)
	}
}

// reserveMessageName reports whether name is available to owner and reserves
// it when available. A repeated reservation by the same owner succeeds.
func (sd *ServiceData) reserveMessageName(name, owner string) bool {
	if sd.generatedMessageNames == nil {
		sd.generatedMessageNames = make(map[string]string)
	}
	if previous, ok := sd.generatedMessageNames[name]; ok {
		return previous == owner
	}
	sd.generatedMessageNames[name] = owner
	return true
}
