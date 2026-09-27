package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// writeViewedStreamResultDecoder decodes a message body with the same projected
// types and view validation as an ordinary response. The envelope's view is
// local to this message; it never changes the selection for a later message.
func writeViewedStreamResultDecoder(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	withID := ed.ClientWebSocket != nil && ed.Result.IDAttribute != ""
	stmt.Func().Id("decode"+ed.Method.VarName+"StreamResult").ParamsFunc(func(g *jen.Group) {
		g.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder"))
		g.Id("data").Qual("encoding/json/jsontext", "Value")
		g.Id("wireView").String()
		if withID {
			g.Id("responseID").Any()
		}
	}).Params(codegen.TypeRef(ed.Result.Ref), jen.Error()).BlockFunc(func(g *jen.Group) {
		if fixed := ed.Method.ViewedResult.ViewName; fixed != "" {
			g.If(jen.Id("wireView").Op("!=").Lit("").Op("&&").Id("wireView").Op("!=").Lit(fixed)).Block(
				jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("unexpected result view %q: expected %q"), jen.Id("wireView"), jen.Lit(fixed))),
			)
		}
		g.Id("resp").Op(":=").Op("&").Qual("net/http", "Response").Values(jen.Dict{
			jen.Id("StatusCode"): jen.Qual("net/http", "StatusOK"),
			jen.Id("Body"):       jen.Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewReader").Call(jen.Id("data"))),
			jen.Id("Header"): jen.Qual("net/http", "Header").Values(jen.Dict{
				jen.Lit("Loom-View"): jen.Index().String().Values(jen.Id("wireView")),
			}),
		})
		response := ed.Result.Responses[0]
		writeSingleResponseDecode(g, response, ed.ServiceName, ed.Method)
		var responseID jen.Code
		if withID {
			responseID = jen.Id("responseID")
		}
		writeJSONRPCViewedInitReturn(g, ed, response, responseID)
	})
}
