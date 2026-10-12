package openapiv3

import (
	json "encoding/json/v2"
	"fmt"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
)

const openAPIAsyncExtension = "x-loom-async"

type pendingMediaTypeComponent struct {
	mediaType *MediaType
	name      string
	value     MediaType
	encoded   string
}

func renderOpenAPI(root *expr.RootExpr, spec *OpenAPI, target openAPIVersion) ([]string, error) {
	if spec == nil {
		return nil, nil
	}
	router := &versionRouter{target: target}
	version, err := renderOpenAPIVersion(router)
	if err != nil {
		return nil, err
	}
	spec.OpenAPI = version
	router.runPasses(
		versionedPass{
			versions: versionRange{through: openAPIVersion31},
			apply: func() ([]string, error) {
				return filterOpenAPI31(spec), nil
			},
		},
		versionedPass{
			versions: versionRange{from: openAPIVersion32},
			apply: func() ([]string, error) {
				return nil, applyOpenAPI32(root, spec)
			},
		},
	)
	if err := router.err(); err != nil {
		return nil, err
	}
	return router.warnings(), nil
}

func applyOpenAPI32(root *expr.RootExpr, spec *OpenAPI) error {
	if self, ok := root.API.Meta.Last("openapi:self"); ok {
		spec.Self = self
	}
	promoteConfiguredItemSchemas(spec)
	promoteOpenAPI32PathValues(spec)
	promoteOpenAPI32ComponentValues(spec)
	applySecuritySchemeURIs(spec)
	return componentizeMediaTypes(spec)
}

func promoteOpenAPI32PathValues(spec *OpenAPI) {
	for _, pathKey := range sortedStringMapKeys(spec.Paths) {
		path := spec.Paths[pathKey]
		promoteQueryStringParameters(path.Parameters)
		promoteOpenAPI32PathOperations(path, spec.Components)
	}
}

func promoteOpenAPI32PathOperations(path *PathItem, components *Components) {
	for _, operation := range pathItemOperations(path) {
		if operation == nil {
			continue
		}
		promoteSSEMediaTypes(operation.Responses, sseFieldMappings(operation.Extensions), components)
		promoteQueryStringParameters(operation.Parameters)
	}
}

func promoteOpenAPI32ComponentValues(spec *OpenAPI) {
	if spec.Components == nil {
		return
	}
	for _, name := range sortedStringMapKeys(spec.Components.Parameters) {
		parameter := spec.Components.Parameters[name]
		promoteQueryStringParameters([]*ParameterRef{parameter})
	}
	for _, name := range sortedStringMapKeys(spec.Components.Responses) {
		response := spec.Components.Responses[name]
		if response != nil && response.Value != nil {
			promoteSSEMediaType(response.Value.Content["text/event-stream"], nil, spec.Components)
		}
	}
}

func promoteConfiguredItemSchemas(spec *OpenAPI) {
	visitMediaTypes(spec, func(mediaTypes map[string]*MediaType) {
		for _, mediaType := range mediaTypes {
			if mediaType != nil && mediaType.UseItemSchema && mediaType.ItemSchema == nil {
				mediaType.ItemSchema = mediaType.Schema
				mediaType.Schema = nil
			}
		}
	})
}

func componentizeMediaTypes(spec *OpenAPI) error {
	if spec == nil || spec.Components == nil {
		return nil
	}
	pending, err := collectPendingMediaTypeComponents(spec)
	if err != nil {
		return err
	}
	claims, err := collectExistingMediaTypeComponentClaims(spec)
	if err != nil {
		return err
	}
	for _, component := range pending {
		claims = append(claims, openapiir.ComponentIdentity{
			Kind: "mediaTypes", Name: component.name, Value: component.encoded,
		})
	}
	if err := openapiir.ValidateComponentIdentities(claims); err != nil {
		return err
	}
	publishMediaTypeComponents(spec, pending)
	return nil
}

func collectPendingMediaTypeComponents(spec *OpenAPI) ([]pendingMediaTypeComponent, error) {
	var pending []pendingMediaTypeComponent
	var marshalErr error
	visitMediaTypes(spec, func(mediaTypes map[string]*MediaType) {
		keys := make([]string, 0, len(mediaTypes))
		for key := range mediaTypes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			mediaType := mediaTypes[key]
			if mediaType == nil || mediaType.ComponentName == "" {
				continue
			}
			component := *mediaType
			component.ComponentName = ""
			encoded, err := json.Marshal(component, json.Deterministic(true))
			if err != nil {
				marshalErr = err
				return
			}
			pending = append(pending, pendingMediaTypeComponent{mediaType: mediaType, name: mediaType.ComponentName, value: component, encoded: string(encoded)})
		}
	})
	if marshalErr != nil {
		return nil, fmt.Errorf("marshal OpenAPI media type component claim: %w", marshalErr)
	}
	return pending, nil
}

