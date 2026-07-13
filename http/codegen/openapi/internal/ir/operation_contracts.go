package ir

import (
	"fmt"
	"sort"
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

	messages := make(map[string]any)
	if endpointIR.Stream.RequestMessage != nil && endpointIR.Stream.RequestHasBody {
		payloadAttr := attributeForSchemaUsage(endpointIR.Stream.RequestMessage, schemaUsageRequest)
		messages["inbound"] = map[string]any{
			"contentType": "application/json",
			"schema":      asyncSchemaValue(buildInlineAsyncSchema(payloadAttr, rand, closeObjects)),
		}
	}
	if endpointIR.Stream.ResponseMessage != nil {
		resultAttr := attributeForSchemaUsage(endpointIR.Stream.ResponseMessage, schemaUsageResponse)
		resultSchema := buildInlineAsyncSchema(resultAttr, rand, closeObjects)
		if endpointIR.Stream.SSE != nil && len(endpointIR.Stream.SSE.Projections) > 0 {
			resultSchema = buildInlineSSEProjectionSchema(endpointIR, rand, closeObjects)
		}
		outbound := map[string]any{
			"contentType": "application/json",
			"schema":      asyncSchemaValue(resultSchema),
		}
		if endpointIR.Stream.SSE != nil {
			sseContract := map[string]any{
				"requestIDField": emptyStringAsNil(endpointIR.Stream.SSE.RequestIDField),
				"dataField":      emptyStringAsNil(endpointIR.Stream.SSE.DataField),
				"idField":        emptyStringAsNil(endpointIR.Stream.SSE.IDField),
				"eventField":     emptyStringAsNil(endpointIR.Stream.SSE.EventField),
				"retryField":     emptyStringAsNil(endpointIR.Stream.SSE.RetryField),
			}
			if len(endpointIR.Stream.SSE.Projections) > 0 {
				projections := make([]map[string]string, 0, len(endpointIR.Stream.SSE.Projections))
				for _, projection := range endpointIR.Stream.SSE.Projections {
					projections = append(projections, map[string]string{
						"event": projection.EventType,
						"view":  projection.View,
					})
				}
				sseContract["projections"] = projections
			}
			outbound["sse"] = sseContract
		}
		messages["outbound"] = outbound
	}
	if len(messages) > 0 {
		contract["messages"] = messages
	}
	return map[string]any{asyncContractExtensionName: contract}
}

func buildInlineSSEProjectionSchema(endpoint *transportir.Endpoint, rand *expr.ExampleGenerator, closeObjects bool) *Schema {
	attrs, err := sseProjectionAttributes(endpoint)
	if err != nil {
		panic(err)
	}
	schema := &Schema{OneOf: make([]*Schema, 0, len(attrs))}
	for _, attr := range attrs {
		schema.OneOf = append(schema.OneOf, buildInlineAsyncSchema(attr, rand, closeObjects))
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

func buildInlineAsyncSchema(attr *expr.AttributeExpr, rand *expr.ExampleGenerator, closeObjects bool) *Schema {
	if attr == nil {
		return nil
	}
	analyzer := NewAnalyzer(rand, closeObjects)
	return analyzer.AnalyzeSchema(inlineContractAttribute(attr), false)
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
