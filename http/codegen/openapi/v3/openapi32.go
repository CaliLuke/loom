package openapiv3

import (
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

const openAPIAsyncExtension = "x-loom-async"

func renderOpenAPI(root *expr.RootExpr, spec *OpenAPI, target openAPIVersion) []string {
	if spec == nil {
		return nil
	}
	spec.OpenAPI = renderOpenAPIVersion(target)
	if target == openAPIVersion32 {
		applyOpenAPI32(root, spec)
		return nil
	}
	return filterOpenAPI31(spec)
}

func applyOpenAPI32(root *expr.RootExpr, spec *OpenAPI) {
	if self, ok := root.API.Meta.Last("openapi:self"); ok {
		spec.Self = self
	}
	promoteConfiguredItemSchemas(spec)
	for _, path := range spec.Paths {
		promoteQueryStringParameters(path.Parameters)
		for _, operation := range pathItemOperations(path) {
			if operation == nil {
				continue
			}
			promoteSSEMediaTypes(operation.Responses, sseFieldMappings(operation.Extensions), spec.Components)
			promoteQueryStringParameters(operation.Parameters)
		}
	}
	if spec.Components != nil {
		for _, parameter := range spec.Components.Parameters {
			promoteQueryStringParameters([]*ParameterRef{parameter})
		}
		for _, response := range spec.Components.Responses {
			if response != nil && response.Value != nil {
				promoteSSEMediaType(response.Value.Content["text/event-stream"], nil, spec.Components)
			}
		}
	}
	applySecuritySchemeURIs(root, spec)
	componentizeMediaTypes(spec)
}

func promoteConfiguredItemSchemas(spec *OpenAPI) {
	visit := func(mediaTypes map[string]*MediaType) {
		for _, mediaType := range mediaTypes {
			if mediaType != nil && mediaType.UseItemSchema && mediaType.ItemSchema == nil {
				mediaType.ItemSchema = mediaType.Schema
				mediaType.Schema = nil
			}
		}
	}
	for _, path := range spec.Paths {
		for _, operation := range pathItemOperations(path) {
			if operation == nil {
				continue
			}
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				visit(operation.RequestBody.Value.Content)
			}
			for _, response := range operation.Responses {
				if response != nil && response.Value != nil {
					visit(response.Value.Content)
				}
			}
		}
	}
	if spec.Components == nil {
		return
	}
	for _, body := range spec.Components.RequestBodies {
		if body != nil && body.Value != nil {
			visit(body.Value.Content)
		}
	}
	for _, response := range spec.Components.Responses {
		if response != nil && response.Value != nil {
			visit(response.Value.Content)
		}
	}
}

func componentizeMediaTypes(spec *OpenAPI) {
	if spec == nil || spec.Components == nil {
		return
	}
	visit := func(mediaTypes map[string]*MediaType) {
		for _, mediaType := range mediaTypes {
			if mediaType == nil || mediaType.ComponentName == "" {
				continue
			}
			name := mediaType.ComponentName
			component := *mediaType
			component.ComponentName = ""
			if spec.Components.MediaTypes == nil {
				spec.Components.MediaTypes = make(map[string]*MediaTypeRef)
			}
			spec.Components.MediaTypes[name] = &MediaTypeRef{Value: &component}
			*mediaType = MediaType{Ref: "#/components/mediaTypes/" + name}
		}
	}
	visitOperation := func(operation *Operation) {
		if operation == nil {
			return
		}
		if operation.RequestBody != nil && operation.RequestBody.Value != nil {
			visit(operation.RequestBody.Value.Content)
		}
		for _, response := range operation.Responses {
			if response != nil && response.Value != nil {
				visit(response.Value.Content)
			}
		}
	}
	for _, path := range spec.Paths {
		for _, operation := range pathItemOperations(path) {
			visitOperation(operation)
		}
	}
	for _, body := range spec.Components.RequestBodies {
		if body != nil && body.Value != nil {
			visit(body.Value.Content)
		}
	}
	for _, response := range spec.Components.Responses {
		if response != nil && response.Value != nil {
			visit(response.Value.Content)
		}
	}
}

func promoteQueryStringParameters(parameters []*ParameterRef) {
	for _, ref := range parameters {
		if ref == nil || ref.Value == nil || !ref.Value.WholeQueryString {
			continue
		}
		parameter := ref.Value
		parameter.In = "querystring"
		parameter.Style = ""
		parameter.Content = map[string]*MediaType{
			"application/x-www-form-urlencoded": {Schema: parameter.Schema},
		}
		parameter.Schema = nil
	}
}

