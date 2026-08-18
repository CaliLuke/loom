package openapiv3

import (
	"strings"

	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func schemaNameFromRef(ref string) (string, bool) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return "", false
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" {
		return "", false
	}
	return name, true
}

func rewritePathItemSchemaRefs(pathItem *PathItem, resolveRef func(string) string) {
	if pathItem == nil {
		return
	}
	for _, op := range pathItemOperations(pathItem) {
		rewriteOperationSchemaRefs(op, resolveRef)
	}
}

func rewriteOperationSchemaRefs(op *Operation, resolveRef func(string) string) {
	if op == nil {
		return
	}
	for _, param := range op.Parameters {
		if param != nil && param.Value != nil {
			rewriteSchemaRefs(param.Value.Schema, resolveRef)
		}
	}
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		for _, content := range op.RequestBody.Value.Content {
			if content != nil {
				rewriteSchemaRefs(content.Schema, resolveRef)
			}
		}
	}
	for _, response := range op.Responses {
		if response == nil || response.Value == nil {
			continue
		}
		for _, header := range response.Value.Headers {
			if header != nil && header.Value != nil {
				rewriteSchemaRefs(header.Value.Schema, resolveRef)
				rewriteMediaTypeSchemaRefs(header.Value.Content, resolveRef)
			}
		}
		rewriteMediaTypeSchemaRefs(response.Value.Content, resolveRef)
	}
}

func rewriteReusableSchemaRefs(reusable reusableComponents, resolveRef func(string) string) {
	for _, parameter := range reusable.Parameters {
		if parameter != nil && parameter.Value != nil {
			rewriteSchemaRefs(parameter.Value.Schema, resolveRef)
		}
	}
	for _, header := range reusable.Headers {
		if header == nil || header.Value == nil {
			continue
		}
		rewriteSchemaRefs(header.Value.Schema, resolveRef)
		rewriteMediaTypeSchemaRefs(header.Value.Content, resolveRef)
	}
	for _, requestBody := range reusable.RequestBodies {
		if requestBody == nil || requestBody.Value == nil {
			continue
		}
		rewriteMediaTypeSchemaRefs(requestBody.Value.Content, resolveRef)
	}
	for _, response := range reusable.Responses {
		if response == nil || response.Value == nil {
			continue
		}
		for _, header := range response.Value.Headers {
			if header == nil || header.Value == nil {
				continue
			}
			rewriteSchemaRefs(header.Value.Schema, resolveRef)
			rewriteMediaTypeSchemaRefs(header.Value.Content, resolveRef)
		}
		rewriteMediaTypeSchemaRefs(response.Value.Content, resolveRef)
	}
}

func rewriteSchemaRefs(schema *openapi.Schema, resolveRef func(string) string) {
	if schema == nil {
		return
	}
	if schema.Ref != "" {
		schema.Ref = resolveRef(schema.Ref)
		return
	}
	rewriteSchemaRefs(schema.Items, resolveRef)
	rewriteSchemaRefs(schema.ContentSchema, resolveRef)
	for _, prop := range schema.Properties {
		rewriteSchemaRefs(prop, resolveRef)
	}
	for _, def := range schema.Defs {
		rewriteSchemaRefs(def, resolveRef)
	}
	for _, item := range schema.AllOf {
		rewriteSchemaRefs(item, resolveRef)
	}
	for _, item := range schema.AnyOf {
		rewriteSchemaRefs(item, resolveRef)
	}
	for _, item := range schema.OneOf {
		rewriteSchemaRefs(item, resolveRef)
	}
	if nested, ok := schema.AdditionalProperties.(*openapi.Schema); ok {
		rewriteSchemaRefs(nested, resolveRef)
	}
	if nested, ok := schema.UnevaluatedProperties.(*openapi.Schema); ok {
		rewriteSchemaRefs(nested, resolveRef)
	}
	for _, link := range schema.Links {
		if link == nil {
			continue
		}
		rewriteSchemaRefs(link.Schema, resolveRef)
		rewriteSchemaRefs(link.TargetSchema, resolveRef)
	}
}

func collectPathItemSchemaRefs(pathItem *PathItem, addRef func(string)) {
	if pathItem == nil {
		return
	}
	for _, op := range pathItemOperations(pathItem) {
		collectOperationSchemaRefs(op, addRef)
	}
}

