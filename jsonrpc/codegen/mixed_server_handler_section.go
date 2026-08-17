package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcMixedServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-mixed-server-handler", func(stmt *jen.Statement) {
		stmt.Comment("serveHTTP delegates mixed JSON-RPC HTTP/SSE negotiation to the transport runtime.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("serveHTTP").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			Block(
				jen.Id("jsonrpc.ServeMixed").Call(
					jen.Id("w"),
					jen.Id("r"),
					codegen.TypeRef("jsonrpc.MixedHandlerSpec").Values(jen.Dict{
						jen.Id("HTTP"):        jen.Id("s").Dot("httpHandlerSpec").Call(),
						jen.Id("SSE"):         jen.Id("s").Dot("sseHandlerSpec").Call(),
						jen.Id("SupportsGET"): jen.Lit(hasEventsStreamSSEEndpoint(data)),
					}),
				),
			)
		stmt.Line()
	})
}

func hasEventsStreamSSEEndpoint(data *httpcodegen.ServiceData) bool {
	for _, endpoint := range data.Endpoints {
		if endpoint.SSE != nil && endpoint.Method.Name == "events/stream" {
			return true
		}
	}
	return false
}
