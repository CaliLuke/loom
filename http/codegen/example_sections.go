package codegen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func exampleCLIStartSection(services []*ServiceData, interceptorsPkg string) codegen.Section {
	return codegen.NewRawSection("cli-http-start", renderExampleCLIStart(services, interceptorsPkg))
}

func renderExampleCLIStart(services []*ServiceData, interceptorsPkg string) string {
	var b strings.Builder
	b.WriteString("func doHTTP(scheme, host string, timeout int, debug bool) (goa.Endpoint, any, error) {\n")
	b.WriteString("\tvar (\n")
	b.WriteString("\t\tdoer goahttp.Doer\n")
	for _, svc := range servicesWithClientInterceptors(services) {
		fmt.Fprintf(&b, "\t\t%sInterceptors %s.ClientInterceptors\n", svc.Service.VarName, svc.Service.PkgName)
	}
	b.WriteString("\t)\n")
	b.WriteString("\t{\n")
	b.WriteString("\t\tdoer = &http.Client{Timeout: time.Duration(timeout) * time.Second}\n")
	b.WriteString("\t\tif debug {\n")
	b.WriteString("\t\t\tdoer = goahttp.NewDebugDoer(doer)\n")
	b.WriteString("\t\t}\n")
	for _, svc := range servicesWithClientInterceptors(services) {
		fmt.Fprintf(&b, "\t\t%sInterceptors = %s.New%sClientInterceptors()\n", svc.Service.VarName, interceptorsPkg, svc.Service.StructName)
	}
	b.WriteString("\t}\n")
	return b.String()
}

func exampleCLIStreamingSection(services []*ServiceData) codegen.Section {
	return codegen.NewRawSection("cli-http-streaming", renderExampleCLIStreaming(services))
}

func renderExampleCLIStreaming(services []*ServiceData) string {
	if !NeedDialer(services) {
		return ""
	}
	return "\nvar (\n\tdialer *websocket.Dialer\n)\n{\n\tdialer = websocket.DefaultDialer\n}\n"
}

func exampleCLIEndSection(services []*ServiceData, apiPkg string) codegen.Section {
	return codegen.NewRawSection("cli-http-end", renderExampleCLIEnd(services, apiPkg))
}

func renderExampleCLIEnd(services []*ServiceData, apiPkg string) string {
	var b strings.Builder
	b.WriteString("\nreturn cli.ParseEndpoint(\n")
	b.WriteString("\t\tscheme,\n")
	b.WriteString("\t\thost,\n")
	b.WriteString("\t\tdoer,\n")
	b.WriteString("\t\tgoahttp.RequestEncoder,\n")
	b.WriteString("\t\tgoahttp.ResponseDecoder,\n")
	b.WriteString("\t\tdebug,\n")
	if NeedDialer(services) {
		b.WriteString("\t\tdialer,\n")
		for _, svc := range services {
			if HasWebSocket(svc) {
				b.WriteString("\t\tnil,\n")
			}
		}
	}
	for _, svc := range services {
		for _, endpoint := range svc.Endpoints {
			if endpoint.MultipartRequestDecoder != nil {
				fmt.Fprintf(&b, "\t\t%s.%s,\n", apiPkg, endpoint.MultipartRequestEncoder.FuncName)
			}
		}
	}
	for _, svc := range servicesWithClientInterceptors(services) {
		fmt.Fprintf(&b, "\t\t%sInterceptors,\n", svc.Service.VarName)
	}
	b.WriteString("\t)\n}\n")
	return b.String()
}

func exampleCLIUsageSection() codegen.Section {
	return codegen.NewRawSection("cli-http-usage", "\nfunc httpUsageCommands() []string {\n\treturn cli.UsageCommands()\n}\n\nfunc httpUsageExamples() string {\n\treturn cli.UsageExamples()\n}\n")
}

func exampleServerStartSection(services []*ServiceData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment("handleHTTPServer starts configures and starts a HTTP server on the given URL. It shuts down the server if any error is received in the error channel."))
	b.WriteString("\n")
	b.WriteString("func handleHTTPServer(ctx context.Context, u *url.URL")
	for _, svc := range services {
		if len(svc.Service.Methods) > 0 {
			fmt.Fprintf(&b, ", %sEndpoints *%s.Endpoints", svc.Service.VarName, svc.Service.PkgName)
		}
	}
	b.WriteString(", wg *sync.WaitGroup, errc chan error, dbg bool) {\n")
	return codegen.NewRawSection("server-http-start", b.String())
}