func collectOperationSchemaRefs(op *Operation, addRef func(string)) {
	if op == nil {
		return
	}
	for _, param := range op.Parameters {
		if param != nil && param.Value != nil {
			collectSchemaRefs(param.Value.Schema, addRef)
		}
	}
	if op.RequestBody != nil && op.RequestBody.Value != nil {
		collectOperationMediaTypeSchemaRefs(op.RequestBody.Value.Content, addRef)
	}
	for _, response := range op.Responses {
		if response == nil || response.Value == nil {
			continue
		}
		for _, header := range response.Value.Headers {
			if header != nil && header.Value != nil {
				collectSchemaRefs(header.Value.Schema, addRef)
				collectOperationMediaTypeSchemaRefs(header.Value.Content, addRef)
			}
		}
		collectOperationMediaTypeSchemaRefs(response.Value.Content, addRef)
	}
}

func collectSchemaRefs(schema *openapi.Schema, addRef func(string)) {
	if schema == nil {
		return
	}
	if schema.Ref != "" {
		addRef(schema.Ref)
		return
	}
	collectSchemaRefs(schema.Items, addRef)
	collectSchemaRefs(schema.ContentSchema, addRef)
	for _, prop := range schema.Properties {
		collectSchemaRefs(prop, addRef)
	}
	for _, def := range schema.Defs {
		collectSchemaRefs(def, addRef)
	}
	for _, item := range schema.AllOf {
		collectSchemaRefs(item, addRef)
	}
	for _, item := range schema.AnyOf {
		collectSchemaRefs(item, addRef)
	}
	for _, item := range schema.OneOf {
		collectSchemaRefs(item, addRef)
	}
	if nested, ok := schema.AdditionalProperties.(*openapi.Schema); ok {
		collectSchemaRefs(nested, addRef)
	}
	if nested, ok := schema.UnevaluatedProperties.(*openapi.Schema); ok {
		collectSchemaRefs(nested, addRef)
	}
	for _, link := range schema.Links {
		if link == nil {
			continue
		}
		collectSchemaRefs(link.Schema, addRef)
		collectSchemaRefs(link.TargetSchema, addRef)
	}
}

func rewriteMediaTypeSchemaRefs(mediaTypes map[string]*MediaType, resolveRef func(string) string) {
	for _, mediaType := range mediaTypes {
		if mediaType == nil {
			continue
		}
		rewriteSchemaRefs(mediaType.Schema, resolveRef)
		rewriteSchemaRefs(mediaType.ItemSchema, resolveRef)
	}
}

func collectOperationMediaTypeSchemaRefs(mediaTypes map[string]*MediaType, addRef func(string)) {
	for _, mediaType := range mediaTypes {
		if mediaType == nil {
			continue
		}
		collectSchemaRefs(mediaType.Schema, addRef)
		collectSchemaRefs(mediaType.ItemSchema, addRef)
	}
}

func isPureRefSchema(schema *openapi.Schema) bool {
	if schema == nil || schema.Ref == "" {
		return false
	}
	return schema.Schema == "" &&
		schema.ID == "" &&
		schema.Title == "" &&
		schema.Type == "" &&
		schema.Items == nil &&
		len(schema.Properties) == 0 &&
		len(schema.Defs) == 0 &&
		schema.Description == "" &&
		schema.DefaultValue == nil &&
		schema.Example == nil &&
		schema.Media == nil &&
		!schema.ReadOnly &&
		!schema.WriteOnly &&
		!schema.Deprecated &&
		schema.ContentEncoding == "" &&
		schema.ContentMediaType == "" &&
		schema.ContentSchema == nil &&
		schema.PathStart == "" &&
		len(schema.Links) == 0 &&
		len(schema.Enum) == 0 &&
		schema.Format == "" &&
		schema.Pattern == "" &&
		schema.ExclusiveMinimum == nil &&
		schema.Minimum == nil &&
		schema.ExclusiveMaximum == nil &&
		schema.Maximum == nil &&
		schema.MinLength == nil &&
		schema.MaxLength == nil &&
		schema.MinItems == nil &&
		schema.MaxItems == nil &&
		len(schema.Required) == 0 &&
		schema.AdditionalProperties == nil &&
		schema.UnevaluatedProperties == nil &&
		len(schema.AllOf) == 0 &&
		len(schema.AnyOf) == 0 &&
		len(schema.OneOf) == 0 &&
		schema.Discriminator == nil &&
		schema.XML == nil &&
		len(schema.Extensions) == 0
}

func toSchemaComponentRef(name string) string {
	return "#/components/schemas/" + name
}
