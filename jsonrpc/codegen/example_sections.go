package codegen

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

type exampleSourceBuilder struct {
	parts []string
}

func (b *exampleSourceBuilder) Add(s string) {
	if s == "" {
		return
	}
	b.parts = append(b.parts, s)
}

func (b *exampleSourceBuilder) Addf(format string, args ...any) {
	b.Add(fmt.Sprintf(format, args...))
}

func (b *exampleSourceBuilder) String() string {
	return strings.Join(b.parts, "")
}

func renderSectionSource(section codegen.Section) string {
	var b bytes.Buffer
	if err := section.Write(&b); err != nil {
		panic(err)
	}
	return b.String()
}

func jsonrpcExampleServerStartSource(httpServices, jsonrpcServices []*httpcodegen.ServiceData) string {
	var b exampleSourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment("handleHTTPServer starts configures and starts a HTTP server on the given URL. It shuts down the server if any error is received in the error channel."))
	b.Add("\n")
	b.Add("func handleHTTPServer(ctx context.Context, u *url.URL")
	for _, svc := range httpServices {
		if len(svc.Service.Methods) > 0 {
			b.Addf(", %sEndpoints *%s.Endpoints", svc.Service.VarName, svc.Service.PkgName)
		}
	}
	for _, svc := range jsonrpcServices {
		if !hasServiceName(httpServices, svc.Service.Name) {
			b.Addf(", %sEndpoints *%s.Endpoints", svc.Service.VarName, svc.Service.PkgName)
		}
	}
	for _, svc := range jsonrpcServices {
		b.Addf(", %sSvc %s.Service", svc.Service.VarName, svc.Service.PkgName)
	}
	b.Add(", wg *sync.WaitGroup, errc chan error, dbg bool) {\n")
	return b.String()
}

func jsonrpcExampleServerConfigureSource(httpServices, jsonrpcServices []*httpcodegen.ServiceData, apiPkg string) string {
	var b exampleSourceBuilder
	b.Add("\n\t// Wrap the endpoints with the transport specific layers. The generated\n")
	b.Add("\t// server packages contains code generated from the design which maps\n")
	b.Add("\t// the service input and output data structures to HTTP requests and\n")
	b.Add("\t// responses.\n")
	b.Add("\tvar (\n")
	for _, svc := range httpServices {
		b.Addf("\t\t%sServer *%ssvr.Server\n", svc.Service.VarName, svc.Service.PkgName)
	}
	for _, svc := range jsonrpcServices {
		b.Addf("\t\t%sJSONRPCServer *%sjssvr.Server\n", svc.Service.VarName, svc.Service.PkgName)
	}
	b.Add("\t)\n")
	b.Add("\t{\n")
	b.Add("\t\teh := errorHandler(ctx)\n")
	if httpcodegen.NeedDialer(httpServices) || httpcodegen.NeedDialer(jsonrpcServices) {
		b.Add("\t\tupgrader := &websocket.Upgrader{}\n")
	}
	for _, svc := range httpServices {
		b.Addf("\t\t%s\n", httpcodegenServerConstructorCall(svc, apiPkg))
	}
	for _, svc := range jsonrpcServices {
		if len(svc.Endpoints) == 0 {
			continue
		}
		b.Addf("\t\t%sJSONRPCServer = %sjssvr.New(", svc.Service.VarName, svc.Service.PkgName)
		if httpcodegen.HasWebSocket(svc) {
			b.Addf("%sSvc.HandleStream, ", svc.Service.VarName)
		}
		b.Addf("%sEndpoints, mux, dec, enc, eh", svc.Service.VarName)
		if httpcodegen.HasWebSocket(svc) {
			b.Add(", upgrader, nil")
		}
		b.Add(")\n")
	}
	b.Add("\t}\n\n")
	b.Add("\t// Configure the mux.\n")
	for _, svc := range httpServices {
		b.Addf("\t%ssvr.Mount(mux, %sServer)\n", svc.Service.PkgName, svc.Service.VarName)
	}
	for _, svc := range jsonrpcServices {
		b.Addf("\t%sjssvr.Mount(mux, %sJSONRPCServer)\n", svc.Service.PkgName, svc.Service.VarName)
	}
	return b.String()
}

func jsonrpcHTTPMountLogSource(jsonrpcServices []*httpcodegen.ServiceData) string {
	var b exampleSourceBuilder
	for _, svc := range jsonrpcServices {
		b.Addf("\tfor _, m := range %sJSONRPCServer.Methods {\n", svc.Service.VarName)
		for _, route := range svc.Endpoints[0].Routes {
			b.Addf("\t\tlog.Printf(ctx, \"JSON-RPC method %%q mounted on %s %s\", m)\n", route.Verb, route.Path)
		}
		b.Add("\t}\n")
	}
	return b.String()
}

func jsonrpcExampleServerEndSource(httpServices, jsonrpcServices []*httpcodegen.ServiceData) string {
	var b exampleSourceBuilder
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
	for _, svc := range httpServices {
		b.Addf("\tfor _, m := range %sServer.Mounts {\n", svc.Service.VarName)
		b.Add("\t\tlog.Printf(ctx, \"HTTP %q mounted on %s %s\", m.Method, m.Verb, m.Pattern)\n")
		b.Add("\t}\n")
	}
	if len(jsonrpcServices) > 0 {
		b.Add("\n")
		b.Add(jsonrpcHTTPMountLogSource(jsonrpcServices))
	}
	b.Add("\n\t(*wg).Add(1)\n")
	b.Add("\tgo func() {\n")
	b.Add("\t\tdefer (*wg).Done()\n\n")
	b.Add("\t\t// Start HTTP server in a separate goroutine.\n")
	b.Add("\t\tgo func() {\n")
	b.Add("\t\t\tlog.Printf(ctx, \"HTTP server listening on %q\", u.Host)\n")
	b.Add("\t\t\terrc <- srv.ListenAndServe()\n")
	b.Add("\t\t}()\n\n")
	b.Add("\t\t<-ctx.Done()\n")
	b.Add("\t\tlog.Printf(ctx, \"shutting down HTTP server at %q\", u.Host)\n\n")
	b.Add("\t\t// Shutdown gracefully with a 30s timeout.\n")
	b.Add("\t\tshutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)\n")
	b.Add("\t\tdefer cancel()\n\n")
	b.Add("\t\terr := srv.Shutdown(shutdownCtx)\n")
	b.Add("\t\tif err != nil {\n")
	b.Add("\t\t\tlog.Printf(shutdownCtx, \"failed to shutdown: %v\", err)\n")
	b.Add("\t\t}\n")
	b.Add("\t}()\n}\n")
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
	var b exampleSourceBuilder
	b.Addf("%sServer = %ssvr.New(", svc.Service.VarName, svc.Service.PkgName)
	if len(svc.Endpoints) > 0 {
		b.Addf("%sEndpoints", svc.Service.VarName)
	} else {
		b.Add("nil")
	}
	b.Add(", mux, dec, enc, eh, nil")
	if httpcodegen.HasWebSocket(svc) {
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
