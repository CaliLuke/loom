package ir

import (
	"maps"
	"slices"
	"strconv"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

// BuildBodyTypes analyzes endpoint request and response bodies plus referenced components.
func BuildBodyTypes(api *expr.APIExpr, types []expr.UserType, resultTypes []*expr.ResultTypeExpr, options ...AnalyzerOption) *BodyTypes {
	a := NewAnalyzer(api.ExampleGenerator, openapi.ClosedObjectModeFromExpr(api.Meta), options...)
	bodies := &BodyTypes{
		Services:  make(map[string]map[string]*EndpointBodies),
		prepared:  make(map[string]*transportir.Service),
		locations: make(map[*expr.AttributeExpr]*Schema),
	}

	analyzeComponentTypes(a, types, resultTypes)
	analyzeServiceBodies(a, api, bodies)
	for _, service := range orderedStringKeys(bodies.Services) {
		endpoints := bodies.Services[service]
		for _, method := range orderedStringKeys(endpoints) {
			endpoint := endpoints[method]
			endpoint.async = analyzeAsyncSchemas(a, endpoint.prepared)
			endpoint.asyncAnalyzer = a
		}
	}
	a.finalizeRepresentations()

	bodies.Components = a.schemas
	for _, endpoints := range bodies.Services {
		for _, endpoint := range endpoints {
			endpoint.asyncComponents = a.schemas
		}
	}
	return bodies
}

func analyzeComponentTypes(a *Analyzer, types []expr.UserType, resultTypes []*expr.ResultTypeExpr) {
	for _, t := range types {
		if !mustGenerateType(t.Attribute().Meta) {
			continue
		}
		attribute := &expr.AttributeExpr{Type: t}
		schema := a.analyzeStandaloneSchema(attribute, exampleContext("component-type", t.ID()))
		a.declarationRoots[t.ID()] = schema
		a.registrations = append(a.registrations, schema)
	}
	for _, t := range resultTypes {
		if !mustGenerateType(t.Attribute().Meta) {
			continue
		}
		attribute := &expr.AttributeExpr{Type: t}
		schema := a.analyzeStandaloneSchema(attribute, exampleContext("component-result-type", t.ID()))
		a.declarationRoots[t.ID()] = schema
		a.registrations = append(a.registrations, schema)
	}
}

func analyzeServiceBodies(a *Analyzer, api *expr.APIExpr, bodies *BodyTypes) {
	for _, svc := range api.HTTP.Services {
		if !openapi.MustGenerate(svc.Meta) || !openapi.MustGenerate(svc.ServiceExpr.Meta) {
			continue
		}
		serviceIR, err := representation.PrepareService(svc, nil)
		if err != nil {
			panic(err)
		}
		if !a.customExampleValue {
			if err := representation.PrepareServiceExamples(serviceIR, svc, a.rand); err != nil {
				panic(err)
			}
			for _, endpoint := range serviceIR.Endpoints {
				if err := representation.PrepareEndpointExamples(endpoint, a.rand); err != nil {
					panic(err)
				}
			}
		}
		bodies.prepared[svc.Name()] = serviceIR
		serviceBodies := make(map[string]*EndpointBodies, len(serviceIR.Endpoints))
		for _, endpoint := range serviceIR.Endpoints {
			if !endpoint.Generate || !endpoint.MethodGenerate {
				continue
			}
			serviceBodies[endpoint.Name] = analyzeEndpointBodies(a, endpoint)
			serviceBodies[endpoint.Name].locations = bodies.locations
			analyzeLocationSchemas(a, endpoint, bodies.locations)
		}
		bodies.Services[svc.Name()] = serviceBodies
	}
}

func analyzeEndpointBodies(a *Analyzer, endpoint *transportir.Endpoint) *EndpointBodies {
	req := analyzeRequestBody(a, endpoint)
	responseBodies := analyzeResponseBodies(a, endpoint)
	bodies := &EndpointBodies{
		prepared:       endpoint,
		RequestBody:    req,
		ResponseBodies: responseBodies,
	}
	bodies.requestMedia = analyzeDocumentMedia(a, endpoint.Request.DocumentBody, endpoint.Request.DocumentValues, schemaUsageRequest, "request-schema")
	if len(endpoint.Request.DocumentContentTypes) > 0 && bodies.requestMedia != nil {
		bodies.RequestBody = bodies.requestMedia[endpoint.Request.DocumentContentTypes[0]]
	}
	bodies.responseMedia = make(map[*transportir.ResponseStatus]map[string]*Schema)
	for _, responses := range [][]*transportir.ResponseStatus{endpoint.Response.Responses, endpoint.Response.ErrorResponses} {
		for _, response := range responses {
			bodies.responseMedia[response] = analyzeDocumentMedia(a, response.DocumentBody, response.DocumentValues, schemaUsageResponse, "response-schema", strconv.Itoa(response.StatusCode))
		}
	}
	indexes := make(map[int]int)
	for _, responses := range [][]*transportir.ResponseStatus{endpoint.Response.Responses, endpoint.Response.ErrorResponses} {
		for _, response := range responses {
			index := indexes[response.StatusCode]
			indexes[response.StatusCode]++
			if len(response.ContentTypes) > 0 && bodies.responseMedia[response] != nil {
				bodies.ResponseBodies[response.StatusCode][index] = bodies.responseMedia[response][response.ContentTypes[0]]
			}
		}
	}
	return bodies
}

func analyzeRequestBody(a *Analyzer, endpoint *transportir.Endpoint) *Schema {
	body := expr.UnwrapInlineHTTPBody(endpoint.Request.Body)
	if endpoint.Request.DocumentBody != nil {
		body = endpoint.Request.DocumentBody
	}
	requestAttr := attributeForSchemaUsage(body, schemaUsageRequest)
	requestContext := attributeExampleContext(requestAttr, a.closeObjects, "request-schema")
	target := endpoint.Request.BodyValue
	if endpoint.Request.DocumentBody != nil {
		target = endpoint.Request.DocumentValue
	}
	var req *Schema
	if len(endpoint.Request.DocumentValues) == 0 {
		req = a.analyzePreparedOccurrence(requestAttr, requestContext, target)
	}
	if endpoint.Request.StreamingBody == nil {
		return req
	}
	streamingAttr := attributeForSchemaUsage(endpoint.Request.StreamingBody, schemaUsageRequest)
	streamingContext := attributeExampleContext(streamingAttr, a.closeObjects, "streaming-request-schema")
	streaming := a.analyzePreparedOccurrence(streamingAttr, streamingContext, endpoint.Request.StreamingValue)
	return mergeStreamingBodyNote(req, streaming)
}

func analyzeResponseBodies(a *Analyzer, endpoint *transportir.Endpoint) map[int][]*Schema {
	responseBodies := make(map[int][]*Schema)
	appendBodies := func(responses []*transportir.ResponseStatus, projectSSE bool) {
		for _, resp := range responses {
			if len(resp.DocumentValues) > 0 {
				responseBodies[resp.StatusCode] = append(responseBodies[resp.StatusCode], nil)
				continue
			}
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
				a.analyzePreparedOccurrence(body, context, resp.DocumentValue),
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
		target := representation.PrepareStreamSchema(endpoint, attr, false)
		if !a.customExampleValue {
			if err := representation.PrepareTargetExamples(target, attr, exampleGeneratorForAttribute(a.rand, attr, a.closeObjects, context)); err != nil {
				panic(err)
			}
		}
		schema.OneOf = append(
			schema.OneOf,
			a.analyzePreparedOccurrence(attributeForSchemaUsage(attr, schemaUsageResponse), context, target),
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

func analyzeLocationSchemas(a *Analyzer, endpoint *transportir.Endpoint, locations map[*expr.AttributeExpr]*Schema) {
	for _, group := range []struct {
		location   string
		parameters []*transportir.Parameter
	}{
		{"path", endpoint.Request.PathParams}, {"query", endpoint.Request.QueryParams}, {"header", endpoint.Request.Headers}, {"cookie", endpoint.Request.Cookies},
	} {
		for _, parameter := range group.parameters {
			if endpoint.Security.IsParameter(group.location, parameter.HTTPName) {
				continue
			}
			context := attributeExampleContext(parameter.Attribute, a.closeObjects, "parameter", group.location, parameter.HTTPName)
			locations[parameter.Attribute] = a.analyzePreparedOccurrence(parameter.Attribute, context, parameter.Value)
		}
	}
	for _, responses := range [][]*transportir.ResponseStatus{endpoint.Response.Responses, endpoint.Response.ErrorResponses} {
		for _, response := range responses {
			for _, header := range response.Headers {
				context := attributeExampleContext(header.Attribute, a.closeObjects, "response-header", header.HTTPName)
				locations[header.Attribute] = a.analyzePreparedOccurrence(header.Attribute, context, header.Value)
			}
		}
	}
}

func analyzeDocumentMedia(a *Analyzer, attribute *expr.AttributeExpr, targets map[string]*transportir.ValueTarget, usage schemaUsage, contextParts ...string) map[string]*Schema {
	if len(targets) == 0 {
		return nil
	}
	result := make(map[string]*Schema, len(targets))
	attribute = attributeForSchemaUsage(attribute, usage)
	context := attributeExampleContext(attribute, a.closeObjects, contextParts...)
	for _, media := range slices.Sorted(maps.Keys(targets)) {
		result[media] = a.analyzePreparedOccurrence(attribute, context, targets[media])
	}
	return result
}
