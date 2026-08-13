package openapiv3

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

const parameterComponentRefPrefix = "#/components/parameters/"

type reusableComponents struct {
	Parameters    map[string]*ParameterRef
	Headers       map[string]*HeaderRef
	RequestBodies map[string]*RequestBodyRef
	Responses     map[string]*ResponseRef
	Examples      map[string]*ExampleRef
}

func collectReusableComponentSchemaRefs(reusable reusableComponents, addRef func(string)) {
	for _, parameter := range reusable.Parameters {
		if parameter == nil || parameter.Value == nil {
			continue
		}
		collectSchemaRefs(parameter.Value.Schema, addRef)
	}
	for _, header := range reusable.Headers {
		if header == nil || header.Value == nil {
			continue
		}
		collectSchemaRefs(header.Value.Schema, addRef)
		collectMediaTypeSchemaRefs(header.Value.Content, addRef)
	}
	for _, requestBody := range reusable.RequestBodies {
		if requestBody == nil || requestBody.Value == nil {
			continue
		}
		collectMediaTypeSchemaRefs(requestBody.Value.Content, addRef)
	}
	for _, response := range reusable.Responses {
		if response == nil || response.Value == nil {
			continue
		}
		for _, header := range response.Value.Headers {
			if header == nil || header.Value == nil {
				continue
			}
			collectSchemaRefs(header.Value.Schema, addRef)
			collectMediaTypeSchemaRefs(header.Value.Content, addRef)
		}
		collectMediaTypeSchemaRefs(response.Value.Content, addRef)
	}
}

func collectMediaTypeSchemaRefs(mediaTypes map[string]*MediaType, addRef func(string)) {
	for _, mediaType := range mediaTypes {
		if mediaType == nil {
			continue
		}
		collectSchemaRefs(mediaType.Schema, addRef)
		collectSchemaRefs(mediaType.ItemSchema, addRef)
	}
}

func operationTagNames(endpointMeta, serviceMeta expr.MetaExpr, serviceName string) []string {
	tagNames := openapi.TagNamesFromExpr(endpointMeta)
	if len(tagNames) > 0 {
		return tagNames
	}
	tagNames = openapi.TagNamesFromExpr(serviceMeta)
	if len(tagNames) > 0 {
		return tagNames
	}
	return []string{serviceName}
}

func parameterComponentName(parameter *Parameter) string {
	if parameter == nil {
		return "Parameter"
	}
	base := codegen.Goify(parameter.In, true) + codegen.Goify(parameterComponentSuffix(parameter.Name), true)
	if base == "" {
		return "Parameter"
	}
	return base
}

func parameterComponentSuffix(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "Parameter"
	}
	return trimmed
}
