package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	codegenpkg "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/expr"
)

func grpcClientStructSection(data *ServiceData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("client-struct", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s lists the service endpoint gRPC clients.", data.ClientStruct))
		stmt.Type().Id(data.ClientStruct).Struct(
			jen.Id("grpccli").Add(codegenpkg.TypeRef(data.PkgName+"."+data.ClientInterface)),
			jen.Id("opts").Index().Qual("google.golang.org/grpc", "CallOption"),
		)
	})
}

func grpcClientInitSection(data *ServiceData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("grpc-client-init", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("New%s instantiates gRPC client for all the %s service servers.", data.ClientStruct, data.Service.Name))
		stmt.Func().Id("New"+data.ClientStruct).
			Params(
				jen.Id("cc").Op("*").Qual("google.golang.org/grpc", "ClientConn"),
				jen.Id("opts").Op("...").Qual("google.golang.org/grpc", "CallOption"),
			).
			Op("*").Id(data.ClientStruct).
			Block(
				jen.Return(
					jen.Op("&").Id(data.ClientStruct).Values(jen.Dict{
						jen.Id("grpccli"): codegenpkg.Expr(data.ClientInterfaceInit).Call(jen.Id("cc")),
						jen.Id("opts"):    jen.Id("opts"),
					}),
				),
			)
	})
}

func grpcClientEndpointInitSection(endpoint *EndpointData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("client-endpoint-init", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s calls the %q function in %s.%s interface.", endpoint.Method.VarName, endpoint.Method.VarName, endpoint.PkgName, endpoint.ClientInterface))
		stmt.Func().Params(jen.Id("c").Op("*").Id(endpoint.ClientStruct)).
			Id(endpoint.Method.VarName).
			Params().
			Add(codegenpkg.TypeRef("goa.Endpoint")).
			Block(
				jen.Return(
					jen.Func().
						Params(
							jen.Id("ctx").Qual("context", "Context"),
							jen.Id("v").Any(),
						).
						Params(jen.Any(), jen.Error()).
						BlockFunc(func(g *jen.Group) {
							encodeFn := jen.Id("nil")
							if endpoint.PayloadRef != "" {
								encodeFn = jen.Id("Encode" + endpoint.Method.VarName + "Request")
							}
							decodeFn := jen.Id("nil")
							if endpoint.ResultRef != "" || endpoint.ClientStream != nil {
								decodeFn = jen.Id("Decode" + endpoint.Method.VarName + "Response")
							}
							g.Id("inv").Op(":=").Add(codegenpkg.Expr("goagrpc.NewInvoker")).Call(
								jen.Id("Build"+endpoint.Method.VarName+"Func").Call(jen.Id("c").Dot("grpccli"), jen.Id("c").Dot("opts").Op("...")),
								encodeFn,
								decodeFn,
							)
							g.List(jen.Id("res"), jen.Err()).Op(":=").Id("inv").Dot("Invoke").Call(jen.Id("ctx"), jen.Id("v"))
							g.If(jen.Err().Op("!=").Nil()).BlockFunc(func(eg *jen.Group) {
								eg.Id("resp").Op(":=").Add(codegenpkg.Expr("goagrpc.DecodeError")).Call(jen.Err())
								if len(endpoint.Errors) > 0 {
									eg.Switch(jen.Id("message").Op(":=").Id("resp").Assert(jen.Type())).BlockFunc(func(sg *jen.Group) {
										for _, errData := range endpoint.Errors {
											if errData.Response.ClientConvert == nil {
												continue
											}
											caseBody := []jen.Code{}
											if errData.Response.ClientConvert.Validation != nil {
												caseBody = append(caseBody,
													jen.If(
														jen.Err().Op(":=").Id(errData.Response.ClientConvert.Validation.Name).Call(jen.Id("message")),
														jen.Err().Op("!=").Nil(),
													).Block(
														jen.Return(jen.Nil(), jen.Err()),
													),
												)
											}
											initArgs := make([]jen.Code, 0, len(errData.Response.ClientConvert.Init.Args))
											for _, arg := range errData.Response.ClientConvert.Init.Args {
												initArgs = append(initArgs, codegenpkg.Expr(arg.Name))
											}
											caseBody = append(caseBody,
												jen.Return(
													jen.Nil(),
													jen.Id(errData.Response.ClientConvert.Init.Name).Call(initArgs...),
												),
											)
											sg.Case(codegenpkg.Expr(errData.Response.ClientConvert.SrcRef)).Block(caseBody...)
										}
										sg.Case(jen.Op("*").Id("goapb").Dot("ErrorResponse")).Block(
											jen.Return(
												jen.Nil(),
												codegenpkg.Expr("goagrpc.NewServiceError").Call(jen.Id("message")),
											),
										)
										sg.Default().Block(
											jen.Return(
												jen.Nil(),
												codegenpkg.Expr("goa.Fault").Call(jen.Lit("%s"), jen.Err().Dot("Error").Call()),
											),
										)
									})
									return
								}
								eg.Comment(codegenpkg.Comment("Try to decode a Loom error response detail before falling back to Fault."))
								eg.If(
									jen.List(jen.Id("eresp"), jen.Id("ok")).Op(":=").Id("resp").Assert(jen.Op("*").Id("goapb").Dot("ErrorResponse")),
									jen.Id("ok"),
								).Block(
									jen.Return(
										jen.Nil(),
										codegenpkg.Expr("goagrpc.NewServiceError").Call(jen.Id("eresp")),
									),
								)
								eg.Return(
									jen.Nil(),
									codegenpkg.Expr("goa.Fault").Call(jen.Lit("%s"), jen.Err().Dot("Error").Call()),
								)
							})
							g.Return(jen.Id("res"), jen.Nil())
						}),
				),
			)
	})
}

