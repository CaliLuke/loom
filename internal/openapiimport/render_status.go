package openapiimport

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

type renderedResponses struct {
	successes, failures, informational []renderedResponse
	mode                               responseBodyMode
}

// allocateFields uses one field namespace for a status-selected success result.
// Failure and informational values have separate native carrier namespaces.
func (p *renderedResponses) allocateFields() {
	if len(p.successes) > 1 || len(p.successes) == 1 && p.successes[0].statusField != "" {
		used := map[string]int{"http_status": 1, "httpstatus": 1}
		for i := range p.successes {
			response := &p.successes[i]
			response.statusField = "http_status"
			if len(p.successes) > 1 {
				response.branch = "status_" + strings.ToLower(selectorName(response.selector))
				if response.response.stream == nil && response.response.Schema != nil && !response.rawBody {
					response.body = response.branch
				}
				used[strings.ToLower(codegen.Goify(response.branch, false))] = 1
			}
		}
		for i := range p.successes {
			allocateResponseFields(&p.successes[i], used)
		}
	}
	for _, responses := range [][]renderedResponse{p.failures, p.informational} {
		for i := range responses {
			if responses[i].statusField != "" {
				allocateResponseFields(&responses[i], map[string]int{"http_status": 1, "httpstatus": 1})
			}
		}
	}
}

func allocateResponseFields(response *renderedResponse, used map[string]int) {
	for i := range response.headers {
		response.headers[i].field = uniqueName(codegen.Goify(response.headers[i].name, false), used)
	}
	if response.response.stream == nil && response.response.Schema != nil && !response.rawBody && response.branch == "" {
		response.body = uniqueName("body", used)
	}
}

func (r *renderer) operationResult(plan operationPlan, path string) error {
	if plan.mode == responseBodySSE {
		response := plan.successes[0].response
		return r.responseSchemaBody("StreamingResult", response.Schema, response.stream.path)
	}
	if len(plan.successes) == 0 {
		return nil
	}
	if len(plan.successes) == 1 {
		return r.result(plan.successes[0], path+"/responses/"+plan.successes[0].status)
	}
	r.open("Result(func()")
	r.line("Attribute(%q, Int)", "http_status")
	required := []string{"http_status"}
	for _, response := range plan.successes {
		p := path + "/responses/" + response.status
		if err := r.responseHeaderAttributes(response.headers, p+"/headers"); err != nil {
			return err
		}
		for _, header := range response.headers {
			if header.header.Required {
				required = append(required, header.field)
			}
		}
		if response.response.stream == nil && response.response.Schema != nil && !response.rawBody {
			if err := r.attribute(response.body, schemaWithExamples(response.response.Schema, response.response.Examples), nil, p+"/content/schema"); err != nil {
				return err
			}
			required = append(required, response.body)
		}
	}
	r.quotedCall("Required", required)
	r.close()
	return nil
}

func (r *renderer) informationalTypes() error {
	for i, operation := range r.operations {
		source := r.document.Operations[i]
		for _, response := range operation.informational {
			name := r.document.names.informational[cloneIdentity{source.Method, source.Path, response.status}]
			r.open("var %s = Type(%q, func()", name.variable, name.typeName)
			if response.statusField != "" {
				r.line("Attribute(%q, Int)", response.statusField)
			}
			if err := r.responseHeaderAttributes(response.headers, fmt.Sprintf("#/operations/%d/responses/%s/headers", i, response.status)); err != nil {
				return err
			}
			required := []string{}
			if response.statusField != "" {
				required = append(required, response.statusField)
			}
			for _, header := range response.headers {
				if header.header.Required {
					required = append(required, header.field)
				}
			}
			if len(required) > 0 {
				r.quotedCall("Required", required)
			}
			r.close()
			r.line("")
		}
	}
	return nil
}

func (r *renderer) informationalMapping(operation *Operation, response renderedResponse) error {
	name := r.document.names.informational[cloneIdentity{operation.Method, operation.Path, response.status}]
	r.open("Informational(%q, %s, %s, func()", name.typeName, selectorDSL(response.selector), name.variable)
	if response.statusField != "" {
		r.line("StatusField(%q)", response.statusField)
	}
	if err := r.responseMappingMetadata(response, false); err != nil {
		return err
	}
	r.responseMappingHeaders(response.headers)
	r.close()
	return nil
}
