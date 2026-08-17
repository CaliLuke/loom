package openapiimport

import (
	"fmt"
	"strconv"
)

func headerMetadata(header Header) []renderedMetadata {
	if !header.AllowReserved {
		return nil
	}
	return []renderedMetadata{{name: "openapi:allowReserved", value: "true"}}
}

func (r *renderer) responseHeaderAttributes(headers []renderedHeader, path string) error {
	for _, header := range headers {
		metadata := headerMetadata(header.header)
		if err := r.attribute(
			header.field,
			header.header.Schema,
			header.header.Description,
			path+"/"+escapeJSONPointer(header.name),
			metadata...,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) responseMapping(response renderedResponse, failure bool, path string) error {
	prefix, err := responseMappingPrefix(response, failure, path)
	if err != nil {
		return err
	}
	if !responseMappingNeedsBlock(response, failure) {
		r.line("%s)", prefix)
		return nil
	}
	r.open("%s, func()", prefix)
	if err := r.responseMappingMetadata(response, failure); err != nil {
		return err
	}
	if err := r.responseMappingRawBody(response, path); err != nil {
		return err
	}
	r.responseMappingHeaders(response.headers)
	if err := r.responseMappingBody(response, failure, path); err != nil {
		return err
	}
	r.close()
	return nil
}

func responseMappingPrefix(response renderedResponse, failure bool, path string) (string, error) {
	status, err := strconv.Atoi(response.status)
	if err != nil {
		return "", fmt.Errorf("render OpenAPI design: %s status %q is not numeric", path, response.status)
	}
	prefix := "Response("
	if failure {
		prefix += strconv.Quote(response.errorName) + ", "
	}
	return prefix + strconv.Itoa(status), nil
}

func responseMappingNeedsBlock(response renderedResponse, failure bool) bool {
	return failure || response.response.Summary != "" || response.response.Description != "" ||
		response.response.ContentType != "" || len(response.headers) > 0 || len(response.response.Extensions) > 0
}

func (r *renderer) responseMappingMetadata(response renderedResponse, failure bool) error {
	if response.response.Summary != "" {
		r.line("Meta(%q, %q)", "openapi:summary", response.response.Summary)
	}
	if response.response.Description != "" {
		r.line("Description(%q)", response.response.Description)
	}
	if failure {
		r.line("Meta(%q, %q)", "openapi:description:errorName", "false")
	}
	if err := r.emitExtensions("response", response.response.Extensions); err != nil {
		return err
	}
	if response.response.ContentType != "" {
		r.line("ContentType(%q)", response.response.ContentType)
	}
	if failure && response.response.Schema == nil {
		r.line("Body(Empty)")
	}
	return nil
}

func (r *renderer) responseMappingRawBody(response renderedResponse, path string) error {
	if response.rawBody {
		if err := r.openAPIBody(schemaWithExamples(response.response.Schema, response.response.Examples), path+"/content/schema"); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) responseMappingBody(response renderedResponse, failure bool, path string) error {
	wrapUnconstrainedError := failure && r.effectiveUnconstrained(response.response.Schema)
	if response.body != "" && (len(response.headers) > 0 || wrapUnconstrainedError) {
		r.line("Body(%q)", response.body)
	}
	if wrapUnconstrainedError || response.response.Schema != nil && response.response.Schema.Ref != "" {
		if err := r.openAPIBody(schemaWithExamples(response.response.Schema, response.response.Examples), path+"/content/schema"); err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) responseMappingHeaders(headers []renderedHeader) {
	for _, header := range headers {
		name := header.field
		if name != header.name {
			name += ":" + header.name
		}
		r.line("Header(%q)", name)
	}
}

func schemaWithExamples(schema *Schema, examples []Example) *Schema {
	if schema == nil || len(examples) == 0 {
		return schema
	}
	combined := *schema
	combined.Examples = append(append([]Example(nil), schema.Examples...), examples...)
	return &combined
}