func exampleServerEncodingSection() codegen.Section {
	return codegen.NewRawSection("server-http-encoding", "\n\t// Provide the transport specific request decoder and response encoder.\n\t// The goa http package has built-in support for JSON, XML and gob.\n\t// Other encodings can be used by providing the corresponding functions,\n\t// see goa.design/implement/encoding.\n\tvar (\n\t\tdec = goahttp.RequestDecoder\n\t\tenc = goahttp.ResponseEncoder\n\t)\n")
}

func exampleServerMuxSection() codegen.Section {
	return codegen.NewRawSection("server-http-mux", "\n\t// Build the service HTTP request multiplexer and mount debug and profiler\n\t// endpoints in debug mode.\n\tvar mux goahttp.Muxer\n\t{\n\t\tmux = goahttp.NewMuxer()\n\t\tif dbg {\n\t\t\t// Mount pprof handlers for memory profiling under /debug/pprof.\n\t\t\tdebug.MountPprofHandlers(debug.Adapt(mux))\n\t\t\t// Mount /debug endpoint to enable or disable debug logs at runtime.\n\t\t\tdebug.MountDebugLogEnabler(debug.Adapt(mux))\n\t\t}\n\t}\n")
}

func exampleServerConfigureSection(services []*ServiceData, apiPkg string) codegen.Section {
	return codegen.NewRawSection("server-http-init", renderExampleServerConfigure(services, apiPkg))
}

func renderExampleServerConfigure(services []*ServiceData, apiPkg string) string {
	var b strings.Builder
	b.WriteString("\n\t// Wrap the endpoints with the transport specific layers. The generated\n")
	b.WriteString("\t// server packages contains code generated from the design which maps\n")
	b.WriteString("\t// the service input and output data structures to HTTP requests and\n")
	b.WriteString("\t// responses.\n")
	b.WriteString("\tvar (\n")
	for _, svc := range services {
		fmt.Fprintf(&b, "\t\t%sServer *%ssvr.Server\n", svc.Service.VarName, svc.Service.PkgName)
	}
	b.WriteString("\t)\n")
	b.WriteString("\t{\n")
	b.WriteString("\t\teh := errorHandler(ctx)\n")
	if NeedDialer(services) {
		b.WriteString("\t\tupgrader := &websocket.Upgrader{}\n")
	}
	for _, svc := range services {
		fmt.Fprintf(&b, "\t\t%s\n", exampleServerConstructorCall(svc, apiPkg))
	}
	b.WriteString("\t}\n\n")
	b.WriteString("\t// Configure the mux.\n")
	for _, svc := range services {
		fmt.Fprintf(&b, "\t%ssvr.Mount(mux, %sServer)\n", svc.Service.PkgName, svc.Service.VarName)
	}
	return b.String()
}

func exampleServerMiddlewareSection() codegen.Section {
	return codegen.NewRawSection("server-http-middleware", "\n\tvar handler http.Handler = mux\n\tif dbg {\n\t\t// Log query and response bodies if debug logs are enabled.\n\t\thandler = debug.HTTP()(handler)\n\t}\n\thandler = log.HTTP(ctx)(handler)\n")
}

func exampleServerEndSection(services []*ServiceData) codegen.Section {
	return codegen.NewRawSection("server-http-end", renderExampleServerEnd(services))
}

