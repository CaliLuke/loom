package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	codegenpkg "github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
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
			Add(codegenpkg.TypeRef("loom.Endpoint")).
			Block(
				jen.Return(
					jen.Func().
						Params(
							jen.Id("ctx").Qual("context", "Context"),
							jen.Id("v").Any(),
						).
						Params(jen.Any(), jen.Error()).
						BlockFunc(func(g *jen.Group) {
							writeGRPCClientEndpointInvoker(g, endpoint)
							writeGRPCClientEndpointErrorHandling(g, endpoint)
							g.Return(jen.Id("res"), jen.Nil())
						}),
				),
			)
	})
}

func writeGRPCClientEndpointInvoker(g *jen.Group, endpoint *EndpointData) {
	g.Id("inv").Op(":=").Add(codegenpkg.Expr("loomgrpc.NewInvoker")).Call(
		jen.Id("Build"+endpoint.Method.VarName+"Func").Call(jen.Id("c").Dot("grpccli"), jen.Id("c").Dot("opts").Op("...")),
		grpcClientEndpointEncodeFn(endpoint),
		grpcClientEndpointDecodeFn(endpoint),
	)
	g.List(jen.Id("res"), jen.Err()).Op(":=").Id("inv").Dot("Invoke").Call(jen.Id("ctx"), jen.Id("v"))
}

func grpcClientEndpointEncodeFn(endpoint *EndpointData) *jen.Statement {
	if endpoint.PayloadRef == "" {
		return jen.Id("nil")
	}
	return jen.Id("Encode" + endpoint.Method.VarName + "Request")
}

func grpcClientEndpointDecodeFn(endpoint *EndpointData) *jen.Statement {
	if endpoint.ResultRef == "" && endpoint.ClientStream == nil {
		return jen.Id("nil")
	}
	return jen.Id("Decode" + endpoint.Method.VarName + "Response")
}

func writeGRPCClientEndpointErrorHandling(g *jen.Group, endpoint *EndpointData) {
	g.If(jen.Err().Op("!=").Nil()).BlockFunc(func(eg *jen.Group) {
		eg.Id("resp").Op(":=").Add(codegenpkg.Expr("loomgrpc.DecodeError")).Call(jen.Err())
		if len(endpoint.Errors) > 0 {
			writeGRPCClientEndpointTypedErrors(eg, endpoint)
			return
		}
		writeGRPCClientEndpointFallbackError(eg)
	})
}

func writeGRPCClientEndpointTypedErrors(eg *jen.Group, endpoint *EndpointData) {
	eg.Switch(jen.Id("message").Op(":=").Id("resp").Assert(jen.Type())).BlockFunc(func(sg *jen.Group) {
		for _, errData := range endpoint.Errors {
			if errData.Response.ClientConvert == nil {
				continue
			}
			sg.Case(codegenpkg.Expr(errData.Response.ClientConvert.SrcRef)).Block(grpcClientEndpointErrorCaseBody(errData)...)
		}
		sg.Case(jen.Op("*").Id("loompb").Dot("ErrorResponse")).Block(
			jen.Return(
				jen.Nil(),
				codegenpkg.Expr("loomgrpc.NewServiceError").Call(jen.Id("message")),
			),
		)
		sg.Default().Block(
			jen.Return(
				jen.Nil(),
				codegenpkg.Expr("loom.Fault").Call(jen.Lit("%s"), jen.Err().Dot("Error").Call()),
			),
		)
	})
}

func grpcClientEndpointErrorCaseBody(errData *ErrorData) []jen.Code {
	caseBody := make([]jen.Code, 0, 2)
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
	caseBody = append(caseBody,
		jen.Return(
			jen.Nil(),
			jen.Id(errData.Response.ClientConvert.Init.Name).Call(grpcClientEndpointInitArgs(errData)...),
		),
	)
	return caseBody
}

