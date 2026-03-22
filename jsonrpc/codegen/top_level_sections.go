package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcClientStructSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-client-struct", renderJSONRPCClientStruct(data))
}

func renderJSONRPCClientStruct(data *httpcodegen.ServiceData) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s lists the %s service endpoint HTTP clients.", data.ClientStruct, data.Service.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s struct {\n", data.ClientStruct)
	b.WriteString("\t")
	b.WriteString(codegen.Comment(fmt.Sprintf("Doer is the HTTP client used to make requests to the %s service.", data.Service.Name)))
	b.WriteString("\n\tDoer goahttp.Doer\n")
	for _, endpoint := range data.Endpoints {
		if !httpcodegen.IsSSEEndpoint(endpoint) {
			continue
		}
		b.WriteString("\t")
		b.WriteString(codegen.Comment(fmt.Sprintf("%s Doer is the HTTP client used to make requests to the %s endpoint.", endpoint.Method.VarName, endpoint.Method.Name)))
		b.WriteString("\n")
		fmt.Fprintf(&b, "\t%sDoer goahttp.Doer\n", endpoint.Method.VarName)
	}
	b.WriteString("\t// RestoreResponseBody controls whether the response bodies are reset after\n")
	b.WriteString("\t// decoding so they can be read again.\n")
	b.WriteString("\tRestoreResponseBody bool\n\n")
	b.WriteString("\tscheme     string\n")
	b.WriteString("\thost       string\n")
	b.WriteString("\tencoder    func(*http.Request) goahttp.Encoder\n")
	b.WriteString("\tdecoder    func(*http.Response) goahttp.Decoder\n")
	if httpcodegen.HasWebSocket(data) {
		b.WriteString("\tdialer goahttp.Dialer\n")
		b.WriteString("\tconfigfn goahttp.ConnConfigureFunc\n\n")
		b.WriteString("\tconnMu sync.RWMutex\n")
		b.WriteString("\tconn   *websocket.Conn\n")
		b.WriteString("\tclosed atomic.Bool\n\n")
		b.WriteString("\t// Stream configuration (shared by all WebSocket streams)\n")
		b.WriteString("\tstreamConfig *jsonrpc.StreamConfig\n")
	}
	b.WriteString("}\n")
	if !httpcodegen.HasWebSocket(data) {
		b.WriteString("\n// bufferPool is a pool of bytes.Buffers for encoding requests.\n")
		b.WriteString("var bufferPool = sync.Pool{\n")
		b.WriteString("\tNew: func() any { return new(bytes.Buffer) },\n")
		b.WriteString("}\n")
	}
	return b.String()
}

func jsonrpcClientInitSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-client-init", renderJSONRPCClientInit(data))
}

func renderJSONRPCClientInit(data *httpcodegen.ServiceData) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("New%s instantiates HTTP clients for all the %s service servers.", data.ClientStruct, data.Service.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func New%s(\n", data.ClientStruct)
	b.WriteString("\tscheme string,\n")
	b.WriteString("\thost string,\n")
	b.WriteString("\tdoer goahttp.Doer,\n")
	b.WriteString("\tenc func(*http.Request) goahttp.Encoder,\n")
	b.WriteString("\tdec func(*http.Response) goahttp.Decoder,\n")
	b.WriteString("\trestoreBody bool,\n")
	if httpcodegen.HasWebSocket(data) {
		b.WriteString("\tdialer goahttp.Dialer,\n")
		b.WriteString("\tcfn goahttp.ConnConfigureFunc,\n")
		b.WriteString("\tstreamOpts ...jsonrpc.StreamConfigOption,\n")
	}
	fmt.Fprintf(&b, ") *%s {\n", data.ClientStruct)
	if httpcodegen.HasWebSocket(data) {
		b.WriteString("\t// Create stream configuration from options\n")
		b.WriteString("\tstreamConfig := jsonrpc.NewStreamConfig(streamOpts...)\n\n")
	}
	fmt.Fprintf(&b, "\treturn &%s{\n", data.ClientStruct)
	b.WriteString("\t\tDoer:                doer,\n")
	for _, endpoint := range data.Endpoints {
		if !httpcodegen.IsSSEEndpoint(endpoint) {
			continue
		}
		fmt.Fprintf(&b, "\t\t%sDoer: %s,\n", endpoint.Method.VarName, "doer")
	}
	b.WriteString("\t\tRestoreResponseBody: restoreBody,\n")
	b.WriteString("\t\tscheme:              scheme,\n")
	b.WriteString("\t\thost:                host,\n")
	b.WriteString("\t\tdecoder:             dec,\n")
	b.WriteString("\t\tencoder:             enc,\n")
	if httpcodegen.HasWebSocket(data) {
		b.WriteString("\t\tdialer:              dialer,\n")
		b.WriteString("\t\tconfigfn:            cfn,\n")
		b.WriteString("\t\tstreamConfig:        streamConfig,\n")
	}
	b.WriteString("\t}\n}\n")
	return b.String()
}

func jsonrpcServerStructSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-server-struct", renderJSONRPCServerStruct(data))
}

