package expr

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// PathParams computes a mapped attribute containing the subset of e.Params that
// describe path parameters. A route wildcard names the transport element of a
// param: the route "/{k}" matches Param("key:k"), the path parameter of the
// key attribute.
func (e *HTTPEndpointExpr) PathParams() *MappedAttributeExpr {
	obj := Object{}
	v := &ValidationExpr{}
	for _, r := range e.Routes {
		for _, p := range r.Params() {
			name, ok := e.Params.attributeOfElem(p)
			if !ok {
				continue
			}
			key := name
			if p != name {
				key = name + ":" + p
			}
			obj.Set(key, AsObject(e.Params.Type).Attribute(name))
			if e.Params.IsRequired(name) {
				v.AddRequired(key)
			}
		}
	}
	at := &AttributeExpr{Type: &obj, Validation: v}
	return NewMappedAttributeExpr(at)
}

// QueryParams computes a mapped attribute containing the subset of e.Params
// that describe query parameters: the params whose transport element no route
// wildcard names.
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
		if _, found := pp[ElementName(at.Name)]; !found {
			obj.Set(at.Name, at.Attribute)
			attName := AttributeName(at.Name)
			if e.Params.IsRequired(attName) {
				v.AddRequired(attName)
			}
		}
	}
	at := &AttributeExpr{Type: &obj, Validation: v}
	return NewMappedAttributeExpr(at)
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
	location := httpencoding.NamedQuery
	if path {
		location = httpencoding.Path
	}
	validateMappedParameterCodecs(params, location, e, verr)
	for _, field := range *AsObject(params.Type) {
		verr.Merge(field.Attribute.Validate(fmt.Sprintf("%s %s", parameterLocationName(location), field.Name), e))
	}
}

func (e *HTTPEndpointExpr) validatePayloadParamCompatibility(verr *eval.ValidationErrors, pparams, qparams *MappedAttributeExpr) {
	if e.MethodExpr.Payload == nil {
		return
	}
	if IsObject(e.MethodExpr.Payload.Type) {
		e.validateMappedAttributesExist(verr, pparams, "Path parameter %q not found in payload.")
		e.validateMappedAttributesExist(verr, qparams, "Query string parameter %q not found in payload.")
	}
}

func (e *HTTPEndpointExpr) validateHeaderTypes(verr *eval.ValidationErrors, headers *MappedAttributeExpr) {
	validateHeaderCollections(headers, e.MethodExpr.Payload, e, verr)
	validateMappedParameterCodecs(headers, httpencoding.Header, e, verr)
	for _, field := range *AsObject(headers.Type) {
		verr.Merge(field.Attribute.Validate(fmt.Sprintf("header %q", field.Name), e))
	}
}

func (e *HTTPEndpointExpr) validateCookieTypes(verr *eval.ValidationErrors, cookies *MappedAttributeExpr) {
	validateMappedParameterCodecs(cookies, httpencoding.Cookie, e, verr)
	for _, field := range *AsObject(cookies.Type) {
		verr.Merge(field.Attribute.Validate(fmt.Sprintf("cookie %q", field.Name), e))
	}
}

func (e *HTTPEndpointExpr) validatePayloadHeaderCookieCompatibility(verr *eval.ValidationErrors, headers, cookies *MappedAttributeExpr) {
	if IsObject(e.MethodExpr.Payload.Type) {
		e.validateMappedAttributesExist(verr, headers, `header %q not found in payload.`)
		WalkMappedAttr(cookies, func(name, _ string, _ *AttributeExpr) error { // nolint: errcheck
			if e.isTransportOnlySessionCookie(name) {
				return nil
			}
			if _, att := e.MethodExpr.Payload.FindAttribute(name); att == nil {
				verr.Add(e, `cookie %q not found in payload.`, name)
			}
			return nil
		})
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
		if _, att := e.MethodExpr.Payload.FindAttribute(name); att == nil {
			verr.Add(e, format, name)
		}
		return nil
	})
}
