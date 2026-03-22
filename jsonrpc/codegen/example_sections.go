package codegen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func renderSectionSource(section codegen.Section) string {
	var b bytes.Buffer
	if err := section.Write(&b); err != nil {
		panic(err)
	}
	return b.String()
}

func jsonrpcExampleServerStartSource(httpServices, jsonrpcServices []*httpcodegen.ServiceData) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment("handleHTTPServer starts configures and starts a HTTP server on the given URL. It shuts down the server if any error is received in the error channel."))
	b.WriteString("\n")
	b.WriteString("func handleHTTPServer(ctx context.Context, u *url.URL")
	for _, svc := range httpServices {
		if len(svc.Service.Methods) > 0 {
			fmt.Fprintf(&b, ", %sEndpoints *%s.Endpoints", svc.Service.VarName, svc.Service.PkgName)
		}
	}
	for _, svc := range jsonrpcServices {
		if !hasServiceName(httpServices, svc.Service.Name) {
			fmt.Fprintf(&b, ", %sEndpoints *%s.Endpoints", svc.Service.VarName, svc.Service.PkgName)
		}
	}
	for _, svc := range jsonrpcServices {
		fmt.Fprintf(&b, ", %sSvc %s.Service", svc.Service.VarName, svc.Service.PkgName)
	}
	b.WriteString(", wg *sync.WaitGroup, errc chan error, dbg bool) {\n")
	return b.String()
}

func jsonrpcExampleServerConfigureSource(httpServices, jsonrpcServices []*httpcodegen.ServiceData, apiPkg string) string {
	var b strings.Builder
	b.WriteString("\n\t// Wrap the endpoints with the transport specific layers. The generated\n")
	b.WriteString("\t// server packages contains code generated from the design which maps\n")
	b.WriteString("\t// the service input and output data structures to HTTP requests and\n")
	b.WriteString("\t// responses.\n")
	b.WriteString("\tvar (\n")
	for _, svc := range httpServices {
		fmt.Fprintf(&b, "\t\t%sServer *%ssvr.Server\n", svc.Service.VarName, svc.Service.PkgName)
	}
	for _, svc := range jsonrpcServices {
		fmt.Fprintf(&b, "\t\t%sJSONRPCServer *%sjssvr.Server\n", svc.Service.VarName, svc.Service.PkgName)
	}
	b.WriteString("\t)\n")
	b.WriteString("\t{\n")
	b.WriteString("\t\teh := errorHandler(ctx)\n")
	if httpcodegen.NeedDialer(httpServices) || httpcodegen.NeedDialer(jsonrpcServices) {
		b.WriteString("\t\tupgrader := &websocket.Upgrader{}\n")
	}
	for _, svc := range httpServices {
		fmt.Fprintf(&b, "\t\t%s\n", httpcodegenServerConstructorCall(svc, apiPkg))
	}
	for _, svc := range jsonrpcServices {
		if len(svc.Endpoints) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\t\t%sJSONRPCServer = %sjssvr.New(", svc.Service.VarName, svc.Service.PkgName)
		if httpcodegen.HasWebSocket(svc) {
			fmt.Fprintf(&b, "%sSvc.HandleStream, ", svc.Service.VarName)
		}
		fmt.Fprintf(&b, "%sEndpoints, mux, dec, enc, eh", svc.Service.VarName)
		if httpcodegen.HasWebSocket(svc) {
			b.WriteString(", upgrader, nil")
		}
		b.WriteString(")\n")
	}
	b.WriteString("\t}\n\n")
	b.WriteString("\t// Configure the mux.\n")
	for _, svc := range httpServices {
		fmt.Fprintf(&b, "\t%ssvr.Mount(mux, %sServer)\n", svc.Service.PkgName, svc.Service.VarName)
	}
	for _, svc := range jsonrpcServices {
		fmt.Fprintf(&b, "\t%sjssvr.Mount(mux, %sJSONRPCServer)\n", svc.Service.PkgName, svc.Service.VarName)
	}
	return b.String()
}

func jsonrpcHTTPMountLogSource(jsonrpcServices []*httpcodegen.ServiceData) string {
	var b strings.Builder
	for _, svc := range jsonrpcServices {
		fmt.Fprintf(&b, "\tfor _, m := range %sJSONRPCServer.Methods {\n", svc.Service.VarName)
		for _, route := range svc.Endpoints[0].Routes {
			fmt.Fprintf(&b, "\t\tlog.Printf(ctx, \"JSON-RPC method %%q mounted on %s %s\", m)\n", route.Verb, route.Path)
		}
		b.WriteString("\t}\n")
	}
	return b.String()
}

func jsonrpcExampleServerEndSource(httpServices, jsonrpcServices []*httpcodegen.ServiceData) string {
	var b strings.Builder
	b.WriteString("\n\t// Start HTTP server using default configuration, change the code to\n")
	b.WriteString("\t// configure the server as required by your service.\n")
	b.WriteString("\tsrv := &http.Server{Addr: u.Host, Handler: handler, ReadHeaderTimeout: time.Second * 60}\n")
	for _, svc := range httpServices {
		fmt.Fprintf(&b, "\tfor _, m := range %sServer.Mounts {\n", svc.Service.VarName)
		b.WriteString("\t\tlog.Printf(ctx, \"HTTP %q mounted on %s %s\", m.Method, m.Verb, m.Pattern)\n")
		b.WriteString("\t}\n")
	}
	if len(jsonrpcServices) > 0 {
		b.WriteString("\n")
		b.WriteString(jsonrpcHTTPMountLogSource(jsonrpcServices))
	}
	b.WriteString("\n\t(*wg).Add(1)\n")
	b.WriteString("\tgo func() {\n")
	b.WriteString("\t\tdefer (*wg).Done()\n\n")
	b.WriteString("\t\t// Start HTTP server in a separate goroutine.\n")
	b.WriteString("\t\tgo func() {\n")
	b.WriteString("\t\t\tlog.Printf(ctx, \"HTTP server listening on %q\", u.Host)\n")
	b.WriteString("\t\t\terrc <- srv.ListenAndServe()\n")
	b.WriteString("\t\t}()\n\n")
	b.WriteString("\t\t<-ctx.Done()\n")
	b.WriteString("\t\tlog.Printf(ctx, \"shutting down HTTP server at %q\", u.Host)\n\n")
	b.WriteString("\t\t// Shutdown gracefully with a 30s timeout.\n")
	b.WriteString("\t\tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)\n")
	b.WriteString("\t\tdefer cancel()\n\n")
	b.WriteString("\t\terr := srv.Shutdown(ctx)\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\tlog.Printf(ctx, \"failed to shutdown: %v\", err)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}()\n}\n")
	return b.String()
}

func hasServiceName(services []*httpcodegen.ServiceData, name string) bool {
	for _, svc := range services {
		if svc.Service.Name == name {
			return true
		}
	}
	return false
}

func httpcodegenServerConstructorCall(svc *httpcodegen.ServiceData, apiPkg string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%sServer = %ssvr.New(", svc.Service.VarName, svc.Service.PkgName)
	if len(svc.Endpoints) > 0 {
		fmt.Fprintf(&b, "%sEndpoints", svc.Service.VarName)
	} else {
		b.WriteString("nil")
	}
	b.WriteString(", mux, dec, enc, eh, nil")
	if httpcodegen.HasWebSocket(svc) {
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
