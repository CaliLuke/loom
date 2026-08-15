package ir

import (
	"strconv"

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
		a.AnalyzeSchemaWithContext(
			&expr.AttributeExpr{Type: t},
			exampleContext("component-type", t.ID()),
		)
	}
	for _, t := range resultTypes {
		if !mustGenerateType(t.Attribute().Meta) {
			continue
		}
		a.AnalyzeSchemaWithContext(
			&expr.AttributeExpr{Type: t},
			exampleContext("component-result-type", t.ID()),
		)
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
	body := endpoint.Request.Body
	if endpoint.Request.DocumentBody != nil {
		body = endpoint.Request.DocumentBody
	}
	requestAttr := attributeForSchemaUsage(body, schemaUsageRequest)
	requestContext := attributeExampleContext(requestAttr, a.closeObjects, "request-schema")
	req := a.AnalyzeSchemaWithContext(requestAttr, requestContext)
	if endpoint.Request.StreamingBody == nil {
		return req
	}
	streamingAttr := attributeForSchemaUsage(endpoint.Request.StreamingBody, schemaUsageRequest)
	streamingContext := attributeExampleContext(streamingAttr, a.closeObjects, "streaming-request-schema")
	streaming := a.AnalyzeSchemaWithContext(streamingAttr, streamingContext)
	return mergeStreamingBodyNote(req, streaming)
}

func analyzeResponseBodies(a *Analyzer, endpoint *transportir.Endpoint) map[int][]*Schema {
	responseBodies := make(map[int][]*Schema)
	appendBodies := func(responses []*transportir.ResponseStatus, projectSSE bool) {
		for _, resp := range responses {
			if projectSSE && endpoint.Stream != nil && endpoint.Stream.SSE != nil && len(endpoint.Stream.SSE.Projections) > 0 {
				responseBodies[resp.StatusCode] = append(responseBodies[resp.StatusCode], analyzeSSEProjectionSchema(a, endpoint))
				continue
			}
			body := attributeForSchemaUsage(resp.DocumentBody, schemaUsageResponse)
			context := attributeExampleContext(
				body,
				a.closeObjects,
				"response-schema",
				strconv.Itoa(resp.StatusCode),
			)
			responseBodies[resp.StatusCode] = append(
				responseBodies[resp.StatusCode],
				a.AnalyzeSchemaWithContext(body, context),
			)
		}
	}
	appendBodies(endpoint.Response.Responses, true)
	appendBodies(endpoint.Response.ErrorResponses, false)
	return responseBodies
}

func analyzeSSEProjectionSchema(a *Analyzer, endpoint *transportir.Endpoint) *Schema {
	attrs, err := sseProjectionAttributes(endpoint)
	if err != nil {
		panic(err)
	}
	schema := &Schema{OneOf: make([]*Schema, 0, len(attrs))}
	for index, attr := range attrs {
		context := endpointSchemaExampleContext(endpoint, "sse-projection", strconv.Itoa(index))
		schema.OneOf = append(
			schema.OneOf,
			a.AnalyzeSchemaWithContext(attributeForSchemaUsage(attr, schemaUsageResponse), context),
		)
	}
	return schema
}

func endpointSchemaExampleContext(endpoint *transportir.Endpoint, parts ...string) string {
	serviceName := ""
	methodName := ""
	if endpoint != nil {
		methodName = endpoint.Name
		if endpoint.Service != nil {
			serviceName = endpoint.Service.Name
		}
	}
	return exampleContext(append([]string{"service", serviceName, "method", methodName}, parts...)...)
}
