package openapiimport

import (
	"fmt"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func (r *renderer) importedMediaTypes() ([]string, []string) {
	consumes := make(map[string]struct{})
	produces := make(map[string]struct{})
	nonJSON := false
	for _, operation := range r.document.Operations {
		if operation.RequestBody != nil && operation.RequestBody.ContentType != "" {
			consumes[operation.RequestBody.ContentType] = struct{}{}
			nonJSON = nonJSON || !isJSONMediaType(operation.RequestBody.ContentType)
		}
		for _, response := range operation.Responses {
			if response.Response.ContentType == "" {
				continue
			}
			produces[response.Response.ContentType] = struct{}{}
			nonJSON = nonJSON || !isJSONMediaType(response.Response.ContentType)
		}
	}
	if !nonJSON {
		return nil, nil
	}
	return sortedKeys(consumes), sortedKeys(produces)
}

func sortedKeys(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func (r *renderer) requestBody(source *RequestBody, path string) (*renderedBody, error) {
	if source == nil {
		return nil, nil
	}
	body := *source
	if body.Ref != "" {
		return nil, fmt.Errorf("render OpenAPI design: %s request body references are not renderable", path)
	}
	if body.Schema == nil {
		return nil, fmt.Errorf("render OpenAPI design: %s has no schema", path)
	}
	mode := requestBodyJSON
	switch {
	case isJSONMediaType(body.ContentType):
	case isMultipartMediaType(body.ContentType):
		mode = requestBodyMultipart
	case isFormMediaType(body.ContentType):
		mode = requestBodyForm
	default:
		return nil, fmt.Errorf("render OpenAPI design: %s content type %q is not renderable", path, body.ContentType)
	}
	if mode != requestBodyJSON {
		if err := r.validateRequestTransportBodySchema(body.Schema, path+"/schema"); err != nil {
			return nil, err
		}
	}
	return &renderedBody{body: body, field: "body", mode: mode}, nil
}

func (r *renderer) operationResponses(
	operation *Operation,
	mode responseBodyMode,
	path string,
) (renderedResponse, []renderedResponse, error) {
	var success []renderedResponse
	var failures []renderedResponse
	for _, source := range operation.Responses {
		rawBody := mode != responseBodyEncoded && source.Response.Schema != nil &&
			!isJSONMediaType(source.Response.ContentType)
		response, err := r.renderedResponse(source, rawBody, path+"/responses/"+source.Status)
		if err != nil {
			return renderedResponse{}, nil, err
		}
		if strings.HasPrefix(source.Status, "2") {
			success = append(success, response)
		} else {
			failures = append(failures, response)
		}
	}
	if len(success) != 1 {
		return renderedResponse{}, nil, fmt.Errorf(
			"render OpenAPI design: %s must define exactly one 2xx response, got %d",
			path,
			len(success),
		)
	}
	return success[0], failures, nil
}

func (r *renderer) renderedResponse(source StatusResponse, rawBody bool, path string) (renderedResponse, error) {
	if source.Response.Ref != "" {
		return renderedResponse{}, fmt.Errorf("render OpenAPI design: %s response references are not renderable", path)
	}
	if source.Response.Schema == nil && source.Response.ContentType != "" {
		return renderedResponse{}, fmt.Errorf("render OpenAPI design: %s has content type but no schema", path)
	}
	if source.Response.Schema != nil && (source.Response.ContentType == "" ||
		!rawBody && !isJSONMediaType(source.Response.ContentType)) {
		return renderedResponse{}, fmt.Errorf(
			"render OpenAPI design: %s content type %q is not renderable",
			path,
			source.Response.ContentType,
		)
	}
	used := make(map[string]int)
	headers := make([]renderedHeader, 0, len(source.Response.Headers))
	for _, named := range source.Response.Headers {
		header, err := r.resolveHeader(named.Header, path+"/headers/"+escapeJSONPointer(named.Name))
		if err != nil {
			return renderedResponse{}, err
		}
		// A deprecated header has no faithful Loom DSL representation; the
		// flag is intentionally dropped here. Analyze reports this omission
		// as the lossy-allowed "header-deprecated" diagnostic.
		if header.Schema == nil {
			return renderedResponse{}, fmt.Errorf("render OpenAPI design: %s header %q has no schema", path, named.Name)
		}
		field := uniqueName(codegen.Goify(named.Name, false), used)
		headers = append(headers, renderedHeader{name: named.Name, field: field, header: header})
	}
	bodyField := ""
	if source.Response.Schema != nil && !rawBody {
		bodyField = uniqueName("body", used)
	}
	return renderedResponse{
		status: source.Status, response: source.Response, headers: headers, body: bodyField, rawBody: rawBody,
	}, nil
}
