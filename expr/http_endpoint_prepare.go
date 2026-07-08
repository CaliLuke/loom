package expr

func (e *HTTPEndpointExpr) initTransportAttributes() {
	if e.Headers == nil {
		e.Headers = NewEmptyMappedAttributeExpr()
	}
	if e.Cookies == nil {
		e.Cookies = NewEmptyMappedAttributeExpr()
	}
	if e.Params == nil {
		e.Params = NewEmptyMappedAttributeExpr()
	}
}

func (e *HTTPEndpointExpr) inheritTransportAttributes() {
	headers := NewEmptyMappedAttributeExpr()
	headers.Merge(Root.API.HTTP.Headers)
	headers.Merge(e.Service.Headers)

	cookies := NewEmptyMappedAttributeExpr()
	cookies.Merge(Root.API.HTTP.Cookies)
	cookies.Merge(e.Service.Cookies)

	params := NewEmptyMappedAttributeExpr()
	params.Merge(Root.API.HTTP.Params)
	params.Merge(e.Service.Params)

	e.inheritCanonicalEndpointTransport(headers, cookies, params)

	headers.Merge(e.Headers)
	cookies.Merge(e.Cookies)
	params.Merge(e.Params)

	e.Headers = headers
	e.Cookies = cookies
	e.Params = params
}

func (e *HTTPEndpointExpr) inheritCanonicalEndpointTransport(headers, cookies, params *MappedAttributeExpr) {
	parent := e.Service.Parent()
	if parent == nil {
		return
	}
	canonical := parent.CanonicalEndpoint()
	if canonical == nil {
		return
	}
	canonical.Prepare()
	if e.HasAbsoluteRoutes() {
		return
	}
	headers.Merge(canonical.Headers)
	cookies.Merge(canonical.Cookies)
	cpp := canonical.PathParams()
	params.Merge(cpp)
	e.inheritCanonicalPathParams(canonical, cpp)
}

func (e *HTTPEndpointExpr) inheritCanonicalPathParams(canonical *HTTPEndpointExpr, pathParams *MappedAttributeExpr) {
	WalkMappedAttr(pathParams, func(name, _ string, _ *AttributeExpr) error { // nolint: errcheck
		att := canonical.MethodExpr.Payload.Find(name)
		if att == nil {
			return nil
		}
		if e.MethodExpr.Payload.Type == Empty {
			e.MethodExpr.Payload.Type = &Object{}
		}
		object := AsObject(e.MethodExpr.Payload.Type)
		if object == nil || object.Attribute(name) != nil {
			return nil
		}
		if canonical.MethodExpr.Payload.IsRequired(name) {
			if e.MethodExpr.Payload.Validation == nil {
				e.MethodExpr.Payload.Validation = &ValidationExpr{}
			}
			e.MethodExpr.Payload.Validation.AddRequired(name)
		}
		object.Set(name, att)
		return nil
	})
}

func (e *HTTPEndpointExpr) ensureRouteParams() {
	for _, route := range e.Routes {
		for _, param := range route.Params() {
			if e.Params.Find(param) != nil {
				continue
			}
			e.Params.Merge(NewMappedAttributeExpr(&AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{
						Name:      param,
						Attribute: &AttributeExpr{Type: String},
					},
				},
			}))
		}
	}
}

func (e *HTTPEndpointExpr) ensureDefaultResponse() {
	if len(e.Responses) > 0 {
		return
	}
	status := StatusOK
	if e.Redirect != nil {
		status = e.Redirect.StatusCode
	} else if e.MethodExpr.Result.Type == Empty && !e.SkipResponseBodyEncodeDecode {
		status = StatusNoContent
	}
	e.Responses = []*HTTPResponseExpr{{StatusCode: status}}
}

func (e *HTTPEndpointExpr) inheritSSE() {
	if e.MethodExpr.Stream != ServerStreamKind || e.SSE != nil {
		return
	}
	if e.Service.SSE != nil {
		e.SSE = e.Service.SSE
		return
	}
	if Root.API.HTTP.SSE != nil {
		e.SSE = Root.API.HTTP.SSE
	}
}

func (e *HTTPEndpointExpr) inheritHTTPErrors() {
	methodErrors := make(map[string]struct{}, len(e.HTTPErrors))
	for _, httpError := range e.HTTPErrors {
		methodErrors[httpError.Name] = struct{}{}
	}
	for _, methodError := range e.MethodExpr.Errors {
		if _, ok := methodErrors[methodError.Name]; ok {
			continue
		}
		methodErrors[methodError.Name] = struct{}{}
		if e.appendServiceHTTPErrors(methodError.Name) {
			continue
		}
		e.appendAPIHTTPErrors(methodError.Name, e.Service.Root.Errors)
	}
	for _, serviceError := range e.Service.ServiceExpr.Errors {
		if _, ok := methodErrors[serviceError.Name]; ok {
			continue
		}
		if e.appendServiceHTTPErrors(serviceError.Name) {
			continue
		}
		e.appendAPIHTTPErrors(serviceError.Name, Root.API.HTTP.Errors)
	}
}

func (e *HTTPEndpointExpr) appendServiceHTTPErrors(name string) bool {
	for _, httpError := range e.Service.HTTPErrors {
		if name != httpError.Name {
			continue
		}
		e.HTTPErrors = append(e.HTTPErrors, httpError.Dup())
		return true
	}
	return false
}

func (e *HTTPEndpointExpr) appendAPIHTTPErrors(name string, errors []*HTTPErrorExpr) {
	for _, httpError := range errors {
		if name == httpError.Name {
			e.HTTPErrors = append(e.HTTPErrors, httpError.Dup())
		}
	}
}

func (e *HTTPEndpointExpr) forceWebSocketRouteMethod() {
	if e.MethodExpr.IsStreaming() && e.SSE == nil && len(e.Routes) > 0 && e.Routes[0].Method == "" {
		e.Routes[0].Method = "GET"
	}
}

func (e *HTTPEndpointExpr) prepareResponses() {
	for _, response := range e.Responses {
		response.Prepare()
	}
	for _, httpError := range e.HTTPErrors {
		httpError.Response.Prepare()
	}
}