func grpcServerStructSection(data *ServiceData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("server-struct", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s implements the %s.%s interface.", data.ServerStruct, data.PkgName, data.ServerInterface))
		fields := make([]jen.Code, 0, len(data.Endpoints)+1)
		for _, endpoint := range data.Endpoints {
			handlerType := "UnaryHandler"
			if endpoint.ServerStream != nil {
				handlerType = "StreamHandler"
			}
			fields = append(fields, jen.Id(endpoint.Method.VarName+"H").Add(codegenpkg.TypeRef("goagrpc."+handlerType)))
		}
		fields = append(fields, jen.Qual(data.PkgName, "Unimplemented"+data.ServerInterface))
		stmt.Type().Id(data.ServerStruct).Struct(fields...)
	})
}

func grpcServerInitSection(data *ServiceData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("server-init", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s instantiates the server struct with the %s service endpoints.", data.ServerInit, data.Service.Name))
		params := []jen.Code{jen.Id("e").Op("*").Qual(data.Service.PkgName, "Endpoints")}
		if data.HasUnaryEndpoint() {
			params = append(params, jen.Id("uh").Add(codegenpkg.TypeRef("goagrpc.UnaryHandler")))
		}
		if data.HasStreamingEndpoint() {
			params = append(params, jen.Id("sh").Add(codegenpkg.TypeRef("goagrpc.StreamHandler")))
		}
		dict := jen.Dict{}
		for _, endpoint := range data.Endpoints {
			handlerCtor := "New" + endpoint.Method.VarName + "Handler"
			handlerArg := jen.Id("uh")
			if endpoint.ServerStream != nil {
				handlerArg = jen.Id("sh")
			}
			dict[jen.Id(endpoint.Method.VarName+"H")] = jen.Id(handlerCtor).Call(
				jen.Id("e").Dot(endpoint.Method.VarName),
				handlerArg,
			)
		}
		stmt.Func().Id(data.ServerInit).
			Params(params...).
			Op("*").Id(data.ServerStruct).
			Block(
				jen.Return(jen.Op("&").Id(data.ServerStruct).Values(dict)),
			)
	})
}

func grpcHandlerInitSection(endpoint *EndpointData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("grpc-handler-init", func(stmt *jen.Statement) {
		handlerKind := "Unary"
		decodeArg := jen.Id("nil")
		encodeArgs := []jen.Code{decodeArg, jen.Id("Encode" + endpoint.Method.VarName + "Response")}
		if endpoint.ServerStream != nil {
			handlerKind = "Stream"
			encodeArgs = []jen.Code{decodeArg}
		}
		if endpoint.Method.PayloadRef != "" {
			decodeArg = jen.Id("Decode" + endpoint.Method.VarName + "Request")
			encodeArgs[0] = decodeArg
		}
		codegenpkg.Doc(stmt, fmt.Sprintf("New%sHandler creates a gRPC handler which serves the %q service %q endpoint.", endpoint.Method.VarName, endpoint.ServiceName, endpoint.Method.Name))
		stmt.Func().Id("New"+endpoint.Method.VarName+"Handler").
			Params(
				jen.Id("endpoint").Add(codegenpkg.TypeRef("goa.Endpoint")),
				jen.Id("h").Add(codegenpkg.TypeRef("goagrpc."+handlerKind+"Handler")),
			).
			Add(codegenpkg.TypeRef("goagrpc."+handlerKind+"Handler")).
			Block(
				jen.If(jen.Id("h").Op("==").Nil()).Block(
					jen.Id("h").Op("=").Add(codegenpkg.TypeRef("goagrpc.New"+handlerKind+"Handler")).Call(
						append([]jen.Code{jen.Id("endpoint")}, encodeArgs...)...,
					),
				),
				jen.Return(jen.Id("h")),
			)
	})
}

