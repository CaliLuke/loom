package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

//nolint:maintidx // Generator entrypoint intentionally keeps SSE routing branches together.
func jsonrpcSSEServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-sse-server-handler", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "SSEStream"

		stmt.Comment("handleSSE handles JSON-RPC SSE requests by dispatching to the appropriate method.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("handleSSE").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			BlockFunc(func(g *jen.Group) {
				g.Id("ctx").Op(":=").Id("r").Dot("Context").Call()
				g.Line()
				g.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest")
				g.If(jen.Id("r").Dot("Method").Op("==").Qual("net/http", "MethodGet")).Block(
					jen.Id("req").Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest").Values(jen.Dict{
						jen.Id("JSONRPC"): jen.Lit("2.0"),
						jen.Id("Method"):  jen.Lit("events/stream"),
					}),
				).Else().If(
					jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("req")),
					jen.Err().Op("!=").Nil(),
				).BlockFunc(func(eg *jen.Group) {
					writeSSEErrorStreamInit(eg, streamName)
					eg.Id("stream").Dot("sendError").Call(
						jen.Id("ctx"),
						jen.Nil(),
						jen.Qual("github.com/CaliLuke/loom/jsonrpc", "ParseError"),
						jen.Lit("Parse error"),
						jen.Nil(),
					)
					eg.Return()
				})
				g.Line()
				writeSSERequestValidation(g, streamName)
				g.Var().Id("handler").Func().Params(
					jen.Qual("context", "Context"),
					jen.Op("*").Qual("net/http", "Request"),
					jen.Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
					jen.Qual("net/http", "ResponseWriter"),
				).Error()
				g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(sg *jen.Group) {
					for _, endpoint := range data.Endpoints {
						if endpoint.SSE == nil {
							continue
						}
						sg.Case(jen.Lit(endpoint.Method.Name)).Block(
							jen.Id("handler").Op("=").Id("s").Dot(endpoint.Method.VarName),
						)
					}
					sg.Default().BlockFunc(func(dg *jen.Group) {
						dg.If(
							jen.Id("req").Dot("ID").Op("==").Nil().Op("||").Id("req").Dot("ID").Op("==").Lit(""),
						).Block(
							jen.Id("w").Dot("WriteHeader").Call(jen.Qual("net/http", "StatusNoContent")),
							jen.Return(),
						)
						dg.Id("stream").Op(":=").Op("&").Id(streamName).Values(jen.Dict{
							jen.Id("w"):       jen.Id("w"),
							jen.Id("r"):       jen.Id("r"),
							jen.Id("encoder"): jen.Id("s").Dot("encoder"),
							jen.Id("decoder"): jen.Id("s").Dot("decoder"),
						})
						dg.Id("stream").Dot("sendError").Call(
							jen.Id("ctx"),
							jen.Id("req").Dot("ID"),
							jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"),
							jen.Lit("Method not found"),
							jen.Nil(),
						)
						dg.Return()
					})
				})
				g.Line()
				g.If(
					jen.Err().Op(":=").Id("handler").Call(jen.Id("ctx"), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.Id("s").Dot("errhandler").Call(
						jen.Id("ctx"),
						jen.Id("w"),
						jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for %s: %w"), jen.Id("req").Dot("Method"), jen.Err()),
					),
					jen.Return(),
				)
				g.Line()
				g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(sg *jen.Group) {
					for _, endpoint := range data.Endpoints {
						if endpoint.SSE == nil || endpoint.Method.ServerStream != nil {
							continue
						}
						sg.Case(jen.Lit(endpoint.Method.Name)).Block(
							jen.If(jen.Id("req").Dot("ID").Op("==").Nil()).Block(
								jen.Id("w").Dot("WriteHeader").Call(jen.Qual("net/http", "StatusNoContent")),
							),
						)
					}
				})
			})
	})
}

func writeSSEErrorStreamInit(g *jen.Group, streamName string) {
	g.Id("stream").Op(":=").Op("&").Id(streamName).Values(jen.Dict{
		jen.Id("w"):       jen.Id("w"),
		jen.Id("r"):       jen.Id("r"),
		jen.Id("encoder"): jen.Id("s").Dot("encoder"),
		jen.Id("decoder"): jen.Id("s").Dot("decoder"),
	})
}

func writeSSEValidationError(g *jen.Group, streamName, message string) {
	g.If(
		jen.Id("req").Dot("ID").Op("==").Nil().Op("||").Id("req").Dot("ID").Op("==").Lit(""),
	).Block(
		jen.Id("w").Dot("WriteHeader").Call(jen.Qual("net/http", "StatusNoContent")),
		jen.Return(),
	)
	writeSSEErrorStreamInit(g, streamName)
	g.Id("stream").Dot("sendError").Call(
		jen.Id("ctx"),
		jen.Id("req").Dot("ID"),
		jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"),
		jen.Lit(message),
		jen.Nil(),
	)
	g.Return()
}

func writeSSERequestValidation(g *jen.Group, streamName string) {
	g.If(jen.Id("req").Dot("JSONRPC").Op("!=").Lit("2.0")).BlockFunc(func(eg *jen.Group) {
		writeSSEValidationError(eg, streamName, "Invalid request")
	})
	g.Line()
	g.If(jen.Id("req").Dot("Method").Op("==").Lit("")).BlockFunc(func(eg *jen.Group) {
		writeSSEValidationError(eg, streamName, "Invalid request")
	})
	g.Line()
}
