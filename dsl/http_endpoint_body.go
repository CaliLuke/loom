package dsl

import (
	"github.com/CaliLuke/loom/v3/eval"
	"github.com/CaliLuke/loom/v3/expr"
)

// Body describes a HTTP request or response body.
//
// Body must appear in a Method HTTP expression to define the request body or in
// an Error or Result HTTP expression to define the response body. If Body is
// absent then the body is built using the HTTP endpoint request or response
// type attributes not used to describe parameters (request only) or headers.
//
// Body accepts one argument which describes the shape of the body, it can be:
//
//   - The name of an attribute of the request or response type. In this case the
//     attribute type describes the shape of the body.
//
//   - A function listing the body attributes. The attributes inherit the
//     properties (description, type, validations etc.) of the request or
//     response type attributes with identical names.
//
// Assuming the type:
//
//	var CreatePayload = Type("CreatePayload", func() {
//	    Attribute("name", String, "Name of account")
//	})
//
// The following:
//
//	Method("create", func() {
//	    Payload(CreatePayload)
//	})
//
// is equivalent to:
//
//	Method("create", func() {
//	    Payload(CreatePayload)
//	    HTTP(func() {
//	        Body(func() {
//	            Attribute("name")
//	        })
//	    })
//	})
func Body(args ...any) {
	body(args, false)
}

// OpenAPIBody describes a documentation-only HTTP response body.
//
// OpenAPIBody must appear in a Response or HTTP Error expression. It affects
// the generated OpenAPI contract only and does not change runtime transport
// encode/decode behavior.
//
// OpenAPIBody accepts the same arguments as Body.
func OpenAPIBody(args ...any) {
	body(args, true)
}

func body(args []any, openAPIOnly bool) {
	if len(args) == 0 {
		eval.TooFewArgError()
		return
	}

	var (
		ref    *expr.AttributeExpr
		setter func(*expr.AttributeExpr)
		kind   string
	)

	switch e := eval.Current().(type) {
	case *expr.HTTPEndpointExpr:
		if openAPIOnly {
			eval.IncompatibleDSL()
			return
		}
		ref = e.MethodExpr.Payload
		setter = func(att *expr.AttributeExpr) {
			e.Body = att
		}
		kind = "Request"
	case *expr.HTTPErrorExpr:
		ref = e.AttributeExpr
		setter = func(att *expr.AttributeExpr) {
			if e.Response == nil {
				e.Response = &expr.HTTPResponseExpr{}
			}
			if openAPIOnly {
				e.Response.OpenAPIBody = att
			} else {
				e.Response.Body = att
			}
		}
		kind = "Error"
		if e.Name != "" {
			kind += " " + e.Name
		}
	case *expr.HTTPResponseExpr:
		p, ok := e.Parent.(*expr.HTTPEndpointExpr)
		if !ok {
			eval.IncompatibleDSL()
			return
		}
		ref = p.MethodExpr.Result
		setter = func(att *expr.AttributeExpr) {
			if openAPIOnly {
				e.OpenAPIBody = att
			} else {
				e.Body = att
			}
		}
		kind = "Response"
	default:
		eval.IncompatibleDSL()
		return
	}

	var (
		attr *expr.AttributeExpr
		fn   func()
	)
	switch a := args[0].(type) {
	case string:
		if ref == nil {
			eval.ReportError("Body is set but %s is not defined", kind)
			return
		}
		if !expr.IsObject(ref.Type) {
			eval.ReportError("%s type must be an object with an attribute with name %#v, got %T", kind, a, ref.Type)
			return
		}
		attr = ref.Find(a)
		if attr == nil {
			eval.ReportError("%s type does not have an attribute named %#v", kind, a)
			return
		}
		attr = expr.DupAtt(attr)
		attr.AddMeta("origin:attribute", a)
		if rt, ok := attr.Type.(*expr.ResultTypeExpr); ok && expr.IsArray(rt.Type) {
			expr.GeneratedResultTypes.Append(rt)
		}
		if len(args) > 1 {
			var ok bool
			fn, ok = args[1].(func())
			if !ok {
				eval.InvalidArgError("function", args[1])
				return
			}
		}
	case expr.UserType:
		attr = &expr.AttributeExpr{Type: a}
		if len(args) > 1 {
			var ok bool
			fn, ok = args[1].(func())
			if !ok {
				eval.InvalidArgError("function", args[1])
				return
			}
		}
	case expr.DataType:
		if !openAPIOnly {
			eval.InvalidArgError("attribute name, user type or DSL", a)
			return
		}
		attr = &expr.AttributeExpr{Type: a}
		if len(args) > 1 {
			var ok bool
			fn, ok = args[1].(func())
			if !ok {
				eval.InvalidArgError("function", args[1])
				return
			}
		}
	case func():
		fn = a
		if ref == nil {
			eval.ReportError("Body is set but Payload is not defined")
			return
		}
		attr = &expr.AttributeExpr{References: []expr.DataType{ref.Type}}
	default:
		if openAPIOnly {
			eval.InvalidArgError("attribute name, data type, user type or DSL", a)
			return
		}
		eval.InvalidArgError("attribute name, user type or DSL", a)
		return
	}

	if fn != nil {
		eval.Execute(fn, attr)
	}
	if openAPIOnly {
		attr.AddMeta("http:openapi:body")
	} else {
		attr.AddMeta("http:body")
	}
	setter(attr)
}
