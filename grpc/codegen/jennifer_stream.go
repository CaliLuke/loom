package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	codegenpkg "github.com/CaliLuke/loom/codegen"
)

func grpcStreamStructSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.NewJenniferSection(stream.Type+"-stream-struct-type", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s implements the %s interface.", stream.VarName, stream.ServiceInterface))
		fields := []jen.Code{
			jen.Id("stream").Add(codegenpkg.TypeRef(stream.Interface)),
		}
		if stream.Type == "server" {
			fields = append(fields, jen.Id("ctx").Qual("context", "Context"))
		}
		if stream.Endpoint.Method.ViewedResult != nil {
			fields = append(fields, jen.Id("view").String())
		}
		stmt.Type().Id(stream.VarName).Struct(fields...)
	})
}

func grpcStreamSendSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.NewJenniferSection(stream.Type+"-stream-send", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, stream.SendDesc)
		body := []jen.Code{}
		sendArg := "res"
		if stream.Endpoint.Method.ViewedResult != nil && stream.Type == "server" {
			viewArg := fmt.Sprintf("%q", stream.Endpoint.Method.ViewedResult.ViewName)
			if stream.Endpoint.Method.ViewedResult.ViewName == "" {
				viewArg = "s.view"
			}
			body = append(body,
				jen.List(jen.Id("vres"), jen.Err()).Op(":=").Qual(stream.Endpoint.ServicePkgName, stream.Endpoint.Method.ViewedResult.Init.Name).
					Call(jen.Id("res"), codegenpkg.Expr(viewArg)),
				jen.If(jen.Err().Op("!=").Nil()).Block(
					jen.Return(grpcStreamEncodeError(stream, jen.Err())),
				),
			)
			sendArg = "vres.Projected"
		}
		if stream.SendConvert.Init.ErrorAware {
			body = append(body,
				jen.List(jen.Id("v"), jen.Err()).Op(":=").Id(stream.SendConvert.Init.Name).Call(codegenpkg.Expr(sendArg)),
				jen.If(jen.Err().Op("!=").Nil()).Block(jen.Return(grpcStreamEncodeError(stream, jen.Err()))),
			)
		} else {
			body = append(body, jen.Id("v").Op(":=").Id(stream.SendConvert.Init.Name).Call(codegenpkg.Expr(sendArg)))
		}
		if stream.Type == "client" && stream.Endpoint.Request.StreamEnvelope != nil {
			env := stream.Endpoint.Request.StreamEnvelope
			body = append(body, jen.Return(jen.Id("s").Dot("stream").Dot(stream.SendName).Call(
				jen.Op("&").Qual(stream.Endpoint.PkgName, stream.Endpoint.Request.Message.VarName).Values(jen.Dict{
					jen.Id(env.FieldName): jen.Op("&").Add(codegenpkg.TypeRef(env.StreamItemWrapperRef)).Values(jen.Dict{
						jen.Id(env.StreamItemFieldName): jen.Id("v"),
					}),
				}),
			)))
		} else {
			send := jen.Id("s").Dot("stream").Dot(stream.SendName).Call(jen.Id("v"))
			if stream.Type == "server" {
				send = codegenpkg.Expr("loomgrpc.ObserveStreamWriteError").Call(jen.Id("s").Dot("ctx"), send)
			}
			body = append(body, jen.Return(send))
		}
		stmt.Func().Params(jen.Id("s").Op("*").Id(stream.VarName)).
			Id(stream.SendName).
			Params(jen.Id("res").Add(codegenpkg.TypeRef(stream.SendRef))).
			Error().
			Block(body...)
		stmt.Line()
		codegenpkg.Doc(stmt, stream.SendWithContextDesc)
		stmt.Func().Params(jen.Id("s").Op("*").Id(stream.VarName)).
			Id(stream.SendWithContextName).
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("res").Add(codegenpkg.TypeRef(stream.SendRef)),
			).
			Error().
			Block(
				jen.Return(jen.Id("s").Dot(stream.SendName).Call(jen.Id("res"))),
			)
	})
}

func grpcStreamRecvSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.NewJenniferSection(stream.Type+"-stream-recv", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, stream.RecvDesc)
		stmt.Func().Params(jen.Id("s").Op("*").Id(stream.VarName)).
			Id(stream.RecvName).
			Params().
			Params(codegenpkg.TypeRef(stream.RecvRef), jen.Error()).
			BlockFunc(func(g *jen.Group) {
				g.Var().Id("res").Add(codegenpkg.TypeRef(stream.RecvRef))
				if stream.Type == "server" && stream.Endpoint.Request.StreamEnvelope != nil {
					g.List(jen.Id("message"), jen.Err()).Op(":=").Id("s").Dot("stream").Dot(stream.RecvName).Call()
				} else {
					g.List(jen.Id("v"), jen.Err()).Op(":=").Id("s").Dot("stream").Dot(stream.RecvName).Call()
				}
				appendGRPCStreamRecvErrorHandling(g, stream)
				if stream.Type == "server" && stream.Endpoint.Request.StreamEnvelope != nil {
					appendGRPCStreamRecvEnvelopeUnpack(g, stream, stream.Endpoint.Request.StreamEnvelope)
				}
				if appendGRPCStreamRecvViewedResult(g, stream) {
					return
				}
				if stream.RecvConvert.Validation != nil {
					g.If(
						jen.Err().Op("=").Id(stream.RecvConvert.Validation.Name).Call(jen.Id("v")),
						jen.Err().Op("!=").Nil(),
					).Block(
						jen.Return(jen.Id("res"), grpcStreamDecodeError(stream, jen.Err())),
					)
				}
				initArgs := make([]jen.Code, 0, len(stream.RecvConvert.Init.Args))
				for _, arg := range stream.RecvConvert.Init.Args {
					initArgs = append(initArgs, codegenpkg.Expr(arg.Name))
				}
				g.Return(jen.Id(stream.RecvConvert.Init.Name).Call(initArgs...), jen.Nil())
			})
		stmt.Line()
		codegenpkg.Doc(stmt, stream.RecvWithContextDesc)
		stmt.Func().Params(jen.Id("s").Op("*").Id(stream.VarName)).
			Id(stream.RecvWithContextName).
			Params(jen.Id("ctx").Qual("context", "Context")).
			Params(codegenpkg.TypeRef(stream.RecvRef), jen.Error()).
			Block(
				jen.Return(jen.Id("s").Dot(stream.RecvName).Call()),
			)
	})
}

func appendGRPCStreamRecvEnvelopeUnpack(g *jen.Group, stream *StreamData, env *StreamEnvelopeData) {
	g.List(jen.Id("body"), jen.Id("ok")).Op(":=").Id("message").Dot(env.FieldName).Assert(jen.Op("*").Add(codegenpkg.TypeRef(env.StreamItemWrapperRef)))
	g.If(jen.Op("!").Id("ok")).Block(
		jen.Switch(jen.Id("message").Dot(env.FieldName).Assert(jen.Type())).Block(
			jen.Case(jen.Op("*").Add(codegenpkg.TypeRef(env.InitialWrapperRef))).Block(
				jen.Return(jen.Id("res"), grpcStreamDecodeError(stream, codegenpkg.Expr(`loom.InvalidFieldTypeError("body", "initial_payload", "stream_item")`))),
			),
			jen.Default().Block(
				jen.Return(jen.Id("res"), grpcStreamDecodeError(stream, codegenpkg.Expr(`loom.MissingFieldError("stream_item", "stream")`))),
			),
		),
	)
	g.If(jen.Id("body").Dot(env.StreamItemFieldName).Op("==").Nil()).Block(
		jen.Return(jen.Id("res"), grpcStreamDecodeError(stream, codegenpkg.Expr(`loom.MissingFieldError("stream_item", "stream")`))),
	)
	g.Id("v").Op(":=").Id("body").Dot(env.StreamItemFieldName)
}

