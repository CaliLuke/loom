package openapiv3

import (
	"github.com/CaliLuke/loom/http/codegen/openapi"
	openapiir "github.com/CaliLuke/loom/http/codegen/openapi/internal/ir"
)

func reusableComponentsFromIR(components *openapiir.Components) reusableComponents {
	if components == nil {
		return reusableComponents{}
	}
	return reusableComponents{
		Parameters:    parameterComponentsFromIR(components.Parameters),
		Headers:       headersFromIR(components.Headers),
		RequestBodies: requestBodyComponentsFromIR(components.RequestBodies),
		Responses:     responsesFromIR(components.Responses),
		Examples:      examplesFromIR(components.Examples),
	}
}

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

func parameterComponentsFromIR(parameters map[string]*openapiir.ParameterRef) map[string]*ParameterRef {
	if len(parameters) == 0 {
		return nil
	}
	out := make(map[string]*ParameterRef, len(parameters))
	for name, parameter := range parameters {
		if converted := parameterRefFromIR(parameter); converted != nil {
			out[name] = converted
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func requestBodyComponentsFromIR(requestBodies map[string]*openapiir.RequestBodyRef) map[string]*RequestBodyRef {
	if len(requestBodies) == 0 {
		return nil
	}
	out := make(map[string]*RequestBodyRef, len(requestBodies))
	for name, body := range requestBodies {
		if converted := requestBodyRefFromIR(body); converted != nil {
			out[name] = converted
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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

func requestBodyRefFromIR(body *openapiir.RequestBodyRef) *RequestBodyRef {
	if body == nil {
		return nil
	}
	if body.Ref != "" {
		return &RequestBodyRef{Ref: body.Ref}
	}
	return requestBodyFromIR(body.Value)
}

func responsesFromIR(responses map[string]*openapiir.ResponseRef) map[string]*ResponseRef {
	if len(responses) == 0 {
		return nil
	}
	out := make(map[string]*ResponseRef, len(responses))
	for status, response := range responses {
		if converted := responseRefFromIR(response); converted != nil {
			out[status] = converted
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func responseRefFromIR(response *openapiir.ResponseRef) *ResponseRef {
	if response == nil {
		return nil
	}
	if response.Ref != "" {
		return &ResponseRef{Ref: response.Ref}
	}
	return &ResponseRef{Value: responseFromIR(response.Value)}
}

func responseFromIR(response *openapiir.Response) *Response {
	if response == nil {
		return nil
	}
	desc := response.Description
	var description *string
	if !response.OmitDescription {
		description = &desc
	}
	return &Response{
		Summary:                  response.Summary,
		Description:              description,
		Headers:                  headersFromIR(response.Headers),
		Content:                  mediaTypesFromIR(response.Content),
		Links:                    responseLinksFromIR(response.Links),
		Extensions:               cloneStringAnyMap(response.Extensions),
		CompatibilityDescription: desc,
	}
}

func responseLinksFromIR(links map[string]*openapiir.ResponseLinkRef) map[string]*LinkRef {
	if len(links) == 0 {
		return nil
	}
	out := make(map[string]*LinkRef, len(links))
	for name, link := range links {
		if converted := responseLinkRefFromIR(link); converted != nil {
			out[name] = converted
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func responseLinkRefFromIR(link *openapiir.ResponseLinkRef) *LinkRef {
	if link == nil {
		return nil
	}
	if link.Ref != "" {
		return &LinkRef{Ref: link.Ref}
	}
	if link.Value == nil {
		return nil
	}
	return &LinkRef{Value: &Link{
		OperationID:  link.Value.OperationID,
		OperationRef: link.Value.OperationRef,
		Description:  link.Value.Description,
		Parameters:   cloneStringAnyMap(link.Value.Parameters),
		RequestBody:  link.Value.RequestBody,
		Extensions:   cloneStringAnyMap(link.Value.Extensions),
	}}
}

func headersFromIR(headers map[string]*openapiir.HeaderRef) map[string]*HeaderRef {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]*HeaderRef, len(headers))
	for name, header := range headers {
		if converted := headerRefFromIR(header); converted != nil {
			out[name] = converted
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func headerRefFromIR(header *openapiir.HeaderRef) *HeaderRef {
	if header == nil {
		return nil
	}
	if header.Ref != "" {
		return &HeaderRef{Ref: header.Ref}
	}
	return &HeaderRef{Value: &Header{
		Description:   header.Value.Description,
		Required:      header.Value.Required,
		AllowReserved: header.Value.AllowReserved,
		Schema:        openapiir.RenderSchema(header.Value.Schema),
		Example:       header.Value.Example,
		Examples:      examplesFromIR(header.Value.Examples),
		Extensions:    cloneStringAnyMap(header.Value.Extensions),
	}}
}

func parametersFromIR(parameters []*openapiir.ParameterRef) []*ParameterRef {
	if len(parameters) == 0 {
		return nil
	}
	out := make([]*ParameterRef, len(parameters))
	for i, parameter := range parameters {
		out[i] = parameterRefFromIR(parameter)
	}
	return out
}

func parameterRefFromIR(parameter *openapiir.ParameterRef) *ParameterRef {
	if parameter == nil {
		return nil
	}
	if parameter.Ref != "" {
		return &ParameterRef{Ref: parameter.Ref}
	}
	if parameter.Value == nil {
		return nil
	}
	return &ParameterRef{Value: &Parameter{
		Name:             parameter.Value.Name,
		In:               parameter.Value.In,
		Description:      parameter.Value.Description,
		Style:            parameter.Value.Style,
		Explode:          parameter.Value.Explode,
		AllowEmptyValue:  parameter.Value.AllowEmptyValue,
		AllowReserved:    parameter.Value.AllowReserved,
		Deprecated:       parameter.Value.Deprecated,
		Required:         parameter.Value.Required,
		Schema:           openapiir.RenderSchema(parameter.Value.Schema),
		Example:          parameter.Value.Example,
		Examples:         examplesFromIR(parameter.Value.Examples),
		Content:          mediaTypesFromIR(parameter.Value.Content),
		WholeQueryString: parameter.Value.WholeQueryString,
		Extensions:       cloneStringAnyMap(parameter.Value.Extensions),
	}}
}

func mediaTypesFromIR(mediaTypes map[string]*openapiir.MediaType) map[string]*MediaType {
	if len(mediaTypes) == 0 {
		return nil
	}
	out := make(map[string]*MediaType, len(mediaTypes))
	for contentType, mediaType := range mediaTypes {
		converted := &MediaType{
			Schema:        openapiir.RenderSchema(mediaType.Schema),
			Example:       mediaType.Example,
			Examples:      examplesFromIR(mediaType.Examples),
			ComponentName: mediaType.ComponentName,
			Extensions:    cloneStringAnyMap(mediaType.Extensions),
		}
		applyMediaTypeMetadata(converted, mediaType.Metadata)
		switch contentType {
		case "application/jsonl", "application/json-seq":
			converted.UseItemSchema = true
		}
		out[contentType] = converted
	}
	return out
}

func examplesFromIR(examples map[string]*openapiir.ExampleRef) map[string]*ExampleRef {
	if len(examples) == 0 {
		return nil
	}
	out := make(map[string]*ExampleRef, len(examples))
	for name, example := range examples {
		if converted := exampleRefFromIR(example); converted != nil {
			out[name] = converted
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func exampleRefFromIR(example *openapiir.ExampleRef) *ExampleRef {
	if example == nil {
		return nil
	}
	if example.Ref != "" {
		return &ExampleRef{Ref: example.Ref}
	}
	if example.Value == nil {
		return nil
	}
	return &ExampleRef{Value: &Example{
		Summary:            example.Value.Summary,
		Description:        example.Value.Description,
		Value:              example.Value.Value,
		DataValue:          example.Value.DataValue,
		SerializedValue:    example.Value.SerializedValue,
		CompatibilityValue: firstNonNil(example.Value.Value, example.Value.DataValue),
	}}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func externalDocsFromIR(docs *openapiir.ExternalDocs) *openapi.ExternalDocs {
	return openapiir.RenderExternalDocs(docs)
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
		Title:                 schema.Title,
		Description:           schema.Description,
		DefaultValue:          schema.DefaultValue,
		Example:               schema.Example,
		ReadOnly:              schema.ReadOnly,
		WriteOnly:             schema.WriteOnly,
		Deprecated:            schema.Deprecated,
		ContentEncoding:       schema.ContentEncoding,
		ContentMediaType:      schema.ContentMediaType,
		ContentSchema:         schemaToIR(schema.ContentSchema),
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
		AllOf:                 schemaSliceToIR(schema.AllOf),
		AnyOf:                 schemaSliceToIR(schema.AnyOf),
		OneOf:                 schemaSliceToIR(schema.OneOf),
		Discriminator:         discriminatorToIR(schema.Discriminator),
		XML:                   xmlToIR(schema.XML),
		Extensions:            cloneStringAnyMap(schema.Extensions),
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
		PropertyName:   discriminator.PropertyName,
		Mapping:        cloneStringMap(discriminator.Mapping),
		DefaultMapping: discriminator.DefaultMapping,
		Optional:       discriminator.Optional,
	}
}

func xmlToIR(xml *openapi.XML) *openapiir.XML {
	if xml == nil {
		return nil
	}
	return &openapiir.XML{
		Name:      xml.Name,
		Namespace: xml.Namespace,
		Prefix:    xml.Prefix,
		NodeType:  xml.NodeType,
	}
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