func grpcClientEndpointInitArgs(errData *ErrorData) []jen.Code {
	initArgs := make([]jen.Code, 0, len(errData.Response.ClientConvert.Init.Args))
	for _, arg := range errData.Response.ClientConvert.Init.Args {
		initArgs = append(initArgs, codegenpkg.Expr(arg.Name))
	}
	return initArgs
}

func writeGRPCClientEndpointFallbackError(eg *jen.Group) {
	eg.Comment(codegenpkg.Comment("Try to decode a Loom error response detail before falling back to Fault."))
	eg.If(
		jen.List(jen.Id("eresp"), jen.Id("ok")).Op(":=").Id("resp").Assert(jen.Op("*").Id("loompb").Dot("ErrorResponse")),
		jen.Id("ok"),
	).Block(
		jen.Return(
			jen.Nil(),
			codegenpkg.Expr("loomgrpc.NewServiceError").Call(jen.Id("eresp")),
		),
	)
	eg.Return(
		jen.Nil(),
		codegenpkg.Expr("loom.Fault").Call(jen.Lit("%s"), jen.Err().Dot("Error").Call()),
	)
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
			fields = append(fields, jen.Id(endpoint.Method.VarName+"H").Add(codegenpkg.TypeRef("loomgrpc."+handlerType)))
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
			params = append(params, jen.Id("uh").Add(codegenpkg.TypeRef("loomgrpc.UnaryHandler")))
		}
		if data.HasStreamingEndpoint() {
			params = append(params, jen.Id("sh").Add(codegenpkg.TypeRef("loomgrpc.StreamHandler")))
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
				jen.Id("endpoint").Add(codegenpkg.TypeRef("loom.Endpoint")),
				jen.Id("h").Add(codegenpkg.TypeRef("loomgrpc."+handlerKind+"Handler")),
			).
			Add(codegenpkg.TypeRef("loomgrpc."+handlerKind+"Handler")).
			Block(
				jen.If(jen.Id("h").Op("==").Nil()).Block(
					jen.Id("h").Op("=").Add(codegenpkg.TypeRef("loomgrpc.New"+handlerKind+"Handler")).Call(
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
		params := grpcServerInterfaceParams(endpoint)
		results := grpcServerInterfaceResults(endpoint)
		stmt.Func().Params(jen.Id("s").Op("*").Id(endpoint.ServerStruct)).
			Id(endpoint.Method.VarName).
			Params(params...).
			Params(results...).
			BlockFunc(func(g *jen.Group) {
				addGRPCServerContext(g, endpoint)
				addGRPCServerHandleCall(g, endpoint)
				appendGRPCServerErrorHandler(g, endpoint, endpoint.ServerStream != nil)
				addGRPCServerReturn(g, endpoint)
			})
	})
}

func grpcServerInterfaceParams(endpoint *EndpointData) []jen.Code {
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
	return params
}

func grpcServerInterfaceResults(endpoint *EndpointData) []jen.Code {
	if endpoint.ServerStream == nil && endpoint.Response.Message != nil {
		return []jen.Code{codegenpkg.TypeRef(endpoint.Response.Message.Ref), jen.Error()}
	}
	return []jen.Code{jen.Error()}
}

func addGRPCServerContext(g *jen.Group, endpoint *EndpointData) {
	if endpoint.ServerStream != nil {
		g.Id("ctx").Op(":=").Id("stream").Dot("Context").Call()
	}
	g.Id("ctx").Op("=").Qual("context", "WithValue").Call(jen.Id("ctx"), codegenpkg.Expr("loom.MethodKey"), jen.Lit(endpoint.Method.Name))
	g.Id("ctx").Op("=").Qual("context", "WithValue").Call(jen.Id("ctx"), codegenpkg.Expr("loom.ServiceKey"), jen.Lit(endpoint.ServiceName))
}

