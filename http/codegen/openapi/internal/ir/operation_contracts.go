package ir

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

const asyncContractExtensionName = "x-loom-async"

func buildResponseLinks(links []*transportir.ResponseLink, currentService string) map[string]*ResponseLinkRef {
	if len(links) == 0 {
		return nil
	}
	result := make(map[string]*ResponseLinkRef, len(links))
	for _, link := range links {
		if link == nil {
			continue
		}
		value := &ResponseLink{
			OperationID:  resolveLinkedOperationID(link.Operation, currentService),
			OperationRef: link.OperationRef,
			Description:  link.Description,
			RequestBody:  emptyStringAsNil(link.RequestBody),
			Extensions:   nil,
		}
		if len(link.Parameters) > 0 {
			value.Parameters = make(map[string]any, len(link.Parameters))
			for _, name := range orderedAsyncLinkParameterNames(link.Parameters) {
				value.Parameters[name] = link.Parameters[name]
			}
		}
		if value.OperationID == "" && value.OperationRef == "" {
			continue
		}
		result[link.Name] = &ResponseLinkRef{Value: value}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func orderedAsyncLinkParameterNames(parameters map[string]string) []string {
	if len(parameters) == 0 {
		return nil
	}
	names := make([]string, 0, len(parameters))
	for name := range parameters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func resolveLinkedOperationID(target string, currentService string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}

	serviceName := currentService
	methodName := target
	if dot := strings.Index(target, "."); dot >= 0 {
		serviceName = strings.TrimSpace(target[:dot])
		methodName = strings.TrimSpace(target[dot+1:])
	}

	api := expr.Root.API
	if api == nil || api.HTTP == nil {
		return target
	}
	for _, svc := range api.HTTP.Services {
		if svc == nil || svc.Name() != serviceName {
			continue
		}
		endpoint := svc.Endpoint(methodName)
		if endpoint == nil || endpoint.MethodExpr == nil {
			break
		}
		operationIDFormat := defaultOperationIDFormat
		for _, meta := range []expr.MetaExpr{api.Meta, svc.ServiceExpr.Meta, endpoint.Meta, endpoint.MethodExpr.Meta} {
			if value, ok := meta.Last("openapi:operationId"); ok {
				operationIDFormat = value
			}
		}
		return ParseOperationIDTemplate(operationIDFormat, svc.Name(), endpoint.Name(), 0)
	}
	return target
}

func buildAsyncOperationExtension(endpointIR *transportir.Endpoint, path string, bodies *EndpointBodies) map[string]any {
	if endpointIR == nil || endpointIR.Stream == nil || !endpointIR.Stream.IsStreaming {
		return nil
	}
	contract := map[string]any{
		"transport": endpointIR.Stream.Transport,
		"handshake": map[string]any{
			"path": path,
			"request": map[string]any{
				"method": endpointIR.Stream.HandshakeMethod,
			},
			"response": map[string]any{
				"status":      endpointIR.Stream.HandshakeStatus,
				"contentType": endpointIR.Stream.HandshakeContent,
			},
		},
		"direction": endpointIR.Stream.Direction,
	}

	if messages := buildAsyncMessages(endpointIR, path, bodies); len(messages) > 0 {
		contract["messages"] = messages
	}
	return map[string]any{asyncContractExtensionName: contract}
}

func buildAsyncMessages(endpoint *transportir.Endpoint, path string, bodies *EndpointBodies) map[string]any {
	messages := make(map[string]any)
	if bodies == nil {
		return messages
	}
	for direction, prepared := range bodies.async[path] {
		materialized := materializeAsyncSchema(prepared, bodies.asyncComponents)
		schema := materialized.schema
		applyPreparedAsyncExamples(bodies.asyncAnalyzer, schema, prepared, materialized.structures)
		message := map[string]any{
			"contentType": "application/json",
			"schema":      asyncSchemaValue(schema),
		}
		if direction == "outbound" && endpoint.Stream.SSE != nil {
			message["sse"] = buildAsyncSSEContract(endpoint.Stream.SSE)
		}
		messages[direction] = message
	}
	return messages
}

func analyzeAsyncSchemas(a *Analyzer, endpoint *transportir.Endpoint) map[string]map[string]*asyncSchema {
	if endpoint.Stream == nil || !endpoint.Stream.IsStreaming {
		return nil
	}
	result := make(map[string]map[string]*asyncSchema)
	for _, route := range endpoint.Routes {
		path := expr.HTTPWildcardRegex.ReplaceAllString(route.Path, "/{$1}")
		context := endpointSchemaExampleContext(endpoint, "async", path)
		messages := make(map[string]*asyncSchema)
		if endpoint.Stream.RequestMessage != nil && endpoint.Stream.RequestHasBody {
			attr := attributeForSchemaUsage(endpoint.Stream.RequestMessage, schemaUsageRequest)
			messages["inbound"] = analyzeAsyncSchema(a, attr, endpoint, true, childExampleContext(context, "inbound"))
		}
		if endpoint.Stream.ResponseMessage != nil {
			if endpoint.Stream.SSE != nil && len(endpoint.Stream.SSE.Projections) > 0 {
				messages["outbound"] = analyzeAsyncSSESchema(a, endpoint, context)
			} else {
				attr := attributeForSchemaUsage(endpoint.Stream.ResponseMessage, schemaUsageResponse)
				messages["outbound"] = analyzeAsyncSchema(a, attr, endpoint, false, childExampleContext(context, "outbound"))
			}
		}
		result[path] = messages
	}
	return result
}

func analyzeAsyncSchema(a *Analyzer, attr *expr.AttributeExpr, endpoint *transportir.Endpoint, inbound bool, context string) *asyncSchema {
	target := representation.PrepareStreamSchema(endpoint, attr, inbound)
	return a.acquireAsyncBaseline(attr, representationRoot(attr, target), asyncSamplerAttribute(attr), context)
}

func buildAsyncSSEContract(sse *transportir.SSE) map[string]any {
	contract := map[string]any{
		"requestIDField": emptyStringAsNil(sse.RequestIDField),
		"dataField":      emptyStringAsNil(sse.DataField),
		"idField":        emptyStringAsNil(sse.IDField),
		"eventField":     emptyStringAsNil(sse.EventField),
		"retryField":     emptyStringAsNil(sse.RetryField),
	}
	if len(sse.Projections) > 0 {
		projections := make([]map[string]string, 0, len(sse.Projections))
		for _, projection := range sse.Projections {
			projections = append(projections, map[string]string{
				"event": projection.EventType,
				"view":  projection.View,
			})
		}
		contract["projections"] = projections
	}
	return contract
}

func analyzeAsyncSSESchema(a *Analyzer, endpoint *transportir.Endpoint, context string) *asyncSchema {
	attrs, err := sseProjectionAttributes(endpoint)
	if err != nil {
		panic(err)
	}
	result := &asyncSchema{schema: &Schema{OneOf: make([]*Schema, 0, len(attrs))}, constructions: a.constructions}
	for index, attr := range attrs {
		projectionContext := childExampleContext(context, "projection", strconv.Itoa(index))
		child := analyzeAsyncSchema(a, attr, endpoint, false, projectionContext)
		result.schema.OneOf = append(result.schema.OneOf, child.schema)
		result.projections = append(result.projections, child)
	}
	return result
}

func sseProjectionAttributes(endpoint *transportir.Endpoint) ([]*expr.AttributeExpr, error) {
	if endpoint == nil || endpoint.Stream == nil || endpoint.Stream.SSE == nil || endpoint.Stream.ResponseMessage == nil {
		return nil, fmt.Errorf("SSE projection endpoint is incomplete")
	}
	resultType, ok := endpoint.Stream.ResponseMessage.Type.(*expr.ResultTypeExpr)
	if !ok {
		return nil, fmt.Errorf("SSE projections require a result type")
	}
	attrs := make([]*expr.AttributeExpr, 0, len(endpoint.Stream.SSE.Projections))
	for _, projection := range endpoint.Stream.SSE.Projections {
		projected, err := expr.Project(resultType, projection.View)
		if err != nil {
			return nil, fmt.Errorf("project SSE view %q: %w", projection.View, err)
		}
		attr := expr.DupAtt(endpoint.Stream.ResponseMessage)
		attr.Type = projected
		attr.Validation = projected.Validation
		attrs = append(attrs, attr)
	}
	return attrs, nil
}

func asyncSchemaValue(schema *Schema) any {
	if schema == nil {
		return nil
	}
	// Keep schema roots typed through component cleanup, including pure refs.
	// Their JSON bytes are unchanged; reference lifecycle visitors can now see
	// every framework-owned edge without interpreting arbitrary extension data.
	return RenderSchema(schema)
}

func mergeExtensions(dst map[string]any, src map[string]any) map[string]any {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]any, len(src))
	}
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func emptyStringAsNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