func grpcServerInterfaceSection(endpoint *EndpointData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("server-grpc-interface", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s implements the %q method in %s.%s interface.", endpoint.Method.VarName, endpoint.Method.VarName, endpoint.PkgName, endpoint.ServerInterface))
		params := []jen.Code{}
		if endpoint.ServerStream == nil {
			params = append(params, jen.Id("ctx").Qual("context", "Context"))
		}
		if endpoint.Method.StreamingPayload == "" {
			params = append(params, jen.Id("message").Add(codegenpkg.TypeRef(endpoint.Request.Message.Ref)))
		}
		if endpoint.ServerStream != nil {
			params = append(params, jen.Id("stream").Add(codegenpkg.TypeRef(endpoint.ServerStream.Interface)))
		}
		results := []jen.Code{jen.Error()}
		if endpoint.ServerStream == nil && endpoint.Response.Message != nil {
			results = []jen.Code{codegenpkg.TypeRef(endpoint.Response.Message.Ref), jen.Error()}
		}
		stmt.Func().Params(jen.Id("s").Op("*").Id(endpoint.ServerStruct)).
			Id(endpoint.Method.VarName).
			Params(params...).
			Params(results...).
			BlockFunc(func(g *jen.Group) {
				if endpoint.ServerStream != nil {
					g.Id("ctx").Op(":=").Id("stream").Dot("Context").Call()
				}
				g.Id("ctx").Op("=").Qual("context", "WithValue").Call(jen.Id("ctx"), codegenpkg.Expr("goa.MethodKey"), jen.Lit(endpoint.Method.Name))
				g.Id("ctx").Op("=").Qual("context", "WithValue").Call(jen.Id("ctx"), codegenpkg.Expr("goa.ServiceKey"), jen.Lit(endpoint.ServiceName))
				if endpoint.ServerStream != nil {
					decodeTarget := "_"
					if endpoint.PayloadRef != "" {
						decodeTarget = "p"
					}
					decodeArg := jen.Id("message")
					if endpoint.Method.StreamingPayload != "" {
						decodeArg = jen.Nil()
					}
					g.List(jen.Id(decodeTarget), jen.Err()).Op(":=").Id("s").Dot(endpoint.Method.VarName+"H").Dot("Decode").Call(jen.Id("ctx"), decodeArg)
					appendGRPCServerErrorHandler(g, endpoint, true)
					g.Id("ep").Op(":=").Op("&").Qual(endpoint.ServicePkgName, endpoint.Method.VarName+"EndpointInput").ValuesFunc(func(dict *jen.Group) {
						dict.Id("Stream").Op(":").Op("&").Id(endpoint.ServerStream.VarName).Values(jen.Dict{
							jen.Id("stream"): jen.Id("stream"),
						})
						if endpoint.PayloadRef != "" {
							dict.Id("Payload").Op(":").Id("p").Assert(codegenpkg.Expr(endpoint.PayloadRef))
						}
					})
					g.Err().Op("=").Id("s").Dot(endpoint.Method.VarName+"H").Dot("Handle").Call(jen.Id("ctx"), jen.Id("ep"))
				} else {
					g.List(jen.Id("resp"), jen.Err()).Op(":=").Id("s").Dot(endpoint.Method.VarName+"H").Dot("Handle").Call(jen.Id("ctx"), jen.Id("message"))
				}
				appendGRPCServerErrorHandler(g, endpoint, endpoint.ServerStream != nil)
				if endpoint.ServerStream == nil {
					g.Return(jen.Id("resp").Assert(codegenpkg.Expr(endpoint.Response.ServerConvert.TgtRef)), jen.Nil())
					return
				}
				g.Return(jen.Nil())
			})
	})
}

func grpcStreamStructSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection(stream.Type+"-stream-struct-type", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s implements the %s interface.", stream.VarName, stream.ServiceInterface))
		fields := []jen.Code{
			jen.Id("stream").Add(codegenpkg.TypeRef(stream.Interface)),
		}
		if stream.Endpoint.Method.ViewedResult != nil {
			fields = append(fields, jen.Id("view").String())
		}
		stmt.Type().Id(stream.VarName).Struct(fields...)
	})
}

func grpcStreamSendSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection(stream.Type+"-stream-send", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, stream.SendDesc)
		body := []jen.Code{}
		sendArg := "res"
		if stream.Endpoint.Method.ViewedResult != nil && stream.Type == "server" {
			viewArg := fmt.Sprintf("%q", stream.Endpoint.Method.ViewedResult.ViewName)
			if stream.Endpoint.Method.ViewedResult.ViewName == "" {
				viewArg = "s.view"
			}
			body = append(body,
				jen.Id("vres").Op(":=").Qual(stream.Endpoint.ServicePkgName, stream.Endpoint.Method.ViewedResult.Init.Name).
					Call(jen.Id("res"), codegenpkg.Expr(viewArg)),
			)
			sendArg = "vres.Projected"
		}
		body = append(body,
			jen.Id("v").Op(":=").Id(stream.SendConvert.Init.Name).Call(codegenpkg.Expr(sendArg)),
			jen.Return(jen.Id("s").Dot("stream").Dot(stream.SendName).Call(jen.Id("v"))),
		)
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
	return codegenpkg.MustJenniferSection(stream.Type+"-stream-recv", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, stream.RecvDesc)
		stmt.Func().Params(jen.Id("s").Op("*").Id(stream.VarName)).
			Id(stream.RecvName).
			Params().
			Params(codegenpkg.TypeRef(stream.RecvRef), jen.Error()).
			BlockFunc(func(g *jen.Group) {
				g.Var().Id("res").Add(codegenpkg.TypeRef(stream.RecvRef))
				g.List(jen.Id("v"), jen.Err()).Op(":=").Id("s").Dot("stream").Dot(stream.RecvName).Call()
				g.If(jen.Err().Op("!=").Nil()).BlockFunc(func(eg *jen.Group) {
					if stream.Endpoint != nil && len(stream.Endpoint.Errors) > 0 && stream.Type == "client" {
						eg.Id("resp").Op(":=").Add(codegenpkg.Expr("goagrpc.DecodeError")).Call(jen.Err())
						eg.Switch(jen.Id("message").Op(":=").Id("resp").Assert(jen.Type())).BlockFunc(func(sg *jen.Group) {
							for _, errData := range stream.Endpoint.Errors {
								if errData.Response.ClientConvert == nil {
									continue
								}
								caseBody := []jen.Code{}
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
								initArgs := make([]jen.Code, 0, len(errData.Response.ClientConvert.Init.Args))
								for _, arg := range errData.Response.ClientConvert.Init.Args {
									initArgs = append(initArgs, codegenpkg.Expr(arg.Name))
								}
								caseBody = append(caseBody,
									jen.Return(jen.Id("res"), jen.Id(errData.Response.ClientConvert.Init.Name).Call(initArgs...)),
								)
								sg.Case(codegenpkg.Expr(errData.Response.ClientConvert.SrcRef)).Block(caseBody...)
							}
							sg.Case(jen.Op("*").Id("goapb").Dot("ErrorResponse")).Block(
								jen.Return(jen.Id("res"), codegenpkg.Expr("goagrpc.NewServiceError").Call(jen.Id("message"))),
							)
							sg.Default().Block(
								jen.Return(jen.Id("res"), jen.Err()),
							)
						})
						return
					}
					eg.Return(jen.Id("res"), jen.Err())
				})
				if stream.Endpoint != nil && stream.Endpoint.Method.ViewedResult != nil && stream.Type == "client" {
					viewArg := fmt.Sprintf("%q", stream.Endpoint.Method.ViewedResult.ViewName)
					if stream.Endpoint.Method.ViewedResult.ViewName == "" {
						viewArg = "s.view"
					}
					initArgs := make([]jen.Code, 0, len(stream.RecvConvert.Init.Args))
					for _, arg := range stream.RecvConvert.Init.Args {
						initArgs = append(initArgs, codegenpkg.Expr(arg.Name))
					}
					g.Id("proj").Op(":=").Id(stream.RecvConvert.Init.Name).Call(initArgs...)
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
					g.Return(
						jen.Qual(stream.Endpoint.ServicePkgName, stream.Endpoint.Method.ViewedResult.ResultInit.Name).Call(jen.Id("vres")),
						jen.Nil(),
					)
					return
				}
				if stream.RecvConvert.Validation != nil {
					g.If(
						jen.Err().Op("=").Id(stream.RecvConvert.Validation.Name).Call(jen.Id("v")),
						jen.Err().Op("!=").Nil(),
					).Block(
						jen.Return(jen.Id("res"), jen.Err()),
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

func grpcStreamCloseSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection(stream.Type+"-stream-close", func(stmt *jen.Statement) {
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
			g.Return(jen.Id("s").Dot("stream").Dot("SendAndClose").Call(jen.Op("&").Id(stream.Endpoint.Response.ServerConvert.TgtName).Values()))
		})
	})
}

func appendGRPCServerErrorHandler(g *jen.Group, endpoint *EndpointData, serverStream bool) {
	g.If(jen.Err().Op("!=").Nil()).BlockFunc(func(eg *jen.Group) {
		if len(endpoint.Errors) > 0 {
			prefix := []jen.Code{}
			if !serverStream {
				prefix = append(prefix, jen.Nil())
			}
			eg.Var().Id("en").Add(codegenpkg.TypeRef("goa.LoomErrorNamer"))
			eg.If(jen.Qual("errors", "As").Call(jen.Err(), jen.Op("&").Id("en"))).Block(
				jen.Switch(jen.Id("en").Dot("LoomErrorName").Call()).BlockFunc(func(sg *jen.Group) {
					for _, errData := range endpoint.Errors {
						body := []jen.Code{}
						if errData.Response.ServerConvert != nil {
							body = append(body,
								jen.Var().Id("er").Add(codegenpkg.TypeRef(errData.Response.ServerConvert.SrcRef)),
								jen.Qual("errors", "As").Call(jen.Err(), jen.Op("&").Id("er")),
							)
						}
						statusArg := codegenpkg.Expr("goagrpc.NewErrorResponse(err)")
						if errData.Response.ServerConvert != nil {
							initArgs := make([]jen.Code, 0, len(errData.Response.ServerConvert.Init.Args))
							for _, arg := range errData.Response.ServerConvert.Init.Args {
								initArgs = append(initArgs, codegenpkg.Expr(arg.Name))
							}
							statusArg = jen.Id(errData.Response.ServerConvert.Init.Name).Call(initArgs...)
						}
						body = append(body,
							jen.Return(append(prefix,
								codegenpkg.Expr("goagrpc.NewStatusError").Call(
									codegenpkg.Expr(errData.Response.StatusCode),
									jen.Err(),
									statusArg,
								),
							)...),
						)
						sg.Case(jen.Lit(errData.Name)).Block(body...)
					}
				}),
			)
		}
		ret := []jen.Code{}
		if !serverStream {
			ret = append(ret, jen.Nil())
		}
		ret = append(ret, codegenpkg.Expr("goagrpc.EncodeError").Call(jen.Err()))
		eg.Return(ret...)
	})
}

func grpcStreamSetViewSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection(stream.Type+"-stream-set-view", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, "SetView sets the view.")
		stmt.Func().Params(jen.Id("s").Op("*").Id(stream.VarName)).
			Id("SetView").
			Params(jen.Id("view").String()).
			Block(
				jen.Id("s").Dot("view").Op("=").Id("view"),
			)
	})
}

