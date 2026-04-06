package transportir

import "github.com/CaliLuke/loom/expr"

func BuildService(svc *expr.GRPCServiceExpr) *Service {
	if svc == nil {
		return nil
	}
	service := &Service{
		Name:        svc.Name(),
		Description: svc.Description(),
		Expr:        svc,
		Endpoints:   make([]*Endpoint, 0, len(svc.GRPCEndpoints)),
	}
	for _, endpoint := range svc.GRPCEndpoints {
		service.Endpoints = append(service.Endpoints, BuildEndpoint(service, endpoint))
	}
	return service
}

func BuildEndpoint(service *Service, endpoint *expr.GRPCEndpointExpr) *Endpoint {
	if endpoint == nil {
		return nil
	}
	ir := &Endpoint{
		Name:         endpoint.Name(),
		Description:  endpoint.Description(),
		Method:       endpoint.MethodExpr,
		Service:      service,
		Requirements: append([]*expr.SecurityExpr(nil), endpoint.Requirements...),
		Request: &Request{
			Payload:          expr.DupAtt(endpoint.MethodExpr.Payload),
			Message:          expr.DupAtt(endpoint.Request),
			StreamingPayload: expr.DupAtt(endpoint.MethodExpr.StreamingPayload),
			StreamingMessage: expr.DupAtt(endpoint.StreamingRequest),
			Metadata:         expr.DupMappedAtt(endpoint.Metadata),
		},
		Response: &Response{
			Result:      expr.DupAtt(endpoint.MethodExpr.Result),
			Message:     expr.DupAtt(endpoint.Response.Message),
			StatusCode:  endpoint.Response.StatusCode,
			Description: endpoint.Response.Description,
			Headers:     expr.DupMappedAtt(endpoint.Response.Headers),
			Trailers:    expr.DupMappedAtt(endpoint.Response.Trailers),
		},
		Errors: make([]*Error, 0, len(endpoint.GRPCErrors)),
		Stream: &Stream{
			IsStreaming:        endpoint.MethodExpr.IsStreaming(),
			IsPayloadStreaming: endpoint.MethodExpr.IsPayloadStreaming(),
		},
	}
	for _, grpcErr := range endpoint.GRPCErrors {
		ir.Errors = append(ir.Errors, &Error{
			Name:      grpcErr.Name,
			Type:      grpcErr.Type,
			Attribute: expr.DupAtt(grpcErr.AttributeExpr),
			Response: &Response{
				Result:      expr.DupAtt(grpcErr.AttributeExpr),
				Message:     expr.DupAtt(grpcErr.Response.Message),
				StatusCode:  grpcErr.Response.StatusCode,
				Description: grpcErr.Response.Description,
				Headers:     expr.DupMappedAtt(grpcErr.Response.Headers),
				Trailers:    expr.DupMappedAtt(grpcErr.Response.Trailers),
			},
		})
	}
	return ir
}
