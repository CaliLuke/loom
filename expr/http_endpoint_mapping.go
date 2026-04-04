package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
)

// PathParams computes a mapped attribute containing the subset of e.Params that
// describe path parameters.
func (e *HTTPEndpointExpr) PathParams() *MappedAttributeExpr {
	obj := Object{}
	v := &ValidationExpr{}
	pat := e.Params.Attribute() // need "attribute:name" style keys
	for _, r := range e.Routes {
		for _, p := range r.Params() {
			att := pat.Find(p)
			if att == nil {
				continue
			}
			obj.Set(p, att)
			if e.Params.IsRequired(p) {
				v.AddRequired(p)
			}
		}
	}
	at := &AttributeExpr{Type: &obj, Validation: v}
	return NewMappedAttributeExpr(at)
}

// QueryParams computes a mapped attribute containing the subset of e.Params
// that describe query parameters.
func (e *HTTPEndpointExpr) QueryParams() *MappedAttributeExpr {
	obj := Object{}
	v := &ValidationExpr{}
	pp := make(map[string]struct{})
	for _, r := range e.Routes {
		for _, p := range r.Params() {
			pp[p] = struct{}{}
		}
	}
	pat := e.Params.Attribute() // need "attribute:name" style keys
	for _, at := range *(pat.Type.(*Object)) {
		found := false
		for n := range pp {
			if n == at.Name {
				found = true
				break
			}
		}
		if !found {
			obj.Set(at.Name, at.Attribute)
			attName := splitMappedAttributeName(at.Name)
			if e.Params.IsRequired(attName) {
				v.AddRequired(attName)
			}
		}
	}
	at := &AttributeExpr{Type: &obj, Validation: v}
	return NewMappedAttributeExpr(at)
}

func splitMappedAttributeName(name string) string {
	for i := 0; i < len(name); i++ {
		if name[i] == ':' {
			return name[:i]
		}
	}
	return name
}

// validateParams checks the endpoint parameters are of an allowed type and the
// method payload contains the parameters.
func (e *HTTPEndpointExpr) validateParams() *eval.ValidationErrors {
	if e.Params.IsEmpty() {
		return nil
	}
	pparams, qparams := e.transportParamsForValidation()
	verr := new(eval.ValidationErrors)
	e.validateMappedParams(verr, pparams, true)
	e.validateMappedParams(verr, qparams, false)
	e.validatePayloadParamCompatibility(verr, pparams, qparams)
	return verr
}

// validateHeadersAndCookies makes sure headers and cookies are of an allowed
// type and the method payload defines the corresponding attributes.
func (e *HTTPEndpointExpr) validateHeadersAndCookies() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	headers, cookies := e.transportHeadersAndCookiesForValidation()
	e.validateHeaderTypes(verr, headers)
	e.validateCookieTypes(verr, cookies)
	e.validatePayloadHeaderCookieCompatibility(verr, headers, cookies)
	return verr
}

func (e *HTTPEndpointExpr) transportParamsForValidation() (*MappedAttributeExpr, *MappedAttributeExpr) {
	pparams := DupMappedAtt(e.PathParams())
	qparams := DupMappedAtt(e.QueryParams())
	initAttr(pparams, e.MethodExpr.Payload)
	initAttr(qparams, e.MethodExpr.Payload)
	return pparams, qparams
}

func (e *HTTPEndpointExpr) transportHeadersAndCookiesForValidation() (*MappedAttributeExpr, *MappedAttributeExpr) {
	headers := DupMappedAtt(e.Headers)
	cookies := DupMappedAtt(e.Cookies)
	initAttr(headers, e.MethodExpr.Payload)
	initAttr(cookies, e.MethodExpr.Payload)
	return headers, cookies
}

func (e *HTTPEndpointExpr) validateMappedParams(verr *eval.ValidationErrors, params *MappedAttributeExpr, path bool) {
	WalkMappedAttr(params, func(name, _ string, a *AttributeExpr) error { // nolint: errcheck
		switch {
		case e.invalidMappedParamType(a, path):
			verr.Add(e, "path parameter %s cannot be an object, path parameter types must be primitive, array or map (query string only)", name)
		case IsArray(a.Type):
			arr := AsArray(a.Type)
			if !IsPrimitive(arr.ElemType.Type) {
				if path {
					verr.Add(e, "elements of array path parameter %q must be primitive", name)
				} else {
					verr.Add(e, "elements of array query parameter %q must be primitive", name)
				}
			}
		default:
			if path {
				verr.Merge(a.Validate(fmt.Sprintf("path parameter %s", name), e))
			} else {
				verr.Merge(a.Validate(fmt.Sprintf("query parameter %s", name), e))
			}
		}
		return nil
	})
}

func (e *HTTPEndpointExpr) invalidMappedParamType(a *AttributeExpr, path bool) bool {
	if path {
		return IsObject(a.Type) || IsMap(a.Type) || IsUnion(a.Type)
	}
	return IsObject(a.Type) || IsUnion(a.Type)
}

