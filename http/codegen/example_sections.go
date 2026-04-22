package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func exampleCLIStartSection(services []*ServiceData, interceptorsPkg string) codegen.Section {
	return codegen.NewRenderSection("cli-http-start", func() string {
		return renderExampleCLIStart(services, interceptorsPkg)
	})
}

func renderExampleCLIStart(services []*ServiceData, interceptorsPkg string) string {
	var b sourceBuilder
	b.Add("func doHTTP(scheme, host string, timeout int, debug bool) (loom.Endpoint, any, error) {\n")
	b.Add("\tvar (\n")
	b.Add("\t\tdoer loomhttp.Doer\n")
	for _, svc := range servicesWithClientInterceptors(services) {
		b.Addf("\t\t%sInterceptors %s.ClientInterceptors\n", svc.Service.VarName, svc.Service.PkgName)
	}
	b.Add("\t)\n")
	b.Add("\t{\n")
	b.Add("\t\tdoer = &http.Client{Timeout: time.Duration(timeout) * time.Second}\n")
	b.Add("\t\tif debug {\n")
	b.Add("\t\t\tdoer = loomhttp.NewDebugDoer(doer)\n")
	b.Add("\t\t}\n")
	for _, svc := range servicesWithClientInterceptors(services) {
		b.Addf("\t\t%sInterceptors = %s.New%sClientInterceptors()\n", svc.Service.VarName, interceptorsPkg, svc.Service.StructName)
	}
	b.Add("\t}\n")
	return b.String()
}

func exampleCLIStreamingSection(services []*ServiceData) codegen.Section {
	return codegen.NewRenderSection("cli-http-streaming", func() string {
		return renderExampleCLIStreaming(services)
	})
}

func renderExampleCLIStreaming(services []*ServiceData) string {
	if !NeedDialer(services) {
		return ""
	}
	return "\nvar (\n\tdialer *websocket.Dialer\n)\n{\n\tdialer = websocket.DefaultDialer\n}\n"
}

func exampleCLIEndSection(services []*ServiceData, apiPkg string) codegen.Section {
	return codegen.NewRenderSection("cli-http-end", func() string {
		return renderExampleCLIEnd(services, apiPkg)
	})
}

func renderExampleCLIEnd(services []*ServiceData, apiPkg string) string {
	var b sourceBuilder
	b.Add("\nreturn cli.ParseEndpoint(\n")
	b.Add("\t\tscheme,\n")
	b.Add("\t\thost,\n")
	b.Add("\t\tdoer,\n")
	b.Add("\t\tloomhttp.RequestEncoder,\n")
	b.Add("\t\tloomhttp.ResponseDecoder,\n")
	b.Add("\t\tdebug,\n")
	if NeedDialer(services) {
		b.Add("\t\tdialer,\n")
		for _, svc := range services {
			if HasWebSocket(svc) {
				b.Add("\t\tnil,\n")
			}
		}
	}
	for _, svc := range services {
		for _, endpoint := range svc.Endpoints {
			if endpoint.MultipartRequestDecoder != nil {
				b.Addf("\t\t%s.%s,\n", apiPkg, endpoint.MultipartRequestEncoder.FuncName)
			}
		}
	}
	for _, svc := range servicesWithClientInterceptors(services) {
		b.Addf("\t\t%sInterceptors,\n", svc.Service.VarName)
	}
	b.Add("\t)\n}\n")
	return b.String()
}

func exampleCLIUsageSection() codegen.Section {
	return codegen.MustJenniferSection("cli-http-usage", func(stmt *jen.Statement) {
		stmt.Line()
		stmt.Func().Id("httpUsageCommands").Params().Index().String().Block(
			jen.Return(jen.Id("cli").Dot("UsageCommands").Call()),
		)
		stmt.Line()
		stmt.Line()
		stmt.Func().Id("httpUsageExamples").Params().String().Block(
			jen.Return(jen.Id("cli").Dot("UsageExamples").Call()),
		)
		stmt.Line()
	})
}

func exampleServerStartSection(services []*ServiceData) codegen.Section {
	return codegen.NewRenderSection("server-http-start", func() string {
		return renderExampleServerStart(services)
	})
}

func exampleServerEncodingSection() codegen.Section {
	return codegen.NewRenderSection("server-http-encoding", func() string {
		return "\n\t// Provide the transport specific request decoder and response encoder.\n\t// The Loom http package has built-in support for JSON, XML and gob.\n\t// Other encodings can be used by providing the corresponding functions.\n\tvar (\n\t\tdec = loomhttp.RequestDecoder\n\t\tenc = loomhttp.ResponseEncoder\n\t)\n"
	})
}

