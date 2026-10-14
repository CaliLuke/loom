package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcResponseDecoderSection(e *httpcodegen.EndpointData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-response-decoder", func(stmt *jen.Statement) {
		comment := fmt.Sprintf("%s returns a decoder for responses returned by the %s service %s JSON-RPC method. restoreBody controls whether the response body should be restored after having been read.", e.ResponseDecoder, e.ServiceName, e.Method.Name)
		codegen.Doc(stmt, comment)
		stmt.Func().Id(e.ResponseDecoder).
			Params(
				jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")),
				jen.Id("restoreBody").Bool(),
			).
			Params(
				jen.Func().
					Params(jen.Op("*").Qual("net/http", "Response")).
					Params(jen.Any(), jen.Error()),
			).
			Block(
				jen.Return(
					jen.Func().
						Params(jen.Id("resp").Op("*").Qual("net/http", "Response")).
						Params(jen.Any(), jen.Error()).
						BlockFunc(func(g *jen.Group) {
							g.Return(jen.Id("loomhttp").Dot("DecodeResponse").Call(
								jen.Id("resp"), jen.Id("restoreBody"), jen.False(),
								jen.Func().Params(jen.Id("resp").Op("*").Qual("net/http", "Response")).Params(jen.Any(), jen.Error()).BlockFunc(func(body *jen.Group) {
									writeJSONRPCResponseDecoderBody(body, e)
								}),
							))
						}),
				),
			)
	})
}

func writeJSONRPCResponseDecoderBody(g *jen.Group, e *httpcodegen.EndpointData) {
	writeJSONRPCResponseStatusCheck(g, e)
	writeJSONRPCResponseDecodeEnvelope(g, e)
	writeJSONRPCResponseErrorHandling(g, e)
	writeJSONRPCResponseSuccessHandling(g, e)
}

func writeJSONRPCResponseStatusCheck(g *jen.Group, e *httpcodegen.EndpointData) {
	g.If(jen.Id("resp").Dot("StatusCode").Op("!=").Qual("net/http", "StatusOK")).Block(
		jen.List(jen.Id("body"), jen.Err()).Op(":=").Id("loomhttp").Dot("ReadUnexpectedResponseBody").Call(jen.Id("resp")),
		jen.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Nil(), jen.Err())),
		jen.Return(
			jen.Nil(),
			errInvalidResponseExpr(e.ServiceName, e.Method.Name, jen.Id("resp").Dot("StatusCode"), jen.String().Call(jen.Id("body"))),
		),
	)
	g.Line()
}

func writeJSONRPCResponseDecodeEnvelope(g *jen.Group, e *httpcodegen.EndpointData) {
	g.Var().Id("jresp").Qual("github.com/CaliLuke/loom/jsonrpc", "RawResponse")
	g.If(
		jen.Err().Op(":=").Id("decoder").Call(jen.Id("resp")).Dot("Decode").Call(jen.Op("&").Id("jresp")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(
			jen.Nil(),
			errDecodingExpr(e.ServiceName, e.Method.Name, jen.Id("err")),
		),
	)
	g.Line()
}

func writeJSONRPCResponseErrorHandling(g *jen.Group, e *httpcodegen.EndpointData) {
	g.If(jen.Id("jresp").Dot("Error").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Id(e.ErrorDecoder).Call(jen.Id("decoder"), jen.Id("resp"), jen.Id("jresp").Dot("Error"))),
	)
	g.Line()
}

func writeJSONRPCResponseSuccessHandling(g *jen.Group, e *httpcodegen.EndpointData) {
	if e.Result == nil || len(e.Result.Responses) == 0 {
		g.Return(jen.Nil(), jen.Nil())
		return
	}
	resp := e.Result.Responses[0]
	g.Id("resp").Dot("Body").Op("=").Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewBuffer").Call(jen.Id("jresp").Dot("Result")))
	writeSingleResponseDecode(g, resp, e.ServiceName, e.Method)
	writeJSONRPCDecodedResponseReturn(g, e, resp)
}

func writeJSONRPCDecodedResponseReturn(g *jen.Group, e *httpcodegen.EndpointData, resp *httpcodegen.ResponseData) {
	switch {
	case resp.ResultInit != nil:
		writeJSONRPCDecodedInitReturn(g, e, resp)
	case resp.ClientBody != nil:
		g.Return(jen.Id("body"), jen.Nil())
	default:
		g.Return(jen.Nil(), jen.Nil())
	}
}

func writeJSONRPCDecodedInitReturn(g *jen.Group, e *httpcodegen.EndpointData, resp *httpcodegen.ResponseData) {
	if resp.ViewedResult != nil {
		writeJSONRPCViewedInitReturn(g, e, resp, nil)
		return
	}
	writeJSONRPCConstructorCall(g, e, resp.ResultInit, "res")
	if resp.Tag != nil {
		g.Add(codegen.Expr(resp.Tag.Assignment))
	}
	g.Return(jen.Id("res"), jen.Nil())
}