func applySecuritySchemeURIs(root *expr.RootExpr, spec *OpenAPI) {
	if root == nil || spec == nil || spec.Components == nil {
		return
	}
	replacements := make(map[string]string)
	for _, scheme := range root.Schemes {
		if scheme == nil {
			continue
		}
		if uri, ok := scheme.Meta.Last("openapi:security:uri"); ok && uri != "" {
			replacements[scheme.Hash()] = uri
			delete(spec.Components.SecuritySchemes, scheme.Hash())
		}
	}
	if len(replacements) == 0 {
		return
	}
	rewriteSecurityRequirementURIs(spec.Security, replacements)
	for _, path := range spec.Paths {
		for _, operation := range pathItemOperations(path) {
			if operation != nil {
				rewriteSecurityRequirementURIs(operation.Security, replacements)
			}
		}
	}
}

func rewriteSecurityRequirementURIs(requirements []map[string][]string, replacements map[string]string) {
	for _, requirement := range requirements {
		for name, scopes := range requirement {
			uri, ok := replacements[name]
			if !ok {
				continue
			}
			delete(requirement, name)
			requirement[uri] = scopes
		}
	}
}

func pathItemOperations(path *PathItem) []*Operation {
	if path == nil {
		return nil
	}
	operations := []*Operation{
		path.Connect,
		path.Delete,
		path.Get,
		path.Head,
		path.Options,
		path.Patch,
		path.Post,
		path.Put,
		path.Trace,
		path.Query,
	}
	for _, operation := range path.AdditionalOperations {
		operations = append(operations, operation)
	}
	return operations
}

func promoteSSEMediaTypes(responses map[string]*ResponseRef, fields map[string]string, components *Components) {
	for _, response := range responses {
		if response == nil || response.Value == nil {
			continue
		}
		promoteSSEMediaType(response.Value.Content["text/event-stream"], fields, components)
	}
}

func promoteSSEMediaType(mediaType *MediaType, fields map[string]string, components *Components) {
	if mediaType == nil || mediaType.Schema == nil || mediaType.ItemSchema != nil {
		return
	}
	payloadSchema := mediaType.Schema
	if dataField := fields["dataField"]; dataField != "" {
		if fieldSchema := schemaProperty(payloadSchema, dataField, components); fieldSchema != nil {
			payloadSchema = fieldSchema
		}
	}
	properties := map[string]*openapi.Schema{
		"data": {
			Type:             openapi.String,
			ContentMediaType: "application/json",
			ContentSchema:    payloadSchema,
		},
	}
	if fields["eventField"] != "" {
		properties["event"] = &openapi.Schema{Type: openapi.String}
	}
	if fields["idField"] != "" {
		properties["id"] = &openapi.Schema{Type: openapi.String}
	}
	if fields["retryField"] != "" {
		zero := 0.0
		properties["retry"] = &openapi.Schema{Type: openapi.Integer, Minimum: &zero}
	}
	mediaType.Schema = nil
	mediaType.ItemSchema = &openapi.Schema{
		Type:                 openapi.Object,
		Properties:           properties,
		Required:             []string{"data"},
		AdditionalProperties: false,
	}
}

func schemaProperty(schema *openapi.Schema, name string, components *Components) *openapi.Schema {
	if schema == nil {
		return nil
	}
	resolved := schema
	if strings.HasPrefix(schema.Ref, "#/components/schemas/") && components != nil {
		resolved = components.Schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")]
	}
	if resolved == nil {
		return nil
	}
	return resolved.Properties[name]
}

func sseFieldMappings(extensions map[string]any) map[string]string {
	async, ok := extensions[openAPIAsyncExtension].(map[string]any)
	if !ok {
		return nil
	}
	messages, ok := async["messages"].(map[string]any)
	if !ok {
		return nil
	}
	outbound, ok := messages["outbound"].(map[string]any)
	if !ok {
		return nil
	}
	sse, ok := outbound["sse"].(map[string]any)
	if !ok {
		return nil
	}
	fields := make(map[string]string)
	for _, name := range []string{"dataField", "eventField", "idField", "retryField"} {
		if value, ok := sse[name].(string); ok {
			fields[name] = value
		}
	}
	return fields
}