func grpcExampleCLISection(defaultTransportType string, services []*ServiceData, interceptorsPkg string) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("do-grpc-cli", func(stmt *jen.Statement) {
		stmt.Func().Id("doGRPC").Params(
			jen.List(jen.Id("_"), jen.Id("host")).String(),
			jen.Id("_").Int(),
			jen.Id("_").Bool(),
		).Params(codegenpkg.TypeRef("goa.Endpoint"), jen.Any(), jen.Error()).BlockFunc(func(g *jen.Group) {
			g.List(jen.Id("conn"), jen.Err()).Op(":=").Qual("google.golang.org/grpc", "NewClient").Call(
				jen.Id("host"),
				jen.Qual("google.golang.org/grpc", "WithTransportCredentials").Call(
					jen.Qual("google.golang.org/grpc/credentials/insecure", "NewCredentials").Call(),
				),
			)
			g.If(jen.Err().Op("!=").Nil()).Block(
				jen.Qual("fmt", "Fprintf").Call(
					jen.Qual("os", "Stderr"),
					jen.Lit("could not connect to gRPC server at %s: %v\n"),
					jen.Id("host"),
					jen.Err(),
				),
			)
			for _, service := range services {
				if len(service.Service.ClientInterceptors) == 0 {
					continue
				}
				g.Id(service.Service.VarName+"Interceptors").Op(":=").Qual(interceptorsPkg, "New"+service.Service.StructName+"ClientInterceptors").Call()
			}
			callArgs := []jen.Code{jen.Id("conn")}
			for _, service := range services {
				if len(service.Service.ClientInterceptors) == 0 {
					continue
				}
				callArgs = append(callArgs, jen.Id(service.Service.VarName+"Interceptors"))
			}
			g.Return(jen.Qual("cli", "ParseEndpoint").Call(callArgs...))
		})
		if defaultTransportType == "grpc" {
			stmt.Line()
			stmt.Func().Id("grpcUsageCommands").Params().Index().String().Block(
				jen.Return(jen.Qual("cli", "UsageCommands").Call()),
			)
			stmt.Line()
			stmt.Func().Id("grpcUsageExamples").Params().String().Block(
				jen.Return(jen.Qual("cli", "UsageExamples").Call()),
			)
		}
	})
}

