package codegen

import (
	"github.com/dave/jennifer/jen"

	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func addJSONRPCServeHTTPSection(stmt *jen.Statement, data *httpcodegen.ServiceData, mixed bool) {
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) || mixed {
		return
	}

	stmt.Comment("ServeHTTP handles JSON-RPC requests.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("ServeHTTP").
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
	stmt.Comment("handleHTTP handles JSON-RPC requests.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
		Id("handleHTTP").
		Params(
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("r").Op("*").Qual("net/http", "Request"),
		).
		BlockFunc(writeBufferedRequestHandling)
	stmt.Line()
}

func writeBufferedRequestHandling(g *jen.Group) {
	g.Comment("Peek at the first byte to determine request type")
	g.Id("bufReader").Op(":=").Qual("bufio", "NewReader").Call(jen.Id("r").Dot("Body"))
	g.List(jen.Id("peek"), jen.Err()).Op(":=").Id("bufReader").Dot("Peek").Call(jen.Lit(1))
	g.If(
		jen.Err().Op("!=").Nil().Op("&&").Err().Op("!=").Qual("io", "EOF"),
	).Block(
		jen.Id("r").Dot("Body").Dot("Close").Call(),
		jen.Id("s").Dot("errhandler").Call(
			jen.Id("r").Dot("Context").Call(),
			jen.Id("w"),
			jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to read request body: %w"), jen.Err()),
		),
		jen.Return(),
	)
	g.Line()
	g.Comment("Wrap the buffered reader with the original closer")
	g.Id("r").Dot("Body").Op("=").Struct(
		jen.Qual("io", "Reader"),
		jen.Qual("io", "Closer"),
	).Values(jen.Dict{
		jen.Id("Reader"): jen.Id("bufReader"),
		jen.Id("Closer"): jen.Id("r").Dot("Body"),
	})
	g.Defer().Func().Params(jen.Id("r").Op("*").Qual("net/http", "Request")).Block(
		jen.If(
			jen.Err().Op(":=").Id("r").Dot("Body").Dot("Close").Call(),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("s").Dot("errhandler").Call(
				jen.Id("r").Dot("Context").Call(),
				jen.Id("w"),
				jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to close request body: %w"), jen.Err()),
			),
		),
	).Call(jen.Id("r"))
	g.Line()
	g.Comment("Route to appropriate handler")
	g.If(
		jen.Len(jen.Id("peek")).Op(">").Lit(0).Op("&&").Id("peek").Index(jen.Lit(0)).Op("==").LitByte('['),
	).Block(
		jen.Id("s").Dot("handleBatch").Call(jen.Id("w"), jen.Id("r")),
		jen.Return(),
	)
	g.Id("s").Dot("handleSingle").Call(jen.Id("w"), jen.Id("r"))
}
