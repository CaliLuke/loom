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
		if operation.RequestBody != nil {
			for _, contentType := range operation.RequestBody.ContentTypes {
				consumes[contentType] = struct{}{}
				nonJSON = nonJSON || !isJSONMediaType(contentType)
			}
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
	if len(body.ContentTypes) == 0 {
		return nil, fmt.Errorf("render OpenAPI design: %s has no content type", path)
	}
	if len(body.ContentTypes) > 1 {
		return r.multipleRequestBody(body, path)
	}
	contentType := body.ContentTypes[0]
	mode, err := requestBodyModeFor(contentType, path)
	if err != nil {
		return nil, err
	}
	return r.singleRequestBody(body, mode, path)
}

func (r *renderer) multipleRequestBody(body RequestBody, path string) (*renderedBody, error) {
	for _, contentType := range body.ContentTypes {
		if !isMultipartMediaType(contentType) && !isFormMediaType(contentType) {
			continue
		}
		if err := r.validateRequestTransportBodySchema(
			body.Schema,
			path+"/content/"+escapeJSONPointer(contentType)+"/schema",
		); err != nil {
			return nil, err
		}
	}
	return &renderedBody{body: body, mode: requestBodyRaw}, nil
}

func requestBodyModeFor(contentType, path string) (requestBodyMode, error) {
	switch {
	case isJSONMediaType(contentType):
		return requestBodyJSON, nil
	case isMultipartMediaType(contentType):
		return requestBodyMultipart, nil
	case isFormMediaType(contentType):
		return requestBodyForm, nil
	default:
		return requestBodyJSON, fmt.Errorf("render OpenAPI design: %s content type %q is not renderable", path, contentType)
	}
}

func (r *renderer) singleRequestBody(body RequestBody, mode requestBodyMode, path string) (*renderedBody, error) {
	if mode != requestBodyJSON {
		if err := r.validateRequestTransportBodySchema(body.Schema, path+"/schema"); err != nil {
			return nil, err
		}
	}
	if mode == requestBodyForm {
		requiresRaw, err := r.formBodyRequiresRaw(body.Schema, path+"/schema", make(map[string]struct{}))
		if err != nil {
			return nil, err
		}
		if requiresRaw {
			return &renderedBody{body: body, mode: requestBodyRaw}, nil
		}
	}
	mapPayload := false
	if mode != requestBodyJSON {
		var err error
		mapPayload, err = r.requestTransportBodyIsMap(body.Schema, path+"/schema")
		if err != nil {
			return nil, err
		}
		if mapPayload && mode == requestBodyMultipart && !body.Required {
			return &renderedBody{body: body, mode: requestBodyRaw, mapPayload: true}, nil
		}
		if !mapPayload && mode == requestBodyForm && !body.Required {
			resolved, resolveErr := r.resolveRequestTransportBodySchema(body.Schema, path+"/schema")
			if resolveErr != nil {
				return nil, resolveErr
			}
			if len(resolved.Required) > 0 {
				return &renderedBody{body: body, mode: requestBodyRaw}, nil
			}
		}
	}
	return &renderedBody{body: body, field: "body", mode: mode, mapPayload: mapPayload}, nil
}

func (r *renderer) formBodyRequiresRaw(schema *Schema, path string, seen map[string]struct{}) (bool, error) {
	if schema == nil {
		return false, fmt.Errorf("render OpenAPI design: %s schema is nil", path)
	}
	if schema.Ref != "" {
		name := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		named, ok := r.schemas[name]
		if name == schema.Ref || name == "" || !ok {
			return false, fmt.Errorf("render OpenAPI design: %s schema reference %q does not resolve", path, schema.Ref)
		}
		if _, ok := seen[name]; ok {
			return false, nil
		}
		seen[name] = struct{}{}
		return r.formBodyRequiresRaw(named.Schema, path, seen)
	}
	if schema.Unconstrained {
		return true, nil
	}
	for index, property := range schema.Properties {
		requiresRaw, err := r.formBodyRequiresRaw(property.Schema, fmt.Sprintf("%s/properties/%d", path, index), seen)
		if err != nil || requiresRaw {
			return requiresRaw, err
		}
	}
	if schema.Items != nil {
		requiresRaw, err := r.formBodyRequiresRaw(schema.Items, path+"/items", seen)
		if err != nil || requiresRaw {
			return requiresRaw, err
		}
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Allowed != nil && *schema.AdditionalProperties.Allowed {
		return true, nil
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		return r.formBodyRequiresRaw(schema.AdditionalProperties.Schema, path+"/additionalProperties", seen)
	}
	return false, nil
}

func (r *renderer) operationResponses(
	operation *Operation,
	mode responseBodyMode,
	path string,
) (renderedResponse, []renderedResponse, error) {
	var success []renderedResponse
	var failures []renderedResponse
	successClass := successResponseClass(operation.Responses)
	for _, source := range operation.Responses {
		rawBody := mode != responseBodyEncoded && source.Response.Schema != nil &&
			!isJSONMediaType(source.Response.ContentType)
		response, err := r.renderedResponse(source, rawBody, path+"/responses/"+source.Status)
		if err != nil {
			return renderedResponse{}, nil, err
		}
		if isSuccessResponseStatus(source.Status, successClass) {
			success = append(success, response)
		} else {
			failures = append(failures, response)
		}
	}
	if len(success) != 1 {
		return renderedResponse{}, nil, fmt.Errorf(
			"render OpenAPI design: %s must define exactly one 2xx response or, when absent, exactly one 3xx response, got %d",
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
		status:    source.Status,
		errorName: "Status" + source.Status,
		response:  source.Response,
		headers:   headers,
		body:      bodyField,
		rawBody:   rawBody,
	}, nil
}