func grpcParseEndpointSection(flagsCode string, commands []*cli.CommandData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("parse-endpoint-grpc", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, "ParseEndpoint returns the endpoint and payload as specified on the command line.")
		params := []jen.Code{
			jen.Id("cc").Op("*").Qual("google.golang.org/grpc", "ClientConn"),
		}
		for _, command := range commands {
			if command.Interceptors != nil {
				params = append(params,
					jen.Id(command.Interceptors.VarName).Qual(command.Interceptors.PkgName, "ClientInterceptors"),
				)
			}
		}
		params = append(params, jen.Id("opts").Op("...").Qual("google.golang.org/grpc", "CallOption"))
		stmt.Func().Id("ParseEndpoint").
			Params(params...).
			Params(codegenpkg.TypeRef("goa.Endpoint"), jen.Any(), jen.Error()).
			BlockFunc(func(g *jen.Group) {
				g.Add(codegenpkg.Expr(flagsCode))
				g.Var().Defs(
					jen.Id("data").Any(),
					jen.Id("endpoint").Add(codegenpkg.TypeRef("goa.Endpoint")),
					jen.Err().Error(),
				)
				g.BlockFunc(func(bg *jen.Group) {
					bg.Switch(jen.Id("svcn")).BlockFunc(func(sg *jen.Group) {
						for _, command := range commands {
							sg.Case(jen.Lit(command.Name)).BlockFunc(func(cg *jen.Group) {
								cg.Id("c").Op(":=").Id(command.PkgName).Dot("NewClient").Call(jen.Id("cc"), jen.Id("opts").Op("..."))
								cg.Switch(jen.Id("epn")).BlockFunc(func(eg *jen.Group) {
									for _, subcommand := range command.Subcommands {
										eg.Case(jen.Lit(subcommand.Name)).BlockFunc(func(scg *jen.Group) {
											scg.Id("endpoint").Op("=").Id("c").Dot(subcommand.MethodVarName).Call()
											if subcommand.Interceptors != nil {
												scg.Id("endpoint").Op("=").Id(subcommand.Interceptors.PkgName).Dot("Wrap"+subcommand.MethodVarName+"ClientEndpoint").Call(
													jen.Id("endpoint"),
													jen.Id(subcommand.Interceptors.VarName),
												)
											}
											switch {
											case subcommand.BuildFunction != nil:
												args := make([]jen.Code, 0, len(subcommand.BuildFunction.ActualParams))
												for _, param := range subcommand.BuildFunction.ActualParams {
													args = append(args, jen.Op("*").Id(param+"Flag"))
												}
												scg.List(jen.Id("data"), jen.Err()).Op("=").Id(command.PkgName).Dot(subcommand.BuildFunction.Name).Call(args...)
											case subcommand.Conversion != "":
												scg.Add(codegenpkg.Expr(subcommand.Conversion))
											}
										})
									}
								})
							})
						}
					})
				})
				g.If(jen.Err().Op("!=").Nil()).Block(
					jen.Return(jen.Nil(), jen.Nil(), jen.Err()),
				)
				g.Return(jen.Id("endpoint"), jen.Id("data"), jen.Nil())
			})
	})
}