func renderJSONRPCServerStruct(data *httpcodegen.ServiceData) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s handles JSON-RPC requests for the %s service.", data.ServerStruct, data.Service.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s struct {\n", data.ServerStruct)
	b.WriteString("\thttp.Handler\n")
	b.WriteString("\t// Methods is the list of methods served by this server.\n")
	b.WriteString("\tMethods []string\n")
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		b.WriteString("\t// StreamHandler is the handler for the streaming service.\n")
		fmt.Fprintf(&b, "\tStreamHandler func(context.Context, %s.Stream) error\n", data.Service.PkgName)
	}
	for _, endpoint := range data.Endpoints {
		if httpcodegen.IsWebSocketEndpoint(endpoint) {
			fmt.Fprintf(&b, "\t%s func(context.Context, *http.Request, *jsonrpc.RawRequest) (any, error)\n", lowerInitial(endpoint.Method.VarName))
			if endpoint.Method.ServerStream != nil && (endpoint.Method.ServerStream.Kind == 3 || endpoint.Method.ServerStream.Kind == 4) {
				fmt.Fprintf(&b, "\t%sEndpoint goa.Endpoint\n", lowerInitial(endpoint.Method.VarName))
			}
			continue
		}
		b.WriteString("\t")
		b.WriteString(codegen.Comment(fmt.Sprintf("%s is the handler for the %s method.", endpoint.Method.VarName, endpoint.Method.Name)))
		b.WriteString("\n")
		fmt.Fprintf(&b, "\t%s func(context.Context, *http.Request, *jsonrpc.RawRequest, http.ResponseWriter) error\n", endpoint.Method.VarName)
	}
	b.WriteString("\n\tdecoder func(*http.Request) goahttp.Decoder\n")
	b.WriteString("\tencoder func(context.Context, http.ResponseWriter) goahttp.Encoder\n")
	b.WriteString("\terrhandler func(context.Context, http.ResponseWriter, error)\n")
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		b.WriteString("\tupgrader goahttp.Upgrader\n")
		b.WriteString("\tconfigfn goahttp.ConnConfigureFunc\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func jsonrpcServerInitSection(data *httpcodegen.ServiceData, hasSSE bool) codegen.Section {
	return codegen.NewRawSection("jsonrpc-server-init", renderJSONRPCServerInit(data, hasSSE))
}

