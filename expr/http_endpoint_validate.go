package expr

import (
	"errors"
	"mime"
	"strings"

	"github.com/CaliLuke/loom/eval"
)

// Validate validates the endpoint expression.
func (e *HTTPEndpointExpr) Validate() error {
	verr := new(eval.ValidationErrors)
	e.captureResponseBindings()
	e.ensureWireBindings().validate(e, verr)
	if e.Name() == "" {
		verr.Add(e, "Endpoint name cannot be empty")
	}
	e.validateSkipBodyEncoding(verr)
	e.validateFileResponse(verr)
	e.validateStreamingSSE(verr)
	e.validateJSONRPCTransport(verr)
	e.validateRedirect(verr)
	e.validateRoutes(verr)
	e.validateResponses(verr)
	if !e.validateBodyAndPayload(verr) {
		return verr
	}
	e.validateParameterSerialization(verr)
	e.validateRequestContentType(verr)
	e.validateMediaTypes(verr)
	e.validateRequestReconstruction(verr)
	e.validateNullableTransportLocations(verr)
	e.validateBodyElementNames(verr)
	e.validateAPIKeyLocations(verr)
	for _, er := range e.HTTPErrors {
		verr.Merge(er.Validate())
		if definition := e.MethodExpr.Error(er.Name); definition != nil {
			er.Response.validateMediaTypes(e, definition.AttributeExpr, verr)
		}
	}
	e.validateStatusErrorNames(verr)

	body := httpRequestBody(e)
	if e.MethodExpr.IsStreaming() && body.Type != Empty {
		isJSONRPC := e.IsJSONRPC()
		if e.SSE == nil && !isJSONRPC {
			verr.Add(e, "HTTP endpoint request body must be empty when the endpoint uses streaming. Payload attributes must be mapped to headers and/or params.")
		}
	}

	return verr
}
func (e *HTTPEndpointExpr) validateNullableTransportLocations(verr *eval.ValidationErrors) {
	if e == nil {
		return
	}
	if e.MethodExpr != nil && e.SSE == nil {
		jsonrpc := e.IsJSONRPC()
		if !jsonrpc && IsNullable(e.MethodExpr.StreamingPayload) {
			verr.Add(e, "HTTP WebSocket streaming payload root cannot be nullable: JSON null marks end of input; use nullable fields inside a non-null payload")
		}
	}
	if e.Params != nil && containsNullable(e.Params.Attribute()) {
		verr.Add(e, "HTTP query and path parameters do not support nullable attributes")
	}
	if e.Headers != nil && containsNullable(e.Headers.Attribute()) {
		verr.Add(e, "HTTP request headers do not support nullable attributes")
	}
	if e.Cookies != nil && containsNullable(e.Cookies.Attribute()) {
		verr.Add(e, "HTTP request cookies do not support nullable attributes")
	}
	if e.MapQueryParams != nil && e.MethodExpr != nil {
		mapped := e.MethodExpr.Payload
		if name := *e.MapQueryParams; name != "" && mapped != nil {
			_, mapped = mapped.FindAttribute(name)
		}
		if containsNullable(mapped) {
			verr.Add(e, "HTTP map query parameters do not support nullable attributes")
		}
	}
	for _, response := range e.Responses {
		validateNullableHTTPResponse(response, verr)
	}
	for _, httpError := range e.HTTPErrors {
		if httpError != nil {
			validateNullableHTTPResponse(httpError.Response, verr)
		}
	}
}

func validateNullableHTTPResponse(response *HTTPResponseExpr, verr *eval.ValidationErrors) {
	if response == nil {
		return
	}
	if response.Headers != nil && containsNullable(response.Headers.Attribute()) {
		verr.Add(response, "HTTP response headers do not support nullable attributes")
	}
	for _, cookie := range response.Cookies {
		if cookie != nil && containsNullable(cookie.Attribute()) {
			verr.Add(response, "HTTP response cookies do not support nullable attributes")
			break
		}
	}
}