func grpcRemoteMethodBuilderSection(endpoint *EndpointData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("remote-method-builder", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("Build%sFunc builds the remote method to invoke for %q service %q endpoint.", endpoint.Method.VarName, endpoint.ServiceName, endpoint.Method.Name))
		stmt.Func().Id("Build"+endpoint.Method.VarName+"Func").
			Params(
				jen.Id("grpccli").Add(codegenpkg.TypeRef(endpoint.PkgName+"."+endpoint.ClientInterface)),
				jen.Id("cliopts").Op("...").Qual("google.golang.org/grpc", "CallOption"),
			).
			Add(codegenpkg.TypeRef("goagrpc.RemoteFunc")).
			Block(
				jen.Return(
					jen.Func().
						Params(
							jen.Id("ctx").Qual("context", "Context"),
							jen.Id("reqpb").Any(),
							jen.Id("opts").Op("...").Qual("google.golang.org/grpc", "CallOption"),
						).
						Params(jen.Any(), jen.Error()).
						BlockFunc(func(g *jen.Group) {
							g.For(
								jen.List(jen.Id("_"), jen.Id("opt")).Op(":=").Range().Id("cliopts"),
							).Block(
								jen.Id("opts").Op("=").Append(jen.Id("opts"), jen.Id("opt")),
							)
							callArgs := []jen.Code{jen.Id("ctx")}
							if endpoint.Method.StreamingPayload == "" {
								callArgs = append(callArgs, jen.Id("reqpb").Assert(codegenpkg.Expr(endpoint.Request.ClientConvert.TgtRef)))
							}
							callArgs = append(callArgs, jen.Id("opts").Op("..."))
							g.If(jen.Id("reqpb").Op("!=").Nil()).Block(
								jen.Return(jen.Id("grpccli").Dot(endpoint.ClientMethodName).Call(callArgs...)),
							)
							nilArgs := []jen.Code{jen.Id("ctx")}
							if endpoint.Method.StreamingPayload == "" {
								nilArgs = append(nilArgs, jen.Op("&").Id(endpoint.Request.ClientConvert.TgtName).Values())
							}
							nilArgs = append(nilArgs, jen.Id("opts").Op("..."))
							g.Return(jen.Id("grpccli").Dot(endpoint.ClientMethodName).Call(nilArgs...))
						}),
				),
			)
	})
}

