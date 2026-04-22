package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func writeWebSocketRequestValidation(g *jen.Group) {
	g.If(jen.Id("req").Dot("JSONRPC").Op("!=").Lit("2.0")).Block(
		jen.If(jen.Id("req").Dot("HasID")).Block(
			jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"), jen.Lit("Invalid request"), jen.Nil())),
		),
		jen.Return(jen.Nil()),
	)
	g.Line()
	g.If(jen.Id("req").Dot("Method").Op("==").Lit("")).Block(
		jen.If(jen.Id("req").Dot("HasID")).Block(
			jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"), jen.Lit("Invalid request"), jen.Nil())),
		),
		jen.Return(jen.Nil()),
	)
	g.Line()
}

//nolint:maintidx // Generated websocket request dispatch is intentionally centralized.
func writeWebSocketRequestCase(g *jen.Group, ed *httpcodegen.EndpointData) {
	if ed.Method.ServerStream != nil && (ed.Method.ServerStream.Kind == expr.ServerStreamKind || ed.Method.ServerStream.Kind == expr.BidirectionalStreamKind) {
		g.Case(jen.Lit(ed.Method.Name)).BlockFunc(func(cg *jen.Group) {
			if ed.Payload != nil && ed.Payload.Ref != "" {
				cg.List(jen.Id("payload"), jen.Err()).Op(":=").Id("s").Dot(lowerInitial(ed.Method.VarName)).Call(jen.Id("ctx"), jen.Id("s").Dot("r"), jen.Id("req"))
			} else {
				cg.List(jen.Id("_"), jen.Err()).Op(":=").Id("s").Dot(lowerInitial(ed.Method.VarName)).Call(jen.Id("ctx"), jen.Id("s").Dot("r"), jen.Id("req"))
			}
			cg.If(jen.Err().Op("!=").Nil()).Block(
				jen.If(jen.Id("req").Dot("HasID")).Block(
					jen.If(
						jen.Id("sendErr").Op(":=").Id("s").Dot("SendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Err()),
						jen.Id("sendErr").Op("!=").Nil(),
					).Block(
						jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send error response: %w"), jen.Id("sendErr"))),
					),
					jen.Return(jen.Nil()),
				),
				jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for "+ed.Method.Name+": %w"), jen.Err())),
			)
			cg.Id("streamWrapper").Op(":=").Op("&").Id(lowerInitial(ed.Method.VarName) + "StreamWrapper").Values(jen.Dict{
				jen.Id("stream"):    jen.Id("s"),
				jen.Id("requestID"): jen.Id("req").Dot("ID"),
			})
			fields := jen.Dict{
				jen.Id("Stream"): jen.Id("streamWrapper"),
			}
			if ed.Payload != nil && ed.Payload.Ref != "" {
				fields[jen.Id("Payload")] = jen.Id("payload").Assert(codegen.TypeRef(ed.Payload.Ref))
			}
			cg.Id("endpointInput").Op(":=").Op("&").Qual(ed.ServicePkgName, ed.Method.ServerStream.EndpointStruct).Values(fields)
			cg.If(
				jen.List(jen.Id("_"), jen.Err()).Op(":=").Id("s").Dot(lowerInitial(ed.Method.VarName)+"Endpoint").Call(jen.Id("ctx"), jen.Id("endpointInput")),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.If(jen.Id("req").Dot("HasID")).Block(
					jen.If(
						jen.Id("sendErr").Op(":=").Id("streamWrapper").Dot("SendError").Call(jen.Id("ctx"), jen.Err()),
						jen.Id("sendErr").Op("!=").Nil(),
					).Block(
						jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send error response: %w"), jen.Id("sendErr"))),
					),
					jen.Return(jen.Nil()),
				),
				jen.Return(jen.Nil()),
			)
			cg.Return(jen.Nil())
		})
		return
	}
	g.Case(jen.Lit(ed.Method.Name)).BlockFunc(func(cg *jen.Group) {
		cg.List(jen.Id("res"), jen.Err()).Op(":=").Id("s").Dot(lowerInitial(ed.Method.VarName)).Call(jen.Id("ctx"), jen.Id("s").Dot("r"), jen.Id("req"))
		cg.If(jen.Err().Op("!=").Nil()).Block(
			jen.If(jen.Id("req").Dot("HasID")).Block(
				jen.If(
					jen.Id("sendErr").Op(":=").Id("s").Dot("SendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Err()),
					jen.Id("sendErr").Op("!=").Nil(),
				).Block(
					jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send error response: %w"), jen.Id("sendErr"))),
				),
			),
			jen.Return(jen.Nil()),
		)
		cg.If(jen.Id("req").Dot("HasID")).Block(
			jen.If(jen.Id("res").Op("==").Nil()).Block(
				jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError"), jen.Lit("Internal error"), jen.Nil())),
			),
			jen.If(
				jen.List(jen.Id("r"), jen.Id("ok")).Op(":=").Id("res").Assert(jen.Op("*").Qual(ed.ServicePkgName, ed.Method.VarName+"Result")),
				jen.Id("ok"),
			).Block(
				jen.If(
					jen.Err().Op(":=").Id("s").Dot("Send"+ed.Method.VarName+"Response").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Id("r")),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("send response error for "+ed.Method.Name+": %w"), jen.Err())),
				),
			).Else().Block(
				jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError"), jen.Lit("Internal error"), jen.Nil())),
			),
		)
		cg.Return(jen.Nil())
	})
}
