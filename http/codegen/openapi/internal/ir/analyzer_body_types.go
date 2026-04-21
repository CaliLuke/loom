package ir

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

// BuildBodyTypes analyzes endpoint request and response bodies plus referenced components.
func BuildBodyTypes(api *expr.APIExpr, types []expr.UserType, resultTypes []*expr.ResultTypeExpr, options ...AnalyzerOption) *BodyTypes {
	a := NewAnalyzer(api.ExampleGenerator, openapi.ClosedObjectModeFromExpr(api.Meta), options...)
	bodies := &BodyTypes{
		Services: make(map[string]map[string]*EndpointBodies),
	}

	analyzeComponentTypes(a, types, resultTypes)
	analyzeServiceBodies(a, api, bodies)

	bodies.Components = a.schemas
	return bodies
}

func analyzeComponentTypes(a *Analyzer, types []expr.UserType, resultTypes []*expr.ResultTypeExpr) {
	for _, t := range types {
		if !mustGenerateType(t.Attribute().Meta) {
			continue
		}
		a.AnalyzeSchema(&expr.AttributeExpr{Type: t})
	}
	for _, t := range resultTypes {
		if !mustGenerateType(t.Attribute().Meta) {
			continue
		}
		a.AnalyzeSchema(&expr.AttributeExpr{Type: t})
	}
}

func analyzeServiceBodies(a *Analyzer, api *expr.APIExpr, bodies *BodyTypes) {
	for _, svc := range api.HTTP.Services {
		if !openapi.MustGenerate(svc.Meta) || !openapi.MustGenerate(svc.ServiceExpr.Meta) {
			continue
		}
		serviceIR := transportir.BuildService(svc)
		serviceBodies := make(map[string]*EndpointBodies, len(serviceIR.Endpoints))
		for _, endpoint := range serviceIR.Endpoints {
			if !endpoint.Generate || !endpoint.MethodGenerate {
				continue
			}
			serviceBodies[endpoint.Name] = analyzeEndpointBodies(a, endpoint)
		}
		bodies.Services[svc.Name()] = serviceBodies
	}
}

func analyzeEndpointBodies(a *Analyzer, endpoint *transportir.Endpoint) *EndpointBodies {
	req := analyzeRequestBody(a, endpoint)
	responseBodies := analyzeResponseBodies(a, endpoint)
	return &EndpointBodies{
		RequestBody:    req,
		ResponseBodies: responseBodies,
	}
}

func analyzeRequestBody(a *Analyzer, endpoint *transportir.Endpoint) *Schema {
	req := a.AnalyzeSchema(attributeForSchemaUsage(endpoint.Request.Body, schemaUsageRequest))
	if endpoint.Request.StreamingBody == nil {
		return req
	}
	streaming := a.AnalyzeSchema(attributeForSchemaUsage(endpoint.Request.StreamingBody, schemaUsageRequest))
	return mergeStreamingBodyNote(req, streaming)
}

func analyzeResponseBodies(a *Analyzer, endpoint *transportir.Endpoint) map[int][]*Schema {
	responseBodies := make(map[int][]*Schema)
	appendBodies := func(responses []*transportir.ResponseStatus) {
		for _, resp := range responses {
			body := attributeForSchemaUsage(resp.DocumentBody, schemaUsageResponse)
			responseBodies[resp.StatusCode] = append(responseBodies[resp.StatusCode], a.AnalyzeSchema(body))
		}
	}
	appendBodies(endpoint.Response.Responses)
	appendBodies(endpoint.Response.ErrorResponses)
	return responseBodies
}