func appendGRPCStreamRecvErrorHandling(g *jen.Group, stream *StreamData) {
	g.If(jen.Err().Op("!=").Nil()).BlockFunc(func(eg *jen.Group) {
		if stream.Endpoint != nil && len(stream.Endpoint.Errors) > 0 && stream.Type == "client" {
			eg.Id("resp").Op(":=").Add(codegenpkg.Expr("loomgrpc.DecodeError")).Call(jen.Err())
			eg.Switch(jen.Id("message").Op(":=").Id("resp").Assert(jen.Type())).BlockFunc(func(sg *jen.Group) {
				for _, errData := range stream.Endpoint.Errors {
					if errData.Response.ClientConvert == nil {
						continue
					}
					sg.Case(codegenpkg.Expr(errData.Response.ClientConvert.SrcRef)).Block(grpcStreamRecvErrorCase(errData)...)
				}
				sg.Case(jen.Op("*").Id("loompb").Dot("ErrorResponse")).Block(
					jen.Return(jen.Id("res"), codegenpkg.Expr("loomgrpc.NewServiceError").Call(jen.Id("message"))),
				)
				sg.Default().Block(
					jen.Return(jen.Id("res"), jen.Err()),
				)
			})
			return
		}
		eg.Return(jen.Id("res"), grpcStreamDecodeError(stream, jen.Err()))
	})
}

func grpcStreamEncodeError(stream *StreamData, err jen.Code) jen.Code {
	if stream.Type != "server" {
		return err
	}
	return codegenpkg.Expr("loomgrpc.ObserveStreamEncodeError").Call(jen.Id("s").Dot("ctx"), err)
}

func grpcStreamDecodeError(stream *StreamData, err jen.Code) jen.Code {
	if stream.Type != "server" {
		return err
	}
	return codegenpkg.Expr("loomgrpc.ObserveStreamDecodeError").Call(jen.Id("s").Dot("ctx"), err)
}

func grpcStreamRecvErrorCase(errData *ErrorData) []jen.Code {
	caseBody := make([]jen.Code, 0, 2)
	if errData.Response.ClientConvert.Validation != nil {
		caseBody = append(caseBody,
			jen.If(
				jen.Err().Op(":=").Id(errData.Response.ClientConvert.Validation.Name).Call(jen.Id("message")),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Return(jen.Id("res"), jen.Err()),
			),
		)
	}
	caseBody = append(caseBody,
		jen.Return(jen.Id("res"), jen.Id(errData.Response.ClientConvert.Init.Name).Call(grpcStreamRecvInitArgs(errData.Response.ClientConvert.Init)...)),
	)
	return caseBody
}

func appendGRPCStreamRecvViewedResult(g *jen.Group, stream *StreamData) bool {
	if stream.Endpoint == nil || stream.Endpoint.Method.ViewedResult == nil || stream.Type != "client" {
		return false
	}
	viewArg := fmt.Sprintf("%q", stream.Endpoint.Method.ViewedResult.ViewName)
	if stream.Endpoint.Method.ViewedResult.ViewName == "" {
		viewArg = "s.view"
	}
	g.Id("proj").Op(":=").Id(stream.RecvConvert.Init.Name).Call(grpcStreamRecvInitArgs(stream.RecvConvert.Init)...)
	if !stream.Endpoint.Method.ViewedResult.IsCollection {
		g.Add(codegenpkg.Expr("vres := &" + stream.Endpoint.Method.ViewedResult.FullName + "{Projected: proj, View: " + viewArg + "}"))
	} else {
		g.Add(codegenpkg.Expr("vres := " + stream.Endpoint.Method.ViewedResult.FullName + "{Projected: proj, View: " + viewArg + "}"))
	}
	g.If(
		jen.Err().Op(":=").Qual(stream.Endpoint.Method.ViewedResult.ViewsPkg, "Validate"+stream.Endpoint.Method.Result).Call(jen.Id("vres")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Nil(), jen.Err()),
	)
	g.List(jen.Id("out"), jen.Err()).Op(":=").Qual(stream.Endpoint.ServicePkgName, stream.Endpoint.Method.ViewedResult.ResultInit.Name).Call(jen.Id("vres"))
	g.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Err()),
	)
	g.Return(jen.Id("out"), jen.Nil())
	return true
}

func grpcStreamRecvInitArgs(init *InitData) []jen.Code {
	args := make([]jen.Code, 0, len(init.Args))
	for _, arg := range init.Args {
		args = append(args, codegenpkg.Expr(arg.Name))
	}
	return args
}

func grpcStreamCloseSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.NewJenniferSection(stream.Type+"-stream-close", func(stmt *jen.Statement) {
		stmt.Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(stream.VarName)).Id("Close").Params().Error().BlockFunc(func(g *jen.Group) {
			if stream.Type == "client" {
				if stream.Endpoint.Method.ResultRef != "" {
					g.Comment(codegenpkg.Comment("Close the send direction of the stream"))
					g.Return(jen.Id("s").Dot("stream").Dot("CloseSend").Call())
				} else {
					g.Comment(codegenpkg.Comment("synchronize and report any server error"))
					g.List(jen.Id("_"), jen.Err()).Op(":=").Id("s").Dot("stream").Dot("CloseAndRecv").Call()
					g.Return(jen.Err())
				}
				return
			}
			if stream.Endpoint.Method.ResultRef != "" {
				g.Comment(codegenpkg.Comment("nothing to do here"))
				g.Return(jen.Nil())
				return
			}
			g.Comment(codegenpkg.Comment("synchronize stream"))
			g.Return(codegenpkg.Expr("loomgrpc.ObserveStreamWriteError").Call(
				jen.Id("s").Dot("ctx"),
				jen.Id("s").Dot("stream").Dot("SendAndClose").Call(
					jen.Op("&").Id(stream.Endpoint.Response.ServerConvert.TgtName).Values(),
				),
			))
		})
	})
}

func writeGRPCServerErrorMapper(stmt *jen.Statement, endpoint *EndpointData) {
	stmt.Func().Id("map"+endpoint.Method.VarName+"Error").
		Params(jen.Id("name").String(), jen.Err().Error()).
		Params(codegenpkg.TypeRef("loomgrpc.ErrorMapping"), jen.Bool(), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Switch(jen.Id("name")).BlockFunc(func(cases *jen.Group) {
				for _, errData := range endpoint.Errors {
					cases.Case(jen.Lit(errData.Name)).Block(grpcServerErrorMappingCase(errData)...)
				}
			})
			g.Return(emptyGRPCServerErrorMapping()...)
		})
}

func grpcServerErrorMappingCase(errData *ErrorData) []jen.Code {
	body := []jen.Code{}
	if errData.Response.ServerConvert != nil {
		body = append(body,
			jen.Var().Id("er").Add(codegenpkg.TypeRef(errData.Response.ServerConvert.SrcRef)),
			jen.If(
				jen.Op("!").Qual("errors", "As").Call(jen.Err(), jen.Op("&").Id("er")),
			).Block(
				jen.Return(emptyGRPCServerErrorMapping()...),
			),
		)
	}
	detail := codegenpkg.Expr("loomgrpc.NewErrorResponse(err)")
	if errData.Response.ServerConvert != nil {
		initArgs := make([]jen.Code, 0, len(errData.Response.ServerConvert.Init.Args))
		for _, arg := range errData.Response.ServerConvert.Init.Args {
			initArgs = append(initArgs, codegenpkg.Expr(arg.Name))
		}
		if errData.Response.ServerConvert.Init.ErrorAware {
			body = append(body,
				jen.List(jen.Id("message"), jen.Id("conversionErr")).Op(":=").Id(errData.Response.ServerConvert.Init.Name).Call(initArgs...),
				jen.If(jen.Id("conversionErr").Op("!=").Nil()).Block(
					jen.Return(
						codegenpkg.TypeRef("loomgrpc.ErrorMapping").Values(),
						jen.False(),
						jen.Id("conversionErr"),
					),
				),
			)
			detail = jen.Id("message")
		} else {
			detail = jen.Id(errData.Response.ServerConvert.Init.Name).Call(initArgs...)
		}
	}
	body = append(body,
		jen.Return(
			codegenpkg.TypeRef("loomgrpc.ErrorMapping").Values(jen.Dict{
				jen.Id("Code"):   codegenpkg.Expr(errData.Response.StatusCode),
				jen.Id("Detail"): detail,
			}),
			jen.True(),
			jen.Nil(),
		),
	)
	return body
}

func emptyGRPCServerErrorMapping() []jen.Code {
	return []jen.Code{
		codegenpkg.TypeRef("loomgrpc.ErrorMapping").Values(),
		jen.False(),
		jen.Nil(),
	}
}

func grpcStreamSetViewSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.NewJenniferSection(stream.Type+"-stream-set-view", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, "SetView sets the view.")
		stmt.Func().Params(jen.Id("s").Op("*").Id(stream.VarName)).
			Id("SetView").
			Params(jen.Id("view").String()).
			Block(
				jen.Id("s").Dot("view").Op("=").Id("view"),
			)
	})
}