func exampleServerMuxSection() codegen.Section {
	return codegen.NewRenderSection("server-http-mux", func() string {
		return "\n\t// Build the service HTTP request multiplexer and mount debug and profiler\n\t// endpoints in debug mode.\n\tvar mux loomhttp.Muxer\n\t{\n\t\tmux = loomhttp.NewMuxer()\n\t\tif dbg {\n\t\t\t// Mount pprof handlers for memory profiling under /debug/pprof.\n\t\t\tdebug.MountPprofHandlers(debug.Adapt(mux))\n\t\t\t// Mount /debug endpoint to enable or disable debug logs at runtime.\n\t\t\tdebug.MountDebugLogEnabler(debug.Adapt(mux))\n\t\t}\n\t}\n"
	})
}

func exampleServerConfigureSection(services []*ServiceData, apiPkg string) codegen.Section {
	return codegen.NewRenderSection("server-http-init", func() string {
		return renderExampleServerConfigure(services, apiPkg)
	})
}

func renderExampleServerConfigure(services []*ServiceData, apiPkg string) string {
	var b sourceBuilder
	b.Add("\n\t// Wrap the endpoints with the transport specific layers. The generated\n")
	b.Add("\t// server packages contains code generated from the design which maps\n")
	b.Add("\t// the service input and output data structures to HTTP requests and\n")
	b.Add("\t// responses.\n")
	b.Add("\tvar (\n")
	for _, svc := range services {
		b.Addf("\t\t%sServer *%ssvr.Server\n", svc.Service.VarName, svc.Service.PkgName)
	}
	b.Add("\t)\n")
	b.Add("\t{\n")
	b.Add("\t\teh := errorHandler(ctx)\n")
	if NeedDialer(services) {
		b.Add("\t\tupgrader := &websocket.Upgrader{}\n")
	}
	for _, svc := range services {
		b.Addf("\t\t%s\n", exampleServerConstructorCall(svc, apiPkg))
	}
	b.Add("\t}\n\n")
	b.Add("\t// Configure the mux.\n")
	for _, svc := range services {
		b.Addf("\t%ssvr.Mount(mux, %sServer)\n", svc.Service.PkgName, svc.Service.VarName)
	}
	return b.String()
}

func exampleServerMiddlewareSection() codegen.Section {
	return codegen.NewRenderSection("server-http-middleware", func() string {
		return "\n\tvar handler http.Handler = mux\n\tif dbg {\n\t\t// Log query and response bodies if debug logs are enabled.\n\t\thandler = debug.HTTP()(handler)\n\t}\n\thandler = log.HTTP(ctx)(handler)\n"
	})
}

func exampleServerEndSection(services []*ServiceData) codegen.Section {
	return codegen.NewRenderSection("server-http-end", func() string {
		return renderExampleServerEnd(services)
	})
}

func renderExampleServerStart(services []*ServiceData) string {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment("handleHTTPServer starts configures and starts a HTTP server on the given URL. It shuts down the server if any error is received in the error channel."))
	b.Add("\n")
	b.Add("func handleHTTPServer(ctx context.Context, u *url.URL")
	for _, svc := range services {
		if len(svc.Service.Methods) > 0 {
			b.Addf(", %sEndpoints *%s.Endpoints", svc.Service.VarName, svc.Service.PkgName)
		}
	}
	b.Add(", wg *sync.WaitGroup, errc chan error, dbg bool) {\n")
	return b.String()
}

func renderExampleServerEnd(services []*ServiceData) string {
	var b sourceBuilder
	b.Add("\n\t// Start HTTP server using default configuration, change the code to\n")
	b.Add("\t// configure the server as required by your service.\n")
	b.Add("\tsrv := &http.Server{\n")
	b.Add("\t\tAddr:              u.Host,\n")
	b.Add("\t\tHandler:           handler,\n")
	b.Add("\t\tReadHeaderTimeout: time.Second * 60,\n")
	b.Add("\t\tReadTimeout:       time.Second * 15,\n")
	b.Add("\t\tWriteTimeout:      time.Second * 30,\n")
	b.Add("\t\tIdleTimeout:       time.Second * 60,\n")
	b.Add("\t}\n")
	for _, svc := range services {
		b.Addf("\tfor _, m := range %sServer.Mounts {\n", svc.Service.VarName)
		b.Add("\t\tlog.Printf(ctx, \"HTTP %q mounted on %s %s\", m.Method, m.Verb, m.Pattern)\n")
		b.Add("\t}\n")
	}
	b.Add("\n\t(*wg).Add(1)\n")
	b.Add("\tgo func() {\n")
	b.Add("\t\tdefer (*wg).Done()\n\n")
	b.Add("\t\t")
	b.Add(codegen.Comment("Start HTTP server in a separate goroutine."))
	b.Add("\n")
	b.Add("\t\tgo func() {\n")
	b.Add("\t\t\tlog.Printf(ctx, \"HTTP server listening on %q\", u.Host)\n")
	b.Add("\t\t\terrc <- srv.ListenAndServe()\n")
	b.Add("\t\t}()\n\n")
	b.Add("\t\t<-ctx.Done()\n")
	b.Add("\t\tlog.Printf(ctx, \"shutting down HTTP server at %q\", u.Host)\n\n")
	b.Add("\t\t")
	b.Add(codegen.Comment("Shutdown gracefully with a 30s timeout."))
	b.Add("\n")
	b.Add("\t\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n")
	b.Add("\t\tdefer cancel()\n\n")
	b.Add("\t\terr := srv.Shutdown(ctx)\n")
	b.Add("\t\tif err != nil {\n")
	b.Add("\t\t\tlog.Printf(ctx, \"failed to shutdown: %v\", err)\n")
	b.Add("\t\t}\n")
	b.Add("\t}()\n}\n")
	return b.String()
}