func renderJSONRPCServerInit(data *httpcodegen.ServiceData, hasSSE bool) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s creates a JSON-RPC server which loads HTTP requests and calls the %q service methods.", data.ServerInit, data.Service.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(\n", data.ServerInit)
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		fmt.Fprintf(&b, "\tstreamHandler func(context.Context, %s.Stream) error,\n", data.Service.PkgName)
	}
	fmt.Fprintf(&b, "\tendpoints *%s.Endpoints,\n", data.Service.PkgName)
	b.WriteString("\tmux goahttp.Muxer,\n")
	b.WriteString("\tdecoder func(*http.Request) goahttp.Decoder,\n")
	b.WriteString("\tencoder func(context.Context, http.ResponseWriter) goahttp.Encoder,\n")
	b.WriteString("\terrhandler func(context.Context, http.ResponseWriter, error),\n")
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		b.WriteString("\tupgrader goahttp.Upgrader,\n")
		b.WriteString("\tconfigfn goahttp.ConnConfigureFunc,\n")
	}
	fmt.Fprintf(&b, ") *%s {\n", data.ServerStruct)
	fmt.Fprintf(&b, "\ts := &%s{\n", data.ServerStruct)
	b.WriteString("\t\tMethods: []string{\n")
	for _, endpoint := range data.Endpoints {
		fmt.Fprintf(&b, "\t\t\t%q,\n", endpoint.Method.Name)
	}
	b.WriteString("\t\t},\n")
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		b.WriteString("\t\tStreamHandler: streamHandler,\n")
	}
	for _, endpoint := range data.Endpoints {
		if httpcodegen.IsWebSocketEndpoint(endpoint) {
			fmt.Fprintf(&b, "\t\t%s: %s(endpoints.%s, mux, decoder),\n", lowerInitial(endpoint.Method.VarName), endpoint.HandlerInit, endpoint.Method.VarName)
			if endpoint.Method.ServerStream != nil && (endpoint.Method.ServerStream.Kind == 3 || endpoint.Method.ServerStream.Kind == 4) {
				fmt.Fprintf(&b, "\t\t%sEndpoint: endpoints.%s,\n", lowerInitial(endpoint.Method.VarName), endpoint.Method.VarName)
			}
			continue
		}
		fmt.Fprintf(&b, "\t\t%s: %s(endpoints.%s, mux, decoder, encoder, errhandler),\n", endpoint.Method.VarName, endpoint.HandlerInit, endpoint.Method.VarName)
	}
	b.WriteString("\t\tdecoder: decoder,\n")
	b.WriteString("\t\tencoder: encoder,\n")
	b.WriteString("\t\terrhandler: errhandler,\n")
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		b.WriteString("\t\tupgrader: upgrader,\n")
		b.WriteString("\t\tconfigfn: configfn,\n")
	}
	b.WriteString("\t}\n")
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		b.WriteString("\t// WebSocket services implement ServeHTTP for upgrade\n")
		b.WriteString("\ts.Handler = http.HandlerFunc(s.ServeHTTP)\n")
	} else if hasSSE {
		b.WriteString("\t// SSE-only services route via handleSSE\n")
		b.WriteString("\ts.Handler = http.HandlerFunc(s.handleSSE)\n")
	} else {
		b.WriteString("\t// Plain HTTP JSON-RPC\n")
		b.WriteString("\ts.Handler = http.HandlerFunc(s.ServeHTTP)\n")
	}
	b.WriteString("\treturn s\n}\n")
	return b.String()
}

func jsonrpcServerServiceSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-server-service", fmt.Sprintf("\n%s\nfunc (s *%s) %s() string { return %q }\n", codegen.Comment(fmt.Sprintf("%s returns the name of the service served.", data.ServerService)), data.ServerStruct, data.ServerService, data.Service.Name))
}

func jsonrpcServerUseSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-server-use", fmt.Sprintf("\n%s\nfunc (s *%s) Use(m func(http.Handler) http.Handler) {\n\ts.Handler = m(s.Handler)\n}\n", codegen.Comment("Use wraps the server handlers with the given middleware."), data.ServerStruct))
}

func jsonrpcServerMethodNamesSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-server-method-names", fmt.Sprintf("\n%s\nfunc (s *%s) MethodNames() []string { return %s.MethodNames[:] }\n", codegen.Comment("MethodNames returns the methods served."), data.ServerStruct, data.Service.PkgName))
}

func jsonrpcMixedServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-mixed-server-handler", "\n// ServeHTTP handles JSON-RPC requests with content negotiation for mixed HTTP/SSE transports.\nfunc (s *"+data.ServerStruct+") ServeHTTP(w http.ResponseWriter, r *http.Request) {\n\taccept := r.Header.Get(\"Accept\")\n\tif strings.Contains(accept, \"text/event-stream\") {\n\t\ts.handleSSE(w, r)\n\t\treturn\n\t}\n\t\n\ts.handleHTTP(w, r)\n}\n")
}

func jsonrpcServerMountSection(data *httpcodegen.ServiceData, hasSSE, hasMixed bool) codegen.Section {
	return codegen.NewRawSection("jsonrpc-server-mount", renderJSONRPCServerMount(data, hasSSE, hasMixed))
}

func renderJSONRPCServerMount(data *httpcodegen.ServiceData, hasSSE, hasMixed bool) string {
	var b strings.Builder
	comment := codegen.Comment(fmt.Sprintf("%s configures the mux to serve the JSON-RPC %s service methods.", data.MountServer, data.Service.Name))
	b.WriteString("\n")
	b.WriteString(comment)
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(mux goahttp.Muxer, h *%s) {\n", data.MountServer, data.ServerStruct)
	switch {
	case hasMixed:
		b.WriteString("\t// Mixed transports: mount unified handler that negotiates HTTP vs SSE by Accept header\n")
		for _, route := range data.Endpoints[0].Routes {
			fmt.Fprintf(&b, "\tmux.Handle(%q, %q, h.ServeHTTP)\n", route.Verb, route.Path)
		}
	case hasSSE:
		b.WriteString("\t// SSE only: mount SSE handler\n")
		for _, endpoint := range data.Endpoints {
			for _, route := range endpoint.Routes {
				fmt.Fprintf(&b, "\tmux.Handle(%q, %q, h.handleSSE)\n", route.Verb, route.Path)
			}
		}
	default:
		b.WriteString("\t// HTTP only\n")
		for _, route := range data.Endpoints[0].Routes {
			fmt.Fprintf(&b, "\tmux.Handle(%q, %q, h.ServeHTTP)\n", route.Verb, route.Path)
		}
	}
	b.WriteString("}\n\n")
	b.WriteString(comment)
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) %s(mux goahttp.Muxer) {\n\t%s(mux, s)\n}\n", data.ServerStruct, data.MountServer, data.MountServer)
	return b.String()
}

func jsonrpcServerEncodeErrorSection(serverStruct string) codegen.Section {
	return codegen.NewRawSection("jsonrpc-server-encode-error", "\n// encodeJSONRPCError creates and sends a JSON-RPC error response (handles nil ID gracefully)\nfunc (s *"+serverStruct+") encodeJSONRPCError(ctx context.Context, w http.ResponseWriter, req *jsonrpc.RawRequest, code jsonrpc.Code, message string, data any) {\n\tencodeJSONRPCError(ctx, w, req, code, message, data, s.encoder, s.errhandler)\n}\n\n// encodeJSONRPCError creates and sends a JSON-RPC error response (handles nil ID gracefully)\nfunc encodeJSONRPCError(\n\tctx context.Context,\n\tw http.ResponseWriter,\n\treq *jsonrpc.RawRequest,\n\tcode jsonrpc.Code,\n\tmessage string,\n\tdata any,\n\tencoder func(context.Context, http.ResponseWriter) goahttp.Encoder,\n\terrhandler func(context.Context, http.ResponseWriter, error),\n) {\n\tif req.ID != nil {\n\t\tresponse := jsonrpc.MakeErrorResponse(req.ID, code, message, data)\n\t\tif err := encoder(ctx, w).Encode(response); err != nil {\n\t\t\terrhandler(ctx, w, fmt.Errorf(\"failed to encode JSON-RPC response: %w\", err))\n\t\t}\n\t}\n}\n")
}
