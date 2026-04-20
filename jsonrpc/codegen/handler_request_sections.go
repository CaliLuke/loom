package codegen

import (
	"github.com/dave/jennifer/jen"

	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func addJSONRPCHandleSingleSection(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	stmt.Comment("handleSingle handles a single JSON-RPC request.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("handleSingle").
		Params(
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
		).
		Block(
			jen.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
			jen.If(
				jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("req")),
				jen.Err().Op("!=").Nil(),
			).BlockFunc(func(g *jen.Group) {
				writeParseErrorResponse(g, jen.Id("r").Dot("Context").Call())
				g.Return()
			}),
			jen.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
		)
	stmt.Line()
}

func addJSONRPCHandleBatchSection(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	stmt.Comment("handleBatch handles a batch of JSON-RPC requests.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("handleBatch").
		Params(
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
		).
		Block(
			jen.Var().Id("reqs").Index().Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
			jen.If(
				jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("reqs")),
				jen.Err().Op("!=").Nil(),
			).BlockFunc(func(g *jen.Group) {
				writeParseErrorResponse(g, jen.Id("r").Dot("Context").Call())
				g.Return()
			}),
			jen.Id("w").Dot("Header").Call().Dot("Set").Call(jen.Lit("Content-Type"), jen.Lit("application/json")),
			jen.Id("writer").Op(":=").Op("&").Id("batchWriter").Values(jen.Dict{
				jen.Id("Writer"): jen.Id("w"),
			}),
			jen.For(
				jen.List(jen.Id("_"), jen.Id("req")).Op(":=").Range().Id("reqs"),
			).Block(
				jen.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("writer")),
			),
			jen.If(jen.Id("writer").Dot("written")).Block(
				jen.Id("writer").Dot("Writer").Dot("Write").Call(jen.Index().Byte().Values(jen.LitByte(']'))),
			),
		)
	stmt.Line()
}

func addJSONRPCProcessRequestSection(stmt *jen.Statement, data *httpcodegen.ServiceData) {
	stmt.Comment("processRequest processes a single JSON-RPC request.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("processRequest").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
			jen.Id("req").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
			jen.Id("w").Qual("net/http", "ResponseWriter"),
		).
		BlockFunc(func(g *jen.Group) {
			writeJSONRPCProcessRequestBody(g, data.Endpoints)
		})
	stmt.Line()
}

func writeJSONRPCProcessRequestBody(g *jen.Group, endpoints []*httpcodegen.EndpointData) {
	writeJSONRPCInvalidRequestCheck(g, jen.Id("req").Dot("JSONRPC").Op("!=").Lit("2.0"), jen.Lit("Invalid request"))
	writeJSONRPCInvalidRequestCheck(g, jen.Id("req").Dot("Method").Op("==").Lit(""), jen.Lit("Missing method field"))
	g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(sg *jen.Group) {
		writeJSONRPCMethodDispatch(sg, endpoints)
		sg.Default().Block(
			jen.Id("s").Dot("encodeJSONRPCError").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Id("req"),
				jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"),
				jen.Lit("Method not found"),
				jen.Nil(),
			),
		)
	})
}

func writeJSONRPCInvalidRequestCheck(g *jen.Group, condition jen.Code, message jen.Code) {
	g.If(condition).Block(
		jen.Id("s").Dot("encodeJSONRPCError").Call(
			jen.Id("ctx"),
			jen.Id("w"),
			jen.Id("req"),
			jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"),
			message,
			jen.Nil(),
		),
		jen.Return(),
	)
}

func writeParseErrorResponse(g *jen.Group, ctx jen.Code) {
	g.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(
		jen.Nil(),
		jen.Qual("github.com/CaliLuke/loom/jsonrpc", "ParseError"),
		jen.Lit("Parse error"),
		jen.Nil(),
	)
	g.If(
		jen.Id("encErr").Op(":=").Id("s").Dot("encoder").Call(ctx, jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
		jen.Id("encErr").Op("!=").Nil(),
	).Block(
		jen.Id("s").Dot("errhandler").Call(
			ctx,
			jen.Id("w"),
			jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode parse error response: %w"), jen.Id("encErr")),
		),
	)
}

func writeJSONRPCMethodDispatch(g *jen.Group, endpoints []*httpcodegen.EndpointData) {
	for _, endpoint := range endpoints {
		g.Case(jen.Lit(endpoint.Method.Name)).Block(
			jen.If(
				jen.Err().Op(":=").Id("s").Dot(endpoint.Method.VarName).Call(jen.Id("ctx"), jen.Id("r"), jen.Id("req"), jen.Id("w")),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Id("s").Dot("errhandler").Call(
					jen.Id("ctx"),
					jen.Id("w"),
					jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for "+endpoint.Method.Name+": %w"), jen.Err()),
				),
			),
		)
	}
}
