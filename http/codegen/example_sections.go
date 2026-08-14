package codegen

import (
	"fmt"
	"slices"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func exampleCLIStartSection(
	services []exampleCLIServiceData,
	interceptorsPkg string,
	functionName string,
) codegen.Section {
	return codegen.NewRenderSection("cli-http-start", func() string {
		return renderExampleCLIStart(services, interceptorsPkg, functionName)
	})
}

func renderExampleCLIStart(services []exampleCLIServiceData, interceptorsPkg, functionName string) string {
	var b sourceBuilder
	b.Addf("func %s(scheme, host string, timeout int, debug bool) (loom.Endpoint, any, error) {\n", functionName)
	b.Add("\tvar (\n")
	b.Add("\t\tdoer loomhttp.Doer\n")
	for _, svc := range cliServicesWithClientInterceptors(services) {
		b.Addf("\t\t%sInterceptors %s.ClientInterceptors\n", svc.Data.Service.VarName, svc.ServiceImport)
	}
	b.Add("\t)\n")
	b.Add("\t{\n")
	b.Add("\t\tdoer = &http.Client{Timeout: time.Duration(timeout) * time.Second}\n")
	b.Add("\t\tif debug {\n")
	b.Add("\t\t\tdoer = loomhttp.NewDebugDoer(doer)\n")
	b.Add("\t\t}\n")
	for _, svc := range cliServicesWithClientInterceptors(services) {
		b.Addf("\t\t%sInterceptors = %s.New%sClientInterceptors()\n", svc.Data.Service.VarName, interceptorsPkg, svc.Data.Service.StructName)
	}
	b.Add("\t}\n")
	return b.String()
}

func exampleCLIStreamingSection(services []exampleCLIServiceData) codegen.Section {
	return codegen.NewRenderSection("cli-http-streaming", func() string {
		return renderExampleCLIStreaming(services)
	})
}

func renderExampleCLIStreaming(services []exampleCLIServiceData) string {
	if !exampleCLINeedsDialer(services) {
		return ""
	}
	return "\nvar (\n\tdialer *websocket.Dialer\n)\n{\n\tdialer = websocket.DefaultDialer\n}\n"
}

func exampleCLIEndSection(services []exampleCLIServiceData, apiPkg string) codegen.Section {
	return codegen.NewRenderSection("cli-http-end", func() string {
		return renderExampleCLIEnd(services, apiPkg)
	})
}

func renderExampleCLIEnd(services []exampleCLIServiceData, apiPkg string) string {
	var b sourceBuilder
	b.Add("\nendpoint, payload, err := cli.ParseEndpoint(\n")
	b.Add("\t\tscheme,\n")
	b.Add("\t\thost,\n")
	b.Add("\t\tdoer,\n")
	b.Add("\t\tloomhttp.RequestEncoder,\n")
	b.Add("\t\tloomhttp.ResponseDecoder,\n")
	b.Add("\t\tdebug,\n")
	if exampleCLINeedsDialer(services) {
		b.Add("\t\tdialer,\n")
		for _, svc := range services {
			if HasWebSocket(svc.Data) {
				b.Add("\t\tnil,\n")
			}
		}
	}
	for _, svc := range services {
		for _, endpoint := range svc.Data.Endpoints {
			if endpoint.MultipartRequestDecoder != nil {
				b.Addf("\t\t%s.%s,\n", apiPkg, endpoint.MultipartRequestEncoder.FuncName)
			}
		}
	}
	for _, svc := range cliServicesWithClientInterceptors(services) {
		b.Addf("\t\t%sInterceptors,\n", svc.Data.Service.VarName)
	}
	b.Add("\t)\n")
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn nil, nil, fmt.Errorf(\"parse endpoint: %w\", err)\n")
	b.Add("\t}\n")
	b.Add("\treturn endpoint, payload, nil\n")
	b.Add("}\n")
	return b.String()
}

func exampleCLIUsageSection(usagePrefix string) codegen.Section {
	return codegen.NewJenniferSection("cli-http-usage", func(stmt *jen.Statement) {
		stmt.Line()
		stmt.Func().Id(usagePrefix + "UsageCommands").Params().Index().String().Block(
			jen.Return(jen.Id("cli").Dot("UsageCommands").Call()),
		)
		stmt.Line()
		stmt.Line()
		stmt.Func().Id(usagePrefix + "UsageExamples").Params().String().Block(
			jen.Return(jen.Id("cli").Dot("UsageExamples").Call()),
		)
		stmt.Line()
	})
}

func exampleServerStartSection(services []exampleServerServiceData) codegen.Section {
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

func exampleServerConfigureSection(services []exampleServerServiceData, apiPkg string) codegen.Section {
	return codegen.NewRenderSection("server-http-init", func() string {
		return renderExampleServerConfigure(services, apiPkg)
	})
}

func renderExampleServerConfigure(services []exampleServerServiceData, apiPkg string) string {
	var b sourceBuilder
	b.Add("\n\t// Wrap the endpoints with the transport specific layers. The generated\n")
	b.Add("\t// server packages contains code generated from the design which maps\n")
	b.Add("\t// the service input and output data structures to HTTP requests and\n")
	b.Add("\t// responses.\n")
	b.Add("\tvar (\n")
	for _, svc := range services {
		b.Addf("\t\t%sServer *%s.Server\n", svc.Data.Service.VarName, svc.ServerImport)
	}
	b.Add("\t)\n")
	b.Add("\t{\n")
	b.Add("\t\teh := errorHandler(ctx)\n")
	if hasRuntimeCORS(services) {
		b.Add("\t\t// Replace this same-origin default with the deployment-configured browser origins.\n")
		b.Add("\t\truntimeCORSPolicy, err := loomhttp.NewRuntimeCORSPolicy(loomhttp.CORSPolicy{\n")
		b.Add("\t\t\tOrigins: []loomhttp.CORSOrigin{{Pattern: u.Scheme + \"://\" + u.Host}},\n")
		b.Add("\t\t})\n")
		b.Add("\t\tif err != nil {\n\t\t\tpanic(err)\n\t\t}\n")
	}
	if exampleServerNeedsDialer(services) {
		b.Add("\t\tupgrader := &websocket.Upgrader{}\n")
	}
	for _, svc := range services {
		b.Addf("\t\t%s\n", exampleServerConstructorCall(svc, apiPkg))
	}
	b.Add("\t}\n\n")
	b.Add("\t// Configure the mux.\n")
	for _, svc := range services {
		b.Addf("\t%s.Mount(mux, %sServer)\n", svc.ServerImport, svc.Data.Service.VarName)
	}
	return b.String()
}

func exampleServerMiddlewareSection() codegen.Section {
	return codegen.NewRenderSection("server-http-middleware", func() string {
		return "\n\tvar handler http.Handler = mux\n\tif dbg {\n\t\t// Log query and response bodies if debug logs are enabled.\n\t\thandler = debug.HTTP()(handler)\n\t}\n\thandler = log.HTTP(ctx)(handler)\n"
	})
}

func exampleServerEndSection(services []exampleServerServiceData) codegen.Section {
	return codegen.NewRenderSection("server-http-end", func() string {
		return renderExampleServerEnd(services)
	})
}

func renderExampleServerStart(services []exampleServerServiceData) string {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment("handleHTTPServer starts configures and starts a HTTP server on the given URL. It shuts down the server if any error is received in the error channel."))
	b.Add("\n")
	b.Add("func handleHTTPServer(ctx context.Context, u *url.URL")
	for _, svc := range services {
		if len(svc.Data.Service.Methods) > 0 {
			b.Addf(", %sEndpoints *%s.Endpoints", svc.Data.Service.VarName, svc.ServiceImport)
		}
	}
	b.Add(", wg *sync.WaitGroup, errc chan error, dbg bool) {\n")
	return b.String()
}

func renderExampleServerEnd(services []exampleServerServiceData) string {
	var b sourceBuilder
	b.Add("\n\t// Start HTTP server using default configuration, change the code to\n")
	b.Add("\t// configure the server as required by your service.\n")
	b.Add("\tsrv := &http.Server{\n")
	b.Add("\t\tAddr:              u.Host,\n")
	b.Add("\t\tHandler:           handler,\n")
	b.Add("\t\tReadHeaderTimeout: time.Second * 60,\n")
	b.Add("\t\tMaxHeaderValueCount: http.DefaultMaxHeaderValueCount,\n")
	b.Add("\t\tReadTimeout:       time.Second * 15,\n")
	if exampleServerHasStreamingEndpoint(services) {
		b.Add("\t\tWriteTimeout:      0,\n")
	} else {
		b.Add("\t\tWriteTimeout:      time.Second * 30,\n")
	}
	b.Add("\t\tIdleTimeout:       time.Second * 60,\n")
	b.Add("\t}\n")
	for _, svc := range services {
		b.Addf("\tfor _, m := range %sServer.Mounts {\n", svc.Data.Service.VarName)
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
	b.Add("\t\tshutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)\n")
	b.Add("\t\tdefer cancel()\n\n")
	b.Add("\t\terr := srv.Shutdown(shutdownCtx)\n")
	b.Add("\t\tif err != nil {\n")
	b.Add("\t\t\tlog.Printf(shutdownCtx, \"failed to shutdown: %v\", err)\n")
	b.Add("\t\t}\n")
	b.Add("\t}()\n}\n")
	return b.String()
}

func exampleServerErrorHandlerSection() codegen.Section {
	return codegen.NewJenniferSection("server-http-errorhandler", func(stmt *jen.Statement) {
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
	return codegen.NewJenniferSection("dummy-multipart-request-decoder", func(stmt *jen.Statement) {
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
	return codegen.NewJenniferSection("dummy-multipart-request-encoder", func(stmt *jen.Statement) {
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

func cliServicesWithClientInterceptors(services []exampleCLIServiceData) []exampleCLIServiceData {
	filtered := make([]exampleCLIServiceData, 0, len(services))
	for _, svc := range services {
		if len(svc.Data.Service.ClientInterceptors) == 0 {
			continue
		}
		filtered = append(filtered, svc)
	}
	return filtered
}

func exampleCLINeedsDialer(services []exampleCLIServiceData) bool {
	for _, service := range services {
		if HasWebSocket(service.Data) {
			return true
		}
	}
	return false
}

func exampleServerConstructorCall(svc exampleServerServiceData, apiPkg string) string {
	var b sourceBuilder
	b.Addf("%sServer = %s.New(", svc.Data.Service.VarName, svc.ServerImport)
	if len(svc.Data.Endpoints) > 0 {
		b.Addf("%sEndpoints", svc.Data.Service.VarName)
	} else {
		b.Add("nil")
	}
	b.Add(", mux, dec, enc, eh, nil")
	if svc.Data.CORS != nil && svc.Data.CORS.Runtime {
		b.Add(", runtimeCORSPolicy")
	}
	if HasWebSocket(svc.Data) {
		b.Add(", upgrader, nil")
	}
	for _, endpoint := range svc.Data.Endpoints {
		if endpoint.MultipartRequestDecoder != nil {
			b.Addf(", %s.%s", apiPkg, endpoint.MultipartRequestDecoder.FuncName)
		}
	}
	for range svc.Data.FileServers {
		b.Add(", nil")
	}
	b.Add(")")
	return b.String()
}

func hasRuntimeCORS(services []exampleServerServiceData) bool {
	for _, svc := range services {
		if svc.Data.CORS != nil && svc.Data.CORS.Runtime {
			return true
		}
	}
	return false
}

func exampleServerNeedsDialer(services []exampleServerServiceData) bool {
	for _, service := range services {
		if HasWebSocket(service.Data) {
			return true
		}
	}
	return false
}

func exampleServerHasStreamingEndpoint(services []exampleServerServiceData) bool {
	for _, service := range services {
		if HasWebSocket(service.Data) || slices.ContainsFunc(service.Data.Endpoints, IsSSEEndpoint) {
			return true
		}
	}
	return false
}
