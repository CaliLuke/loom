package openapiv3

import (
	"goa.design/goa/v3/http/codegen/openapi"
	openapiir "goa.design/goa/v3/http/codegen/openapi/internal/ir"
)

func endpointBodiesToIR(bodies *EndpointBodies) *openapiir.EndpointBodies {
	if bodies == nil {
		return nil
	}
	responseBodies := make(map[int][]*openapiir.Schema, len(bodies.ResponseBodies))
	for status, schemas := range bodies.ResponseBodies {
		responseBodies[status] = schemaSliceToIR(schemas)
	}
	return &openapiir.EndpointBodies{
		RequestBody:    schemaToIR(bodies.RequestBody),
		ResponseBodies: responseBodies,
	}
}

func requestBodyFromIR(body *openapiir.RequestBody) *RequestBodyRef {
	if body == nil {
		return nil
	}
	return &RequestBodyRef{
		Value: &RequestBody{
			Description: body.Description,
			Required:    body.Required,
			Content:     mediaTypesFromIR(body.Content),
			Extensions:  cloneStringAnyMap(body.Extensions),
		},
	}
}

func responsesFromIR(responses map[string]*openapiir.Response) map[string]*ResponseRef {
	if len(responses) == 0 {
		return nil
	}
	out := make(map[string]*ResponseRef, len(responses))
	for status, response := range responses {
		out[status] = &ResponseRef{Value: responseFromIR(response)}
	}
	return out
}

func responseFromIR(response *openapiir.Response) *Response {
	if response == nil {
		return nil
	}
	desc := response.Description
	return &Response{
		Description: &desc,
		Headers:     headersFromIR(response.Headers),
		Content:     mediaTypesFromIR(response.Content),
		Extensions:  cloneStringAnyMap(response.Extensions),
	}
}

func headersFromIR(headers map[string]*openapiir.Header) map[string]*HeaderRef {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]*HeaderRef, len(headers))
	for name, header := range headers {
		out[name] = &HeaderRef{Value: &Header{
			Description: header.Description,
			Required:    header.Required,
			Schema:      openapiir.RenderSchema(header.Schema),
			Example:     header.Example,
			Examples:    examplesFromIR(header.Examples),
			Extensions:  cloneStringAnyMap(header.Extensions),
		}}
	}
	return out
}

func mediaTypesFromIR(mediaTypes map[string]*openapiir.MediaType) map[string]*MediaType {
	if len(mediaTypes) == 0 {
		return nil
	}
	out := make(map[string]*MediaType, len(mediaTypes))
	for contentType, mediaType := range mediaTypes {
		out[contentType] = &MediaType{
			Schema:     openapiir.RenderSchema(mediaType.Schema),
			Example:    mediaType.Example,
			Examples:   examplesFromIR(mediaType.Examples),
			Extensions: cloneStringAnyMap(mediaType.Extensions),
		}
	}
	return out
}

func examplesFromIR(examples map[string]*openapiir.Example) map[string]*ExampleRef {
	if len(examples) == 0 {
		return nil
	}
	out := make(map[string]*ExampleRef, len(examples))
	for name, example := range examples {
		out[name] = &ExampleRef{
			Value: &Example{
				Summary:     example.Summary,
				Description: example.Description,
				Value:       example.Value,
			},
		}
	}
	return out
}

func cloneStringAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func schemaSliceToIR(schemas []*openapi.Schema) []*openapiir.Schema {
	if len(schemas) == 0 {
		return nil
	}
	out := make([]*openapiir.Schema, len(schemas))
	for i, schema := range schemas {
		out[i] = schemaToIR(schema)
	}
	return out
}

func schemaToIR(schema *openapi.Schema) *openapiir.Schema {
	if schema == nil {
		return nil
	}
	out := &openapiir.Schema{
		Ref:                   schema.Ref,
		Type:                  string(schema.Type),
		Format:                schema.Format,
		Items:                 schemaToIR(schema.Items),
		Properties:            schemaMapToIR(schema.Properties),
		Defs:                  schemaMapToIR(schema.Defs),
		Description:           schema.Description,
		DefaultValue:          schema.DefaultValue,
		Example:               schema.Example,
		ReadOnly:              schema.ReadOnly,
		WriteOnly:             schema.WriteOnly,
		Deprecated:            schema.Deprecated,
		ContentEncoding:       schema.ContentEncoding,
		ContentMediaType:      schema.ContentMediaType,
		PathStart:             schema.PathStart,
		Links:                 linksToIR(schema.Links),
		Enum:                  append([]any(nil), schema.Enum...),
		Pattern:               schema.Pattern,
		ExclusiveMinimum:      schema.ExclusiveMinimum,
		Minimum:               schema.Minimum,
		ExclusiveMaximum:      schema.ExclusiveMaximum,
		Maximum:               schema.Maximum,
		MinLength:             schema.MinLength,
		MaxLength:             schema.MaxLength,
		MinItems:              schema.MinItems,
		MaxItems:              schema.MaxItems,
		Required:              append([]string(nil), schema.Required...),
		AdditionalProperties:  boolOrSchemaToIR(schema.AdditionalProperties),
		UnevaluatedProperties: boolOrSchemaToIR(schema.UnevaluatedProperties),
		AnyOf:                 schemaSliceToIR(schema.AnyOf),
		OneOf:                 schemaSliceToIR(schema.OneOf),
		Discriminator:         discriminatorToIR(schema.Discriminator),
		Extensions:            cloneStringAnyMap(schema.Extensions),
	}
	if schema.Media != nil {
		out.Media = &openapiir.Media{
			BinaryEncoding: schema.Media.BinaryEncoding,
			Type:           schema.Media.Type,
		}
	}
	return out
}

func schemaMapToIR(schemas map[string]*openapi.Schema) map[string]*openapiir.Schema {
	if len(schemas) == 0 {
		return nil
	}
	out := make(map[string]*openapiir.Schema, len(schemas))
	for name, schema := range schemas {
		out[name] = schemaToIR(schema)
	}
	return out
}

func boolOrSchemaToIR(value any) *openapiir.BoolOrSchema {
	switch actual := value.(type) {
	case bool:
		return &openapiir.BoolOrSchema{Bool: &actual}
	case *openapi.Schema:
		return &openapiir.BoolOrSchema{Schema: schemaToIR(actual)}
	default:
		return nil
	}
}

func discriminatorToIR(discriminator *openapi.Discriminator) *openapiir.Discriminator {
	if discriminator == nil {
		return nil
	}
	return &openapiir.Discriminator{
		PropertyName: discriminator.PropertyName,
		Mapping:      cloneStringMap(discriminator.Mapping),
	}
}

func linksToIR(links []*openapi.Link) []*openapiir.Link {
	if len(links) == 0 {
		return nil
	}
	out := make([]*openapiir.Link, len(links))
	for i, link := range links {
		out[i] = &openapiir.Link{
			Title:        link.Title,
			Description:  link.Description,
			Rel:          link.Rel,
			Href:         link.Href,
			Method:       link.Method,
			Schema:       schemaToIR(link.Schema),
			TargetSchema: schemaToIR(link.TargetSchema),
			ResultType:   link.ResultType,
			EncType:      link.EncType,
		}
	}
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