func (e *HTTPEndpointExpr) validateResponses(verr *eval.ValidationErrors) {
	e.validateResponseSelection(verr)
	hasTags := false
	untagged := 0
	for _, r := range e.Responses {
		verr.Merge(r.Validate(e))
		if r.Tag[0] == "" {
			untagged++
		} else {
			hasTags = true
		}
	}
	if hasTags && untagged == 0 {
		verr.Add(e, "All responses define a Tag, at least one response must define no Tag.")
	}
	if hasTags && untagged > 1 {
		verr.Add(e, "exactly one response must define no Tag; found %d", untagged)
	}
	if hasTags && !IsObject(e.MethodExpr.Result.Type) {
		verr.Add(e, "Some responses define a Tag but the method Result type is not an object.")
	}
	if len(e.Responses) > 1 && e.MethodExpr.IsStreaming() && (e.SSE == nil || !e.StatusSelected()) {
		verr.Add(e, "At most one success response can be defined for a streaming endpoint except HTTP SSE status mode.")
	}
}

// validateBodyAndPayload reports whether request-body derivation may proceed.
// An invalid authored body must retain its validation errors instead of reaching
// analyses that require a valid attribute graph.
func (e *HTTPEndpointExpr) validateBodyAndPayload(verr *eval.ValidationErrors) bool {
	verr.Merge(e.validateParams())
	verr.Merge(e.validateHeadersAndCookies())

	if e.Body != nil {
		bodyErrors := e.Body.Validate("HTTP body", e)
		verr.Merge(bodyErrors)
		if bodyErrors != nil && len(bodyErrors.Errors) > 0 {
			return false
		}
		validateBodyRequiredKeys(e.Body, "HTTP request body", e, verr)
		if e.SkipRequestBodyEncodeDecode {
			verr.Add(e, "Cannot define a request body when using SkipRequestBodyEncodeDecode.")
		}
		e.validateBodyRequiredPayloadAttributes(verr)
	}

	var (
		hasParams  = !e.Params.IsEmpty()
		hasHeaders = !e.Headers.IsEmpty()
	)
	if e.validateMissingPayload(verr, hasParams, hasHeaders) {
		return true
	}
	e.validateRequestBodyOptions(verr)

	if IsArray(e.MethodExpr.Payload.Type) || IsMap(e.MethodExpr.Payload.Type) {
		e.validateCollectionPayloadTransport(verr)
	}
	if IsObject(e.MethodExpr.Payload.Type) {
		e.validateObjectPayloadTransport(verr)
	}
	if e.SkipRequestBodyEncodeDecode && httpRequestBody(e).Type != Empty {
		verr.Add(e, "HTTP endpoint request body must be empty when using SkipRequestBodyEncodeDecode but not all method payload attributes are mapped to headers and params. Make sure to define Headers and Params as needed.")
	}
	return true
}

