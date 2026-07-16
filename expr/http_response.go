package expr

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

const (
	StatusContinue           = 100 // RFC 7231, 6.2.1
	StatusSwitchingProtocols = 101 // RFC 7231, 6.2.2
	StatusProcessing         = 102 // RFC 2518, 10.1

	StatusOK                   = 200 // RFC 7231, 6.3.1
	StatusCreated              = 201 // RFC 7231, 6.3.2
	StatusAccepted             = 202 // RFC 7231, 6.3.3
	StatusNonAuthoritativeInfo = 203 // RFC 7231, 6.3.4
	StatusNoContent            = 204 // RFC 7231, 6.3.5
	StatusResetContent         = 205 // RFC 7231, 6.3.6
	StatusPartialContent       = 206 // RFC 7233, 4.1
	StatusMultiStatus          = 207 // RFC 4918, 11.1
	StatusAlreadyReported      = 208 // RFC 5842, 7.1
	StatusIMUsed               = 226 // RFC 3229, 10.4.1

	StatusMultipleChoices  = 300 // RFC 7231, 6.4.1
	StatusMovedPermanently = 301 // RFC 7231, 6.4.2
	StatusFound            = 302 // RFC 7231, 6.4.3
	StatusSeeOther         = 303 // RFC 7231, 6.4.4
	StatusNotModified      = 304 // RFC 7232, 4.1
	StatusUseProxy         = 305 // RFC 7231, 6.4.5

	StatusTemporaryRedirect = 307 // RFC 7231, 6.4.7
	StatusPermanentRedirect = 308 // RFC 7538, 3

	StatusBadRequest                   = 400 // RFC 7231, 6.5.1
	StatusUnauthorized                 = 401 // RFC 7235, 3.1
	StatusPaymentRequired              = 402 // RFC 7231, 6.5.2
	StatusForbidden                    = 403 // RFC 7231, 6.5.3
	StatusNotFound                     = 404 // RFC 7231, 6.5.4
	StatusMethodNotAllowed             = 405 // RFC 7231, 6.5.5
	StatusNotAcceptable                = 406 // RFC 7231, 6.5.6
	StatusProxyAuthRequired            = 407 // RFC 7235, 3.2
	StatusRequestTimeout               = 408 // RFC 7231, 6.5.7
	StatusConflict                     = 409 // RFC 7231, 6.5.8
	StatusGone                         = 410 // RFC 7231, 6.5.9
	StatusLengthRequired               = 411 // RFC 7231, 6.5.10
	StatusPreconditionFailed           = 412 // RFC 7232, 4.2
	StatusRequestEntityTooLarge        = 413 // RFC 7231, 6.5.11
	StatusRequestURITooLong            = 414 // RFC 7231, 6.5.12
	StatusUnsupportedMediaType         = 415 // RFC 7231, 6.5.13
	StatusRequestedRangeNotSatisfiable = 416 // RFC 7233, 4.4
	StatusExpectationFailed            = 417 // RFC 7231, 6.5.14
	StatusTeapot                       = 418 // RFC 7168, 2.3.3
	StatusUnprocessableEntity          = 422 // RFC 4918, 11.2
	StatusLocked                       = 423 // RFC 4918, 11.3
	StatusFailedDependency             = 424 // RFC 4918, 11.4
	StatusUpgradeRequired              = 426 // RFC 7231, 6.5.15
	StatusPreconditionRequired         = 428 // RFC 6585, 3
	StatusTooManyRequests              = 429 // RFC 6585, 4
	StatusRequestHeaderFieldsTooLarge  = 431 // RFC 6585, 5
	StatusUnavailableForLegalReasons   = 451 // RFC 7725, 3

	StatusInternalServerError           = 500 // RFC 7231, 6.6.1
	StatusNotImplemented                = 501 // RFC 7231, 6.6.2
	StatusBadGateway                    = 502 // RFC 7231, 6.6.3
	StatusServiceUnavailable            = 503 // RFC 7231, 6.6.4
	StatusGatewayTimeout                = 504 // RFC 7231, 6.6.5
	StatusHTTPVersionNotSupported       = 505 // RFC 7231, 6.6.6
	StatusVariantAlsoNegotiates         = 506 // RFC 2295, 8.1
	StatusInsufficientStorage           = 507 // RFC 4918, 11.5
	StatusLoopDetected                  = 508 // RFC 5842, 7.2
	StatusNotExtended                   = 510 // RFC 2774, 7
	StatusNetworkAuthenticationRequired = 511 // RFC 6585, 6
)

