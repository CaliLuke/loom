package codegen

import "github.com/dave/jennifer/jen"

func addJSONRPCBatchWriterSection(stmt *jen.Statement) {
	stmt.Comment("batchWriter is a helper type that implements http.ResponseWriter for writing multiple JSON-RPC responses").Line()
	stmt.Type().Id("batchWriter").Struct(
		jen.Qual("io", "Writer"),
		jen.Id("header").Qual("net/http", "Header"),
		jen.Id("statusCode").Int(),
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
		Params(jen.Id("statusCode").Int()).
		Block(
			jen.If(jen.Id("rb").Dot("written")).Block(
				jen.Return(),
			),
			jen.Id("rb").Dot("statusCode").Op("=").Id("statusCode"),
		)
	stmt.Line()
	stmt.Func().Params(jen.Id("rb").Op("*").Id("batchWriter")).
		Id("Write").
		Params(jen.Id("data").Index().Byte()).
		Params(jen.Int(), jen.Error()).
		Block(
			jen.If(jen.Op("!").Id("rb").Dot("written")).Block(
				jen.Id("rb").Dot("written").Op("=").True(),
				jen.Id("rb").Dot("Writer").Dot("Write").Call(jen.Index().Byte().Values(jen.LitByte('['))),
			).Else().Block(
				jen.Id("rb").Dot("Writer").Dot("Write").Call(jen.Index().Byte().Values(jen.LitByte(','))),
			),
			jen.Return(jen.Id("rb").Dot("Writer").Dot("Write").Call(jen.Id("data"))),
		)
	stmt.Line()
}
