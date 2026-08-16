package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func serverResponseContractSection(endpoint *EndpointData) codegen.Section {
	if len(endpoint.ResponseContractCases) == 0 {
		return nil
	}
	return codegen.NewJenniferSection("server-response-contract", func(stmt *jen.Statement) {
		comment := fmt.Sprintf(
			"%s returns the declared HTTP wire-response contracts for %s. Callers remain responsible for exercising the application scenarios that produce each response.",
			endpoint.ResponseContractCasesInit,
			endpoint.Method.Name,
		)
		codegen.Doc(stmt, comment)
		stmt.Func().
			Id(endpoint.ResponseContractCasesInit).
			Params().
			Index().
			Add(codegen.TypeRef("loomhttp.ResponseContractCase")).
			Block(
				jen.Return(jen.Index().Add(codegen.TypeRef("loomhttp.ResponseContractCase")).ValuesFunc(func(group *jen.Group) {
					for _, contractCase := range endpoint.ResponseContractCases {
						group.Values(responseContractCaseFields(contractCase)...)
					}
				})),
			)
		stmt.Line()
	})
}

func serverResponseContractsSection(data *ServiceData) codegen.Section {
	caseCount := 0
	for _, endpoint := range data.Endpoints {
		caseCount += len(endpoint.ResponseContractCases)
	}
	if caseCount == 0 {
		return nil
	}

	return codegen.NewJenniferSection("server-response-contracts", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "ResponseContractCases returns every supported declared HTTP wire-response contract for this service. The returned slice is owned by the caller.")
		stmt.Func().
			Id("ResponseContractCases").
			Params().
			Index().
			Add(codegen.TypeRef("loomhttp.ResponseContractCase")).
			BlockFunc(func(group *jen.Group) {
				group.Id("cases").Op(":=").Make(
					jen.Index().Add(codegen.TypeRef("loomhttp.ResponseContractCase")),
					jen.Lit(0),
					jen.Lit(caseCount),
				)
				for _, endpoint := range data.Endpoints {
					if len(endpoint.ResponseContractCases) == 0 {
						continue
					}
					group.Id("cases").Op("=").Append(
						jen.Id("cases"),
						jen.Id(endpoint.ResponseContractCasesInit).Call().Op("..."),
					)
				}
				group.Return(jen.Id("cases"))
			})
		stmt.Line()
	})
}

func responseContractCaseFields(contractCase *ResponseContractCaseData) []jen.Code {
	kind := "loomhttp.ResponseContractSuccess"
	if contractCase.IsError {
		kind = "loomhttp.ResponseContractError"
	}
	fields := []jen.Code{
		jen.Id("ID").Op(":").Lit(contractCase.ID),
		jen.Id("Kind").Op(":").Add(codegen.Expr(kind)),
		jen.Id("Transport").Op(":").Add(codegen.Expr(responseContractTransportRef(contractCase.Transport))),
		jen.Id("StatusCode").Op(":").Lit(contractCase.StatusCode),
	}
	if contractCase.ErrorName != "" {
		fields = append(fields, jen.Id("ErrorName").Op(":").Lit(contractCase.ErrorName))
	}
	if len(contractCase.ContentTypes) > 0 {
		fields = append(fields, jen.Id("ContentTypes").Op(":").Index().String().Values(quotedValues(contractCase.ContentTypes)...))
	}
	if len(contractCase.RequiredHeaders) > 0 {
		fields = append(fields, jen.Id("RequiredHeaders").Op(":").Index().String().Values(quotedValues(contractCase.RequiredHeaders)...))
	}
	if len(contractCase.RequiredCookies) > 0 {
		fields = append(fields, jen.Id("RequiredCookies").Op(":").Index().String().Values(quotedValues(contractCase.RequiredCookies)...))
	}
	if contractCase.Multipart != nil {
		fields = append(fields, responseContractMultipartField(contractCase.Multipart))
	}
	if contractCase.SSE != nil {
		fields = append(fields, responseContractSSEField(contractCase.SSE))
	}
	if contractCase.WebSocket != nil {
		fields = append(fields, responseContractWebSocketField(contractCase.WebSocket))
	}
	return fields
}

func responseContractMultipartField(contract *MultipartRequestContractData) jen.Code {
	return jen.Id("Multipart").Op(":").Op("&").Add(codegen.TypeRef("loomhttp.MultipartRequestContract")).Values(
		jen.Dict{
			jen.Id("ContentType"): jen.Lit(contract.ContentType),
			jen.Id("Parts"): jen.Index().Add(codegen.TypeRef("loomhttp.MultipartPartContract")).ValuesFunc(func(group *jen.Group) {
				for _, part := range contract.Parts {
					group.Values(jen.Dict{
						jen.Id("Name"):      jen.Lit(part.Name),
						jen.Id("MediaType"): jen.Lit(part.MediaType),
						jen.Id("Required"):  jen.Lit(part.Required),
					})
				}
			}),
		},
	)
}

func responseContractSSEField(contract *SSEResponseContractData) jen.Code {
	return jen.Id("SSE").Op(":").Op("&").Add(codegen.TypeRef("loomhttp.SSEResponseContract")).Values(
		jen.Dict{
			jen.Id("Direction"):         jen.Lit(contract.Direction),
			jen.Id("MessageType"):       jen.Lit(contract.MessageType),
			jen.Id("DataField"):         jen.Lit(contract.DataField),
			jen.Id("DataEncoding"):      jen.Lit(contract.DataEncoding),
			jen.Id("IDField"):           jen.Lit(contract.IDField),
			jen.Id("EventField"):        jen.Lit(contract.EventField),
			jen.Id("RetryField"):        jen.Lit(contract.RetryField),
			jen.Id("IDRequired"):        jen.Lit(contract.IDRequired),
			jen.Id("EventTypeRequired"): jen.Lit(contract.EventTypeRequired),
			jen.Id("EventTypes"):        jen.Index().String().Values(quotedValues(contract.EventTypes)...),
			jen.Id("Terminal"):          jen.Lit(contract.Terminal),
		},
	)
}

func responseContractWebSocketField(contract *WebSocketResponseContractData) jen.Code {
	return jen.Id("WebSocket").Op(":").Op("&").Add(codegen.TypeRef("loomhttp.WebSocketResponseContract")).Values(
		jen.Dict{
			jen.Id("Direction"):           jen.Lit(contract.Direction),
			jen.Id("InboundMessageType"):  jen.Lit(contract.InboundMessageType),
			jen.Id("OutboundMessageType"): jen.Lit(contract.OutboundMessageType),
			jen.Id("HandshakeHeaders"):    jen.Index().String().Values(quotedValues(contract.HandshakeHeaders)...),
			jen.Id("Terminal"):            jen.Lit(contract.Terminal),
		},
	)
}

func responseContractTransportRef(transport string) string {
	switch transport {
	case string(transportir.ResponseContractSSETransport):
		return "loomhttp.ResponseContractSSE"
	case string(transportir.ResponseContractWebSocketTransport):
		return "loomhttp.ResponseContractWebSocket"
	default:
		return "loomhttp.ResponseContractHTTP"
	}
}

func quotedValues(values []string) []jen.Code {
	quoted := make([]jen.Code, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, jen.Lit(value))
	}
	return quoted
}
