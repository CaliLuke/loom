package codegen

import (
	"fmt"
	"slices"
	"strings"
)

// HasDynamicFinalSelectors reports whether decoding needs runtime key selection.
// Literal-only designs keep their existing exact-code dispatch.
func (e *EndpointData) HasDynamicFinalSelectors() bool {
	for _, key := range e.FinalKeys {
		if !key.IsExact() {
			return true
		}
	}
	return false
}

// SelectsNativeStatus reports whether successful publication reads a native
// StatusField. The expression owner has already validated its common identity.
func (e *EndpointData) SelectsNativeStatus() bool {
	return e.Method.HasResponseMetadata || len(e.Result.Responses) > 0 && e.Result.Responses[0].statusBinding != nil
}

// PreparesSuccessHeaders reports whether the successful payload stages its
// headers, including a dynamic loom-view, in preparedHeaders. A typed mixed
// unary payload selects status from metadata but publishes its own headers.
func (e *EndpointData) PreparesSuccessHeaders() bool {
	payloads := responsePayloads(e)
	return len(payloads) > 0 && payloads[0].HasStatusBinding()
}

// HasStatusBinding reports whether this response reads a native status field.
func (r *ResponseData) HasStatusBinding() bool {
	return r.statusBinding != nil
}

func finalSelectorValues(endpoint *EndpointData) string {
	keys := make([]string, len(endpoint.FinalKeys))
	for i, key := range endpoint.FinalKeys {
		keys[i] = fmt.Sprintf("%#v", responseContractSelectorValue(key))
	}
	return strings.Join(keys, ", ")
}

func responseSelectorValue(response *ResponseData) string {
	return fmt.Sprintf("%#v", responseContractSelectorValue(response.Selector))
}

func responseWireStatus(response *ResponseData) string {
	if response.HasStatusBinding() {
		return "status"
	}
	return response.LiteralStatus()
}

func renderFinalSuccessSelection(endpoint *EndpointData) string {
	if !endpoint.SelectsNativeStatus() {
		return ""
	}
	if endpoint.Method.HasResponseMetadata {
		return fmt.Sprintf("status := metadata.Status\nselected, found := loomhttp.SelectFinal([]loomhttp.StatusSelector{%s}, status)\nif !found {\nreturn &loomhttp.StatusViolationError{Code: status}\n}\n", finalSelectorValues(endpoint))
	}
	first := endpoint.Result.Responses[0]
	statusResponse := successfulStatusResponse(endpoint.Result.Responses)
	var code strings.Builder
	code.WriteString("transformErr := new(error)\nvar status int\n")
	code.WriteString(buildOutgoingStatusPlan(statusResponse, first.Carrier, first.serverContext, responseNativeSource(first.ViewedResult != nil), "status").Code)
	code.WriteString("if *transformErr != nil {\nreturn *transformErr\n}\n")
	fmt.Fprintf(&code, "selected, found := loomhttp.SelectFinal([]loomhttp.StatusSelector{%s}, status)\nif !found {\nreturn &loomhttp.StatusViolationError{Code: status}\n}\n", finalSelectorValues(endpoint))
	return code.String()
}

// successfulStatusResponse retains the common native status binding and the
// union of expression-owned successful domains; it does not recalculate them.
func successfulStatusResponse(responses []*ResponseData) *ResponseData {
	response := *responses[0]
	response.Domain = nil
	for _, item := range responses {
		response.Domain = append(response.Domain, item.Domain...)
	}
	slices.Sort(response.Domain)
	response.Domain = slices.Compact(response.Domain)
	return &response
}

func renderFinalErrorStatus(response *ResponseData) string {
	if !response.HasStatusBinding() {
		return ""
	}
	return "transformErr := new(error)\nvar status int\n" +
		buildOutgoingStatusPlan(response, response.Carrier, response.serverContext, responseNativeSource(response.ViewedResult != nil), "status").Code +
		"if *transformErr != nil {\nreturn *transformErr\n}\n"
}

func renderDecodedStatusValidation(response *ResponseData, value string) string {
	if !response.HasStatusBinding() {
		return ""
	}
	validation := responseConstructedStatusValidation(response, value)
	if validation == "" {
		return ""
	}
	return "if statusError := func() (err error) {\n" + validation + "\nreturn err\n}(); statusError != nil {\nreturn nil, statusError\n}\n"
}

func responsePayloads(endpoint *EndpointData) []*ResponseData {
	responses := make([]*ResponseData, len(endpoint.Result.Responses))
	for i, response := range endpoint.Result.Responses {
		responses[i] = response
		if response.Unary != nil {
			responses[i] = response.Unary
		}
	}
	return responses
}

func renderFinalMetadataPreparation(response *ResponseData, data *ServiceData) string {
	context := response.serverContext.Dup()
	context.ValidationPrefix = data.outgoingHeaderValidationPrefix
	value, view := responseNativeSource(response.ViewedResult != nil), ""
	if response.ViewedResult != nil {
		view = "res.View"
	}
	code := buildOutgoingHeaderPlan(response, response.Carrier, context, value, "status", "preparedHeaders", view).Code
	code += buildOutgoingCookiePlan(response, response.Carrier, context, value, "status", "preparedHeaders", "ctx").Code
	return code + "if *transformErr != nil {\nreturn *transformErr\n}\n"
}