func grpcExampleServerSection(services []*ServiceData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("server-grpc-main", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, "handleGRPCServer starts configures and starts a gRPC server on the given URL. It shuts down the server if any error is received in the error channel.")
		params := []jen.Code{
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("u").Op("*").Qual("net/url", "URL"),
		}
		for _, service := range services {
			if len(service.Service.Methods) == 0 {
				continue
			}
			params = append(params, jen.Id(service.Service.VarName+"Endpoints").Op("*").Qual(service.Service.PkgName, "Endpoints"))
		}
		params = append(params,
			jen.Id("wg").Op("*").Qual("sync", "WaitGroup"),
			jen.Id("errc").Chan().Error(),
			jen.Id("dbg").Bool(),
		)
		stmt.Func().Id("handleGRPCServer").Params(params...).BlockFunc(func(g *jen.Group) {
			g.Comment(codegenpkg.Comment("Wrap the endpoints with the transport specific layers. The generated"))
			g.Comment(codegenpkg.Comment("server packages contains code generated from the design which maps"))
			g.Comment(codegenpkg.Comment("the service input and output data structures to gRPC requests and"))
			g.Comment(codegenpkg.Comment("responses."))
			g.Var().DefsFunc(func(defs *jen.Group) {
				for _, service := range services {
					defs.Id(service.Service.VarName + "Server").Op("*").Id(service.Service.PkgName + "svr").Dot("Server")
				}
			})
			g.BlockFunc(func(bg *jen.Group) {
				for _, service := range services {
					newArgs := []jen.Code{jen.Nil()}
					if len(service.Endpoints) > 0 {
						newArgs[0] = jen.Id(service.Service.VarName + "Endpoints")
					}
					if service.HasUnaryEndpoint() {
						newArgs = append(newArgs, jen.Nil())
					}
					if service.HasStreamingEndpoint() {
						newArgs = append(newArgs, jen.Nil())
					}
					bg.Id(service.Service.VarName + "Server").Op("=").Id(service.Service.PkgName + "svr").Dot("New").Call(newArgs...)
				}
			})
			g.Line()
			g.Comment(codegenpkg.Comment("Create interceptor which sets up the logger in each request context."))
			g.Id("chain").Op(":=").Qual("google.golang.org/grpc", "ChainUnaryInterceptor").Call(
				jen.Qual("goa.design/clue/log", "UnaryServerInterceptor").Call(jen.Id("ctx")),
			)
			g.If(jen.Id("dbg")).Block(
				jen.Comment(codegenpkg.Comment("Log request and response content if debug logs are enabled.")),
				jen.Id("chain").Op("=").Qual("google.golang.org/grpc", "ChainUnaryInterceptor").Call(
					jen.Qual("goa.design/clue/log", "UnaryServerInterceptor").Call(jen.Id("ctx")),
					jen.Qual("goa.design/clue/debug", "UnaryServerInterceptor").Call(),
				),
			)
			needStream := false
			for _, service := range services {
				if needStreamSection(service) {
					needStream = true
					break
				}
			}
			if needStream {
				g.Id("streamchain").Op(":=").Qual("google.golang.org/grpc", "ChainStreamInterceptor").Call(
					jen.Qual("goa.design/clue/log", "StreamServerInterceptor").Call(jen.Id("ctx")),
				)
				g.If(jen.Id("dbg")).Block(
					jen.Id("streamchain").Op("=").Qual("google.golang.org/grpc", "ChainStreamInterceptor").Call(
						jen.Qual("goa.design/clue/log", "StreamServerInterceptor").Call(jen.Id("ctx")),
						jen.Qual("goa.design/clue/debug", "StreamServerInterceptor").Call(),
					),
				)
			}
			g.Line()
			g.Comment(codegenpkg.Comment("Initialize gRPC server"))
			serverArgs := []jen.Code{jen.Id("chain")}
			if needStream {
				serverArgs = append(serverArgs, jen.Id("streamchain"))
			}
			g.Id("srv").Op(":=").Qual("google.golang.org/grpc", "NewServer").Call(serverArgs...)
			g.Line()
			g.Comment(codegenpkg.Comment("Register the servers."))
			for _, service := range services {
				g.Id(service.PkgName).Dot("Register"+codegenpkg.Goify(service.Service.VarName, true)+"Server").Call(
					jen.Id("srv"),
					jen.Id(service.Service.VarName+"Server"),
				)
			}
			g.Line()
			g.For(
				jen.List(jen.Id("svc"), jen.Id("info")).Op(":=").Range().Id("srv").Dot("GetServiceInfo").Call(),
			).Block(
				jen.For(jen.List(jen.Id("_"), jen.Id("m")).Op(":=").Range().Id("info").Dot("Methods")).Block(
					jen.Qual("goa.design/clue/log", "Printf").Call(
						jen.Id("ctx"),
						jen.Lit("serving gRPC method %s"),
						jen.Id("svc").Op("+").Lit("/").Op("+").Id("m").Dot("Name"),
					),
				),
			)
			g.Line()
			g.Comment(codegenpkg.Comment("Register the server reflection service on the server."))
			g.Comment(codegenpkg.Comment("See https://grpc.github.io/grpc/core/md_doc_server-reflection.html."))
			g.Qual("google.golang.org/grpc/reflection", "Register").Call(jen.Id("srv"))
			g.Line()
			g.Parens(jen.Op("*").Id("wg")).Dot("Add").Call(jen.Lit(1))
			g.Go().Func().Params().Block(
				jen.Defer().Parens(jen.Op("*").Id("wg")).Dot("Done").Call(),
				jen.Comment(codegenpkg.Comment("Start gRPC server in a separate goroutine.")),
				jen.Go().Func().Params().Block(
					jen.List(jen.Id("lis"), jen.Err()).Op(":=").Qual("net", "Listen").Call(jen.Lit("tcp"), jen.Id("u").Dot("Host")),
					jen.If(jen.Err().Op("!=").Nil()).Block(
						jen.Id("errc").Op("<-").Err(),
					),
					jen.If(jen.Id("lis").Op("==").Nil()).Block(
						jen.Id("errc").Op("<-").Qual("fmt", "Errorf").Call(jen.Lit("failed to listen on %q"), jen.Id("u").Dot("Host")),
					),
					jen.Qual("goa.design/clue/log", "Printf").Call(jen.Id("ctx"), jen.Lit("gRPC server listening on %q"), jen.Id("u").Dot("Host")),
					jen.Id("errc").Op("<-").Id("srv").Dot("Serve").Call(jen.Id("lis")),
				).Call(),
				jen.Op("<-").Id("ctx").Dot("Done").Call(),
				jen.Qual("goa.design/clue/log", "Printf").Call(jen.Id("ctx"), jen.Lit("shutting down gRPC server at %q"), jen.Id("u").Dot("Host")),
				jen.Id("srv").Dot("Stop").Call(),
			).Call()
		})
	})
}

func needStreamSection(service *ServiceData) bool {
	for _, endpoint := range service.Endpoints {
		if endpoint.ServerStream != nil || endpoint.ClientStream != nil {
			return true
		}
	}
	return false
}

func grpcStreamSections(endpoint *EndpointData, stream *StreamData, side string) []codegenpkg.Section {
	var sections []codegenpkg.Section
	sections = append(sections, grpcStreamStructSection(stream))
	if stream.SendConvert != nil && (side == "server" || endpoint.Method.StreamKind == expr.ClientStreamKind || endpoint.Method.StreamKind == expr.BidirectionalStreamKind) {
		sections = append(sections, grpcStreamSendSection(stream))
	}
	if stream.MustClose {
		sections = append(sections, grpcStreamCloseSection(stream))
	}
	if endpoint.Method.ViewedResult != nil && endpoint.Method.ViewedResult.ViewName == "" {
		sections = append(sections, grpcStreamSetViewSection(stream))
	}
	return sections
}
