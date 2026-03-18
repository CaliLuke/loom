package expr

// NewJSONRPCHTTPService builds an HTTP service expression for a synthesized
// JSON-RPC service. The returned service uses a single POST route for all
// methods and pre-initializes endpoint body and mapped attributes so transport
// analyzers can consume it directly.
func NewJSONRPCHTTPService(service *ServiceExpr, routePath string) *HTTPServiceExpr {
	if service == nil {
		panic("nil service")
	}
	if routePath == "" {
		routePath = "/"
	}
	if service.Meta == nil {
		service.Meta = MetaExpr{}
	}
	service.Meta["jsonrpc:service"] = []string{}

	httpService := &HTTPServiceExpr{
		ServiceExpr: service,
		JSONRPCRoute: &RouteExpr{
			Method: "POST",
			Path:   routePath,
		},
		Paths: []string{},
		SSE:   &HTTPSSEExpr{},
	}
	httpService.JSONRPCRoute.Endpoint = &HTTPEndpointExpr{Service: httpService}

	for _, method := range service.Methods {
		method.Service = service
		endpoint := &HTTPEndpointExpr{
			MethodExpr: method,
			Service:    httpService,
			Meta: MetaExpr{
				"jsonrpc": []string{},
			},
			Body:    method.Payload,
			Params:  NewEmptyMappedAttributeExpr(),
			Headers: NewEmptyMappedAttributeExpr(),
			Cookies: NewEmptyMappedAttributeExpr(),
		}
		endpoint.Routes = []*RouteExpr{{
			Method:   "POST",
			Path:     routePath,
			Endpoint: endpoint,
		}}
		if method.Stream == ServerStreamKind {
			endpoint.SSE = &HTTPSSEExpr{}
		}
		httpService.HTTPEndpoints = append(httpService.HTTPEndpoints, endpoint)
	}

	return httpService
}