func (e *HTTPEndpointExpr) validatePayloadParamCompatibility(verr *eval.ValidationErrors, pparams, qparams *MappedAttributeExpr) {
	if e.MethodExpr.Payload == nil {
		return
	}
	switch e.MethodExpr.Payload.Type.(type) {
	case *Object, UserType:
		e.validateMappedAttributesExist(verr, pparams, "Path parameter %q not found in payload.")
		e.validateMappedAttributesExist(verr, qparams, "Query string parameter %q not found in payload.")
	case *Array:
		if len(*AsObject(pparams.Type))+len(*AsObject(qparams.Type)) > 1 {
			verr.Add(e, "Payload type is array but HTTP endpoint defines multiple parameters. At most one parameter must be defined and it must be an array.")
		}
	case *Map:
		if len(*AsObject(pparams.Type))+len(*AsObject(qparams.Type)) > 1 {
			verr.Add(e, "Payload type is map but HTTP endpoint defines multiple parameters. At most one query string parameter must be defined and it must be a map.")
		}
	}
}

func (e *HTTPEndpointExpr) validateHeaderTypes(verr *eval.ValidationErrors, headers *MappedAttributeExpr) {
	WalkMappedAttr(headers, func(name, _ string, a *AttributeExpr) error { // nolint: errcheck
		switch {
		case IsObject(a.Type), IsUnion(a.Type):
			verr.Add(e, "header %q must be primitive or array", name)
		case IsArray(a.Type):
			arr := AsArray(a.Type)
			if !IsPrimitive(arr.ElemType.Type) {
				verr.Add(e, "elements of array header %q must be primitive", name)
			}
		default:
			verr.Merge(a.Validate(fmt.Sprintf("header %q", name), e))
		}
		return nil
	})
}

func (e *HTTPEndpointExpr) validateCookieTypes(verr *eval.ValidationErrors, cookies *MappedAttributeExpr) {
	WalkMappedAttr(cookies, func(name, _ string, a *AttributeExpr) error { // nolint: errcheck
		switch {
		case IsObject(a.Type), IsUnion(a.Type), IsArray(a.Type):
			verr.Add(e, "cookie %q must be primitive", name)
		default:
			verr.Merge(a.Validate(fmt.Sprintf("cookie %q", name), e))
		}
		return nil
	})
}

func (e *HTTPEndpointExpr) validatePayloadHeaderCookieCompatibility(verr *eval.ValidationErrors, headers, cookies *MappedAttributeExpr) {
	switch e.MethodExpr.Payload.Type.(type) {
	case *Object, UserType:
		hasBasicAuth := TaggedAttribute(e.MethodExpr.Payload, "security:username") != ""
		e.validateMappedAttributesExist(verr, headers, `header %q not found in payload.`)
		WalkMappedAttr(headers, func(name, elem string, _ *AttributeExpr) error { // nolint: errcheck
			if elem == "Authorization" && hasBasicAuth {
				verr.Add(e, "Attribute %q is mapped to \"Authorization\" header in the endpoint secured by BasicAuth which also sets \"Authorization\" header. Specify a different header to map attribute %q.", name, name)
			}
			return nil
		})
		WalkMappedAttr(cookies, func(name, _ string, _ *AttributeExpr) error { // nolint: errcheck
			if e.isTransportOnlySessionCookie(name) {
				return nil
			}
			if e.MethodExpr.Payload.Find(name) == nil {
				verr.Add(e, `cookie %q not found in payload.`, name)
			}
			return nil
		})
	case *Array:
		if len(*AsObject(headers.Type)) > 1 {
			verr.Add(e, "Payload type is array but HTTP endpoint defines multiple headers. At most one header must be defined and it must be an array.")
		}
	case *Map:
		if len(*AsObject(headers.Type))+len(*AsObject(cookies.Type)) > 0 {
			verr.Add(e, "Payload type is map but HTTP endpoint defines headers or cookies. Map payloads can only be decoded from HTTP request bodies or query strings.")
		}
	}
}

func (e *HTTPEndpointExpr) isTransportOnlySessionCookie(name string) bool {
	if e == nil || e.MethodExpr == nil || e.MethodExpr.Service == nil {
		return false
	}
	for _, sessionAuth := range e.MethodExpr.validationSessionAuths() {
		for _, transport := range sessionAuth.Transports {
			if transport == nil || transport.Kind != SessionCookieTransportKind || transport.PayloadOwned() {
				continue
			}
			if transport.TransportAttributeName() == name {
				return true
			}
		}
	}
	return false
}

func (e *HTTPEndpointExpr) validateMappedAttributesExist(verr *eval.ValidationErrors, attrs *MappedAttributeExpr, format string) {
	WalkMappedAttr(attrs, func(name, _ string, _ *AttributeExpr) error { // nolint: errcheck
		if e.MethodExpr.Payload.Find(name) == nil {
			verr.Add(e, format, name)
		}
		return nil
	})
}
