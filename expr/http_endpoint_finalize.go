package expr

// Finalize is run post DSL execution. It merges response definitions, creates
// implicit endpoint parameters and initializes querystring parameters. It also
// flattens the error responses and makes sure the error types are all user
// types so that the response encoding code can properly use the type to infer
// the response that it needs to build.
func (e *HTTPEndpointExpr) Finalize() {
	e.normalizeJSONRPCServerStreamingPayload()
	e.finalizeRequirements()
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

func (e *HTTPEndpointExpr) finalizeRequirements() {
	e.inferSessionSecurityMappingsForAuths(e.MethodExpr.SessionAuths)
	if len(e.MethodExpr.Requirements) == 0 {
		return
	}
	e.Requirements = make([]*SecurityExpr, 0, len(e.MethodExpr.Requirements))
	for _, req := range e.MethodExpr.Requirements {
		dupReq := DupRequirement(req)
		for _, sch := range dupReq.Schemes {
			e.finalizeRequirementScheme(sch)
		}
		e.Requirements = append(e.Requirements, dupReq)
	}
}

func (e *HTTPEndpointExpr) finalizeRequirementScheme(sch *SchemeExpr) {
	var field string
	switch sch.Kind {
	case NoKind:
		return
	case BasicAuthKind:
		sch.In = "header"
		sch.Name = "Authorization"
		return
	case APIKeyKind:
		field = TaggedAttribute(e.MethodExpr.Payload, "security:apikey:"+sch.SchemeName)
	case JWTKind:
		field = TaggedAttribute(e.MethodExpr.Payload, "security:token")
	case OAuth2Kind:
		field = TaggedAttribute(e.MethodExpr.Payload, "security:accesstoken")
	}
	if sch.Kind == APIKeyKind && field == "" {
		if cookieName := e.transportOwnedCookieName(sch); cookieName != "" {
			sch.Name = cookieName
			sch.In = "cookie"
			return
		}
	}
	sch.Name, sch.In = findKey(e, field)
	if sch.Name != "" {
		return
	}
	// Initialize Authorization header implicitly defined via security DSL if mapping isn't explicit.
	sch.Name = "Authorization"
	attr := e.MethodExpr.Payload.Find(field)
	e.Headers.Type.(*Object).Set(field, attr)
	e.Headers.Map(sch.Name, field)
	if !e.MethodExpr.Payload.IsRequired(field) {
		return
	}
	if e.Headers.Validation == nil {
		e.Headers.Validation = &ValidationExpr{}
	}
	e.Headers.Validation.AddRequired(field)
}

func (e *HTTPEndpointExpr) transportOwnedCookieName(sch *SchemeExpr) string {
	if e == nil || e.MethodExpr == nil || sch == nil {
		return ""
	}
	for _, sessionAuth := range e.MethodExpr.validationSessionAuths() {
		for _, transport := range sessionAuth.Transports {
			if transport == nil || transport.Kind != SessionCookieTransportKind || transport.PayloadOwned() || transport.Scheme == nil {
				continue
			}
			if transport.Scheme.SchemeName == sch.SchemeName {
				if transport.HTTPName != "" {
					return transport.HTTPName
				}
				return transport.TransportAttributeName()
			}
		}
	}
	return ""
}

func (e *HTTPEndpointExpr) finalizeTransportBodies() {
	// Initialize the HTTP specific attributes with the corresponding payload attributes.
	initAttr(e.Params, e.MethodExpr.Payload)
	initAttr(e.Headers, e.MethodExpr.Payload)
	initAttr(e.Cookies, e.MethodExpr.Payload)

	e.Body = httpRequestBody(e)
	e.Body.Finalize()

	e.StreamingBody = httpStreamingBody(e)
	if e.StreamingBody != nil {
		e.StreamingBody.Finalize()
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
		e.MethodExpr.Payload = e.MethodExpr.StreamingPayload
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

func (e *HTTPEndpointExpr) inferSessionSecurityMappingsForAuths(sessionAuths []*SessionAuthExpr) {
	if len(sessionAuths) == 0 {
		return
	}
	for _, sessionAuth := range sessionAuths {
		for _, transport := range sessionAuth.Transports {
			if transport == nil || transport.Scheme == nil {
				continue
			}
			attributeName := transport.TransportAttributeName()
			if name, _ := findKey(e, attributeName); name != "" {
				continue
			}
			if transport.Kind == SessionCookieTransportKind {
				attr := e.MethodExpr.Payload.Find(attributeName)
				if attr == nil {
					if transport.PayloadOwned() {
						continue
					}
					attr = &AttributeExpr{Type: String}
					attr.AddMeta("loom:transport-only-session-cookie", "true")
				}
				cookieName := transport.HTTPName
				if cookieName == "" {
					cookieName = attributeName
				}
				e.Cookies.Type.(*Object).Set(attributeName, attr)
				e.Cookies.Map(cookieName, attributeName)
				if transport.PayloadOwned() && e.MethodExpr.Payload.IsRequired(attributeName) {
					if e.Cookies.Validation == nil {
						e.Cookies.Validation = &ValidationExpr{}
					}
					e.Cookies.Validation.AddRequired(attributeName)
				}
			}
		}
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
