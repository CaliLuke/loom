package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
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
	return fields
}

func quotedValues(values []string) []jen.Code {
	quoted := make([]jen.Code, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, jen.Lit(value))
	}
	return quoted
}
