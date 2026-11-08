package openapiimport

import (
	"cmp"
	"fmt"
	"iter"
	"slices"
	"strconv"

	"github.com/CaliLuke/loom/expr"
)

type (
	// operationResponsePlan classifies source occurrences before allocating any
	// design names. Its groups are the sole policy input for admission and rendering.
	operationResponsePlan struct {
		successes, failures, informational []plannedResponse
		mode                               responseBodyMode
		handshake                          bool
	}
	plannedResponse struct {
		source      StatusResponse
		sourceIndex int
		rawBody     bool
		selector    expr.HTTPStatusSelector
	}
)

func classifyOperationResponses(operation *Operation, path string, components Components) (operationResponsePlan, Diagnostics) {
	var plan operationResponsePlan
	diagnostics := make(Diagnostics, 0, len(operation.Responses))
	var finals []plannedResponse
	preferred := expr.HTTPStatusClass(0)
	for sourceIndex, source := range operation.Responses {
		p := path + "/responses/" + escapeJSONPointer(source.Status)
		selector, err := expr.ParseOpenAPIStatusKey(source.Status)
		if err != nil {
			diagnostics.add("response-status", p, "response status must be an HTTP code, uppercase 1XX-5XX, or default")
			continue
		}
		response := plannedResponse{source: source, sourceIndex: sourceIndex, selector: selector}
		switch {
		case selector.Code == 101:
			diagnostics.add("response-status", p, "101 describes a protocol upgrade OpenAPI cannot define")
		case selector.Phase() == expr.PhaseInformational:
			if source.Response.Schema != nil || source.Response.ContentType != "" {
				diagnostics.add("response-status", p+"/content", "1xx responses cannot contain content")
			}
			plan.informational = append(plan.informational, response)
		default:
			finals = append(finals, response)
			class := selector.Class
			if selector.IsExact() {
				class = expr.HTTPStatusClass(selector.Code / 100)
			}
			if class == expr.Status2XX || class == expr.Status3XX && preferred != expr.Status2XX {
				preferred = class
			}
		}
	}
	for _, response := range finals {
		class := response.selector.Class
		if response.selector.IsExact() {
			class = expr.HTTPStatusClass(response.selector.Code / 100)
		}
		if class == preferred || preferred == 0 && class == expr.StatusFallback {
			plan.successes = append(plan.successes, response)
		} else {
			plan.failures = append(plan.failures, response)
		}
	}
	if len(finals) == 0 && len(diagnostics) == 0 {
		diagnostics.add("response-status", path+"/responses", "Responses must contain a response code")
	}
	diagnostics = append(diagnostics, plan.complete(operation, path, components)...)
	return plan, diagnostics
}

func (plan *operationResponsePlan) complete(operation *Operation, path string, components Components) Diagnostics {
	diagnostics := plan.validateSSE(path, components)
	plan.mode = plan.bodyMode(operation)
	if plan.mode == responseBodySSE {
		plan.handshake = len(plan.successes) != 1 || !plan.successes[0].selector.IsExact() || len(plan.successes[0].source.Response.Headers) > 0
		slices.SortFunc(plan.successes, func(a, b plannedResponse) int {
			return cmp.Compare(a.selector.OpenAPIKey(), b.selector.OpenAPIKey())
		})
	}
	for i := range plan.successes {
		plan.successes[i].rawBody = plan.mode == responseBodyStream || plan.mode == responseBodyFile
	}
	return diagnostics
}

func (p operationResponsePlan) bodyMode(operation *Operation) responseBodyMode {
	for _, response := range p.successes {
		if response.source.Response.stream != nil {
			return responseBodySSE
		}
	}
	for _, planned := range p.successes {
		response := planned.source
		if response.Response.Schema == nil || isJSONMediaType(response.Response.ContentType) || len(response.Response.Schema.MediaTypes) > 0 {
			continue
		}
		if p.fileEligible(operation) {
			return responseBodyFile
		}
		return responseBodyStream
	}
	return responseBodyEncoded
}

// fileEligible preserves ServeContent's exact-only protocol ownership. A class
// anywhere in the operation (including errors or informational declarations)
// requires the ordinary raw-body transport instead.
func (p operationResponsePlan) fileEligible(operation *Operation) bool {
	if len(p.successes) != 1 || p.successes[0].selector.Code != 200 || operation.Method != "GET" && operation.Method != "HEAD" || hasFileProtocolResponse(operation.Responses) {
		return false
	}
	for _, group := range [][]plannedResponse{p.successes, p.failures, p.informational} {
		for _, response := range group {
			if !response.selector.IsExact() {
				return false
			}
		}
	}
	return true
}

func selectorDSL(selector expr.HTTPStatusSelector) string {
	if selector.IsExact() {
		return strconv.Itoa(selector.Code)
	}
	if selector.Class == expr.StatusFallback {
		return "StatusFallback"
	}
	return fmt.Sprintf("Status%dXX", selector.Class)
}

func selectorName(selector expr.HTTPStatusSelector) string {
	if selector.Class == expr.StatusFallback {
		return "Default"
	}
	return selector.OpenAPIKey()
}

// responses visits classified occurrences in source order with their recorded
// transport roles. Promotion and name allocation retain source ordering.
func (p operationResponsePlan) responses() iter.Seq[plannedResponse] {
	responses := make([]plannedResponse, 0, len(p.successes)+len(p.failures)+len(p.informational))
	responses = append(responses, p.successes...)
	responses = append(responses, p.failures...)
	responses = append(responses, p.informational...)
	slices.SortFunc(responses, func(a, b plannedResponse) int {
		return cmp.Compare(a.sourceIndex, b.sourceIndex)
	})
	return slices.Values(responses)
}
