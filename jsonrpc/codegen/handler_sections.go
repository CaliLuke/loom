package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcServerHandlerSection(data *httpcodegen.ServiceData, mixed bool) codegen.Section {
	return codegen.NewRawSection("jsonrpc-server-handler", renderJSONRPCServerHandler(data, mixed))
}

func renderJSONRPCServerHandler(data *httpcodegen.ServiceData, mixed bool) string {
	var b strings.Builder
	b.WriteString("\n")
	if !httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) && !mixed {
		b.WriteString("// ServeHTTP handles JSON-RPC requests.\n")
		fmt.Fprintf(&b, "func (s *%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n", data.ServerStruct)
		b.WriteString("\ts.handleHTTP(w, r)\n")
		b.WriteString("}\n\n")
	}

	b.WriteString(codegen.Comment("handleHTTP handles JSON-RPC requests."))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) handleHTTP(w http.ResponseWriter, r *http.Request) {\n", data.ServerStruct)
	writeBufferedRequestHandling(&b)
	b.WriteString("}\n\n")

	b.WriteString("// handleSingle handles a single JSON-RPC request.\n")
	b.WriteString("func (s *Server) handleSingle(w http.ResponseWriter, r *http.Request) {\n")
	b.WriteString("\tvar req jsonrpc.RawRequest\n")
	b.WriteString("\tif err := s.decoder(r).Decode(&req); err != nil {\n")
	writeParseErrorResponse(&b, "\t\t")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("\ts.processRequest(r.Context(), r, &req, w)\n")
	b.WriteString("}\n\n")

	b.WriteString("// handleBatch handles a batch of JSON-RPC requests.\n")
	b.WriteString("func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {\n")
	b.WriteString("\tvar reqs []jsonrpc.RawRequest\n")
	b.WriteString("\tif err := s.decoder(r).Decode(&reqs); err != nil {\n")
	writeParseErrorResponse(&b, "\t\t")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tw.Header().Set(\"Content-Type\", \"application/json\")\n")
	b.WriteString("\twriter := &batchWriter{Writer: w}\n\n")
	b.WriteString("\tfor _, req := range reqs {\n")
	b.WriteString("\t\ts.processRequest(r.Context(), r, &req, writer)\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tif writer.written {\n")
	b.WriteString("\t\twriter.Writer.Write([]byte{']'})\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	b.WriteString("// ProcessRequest processes a single JSON-RPC request.\n")
	b.WriteString("func (s *Server) processRequest(ctx context.Context, r *http.Request, req *jsonrpc.RawRequest, w http.ResponseWriter) {\n")
	b.WriteString("\tif req.JSONRPC != \"2.0\" {\n")
	b.WriteString("\t\ts.encodeJSONRPCError(ctx, w, req, jsonrpc.InvalidRequest, \"Invalid request\", nil)\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tif req.Method == \"\" {\n")
	b.WriteString("\t\ts.encodeJSONRPCError(ctx, w, req, jsonrpc.InvalidRequest, \"Missing method field\", nil)\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tswitch req.Method {\n")
	writeJSONRPCMethodDispatch(&b, data.Endpoints)
	b.WriteString("\tdefault:\n")
	b.WriteString("\t\ts.encodeJSONRPCError(ctx, w, req, jsonrpc.MethodNotFound, \"Method not found\", nil)\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	b.WriteString("// batchWriter is a helper type that implements http.ResponseWriter for writing multiple JSON-RPC responses\n")
	b.WriteString("type batchWriter struct {\n")
	b.WriteString("\tio.Writer\n")
	b.WriteString("\theader http.Header\n")
	b.WriteString("\tstatusCode int\n")
	b.WriteString("\twritten bool\n")
	b.WriteString("}\n\n")
	b.WriteString("func (rb *batchWriter) Header() http.Header {\n")
	b.WriteString("\tif rb.header == nil {\n")
	b.WriteString("\t\trb.header = make(http.Header)\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn rb.header\n")
	b.WriteString("}\n\n")
	b.WriteString("func (rb *batchWriter) WriteHeader(statusCode int) {\n")
	b.WriteString("\tif rb.written {\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("\trb.statusCode = statusCode\n")
	b.WriteString("}\n\n")
	b.WriteString("func (rb *batchWriter) Write(data []byte) (int, error) {\n")
	b.WriteString("\tif !rb.written {\n")
	b.WriteString("\t\trb.written = true\n")
	b.WriteString("\t\trb.Writer.Write([]byte{'['})\n")
	b.WriteString("\t} else {\n")
	b.WriteString("\t\trb.Writer.Write([]byte{','})\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn rb.Writer.Write(data)\n")
	b.WriteString("}\n")
	return b.String()
}

func writeBufferedRequestHandling(b *strings.Builder) {
	b.WriteString("\t// Peek at the first byte to determine request type\n")
	b.WriteString("\tbufReader := bufio.NewReader(r.Body)\n")
	b.WriteString("\tpeek, err := bufReader.Peek(1)\n")
	b.WriteString("\tif err != nil && err != io.EOF {\n")
	b.WriteString("\t\tr.Body.Close()\n")
	b.WriteString("\t\ts.errhandler(r.Context(), w, fmt.Errorf(\"failed to read request body: %w\", err))\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\t// Wrap the buffered reader with the original closer\n")
	b.WriteString("\tr.Body = struct {\n")
	b.WriteString("\t\tio.Reader\n")
	b.WriteString("\t\tio.Closer\n")
	b.WriteString("\t}{\n")
	b.WriteString("\t\tReader: bufReader,\n")
	b.WriteString("\t\tCloser: r.Body,\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdefer func(r *http.Request) {\n")
	b.WriteString("\t\tif err := r.Body.Close(); err != nil {\n")
	b.WriteString("\t\t\ts.errhandler(r.Context(), w, fmt.Errorf(\"failed to close request body: %w\", err))\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}(r)\n\n")
	b.WriteString("\t// Route to appropriate handler\n")
	b.WriteString("\tif len(peek) > 0 && peek[0] == '[' {\n")
	b.WriteString("\t\ts.handleBatch(w, r)\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("\ts.handleSingle(w, r)\n")
}

func writeParseErrorResponse(b *strings.Builder, indent string) {
	fmt.Fprintf(b, "%sresponse := jsonrpc.MakeErrorResponse(nil, jsonrpc.ParseError, \"Parse error\", nil)\n", indent)
	fmt.Fprintf(b, "%sif encErr := s.encoder(r.Context(), w).Encode(response); encErr != nil {\n", indent)
	fmt.Fprintf(b, "%s\ts.errhandler(r.Context(), w, fmt.Errorf(\"failed to encode parse error response: %%w\", encErr))\n", indent)
	fmt.Fprintf(b, "%s}\n", indent)
}

func writeJSONRPCMethodDispatch(b *strings.Builder, endpoints []*httpcodegen.EndpointData) {
	for _, endpoint := range endpoints {
		fmt.Fprintf(b, "\tcase %q:\n", endpoint.Method.Name)
		fmt.Fprintf(b, "\t\tif err := s.%s(ctx, r, req, w); err != nil {\n", endpoint.Method.VarName)
		fmt.Fprintf(b, "\t\t\ts.errhandler(ctx, w, fmt.Errorf(\"handler error for %%s: %%w\", %q, err))\n", endpoint.Method.Name)
		b.WriteString("\t\t}\n")
	}
}

func jsonrpcSSEServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-sse-server-handler", renderJSONRPCSSEServerHandler(data))
}

func renderJSONRPCSSEServerHandler(data *httpcodegen.ServiceData) string {
	var b strings.Builder
	streamName := lowerInitial(data.Service.StructName) + "SSEStream"
	b.WriteString("\n// handleSSE handles JSON-RPC SSE requests by dispatching to the appropriate method.\n")
	b.WriteString("func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {\n")
	b.WriteString("\tctx := r.Context()\n\n")
	b.WriteString("\tvar req jsonrpc.RawRequest\n")
	b.WriteString("\tif err := s.decoder(r).Decode(&req); err != nil {\n")
	writeSSEErrorStreamInit(&b, streamName, "\t\t")
	b.WriteString("\t\t_ = stream.sendError(ctx, nil, jsonrpc.ParseError, \"Parse error\", nil)\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n\n")
	writeSSERequestValidation(&b, streamName)
	b.WriteString("\tvar handler func(context.Context, *http.Request, *jsonrpc.RawRequest, http.ResponseWriter) error\n")
	b.WriteString("\tswitch req.Method {\n")
	for _, endpoint := range data.Endpoints {
		if endpoint.SSE == nil {
			continue
		}
		fmt.Fprintf(&b, "\tcase %q:\n", endpoint.Method.Name)
		fmt.Fprintf(&b, "\t\thandler = s.%s\n", endpoint.Method.VarName)
	}
	b.WriteString("\tdefault:\n")
	b.WriteString("\t\tif req.ID == nil || req.ID == \"\" {\n")
	b.WriteString("\t\t\tw.WriteHeader(http.StatusNoContent)\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n")
	fmt.Fprintf(&b, "\t\tstream := &%s{w: w, r: r, encoder: s.encoder, decoder: s.decoder}\n", streamName)
	b.WriteString("\t\t_ = stream.sendError(ctx, req.ID, jsonrpc.MethodNotFound, \"Method not found\", nil)\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tif err := handler(ctx, r, &req, w); err != nil {\n")
	b.WriteString("\t\ts.errhandler(ctx, w, fmt.Errorf(\"handler error for %s: %w\", req.Method, err))\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tswitch req.Method {\n")
	for _, endpoint := range data.Endpoints {
		if endpoint.SSE == nil || endpoint.Method.ServerStream != nil {
			continue
		}
		fmt.Fprintf(&b, "\tcase %q:\n", endpoint.Method.Name)
		b.WriteString("\t\tif req.ID == nil {\n")
		b.WriteString("\t\t\tw.WriteHeader(http.StatusNoContent)\n")
		b.WriteString("\t\t}\n")
	}
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return b.String()
}

func writeSSEErrorStreamInit(b *strings.Builder, streamName, indent string) {
	fmt.Fprintf(b, "%sstream := &%s{w: w, r: r, encoder: s.encoder, decoder: s.decoder}\n", indent, streamName)
}

func writeSSEValidationError(b *strings.Builder, streamName, message string) {
	b.WriteString("\t\tif req.ID == nil || req.ID == \"\" {\n")
	b.WriteString("\t\t\tw.WriteHeader(http.StatusNoContent)\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n")
	writeSSEErrorStreamInit(b, streamName, "\t\t")
	fmt.Fprintf(b, "\t\t_ = stream.sendError(ctx, req.ID, jsonrpc.InvalidRequest, %q, nil)\n", message)
	b.WriteString("\t\treturn\n")
}

func writeSSERequestValidation(b *strings.Builder, streamName string) {
	b.WriteString("\tif req.JSONRPC != \"2.0\" {\n")
	writeSSEValidationError(b, streamName, "Invalid request")
	b.WriteString("\t}\n\n")
	b.WriteString("\tif req.Method == \"\" {\n")
	writeSSEValidationError(b, streamName, "Invalid request")
	b.WriteString("\t}\n\n")
}

func jsonrpcWebSocketServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-websocket-server-handler", renderJSONRPCWebSocketServerHandler(data))
}

func renderJSONRPCWebSocketServerHandler(data *httpcodegen.ServiceData) string {
	var b strings.Builder
	streamName := lowerInitial(data.Service.StructName) + "Stream"
	b.WriteString("\n// ServeHTTP handles WebSocket JSON-RPC requests.\n")
	fmt.Fprintf(&b, "func (s *%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n", data.ServerStruct)
	b.WriteString("\tctx, cancel := context.WithCancel(r.Context())\n")
	b.WriteString("\tconn, err := s.upgrader.Upgrade(w, r, nil)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\ts.errhandler(r.Context(), w, fmt.Errorf(\"failed to upgrade to WebSocket: %w\", err))\n")
	b.WriteString("\t\tcancel()\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif s.configfn != nil {\n")
	b.WriteString("\t\tconn = s.configfn(conn, cancel)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdefer conn.Close()\n\n")
	fmt.Fprintf(&b, "\tstream := &%s{\n", streamName)
	for _, endpoint := range data.Endpoints {
		fmt.Fprintf(&b, "\t\t%s: s.%s,\n", lowerInitial(endpoint.Method.VarName), lowerInitial(endpoint.Method.VarName))
		if endpoint.Method.ServerStream != nil && (endpoint.Method.ServerStream.Kind == 3 || endpoint.Method.ServerStream.Kind == 4) {
			fmt.Fprintf(&b, "\t\t%sEndpoint: s.%sEndpoint,\n", lowerInitial(endpoint.Method.VarName), lowerInitial(endpoint.Method.VarName))
		}
	}
	b.WriteString("\t\tr: r,\n")
	b.WriteString("\t\tw: w,\n")
	b.WriteString("\t\tconn: conn,\n")
	b.WriteString("\t\tcancel: cancel,\n")
	b.WriteString("\t}\n")
	b.WriteString("\ts.StreamHandler(ctx, stream)\n")
	b.WriteString("}\n")
	return b.String()
}

func jsonrpcServerHandlerInitSection(e *httpcodegen.EndpointData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-server-handler-init", renderJSONRPCServerHandlerInit(e))
}

func renderJSONRPCServerHandlerInit(e *httpcodegen.EndpointData) string {
	var b strings.Builder
	comment := fmt.Sprintf("%s creates a JSON-RPC handler which calls the %q service %q endpoint.", e.HandlerInit, e.ServiceName, e.Method.Name)
	b.WriteString("\n")
	b.WriteString(codegen.Comment(comment))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(\n", e.HandlerInit)
	b.WriteString("\tendpoint loom.Endpoint,\n")
	b.WriteString("\tmux loomhttp.Muxer,\n")
	b.WriteString("\tdecoder func(*http.Request) loomhttp.Decoder,\n")
	if !httpcodegen.IsWebSocketEndpoint(e) {
		b.WriteString("\tencoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,\n")
		b.WriteString("\terrhandler func(context.Context, http.ResponseWriter, error),\n")
	}
	fmt.Fprintf(&b, ") func(context.Context, *http.Request, *jsonrpc.RawRequest")
	if !httpcodegen.IsWebSocketEndpoint(e) {
		b.WriteString(", http.ResponseWriter")
	}
	if httpcodegen.IsWebSocketEndpoint(e) {
		b.WriteString(") (any, error) {\n")
	} else {
		b.WriteString(") error {\n")
	}
	if !httpcodegen.IsSSEEndpoint(e) && e.Payload != nil && e.Payload.Ref != "" {
		if !(httpcodegen.IsWebSocketEndpoint(e) && e.Method.ServerStream != nil && (e.Method.ServerStream.Kind == 3 || e.Method.ServerStream.Kind == 4)) {
			fmt.Fprintf(&b, "\tdecodeParams := %s(mux, decoder)\n", e.RequestDecoder)
		}
	}
	if !httpcodegen.IsWebSocketEndpoint(e) && needsJSONRPCResponseCapture(e) {
		fmt.Fprintf(&b, "\tencodeResponse := %s(encoder)\n", e.ResponseEncoder)
	}
	fmt.Fprintf(&b, "\treturn func(ctx context.Context, r *http.Request, req *jsonrpc.RawRequest")
	if !httpcodegen.IsWebSocketEndpoint(e) {
		b.WriteString(", w http.ResponseWriter")
	}
	if httpcodegen.IsWebSocketEndpoint(e) {
		b.WriteString(") (any, error) {\n")
	} else {
		b.WriteString(") error {\n")
	}
	fmt.Fprintf(&b, "\t\tctx = context.WithValue(ctx, loom.MethodKey, %q)\n", e.Method.Name)
	fmt.Fprintf(&b, "\t\tctx = context.WithValue(ctx, loom.ServiceKey, %q)\n\n", e.ServiceName)

	if httpcodegen.IsSSEEndpoint(e) {
		renderSSEHandlerInitBody(&b, e)
	} else {
		renderJSONRPCStandardHandlerInitBody(&b, e)
	}
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return b.String()
}

func renderSSEHandlerInitBody(b *strings.Builder, e *httpcodegen.EndpointData) {
	fmt.Fprintf(b, "\t\tstrm := &%s{\n", e.SSE.StructName)
	b.WriteString("\t\t\tw:         w,\n")
	b.WriteString("\t\t\tr:         r,\n")
	b.WriteString("\t\t\tencoder:   encoder,\n")
	b.WriteString("\t\t\trequestID: req.ID,\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif r.Method == http.MethodGet && req.Method == \"events/stream\" {\n")
	b.WriteString("\t\t\tif err := strm.open(); err != nil {\n")
	b.WriteString("\t\t\t\treturn err\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t}\n")
	if e.Payload != nil && e.Payload.Ref != "" {
		fmt.Fprintf(b, "\t\tdecodeParams := %s(mux, decoder)\n", e.RequestDecoder)
		b.WriteString("\t\tparams, err := decodeParams(r, req)\n")
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString("\t\t\tif req.ID != nil && req.ID != \"\" {\n")
		b.WriteString("\t\t\t\tstrm.SendError(ctx, jsonrpc.IDToString(req.ID), err)\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t\treturn nil\n")
		b.WriteString("\t\t}\n")
		writePayloadIDInjection(b, "\t\t", e.Payload)
	}
	if e.SSE.RequestIDField != "" {
		b.WriteString("\t\tif lastEventID := r.Header.Get(\"Last-Event-ID\"); lastEventID != \"\" {\n")
		b.WriteString("\t\t\tctx = context.WithValue(ctx, \"last-event-id\", lastEventID)\n")
		if e.Payload != nil && e.Payload.Ref != "" && e.Payload.Request != nil && e.Payload.Request.PayloadType != nil && e.Payload.Request.PayloadType.Name() == "Object" {
			fmt.Fprintf(b, "\t\t\tparams.%s = lastEventID\n", e.SSE.RequestIDField)
		}
		b.WriteString("\t\t}\n")
	}
	fmt.Fprintf(b, "\t\tv := &%s.%s{\n", e.ServicePkgName, e.Method.ServerStream.EndpointStruct)
	b.WriteString("\t\t\tStream: strm,\n")
	if e.Payload != nil && e.Payload.Ref != "" {
		b.WriteString("\t\t\tPayload: params,\n")
	}
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif _, err := endpoint(ctx, v); err != nil {\n")
	b.WriteString("\t\t\tif req.ID != nil && req.ID != \"\" {\n")
	b.WriteString("\t\t\t\tvar en loom.LoomErrorNamer\n")
	b.WriteString("\t\t\t\tif errors.As(err, &en) {\n")
	b.WriteString("\t\t\t\t\tswitch en.LoomErrorName() {\n")
	b.WriteString("\t\t\t\t\tcase \"invalid_params\":\n")
	b.WriteString("\t\t\t\t\t\treturn strm.sendError(ctx, jsonrpc.IDToString(req.ID), jsonrpc.InvalidParams, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n")
	b.WriteString("\t\t\t\t\tcase \"method_not_found\":\n")
	b.WriteString("\t\t\t\t\t\treturn strm.sendError(ctx, jsonrpc.IDToString(req.ID), jsonrpc.MethodNotFound, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n")
	b.WriteString("\t\t\t\t\t}\n")
	b.WriteString("\t\t\t\t}\n")
	b.WriteString("\t\t\t\tcode := jsonrpc.InternalError\n")
	b.WriteString("\t\t\t\tif _, ok := err.(*loom.ServiceError); ok {\n")
	b.WriteString("\t\t\t\t\tcode = jsonrpc.InvalidParams\n")
	b.WriteString("\t\t\t\t}\n")
	b.WriteString("\t\t\t\treturn strm.sendError(ctx, jsonrpc.IDToString(req.ID), code, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\treturn nil\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn nil\n")
}

func renderJSONRPCStandardHandlerInitBody(b *strings.Builder, e *httpcodegen.EndpointData) {
	if e.Payload != nil && e.Payload.Ref != "" {
		if httpcodegen.IsWebSocketEndpoint(e) && e.Method.ServerStream != nil && (e.Method.ServerStream.Kind == 3 || e.Method.ServerStream.Kind == 4) {
			fmt.Fprintf(b, "\t\tdecodeParams := %s(mux, decoder)\n", e.RequestDecoder)
		}
		b.WriteString("\t\tparams, err := decodeParams(r, req)\n")
		b.WriteString("\t\tif err != nil {\n")
		if httpcodegen.IsWebSocketEndpoint(e) {
			b.WriteString("\t\t\treturn nil, err\n")
		} else {
			b.WriteString("\t\t\tif req.ID != nil && req.ID != \"\" {\n")
			b.WriteString("\t\t\t\tcode := jsonrpc.InternalError\n")
			b.WriteString("\t\t\t\tif _, ok := err.(*loom.ServiceError); ok {\n")
			b.WriteString("\t\t\t\t\tcode = jsonrpc.InvalidParams\n")
			b.WriteString("\t\t\t\t}\n")
			b.WriteString("\t\t\t\tencodeJSONRPCError(ctx, w, req, code, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err), encoder, errhandler)\n")
			b.WriteString("\t\t\t} else {\n")
			b.WriteString("\t\t\t\terrhandler(ctx, w, fmt.Errorf(\"failed to decode parameters: %w\", err))\n")
			b.WriteString("\t\t\t}\n")
			b.WriteString("\t\t\treturn nil\n")
		}
		b.WriteString("\t\t}\n")
		writePayloadIDInjection(b, "\t\t", e.Payload)
	}

	if httpcodegen.IsWebSocketEndpoint(e) && e.Method.ServerStream != nil && (e.Method.ServerStream.Kind == 3 || e.Method.ServerStream.Kind == 4) {
		if e.Payload != nil && e.Payload.Ref != "" {
			b.WriteString("\t\treturn params, nil\n")
		} else {
			b.WriteString("\t\treturn nil, nil\n")
		}
		return
	}

	resultVar := "res"
	assignOp := ":="
	if e.Result == nil || e.Result.Ref == "" {
		if !needsJSONRPCResponseCapture(e) {
			resultVar = "_"
			assignOp = "="
		}
	}
	if e.Payload != nil && e.Payload.Ref != "" {
		fmt.Fprintf(b, "\t\t%s, err %s endpoint(ctx, params)\n", resultVar, assignOp)
	} else {
		fmt.Fprintf(b, "\t\t%s, err %s endpoint(ctx, nil)\n", resultVar, assignOp)
	}

	if httpcodegen.IsWebSocketEndpoint(e) {
		b.WriteString("\t\treturn res, err\n")
		return
	}

	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\tif req.ID != nil && req.ID != \"\" {\n")
	b.WriteString("\t\t\t\tvar en loom.LoomErrorNamer\n")
	b.WriteString("\t\t\t\tif !errors.As(err, &en) {\n")
	b.WriteString("\t\t\t\t\tencodeJSONRPCError(ctx, w, req, jsonrpc.InternalError, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err), encoder, errhandler)\n")
	b.WriteString("\t\t\t\t\treturn nil\n")
	b.WriteString("\t\t\t\t}\n")
	b.WriteString("\t\t\t\tswitch en.LoomErrorName() {\n")
	for _, gerr := range e.Errors {
		for _, item := range gerr.Errors {
			if item.Response == nil {
				continue
			}
			fmt.Fprintf(b, "\t\t\t\tcase %q:\n", item.Name)
			fmt.Fprintf(b, "\t\t\t\t\tencodeJSONRPCError(ctx, w, req, %d, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err), encoder, errhandler)\n", item.Response.Code)
		}
	}
	b.WriteString("\t\t\t\tcase \"invalid_params\":\n")
	b.WriteString("\t\t\t\t\tencodeJSONRPCError(ctx, w, req, jsonrpc.InvalidParams, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err), encoder, errhandler)\n")
	b.WriteString("\t\t\t\tcase \"method_not_found\":\n")
	b.WriteString("\t\t\t\t\tencodeJSONRPCError(ctx, w, req, jsonrpc.MethodNotFound, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err), encoder, errhandler)\n")
	b.WriteString("\t\t\t\tdefault:\n")
	b.WriteString("\t\t\t\t\tcode := jsonrpc.InternalError\n")
	b.WriteString("\t\t\t\t\tif _, ok := err.(*loom.ServiceError); ok {\n")
	b.WriteString("\t\t\t\t\t\tcode = jsonrpc.InvalidParams\n")
	b.WriteString("\t\t\t\t\t}\n")
	b.WriteString("\t\t\t\t\tencodeJSONRPCError(ctx, w, req, code, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err), encoder, errhandler)\n")
	b.WriteString("\t\t\t\t}\n")
	b.WriteString("\t\t\t} else {\n")
	b.WriteString("\t\t\t\terrhandler(ctx, w, fmt.Errorf(\"endpoint error: %w\", err))\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\treturn nil\n")
	b.WriteString("\t\t}\n\n")

	if e.Result == nil || e.Result.Ref == "" {
		b.WriteString("\t\tif req.ID == nil || req.ID == \"\" {\n")
		b.WriteString("\t\t\treturn nil\n")
		b.WriteString("\t\t}\n")
		if needsJSONRPCResponseCapture(e) {
			b.WriteString("\t\tcapture := &jsonrpcResponseCapture{}\n")
			b.WriteString("\t\tif err := encodeResponse(ctx, capture, res); err != nil {\n")
			b.WriteString("\t\t\terrhandler(ctx, w, fmt.Errorf(\"failed to encode transport response: %w\", err))\n")
			b.WriteString("\t\t\treturn nil\n")
			b.WriteString("\t\t}\n")
			b.WriteString("\t\tcopyJSONRPCResponseMetadata(w, capture)\n")
		}
		b.WriteString("\t\tresponse := jsonrpc.MakeSuccessResponse(req.ID, nil)\n")
		b.WriteString("\t\tif err := encoder(ctx, w).Encode(response); err != nil {\n")
		b.WriteString("\t\t\terrhandler(ctx, w, fmt.Errorf(\"failed to encode JSON-RPC response: %w\", err))\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\treturn nil\n")
		return
	}

	b.WriteString("\t\tvar id any\n")
	if e.Result.IDAttribute != "" {
		b.WriteString("\t\tactual := res.(" + e.Result.Ref + ")\n")
		if e.Result.IDAttributeRequired {
			fmt.Fprintf(b, "\t\tif actual.%s != \"\" {\n", e.Result.IDAttribute)
			fmt.Fprintf(b, "\t\t\tid = actual.%s\n", e.Result.IDAttribute)
			b.WriteString("\t\t} else {\n")
			b.WriteString("\t\t\tid = req.ID\n")
			b.WriteString("\t\t}\n")
		} else {
			fmt.Fprintf(b, "\t\tif actual.%s != nil && *actual.%s != \"\" {\n", e.Result.IDAttribute, e.Result.IDAttribute)
			fmt.Fprintf(b, "\t\t\tid = *actual.%s\n", e.Result.IDAttribute)
			b.WriteString("\t\t} else {\n")
			b.WriteString("\t\t\tid = req.ID\n")
			b.WriteString("\t\t}\n")
		}
	} else {
		b.WriteString("\t\tid = req.ID\n")
	}
	b.WriteString("\t\tif id == nil || id == \"\" {\n")
	b.WriteString("\t\t\treturn nil\n")
	b.WriteString("\t\t}\n")
	if needsJSONRPCResponseCapture(e) {
		b.WriteString("\t\tcapture := &jsonrpcResponseCapture{}\n")
		b.WriteString("\t\tif err := encodeResponse(ctx, capture, res); err != nil {\n")
		b.WriteString("\t\t\terrhandler(ctx, w, fmt.Errorf(\"failed to encode transport response: %w\", err))\n")
		b.WriteString("\t\t\treturn nil\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tcopyJSONRPCResponseMetadata(w, capture)\n")
		b.WriteString("\t\tvar result any\n")
		b.WriteString("\t\tif capture.body.Len() > 0 {\n")
		b.WriteString("\t\t\tresult = json.RawMessage(capture.body.Bytes())\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tresponse := jsonrpc.MakeSuccessResponse(id, result)\n")
		b.WriteString("\t\tif err := encoder(ctx, w).Encode(response); err != nil {\n")
		b.WriteString("\t\t\terrhandler(ctx, w, fmt.Errorf(\"failed to encode JSON-RPC response: %w\", err))\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\treturn nil\n")
		return
	}
	success := e.Result.Responses[0]
	if success != nil && len(success.ServerBody) > 0 && success.ServerBody[0].Init != nil {
		b.WriteString("\t\t// Convert result to response body with proper JSON tags\n")
		if e.Method.ViewedResult != nil {
			fmt.Fprintf(b, "\t\tviewedRes := res.(%s)\n", e.Method.ViewedResult.FullRef)
			fmt.Fprintf(b, "\t\tbody := %s(viewedRes.Projected)\n", success.ServerBody[0].Init.Name)
		} else {
			fmt.Fprintf(b, "\t\tbody := %s(res.(%s))\n", success.ServerBody[0].Init.Name, e.Result.Ref)
		}
		b.WriteString("\t\tresponse := jsonrpc.MakeSuccessResponse(id, body)\n")
	} else {
		b.WriteString("\t\tresponse := jsonrpc.MakeSuccessResponse(id, res)\n")
	}
	b.WriteString("\t\tif err := encoder(ctx, w).Encode(response); err != nil {\n")
	b.WriteString("\t\t\terrhandler(ctx, w, fmt.Errorf(\"failed to encode JSON-RPC response: %w\", err))\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn nil\n")
}

func needsJSONRPCResponseCapture(e *httpcodegen.EndpointData) bool {
	if e == nil || e.Result == nil || len(e.Result.Responses) == 0 {
		return false
	}
	success := e.Result.Responses[0]
	if success == nil {
		return false
	}
	return len(success.Headers) > 0 || len(success.Cookies) > 0
}

func writePayloadIDInjection(b *strings.Builder, indent string, payload *httpcodegen.PayloadData) {
	if payload.IDAttribute == "" {
		return
	}
	if payload.IDAttributeRequired {
		fmt.Fprintf(b, "%sif req.ID != nil {\n", indent)
		fmt.Fprintf(b, "%s\tparams.%s = jsonrpc.IDToString(req.ID)\n", indent, payload.IDAttribute)
		fmt.Fprintf(b, "%s}\n", indent)
		return
	}
	fmt.Fprintf(b, "%sif req.ID != nil {\n", indent)
	fmt.Fprintf(b, "%s\tidStr := jsonrpc.IDToString(req.ID)\n", indent)
	fmt.Fprintf(b, "%s\tparams.%s = &idStr\n", indent, payload.IDAttribute)
	fmt.Fprintf(b, "%s}\n", indent)
}