func addGRPCServerHandleCall(g *jen.Group, endpoint *EndpointData) {
	if endpoint.ServerStream != nil {
		addGRPCStreamHandleCall(g, endpoint)
		return
	}
	g.List(jen.Id("resp"), jen.Err()).Op(":=").Id("s").Dot(endpoint.Method.VarName+"H").Dot("Handle").Call(jen.Id("ctx"), jen.Id("message"))
}

func addGRPCStreamHandleCall(g *jen.Group, endpoint *EndpointData) {
	decodeTarget := "_"
	if endpoint.PayloadRef != "" {
		decodeTarget = "p"
	}
	var decodeArg jen.Code
	switch {
	case endpoint.Request.StreamEnvelope != nil:
		g.Var().Id("reqpb").Any()
		g.List(jen.Id("message"), jen.Err()).Op(":=").Id("stream").Dot("Recv").Call()
		g.If(jen.Err().Op("!=").Nil()).Block(
			jen.If(jen.Qual("errors", "Is").Call(jen.Err(), jen.Qual("io", "EOF"))).Block(
				jen.Id("reqpb").Op("=").Nil(),
			).Else().Block(
				jen.Return(codegenpkg.Expr("loomgrpc.EncodeError").Call(jen.Err())),
			),
		).Else().Block(
			jen.Id("reqpb").Op("=").Id("message"),
		)
		decodeArg = jen.Id("reqpb")
	case endpoint.Method.StreamingPayload != "":
		decodeArg = jen.Nil()
	default:
		decodeArg = jen.Id("message")
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
}

func addGRPCServerReturn(g *jen.Group, endpoint *EndpointData) {
	if endpoint.ServerStream == nil {
		g.Return(jen.Id("resp").Assert(codegenpkg.Expr(endpoint.Response.ServerConvert.TgtRef)), jen.Nil())
		return
	}
	g.Return(jen.Nil())
}

func grpcExampleCLISection(defaultTransportType string, services []*ServiceData, interceptorsPkg string) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("do-grpc-cli", func(stmt *jen.Statement) {
		stmt.Func().Id("doGRPC").Params(
			jen.List(jen.Id("_"), jen.Id("host")).String(),
			jen.Id("_").Int(),
			jen.Id("_").Bool(),
		).Params(codegenpkg.TypeRef("loom.Endpoint"), jen.Any(), jen.Error()).BlockFunc(func(g *jen.Group) {
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

func grpcParseEndpointSection(commands []*cli.CommandData) codegenpkg.Section {
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
			Params(codegenpkg.TypeRef("loom.Endpoint"), jen.Any(), jen.Error()).
			BlockFunc(func(g *jen.Group) {
				g.Add(cli.FlagsCodeStatement(commands))
				g.Var().Defs(
					jen.Id("data").Any(),
					jen.Id("endpoint").Add(codegenpkg.TypeRef("loom.Endpoint")),
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
											case subcommand.Conversion != nil:
												scg.Add(subcommand.Conversion)
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
			Add(codegenpkg.TypeRef("loomgrpc.RemoteFunc")).
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
							if endpoint.Request.StreamEnvelope != nil {
								g.List(jen.Id("stream"), jen.Err()).Op(":=").Id("grpccli").Dot(endpoint.ClientMethodName).Call(jen.Id("ctx"), jen.Id("opts").Op("..."))
								g.If(jen.Err().Op("!=").Nil()).Block(
									jen.Return(jen.Nil(), jen.Err()),
								)
								g.If(jen.Id("reqpb").Op("!=").Nil()).Block(
									jen.If(
										jen.Err().Op(":=").Id("stream").Dot("Send").Call(jen.Id("reqpb").Assert(codegenpkg.Expr(endpoint.Request.Message.Ref))),
										jen.Err().Op("!=").Nil(),
									).Block(
										jen.Return(jen.Nil(), jen.Err()),
									),
								)
								g.Return(jen.Id("stream"), jen.Nil())
								return
							}
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
