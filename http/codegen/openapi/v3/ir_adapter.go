package openapiv3

import openapiir "goa.design/goa/v3/http/codegen/openapi/internal/ir"

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
