package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func addJSONRPCServeHTTPSection(stmt *jen.Statement, data *httpcodegen.ServiceData, mixed bool) {
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) || mixed {
		return
	}

	stmt.Comment("serveHTTP handles JSON-RPC requests before server middleware.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("serveHTTP").
		Params(
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
		).
		Block(
			jen.Id("s").Dot("handleHTTP").Call(jen.Id("w"), jen.Id("r")),
		)
	stmt.Line()
}

func addJSONRPCHandleHTTPSection(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	serviceName := data.Service.Name
	stmt.Comment("handleHTTP handles JSON-RPC requests.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("handleHTTP").
		Params(
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
		).
		Block(
			jen.Id("jsonrpc.ServeHTTP").Call(jen.Id("w"), jen.Id("r"), jen.Id("s").Dot("httpHandlerSpec").Call()),
		)
	stmt.Line()
	stmt.Comment("httpHandlerSpec returns the generated adapters used by the JSON-RPC HTTP runtime.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("httpHandlerSpec").Params().Add(codegen.TypeRef("jsonrpc.HTTPHandlerSpec")).
		Block(
			jen.Return(codegen.TypeRef("jsonrpc.HTTPHandlerSpec").Values(jen.Dict{
				jen.Id("Service"):       jen.Lit(serviceName),
				jen.Id("Decoder"):       jen.Id("s").Dot("decoder"),
				jen.Id("Encoder"):       jen.Id("s").Dot("encoder"),
				jen.Id("Dispatch"):      jen.Id("s").Dot("dispatchHTTP"),
				jen.Id("HandleFailure"): jen.Id("s").Dot("errhandler"),
			})),
		)
	stmt.Line()
}
