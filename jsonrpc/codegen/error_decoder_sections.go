package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcErrorDecoderSection(e *httpcodegen.EndpointData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-error-decoder", func(stmt *jen.Statement) {
		stmt.Func().Id(e.ErrorDecoder).Params(
			jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")),
			jen.Id("resp").Op("*").Qual("net/http", "Response"),
			jen.Id("wire").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawErrorResponse"),
		).Error().BlockFunc(func(g *jen.Group) {
			g.Switch(jen.Id("wire").Dot("Code")).BlockFunc(func(codes *jen.Group) {
				for _, group := range e.Errors {
					if len(group.Errors) == 0 {
						continue
					}
					codes.Case(codegen.Expr(group.StatusCode)).BlockFunc(func(branch *jen.Group) {
						branch.Return(jen.Qual("github.com/CaliLuke/loom/jsonrpc", "DecodeErrorResponse").Call(
							jen.Lit(e.ServiceName), jen.Lit(e.Method.Name), jen.Id("wire"), jen.Id("resp"),
							jen.Index().Qual("github.com/CaliLuke/loom/jsonrpc", "ErrorCandidate").ValuesFunc(func(candidates *jen.Group) {
								for _, item := range group.Errors {
									if item.Response != nil {
										candidates.ValuesFunc(func(candidate *jen.Group) {
											writeErrorCandidate(candidate, item, e)
										})
									}
								}
							}),
						))
					})
				}
				codes.Default().Block(jen.Return(jen.Id("wire")))
			})
		})
	})
}

func writeErrorCandidate(candidate *jen.Group, item *httpcodegen.ErrorData, e *httpcodegen.EndpointData) {
	candidate.Id("Name").Op(":").Lit(item.Name)
	if item.Selector != nil {
		candidate.Id("Selector").Op(":").Index().String().ValuesFunc(func(path *jen.Group) {
			for _, member := range item.Selector {
				path.Lit(member)
			}
		})
	}
	candidate.Id("Decode").Op(":").Func().Params(jen.Id("resp").Op("*").Qual("net/http", "Response")).Params(jen.Error(), jen.Error()).BlockFunc(func(decode *jen.Group) {
		writeSingleResponseDecode(decode, item.Response, e.ServiceName, e.Method)
		writeErrorCandidateReturn(decode, item.Response)
	})
}
