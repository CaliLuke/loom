package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func informationalClientSections(endpoint *EndpointData, data *ServiceData) []codegen.Section {
	if endpoint.IsJSONRPC || len(endpoint.Informational) == 0 {
		return nil
	}
	sections := make([]codegen.Section, 0, len(endpoint.Informational)+1)
	var observers strings.Builder
	observers.WriteString("observers := []loomhttp.InformationalObserver{\n")
	for _, info := range endpoint.Informational {
		fmt.Fprintf(&observers, "{Selector: %#v},\n", responseContractSelectorValue(info.Response.Selector))
	}
	observers.WriteString("}\n")
	for index, info := range endpoint.Informational {
		carrier := info.Response.Carrier
		pkg := data.Service.LocationPackageName(codegen.UserTypeLocation(carrier.Type))
		ref := data.Service.Scope.GoFullTypeRef(carrier, pkg)
		callback := fmt.Sprintf("func(context.Context, %s, error) error", ref)
		status, headers, decoded := responseMetadataDecode(info.Response, endpoint, data, "")
		fmt.Fprintf(&observers, "if callback, _ := ctx.Value(%s{}).(%s); callback != nil {\nobservers[%d].Observe = func(ctx context.Context, status int, headers http.Header) error {\nvalue, decodeErr := %s(status, headers)\nreturn callback(ctx, value, decodeErr)\n}\n}\n", info.ObserveKey, callback, index, info.Decode)
		sections = append(sections, codegen.NewJenniferSection("informational-client", func(stmt *jen.Statement) {
			stmt.Type().Id(info.ObserveKey).Struct().Line()
			codegen.Doc(stmt, info.Observe+" registers a typed observer for the "+info.Name+" informational response of "+endpoint.Method.Name+". A nil callback disables observation for this declaration. The callback receives any metadata decoding error; only its returned error aborts the request.")
			stmt.Func().Id(info.Observe).Params(jen.Id("ctx").Add(codegen.TypeRef("context.Context")), jen.Id("fn").Add(codegen.TypeRef(callback))).Add(codegen.TypeRef("context.Context")).Block(jen.Return(jen.Qual("context", "WithValue").Call(jen.Id("ctx"), jen.Id(info.ObserveKey).Values(), jen.Id("fn")))).Line()
			stmt.Func().Id(info.Decode).Params(jen.Id(status).Int(), jen.Id(headers).Add(codegen.TypeRef("http.Header"))).Params(codegen.TypeRef(ref), jen.Error()).BlockFunc(func(group *jen.Group) {
				appendHTTPRawBlock(group, decoded)
			}).Line()
		}))
	}
	observers.WriteString("return observers\n")
	body := observers.String()
	sections = append(sections, codegen.NewJenniferSection("informational-observers", func(stmt *jen.Statement) {
		stmt.Func().Id(endpoint.InformationalObservers).Params(jen.Id("ctx").Add(codegen.TypeRef("context.Context"))).Index().Add(codegen.TypeRef("loomhttp.InformationalObserver")).BlockFunc(func(group *jen.Group) {
			appendHTTPRawBlock(group, body)
		})
	}))
	return sections
}
