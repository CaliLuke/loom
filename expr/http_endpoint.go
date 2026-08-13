package expr

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/dimfeld/httppath"
)

type (
	// HTTPEndpointExpr describes a HTTP endpoint. It embeds a MethodExpr and
	// adds HTTP specific properties.
	//
	// It defines both an HTTP endpoint and the shape of HTTP requests and
	// responses made to that endpoint. The shape of requests is defined via
	// "parameters", there are path parameters (i.e. portions of the URL that
	// define parameter values), query string parameters and a payload parameter
	// (request body).
	HTTPEndpointExpr struct {
		eval.DSLFunc
		// MethodExpr is the underlying method expression.
		MethodExpr *MethodExpr
		// Service is the parent service.
		Service *HTTPServiceExpr
		// Endpoint routes
		Routes []*RouteExpr
		// MapQueryParams - when not nil - indicates that the HTTP
		// request query string parameters are used to build a map.
		//    - If the value is the empty string then the map is stored
		//      in the method payload (which must be of type Map)
		//    - If the value is a non-empty string then the map is
		//      stored in the payload attribute with the corresponding
		//      name (which must of be of type Map)
		MapQueryParams *string
		// Params defines the HTTP request path and query parameters.
		Params *MappedAttributeExpr
		// Headers defines the HTTP request headers.
		Headers *MappedAttributeExpr
		// Cookies defines the HTTP request cookies.
		Cookies *MappedAttributeExpr
		// Body describes the HTTP request body.
		Body *AttributeExpr
		// OpenAPIRequestBody describes a documentation-only request body.
		OpenAPIRequestBody *AttributeExpr
		// OpenAPIRequestBodyContentType is the documented request media type.
		OpenAPIRequestBodyContentType string
		// OpenAPIRequestBodyRequired records whether the documented body is required.
		OpenAPIRequestBodyRequired bool
		// StreamingBody describes the body transferred through the websocket
		// stream.
		StreamingBody *AttributeExpr
		// PayloadIDAttribute is the name of the JSON-RPC request ID
		// payload attribute.
		PayloadIDAttribute string
		// ResultIDAttribute is the name of the JSON-RPC result ID
		// result attribute.
		ResultIDAttribute string
		// SkipRequestBodyEncodeDecode indicates that the service method accepts
		// a reader and that the client provides a reader to stream the request
		// body.
		SkipRequestBodyEncodeDecode bool
		// SkipResponseBodyEncodeDecode indicates that the service method
		// returns a reader and that the client accepts a reader to stream the
		// response body.
		SkipResponseBodyEncodeDecode bool
		// FileResponse indicates that the endpoint result includes seekable
		// content served with net/http ServeContent semantics.
		FileResponse bool
		// Responses is the list of all the possible success HTTP
		// responses.
		Responses []*HTTPResponseExpr
		// HTTPErrors is the list of all the possible error HTTP
		// responses.
		HTTPErrors []*HTTPErrorExpr
		// Requirements contains the security requirements for the HTTP endpoint.
		Requirements []*SecurityExpr
		// MultipartRequest indicates that the request content type for
		// the endpoint is a multipart type.
		MultipartRequest bool
		// FormRequest indicates that the request content type for
		// the endpoint is application/x-www-form-urlencoded.
		FormRequest bool
		// OptionalRequestBody indicates that the endpoint accepts an empty
		// request body in addition to a typed JSON request body.
		OptionalRequestBody bool
		// Redirect defines a redirect for the endpoint.
		Redirect *HTTPRedirectExpr
		// SSE defines the Server-Sent Events configuration for this endpoint if it's
		// a streaming endpoint. If nil, the endpoint uses WebSockets by default or
		// inherits the service-level SSE configuration if defined.
		SSE *HTTPSSEExpr
		// Meta is a set of key/value pairs with semantic that is
		// specific to each generator, see dsl.Meta.
		Meta MetaExpr
		// prepared is true if Prepare has been run. This field is required to
		// avoid infinite recursions.
		prepared bool
	}

	// RouteExpr represents an endpoint route (HTTP endpoint).
	RouteExpr struct {
		// Method is the HTTP method, e.g. "GET", "POST", etc.
		Method string
		// Path is the URL path e.g. "/tasks/{id}"
		Path string
		// Endpoint is the endpoint this route applies to.
		Endpoint *HTTPEndpointExpr
		// Meta is an arbitrary set of key/value pairs, see
		// dsl.Meta
		Meta MetaExpr
	}
)

// Name of HTTP endpoint
func (e *HTTPEndpointExpr) Name() string {
	return e.MethodExpr.Name
}

// Description of HTTP endpoint
func (e *HTTPEndpointExpr) Description() string {
	return e.MethodExpr.Description
}

// EvalName returns the generic expression name used in error messages.
func (e *HTTPEndpointExpr) EvalName() string {
	var prefix, suffix string
	if e.Name() != "" {
		suffix = fmt.Sprintf("HTTP endpoint %#v", e.Name())
	} else {
		suffix = "unnamed HTTP endpoint"
	}
	if e.Service != nil {
		prefix = e.Service.EvalName() + " "
	}
	return prefix + suffix
}