const (
	RPCParseError     = -32700 // JSON-RPC 2.0, 5.1
	RPCInvalidRequest = -32600
	RPCMethodNotFound = -32601
	RPCInvalidParams  = -32602
	RPCInternalError  = -32603
)

type (
	// HTTPResponseExpr defines a HTTP response including its status code,
	// headers and result type.
	HTTPResponseExpr struct {
		// HTTP status
		StatusCode int
		// Response description
		Description string
		// Headers describe the HTTP response headers.
		Headers *MappedAttributeExpr
		// Cookies describe the HTTP response cookies.
		Cookies []*HTTPResponseCookieExpr
		// Response body if any
		Body *AttributeExpr
		// OpenAPIBody describes a documentation-only response body used by
		// OpenAPI generation when runtime response encoding/decoding is skipped.
		OpenAPIBody *AttributeExpr
		// Response Content-Type header value
		ContentType string
		// Tag the value a field of the result must have for this
		// response to be used.
		Tag [2]string
		// Parent expression, one of EndpointExpr, ServiceExpr or
		// RootExpr.
		Parent eval.Expression
		// Meta is a list of key/value pairs
		Meta MetaExpr
		// Links describes the OpenAPI links emitted for this response.
		Links []*HTTPResponseLinkExpr
		// currentCookie tracks the cookie currently configured by the DSL.
		currentCookie *HTTPResponseCookieExpr
	}
)

// EvalName returns the generic definition name used in error messages.
func (r *HTTPResponseExpr) EvalName() string {
	var suffix string
	if r.Parent != nil {
		suffix = fmt.Sprintf(" of %s", r.Parent.EvalName())
	}
	return "HTTP response" + suffix
}

// Prepare makes sure the response is initialized even if not done explicitly
// by
func (r *HTTPResponseExpr) Prepare() {
	if r.Headers == nil {
		r.Headers = NewEmptyMappedAttributeExpr()
	}
}

// Validate checks that the response definition is consistent: its status is set
// and the result type definition if any is valid.
func (r *HTTPResponseExpr) Validate(e *HTTPEndpointExpr) *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	resultType := newHTTPResponseResultType(e)

	r.validateStatusAndBody(e, verr)
	r.validateTextContentType(e, verr)
	r.validateHeadersAndCookies(e, resultType, verr)
	r.validateBodyAndLinks(e, resultType, verr)
	return verr
}

func (r *HTTPResponseExpr) validateStatusAndBody(e *HTTPEndpointExpr, verr *eval.ValidationErrors) {
	if r.StatusCode == 0 {
		verr.Add(r, "HTTP response status not defined")
		return
	}
	if bodyAllowedForStatus(r.StatusCode) || e.MethodExpr.IsStreaming() {
		return
	}
	ep, ok := r.Parent.(*HTTPEndpointExpr)
	if ok && httpResponseBody(ep, r).Type != Empty {
		verr.Add(r, "Response body defined for status code %d which does not allow response body.", r.StatusCode)
	}
}