// validateBodyRequiredPayloadAttributes checks that the payload requires each
// attribute that the explicit request body requires. It compares attribute
// names, so a body that requires "name:n" matches a payload that requires
// "name".
func (e *HTTPEndpointExpr) validateBodyRequiredPayloadAttributes(verr *eval.ValidationErrors) {
	if e.Body == nil {
		return
	}
	if _, selected := e.Body.Meta["origin:attribute"]; selected {
		return
	}
	var preqs, missing []string
	if e.MethodExpr.Payload != nil {
		for _, req := range e.MethodExpr.Payload.AllRequired() {
			preqs = append(preqs, AttributeName(req))
		}
	}
	for _, req := range e.Body.AllRequired() {
		if containsString(preqs, AttributeName(req)) {
			continue
		}
		authored, _ := requiredDiagnosticName(e.Body, req, make(map[*AttributeExpr]bool))
		missing = append(missing, authored)
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

// payloadKey returns the object key of the attribute of payload named name,
// such as "n:m" for "n", or name when payload has no such attribute.
func payloadKey(payload *AttributeExpr, name string) string {
	if key, att := payload.FindAttribute(name); att != nil {
		return key
	}
	return name
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
	payload := e.MethodExpr.Payload
	encodedObject := payload.Type != Empty && (e.FormRequest || e.MultipartRequest || len(e.MediaTypes) > 0)
	if !isEmpty(payload) || encodedObject {
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
	if e.OpenAPIRequestBody != nil {
		e.validateOpenAPIRequestBody(verr)
	}
	if hasParams {
		verr.Add(e, "Params are set but Payload is not defined.")
	}
	if hasHeaders {
		verr.Add(e, "Headers are set but Payload is not defined.")
	}
	return true
}

// validateCollectionPayloadTransport gives a whole collection one wire owner.
// Shape and style admission belong to validateMappedParameterCodecs.
func (e *HTTPEndpointExpr) validateCollectionPayloadTransport(verr *eval.ValidationErrors) {
	kind := "array"
	pMap := AsMap(e.MethodExpr.Payload.Type)
	if pMap != nil {
		kind = "map"
	}
	bindings := len(*AsObject(e.Params.Type)) + len(*AsObject(e.Headers.Type))
	for _, field := range *AsObject(e.Cookies.Type) {
		if !e.isTransportOnlySessionCookie(field.Name) {
			bindings++
		}
	}
	if e.MapQueryParams != nil {
		bindings++
		if pMap == nil {
			verr.Add(e, "MapParams is set but Payload type is array. Payload type must be map or an object with a map attribute")
		} else {
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
	}
	metadataBindings := bindings
	if e.Body != nil && e.Body.Type != Empty {
		bindings++
		if pMap == nil && !IsArray(e.Body.Type) || pMap != nil && !IsMap(e.Body.Type) {
			verr.Add(e, "Payload type is %s but HTTP endpoint body is not.", kind)
		}
	}
	if e.FormRequest || e.MultipartRequest {
		bindings++
	}
	if bindings > 1 {
		verr.Add(e, "Payload type is %s and requires at most one transport binding across params, headers, cookies and body.", kind)
	}
	if metadataBindings == 0 && e.SkipRequestBodyEncodeDecode {
		verr.Add(e, "Payload type is %s but HTTP endpoint uses SkipRequestBodyEncodeDecode and does not define params, headers or cookies.", kind)
	}
}

func (e *HTTPEndpointExpr) validateObjectPayloadTransport(verr *eval.ValidationErrors) {
	if e.MapQueryParams != nil {
		if pAttr := *e.MapQueryParams; pAttr == "" {
			verr.Add(e, "MapParams is set to map entire payload but payload is an object. Payload must be a map.")
		} else if _, att := e.MethodExpr.Payload.FindAttribute(pAttr); att == nil {
			verr.Add(e, "MapParams is set to an attribute in Payload. But payload has no attribute with type map and name %s", pAttr)
		}
	}
	if e.Body == nil {
		return
	}
	props, selected := e.Body.Meta["origin:attribute"]
	if !selected {
		bObj := AsObject(e.Body.Type)
		if bObj == nil {
			return
		}
		for _, nat := range *bObj {
			props = append(props, AttributeName(nat.Name))
		}
	}
	for _, prop := range props {
		if _, att := e.MethodExpr.Payload.FindAttribute(prop); att == nil {
			verr.Add(e, "Body %q is not found in Payload.", prop)
		}
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
	if e.OpenAPIRequestBody != nil {
		e.validateOpenAPIRequestBody(verr)
	}
	optionalBody := e.Body
	if optionalBody == nil && (e.FormRequest || e.MultipartRequest || IsUnion(e.MethodExpr.Payload.Type)) {
		optionalBody = httpRequestBody(e)
	}
	if e.OptionalRequestBody && optionalBody != nil && !(IsObject(optionalBody.Type) || IsMap(optionalBody.Type) || IsUnion(optionalBody.Type)) {
		verr.Add(e, "OptionalRequestBody requires an object, map, or union request body.")
	}
	if e.OptionalRequestBody && (optionalBody == nil || optionalBody.Type == Empty) {
		verr.Add(e, "HTTP endpoint uses OptionalRequestBody but does not define a request body.")
	}
	if !e.OptionalRequestBody || optionalBody == nil || !IsObject(e.MethodExpr.Payload.Type) {
		return
	}
	props, selected := optionalBody.Meta["origin:attribute"]
	if selected && len(props) == 1 && e.MethodExpr.Payload.IsRequired(payloadKey(e.MethodExpr.Payload, props[0])) {
		verr.Add(e, "OptionalRequestBody requires the payload attribute mapped to the request body to be optional.")
	}
	if !selected && !e.MultipartRequest && hasRequiredBodyAttributes(optionalBody) {
		verr.Add(e, "OptionalRequestBody requires the request body to have no required attributes.")
	}
}

func (e *HTTPEndpointExpr) validateOpenAPIRequestBody(verr *eval.ValidationErrors) {
	verr.Merge(e.OpenAPIRequestBody.validateSchema("HTTP OpenAPI request body", e))
	if !e.SkipRequestBodyEncodeDecode {
		verr.Add(e, "OpenAPIRequestBody requires SkipRequestBodyEncodeDecode.")
	}
	if e.MultipartRequest {
		verr.Add(e, "HTTP endpoint cannot use OpenAPIRequestBody with MultipartRequest.")
	}
	if e.FormRequest {
		verr.Add(e, "HTTP endpoint cannot use OpenAPIRequestBody with FormRequest.")
	}
	if e.OptionalRequestBody {
		verr.Add(e, "HTTP endpoint cannot use OpenAPIRequestBody with OptionalRequestBody.")
	}
	if len(e.OpenAPIRequestBodyContentTypes) == 0 {
		verr.Add(e, "OpenAPIRequestBody requires at least one content type.")
	}
	seen := make(map[string]struct{}, len(e.OpenAPIRequestBodyContentTypes))
	for _, contentType := range e.OpenAPIRequestBodyContentTypes {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil || strings.TrimSpace(mediaType) == "" {
			verr.Add(e, "OpenAPIRequestBody content type %q is invalid.", contentType)
			continue
		}
		if _, ok := seen[contentType]; ok {
			verr.Add(e, "OpenAPIRequestBody content type %q is duplicated.", contentType)
		}
		seen[contentType] = struct{}{}
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

func (e *HTTPEndpointExpr) validateFileResponse(verr *eval.ValidationErrors) {
	if !e.FileResponse {
		return
	}
	if e.SkipResponseBodyEncodeDecode {
		verr.Add(e, "Endpoint cannot use FileResponse with SkipResponseBodyEncodeDecode.")
	}
	if e.SkipRequestBodyEncodeDecode {
		verr.Add(e, "Endpoint cannot use FileResponse with SkipRequestBodyEncodeDecode.")
	}
	if s := Root.API.GRPC.Service(e.Service.Name()); s != nil && s.Endpoint(e.Name()) != nil {
		verr.Add(e, "Endpoint cannot use FileResponse and define a gRPC transport.")
	}
	if e.IsJSONRPC() {
		verr.Add(e, "Endpoint cannot use FileResponse and define a JSON-RPC transport.")
	}
	if e.MethodExpr.IsStreaming() {
		verr.Add(e, "Endpoint cannot use FileResponse when method defines streaming.")
	}
	if _, ok := e.MethodExpr.Result.Type.(*ResultTypeExpr); ok {
		verr.Add(e, "Endpoint cannot use FileResponse when method result type defines views.")
	}
	if len(e.Responses) != 1 || e.Responses[0].Status.Code != StatusOK || e.Responses[0].Tag[0] != "" {
		verr.Add(e, "FileResponse requires exactly one untagged 200 application response; ServeContent owns protocol response statuses.")
	}
	for _, httpError := range e.HTTPErrors {
		if isFileProtocolStatus(httpError.Response.Status.Code) {
			verr.Add(e, "FileResponse error response status %d conflicts with ServeContent protocol behavior.", httpError.Response.Status.Code)
		}
	}
	for _, response := range e.Responses {
		if response == nil || response.Headers == nil {
			continue
		}
		for _, namedAttribute := range *AsObject(response.Headers.Type) {
			attribute := namedAttribute.Name
			header := response.Headers.ElemName(attribute)
			if isFileResponseOwnedHeader(header) {
				verr.Add(response, "FileResponse result attribute %q cannot map to transport-owned response header %q.", attribute, header)
			}
		}
	}
	for _, route := range e.Routes {
		if route.Method != "GET" && route.Method != "HEAD" {
			verr.Add(e, "FileResponse supports only explicit GET or HEAD routes. Got %q.", route.Method)
		}
	}
}

func isFileResponseOwnedHeader(header string) bool {
	switch strings.ToLower(header) {
	case "content-type", "content-length", "content-range", "accept-ranges", "last-modified":
		return true
	default:
		return false
	}
}

func isFileProtocolStatus(status int) bool {
	switch status {
	case StatusPartialContent, StatusNotModified, StatusPreconditionFailed, StatusRequestedRangeNotSatisfiable:
		return true
	default:
		return false
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
	if e.SSE.NotificationMethod != "" && !e.IsJSONRPC() {
		verr.Add(e, "SSENotificationMethod can only be used by JSON-RPC endpoints")
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
	e.validateJSONRPCResponseCarriers(verr)
	e.validateJSONRPCID(verr)
	e.validateJSONRPCWebSocketCredentials(verr)
	if e.MethodExpr.Stream == ServerStreamKind && e.SSE == nil &&
		e.MethodExpr.Payload.Type != Empty && e.MethodExpr.StreamingPayload.Type != Empty {
		verr.Add(e, "JSON-RPC WebSocket server streaming method %q cannot define both Payload and StreamingPayload. Use Payload for the request data", e.MethodExpr.Name)
	}
	if e.MethodExpr.Result == nil || e.MethodExpr.Result.Type == Empty || !hasJSONRPCIDField(e.MethodExpr.Result) {
		return
	}
	requestHasID := hasJSONRPCIDField(e.jsonrpcRequestPayload())
	if !requestHasID {
		verr.Add(e, "JSON-RPC method %q result defines an ID field but the request (payload) does not. Result may only have ID field if request does", e.MethodExpr.Name)
	}
}

func (e *HTTPEndpointExpr) validateRedirect(verr *eval.ValidationErrors) {
	if e.Redirect == nil {
		return
	}
	for _, response := range e.Responses {
		if response.Status.Code != e.Redirect.StatusCode {
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
	return att != nil && len(att.AllRequired()) > 0
}

// validateJSONRPCResponseCarriers admits only per-message application values.
// Inherited mappings have already been copied onto the prepared endpoint.
func (e *HTTPEndpointExpr) validateJSONRPCResponseCarriers(verr *eval.ValidationErrors) {
	responses := append([]*HTTPResponseExpr(nil), e.Responses...)
	for _, failure := range e.HTTPErrors {
		responses = append(responses, failure.Response)
	}
	for _, response := range responses {
		if response.Headers != nil && !response.Headers.IsEmpty() {
			verr.Add(response, "JSON-RPC response Header mappings are not supported; carry application values in result or error.data using Body")
		}
		if len(response.Cookies) > 0 {
			verr.Add(response, "JSON-RPC response Cookie mappings are not supported; carry application values in result or error.data using Body")
		}
	}
}