// IsJSONRPC returns true if the endpoint is a JSON-RPC endpoint.
func (e *HTTPEndpointExpr) IsJSONRPC() bool {
	if _, ok := e.Meta["jsonrpc"]; ok {
		return true
	}
	_, ok := e.MethodExpr.Meta["jsonrpc"]
	return ok
}

// HasAbsoluteRoutes returns true if all the endpoint routes are absolute.
func (e *HTTPEndpointExpr) HasAbsoluteRoutes() bool {
	for _, r := range e.Routes {
		if !r.IsAbsolute() {
			return false
		}
	}
	return true
}

// Prepare computes the request path and query string parameters as well as the
// headers and body taking into account the inherited values from the service.
func (e *HTTPEndpointExpr) Prepare() {
	// Avoid infinite recursions when traversing parents.
	if e.prepared {
		return
	}
	e.prepared = true
	e.normalizeRouteMethods()
	e.initTransportAttributes()
	e.inheritTransportAttributes()
	e.ensureRouteParams()
	e.ensureDefaultResponse()
	e.inheritSSE()
	e.inheritHTTPErrors()
	e.forceWebSocketRouteMethod()
	e.prepareResponses()
}

func (e *HTTPEndpointExpr) normalizeRouteMethods() {
	for _, route := range e.Routes {
		route.Method = strings.ToUpper(route.Method)
	}
}

// EvalName returns the generic definition name used in error messages.
func (r *RouteExpr) EvalName() string {
	return fmt.Sprintf(`route %s %q of %s`, r.Method, r.Path, r.Endpoint.EvalName())
}

// Validate validates a route expression by ensuring that the route parameters
// can be inferred from the method payload and there is no duplicate parameters
// in an absolute route.
func (r *RouteExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	r.validateMethod(verr)

	// Make sure route params are defined in the method payload
	if rparams := r.Params(); len(rparams) > 0 {
		if r.Endpoint.MethodExpr.Payload == nil {
			verr.Add(r, "Route parameters are defined, but method payload is not defined.")
		} else {
			switch r.Endpoint.MethodExpr.Payload.Type.(type) {
			case *Map:
				verr.Add(r, "Route parameters are defined, but method payload is a map. Method payload must be a primitive or an object.")
			case *Object, UserType:
				for _, p := range rparams {
					if r.Endpoint.MethodExpr.Payload.Find(p) == nil {
						verr.Add(r, "Route param %q not found in method payload", p)
					}
				}
			}
			if len(rparams) > 1 && IsPrimitive(r.Endpoint.MethodExpr.Payload.Type) {
				verr.Add(r, "Multiple route parameters are defined, but method payload is a primitive. Only one router parameter can be defined if payload is primitive.")
			}
		}
	}

	// Make sure there's no duplicate params in absolute route
	paths := r.FullPaths()
	for _, path := range paths {
		matches := HTTPWildcardRegex.FindAllStringSubmatch(path, -1)
		wcs := make(map[string]struct{}, len(matches))
		for _, match := range matches {
			if _, ok := wcs[match[1]]; ok {
				verr.Add(r, "Wildcard %q appears multiple times in full path %q", match[1], path)
			}
			wcs[match[1]] = struct{}{}
		}
	}

	// For WebSocket streaming endpoints, only GET is supported
	// SSE endpoints can use both GET and POST (JSON-RPC SSE uses POST)
	if r.Endpoint.MethodExpr.IsStreaming() && len(r.Endpoint.Responses) > 0 && r.Endpoint.SSE == nil {
		if r.Method != "GET" {
			verr.Add(r, "WebSocket endpoint supports only \"GET\" method. Got %q.", r.Method)
		}
	}

	// HEAD method must not return a response body as per RFC 2616 section 9.4
	if r.Method == "HEAD" && !r.Endpoint.FileResponse {
		disallowBody := func(resp *HTTPResponseExpr) {
			if httpResponseBody(r.Endpoint, resp).Type != Empty {
				verr.Add(r, "HTTP status %d: Response body defined for HEAD method which does not allow response body.", resp.StatusCode)
			}
		}
		for _, resp := range r.Endpoint.Responses {
			disallowBody(resp)
		}
		for _, e := range r.Endpoint.HTTPErrors {
			disallowBody(e.Response)
		}
	}
	return verr
}

func (r *RouteExpr) validateMethod(verr *eval.ValidationErrors) {
	if !validHTTPMethodToken(r.Method) {
		verr.Add(r, "HTTP method %q is invalid: must be a non-empty RFC 9110 token", r.Method)
	}
}

func validHTTPMethodToken(method string) bool {
	if method == "" {
		return false
	}
	for i := 0; i < len(method); i++ {
		char := method[i]
		if ('a' <= char && char <= 'z') || ('A' <= char && char <= 'Z') || ('0' <= char && char <= '9') {
			continue
		}
		switch char {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return false
		}
	}
	return true
}