func (r *HTTPResponseExpr) validateTextContentType(e *HTTPEndpointExpr, verr *eval.ValidationErrors) {
	if (r.ContentType != "text/html" && r.ContentType != "text/plain") || e.SkipRequestBodyEncodeDecode {
		return
	}
	if r.OpenAPIBody != nil && r.OpenAPIBody.Type != String && r.OpenAPIBody.Type != Bytes {
		verr.Add(r, "Result type must be String or Bytes when ContentType is '%s'", r.ContentType)
	}
	if r.OpenAPIBody == nil && e.MethodExpr.Result.Type != nil && e.MethodExpr.Result.Type != String && e.MethodExpr.Result.Type != Bytes && r.Body == nil {
		verr.Add(r, "Result type must be String or Bytes when ContentType is '%s'", r.ContentType)
	}
	if r.Body != nil && r.Body.Type != String && r.Body.Type != Bytes {
		verr.Add(r, "Result type must be String or Bytes when ContentType is '%s'", r.ContentType)
	}
}

func (r *HTTPResponseExpr) validateHeadersAndCookies(e *HTTPEndpointExpr, resultType *httpResponseResultType, verr *eval.ValidationErrors) {
	if !r.Headers.IsEmpty() {
		r.validateHeaders(e, resultType, verr)
	}
	if len(r.Cookies) > 0 {
		r.validateCookies(e, resultType, verr)
	}
}

func (r *HTTPResponseExpr) validateHeaders(e *HTTPEndpointExpr, resultType *httpResponseResultType, verr *eval.ValidationErrors) {
	verr.Merge(r.Headers.Validate("HTTP response headers", r))
	switch {
	case isEmpty(e.MethodExpr.Result):
		verr.Add(r, "response defines headers but result is empty")
	case IsObject(e.MethodExpr.Result.Type):
		mobj := AsObject(r.Headers.Type)
		for _, header := range *mobj {
			t := resultType.AttributeType(header.Name)
			switch {
			case t == nil:
				verr.Add(r, "header %q has no equivalent attribute in%s result type, use notation 'attribute_name:header_name' to identify corresponding result type attribute.", header.Name, resultType.InView)
			case IsArray(t):
				if !IsPrimitive(AsArray(t).ElemType.Type) {
					verr.Add(e, "attribute %q used in HTTP headers must be a primitive type or an array of primitive types.", header.Name)
				}
			case !IsPrimitive(t):
				verr.Add(e, "attribute %q used in HTTP headers must be a primitive type or an array of primitive types.", header.Name)
			}
		}
	case len(*AsObject(r.Headers.Type)) > 1:
		verr.Add(r, "response defines more than one headers but result type is not an object")
	case IsArray(e.MethodExpr.Result.Type):
		if !IsPrimitive(AsArray(e.MethodExpr.Result.Type).ElemType.Type) {
			verr.Add(e, "Array result is mapped to an HTTP header but is not an array of primitive types.")
		}
	}
}

func (r *HTTPResponseExpr) validateCookies(e *HTTPEndpointExpr, resultType *httpResponseResultType, verr *eval.ValidationErrors) {
	seenNames := make(map[string]struct{}, len(r.Cookies))
	seenAttrs := make(map[string]struct{}, len(r.Cookies))
	switch {
	case isEmpty(e.MethodExpr.Result):
		verr.Add(r, "response defines cookies but result is empty")
	case IsObject(e.MethodExpr.Result.Type):
		for _, cookie := range r.Cookies {
			verr.Merge(cookie.Validate("HTTP response cookie", r))
			httpName := cookie.HTTPName()
			if _, ok := seenNames[httpName]; ok {
				verr.Add(r, "response defines duplicate cookie %q", httpName)
			} else {
				seenNames[httpName] = struct{}{}
			}
			if attrName := cookie.AttributeName(); attrName != "" {
				if _, ok := seenAttrs[attrName]; ok {
					verr.Add(r, "response defines duplicate cookie mapping for attribute %q", attrName)
				} else {
					seenAttrs[attrName] = struct{}{}
				}
			}
			t := resultType.AttributeType(cookie.AttributeName())
			if t == nil {
				verr.Add(r, "cookie %q has no equivalent attribute in%s result type, use notation 'attribute_name:cookie_name' to identify corresponding result type attribute.", httpName, resultType.InView)
			}
			if !IsPrimitive(t) {
				verr.Add(e, "attribute %q used in HTTP cookies must be a primitive type.", cookie.AttributeName())
			}
		}
	default:
		for _, cookie := range r.Cookies {
			verr.Merge(cookie.Validate("HTTP response cookie", r))
		}
		if len(r.Cookies) > 1 {
			verr.Add(r, "response defines more than one cookies but result type is not an object")
		} else if IsArray(e.MethodExpr.Result.Type) {
			verr.Add(e, "Array result is mapped to an HTTP cookie.")
		}
	}
}