func renderExampleServerEnd(services []*ServiceData) string {
	var b strings.Builder
	b.WriteString("\n\t// Start HTTP server using default configuration, change the code to\n")
	b.WriteString("\t// configure the server as required by your service.\n")
	b.WriteString("\tsrv := &http.Server{Addr: u.Host, Handler: handler, ReadHeaderTimeout: time.Second * 60}\n")
	for _, svc := range services {
		fmt.Fprintf(&b, "\tfor _, m := range %sServer.Mounts {\n", svc.Service.VarName)
		b.WriteString("\t\tlog.Printf(ctx, \"HTTP %q mounted on %s %s\", m.Method, m.Verb, m.Pattern)\n")
		b.WriteString("\t}\n")
	}
	b.WriteString("\n\t(*wg).Add(1)\n")
	b.WriteString("\tgo func() {\n")
	b.WriteString("\t\tdefer (*wg).Done()\n\n")
	b.WriteString("\t\t")
	b.WriteString(codegen.Comment("Start HTTP server in a separate goroutine."))
	b.WriteString("\n")
	b.WriteString("\t\tgo func() {\n")
	b.WriteString("\t\t\tlog.Printf(ctx, \"HTTP server listening on %q\", u.Host)\n")
	b.WriteString("\t\t\terrc <- srv.ListenAndServe()\n")
	b.WriteString("\t\t}()\n\n")
	b.WriteString("\t\t<-ctx.Done()\n")
	b.WriteString("\t\tlog.Printf(ctx, \"shutting down HTTP server at %q\", u.Host)\n\n")
	b.WriteString("\t\t")
	b.WriteString(codegen.Comment("Shutdown gracefully with a 30s timeout."))
	b.WriteString("\n")
	b.WriteString("\t\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n")
	b.WriteString("\t\tdefer cancel()\n\n")
	b.WriteString("\t\terr := srv.Shutdown(ctx)\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\tlog.Printf(ctx, \"failed to shutdown: %v\", err)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}()\n}\n")
	return b.String()
}

func exampleServerErrorHandlerSection() codegen.Section {
	return codegen.NewRawSection("server-http-errorhandler", "\n// errorHandler returns a function that writes and logs the given error.\n// The function also writes and logs the error unique ID so that it's possible\n// to correlate.\nfunc errorHandler(logCtx context.Context) func(context.Context, http.ResponseWriter, error) {\n\treturn func(ctx context.Context, w http.ResponseWriter, err error) {\n\t\tlog.Printf(logCtx, \"ERROR: %s\", err.Error())\n\t}\n}\n")
}

func dummyMultipartRequestDecoderSection(data *MultipartData) codegen.Section {
	return codegen.NewRawSection("dummy-multipart-request-decoder", renderDummyMultipartRequestDecoder(data))
}

func renderDummyMultipartRequestDecoder(data *MultipartData) string {
	return fmt.Sprintf("\n%s\nfunc %s(mr *multipart.Reader, p *%s) error {\n\t// Add multipart request decoder logic here\n\treturn nil\n}\n",
		codegen.Comment(fmt.Sprintf("%s implements the multipart decoder for service %q endpoint %q. The decoder must populate the argument p after encoding.", data.FuncName, data.ServiceName, data.MethodName)),
		data.FuncName,
		data.Payload.Ref,
	)
}

func dummyMultipartRequestEncoderSection(data *MultipartData) codegen.Section {
	return codegen.NewRawSection("dummy-multipart-request-encoder", renderDummyMultipartRequestEncoder(data))
}

func renderDummyMultipartRequestEncoder(data *MultipartData) string {
	return fmt.Sprintf("\n%s\nfunc %s(mw *multipart.Writer, p %s) error {\n\t// Add multipart request encoder logic here\n\treturn nil\n}\n",
		codegen.Comment(fmt.Sprintf("%s implements the multipart encoder for service %q endpoint %q.", data.FuncName, data.ServiceName, data.MethodName)),
		data.FuncName,
		data.Payload.Ref,
	)
}

func renderFileServerBasePath(filePath string) string {
	return "/" + filepath.Base(filePath)
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
	var b strings.Builder
	fmt.Fprintf(&b, "%sServer = %ssvr.New(", svc.Service.VarName, svc.Service.PkgName)
	if len(svc.Endpoints) > 0 {
		fmt.Fprintf(&b, "%sEndpoints", svc.Service.VarName)
	} else {
		b.WriteString("nil")
	}
	b.WriteString(", mux, dec, enc, eh, nil")
	if HasWebSocket(svc) {
		b.WriteString(", upgrader, nil")
	}
	for _, endpoint := range svc.Endpoints {
		if endpoint.MultipartRequestDecoder != nil {
			fmt.Fprintf(&b, ", %s.%s", apiPkg, endpoint.MultipartRequestDecoder.FuncName)
		}
	}
	for range svc.FileServers {
		b.WriteString(", nil")
	}
	b.WriteString(")")
	return b.String()
}