// Params returns all the route parameters across all the base paths. For
// example for the route "GET /foo/{fooID:foo_id}" Params returns
// []string{"fooID:foo_id"}.
func (r *RouteExpr) Params() []string {
	paths := r.FullPaths()
	var res []string
	for _, p := range paths {
		ws := ExtractHTTPWildcards(p)
		for _, w := range ws {
			found := slices.Contains(res, w)
			if !found {
				res = append(res, w)
			}
		}
	}
	return res
}

// FullPaths returns the endpoint full paths computed by concatenating the
// service base paths with the route specific path.
func (r *RouteExpr) FullPaths() []string {
	if r.IsAbsolute() {
		return []string{httppath.Clean(r.Path[1:])}
	}
	bases := r.Endpoint.Service.FullPaths()
	res := make([]string, len(bases))
	for i, b := range bases {
		res[i] = httppath.Clean(path.Join(b, r.Path))
		if res[i] == "/" {
			continue
		}
		// path has trailing slash
		if r.Path == "/" && strings.HasSuffix(b, "/") {
			res[i] += "/"
		} else if r.Path != "/" && strings.HasSuffix(r.Path, "/") {
			res[i] += "/"
		}
	}
	return res
}

// IsAbsolute returns true if the endpoint path should not be concatenated to
// the service and API base paths.
func (r *RouteExpr) IsAbsolute() bool {
	return strings.HasPrefix(r.Path, "//")
}

// initAttr initializes the given mapped attribute with the given service
// attribute.
func initAttr(ma *MappedAttributeExpr, svcAtt *AttributeExpr) {
	svcObj := AsObject(svcAtt.Type)
	for _, nat := range *AsObject(ma.Type) {
		var patt *AttributeExpr
		var required bool
		if svcObj != nil {
			patt = svcObj.Attribute(nat.Name)
			required = svcAtt.IsRequired(nat.Name)
		} else {
			patt = svcAtt
			required = true
		}
		initAttrFromDesign(nat.Attribute, patt)
		if required {
			if ma.Validation == nil {
				ma.Validation = &ValidationExpr{}
			}
			ma.Validation.AddRequired(nat.Name)
		}
	}
}

// initAttrFromDesign overrides the type of att with the one of patt and
// initializes other non-initialized fields of att with the one of patt except
// Meta.
func initAttrFromDesign(att, patt *AttributeExpr) {
	if patt == nil || patt.Type == Empty {
		return
	}
	att.Type = patt.Type
	if att.Description == "" {
		att.Description = patt.Description
	}
	if att.Docs == nil {
		att.Docs = patt.Docs
	}
	if att.Validation == nil {
		att.Validation = patt.Validation
	}
	if att.DefaultValue == nil {
		att.DefaultValue = patt.DefaultValue
	}
	if att.UserExamples == nil {
		att.UserExamples = patt.UserExamples
	}
	if att.Meta == nil {
		att.Meta = patt.Meta
	}
}

// isEmpty returns true if an attribute is Empty type and it has no bases and
// references, or if an attribute is an empty object.
func isEmpty(a *AttributeExpr) bool {
	if !IsObject(a.Type) {
		return false
	}
	if obj := AsObject(a.Type); obj != nil && len(*obj) != 0 {
		if a.Type == Empty {
			panic("Empty should have no attribute") // bug
		}
		return false
	}
	if len(a.Bases) != 0 || len(a.References) != 0 {
		return false
	}
	if ut, ok := a.Type.(UserType); ok {
		uatt := ut.Attribute()
		if len(uatt.Bases) != 0 || len(uatt.References) != 0 {
			return false
		}
	}
	return true
}

// hasJSONRPCIDField returns true if an attribute or any of its nested attributes
// has the "jsonrpc:id" meta tag, indicating it's designated as the JSON-RPC ID field.
func hasJSONRPCIDField(attr *AttributeExpr) bool {
	return hasJSONRPCIDFieldRec(attr, make(map[*AttributeExpr]struct{}), make(map[string]struct{}))
}

// hasJSONRPCIDFieldRec walks the attribute graph looking for the jsonrpc:id meta
// while guarding against cycles that may occur with recursive user types.
func hasJSONRPCIDFieldRec(attr *AttributeExpr, seen map[*AttributeExpr]struct{}, seenUT map[string]struct{}) bool {
	if attr == nil || attr.Type == Empty {
		return false
	}
	if _, ok := seen[attr]; ok {
		return false
	}
	seen[attr] = struct{}{}

	// Check if this attribute itself has the jsonrpc:id meta tag
	if attr.Meta != nil {
		if _, hasID := attr.Meta["jsonrpc:id"]; hasID {
			return true
		}
	}

	// For object types, check all nested attributes
	if obj := AsObject(attr.Type); obj != nil {
		for _, nat := range *obj {
			if hasJSONRPCIDFieldRec(nat.Attribute, seen, seenUT) {
				return true
			}
		}
	}

	// For user types, check the underlying attribute (guarding for recursion)
	if ut, ok := attr.Type.(UserType); ok {
		if ut != nil {
			if _, ok := seenUT[ut.ID()]; ok {
				return false
			}
			seenUT[ut.ID()] = struct{}{}
			return hasJSONRPCIDFieldRec(ut.Attribute(), seen, seenUT)
		}
	}
	return false
}