func (r *HTTPResponseExpr) validateBodyAndLinks(e *HTTPEndpointExpr, resultType *httpResponseResultType, verr *eval.ValidationErrors) {
	if r.Body != nil {
		verr.Merge(r.Body.Validate("HTTP response body", r))
		if e.SkipResponseBodyEncodeDecode {
			verr.Add(r, "Cannot define a response body when endpoint uses SkipResponseBodyEncodeDecode.")
		}
		if att, ok := r.Body.Meta["origin:attribute"]; ok {
			if resultType.AttributeType(att[0]) == nil {
				verr.Add(r, "body %q has no equivalent attribute in%s result type", att[0], resultType.InView)
			}
		} else if bobj := AsObject(r.Body.Type); bobj != nil {
			for _, n := range *bobj {
				if resultType.AttributeType(n.Name) == nil {
					verr.Add(r, "body %q has no equivalent attribute in%s result type", n.Name, resultType.InView)
				}
			}
		}
	} else if e.SkipResponseBodyEncodeDecode {
		body := httpResponseBody(e, r)
		if body.Type != Empty {
			verr.Add(e, "HTTP endpoint response body must be empty when using SkipResponseBodyEncodeDecode. Make sure to define headers and cookies as needed.")
		}
	}
	if r.OpenAPIBody != nil {
		verr.Merge(r.OpenAPIBody.Validate("HTTP response OpenAPI body", r))
	}
	if len(r.Links) > 0 {
		seen := make(map[string]struct{}, len(r.Links))
		for _, link := range r.Links {
			if link == nil {
				verr.Add(r, "response defines a nil link")
				continue
			}
			verr.Merge(link.Validate())
			if _, ok := seen[link.Name]; ok {
				verr.Add(r, "response defines duplicate link %q", link.Name)
				continue
			}
			seen[link.Name] = struct{}{}
		}
	}
}

type httpResponseResultType struct {
	AttributeType func(string) DataType
	InView        string
}

func newHTTPResponseResultType(e *HTTPEndpointExpr) *httpResponseResultType {
	rt, isrt := e.MethodExpr.Result.Type.(*ResultTypeExpr)
	resultType := &httpResponseResultType{}
	resultType.AttributeType = func(name string) DataType {
		if !IsObject(e.MethodExpr.Result.Type) {
			return nil
		}
		if isrt {
			if view, ok := e.MethodExpr.Result.Meta.Last(ViewMetaKey); ok {
				v := rt.View(view)
				if v == nil {
					return nil
				}
				return v.AttributeExpr.Find(name).Type
			}
			for _, v := range rt.Views {
				if !rt.ViewHasAttribute(v.Name, name) {
					return nil
				}
			}
		}
		att := e.MethodExpr.Result.Find(name)
		if att == nil || att.Type == nil {
			return nil
		}
		return att.Type
	}
	if isrt {
		resultType.InView = " all views of"
	}
	return resultType
}

// Finalize sets the response result type from its type if the type is a result
// type and no result type is already specified.
func (r *HTTPResponseExpr) Finalize(a *HTTPEndpointExpr, svcAtt *AttributeExpr) {
	r.Parent = a

	if r.Body != nil && r.Body.Type != Empty {
		r.finalizeBody(a, svcAtt)
	}
	if r.OpenAPIBody != nil {
		r.finalizeOpenAPIBody(a)
	}
	initAttr(r.Headers, svcAtt)
	initResponseCookies(r.Cookies, svcAtt)
}