func writeJSONRPCViewedInitReturn(g *jen.Group, e *httpcodegen.EndpointData, resp *httpcodegen.ResponseData, responseID jen.Code) {
	writeJSONRPCConstructorCall(g, e, resp.ResultInit, "p")
	if responseID != nil && e.Result.IDAttribute != "" {
		attr := e.Result.IDAttribute
		g.If(jen.Id("p").Op("!=").Nil().Op("&&").Parens(jen.Id("p").Dot(attr).Op("==").Nil().Op("||").Op("*").Id("p").Dot(attr).Op("==").Lit(""))).Block(
			jen.Id("idCopy").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "IDToString").Call(responseID),
			jen.Id("p").Dot(attr).Op("=").Op("&").Id("idCopy"),
		)
	}
	if resp.Tag != nil {
		g.Add(codegen.Expr(resp.Tag.Assignment))
	}

	if e.Method.ViewedResult != nil && e.Method.ViewedResult.ViewName != "" {
		g.Id("view").Op(":=").Lit(e.Method.ViewedResult.ViewName)
	} else {
		g.Id("view").Op(":=").Id("resp").Dot("Header").Dot("Get").Call(jen.Lit("loom-view"))
	}
	g.Id("vres").Op(":=").Add(codegen.Expr(viewedResultPrefix(e.Method.ViewedResult) + e.Method.ViewedResult.ViewsPkg + "." + e.Method.ViewedResult.VarName + "{Projected: p, View: view}"))
	if resp.ClientBody != nil {
		g.If(
			jen.Id("err").Op("=").Add(codegen.Expr(e.Method.ViewedResult.ViewsPkg+"."+e.Method.ViewedResult.Validate.Name+"(vres)")),
			jen.Id("err").Op("!=").Nil(),
		).Block(
			jen.Return(jen.Nil(), errValidationExpr(e.ServiceName, e.Method.Name, jen.Id("err"))),
		)
	}
	g.List(jen.Id("res"), jen.Id("err")).Op(":=").Add(codegen.Expr(e.ServicePkgName + "." + e.Method.ViewedResult.ResultInit.Name + "(vres)"))
	g.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), errValidationExpr(e.ServiceName, e.Method.Name, jen.Id("err"))),
	)
	g.Return(jen.Id("res"), jen.Nil())
}

func jsonrpcInitArgs(args []*httpcodegen.InitArgData) []jen.Code {
	initArgs := make([]jen.Code, 0, len(args))
	for _, arg := range args {
		initArgs = append(initArgs, codegen.Expr(arg.Ref))
	}
	return initArgs
}

func writeErrorCandidateReturn(g *jen.Group, resp *httpcodegen.ResponseData) {
	switch {
	case resp.ResultInit != nil:
		if resp.ResultInit.ClientErrorAware {
			g.Return(jen.Id(resp.ResultInit.Name).Call(jsonrpcInitArgs(resp.ResultInit.ClientArgs)...))
		} else {
			g.Return(jen.Id(resp.ResultInit.Name).Call(jsonrpcInitArgs(resp.ResultInit.ClientArgs)...), jen.Nil())
		}
	case resp.ClientBody != nil:
		g.Return(jen.Id("body"), jen.Nil())
	default:
		g.Return(jen.Nil(), jen.Nil())
	}
}

func writeSingleResponseDecode(g *jen.Group, data *httpcodegen.ResponseData, serviceName string, method *service.MethodData) {
	if data.ClientBody != nil {
		g.Var().Defs(
			jen.Id("body").Add(codegen.TypeRef(data.ClientBody.ValueRef)),
			jen.Id("err").Error(),
		)
		g.Id("err").Op("=").Id("decoder").Call(jen.Id("resp")).Dot("Decode").Call(jen.Op("&").Id("body"))
		g.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), errDecodingExpr(serviceName, method.Name, jen.Id("err"))),
		)
		if data.ClientBody.ValidateRef != "" {
			g.Add(codegen.Expr(data.ClientBody.ValidateRef))
			g.If(jen.Id("err").Op("!=").Nil()).Block(
				jen.Return(jen.Nil(), errValidationExpr(serviceName, method.Name, jen.Id("err"))),
			)
		}
	}
}

func viewedResultPrefix(v *service.ViewedResultTypeData) string {
	if v == nil || v.IsCollection {
		return ""
	}
	return "&"
}

func errInvalidResponseExpr(serviceName, methodName string, statusCode, body jen.Code) jen.Code {
	return jen.Id("loomhttp").Dot("ErrInvalidResponse").Call(
		jen.Lit(serviceName),
		jen.Lit(methodName),
		statusCode,
		body,
	)
}

func errDecodingExpr(serviceName, methodName string, errExpr jen.Code) jen.Code {
	return jen.Id("loomhttp").Dot("ErrDecodingError").Call(
		jen.Lit(serviceName),
		jen.Lit(methodName),
		errExpr,
	)
}

func errValidationExpr(serviceName, methodName string, errExpr jen.Code) jen.Code {
	return jen.Id("loomhttp").Dot("ErrValidationError").Call(
		jen.Lit(serviceName),
		jen.Lit(methodName),
		errExpr,
	)
}

func writeJSONRPCConstructorCall(group *jen.Group, endpoint *httpcodegen.EndpointData, init *httpcodegen.InitData, target string) {
	assignment := jen.Id(target)
	if init.ClientErrorAware {
		assignment = jen.List(jen.Id(target), jen.Id("err"))
	}
	group.Add(assignment).Op(":=").Id(init.Name).Call(jsonrpcInitArgs(init.ClientArgs)...)
	if init.ClientErrorAware {
		group.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), errValidationExpr(endpoint.ServiceName, endpoint.Method.Name, jen.Id("err"))),
		)
	}
}
