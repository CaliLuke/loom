package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	codegenpkg "goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
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

func grpcServerStructSection(data *ServiceData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("server-struct", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, fmt.Sprintf("%s implements the %s.%s interface.", data.ServerStruct, data.PkgName, data.ServerInterface))
		fields := make([]jen.Code, 0, len(data.Endpoints)+1)
		for _, endpoint := range data.Endpoints {
			handlerType := "UnaryHandler"
			if endpoint.ServerStream != nil {
				handlerType = "StreamHandler"
			}
			fields = append(fields, jen.Id(endpoint.Method.VarName+"H").Qual("goa.design/goa/v3/grpc", handlerType))
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
			params = append(params, jen.Id("uh").Qual("goa.design/goa/v3/grpc", "UnaryHandler"))
		}
		if data.HasStreamingEndpoint() {
			params = append(params, jen.Id("sh").Qual("goa.design/goa/v3/grpc", "StreamHandler"))
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

func grpcStreamCloseSection(stream *StreamData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection(stream.Type+"-stream-close", func(stmt *jen.Statement) {
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