func (r *HTTPResponseExpr) finalizeBody(a *HTTPEndpointExpr, svcAtt *AttributeExpr) {
	bodyAtt := responseBodyAttribute(r.Body, svcAtt)
	if body := AsObject(r.Body.Type); body != nil {
		r.finalizeObjectBody(a, bodyAtt, body)
	}
	if r.Body.Meta == nil {
		r.Body.Meta = bodyAtt.Meta
	}
}

func responseBodyAttribute(body, svcAtt *AttributeExpr) *AttributeExpr {
	if origin, ok := body.Meta["origin:attribute"]; ok {
		return svcAtt.Find(origin[0])
	}
	return svcAtt
}

func (r *HTTPResponseExpr) finalizeObjectBody(a *HTTPEndpointExpr, bodyAtt *AttributeExpr, body *Object) {
	bodyObj := AsObject(bodyAtt.Type)
	for _, nat := range *body {
		name := strings.Split(nat.Name, ":")[0]
		source, required := responseBodyFieldSource(bodyAtt, bodyObj, name)
		initAttrFromDesign(nat.Attribute, source)
		if required {
			ensureValidation(r.Body).AddRequired(name)
		}
	}
	r.rememberOriginalBodyName()
	r.wrapBodyUserType(a)
	r.propagateOpenAPITypename(bodyAtt)
}

func responseBodyFieldSource(bodyAtt *AttributeExpr, bodyObj *Object, name string) (*AttributeExpr, bool) {
	if bodyObj != nil {
		return bodyObj.Attribute(name), bodyAtt.IsRequired(name)
	}
	return bodyAtt, bodyAtt.Type != Empty
}

func (r *HTTPResponseExpr) rememberOriginalBodyName() {
	if t, ok := r.Body.Type.(UserType); ok {
		t.Attribute().AddMeta("name:original", t.Name())
	}
}

func (r *HTTPResponseExpr) wrapBodyUserType(a *HTTPEndpointExpr) {
	r.Body.Type = &UserTypeExpr{
		AttributeExpr: DupAtt(r.Body),
		TypeName:      fmt.Sprintf("%s%sResponseBody", a.Service.Name(), a.Name()),
	}
}

func (r *HTTPResponseExpr) propagateOpenAPITypename(bodyAtt *AttributeExpr) {
	ut, ok := bodyAtt.Type.(UserType)
	if !ok {
		return
	}
	name, ok := ut.Attribute().Meta.Last("openapi:typename")
	if !ok || strings.TrimSpace(name) == "" {
		return
	}
	r.Body.AddMeta("openapi:typename", name)
	if utBody, ok := r.Body.Type.(UserType); ok {
		utBody.Attribute().AddMeta("openapi:typename", name)
	}
}

func ensureValidation(att *AttributeExpr) *ValidationExpr {
	if att.Validation == nil {
		att.Validation = &ValidationExpr{}
	}
	return att.Validation
}

func (r *HTTPResponseExpr) finalizeOpenAPIBody(a *HTTPEndpointExpr) {
	r.OpenAPIBody = httpOpenAPIResponseBody(a, r)
	r.OpenAPIBody.Finalize()
}

