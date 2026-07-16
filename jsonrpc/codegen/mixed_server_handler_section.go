package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

//nolint:maintidx // Mixed HTTP/SSE negotiation is intentionally centralized here.
func jsonrpcMixedServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-mixed-server-handler", func(stmt *jen.Statement) {
		stmt.Comment("serveHTTP handles mixed HTTP/SSE requests before server middleware.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("serveHTTP").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			BlockFunc(func(g *jen.Group) {
				g.Switch(jen.Id("r").Dot("Method")).BlockFunc(func(sg *jen.Group) {
					sg.Case(jen.Qual("net/http", "MethodGet")).BlockFunc(func(getg *jen.Group) {
						getg.Id("req").Op(":=").Add(codegen.Expr(`&jsonrpc.RawRequest{JSONRPC: "2.0", Method: "events/stream"}`))
						getg.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(dispatch *jen.Group) {
							for _, endpoint := range data.Endpoints {
								if endpoint.SSE == nil || endpoint.Method.Name != "events/stream" {
									continue
								}
								dispatch.Case(jen.Lit(endpoint.Method.Name)).Block(
									jen.If(
										jen.Err().Op(":=").Id("s").Dot(endpoint.Method.VarName).Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Id("req"), jen.Id("w")),
										jen.Err().Op("!=").Nil(),
									).Block(
										jen.Id("s").Dot("errhandler").Call(
											jen.Id("r").Dot("Context").Call(),
											jen.Id("w"),
											jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for "+endpoint.Method.Name+": %w"), jen.Err()),
										),
									),
									jen.Return(),
								)
							}
							dispatch.Default().Block(
								jen.Qual("net/http", "NotFound").Call(jen.Id("w"), jen.Id("r")),
								jen.Return(),
							)
						})
					})
					sg.Case(jen.Qual("net/http", "MethodPost")).BlockFunc(func(postg *jen.Group) {
						postg.Id("accept").Op(":=").Id("r").Dot("Header").Dot("Get").Call(jen.Lit("Accept"))
						postg.If(
							jen.Op("!").Qual("strings", "Contains").Call(jen.Id("accept"), jen.Lit("text/event-stream")),
						).Block(
							jen.Id("s").Dot("handleHTTP").Call(jen.Id("w"), jen.Id("r")),
							jen.Return(),
						)
						postg.Line()
						postg.Id("reader").Op(":=").Qual("bufio", "NewReader").Call(jen.Id("r").Dot("Body"))
						postg.Const().Id("maxNegotiationWhitespace").Op("=").Lit(4096)
						postg.Var().Id("first").Byte()
						postg.Id("sniffed").Op(":=").Lit(0)
						postg.For(jen.Id("sniffed").Op("<").Id("maxNegotiationWhitespace")).Block(
							jen.List(jen.Id("peek"), jen.Err()).Op(":=").Id("reader").Dot("Peek").Call(jen.Lit(1)),
							jen.If(
								jen.Err().Op("!=").Nil().Op("&&").Err().Op("!=").Qual("io", "EOF"),
							).Block(
								jen.Id("s").Dot("errhandler").Call(
									jen.Id("r").Dot("Context").Call(),
									jen.Id("w"),
									jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to read request body: %w"), jen.Err()),
								),
								jen.Return(),
							),
							jen.If(jen.Len(jen.Id("peek")).Op("==").Lit(0)).Block(
								jen.Break(),
							),
							jen.Id("first").Op("=").Id("peek").Index(jen.Lit(0)),
							jen.If(
								jen.Id("first").Op("!=").LitByte(' ').
									Op("&&").Id("first").Op("!=").LitByte('\t').
									Op("&&").Id("first").Op("!=").LitByte('\r').
									Op("&&").Id("first").Op("!=").LitByte('\n'),
							).Block(
								jen.Break(),
							),
							jen.If(
								jen.List(jen.Id("_"), jen.Err()).Op(":=").Id("reader").Dot("Discard").Call(jen.Lit(1)),
								jen.Err().Op("!=").Nil(),
							).Block(
								jen.Id("s").Dot("errhandler").Call(
									jen.Id("r").Dot("Context").Call(),
									jen.Id("w"),
									jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to read request body: %w"), jen.Err()),
								),
								jen.Return(),
							),
							jen.Id("sniffed").Op("++"),
						)
						postg.Line()
						postg.Id("r").Dot("Body").Op("=").Struct(
							jen.Qual("io", "Reader"),
							jen.Qual("io", "Closer"),
						).Values(jen.Dict{
							jen.Id("Reader"): jen.Id("reader"),
							jen.Id("Closer"): jen.Id("r").Dot("Body"),
						})
						postg.Line()
						postg.If(
							jen.Id("first").Op("==").Lit(0).Op("||").Id("first").Op("==").LitByte('[').Op("||").Id("sniffed").Op(">=").Id("maxNegotiationWhitespace"),
						).Block(
							jen.Id("s").Dot("handleHTTP").Call(jen.Id("w"), jen.Id("r")),
							jen.Return(),
						)
						postg.Line()
						postg.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest")
						postg.If(
							jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("req")),
							jen.Err().Op("!=").Nil(),
						).Block(
							jen.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(
								jen.Nil(),
								jen.Qual("github.com/CaliLuke/loom/jsonrpc", "ParseError"),
								jen.Lit("Parse error"),
								jen.Nil(),
							),
							jen.If(
								jen.Id("encErr").Op(":=").Id("s").Dot("encoder").Call(jen.Id("r").Dot("Context").Call(), jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
								jen.Id("encErr").Op("!=").Nil(),
							).Block(
								jen.Id("s").Dot("errhandler").Call(
									jen.Id("r").Dot("Context").Call(),
									jen.Id("w"),
									jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode parse error response: %w"), jen.Id("encErr")),
								),
							),
							jen.Return(),
						)
						postg.Line()
						postg.If(jen.Id("req").Dot("Invalid")).Block(
							jen.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
							jen.Return(),
						)
						postg.Line()
						postg.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(dispatch *jen.Group) {
							for _, endpoint := range data.Endpoints {
								if endpoint.SSE == nil {
									continue
								}
								dispatch.Case(jen.Lit(endpoint.Method.Name)).Block(
									jen.If(
										jen.Err().Op(":=").Id("s").Dot(endpoint.Method.VarName).Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
										jen.Err().Op("!=").Nil(),
									).Block(
										jen.Id("s").Dot("errhandler").Call(
											jen.Id("r").Dot("Context").Call(),
											jen.Id("w"),
											jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for "+endpoint.Method.Name+": %w"), jen.Err()),
										),
									),
								)
							}
							dispatch.Default().Block(
								jen.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
							)
						})
					})
					sg.Default().Block(
						jen.Qual("net/http", "NotFound").Call(jen.Id("w"), jen.Id("r")),
					)
				})
			})
		stmt.Line()
	})
}
