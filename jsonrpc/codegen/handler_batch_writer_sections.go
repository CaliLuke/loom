package codegen

import "github.com/dave/jennifer/jen"

func addJSONRPCBatchWriterSection(stmt *jen.Statement) {
	stmt.Comment("batchWriter is a helper type that implements http.ResponseWriter for writing multiple JSON-RPC responses").Line()
	stmt.Type().Id("batchWriter").Struct(
		jen.Qual("io", "Writer"),
		jen.Id("header").Qual("net/http", "Header"),
		jen.Id("written").Bool(),
	)
	stmt.Line()
	stmt.Func().Params(jen.Id("rb").Op("*").Id("batchWriter")).
		Id("Header").
		Params().
		Qual("net/http", "Header").
		Block(
			jen.If(jen.Id("rb").Dot("header").Op("==").Nil()).Block(
				jen.Id("rb").Dot("header").Op("=").Make(jen.Qual("net/http", "Header")),
			),
			jen.Return(jen.Id("rb").Dot("header")),
		)
	stmt.Line()
	stmt.Func().Params(jen.Id("rb").Op("*").Id("batchWriter")).
		Id("WriteHeader").
		Params(jen.Id("_").Int()).
		Block(
			jen.Comment("JSON-RPC batch items do not control the outer HTTP status."),
		)
	stmt.Line()
	stmt.Func().Params(jen.Id("rb").Op("*").Id("batchWriter")).
		Id("Write").
		Params(jen.Id("data").Index().Byte()).
		Params(jen.Int(), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Id("delimiter").Op(":=").LitByte(',')
			g.If(jen.Op("!").Id("rb").Dot("written")).Block(
				jen.Id("delimiter").Op("=").LitByte('['),
			)
			g.If(
				jen.List(jen.Id("_"), jen.Id("err")).Op(":=").Id("rb").Dot("Writer").Dot("Write").Call(jen.Index().Byte().Values(jen.Id("delimiter"))),
				jen.Id("err").Op("!=").Nil(),
			).Block(
				jen.Return(jen.Lit(0), jen.Qual("fmt", "Errorf").Call(jen.Lit("write JSON-RPC batch delimiter: %w"), jen.Id("err"))),
			)
			g.Id("rb").Dot("written").Op("=").True()
			g.Return(jen.Id("rb").Dot("Writer").Dot("Write").Call(jen.Id("data")))
		})
	stmt.Line()
}
