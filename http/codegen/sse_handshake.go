package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func sseHandshakeServerSections(endpoint *EndpointData, data *ServiceData) []codegen.Section {
	if endpoint.SSE == nil {
		return nil
	}
	return []codegen.Section{codegen.NewJenniferSection("sse-handshake-encoder", func(statement *jen.Statement) {
		statement.Add(codegen.Expr(renderSSEHandshakeEncoder(endpoint, data)))
	})}
}

func renderSSEHandshakeEncoder(endpoint *EndpointData, data *ServiceData) string {
	var b strings.Builder
	sse := endpoint.SSE
	parameter := "ctx context.Context"
	if sse.HandshakeRef != "" {
		parameter += ", value " + sse.HandshakeRef
	}
	fmt.Fprintf(&b, "// %s prepares immutable stream response metadata before publication.\nfunc %s(%s) (loomhttp.SSEHandshake, error) {\n", sse.PrepareHandshake, sse.PrepareHandshake, parameter)
	responses := endpoint.Result.Responses
	if len(responses) == 0 {
		panic("SSE endpoint has no successful response")
	}
	first := responses[0]
	if sse.HandshakeRef == "" {
		fmt.Fprintf(&b, "return loomhttp.SSEHandshake{Status: %d}, nil\n}\n", first.LiteralCode())
		return b.String()
	}
	context := outgoingHeaderContext(data, first.Carrier)
	// Domains are already partitioned by the expression owner across all final
	// declarations. Their union admits only successful handshake statuses, while
	// SelectFinal below retains the actual selected response for its bindings.
	statusResponse := successfulStatusResponse(responses)
	b.WriteString("transformErr := new(error)\nvar status int\n")
	b.WriteString(buildOutgoingStatusPlan(statusResponse, first.Carrier, context, "value", "status").Code)
	b.WriteString("if *transformErr != nil {\nreturn loomhttp.SSEHandshake{}, *transformErr\n}\n")
	fmt.Fprintf(&b, "selected, ok := loomhttp.SelectFinal([]loomhttp.StatusSelector{%s}, status)\nif !ok {\nreturn loomhttp.SSEHandshake{}, &loomhttp.StatusViolationError{Code: status}\n}\nswitch selected {\n", finalSelectorValues(endpoint))
	for _, response := range responses {
		fmt.Fprintf(&b, "case %#v:\n", responseContractSelectorValue(response.Selector))
		b.WriteString(buildOutgoingHeaderPlan(response, response.Carrier, outgoingHeaderContext(data, response.Carrier), "value", "status", "headers", "").Code)
		b.WriteString(buildOutgoingCookiePlan(response, response.Carrier, outgoingHeaderContext(data, response.Carrier), "value", "status", "headers", "ctx").Code)
		b.WriteString("if *transformErr != nil {\nreturn loomhttp.SSEHandshake{}, *transformErr\n}\nreturn loomhttp.SSEHandshake{Status: status, Header: headers}, nil\n")
	}
	b.WriteString("default:\nreturn loomhttp.SSEHandshake{}, &loomhttp.StatusViolationError{Code: status, Selector: selected}\n}\n}\n")
	return b.String()
}

func sseHandshakeClientSections(endpoint *EndpointData, data *ServiceData) []codegen.Section {
	if endpoint.SSE == nil || endpoint.SSE.HandshakeRef == "" {
		return nil
	}
	return []codegen.Section{codegen.NewJenniferSection("sse-handshake-decoder", func(statement *jen.Statement) {
		var b strings.Builder
		fmt.Fprintf(&b, "// %s reconstructs the selected native metadata without reading the stream body.\nfunc %s(selected loomhttp.StatusSelector, resp *http.Response) (%s, error) {\nswitch selected {\n", endpoint.SSE.DecodeHandshake, endpoint.SSE.DecodeHandshake, endpoint.SSE.HandshakeRef)
		for _, response := range endpoint.Result.Responses {
			status, headers, body := responseMetadataDecode(response, endpoint, data, "resp.Cookies()")
			fmt.Fprintf(&b, "case %#v:\nreturn func(%s int, %s http.Header) (%s, error) {\n%s\n}(resp.StatusCode, resp.Header)\n", responseContractSelectorValue(response.Selector), status, headers, endpoint.SSE.HandshakeRef, body)
		}
		b.WriteString("default:\nreturn nil, &loomhttp.StatusViolationError{Code: resp.StatusCode, Selector: selected}\n}\n}\n")
		statement.Add(codegen.Expr(b.String()))
	})}
}