// Dup creates a copy of the response expression.
func (r *HTTPResponseExpr) Dup() *HTTPResponseExpr {
	res := HTTPResponseExpr{
		StatusCode:  r.StatusCode,
		Description: r.Description,
		ContentType: r.ContentType,
		Parent:      r.Parent,
		Meta:        r.Meta,
	}
	if r.Body != nil {
		res.Body = DupAtt(r.Body)
	}
	if r.OpenAPIBody != nil {
		res.OpenAPIBody = DupAtt(r.OpenAPIBody)
	}
	if r.Headers != nil {
		res.Headers = DupMappedAtt(r.Headers)
	}
	if len(r.Cookies) > 0 {
		res.Cookies = make([]*HTTPResponseCookieExpr, len(r.Cookies))
		for i, c := range r.Cookies {
			res.Cookies[i] = c.Dup()
		}
	}
	if len(r.Links) > 0 {
		res.Links = make([]*HTTPResponseLinkExpr, len(r.Links))
		for i, link := range r.Links {
			if link == nil {
				continue
			}
			dup := *link
			if len(link.Parameters) > 0 {
				dup.Parameters = make(map[string]string, len(link.Parameters))
				for name, expression := range link.Parameters {
					dup.Parameters[name] = expression
				}
			}
			res.Links[i] = &dup
		}
	}
	return &res
}

// AddCookie appends a response cookie to the response and marks it active for
// subsequent cookie attribute setters.
func (r *HTTPResponseExpr) AddCookie(cookie *HTTPResponseCookieExpr) {
	r.Cookies = append(r.Cookies, cookie)
	r.currentCookie = cookie
}

// CurrentCookie returns the cookie currently configured by the DSL.
func (r *HTTPResponseExpr) CurrentCookie() *HTTPResponseCookieExpr {
	return r.currentCookie
}

// mapUnmappedAttrs maps any unmapped attributes in ErrorResult type to the
// response headers. Unmapped attributes refer to the attributes in ErrorResult
// type that are not mapped to response body or headers. Such unmapped
// attributes are mapped to special Loom headers in the form of
// "Loom-Attribute(-<Attribute Name>)".
func (r *HTTPResponseExpr) mapUnmappedAttrs(svcAtt *AttributeExpr) {
	if svcAtt.Type != ErrorResult {
		return
	}

	// map attributes to headers that are not explicitly mapped
	switch {
	case IsObject(svcAtt.Type):
		// map the attribute names in the service type to response headers if
		// not mapped explicitly.

		var originAttr string
		if r.Body != nil {
			if o, ok := r.Body.Meta["origin:attribute"]; ok {
				originAttr = o[0]
			}
		}
		// if response body was mapped explicitly using Body(<attribute name>) then
		// we must make sure we map all the other unmapped attributes to headers.
		if r.Body == nil || r.Body.Type == Empty || originAttr != "" {
			for _, nat := range *(AsObject(svcAtt.Type)) {
				if originAttr == nat.Name {
					continue
				}
				if _, ok := r.Headers.FindKey(nat.Name); ok {
					continue
				}
				r.Headers.Type.(*Object).Set(nat.Name, nat.Attribute)
				r.Headers.Map("loom-attribute-"+nat.Name, nat.Name)
				if svcAtt.IsRequired(nat.Name) {
					if r.Headers.Validation == nil {
						r.Headers.Validation = &ValidationExpr{}
					}
					r.Headers.Validation.AddRequired(nat.Name)
				}
			}
		}
	default:
		if r.Headers.IsEmpty() && (r.Body == nil || r.Body.Type == Empty) {
			r.Headers.Type.(*Object).Set("loom-attribute", svcAtt)
		}
	}
}

// bodyAllowedForStatus reports whether a given response status code
// permits a body. See RFC 2616, section 4.4.
// See https://golang.org/src/net/http/transfer.go
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}

func initResponseCookies(cookies []*HTTPResponseCookieExpr, svcAtt *AttributeExpr) {
	svcObj := AsObject(svcAtt.Type)
	for _, cookie := range cookies {
		name := cookie.AttributeName()
		if name == "" {
			continue
		}
		var (
			patt     *AttributeExpr
			required bool
		)
		if svcObj != nil {
			patt = svcObj.Attribute(name)
			required = svcAtt.IsRequired(name)
		} else {
			patt = svcAtt
			required = true
		}
		initAttrFromDesign(cookie.Attribute(), patt)
		if required {
			if cookie.Validation == nil {
				cookie.Validation = &ValidationExpr{}
			}
			cookie.Validation.AddRequired(name)
		}
	}
}
