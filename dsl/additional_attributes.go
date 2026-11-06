package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// AdditionalAttributes describes the members an object does not declare. Each
// undeclared member must hold a value of the given type and satisfy the
// validations of the optional function, as JSON Schema additionalProperties
// with a schema value does. Members declared with Attribute keep their own
// types.
//
// AdditionalAttributes must appear in the definition of an object: a Type,
// an inline Payload, Result or Attribute object, or the Attributes of a
// ResultType. It cannot be combined with Meta("openapi:additionalProperties").
//
// Required may name a member that the object does not declare. That member
// must then be present and hold a value of the AdditionalAttributes type.
//
// An object may declare only AdditionalAttributes, for example
// Attribute("labels", func() { AdditionalAttributes(String) }). A second
// AdditionalAttributes in the same object is an error.
//
// The first argument is a type or the name of a user type.
//
// Example:
//
//	var Labels = Type("Labels", func() {
//	    Attribute("id", String)
//	    AdditionalAttributes(Int, func() {
//	        Minimum(0)
//	    })
//	    Required("id", "count") // "count" is an additional attribute
//	})
func AdditionalAttributes(t any, fn ...func()) {
	var at *expr.AttributeExpr
	switch def := eval.Current().(type) {
	case *expr.AttributeExpr:
		at = def
	case *expr.ResultTypeExpr:
		at = def.AttributeExpr
	default:
		eval.IncompatibleDSL()
		return
	}
	if _, named := at.Type.(expr.UserType); named || at.Type != nil && !expr.IsObject(at.Type) {
		eval.ReportError("AdditionalAttributes must appear in an object declaration")
		return
	}
	if at.AdditionalAttributes != nil {
		eval.ReportError("AdditionalAttributes is already declared for this object")
		return
	}
	if len(fn) > 1 {
		eval.TooManyArgError()
		return
	}
	typ, ok := t.(expr.DataType)
	if !ok {
		if name, isName := t.(string); isName {
			typ = expr.Root.UserType(name)
		}
	}
	if typ == nil {
		eval.ReportError("invalid AdditionalAttributes argument: not a type and not a known user type name")
		return
	}
	additional := &expr.AttributeExpr{Type: typ}
	if len(fn) == 1 {
		localizeAttribute(additional, at)
		eval.Execute(fn[0], additional)
	}
	if at.Type == nil {
		at.Type = &expr.Object{}
	}
	at.AdditionalAttributes = additional
}
