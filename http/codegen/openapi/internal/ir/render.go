package ir

import "github.com/CaliLuke/loom/http/codegen/openapi"

type (
	// RenderedEndpointBodies is the rendered endpoint body set used by v3 adapters.
	RenderedEndpointBodies struct {
		RequestBody    *openapi.Schema
		ResponseBodies map[int][]*openapi.Schema
	}
)

// RenderExternalDocs converts IR external docs to the concrete OpenAPI type.
func RenderExternalDocs(docs *ExternalDocs) *openapi.ExternalDocs {
	if docs == nil {
		return nil
	}
	return &openapi.ExternalDocs{
		Description: docs.Description,
		URL:         docs.URL,
		Extensions:  cloneMap(docs.Extensions),
	}
}

// RenderSchema converts an IR schema to the concrete OpenAPI schema type.
func RenderSchema(schema *Schema) *openapi.Schema {
	if schema == nil {
		return nil
	}
	out := openapi.NewSchema()
	out.Ref = schema.Ref
	out.Type = openapi.Type(schema.Type)
	out.Format = schema.Format
	out.Items = RenderSchema(schema.Items)
	out.Properties = RenderSchemaMap(schema.Properties)
	out.Defs = RenderSchemaMap(schema.Defs)
	out.Description = schema.Description
	out.DefaultValue = schema.DefaultValue
	out.Example = schema.Example
	out.Media = renderMedia(schema.Media)
	out.ReadOnly = schema.ReadOnly
	out.WriteOnly = schema.WriteOnly
	out.Deprecated = schema.Deprecated
	out.ContentEncoding = schema.ContentEncoding
	out.ContentMediaType = schema.ContentMediaType
	out.ContentSchema = RenderSchema(schema.ContentSchema)
	out.PathStart = schema.PathStart
	out.Links = renderLinks(schema.Links)
	out.Enum = schema.Enum
	out.Pattern = schema.Pattern
	out.ExclusiveMinimum = schema.ExclusiveMinimum
	out.Minimum = schema.Minimum
	out.ExclusiveMaximum = schema.ExclusiveMaximum
	out.Maximum = schema.Maximum
	out.MinLength = schema.MinLength
	out.MaxLength = schema.MaxLength
	out.MinItems = schema.MinItems
	out.MaxItems = schema.MaxItems
	out.Required = append([]string(nil), schema.Required...)
	out.AdditionalProperties = renderBoolOrSchema(schema.AdditionalProperties)
	out.UnevaluatedProperties = renderBoolOrSchema(schema.UnevaluatedProperties)
	out.AnyOf = renderSchemaSlice(schema.AnyOf)
	out.OneOf = renderSchemaSlice(schema.OneOf)
	out.Discriminator = renderDiscriminator(schema.Discriminator)
	out.XML = renderXML(schema.XML)
	out.Extensions = cloneMap(schema.Extensions)
	return out
}

// RenderSchemaMap converts a schema map.
func RenderSchemaMap(schemas map[string]*Schema) map[string]*openapi.Schema {
	if len(schemas) == 0 {
		return nil
	}
	out := make(map[string]*openapi.Schema, len(schemas))
	for name, schema := range schemas {
		out[name] = RenderSchema(schema)
	}
	return out
}

// RenderBodyTypes converts IR endpoint bodies and components to rendered OpenAPI schemas.
func RenderBodyTypes(bodyTypes *BodyTypes) (map[string]map[string]*RenderedEndpointBodies, map[string]*openapi.Schema) {
	if bodyTypes == nil {
		return nil, nil
	}
	services := make(map[string]map[string]*RenderedEndpointBodies, len(bodyTypes.Services))
	for serviceName, methods := range bodyTypes.Services {
		renderedMethods := make(map[string]*RenderedEndpointBodies, len(methods))
		for methodName, bodies := range methods {
			renderedMethods[methodName] = renderEndpointBodies(bodies)
		}
		services[serviceName] = renderedMethods
	}
	return services, RenderSchemaMap(bodyTypes.Components)
}

func renderEndpointBodies(bodies *EndpointBodies) *RenderedEndpointBodies {
	if bodies == nil {
		return nil
	}
	rendered := &RenderedEndpointBodies{
		RequestBody: RenderSchema(bodies.RequestBody),
	}
	if len(bodies.ResponseBodies) > 0 {
		rendered.ResponseBodies = make(map[int][]*openapi.Schema, len(bodies.ResponseBodies))
		for status, schemas := range bodies.ResponseBodies {
			rendered.ResponseBodies[status] = renderSchemaSlice(schemas)
		}
	}
	return rendered
}

func renderSchemaSlice(schemas []*Schema) []*openapi.Schema {
	if len(schemas) == 0 {
		return nil
	}
	out := make([]*openapi.Schema, len(schemas))
	for i, schema := range schemas {
		out[i] = RenderSchema(schema)
	}
	return out
}

func renderBoolOrSchema(value *BoolOrSchema) any {
	if value == nil {
		return nil
	}
	if value.Bool != nil {
		return *value.Bool
	}
	return RenderSchema(value.Schema)
}

func renderDiscriminator(discriminator *Discriminator) *openapi.Discriminator {
	if discriminator == nil {
		return nil
	}
	out := &openapi.Discriminator{
		PropertyName:   discriminator.PropertyName,
		DefaultMapping: discriminator.DefaultMapping,
		Optional:       discriminator.Optional,
	}
	if len(discriminator.Mapping) > 0 {
		out.Mapping = cloneMap(discriminator.Mapping)
	}
	return out
}

func renderXML(xml *XML) *openapi.XML {
	if xml == nil {
		return nil
	}
	return &openapi.XML{
		Name:      xml.Name,
		Namespace: xml.Namespace,
		Prefix:    xml.Prefix,
		NodeType:  xml.NodeType,
	}
}

func renderMedia(media *Media) *openapi.Media {
	if media == nil {
		return nil
	}
	return &openapi.Media{
		BinaryEncoding: media.BinaryEncoding,
		Type:           media.Type,
	}
}

func renderLinks(links []*Link) []*openapi.Link {
	if len(links) == 0 {
		return nil
	}
	out := make([]*openapi.Link, len(links))
	for i, link := range links {
		out[i] = &openapi.Link{
			Title:        link.Title,
			Description:  link.Description,
			Rel:          link.Rel,
			Href:         link.Href,
			Method:       link.Method,
			Schema:       RenderSchema(link.Schema),
			TargetSchema: RenderSchema(link.TargetSchema),
			ResultType:   link.ResultType,
			EncType:      link.EncType,
		}
	}
	return out
}

func cloneMap[T any](in map[string]T) map[string]T {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]T, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
