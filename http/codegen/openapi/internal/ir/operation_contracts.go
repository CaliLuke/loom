package ir

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
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
		return parseOperationIDTemplate(operationIDFormat, svc.Name(), endpoint.Name(), 0)
	}
	return target
}

func buildAsyncOperationExtension(endpointIR *transportir.Endpoint, path string, rand *expr.ExampleGenerator, closeObjects bool) map[string]any {
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

	asyncContext := endpointSchemaExampleContext(endpointIR, "async", path)
	if messages := buildAsyncMessages(endpointIR, rand, closeObjects, asyncContext); len(messages) > 0 {
		contract["messages"] = messages
	}
	return map[string]any{asyncContractExtensionName: contract}
}

func buildAsyncMessages(
	endpointIR *transportir.Endpoint,
	rand *expr.ExampleGenerator,
	closeObjects bool,
	context string,
) map[string]any {
	messages := make(map[string]any)
	if endpointIR.Stream.RequestMessage != nil && endpointIR.Stream.RequestHasBody {
		payloadAttr := attributeForSchemaUsage(endpointIR.Stream.RequestMessage, schemaUsageRequest)
		messages["inbound"] = map[string]any{
			"contentType": "application/json",
			"schema": asyncSchemaValue(buildInlineAsyncSchema(
				payloadAttr,
				rand,
				closeObjects,
				childExampleContext(context, "inbound"),
			)),
		}
	}
	if endpointIR.Stream.ResponseMessage != nil {
		messages["outbound"] = buildAsyncOutboundMessage(endpointIR, rand, closeObjects, context)
	}
	return messages
}

func buildAsyncOutboundMessage(
	endpointIR *transportir.Endpoint,
	rand *expr.ExampleGenerator,
	closeObjects bool,
	context string,
) map[string]any {
	resultAttr := attributeForSchemaUsage(endpointIR.Stream.ResponseMessage, schemaUsageResponse)
	resultSchema := buildInlineAsyncSchema(
		resultAttr,
		rand,
		closeObjects,
		childExampleContext(context, "outbound"),
	)
	if endpointIR.Stream.SSE != nil && len(endpointIR.Stream.SSE.Projections) > 0 {
		resultSchema = buildInlineSSEProjectionSchema(endpointIR, rand, closeObjects, context)
	}
	outbound := map[string]any{
		"contentType": "application/json",
		"schema":      asyncSchemaValue(resultSchema),
	}
	if endpointIR.Stream.SSE != nil {
		outbound["sse"] = buildAsyncSSEContract(endpointIR.Stream.SSE)
	}
	return outbound
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

func buildInlineSSEProjectionSchema(
	endpoint *transportir.Endpoint,
	rand *expr.ExampleGenerator,
	closeObjects bool,
	context ...string,
) *Schema {
	attrs, err := sseProjectionAttributes(endpoint)
	if err != nil {
		panic(err)
	}
	schema := &Schema{OneOf: make([]*Schema, 0, len(attrs))}
	for index, attr := range attrs {
		projectionContext := exampleContext("sse-projection", strconv.Itoa(index))
		if len(context) > 0 && context[0] != "" {
			projectionContext = childExampleContext(context[0], "projection", strconv.Itoa(index))
		}
		schema.OneOf = append(
			schema.OneOf,
			buildInlineAsyncSchema(attr, rand, closeObjects, projectionContext),
		)
	}
	return schema
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
		attrs = append(attrs, &expr.AttributeExpr{
			Type:       projected,
			Validation: projected.Validation,
		})
	}
	return attrs, nil
}

func asyncSchemaValue(schema *Schema) any {
	if schema == nil {
		return nil
	}
	if schema.Ref != "" {
		return map[string]any{"$ref": schema.Ref}
	}
	return RenderSchema(schema)
}

func buildInlineAsyncSchema(
	attr *expr.AttributeExpr,
	rand *expr.ExampleGenerator,
	closeObjects bool,
	context ...string,
) *Schema {
	if attr == nil {
		return nil
	}
	analyzer := NewAnalyzer(rand, closeObjects)
	inlineContext := exampleContext("async-schema", fingerprintAttribute(attr, closeObjects))
	if len(context) > 0 && context[0] != "" {
		inlineContext = context[0]
	}
	return analyzer.AnalyzeSchemaWithContext(inlineContractAttribute(attr), inlineContext, false)
}

func inlineContractAttribute(attr *expr.AttributeExpr) *expr.AttributeExpr {
	if attr == nil {
		return nil
	}
	cloned := expr.DupAtt(attr)
	inlineContractUserTypes(cloned, make(map[string]struct{}))
	return cloned
}

func inlineContractUserTypes(attr *expr.AttributeExpr, seen map[string]struct{}) {
	if attr == nil || attr.Type == nil || attr.Type == expr.Empty {
		return
	}
	switch actual := attr.Type.(type) {
	case expr.UserType:
		key := actual.Hash()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		inlined := componentAttribute(attr, actual)
		*attr = *inlined
		inlineContractUserTypes(attr, seen)
		delete(seen, key)
	case *expr.Array:
		inlineContractUserTypes(actual.ElemType, seen)
	case *expr.Map:
		inlineContractUserTypes(actual.ElemType, seen)
	case *expr.Object:
		for _, named := range *actual {
			inlineContractUserTypes(named.Attribute, seen)
		}
	case *expr.Union:
		for _, named := range actual.Values {
			inlineContractUserTypes(named.Attribute, seen)
		}
	}
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
