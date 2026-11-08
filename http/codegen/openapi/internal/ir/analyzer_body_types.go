package ir

import (
	"fmt"
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

	a.initializeDiscriminatorHints(api, types, resultTypes)
	analyzeComponentTypes(a, types, resultTypes)
	if a.Err() != nil {
		bodies.Err = a.Err()
		return bodies
	}
	a.recordDiscriminatorChildren(types, resultTypes)
	analyzeServiceBodies(a, api, bodies)
	if a.Err() != nil {
		bodies.Err = a.Err()
		return bodies
	}
	for _, service := range orderedStringKeys(bodies.Services) {
		endpoints := bodies.Services[service]
		for _, method := range orderedStringKeys(endpoints) {
			endpoint := endpoints[method]
			endpoint.async = analyzeAsyncSchemas(a, endpoint.prepared, endpoint.event)
			endpoint.asyncAnalyzer = a
		}
	}
	a.finalizeRepresentations()

	bodies.Err = a.Err()
	bodies.Components = a.schemas
	bodies.DiscriminatorChildren = a.discriminatorChildren
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
		if a.err != nil {
			return
		}
		a.declarationRoots[t.ID()] = schema
		a.registrations = append(a.registrations, schema)
	}
	for _, t := range resultTypes {
		if !mustGenerateType(t.Attribute().Meta) {
			continue
		}
		attribute := &expr.AttributeExpr{Type: t}
		schema := a.analyzeStandaloneSchema(attribute, exampleContext("component-result-type", t.ID()))
		if a.err != nil {
			return
		}
		a.declarationRoots[t.ID()] = schema
		a.registrations = append(a.registrations, schema)
	}
}

