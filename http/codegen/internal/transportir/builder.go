package transportir

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func BuildService(service *expr.HTTPServiceExpr) *Service {
	if service == nil {
		return nil
	}
	ir := &Service{
		Name:            service.Name(),
		Meta:            service.Meta,
		ServiceMeta:     service.ServiceExpr.Meta,
		Generate:        openapi.MustGenerate(service.Meta),
		ServiceGenerate: openapi.MustGenerate(service.ServiceExpr.Meta),
	}
	for _, endpoint := range service.HTTPEndpoints {
		ir.Endpoints = append(ir.Endpoints, BuildEndpoint(endpoint))
		ir.Endpoints[len(ir.Endpoints)-1].Service = ir
	}
	return ir
}

func BuildEndpoint(endpoint *expr.HTTPEndpointExpr) *Endpoint {
	if endpoint == nil {
		return nil
	}
	ir := &Endpoint{
		Name:           endpoint.Name(),
		MethodName:     endpoint.MethodExpr.Name,
		Description:    endpoint.Description(),
		Meta:           endpoint.Meta,
		MethodMeta:     endpoint.MethodExpr.Meta,
		MethodDocs:     endpoint.MethodExpr.Docs,
		Generate:       openapi.MustGenerate(endpoint.Meta),
		MethodGenerate: openapi.MustGenerate(endpoint.MethodExpr.Meta),
		IsJSONRPC:      endpoint.IsJSONRPC(),
	}
	if endpoint.Service != nil {
		ir.Service = &Service{
			Name:            endpoint.Service.Name(),
			Meta:            endpoint.Service.Meta,
			ServiceMeta:     endpoint.Service.ServiceExpr.Meta,
			Generate:        openapi.MustGenerate(endpoint.Service.Meta),
			ServiceGenerate: openapi.MustGenerate(endpoint.Service.ServiceExpr.Meta),
		}
	}
	ir.Request = buildRequest(endpoint)
	ir.Response = buildResponse(endpoint)
	ir.Routes = buildRoutes(endpoint)
	ir.Stream = buildStream(endpoint)
	ir.Redirect = buildRedirect(endpoint)
	ir.Security = buildSecurity(endpoint)
	return ir
}
