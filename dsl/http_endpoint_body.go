package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
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

	ref, setter, kind, ok := resolveBodyContext(openAPIOnly)
	if !ok {
		return
	}

	attr, fn, ok := resolveBodyAttribute(args, ref, kind, openAPIOnly)
	if !ok {
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

func resolveBodyContext(openAPIOnly bool) (*expr.AttributeExpr, func(*expr.AttributeExpr), string, bool) {
	switch e := eval.Current().(type) {
	case *expr.HTTPEndpointExpr:
		if openAPIOnly {
			eval.IncompatibleDSL()
			return nil, nil, "", false
		}
		return e.MethodExpr.Payload, func(att *expr.AttributeExpr) {
			e.Body = att
		}, "Request", true
	case *expr.HTTPErrorExpr:
		kind := "Error"
		if e.Name != "" {
			kind += " " + e.Name
		}
		return e.AttributeExpr, func(att *expr.AttributeExpr) {
			if e.Response == nil {
				e.Response = &expr.HTTPResponseExpr{}
			}
			if openAPIOnly {
				e.Response.OpenAPIBody = att
			} else {
				e.Response.Body = att
			}
		}, kind, true
	case *expr.HTTPResponseExpr:
		p, ok := e.Parent.(*expr.HTTPEndpointExpr)
		if !ok {
			eval.IncompatibleDSL()
			return nil, nil, "", false
		}
		return p.MethodExpr.Result, func(att *expr.AttributeExpr) {
			if openAPIOnly {
				e.OpenAPIBody = att
			} else {
				e.Body = att
			}
		}, "Response", true
	default:
		eval.IncompatibleDSL()
		return nil, nil, "", false
	}
}

func resolveBodyAttribute(args []any, ref *expr.AttributeExpr, kind string, openAPIOnly bool) (*expr.AttributeExpr, func(), bool) {
	fn, ok := bodyDSLFunc(args)
	if !ok {
		return nil, nil, false
	}

	switch a := args[0].(type) {
	case string:
		attr, ok := resolveNamedBodyAttribute(ref, kind, a)
		return attr, fn, ok
	case expr.UserType:
		return &expr.AttributeExpr{Type: a}, fn, true
	case expr.DataType:
		if !openAPIOnly {
			eval.InvalidArgError("attribute name, user type or DSL", a)
			return nil, nil, false
		}
		return &expr.AttributeExpr{Type: a}, fn, true
	case func():
		if ref == nil {
			eval.ReportError("Body is set but Payload is not defined")
			return nil, nil, false
		}
		return &expr.AttributeExpr{References: []expr.DataType{ref.Type}}, a, true
	default:
		if openAPIOnly {
			eval.InvalidArgError("attribute name, data type, user type or DSL", a)
		} else {
			eval.InvalidArgError("attribute name, user type or DSL", a)
		}
		return nil, nil, false
	}
}

func bodyDSLFunc(args []any) (func(), bool) {
	if len(args) == 1 {
		return nil, true
	}
	fn, ok := args[1].(func())
	if !ok {
		eval.InvalidArgError("function", args[1])
		return nil, false
	}
	return fn, true
}

func resolveNamedBodyAttribute(ref *expr.AttributeExpr, kind, name string) (*expr.AttributeExpr, bool) {
	if ref == nil {
		eval.ReportError("Body is set but %s is not defined", kind)
		return nil, false
	}
	if !expr.IsObject(ref.Type) {
		eval.ReportError("%s type must be an object with an attribute with name %#v, got %T", kind, name, ref.Type)
		return nil, false
	}
	attr := ref.Find(name)
	if attr == nil {
		eval.ReportError("%s type does not have an attribute named %#v", kind, name)
		return nil, false
	}
	attr = expr.DupAtt(attr)
	attr.AddMeta("origin:attribute", name)
	if rt, ok := attr.Type.(*expr.ResultTypeExpr); ok && expr.IsArray(rt.Type) {
		expr.GeneratedResultTypes.Append(rt)
	}
	return attr, true
}
