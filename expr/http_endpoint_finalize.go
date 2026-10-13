package expr

// Finalize is run post DSL execution. It merges response definitions, creates
// implicit endpoint parameters and initializes querystring parameters. It also
// flattens the error responses and makes sure the error types are all user
// types so that the response encoding code can properly use the type to infer
// the response that it needs to build.
func (e *HTTPEndpointExpr) Finalize() {
	e.ensureWireBindings()
	e.normalizeJSONRPCServerStreamingPayload()
	e.finalizeTransportBodies()
	e.finalizeJSONRPCBodyState()
	e.finalizeResponsesAndErrors()
}

func (e *HTTPEndpointExpr) normalizeJSONRPCServerStreamingPayload() {
	// For JSON-RPC WebSocket endpoints with server streaming and non-streaming payload,
	// move the payload to streaming payload. This is because the payload is sent as
	// JSON-RPC messages after the WebSocket connection is established, making it
	// effectively a streaming payload from the transport perspective.
	if _, isJSONRPC := e.MethodExpr.Meta["jsonrpc"]; !isJSONRPC || e.MethodExpr.Stream != ServerStreamKind || e.SSE != nil {
		return
	}
	if e.MethodExpr.Payload.Type == Empty || e.MethodExpr.StreamingPayload.Type != Empty {
		return
	}
	e.MethodExpr.StreamingPayload = e.MethodExpr.Payload
	e.MethodExpr.Payload = &AttributeExpr{Type: Empty}
	e.MethodExpr.Stream = BidirectionalStreamKind
}

func (e *HTTPEndpointExpr) finalizeTransportBodies() {
	// Initialize the HTTP specific attributes with the corresponding payload attributes.
	initAttr(e.Params, e.MethodExpr.Payload)
	initAttr(e.Headers, e.MethodExpr.Payload)
	initAttr(e.Cookies, e.MethodExpr.Payload)

	explicitBody := e.Body != nil
	e.Body = httpRequestBody(e)
	if explicitBody {
		bindExplicitHTTPRequestBody(e.Body, responseBodyAttribute(e.Body, e.MethodExpr.Payload))
	}
	e.Body.Finalize()
	if e.OpenAPIRequestBody != nil {
		e.OpenAPIRequestBody.Finalize()
	}

	e.StreamingBody = httpStreamingBody(e)
	if e.StreamingBody != nil {
		e.StreamingBody.Finalize()
	}
}

func bindExplicitHTTPRequestBody(body, source *AttributeExpr) {
	bindValueSource(body, source)
	if source == nil {
		return
	}
	bodyObject := AsObject(body.Type)
	sourceObject := AsObject(source.Type)
	if bodyObject == nil || sourceObject == nil {
		return
	}
	for _, named := range *bodyObject {
		_, sourceMember := objectAttribute(sourceObject, named.Name)
		if sourceMember != nil {
			bindValueSource(named.Attribute, sourceMember)
		}
	}
}

func (e *HTTPEndpointExpr) finalizeJSONRPCBodyState() {
	// For JSON-RPC, WebSocket handling is managed at the server level.
	// Each endpoint is treated as a standard HTTP endpoint; the server is responsible
	// for upgrading the connection, decoding incoming JSON-RPC requests, and dispatching
	// them to the appropriate endpoint handlers.
	if !e.IsJSONRPC() {
		return
	}
	if e.MethodExpr.IsPayloadStreaming() {
		e.MethodExpr.Payload = e.jsonrpcRequestPayload()
		e.Body = e.StreamingBody
	}
	e.PayloadIDAttribute = jsonrpcIDAttributeName(e.MethodExpr.Payload)
	e.ResultIDAttribute = jsonrpcIDAttributeName(e.MethodExpr.Result)
}

func (e *HTTPEndpointExpr) finalizeResponsesAndErrors() {
	for _, r := range e.Responses {
		r.Finalize(e, e.MethodExpr.Result)
		r.Body = httpResponseBody(e, r)
		r.Body.Finalize()
	}

	for _, herr := range e.HTTPErrors {
		herr.Finalize(e)
	}
}

func jsonrpcIDAttributeName(att *AttributeExpr) string {
	if att == nil {
		return ""
	}
	obj := AsObject(att.Type)
	if obj == nil {
		return ""
	}
	for _, nat := range *obj {
		if _, ok := nat.Attribute.Meta["jsonrpc:id"]; ok {
			return nat.Name
		}
	}
	return ""
}
