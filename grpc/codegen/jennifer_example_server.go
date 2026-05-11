package codegen

import (
	"github.com/dave/jennifer/jen"

	codegenpkg "github.com/CaliLuke/loom/codegen"
)

func grpcExampleServerSection(services []*ServiceData) codegenpkg.Section {
	return codegenpkg.MustJenniferSection("server-grpc-main", func(stmt *jen.Statement) {
		codegenpkg.Doc(stmt, "handleGRPCServer starts configures and starts a gRPC server on the given URL. It shuts down the server if any error is received in the error channel.")
		needStream := hasStreamSection(services)
		stmt.Func().Id("handleGRPCServer").Params(grpcExampleServerParams(services)...).BlockFunc(func(g *jen.Group) {
			appendGRPCExampleServerInits(g, services)
			appendGRPCExampleInterceptorSetup(g, needStream)
			appendGRPCExampleServerSetup(g, services, needStream)
			appendGRPCExampleServerRun(g)
		})
	})
}

func grpcExampleServerParams(services []*ServiceData) []jen.Code {
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
	return append(params,
		jen.Id("wg").Op("*").Qual("sync", "WaitGroup"),
		jen.Id("errc").Chan().Error(),
		jen.Id("dbg").Bool(),
	)
}

func appendGRPCExampleServerInits(g *jen.Group, services []*ServiceData) {
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
			bg.Id(service.Service.VarName + "Server").Op("=").Id(service.Service.PkgName + "svr").Dot("New").Call(grpcExampleNewServerArgs(service)...)
		}
	})
	g.Line()
}

func grpcExampleNewServerArgs(service *ServiceData) []jen.Code {
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
	return newArgs
}

func appendGRPCExampleInterceptorSetup(g *jen.Group, needStream bool) {
	g.Comment(codegenpkg.Comment("Create interceptor which sets up the logger in each request context."))
	g.Id("chain").Op(":=").Qual("google.golang.org/grpc", "ChainUnaryInterceptor").Call(
		jen.Qual("github.com/CaliLuke/loom/clue/log", "UnaryServerInterceptor").Call(jen.Id("ctx")),
	)
	g.If(jen.Id("dbg")).Block(
		jen.Comment(codegenpkg.Comment("Log request and response content if debug logs are enabled.")),
		jen.Id("chain").Op("=").Qual("google.golang.org/grpc", "ChainUnaryInterceptor").Call(
			jen.Qual("github.com/CaliLuke/loom/clue/log", "UnaryServerInterceptor").Call(jen.Id("ctx")),
			jen.Qual("github.com/CaliLuke/loom/clue/debug", "UnaryServerInterceptor").Call(),
		),
	)
	if !needStream {
		g.Line()
		return
	}
	g.Id("streamchain").Op(":=").Qual("google.golang.org/grpc", "ChainStreamInterceptor").Call(
		jen.Qual("github.com/CaliLuke/loom/clue/log", "StreamServerInterceptor").Call(jen.Id("ctx")),
	)
	g.If(jen.Id("dbg")).Block(
		jen.Id("streamchain").Op("=").Qual("google.golang.org/grpc", "ChainStreamInterceptor").Call(
			jen.Qual("github.com/CaliLuke/loom/clue/log", "StreamServerInterceptor").Call(jen.Id("ctx")),
			jen.Qual("github.com/CaliLuke/loom/clue/debug", "StreamServerInterceptor").Call(),
		),
	)
	g.Line()
}

func appendGRPCExampleServerSetup(g *jen.Group, services []*ServiceData, needStream bool) {
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
			jen.Qual("github.com/CaliLuke/loom/clue/log", "Printf").Call(
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
}

func appendGRPCExampleServerRun(g *jen.Group) {
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
			jen.Qual("github.com/CaliLuke/loom/clue/log", "Printf").Call(jen.Id("ctx"), jen.Lit("gRPC server listening on %q"), jen.Id("u").Dot("Host")),
			jen.Id("errc").Op("<-").Id("srv").Dot("Serve").Call(jen.Id("lis")),
		).Call(),
		jen.Op("<-").Id("ctx").Dot("Done").Call(),
		jen.Qual("github.com/CaliLuke/loom/clue/log", "Printf").Call(jen.Id("ctx"), jen.Lit("shutting down gRPC server at %q"), jen.Id("u").Dot("Host")),
		jen.Id("srv").Dot("Stop").Call(),
	).Call()
}

func hasStreamSection(services []*ServiceData) bool {
	for _, service := range services {
		if needStreamSection(service) {
			return true
		}
	}
	return false
}

func needStreamSection(service *ServiceData) bool {
	for _, endpoint := range service.Endpoints {
		if endpoint.ServerStream != nil || endpoint.ClientStream != nil {
			return true
		}
	}
	return false
}
