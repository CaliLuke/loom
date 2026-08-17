package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcSSEServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-sse-server-handler", func(stmt *jen.Statement) {
		stmt.Comment("handleSSE delegates JSON-RPC SSE lifecycle to the transport runtime.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("handleSSE").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			Block(
				jen.Id("jsonrpc.ServeSSE").Call(jen.Id("w"), jen.Id("r"), jen.Id("s").Dot("sseHandlerSpec").Call()),
			)
		stmt.Line()
		addJSONRPCSSEAdapterSection(stmt, data)
	})
}

func jsonrpcSSEAdapterSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-sse-server-adapters", func(stmt *jen.Statement) {
		addJSONRPCSSEAdapterSection(stmt, data)
	})
}

func addJSONRPCSSEAdapterSection(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	addJSONRPCSSESpecMethod(stmt, data)
	addJSONRPCSSEDispatchMethod(stmt, data)
	addJSONRPCSSEErrorMethod(stmt, data)
}

func addJSONRPCSSESpecMethod(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	stmt.Comment("sseHandlerSpec returns the generated adapters used by the JSON-RPC SSE runtime.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("sseHandlerSpec").Params().Add(codegen.TypeRef("jsonrpc.SSEHandlerSpec")).
		Block(
			jen.Return(codegen.TypeRef("jsonrpc.SSEHandlerSpec").Values(jen.Dict{
				jen.Id("Service"):       jen.Lit(data.Service.Name),
				jen.Id("Decoder"):       jen.Id("s").Dot("decoder"),
				jen.Id("Dispatch"):      jen.Id("s").Dot("dispatchSSE"),
				jen.Id("SendError"):     jen.Id("s").Dot("sendSSEError"),
				jen.Id("HandleFailure"): jen.Id("s").Dot("errhandler"),
			})),
		)
	stmt.Line()
}

func addJSONRPCSSEDispatchMethod(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	stmt.Comment("dispatchSSE calls the typed adapter for a JSON-RPC SSE method.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("dispatchSSE").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
			jen.Id("req").Op("*").Add(codegen.TypeRef("jsonrpc.RawRequest")),
			jen.Id("w").Qual("net/http", "ResponseWriter"),
		).
		Params(jen.Bool(), jen.Bool(), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(cases *jen.Group) {
				for _, endpoint := range data.Endpoints {
					if endpoint.SSE == nil {
						continue
					}
					unary := endpoint.Method.ServerStream == nil
					cases.Case(jen.Lit(endpoint.Method.Name)).Block(
						jen.Return(
							jen.True(),
							jen.Lit(unary),
							jen.Id("s").Dot(endpoint.Method.VarName).Call(jen.Id("ctx"), jen.Id("r"), jen.Id("req"), jen.Id("w")),
						),
					)
				}
			})
			g.Return(jen.False(), jen.False(), jen.Nil())
		})
	stmt.Line()
}

func addJSONRPCSSEErrorMethod(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	streamName := ""
	for _, endpoint := range data.Endpoints {
		if endpoint.SSE != nil {
			streamName = endpoint.SSE.StructName
			break
		}
	}
	stmt.Comment("sendSSEError writes one JSON-RPC error as an SSE message event.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("sendSSEError").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("id").Any(),
			jen.Id("code").Add(codegen.TypeRef("jsonrpc.Code")),
			jen.Id("message").String(),
			jen.Id("data").Any(),
		).
		Error().
		Block(
			jen.Id("stream").Op(":=").Op("&").Id(streamName).Values(jen.Dict{
				jen.Id("w"):       jen.Id("w"),
				jen.Id("r"):       jen.Id("r"),
				jen.Id("writer"):  newJSONRPCSSEWriter(),
				jen.Id("encoder"): jen.Id("s").Dot("encoder"),
			}),
			jen.Return(jen.Id("stream").Dot("sendError").Call(
				jen.Id("ctx"), jen.Id("id"), jen.Id("code"), jen.Id("message"), jen.Id("data"),
			)),
		)
	stmt.Line()
}

func newJSONRPCSSEWriter() jen.Code {
	return jen.Id("loomhttp.NewSSEStreamWriter").Call(
		jen.Id("w"),
		jen.Id("r").Dot("Context").Call(),
		jen.Id("loomtransport.TransportJSONRPC"),
		jen.Id("s").Dot("streamWritePolicy"),
	)
}
