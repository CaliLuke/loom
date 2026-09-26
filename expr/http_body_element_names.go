package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
)

type (
	// bodyElementNames checks the JSON names of the fields of the bodies of
	// one endpoint.
	bodyElementNames struct {
		endpoint *HTTPEndpointExpr
		verr     *eval.ValidationErrors
		// visited lists the IDs of the user types already checked.
		visited map[string]struct{}
		// reported lists the messages already added to verr.
		reported map[string]struct{}
	}

	// bodyField is a field of a body object and its JSON name.
	bodyField struct {
		key  string
		wire string
		// mapped is true when the element name suffix of key sets the JSON
		// name, so that it differs from the JSON name of the service field.
		mapped bool
	}
)

// responseBodySource returns the attribute that the body of the response
// resp is built from, with the element name suffixes of its keys, and the type
// of attr when the attribute holds the attributes of attr that resp maps to no
// header or cookie. It returns nil when the response has no body.
func responseBodySource(attr *AttributeExpr, resp *HTTPResponseExpr) (*AttributeExpr, DataType) {
	if resp == nil || attr == nil || attr.Type == Empty {
		return nil, nil
	}
	if resp.Body != nil {
		return resp.Body, nil
	}
	if !IsObject(attr.Type) {
		if !resp.Headers.IsEmpty() || len(resp.Cookies) > 0 {
			return nil, nil
		}
		return attr, nil
	}
	return responseBodyObject(attr, resp).Attribute(), attr.Type
}

// bodyFieldName returns the JSON name of the field of a body declared with the
// object key key.
func bodyFieldName(key string, att *AttributeExpr) bodyField {
	wire := JSONFieldName(ElementName(key), att)
	return bodyField{
		key:    key,
		wire:   wire,
		mapped: wire != JSONFieldName(AttributeName(key), att),
	}
}

// validateBodyElementNames rejects the fields of the HTTP and JSON-RPC bodies
// of the endpoint whose element name, such as "m" for a field declared as
// "n:m", gives them a JSON name that the body cannot use: an empty name, "-",
// a name that validJSONWireName rejects, or the JSON name of another field of
// the same object. A json struct tag takes precedence over the element name.
// The service types name the field after its attribute name and gRPC ignores
// the element name, so only the objects of the bodies are checked, without
// the attributes that the endpoint maps to params, headers and cookies.
func (e *HTTPEndpointExpr) validateBodyElementNames(verr *eval.ValidationErrors) {
	c := &bodyElementNames{
		endpoint: e,
		verr:     verr,
		visited:  make(map[string]struct{}),
		reported: make(map[string]struct{}),
	}
	transport := "HTTP"
	if e.IsJSONRPC() {
		transport = "JSON-RPC"
	}
	body, owner := e.requestBodySource()
	c.check(transport+" request body", body, owner)
	if e.MethodExpr.IsStreaming() && e.MethodExpr.Stream != ServerStreamKind {
		c.check(transport+" streaming body", e.MethodExpr.StreamingPayload, nil)
	}
	c.check("OpenAPI request body", e.OpenAPIRequestBody, nil)
	for _, response := range e.Responses {
		body, owner := responseBodySource(e.MethodExpr.Result, response)
		c.check(transport+" response body", body, owner)
		c.check("OpenAPI response body", response.OpenAPIBody, nil)
	}
	for _, httpError := range e.HTTPErrors {
		designError := httpError.designError()
		if designError == nil {
			continue
		}
		body, owner := responseBodySource(designError.AttributeExpr, httpError.Response)
		c.check(fmt.Sprintf("%s %q error response body", transport, httpError.Name), body, owner)
	}
}

// requestBodySource returns the attribute that the request body of the
// endpoint is built from, with the element name suffixes of its keys, and the
// type of the payload when the attribute holds the payload attributes that
// the endpoint maps to no param, header or cookie. It returns nil when the
// request has no body.
func (e *HTTPEndpointExpr) requestBodySource() (*AttributeExpr, DataType) {
	if e.Body != nil {
		return e.Body, nil
	}
	payload := e.MethodExpr.Payload
	if payload == nil || payload.Type == Empty {
		return nil, nil
	}
	if !IsObject(payload.Type) {
		if !httpBodyOnly(e) {
			return nil, nil
		}
		return payload, nil
	}
	return objectHTTPBody(e, payload).Attribute(), payload.Type
}

// check checks the objects of the body att, named body in the messages. owner
// is the type whose attributes the top-level object of att holds, if any.
func (c *bodyElementNames) check(body string, att *AttributeExpr, owner DataType) {
	if att == nil {
		return
	}
	var typeName string
	if ut, ok := owner.(UserType); ok {
		typeName = ut.Name()
	}
	c.walk(body, att.Type, typeName)
}

func (c *bodyElementNames) walk(body string, dt DataType, typeName string) {
	switch actual := dt.(type) {
	case UserType:
		if _, ok := c.visited[actual.ID()]; ok {
			return
		}
		c.visited[actual.ID()] = struct{}{}
		c.walk(body, actual.Attribute().Type, actual.Name())
	case *Object:
		c.checkObject(body, actual, typeName)
		for _, nat := range *actual {
			c.walk(body, nat.Attribute.Type, "")
		}
	case *Array:
		c.walk(body, actual.ElemType.Type, "")
	case *Map:
		c.walk(body, actual.KeyType.Type, "")
		c.walk(body, actual.ElemType.Type, "")
	case *Union:
		for _, nat := range actual.Values {
			c.walk(body, nat.Attribute.Type, "")
		}
	}
}

// checkObject checks the JSON names of the fields of obj, a type named
// typeName, if any, in body. The object validation checks the fields whose
// JSON name the element name does not set, so only the fields that it sets
// are checked, and their collisions with any other field.
func (c *bodyElementNames) checkObject(body string, obj *Object, typeName string) {
	var of string
	if typeName != "" {
		of = fmt.Sprintf(" of type %q", typeName)
	}
	fields := make(map[string]bodyField, len(*obj))
	for _, nat := range *obj {
		field := bodyFieldName(nat.Name, nat.Attribute)
		if field.mapped {
			switch {
			case field.wire == "":
				c.report("attribute %q%s has an empty element name, so the %s cannot name its JSON field; write the element name after the colon or remove the colon", field.key, of, body)
				continue
			case field.wire == "-":
				c.report("attribute %q%s has the element name \"-\", which would omit the field from the %s; use another element name", field.key, of, body)
				continue
			case !validJSONWireName(field.wire):
				c.report("attribute %q%s in the %s: %s", field.key, of, body, invalidJSONWireNameMessage(field.wire))
				continue
			}
		}
		if field.wire == "-" {
			continue
		}
		first, exists := fields[field.wire]
		if !exists {
			fields[field.wire] = field
			continue
		}
		if first.mapped || field.mapped {
			c.report("attributes %q and %q%s both use the JSON name %q in the %s; give one of them another element name after the colon or a struct:tag:json name", first.key, field.key, of, field.wire, body)
		}
	}
}

func (c *bodyElementNames) report(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if _, ok := c.reported[message]; ok {
		return
	}
	c.reported[message] = struct{}{}
	c.verr.Add(c.endpoint, "%s", message)
}
