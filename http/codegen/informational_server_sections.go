package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func informationalServerSections(endpoint *EndpointData, data *ServiceData) []codegen.Section {
	sections := make([]codegen.Section, 0, len(endpoint.Informational))
	for _, info := range endpoint.Informational {
		carrier := info.Response.Carrier
		pkg := data.Service.LocationPackageName(codegen.UserTypeLocation(carrier.Type))
		ref := data.Service.Scope.GoFullTypeRef(carrier, pkg)
		context := outgoingHeaderContext(data, carrier)
		vars := newResponseVarScope(data)
		value := vars.names.Unique("value")
		exchange := vars.names.Unique("exchange")
		status := vars.names.Unique("status")
		headers := vars.names.Unique("headers")
		failure := vars.names.Unique("conversionError")
		statusPlan := buildOutgoingStatusPlan(info.Response, carrier, context, value, status)
		headerPlan := buildOutgoingHeaderPlan(info.Response, carrier, context, value, status, headers, "")
		body := fmt.Sprintf("%s := loomhttp.ExchangeFromContext(ctx)\nif %s == nil {\nreturn loomhttp.ErrInformationalClosed\n}\nvar %s int\nvar %s error\ntransformErr := &%s\n%s\n%s\nif %s != nil {\nreturn %s\n}\nreturn %s.Send(%s, %s)\n", exchange, exchange, status, failure, failure, statusPlan.Code, headerPlan.Code, failure, failure, exchange, status, headers)
		sections = append(sections, codegen.NewJenniferSection("informational-server", func(stmt *jen.Statement) {
			description := info.Send + " sends the declared " + info.Name + " informational response for " + endpoint.Method.Name + " before the final response begins."
			if len(info.Response.Domain) == 0 {
				description = info.Send + " rejects the observation-only " + info.Name + " declaration. Its server emission domain is empty."
			}
			codegen.Doc(stmt, description)
			stmt.Func().Id(info.Send).Params(
				jen.Id("ctx").Add(codegen.TypeRef("context.Context")),
				jen.Id(value).Add(codegen.TypeRef(ref)),
			).Error().BlockFunc(func(group *jen.Group) {
				appendHTTPRawBlock(group, body)
			})
			stmt.Line()
		}))
	}
	return sections
}
