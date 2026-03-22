package ir

import (
	"sort"
	"strings"

	"goa.design/goa/v3/expr"
)

const asyncContractExtensionName = "x-goa-async"

func buildResponseLinks(resp *expr.HTTPResponseExpr) map[string]*ResponseLinkRef {
	if resp == nil || len(resp.Links) == 0 {
		return nil
	}
	var currentService string
	if endpoint, ok := resp.Parent.(*expr.HTTPEndpointExpr); ok && endpoint.Service != nil {
		currentService = endpoint.Service.Name()
	}
	links := make(map[string]*ResponseLinkRef, len(resp.Links))
	for _, link := range resp.Links {
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
		links[link.Name] = &ResponseLinkRef{Value: value}
	}
	if len(links) == 0 {
		return nil
	}
	return links
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

func buildAsyncOperationExtension(endpoint *expr.HTTPEndpointExpr, path string, rand *expr.ExampleGenerator, closeObjects bool) map[string]any {
	if endpoint == nil || endpoint.MethodExpr == nil || !endpoint.MethodExpr.IsStreaming() {
		return nil
	}
	contract := map[string]any{
		"transport": asyncTransportName(endpoint),
		"handshake": map[string]any{
			"path": path,
			"request": map[string]any{
				"method": endpoint.Routes[0].Method,
			},
			"response": map[string]any{
				"status":      asyncHandshakeStatus(endpoint),
				"contentType": asyncHandshakeContentType(endpoint),
			},
		},
		"direction": asyncDirection(endpoint),
	}

	messages := make(map[string]any)
	if endpoint.MethodExpr.StreamingPayload != nil && endpoint.StreamingBody != nil {
		payloadAttr, _ := attributeForSchemaUsage(endpoint.StreamingBody, schemaUsageRequest)
		messages["inbound"] = map[string]any{
			"contentType": "application/json",
			"schema":      asyncSchemaValue(buildInlineAsyncSchema(payloadAttr, rand, closeObjects)),
		}
	}
	if endpoint.MethodExpr.StreamingResult != nil {
		resultAttr, _ := attributeForSchemaUsage(endpoint.MethodExpr.StreamingResult, schemaUsageResponse)
		outbound := map[string]any{
			"contentType": "application/json",
			"schema":      asyncSchemaValue(buildInlineAsyncSchema(resultAttr, rand, closeObjects)),
		}
		if endpoint.SSE != nil {
			outbound["sse"] = map[string]any{
				"requestIDField": emptyStringAsNil(endpoint.SSE.RequestIDField),
				"dataField":      emptyStringAsNil(endpoint.SSE.DataField),
				"idField":        emptyStringAsNil(endpoint.SSE.IDField),
				"eventField":     emptyStringAsNil(endpoint.SSE.EventField),
				"retryField":     emptyStringAsNil(endpoint.SSE.RetryField),
			}
		}
		messages["outbound"] = outbound
	}
	if len(messages) > 0 {
		contract["messages"] = messages
	}
	return map[string]any{asyncContractExtensionName: contract}
}

func asyncTransportName(endpoint *expr.HTTPEndpointExpr) string {
	if endpoint != nil && endpoint.SSE != nil {
		return "sse"
	}
	return "websocket"
}

func asyncDirection(endpoint *expr.HTTPEndpointExpr) string {
	if endpoint == nil || endpoint.MethodExpr == nil {
		return ""
	}
	switch endpoint.MethodExpr.Stream {
	case expr.ServerStreamKind:
		return "server"
	case expr.ClientStreamKind:
		return "client"
	default:
		return "bidirectional"
	}
}

func asyncHandshakeStatus(endpoint *expr.HTTPEndpointExpr) int {
	if endpoint != nil && endpoint.SSE != nil {
		for _, resp := range endpoint.Responses {
			if resp != nil {
				return resp.StatusCode
			}
		}
		return expr.StatusOK
	}
	return expr.StatusSwitchingProtocols
}

func asyncHandshakeContentType(endpoint *expr.HTTPEndpointExpr) string {
	if endpoint != nil && endpoint.SSE != nil {
		return "text/event-stream"
	}
	return ""
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
