package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// loomtransportRef renders an in-scope reference to a symbol from the
// `loomtransport` package alias.
func loomtransportRef(symbol string) *jen.Statement {
	return jen.Id("loomtransport." + symbol)
}

func addJSONRPCDispatchHTTPSection(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	stmt.Comment("dispatchHTTP calls the typed adapter for a JSON-RPC method.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("dispatchHTTP").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
			jen.Id("req").Op("*").Add(codegen.TypeRef("jsonrpc.RawRequest")),
			jen.Id("w").Qual("net/http", "ResponseWriter"),
		).
		Params(jen.Bool(), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(dispatch *jen.Group) {
				for _, endpoint := range data.Endpoints {
					dispatch.Case(jen.Lit(endpoint.Method.Name)).Block(
						jen.Return(
							jen.True(),
							jen.Id("s").Dot(endpoint.Method.VarName).Call(
								jen.Id("ctx"),
								jen.Id("r"),
								jen.Id("req"),
								jen.Id("w"),
							),
						),
					)
				}
			})
			g.Return(jen.False(), jen.Nil())
		})
	stmt.Line()
}
