package codegen

import (
	"net/http"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// nameUnionBodies gives every request, streaming and success response body
// whose type is a union a Go type name owned by the endpoint and the body,
// like the name of an object body: "<Endpoint>RequestBody",
// "<Endpoint>StreamingBody" and "<Endpoint>[<Status>]ResponseBody". Without
// it the request and response bodies of an endpoint, and the bodies of every
// endpoint that uses the same union, share one Go type and one constructor
// although their layouts differ. The design names the union after the
// service type, and a named union body keeps that name after the transport
// IR unwraps its body wrapper.
//
// Only the Go transport IR of the HTTP code generator is renamed: the bodies
// are copied, and the evaluated design and the OpenAPI document are not
// affected.
func nameUnionBodies(endpoints []*transportir.Endpoint) {
	for _, endpoint := range endpoints {
		if endpoint.Request != nil {
			endpoint.Request.Body = namedUnionBody(endpoint.Request.Body, endpoint.Name+"RequestBody")
			endpoint.Request.StreamingBody = namedUnionBody(endpoint.Request.StreamingBody, endpoint.Name+"StreamingBody")
		}
		if endpoint.Response == nil {
			continue
		}
		for _, response := range endpoint.Response.Responses {
			name := endpoint.Name
			if len(endpoint.Response.Responses) > 1 {
				name += http.StatusText(response.StatusCode)
			}
			response.Body = namedUnionBody(response.Body, name+"ResponseBody")
		}
	}
}

// namedUnionBody returns a copy of body named name when body is a union and
// body itself otherwise.
func namedUnionBody(body *expr.AttributeExpr, name string) *expr.AttributeExpr {
	if body == nil {
		return nil
	}
	if _, ok := body.Type.(*expr.Union); !ok {
		return body
	}
	named := expr.DupAtt(body)
	union := named.Type.(*expr.Union)
	union.TypeName = name
	union.ExplicitTypeName = true
	return named
}
