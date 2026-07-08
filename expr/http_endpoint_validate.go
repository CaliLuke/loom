package expr

import (
	"errors"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

// Validate validates the endpoint expression.
func (e *HTTPEndpointExpr) Validate() error {
	verr := new(eval.ValidationErrors)
	if e.Name() == "" {
		verr.Add(e, "Endpoint name cannot be empty")
	}
	e.validateSkipBodyEncoding(verr)
	e.validateStreamingSSE(verr)
	e.validateJSONRPCTransport(verr)
	e.validateRedirect(verr)
	e.validateRoutes(verr)
	e.validateResponses(verr)
	e.validateBodyAndPayload(verr)
	for _, er := range e.HTTPErrors {
		verr.Merge(er.Validate())
	}

	body := httpRequestBody(e)
	if e.MethodExpr.IsStreaming() && body.Type != Empty {
		_, isJSONRPC := e.MethodExpr.Meta["jsonrpc"]
		if e.SSE == nil && !isJSONRPC {
			verr.Add(e, "HTTP endpoint request body must be empty when the endpoint uses streaming. Payload attributes must be mapped to headers and/or params.")
		}
	}

	return verr
}

func (e *HTTPEndpointExpr) validateResponses(verr *eval.ValidationErrors) {
	hasTags := false
	allTagged := true
	successResp := false
	statusCounts := make(map[int]int, len(e.Responses))
	for _, r := range e.Responses {
		statusCounts[r.StatusCode]++
	}
	for _, r := range e.Responses {
		verr.Merge(r.Validate(e))
		if count := statusCounts[r.StatusCode]; count > 1 {
			for i := 0; i < count-1; i++ {
				verr.Add(r, "Multiple response definitions with status code %d", r.StatusCode)
			}
		}

		if r.Tag[0] == "" {
			allTagged = false
		} else {
			hasTags = true
		}
		if r.StatusCode < 400 {
			if successResp && e.MethodExpr.Stream == ServerStreamKind {
				verr.Add(r, "At most one success response can be defined for a streaming endpoint.")
				if r.Body != nil && r.Body.Type == Empty {
					verr.Add(r, "Response body empty but endpoint defines streaming WebSocket response.")
				}
			} else if successResp && e.SkipResponseBodyEncodeDecode {
				verr.Add(r, "At most one success response can be defined for a endpoint using SkipResponseBodyEncodeDecode.")
			}
			successResp = true
		}
	}
	if hasTags && allTagged {
		verr.Add(e, "All responses define a Tag, at least one response must define no Tag.")
	}
	if hasTags && !IsObject(e.MethodExpr.Result.Type) {
		verr.Add(e, "Some responses define a Tag but the method Result type is not an object.")
	}
}

func (e *HTTPEndpointExpr) validateBodyAndPayload(verr *eval.ValidationErrors) {
	verr.Merge(e.validateParams())
	verr.Merge(e.validateHeadersAndCookies())

	if e.Body != nil {
		verr.Merge(e.Body.Validate("HTTP body", e))
		if e.SkipRequestBodyEncodeDecode {
			verr.Add(e, "Cannot define a request body when using SkipRequestBodyEncodeDecode.")
		}
		e.validateBodyRequiredPayloadAttributes(verr)
	}

	var (
		hasParams  = !e.Params.IsEmpty()
		hasHeaders = !e.Headers.IsEmpty()
		hasCookies = !e.Cookies.IsEmpty()
	)
	if e.validateMissingPayload(verr, hasParams, hasHeaders) {
		return
	}
	e.validateRequestBodyOptions(verr)

	if IsArray(e.MethodExpr.Payload.Type) {
		e.validateArrayPayloadTransport(verr, hasParams, hasHeaders, hasCookies)
	}
	if pMap := AsMap(e.MethodExpr.Payload.Type); pMap != nil {
		e.validateMapPayloadTransport(verr, pMap, hasParams)
	}
	if IsObject(e.MethodExpr.Payload.Type) {
		e.validateObjectPayloadTransport(verr)
	}
	if e.SkipRequestBodyEncodeDecode && httpRequestBody(e).Type != Empty {
		verr.Add(e, "HTTP endpoint request body must be empty when using SkipRequestBodyEncodeDecode but not all method payload attributes are mapped to headers and params. Make sure to define Headers and Params as needed.")
	}
}

func (e *HTTPEndpointExpr) validateBodyRequiredPayloadAttributes(verr *eval.ValidationErrors) {
	if e.Body == nil || e.Body.Validation == nil {
		return
	}
	var preqs, missing []string
	if e.MethodExpr.Payload != nil && e.MethodExpr.Payload.Validation != nil {
		preqs = e.MethodExpr.Payload.Validation.Required
	}
	for _, req := range e.Body.Validation.Required {
		if containsString(preqs, req) {
			continue
		}
		missing = append(missing, req)
	}
	if len(missing) == 0 {
		return
	}
	is := "is"
	s := ""
	if len(missing) > 1 {
		is = "are"
		s = "s"
	}
	verr.Add(e, "The following HTTP request body attribute%s %s required but the corresponding method payload attribute%s %s not: %s. Use 'Required' to make the attribute%s required in the method payload as well.",
		s, is, s, is, strings.Join(missing, ", "), s)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (e *HTTPEndpointExpr) validateMissingPayload(verr *eval.ValidationErrors, hasParams, hasHeaders bool) bool {
	if !isEmpty(e.MethodExpr.Payload) {
		return false
	}
	if e.MapQueryParams != nil {
		verr.Add(e, "MapParams is set but Payload is not defined")
	}
	if e.MultipartRequest {
		verr.Add(e, "MultipartRequest is set but Payload is not defined")
	}
	if e.FormRequest {
		verr.Add(e, "FormRequest is set but Payload is not defined")
	}
	if e.OptionalRequestBody {
		verr.Add(e, "OptionalRequestBody is set but Payload is not defined")
	}
	if hasParams {
		verr.Add(e, "Params are set but Payload is not defined.")
	}
	if hasHeaders {
		verr.Add(e, "Headers are set but Payload is not defined.")
	}
	return true
}

func (e *HTTPEndpointExpr) validateArrayPayloadTransport(verr *eval.ValidationErrors, hasParams, hasHeaders, hasCookies bool) {
	if e.MapQueryParams != nil {
		verr.Add(e, "MapParams is set but Payload type is array. Payload type must be map or an object with a map attribute")
	}
	if hasParams && (e.MultipartRequest || e.FormRequest) {
		if e.MultipartRequest {
			verr.Add(e, "Payload type is array but HTTP endpoint defines MultipartRequest and route/query string parameters. At most one of these must be defined.")
		} else {
			verr.Add(e, "Payload type is array but HTTP endpoint defines FormRequest and route/query string parameters. At most one of these must be defined.")
		}
	}
	if hasHeaders {
		if hasCookies || e.MultipartRequest || e.FormRequest {
			switch {
			case e.MultipartRequest:
				verr.Add(e, "Payload type is array but HTTP endpoint defines headers and MultipartRequest or cookies. At most one of these must be defined.")
			case e.FormRequest:
				verr.Add(e, "Payload type is array but HTTP endpoint defines headers and FormRequest or cookies. At most one of these must be defined.")
			default:
				verr.Add(e, "Payload type is array but HTTP endpoint defines headers and cookies. At most one of these must be defined.")
			}
		}
		if hasParams {
			verr.Add(e, "Payload type is array but HTTP endpoint defines both route or query string parameters and headers. At most one parameter or header must be defined and it must be of type array.")
		}
		if !IsPrimitive(AsArray(e.MethodExpr.Payload.Type).ElemType.Type) {
			verr.Add(e, "Array payloads used in HTTP headers must be of arrays of primitive types.")
		}
	}
	if e.Body != nil && e.Body.Type != Empty {
		if e.MultipartRequest {
			verr.Add(e, "Payload type is array but HTTP endpoint defines MultipartRequest and body. At most one of these must be defined.")
		}
		if e.FormRequest {
			verr.Add(e, "Payload type is array but HTTP endpoint defines FormRequest and body. At most one of these must be defined.")
		}
		if !IsArray(e.Body.Type) {
			verr.Add(e, "Payload type is array but HTTP endpoint body is not.")
		}
		if hasParams {
			verr.Add(e, "Payload type is array but HTTP endpoint defines both a body and route or query string parameters. At most one of these must be defined and it must be an array.")
		}
		if hasHeaders {
			verr.Add(e, "Payload type is array but HTTP endpoint defines both a body and headers. At most one of these must be defined and it must be an array.")
		}
	}
	if !hasParams && !hasHeaders && e.SkipRequestBodyEncodeDecode {
		verr.Add(e, "Payload type is array but HTTP endpoint uses SkipRequestBodyEncodeDecode and does not define headers or params.")
	}
}

func (e *HTTPEndpointExpr) validateMapPayloadTransport(verr *eval.ValidationErrors, pMap *Map, hasParams bool) {
	if e.MapQueryParams != nil {
		if e.MultipartRequest {
			verr.Add(e, "Payload type is map but HTTP endpoint defines MultipartRequest and MapParams. At most one of these must be defined.")
		}
		if e.FormRequest {
			verr.Add(e, "Payload type is map but HTTP endpoint defines FormRequest and MapParams. At most one of these must be defined.")
		}
		if *e.MapQueryParams != "" {
			verr.Add(e, "MapParams is set to an attribute in the Payload but Payload is a map. Payload must be an object with an attribute of map type")
		}
		if !IsPrimitive(pMap.KeyType.Type) {
			verr.Add(e, "MapParams is set and Payload type is map. But payload key type must be a primitive")
		}
		if !IsPrimitive(pMap.ElemType.Type) && !IsArray(pMap.ElemType.Type) {
			verr.Add(e, "MapParams is set and Payload type is map. But payload element type must be a primitive or array")
		}
		if IsArray(pMap.ElemType.Type) && !IsPrimitive(AsArray(pMap.ElemType.Type).ElemType.Type) {
			verr.Add(e, "MapParams is set and Payload type is map. But array elements in payload element type must be primitive")
		}
	}
	if hasParams && (e.MultipartRequest || e.FormRequest) {
		if e.MultipartRequest {
			verr.Add(e, "Payload type is map but HTTP endpoint defines MultipartRequest and route/query string parameters. At most one of these must be defined.")
		} else {
			verr.Add(e, "Payload type is map but HTTP endpoint defines FormRequest and route/query string parameters. At most one of these must be defined.")
		}
	}
	if e.Body != nil && e.Body.Type != Empty {
		if e.MultipartRequest {
			verr.Add(e, "Payload type is map but HTTP endpoint defines MultipartRequest and body. At most one of these must be defined.")
		}
		if e.FormRequest {
			verr.Add(e, "Payload type is map but HTTP endpoint defines FormRequest and body. At most one of these must be defined.")
		}
		if !IsMap(e.Body.Type) {
			verr.Add(e, "Payload type is map but HTTP endpoint body is not.")
		}
		if hasParams {
			verr.Add(e, "Payload type is map but HTTP endpoint defines both a body and route or query string parameters. At most one of these must be defined and it must be a map.")
		}
	}
	if !hasParams && e.SkipRequestBodyEncodeDecode {
		verr.Add(e, "Payload type is map but HTTP endpoint uses SkipRequestBodyEncodeDecode and does not define headers.")
	}
}

func (e *HTTPEndpointExpr) validateObjectPayloadTransport(verr *eval.ValidationErrors) {
	if e.MapQueryParams != nil {
		if pAttr := *e.MapQueryParams; pAttr == "" {
			verr.Add(e, "MapParams is set to map entire payload but payload is an object. Payload must be a map.")
		} else if e.MethodExpr.Payload.Find(pAttr) == nil {
			verr.Add(e, "MapParams is set to an attribute in Payload. But payload has no attribute with type map and name %s", pAttr)
		}
	}
	if e.Body == nil {
		return
	}
	if e.MultipartRequest {
		verr.Add(e, "HTTP endpoint defines MultipartRequest and body. At most one of these must be defined.")
	}
	if e.FormRequest {
		verr.Add(e, "HTTP endpoint defines FormRequest and body. At most one of these must be defined.")
	}
	bObj := AsObject(e.Body.Type)
	if bObj == nil {
		return
	}
	var props []string
	props, ok := e.Body.Meta["origin:attribute"]
	if !ok {
		for _, nat := range *bObj {
			props = append(props, splitMappedAttributeName(nat.Name))
		}
	}
	for _, prop := range props {
		if e.MethodExpr.Payload.Find(prop) == nil {
			verr.Add(e, "Body %q is not found in Payload.", prop)
		}
	}
	if !e.OptionalRequestBody {
		return
	}
	if ok && len(props) == 1 && e.MethodExpr.Payload.IsRequired(props[0]) {
		verr.Add(e, "OptionalRequestBody requires the payload attribute mapped to the request body to be optional.")
	}
	if !ok && hasRequiredBodyAttributes(e.Body) {
		verr.Add(e, "OptionalRequestBody requires the request body to have no required attributes.")
	}
}

func (e *HTTPEndpointExpr) validateRequestBodyOptions(verr *eval.ValidationErrors) {
	if e.FormRequest && e.MultipartRequest {
		verr.Add(e, "HTTP endpoint cannot define both FormRequest and MultipartRequest.")
	}
	if e.FormRequest && e.SkipRequestBodyEncodeDecode {
		verr.Add(e, "HTTP endpoint cannot use FormRequest with SkipRequestBodyEncodeDecode.")
	}
	if e.OptionalRequestBody && e.SkipRequestBodyEncodeDecode {
		verr.Add(e, "HTTP endpoint cannot use OptionalRequestBody with SkipRequestBodyEncodeDecode.")
	}
	if e.OptionalRequestBody && e.MultipartRequest {
		verr.Add(e, "HTTP endpoint cannot use OptionalRequestBody with MultipartRequest.")
	}
	if e.OptionalRequestBody && e.FormRequest {
		verr.Add(e, "HTTP endpoint cannot use OptionalRequestBody with FormRequest.")
	}
	if e.OptionalRequestBody && e.Body != nil && !IsObject(e.Body.Type) {
		verr.Add(e, "OptionalRequestBody requires an object request body.")
	}
	if e.OptionalRequestBody && (e.Body == nil || e.Body.Type == Empty) {
		verr.Add(e, "HTTP endpoint uses OptionalRequestBody but does not define a request body.")
	}
	if e.MultipartRequest && IsUnion(e.MethodExpr.Payload.Type) {
		verr.Add(e, "MultipartRequest requires an object payload, constructor unions are not supported")
	}
	if e.FormRequest && !(IsUnion(e.MethodExpr.Payload.Type) || IsObject(e.MethodExpr.Payload.Type)) {
		verr.Add(e, "FormRequest requires an object or constructor union payload")
	}
}

func (e *HTTPEndpointExpr) validateSkipBodyEncoding(verr *eval.ValidationErrors) {
	if e.SkipRequestBodyEncodeDecode {
		if s := Root.API.GRPC.Service(e.Service.Name()); s != nil && s.Endpoint(e.Name()) != nil {
			verr.Add(e, "Endpoint cannot use SkipRequestBodyEncodeDecode and define a gRPC transport.")
		}
		if e.MethodExpr.IsPayloadStreaming() {
			verr.Add(e, "Endpoint cannot use SkipRequestBodyEncodeDecode when method defines a StreamingPayload.")
		}
		if e.MethodExpr.IsResultStreaming() {
			verr.Add(e, "Endpoint cannot use SkipRequestBodyEncodeDecode when method defines a StreamingResult. Use SkipResponseBodyEncodeDecode instead.")
		}
	}
	if e.SkipResponseBodyEncodeDecode {
		if s := Root.API.GRPC.Service(e.Service.Name()); s != nil && s.Endpoint(e.Name()) != nil {
			verr.Add(e, "Endpoint response cannot use SkipResponseBodyEncodeDecode and define a gRPC transport.")
		}
		if e.MethodExpr.IsPayloadStreaming() {
			verr.Add(e, "Endpoint cannot use SkipResponseBodyEncodeDecode when method defines a StreamingPayload. Use SkipRequestBodyEncodeDecode instead.")
		}
		if e.MethodExpr.IsResultStreaming() {
			verr.Add(e, "Endpoint cannot use SkipResponseBodyEncodeDecode when method defines a StreamingResult.")
		}
		if rt, ok := e.MethodExpr.Result.Type.(*ResultTypeExpr); ok && len(rt.Views) > 1 {
			verr.Add(e, "Endpoint cannot use SkipResponseBodyEncodeDecode when method result type defines multiple views.")
		}
	}
}

func (e *HTTPEndpointExpr) validateStreamingSSE(verr *eval.ValidationErrors) {
	if e.MethodExpr.Stream == ServerStreamKind && e.SSE != nil {
		if err := e.SSE.Validate(e.MethodExpr); err != nil {
			var valErr *eval.ValidationErrors
			if errors.As(err, &valErr) {
				verr.Merge(valErr)
			}
		}
	}
	if e.MethodExpr.HasMixedResults() {
		if e.SSE == nil {
			verr.Add(e, "Methods with both Result and StreamingResult defined with different types must use ServerSentEvents()")
		}
		if e.MethodExpr.IsPayloadStreaming() {
			verr.Add(e, "Methods with both Result and StreamingResult cannot have StreamingPayload")
		}
		return
	}
	if e.SSE == nil {
		return
	}
	switch e.MethodExpr.Stream {
	case BidirectionalStreamKind:
		verr.Add(e, "Server-Sent Events cannot be used with bidirectional streaming endpoints")
	case ClientStreamKind:
		verr.Add(e, "Server-Sent Events cannot be used with client-to-server streaming endpoints")
	case NoStreamKind:
		verr.Add(e, "Server-Sent Events can only be used with endpoints that have a streaming result or mixed results")
	}
}

func (e *HTTPEndpointExpr) validateJSONRPCTransport(verr *eval.ValidationErrors) {
	if !e.IsJSONRPC() {
		return
	}
	if e.MethodExpr.Stream == ServerStreamKind && e.SSE == nil &&
		e.MethodExpr.Payload.Type != Empty && e.MethodExpr.StreamingPayload.Type != Empty {
		verr.Add(e, "JSON-RPC WebSocket server streaming method %q cannot define both Payload and StreamingPayload. Use Payload for the request data", e.MethodExpr.Name)
	}
	if e.MethodExpr.Result == nil || e.MethodExpr.Result.Type == Empty || !hasJSONRPCIDField(e.MethodExpr.Result) {
		return
	}
	requestHasID := false
	if e.MethodExpr.IsPayloadStreaming() {
		requestHasID = hasJSONRPCIDField(e.MethodExpr.StreamingPayload)
	} else {
		requestHasID = hasJSONRPCIDField(e.MethodExpr.Payload)
	}
	if !requestHasID {
		verr.Add(e, "JSON-RPC method %q result defines an ID field but the request (payload) does not. Result may only have ID field if request does", e.MethodExpr.Name)
	}
}

func (e *HTTPEndpointExpr) validateRedirect(verr *eval.ValidationErrors) {
	if e.Redirect == nil {
		return
	}
	for _, response := range e.Responses {
		if response.StatusCode != e.Redirect.StatusCode {
			verr.Add(e, "Endpoint cannot use Response when using Redirect.")
			return
		}
	}
}

func (e *HTTPEndpointExpr) validateRoutes(verr *eval.ValidationErrors) {
	if len(e.Routes) == 0 {
		verr.Add(e, "No route defined for HTTP endpoint")
		return
	}
	for _, route := range e.Routes {
		verr.Merge(route.Validate())
	}
	params := e.Routes[0].Params()
	for _, route := range e.Routes[1:] {
		for _, param := range params {
			if !containsString(route.Params(), param) {
				verr.Add(e, "Param %q does not appear in all routes", param)
			}
		}
		for _, param := range route.Params() {
			if !containsString(params, param) {
				verr.Add(e, "Param %q does not appear in all routes", param)
			}
		}
	}
}

func hasRequiredBodyAttributes(att *AttributeExpr) bool {
	if att == nil {
		return false
	}
	if att.Validation != nil && len(att.Validation.Required) > 0 {
		return true
	}
	if ut, ok := att.Type.(UserType); ok {
		if v := ut.Attribute().Validation; v != nil && len(v.Required) > 0 {
			return true
		}
	}
	return false
}