func collectExistingMediaTypeComponentClaims(spec *OpenAPI) ([]openapiir.ComponentIdentity, error) {
	claims := make([]openapiir.ComponentIdentity, 0, len(spec.Components.MediaTypes))
	for _, name := range sortedStringMapKeys(spec.Components.MediaTypes) {
		ref := spec.Components.MediaTypes[name]
		if ref == nil || ref.Value == nil {
			continue
		}
		component := *ref.Value
		component.ComponentName = ""
		encoded, err := json.Marshal(component, json.Deterministic(true))
		if err != nil {
			return nil, fmt.Errorf("marshal existing OpenAPI media type component %q: %w", name, err)
		}
		claims = append(claims, openapiir.ComponentIdentity{
			Kind: "mediaTypes", Name: name, Value: string(encoded), Source: "existing component",
		})
	}
	return claims, nil
}

func publishMediaTypeComponents(spec *OpenAPI, pending []pendingMediaTypeComponent) {
	if len(pending) == 0 {
		return
	}
	if spec.Components.MediaTypes == nil {
		spec.Components.MediaTypes = make(map[string]*MediaTypeRef)
	}
	for _, component := range pending {
		if _, exists := spec.Components.MediaTypes[component.name]; !exists {
			value := component.value
			spec.Components.MediaTypes[component.name] = &MediaTypeRef{Value: &value}
		}
		*component.mediaType = MediaType{Ref: "#/components/mediaTypes/" + component.name}
	}
}

func visitMediaTypes(spec *OpenAPI, visit func(map[string]*MediaType)) {
	if spec == nil || visit == nil {
		return
	}
	for _, pathKey := range sortedStringMapKeys(spec.Paths) {
		path := spec.Paths[pathKey]
		if path == nil {
			continue
		}
		visitParameterMediaTypes(path.Parameters, visit)
		for _, operation := range pathItemOperations(path) {
			if operation == nil {
				continue
			}
			visitParameterMediaTypes(operation.Parameters, visit)
			if operation.RequestBody != nil && operation.RequestBody.Value != nil {
				visit(operation.RequestBody.Value.Content)
			}
			for _, status := range sortedStringMapKeys(operation.Responses) {
				response := operation.Responses[status]
				if response != nil && response.Value != nil {
					visit(response.Value.Content)
				}
			}
		}
	}
	if spec.Components == nil {
		return
	}
	for _, name := range sortedStringMapKeys(spec.Components.Parameters) {
		parameter := spec.Components.Parameters[name]
		if parameter != nil && parameter.Value != nil {
			visit(parameter.Value.Content)
		}
	}
	for _, name := range sortedStringMapKeys(spec.Components.RequestBodies) {
		body := spec.Components.RequestBodies[name]
		if body != nil && body.Value != nil {
			visit(body.Value.Content)
		}
	}
	for _, name := range sortedStringMapKeys(spec.Components.Responses) {
		response := spec.Components.Responses[name]
		if response != nil && response.Value != nil {
			visit(response.Value.Content)
		}
	}
}

func visitParameterMediaTypes(parameters []*ParameterRef, visit func(map[string]*MediaType)) {
	for _, parameter := range parameters {
		if parameter != nil && parameter.Value != nil {
			visit(parameter.Value.Content)
		}
	}
}

func sortedStringMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
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

func applySecuritySchemeURIs(spec *OpenAPI) {
	if spec == nil || spec.Components == nil {
		return
	}
	replacements := make(map[string]string)
	for name, ref := range spec.Components.SecuritySchemes {
		if ref == nil || ref.Value == nil {
			continue
		}
		if uri := ref.Value.securityURI; uri != "" {
			replacements[name] = uri
			delete(spec.Components.SecuritySchemes, name)
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
	for _, method := range sortedStringMapKeys(path.AdditionalOperations) {
		operations = append(operations, path.AdditionalOperations[method])
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
		zero := expr.NumericBound{}
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
