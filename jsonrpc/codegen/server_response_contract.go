package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/jsonrpc/codegen/internal/transportir"
)

func serverResponseContractSection(endpoint *responseContractEndpointData) codegen.Section {
	if len(endpoint.Cases) == 0 {
		return nil
	}
	return codegen.NewJenniferSection("server-response-contract", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf(
			"%s returns the declared JSON-RPC wire-response contracts for %s. Callers remain responsible for exercising the application scenarios that produce each response.",
			endpoint.CasesInit,
			endpoint.MethodName,
		))
		stmt.Func().Id(endpoint.CasesInit).Params().Index().Add(codegen.TypeRef("jsonrpc.ResponseContractCase")).Block(
			jen.Return(jen.Index().Add(codegen.TypeRef("jsonrpc.ResponseContractCase")).ValuesFunc(func(group *jen.Group) {
				for _, contractCase := range endpoint.Cases {
					group.Values(responseContractCaseFields(contractCase)...)
				}
			})),
		)
		stmt.Line()
	})
}

func serverResponseContractsSection(data *responseContractServiceData) codegen.Section {
	caseCount := 0
	for _, endpoint := range data.Endpoints {
		caseCount += len(endpoint.Cases)
	}
	if caseCount == 0 {
		return nil
	}
	return codegen.NewJenniferSection("server-response-contracts", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "ResponseContractCases returns every supported declared JSON-RPC wire-response contract for this service. The returned slice is owned by the caller.")
		stmt.Func().Id("ResponseContractCases").Params().Index().Add(codegen.TypeRef("jsonrpc.ResponseContractCase")).BlockFunc(func(group *jen.Group) {
			group.Id("cases").Op(":=").Make(jen.Index().Add(codegen.TypeRef("jsonrpc.ResponseContractCase")), jen.Lit(0), jen.Lit(caseCount))
			for _, endpoint := range data.Endpoints {
				if len(endpoint.Cases) == 0 {
					continue
				}
				group.Id("cases").Op("=").Append(jen.Id("cases"), jen.Id(endpoint.CasesInit).Call().Op("..."))
			}
			group.Return(jen.Id("cases"))
		})
		stmt.Line()
	})
}

func responseContractCaseFields(contractCase *responseContractCaseData) []jen.Code {
	fields := []jen.Code{
		jen.Id("ID").Op(":").Lit(contractCase.ID),
		jen.Id("Kind").Op(":").Add(codegen.Expr(responseContractKindRef(contractCase.Kind))),
	}
	if contractCase.ResultType != "" {
		fields = append(fields, jen.Id("ResultType").Op(":").Lit(contractCase.ResultType))
	}
	if contractCase.HasResult {
		fields = append(fields, jen.Id("HasResult").Op(":").True())
	}
	if contractCase.Kind == transportir.ResponseContractError {
		fields = append(fields, jen.Id("ErrorCode").Op(":").Lit(contractCase.ErrorCode))
	}
	if contractCase.ErrorName != "" {
		fields = append(fields, jen.Id("ErrorName").Op(":").Lit(contractCase.ErrorName))
	}
	if contractCase.ErrorDataType != "" {
		fields = append(fields, jen.Id("ErrorDataType").Op(":").Lit(contractCase.ErrorDataType))
	}
	if contractCase.Stream != nil {
		fields = append(fields, jen.Id("Stream").Op(":").Op("&").Add(codegen.TypeRef("jsonrpc.StreamingResponseContract")).Values(jen.Dict{
			jen.Id("Transport"): jen.Lit(contractCase.Stream.Transport),
			jen.Id("Terminal"):  jen.Lit(contractCase.Stream.Terminal),
		}))
	}
	return fields
}

func responseContractKindRef(kind transportir.ResponseContractCaseKind) string {
	switch kind {
	case transportir.ResponseContractError:
		return "jsonrpc.ResponseContractError"
	case transportir.ResponseContractNotification:
		return "jsonrpc.ResponseContractNotification"
	default:
		return "jsonrpc.ResponseContractSuccess"
	}
}
