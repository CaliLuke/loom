package ir

import (
	"strings"

	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

var fileResponseRequestHeaders = []struct {
	name        string
	description string
}{
	{name: "Range", description: "Byte range requested from the representation."},
	{name: "If-Range", description: "Validator controlling whether a range response may be returned."},
	{name: "If-Match", description: "Entity-tag precondition for serving the representation."},
	{name: "If-None-Match", description: "Entity-tag cache validator for the representation."},
	{name: "If-Modified-Since", description: "Modification-time cache validator for the representation."},
	{name: "If-Unmodified-Since", description: "Modification-time precondition for serving the representation."},
}

func addFileResponseRequestParameters(endpoint *transportir.Endpoint, params []*ParameterRef) []*ParameterRef {
	if endpoint == nil || endpoint.Response == nil || !endpoint.Response.FileResponse {
		return params
	}
	for _, header := range fileResponseRequestHeaders {
		if hasHeaderParameter(params, header.name) {
			continue
		}
		params = append(params, &ParameterRef{Value: &Parameter{
			Name:        header.name,
			In:          "header",
			Description: header.description,
			Schema:      &Schema{Type: "string"},
		}})
	}
	return params
}

func hasHeaderParameter(params []*ParameterRef, name string) bool {
	for _, param := range params {
		if param != nil && param.Value != nil && param.Value.In == "header" && strings.EqualFold(param.Value.Name, name) {
			return true
		}
	}
	return false
}

func addFileResponseProtocolResponses(endpoint *transportir.Endpoint, responses map[string]*Response) {
	if endpoint == nil || endpoint.Response == nil || !endpoint.Response.FileResponse {
		return
	}
	ok := responses["200"]
	if ok == nil {
		return
	}
	ensureFileResponseHeader(ok, "Accept-Ranges", "Range units supported by the response.", true)
	ensureFileResponseHeader(ok, "ETag", "Entity tag used for conditional requests.", false)
	ensureFileResponseHeader(ok, "Last-Modified", "Modification time used for conditional requests.", false)
	ensureFileResponseHeader(ok, "Content-Length", "Length in bytes when the representation is served without content encoding.", false)

	partial := &Response{
		Description: "Partial Content response produced for a satisfiable range request.",
		Headers:     cloneFileResponseHeaders(ok.Headers),
		Content:     cloneFileResponseMediaTypes(ok.Content),
	}
	if partial.Content == nil {
		partial.Content = make(map[string]*MediaType)
	}
	partial.Content["multipart/byteranges"] = &MediaType{Schema: &Schema{Type: "string", Format: "binary"}}
	ensureFileResponseHeader(partial, "Content-Range", "Byte range contained in a single-range response.", false)
	responses["206"] = partial

	notModified := &Response{Description: "Not Modified response produced when cache validators match."}
	ensureFileResponseHeader(notModified, "ETag", "Entity tag used for conditional requests.", false)
	ensureFileResponseHeader(notModified, "Last-Modified", "Modification time used for conditional requests.", false)
	responses["304"] = notModified

	preconditionFailed := &Response{Description: "Precondition Failed response produced when request validators fail."}
	ensureFileResponseHeader(preconditionFailed, "ETag", "Entity tag used for conditional requests.", false)
	ensureFileResponseHeader(preconditionFailed, "Last-Modified", "Modification time used for conditional requests.", false)
	responses["412"] = preconditionFailed

	rangeNotSatisfiable := &Response{
		Description: "Range Not Satisfiable response produced for an invalid byte range.",
		Content: map[string]*MediaType{
			"text/plain": {Schema: &Schema{Type: "string"}},
		},
	}
	ensureFileResponseHeader(rangeNotSatisfiable, "Content-Range", "Unsatisfied byte range and current representation length when available.", false)
	responses["416"] = rangeNotSatisfiable
}

func ensureFileResponseHeader(response *Response, name, description string, required bool) {
	if response.Headers == nil {
		response.Headers = make(map[string]*HeaderRef)
	}
	if response.Headers[name] != nil {
		return
	}
	response.Headers[name] = &HeaderRef{Value: &Header{
		Description: description,
		Required:    required,
		Schema:      &Schema{Type: "string"},
	}}
}

func cloneFileResponseHeaders(headers map[string]*HeaderRef) map[string]*HeaderRef {
	if len(headers) == 0 {
		return nil
	}
	cloned := make(map[string]*HeaderRef, len(headers))
	for name, header := range headers {
		cloned[name] = header
	}
	return cloned
}

func cloneFileResponseMediaTypes(content map[string]*MediaType) map[string]*MediaType {
	if len(content) == 0 {
		return nil
	}
	cloned := make(map[string]*MediaType, len(content))
	for contentType, mediaType := range content {
		cloned[contentType] = mediaType
	}
	return cloned
}