func exampleServerErrorHandlerSection() codegen.Section {
	return codegen.MustJenniferSection("server-http-errorhandler", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.CommentBlock(stmt, "errorHandler returns a function that writes and logs the given error.\nThe function also writes and logs the error unique ID so that it's possible\nto correlate.")
		stmt.Func().
			Id("errorHandler").
			Params(jen.Id("logCtx").Qual("context", "Context")).
			Func().
			Params(
				jen.Qual("context", "Context"),
				jen.Qual("net/http", "ResponseWriter"),
				jen.Error(),
			).
			Block(
				jen.Return(
					jen.Func().
						Params(
							jen.Id("ctx").Qual("context", "Context"),
							jen.Id("w").Qual("net/http", "ResponseWriter"),
							jen.Id("err").Error(),
						).
						Block(
							jen.Id("log").Dot("Printf").Call(jen.Id("logCtx"), jen.Lit("ERROR: %s"), jen.Id("err").Dot("Error").Call()),
						),
				),
			)
		stmt.Line()
	})
}

func dummyMultipartRequestDecoderSection(data *MultipartData) codegen.Section {
	return codegen.MustJenniferSection("dummy-multipart-request-decoder", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s implements the multipart decoder for service %q endpoint %q. The decoder must populate the argument p after encoding.", data.FuncName, data.ServiceName, data.MethodName))
		stmt.Func().
			Id(data.FuncName).
			Params(
				jen.Id("mr").Op("*").Qual("mime/multipart", "Reader"),
				jen.Id("p").Op("*").Add(codegen.TypeRef(data.Payload.Ref)),
			).
			Params(jen.Error()).
			Block(
				jen.Comment("Add multipart request decoder logic here").Line(),
				jen.Return(jen.Nil()),
			)
		stmt.Line()
	})
}

func dummyMultipartRequestEncoderSection(data *MultipartData) codegen.Section {
	return codegen.MustJenniferSection("dummy-multipart-request-encoder", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s implements the multipart encoder for service %q endpoint %q.", data.FuncName, data.ServiceName, data.MethodName))
		stmt.Func().
			Id(data.FuncName).
			Params(
				jen.Id("mw").Op("*").Qual("mime/multipart", "Writer"),
				jen.Id("p").Add(codegen.TypeRef(data.Payload.Ref)),
			).
			Params(jen.Error()).
			Block(
				jen.Comment("Add multipart request encoder logic here").Line(),
				jen.Return(jen.Nil()),
			)
		stmt.Line()
	})
}

func servicesWithClientInterceptors(services []*ServiceData) []*ServiceData {
	filtered := make([]*ServiceData, 0, len(services))
	for _, svc := range services {
		if len(svc.Service.ClientInterceptors) == 0 {
			continue
		}
		filtered = append(filtered, svc)
	}
	return filtered
}

func exampleServerConstructorCall(svc *ServiceData, apiPkg string) string {
	var b sourceBuilder
	b.Addf("%sServer = %ssvr.New(", svc.Service.VarName, svc.Service.PkgName)
	if len(svc.Endpoints) > 0 {
		b.Addf("%sEndpoints", svc.Service.VarName)
	} else {
		b.Add("nil")
	}
	b.Add(", mux, dec, enc, eh, nil")
	if HasWebSocket(svc) {
		b.Add(", upgrader, nil")
	}
	for _, endpoint := range svc.Endpoints {
		if endpoint.MultipartRequestDecoder != nil {
			b.Addf(", %s.%s", apiPkg, endpoint.MultipartRequestDecoder.FuncName)
		}
	}
	for range svc.FileServers {
		b.Add(", nil")
	}
	b.Add(")")
	return b.String()
}