func analyzeServiceBodies(a *Analyzer, api *expr.APIExpr, bodies *BodyTypes) {
	for _, svc := range api.HTTP.Services {
		if !openapi.MustGenerate(svc.Meta) || !openapi.MustGenerate(svc.ServiceExpr.Meta) {
			continue
		}
		serviceIR, err := representation.PrepareService(svc, nil, a.semantic)
		if err != nil {
			panic(err)
		}
		if !a.customExampleValue {
			if err := representation.PrepareServiceExamples(serviceIR, svc, a.rand); err != nil {
				panic(err)
			}
			for _, endpoint := range serviceIR.Endpoints {
				if err := representation.PrepareEndpointExamples(endpoint, a.rand); err != nil {
					a.err = fmt.Errorf("service %q method %q: %w", svc.Name(), endpoint.MethodName, err)
					return
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
	event := analyzeEventOutcome(a, endpoint)
	responseBodies := analyzeResponseBodies(a, endpoint, event)
	bodies := &EndpointBodies{
		prepared:       endpoint,
		event:          event,
		unaryMedia:     analyzeUnaryBodies(a, endpoint),
		RequestBody:    req,
		ResponseBodies: responseBodies,
	}
	bodies.requestMedia = analyzeDocumentMedia(a, endpoint.Request.DocumentBody, endpoint.Request.DocumentValues, schemaUsageRequest, "request-schema")
	if len(endpoint.Request.MediaVariants) > 0 {
		bodies.requestMedia = analyzeMediaVariants(a, endpoint.Request.MediaVariants, schemaUsageRequest)
	}
	if len(endpoint.Request.DocumentContentTypes) > 0 && bodies.requestMedia != nil {
		bodies.RequestBody = bodies.requestMedia[endpoint.Request.DocumentContentTypes[0]]
	}
	bodies.responseMedia = make(map[*transportir.ResponseStatus]map[string]*Schema)
	for _, responses := range [][]*transportir.ResponseStatus{endpoint.Response.Responses, endpoint.Response.ErrorResponses} {
		for _, response := range responses {
			bodies.responseMedia[response] = analyzeDocumentMedia(a, response.DocumentBody, response.DocumentValues, schemaUsageResponse, "response-schema", response.Selector.OpenAPIKey())
			if len(response.MediaVariants) > 0 {
				bodies.responseMedia[response] = analyzeMediaVariants(a, response.MediaVariants, schemaUsageResponse)
			}
		}
	}
	indexes := make(map[string]int)
	for _, responses := range [][]*transportir.ResponseStatus{endpoint.Response.Responses, endpoint.Response.ErrorResponses} {
		for _, response := range responses {
			index := indexes[response.Selector.OpenAPIKey()]
			indexes[response.Selector.OpenAPIKey()]++
			if len(response.ContentTypes) > 0 && bodies.responseMedia[response] != nil {
				bodies.ResponseBodies[response.Selector.OpenAPIKey()][index] = bodies.responseMedia[response][response.ContentTypes[0]]
			}
		}
	}
	return bodies
}

// analyzeUnaryBodies analyzes the whole unary Result representation of each
// typed mixed success from its captured ResultValue target. The event and the
// shared status metadata keep their own owners.
func analyzeUnaryBodies(a *Analyzer, endpoint *transportir.Endpoint) map[*transportir.ResponseStatus]map[string]*Schema {
	result := make(map[*transportir.ResponseStatus]map[string]*Schema)
	for _, response := range endpoint.Response.Responses {
		unary := response.Unary
		if unary == nil {
			continue
		}
		if len(unary.MediaVariants) > 0 {
			result[response] = analyzeMediaVariants(a, unary.MediaVariants, schemaUsageResponse)
			continue
		}
		body := attributeForSchemaUsage(unary.Body, schemaUsageResponse)
		context := attributeExampleContext(body, a.closeObjects, "response-unary-schema", response.Selector.OpenAPIKey())
		schema := a.analyzePreparedOccurrence(body, context, unary.BodyValue)
		media := make(map[string]*Schema, len(unary.ContentTypes))
		for _, contentType := range unary.ContentTypes {
			media[contentType] = schema
		}
		result[response] = media
	}
	return result
}

func analyzeRequestBody(a *Analyzer, endpoint *transportir.Endpoint) *Schema {
	if len(endpoint.Request.MediaVariants) > 0 {
		return nil
	}
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
		req = a.analyzeFormOccurrence(requestAttr, requestContext, target, endpoint.Request.Form)
	}
	if endpoint.Request.StreamingBody == nil {
		return req
	}
	streamingAttr := attributeForSchemaUsage(endpoint.Request.StreamingBody, schemaUsageRequest)
	streamingContext := attributeExampleContext(streamingAttr, a.closeObjects, "streaming-request-schema")
	streaming := a.analyzePreparedOccurrence(streamingAttr, streamingContext, endpoint.Request.StreamingValue)
	return mergeStreamingBodyNote(req, streaming)
}

// analyzeEventOutcome analyzes the effective server SSE event once, in an
// endpoint/event context independent of status and visit order. It runs for
// every typed handshake, eligible or not, and for eligible result-derived
// streams. Handshake metadata never enters this analysis.
func analyzeEventOutcome(a *Analyzer, endpoint *transportir.Endpoint) *sseEventOutcome {
	handshake := hasTypedHandshake(endpoint)
	if !handshake && !derivedAsyncCandidate(endpoint) {
		return nil
	}
	message := effectiveStreamEvent(endpoint.Stream.ResponseMessage)
	target := representation.PrepareStreamSchema(endpoint, message, false)
	eligible := derivedAsyncEligible(endpoint, target)
	if !eligible && !handshake {
		return nil
	}
	context := endpointSchemaExampleContext(endpoint, "sse-event")
	event := &sseEventOutcome{
		attribute: attributeForSchemaUsage(message, schemaUsageResponse),
		target:    target,
		eligible:  eligible,
	}
	if !a.customExampleValue {
		if err := representation.PrepareTargetValues(event.target, message, exampleGeneratorForAttribute(a.rand, message, a.closeObjects, context), true); err != nil {
			panic(err)
		}
	}
	if len(endpoint.Stream.SSE.Projections) > 0 {
		event.schema = analyzeSSEProjectionSchema(a, endpoint)
		return event
	}
	event.schema = a.analyzePreparedOccurrence(event.attribute, context, event.target)
	return event
}

// effectiveStreamEvent applies an explicit result view with the expression
// projection owner, as SSE projections do, so the schema and the prepared value
// target describe the same projected event.
func effectiveStreamEvent(message *expr.AttributeExpr) *expr.AttributeExpr {
	result, ok := message.Type.(*expr.ResultTypeExpr)
	view, selected := message.Meta.Last(expr.ViewMetaKey)
	if !ok || !selected || view == "" {
		return message
	}
	projected, err := expr.Project(result, view)
	if err != nil {
		panic(fmt.Sprintf("project SSE event view %q: %v", view, err))
	}
	event := expr.DupAtt(message)
	event.Type = projected
	event.Validation = projected.Validation
	delete(event.Meta, expr.ViewMetaKey)
	return event
}

func analyzeResponseBodies(a *Analyzer, endpoint *transportir.Endpoint, event *sseEventOutcome) map[string][]*Schema {
	responseBodies := make(map[string][]*Schema)
	appendBodies := func(responses []*transportir.ResponseStatus, projectSSE bool) {
		for _, resp := range responses {
			if eventOwnedResponse(event, resp) {
				responseBodies[resp.Selector.OpenAPIKey()] = append(responseBodies[resp.Selector.OpenAPIKey()], event.schema)
				continue
			}
			if len(resp.DocumentValues) > 0 || len(resp.MediaVariants) > 0 {
				responseBodies[resp.Selector.OpenAPIKey()] = append(responseBodies[resp.Selector.OpenAPIKey()], nil)
				continue
			}
			if projectSSE && endpoint.Stream != nil && endpoint.Stream.SSE != nil && len(endpoint.Stream.SSE.Projections) > 0 {
				responseBodies[resp.Selector.OpenAPIKey()] = append(responseBodies[resp.Selector.OpenAPIKey()], analyzeSSEProjectionSchema(a, endpoint))
				continue
			}
			body := attributeForSchemaUsage(resp.DocumentBody, schemaUsageResponse)
			context := attributeExampleContext(
				body,
				a.closeObjects,
				"response-schema",
				resp.Selector.OpenAPIKey(),
			)
			responseBodies[resp.Selector.OpenAPIKey()] = append(
				responseBodies[resp.Selector.OpenAPIKey()],
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
			if err := representation.PrepareTargetValues(target, attr, exampleGeneratorForAttribute(a.rand, attr, a.closeObjects, context), true); err != nil {
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
	for _, responses := range [][]*transportir.ResponseStatus{endpoint.Response.Responses, endpoint.Response.ErrorResponses, informationalResponses(endpoint.Response)} {
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

func informationalResponses(response *transportir.Response) []*transportir.ResponseStatus {
	responses := make([]*transportir.ResponseStatus, 0, len(response.Informational))
	for _, info := range response.Informational {
		responses = append(responses, info.Response)
	}
	return responses
}
