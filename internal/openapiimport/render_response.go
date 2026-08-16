package openapiimport

import (
	"fmt"
	"strconv"
)

func (r *renderer) responseMapping(response renderedResponse, failure bool, path string) error {
	status, err := strconv.Atoi(response.status)
	if err != nil {
		return fmt.Errorf("render OpenAPI design: %s status %q is not numeric", path, response.status)
	}
	prefix := "Response("
	if failure {
		prefix += strconv.Quote("Status"+response.status) + ", "
	}
	prefix += strconv.Itoa(status)
	needsBlock := failure || response.response.Description != "" || response.response.ContentType != "" ||
		len(response.headers) > 0 || len(response.response.Extensions) > 0
	if !needsBlock {
		r.line("%s)", prefix)
		return nil
	}
	r.open("%s, func()", prefix)
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
	if response.rawBody {
		if err := r.openAPIBody(schemaWithExamples(response.response.Schema, response.response.Examples), path+"/content/schema"); err != nil {
			return err
		}
	}
	for _, header := range response.headers {
		name := header.field
		if name != header.name {
			name += ":" + header.name
		}
		r.line("Header(%q)", name)
	}
	wrapUnconstrainedError := failure && r.effectiveUnconstrained(response.response.Schema)
	if response.body != "" && (len(response.headers) > 0 || wrapUnconstrainedError) {
		r.line("Body(%q)", response.body)
	}
	if wrapUnconstrainedError || response.response.Schema != nil && response.response.Schema.Ref != "" {
		if err := r.openAPIBody(schemaWithExamples(response.response.Schema, response.response.Examples), path+"/content/schema"); err != nil {
			return err
		}
	}
	r.close()
	return nil
}

func schemaWithExamples(schema *Schema, examples []Example) *Schema {
	if schema == nil || len(examples) == 0 {
		return schema
	}
	combined := *schema
	combined.Examples = append(append([]Example(nil), schema.Examples...), examples...)
	return &combined
}
