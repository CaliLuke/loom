package codegen

import (
	"fmt"
	"strings"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/codegen/service"
	"goa.design/goa/v3/expr"
	httpcodegen "goa.design/goa/v3/http/codegen"
)

func jsonrpcSSEServerStreamSection(ed *httpcodegen.EndpointData) codegen.Section {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("%s implements the %s.%s interface using Server-Sent Events.", ed.SSE.StructName, ed.ServicePkgName, ed.Method.ServerStream.Interface)))
	fmt.Fprintf(&b, "type %s struct {\n", ed.SSE.StructName)
	b.WriteString("\t// once ensures headers are written once\n")
	b.WriteString("\tonce sync.Once\n")
	b.WriteString("\t// encoder is the SSE event encoder\n")
	b.WriteString("\tencoder func(context.Context, http.ResponseWriter) goahttp.Encoder\n")
	b.WriteString("\t// w is the HTTP response writer\n")
	b.WriteString("\tw http.ResponseWriter\n")
	b.WriteString("\t// r is the HTTP request\n")
	b.WriteString("\tr *http.Request\n")
	b.WriteString("\t// requestID is the JSON-RPC request ID for sending final response\n")
	b.WriteString("\trequestID any\n")
	b.WriteString("\t// closed indicates if the stream has been closed via SendAndClose\n")
	b.WriteString("\tclosed bool\n")
	b.WriteString("\t// mu protects the closed flag\n")
	b.WriteString("\tmu sync.Mutex\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "%s\n", codegen.Comment("initSSEHeaders initializes the SSE response headers."))
	fmt.Fprintf(&b, "func (s *%s) initSSEHeaders() {\n", ed.SSE.StructName)
	b.WriteString("\ts.once.Do(func() {\n")
	b.WriteString("\t\ts.w.Header().Set(\"Content-Type\", \"text/event-stream\")\n")
	b.WriteString("\t\ts.w.Header().Set(\"Cache-Control\", \"no-cache\")\n")
	b.WriteString("\t\ts.w.Header().Set(\"Connection\", \"keep-alive\")\n")
	b.WriteString("\t\ts.w.Header().Set(\"X-Accel-Buffering\", \"no\")\n")
	b.WriteString("\t\ts.w.WriteHeader(http.StatusOK)\n")
	b.WriteString("\t})\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "%s\n", codegen.Comment("open commits and flushes the SSE headers before the first application event."))
	fmt.Fprintf(&b, "func (s *%s) open() error {\n", ed.SSE.StructName)
	b.WriteString("\ts.initSSEHeaders()\n")
	b.WriteString("\treturn http.NewResponseController(s.w).Flush()\n")
	b.WriteString("}\n\n")

	b.WriteString(codegen.Comment("Send sends a JSON-RPC notification to the client.") + "\n")
	b.WriteString(codegen.Comment("Notifications do not expect a response from the client.") + "\n")
	fmt.Fprintf(&b, "func (s *%s) Send(ctx context.Context, event %s.%sEvent) error {\n", ed.SSE.StructName, ed.ServicePkgName, ed.Method.VarName)
	b.WriteString("\t// Check if stream is closed\n")
	b.WriteString("\ts.mu.Lock()\n")
	b.WriteString("\tif s.closed {\n")
	b.WriteString("\t\ts.mu.Unlock()\n")
	b.WriteString("\t\treturn fmt.Errorf(\"stream closed\")\n")
	b.WriteString("\t}\n")
	b.WriteString("\ts.mu.Unlock()\n\n")
	b.WriteString("\t// Type assert to the specific result type\n")
	b.WriteString("\tresult, ok := event.(" + ed.SSE.EventTypeRef + ")\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn fmt.Errorf(\"unexpected event type: %T\", event)\n")
	b.WriteString("\t}\n\n")
	if streamResultBodyInit("result", ed) != "body := result" {
		b.WriteString("\t// Convert to response body type for proper JSON encoding\n")
	}
	fmt.Fprintf(&b, "\t%s\n", streamResultBodyInit("result", ed))
	writeStreamNotificationMessage(&b, ed.Method.Name)
	b.WriteString("\treturn s.sendSSEEvent(\"message\", message)\n")
	b.WriteString("}\n\n")

	b.WriteString(codegen.Comment("SendAndClose sends a final JSON-RPC response to the client and closes the stream.") + "\n")
	b.WriteString(codegen.Comment("The response will include the original request ID unless the result has an ID field populated.") + "\n")
	b.WriteString(codegen.Comment("After calling this method, no more events can be sent on this stream.") + "\n")
	fmt.Fprintf(&b, "func (s *%s) SendAndClose(ctx context.Context, event %s.%sEvent) error {\n", ed.SSE.StructName, ed.ServicePkgName, ed.Method.VarName)
	b.WriteString("\t// Check if stream is already closed\n")
	b.WriteString("\ts.mu.Lock()\n")
	b.WriteString("\tif s.closed {\n")
	b.WriteString("\t\ts.mu.Unlock()\n")
	b.WriteString("\t\treturn fmt.Errorf(\"stream already closed\")\n")
	b.WriteString("\t}\n")
	b.WriteString("\ts.closed = true\n")
	b.WriteString("\ts.mu.Unlock()\n\n")
	b.WriteString("\t// Type assert to the specific result type\n")
	b.WriteString("\tresult, ok := event.(" + ed.SSE.EventTypeRef + ")\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\treturn fmt.Errorf(\"unexpected event type: %T\", event)\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\t// Determine the ID to use for the response\n")
	b.WriteString("\tvar id any = s.requestID\n")
	if ed.Result != nil && ed.Result.IDAttribute != "" {
		if ed.Result.IDAttributeRequired {
			fmt.Fprintf(&b, "\tif result.%s != \"\" {\n", ed.Result.IDAttribute)
			b.WriteString("\t\t// Use the ID from the result if provided\n")
			fmt.Fprintf(&b, "\t\tid = result.%s\n", ed.Result.IDAttribute)
			b.WriteString("\t\t// Clear the ID field so it's not duplicated in the result\n")
			fmt.Fprintf(&b, "\t\tresult.%s = \"\"\n", ed.Result.IDAttribute)
			b.WriteString("\t}\n")
		} else {
			fmt.Fprintf(&b, "\tif result.%s != nil && *result.%s != \"\" {\n", ed.Result.IDAttribute, ed.Result.IDAttribute)
			b.WriteString("\t\t// Use the ID from the result if provided\n")
			fmt.Fprintf(&b, "\t\tid = *result.%s\n", ed.Result.IDAttribute)
			b.WriteString("\t\t// Clear the ID field so it's not duplicated in the result\n")
			fmt.Fprintf(&b, "\t\tresult.%s = nil\n", ed.Result.IDAttribute)
			b.WriteString("\t}\n")
		}
	}
	if streamResultBodyInit("result", ed) != "body := result" {
		b.WriteString("\t// Convert to response body type for proper JSON encoding\n")
	}
	fmt.Fprintf(&b, "\t%s\n", streamResultBodyInit("result", ed))
	writeStreamResponseMessage(&b)
	b.WriteString("\treturn s.sendSSEEvent(\"response\", message)\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "%s\n", codegen.Comment("SendError sends a JSON-RPC error response."))
	fmt.Fprintf(&b, "func (s *%s) SendError(ctx context.Context, id string, err error) error {\n", ed.SSE.StructName)
	if len(ed.Errors) == 0 {
		b.WriteString("\t// No custom errors defined - check if it's a validation error, otherwise use internal error\n")
	}
	b.WriteString(streamErrorSwitch("return s.sendError(ctx, id, ", ed.Errors))
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "%s\n", codegen.Comment("sendError sends a JSON-RPC error response via SSE."))
	fmt.Fprintf(&b, "func (s *%s) sendError(ctx context.Context, id any, code jsonrpc.Code, message string, data any) error {\n", ed.SSE.StructName)
	b.WriteString("\tresponse := jsonrpc.MakeErrorResponse(id, code, message, data)\n")
	b.WriteString("\treturn s.sendSSEEvent(\"message\", response)\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "%s\n", codegen.Comment("sendSSEEvent sends a single SSE event."))
	fmt.Fprintf(&b, "func (s *%s) sendSSEEvent(eventType string, v any) error {\n", ed.SSE.StructName)
	b.WriteString("\ts.initSSEHeaders()\n")
	b.WriteString("\tif err := goahttp.WriteJSONSSEEvent(s.w, goahttp.SSEMessage{Type: eventType}, v); err != nil {\n")
	b.WriteString("\t\treturn err\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn http.NewResponseController(s.w).Flush()\n")
	b.WriteString("}\n")

	return codegen.NewRawSection("jsonrpc-sse-server-stream", b.String())
}

func writeStreamNotificationMessage(b *strings.Builder, methodName string) {
	b.WriteString("\t// Send as notification (no ID)\n")
	b.WriteString("\tmessage := map[string]any{\n")
	b.WriteString("\t\t\"jsonrpc\": \"2.0\",\n")
	fmt.Fprintf(b, "\t\t\"method\":  %q,\n", methodName)
	b.WriteString("\t\t\"params\":  body,\n")
	b.WriteString("\t}\n\n")
}

func writeStreamResponseMessage(b *strings.Builder) {
	b.WriteString("\t// Send as response with ID\n")
	b.WriteString("\tmessage := map[string]any{\n")
	b.WriteString("\t\t\"jsonrpc\": \"2.0\",\n")
	b.WriteString("\t\t\"id\":      id,\n")
	b.WriteString("\t\t\"result\":  body,\n")
	b.WriteString("\t}\n\n")
}

func jsonrpcSSEServerImplSection(data *httpcodegen.ServiceData) codegen.Section {
	var b strings.Builder
	streamName := lowerInitial(data.Service.StructName) + "SSEStream"
	fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("%s implements the %s.Stream interface for SSE transport.", streamName, data.Service.PkgName)))
	fmt.Fprintf(&b, "type %s struct {\n", streamName)
	b.WriteString("\t" + codegen.Comment("once ensures the headers are written once.") + "\n")
	b.WriteString("\tonce sync.Once\n")
	b.WriteString("\t" + codegen.Comment("w is the HTTP response writer used to send the SSE events.") + "\n")
	b.WriteString("\tw http.ResponseWriter\n")
	b.WriteString("\t" + codegen.Comment("r is the HTTP request.") + "\n")
	b.WriteString("\tr *http.Request\n")
	b.WriteString("\t" + codegen.Comment("encoder is the response encoder.") + "\n")
	b.WriteString("\tencoder func(context.Context, http.ResponseWriter) goahttp.Encoder\n")
	b.WriteString("\t" + codegen.Comment("decoder is the request decoder.") + "\n")
	b.WriteString("\tdecoder func(*http.Request) goahttp.Decoder\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "func (s *%s) initSSEHeaders() {\n", streamName)
	b.WriteString("\ts.once.Do(func() {\n")
	b.WriteString("\t\theader := s.w.Header()\n")
	b.WriteString("\t\theader.Set(\"Content-Type\", \"text/event-stream\")\n")
	b.WriteString("\t\theader.Set(\"Cache-Control\", \"no-cache\")\n")
	b.WriteString("\t\theader.Set(\"Connection\", \"keep-alive\")\n")
	b.WriteString("\t\theader.Set(\"X-Accel-Buffering\", \"no\")\n")
	b.WriteString("\t\ts.w.WriteHeader(http.StatusOK)\n")
	b.WriteString("\t})\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "func (s *%s) sendSSEEvent(eventType string, v any) error {\n", streamName)
	b.WriteString("\ts.initSSEHeaders()\n")
	b.WriteString("\tif err := goahttp.WriteJSONSSEEvent(s.w, goahttp.SSEMessage{Type: eventType}, v); err != nil {\n")
	b.WriteString("\t\treturn err\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn http.NewResponseController(s.w).Flush()\n")
	b.WriteString("}\n\n")

	fmt.Fprintf(&b, "func (s *%s) sendError(ctx context.Context, id any, code jsonrpc.Code, message string, data any) error {\n", streamName)
	b.WriteString("\tresponse := jsonrpc.MakeErrorResponse(id, code, message, data)\n")
	b.WriteString("\treturn s.sendSSEEvent(\"message\", response)\n")
	b.WriteString("}\n\n")

	if hasAnyStreamingResults(data.Endpoints) {
		b.WriteString(codegen.Comment("Send sends an event (notification or response) to the client.") + "\n")
		b.WriteString(codegen.Comment("For notifications, the result should not have an ID field.") + "\n")
		b.WriteString(codegen.Comment("For responses, the result must have an ID field.") + "\n")
		fmt.Fprintf(&b, "func (s *%s) Send(ctx context.Context, event %s.Event) error {\n", streamName, data.Service.PkgName)
		b.WriteString("\tswitch v := event.(type) {\n")
		for _, ed := range dedupeSSEEndpoints(data.Endpoints) {
			if ed.Method.ServerStream == nil || ed.Method.Result == "" {
				continue
			}
			fmt.Fprintf(&b, "\tcase %s:\n", ed.SSE.EventTypeRef)
			fmt.Fprintf(&b, "\t\t%s\n", streamResultBodyInit("v", ed))
			b.WriteString("\t\tvar id string\n")
			b.WriteString("\t\tvar isResponse bool\n")
			if ed.Result != nil && ed.Result.IDAttribute != "" {
				if ed.Result.IDAttributeRequired {
					fmt.Fprintf(&b, "\t\tif v.%s != \"\" {\n", ed.Result.IDAttribute)
					fmt.Fprintf(&b, "\t\t\tid = v.%s\n", ed.Result.IDAttribute)
					b.WriteString("\t\t\tisResponse = true\n")
					fmt.Fprintf(&b, "\t\t\tv.%s = \"\"\n", ed.Result.IDAttribute)
					b.WriteString("\t\t}\n")
				} else {
					fmt.Fprintf(&b, "\t\tif v.%s != nil && *v.%s != \"\" {\n", ed.Result.IDAttribute, ed.Result.IDAttribute)
					fmt.Fprintf(&b, "\t\t\tid = *v.%s\n", ed.Result.IDAttribute)
					b.WriteString("\t\t\tisResponse = true\n")
					fmt.Fprintf(&b, "\t\t\tv.%s = nil\n", ed.Result.IDAttribute)
					b.WriteString("\t\t}\n")
				}
			}
			b.WriteString("\t\tvar message map[string]any\n")
			b.WriteString("\t\tvar eventType string\n")
			b.WriteString("\t\tif isResponse {\n")
			b.WriteString("\t\t\tresp := jsonrpc.MakeSuccessResponse(id, body)\n")
			b.WriteString("\t\t\tmessage = map[string]any{\n")
			b.WriteString("\t\t\t\t\"jsonrpc\": resp.JSONRPC,\n")
			b.WriteString("\t\t\t\t\"id\":      resp.ID,\n")
			b.WriteString("\t\t\t\t\"result\":  resp.Result,\n")
			b.WriteString("\t\t\t}\n")
			b.WriteString("\t\t\teventType = \"response\"\n")
			b.WriteString("\t\t} else {\n")
			b.WriteString("\t\t\tmessage = map[string]any{\n")
			b.WriteString("\t\t\t\t\"jsonrpc\": \"2.0\",\n")
			fmt.Fprintf(&b, "\t\t\t\t\"method\":  %q,\n", ed.Method.Name)
			b.WriteString("\t\t\t\t\"params\":  body,\n")
			b.WriteString("\t\t\t}\n")
			b.WriteString("\t\t\teventType = \"message\"\n")
			b.WriteString("\t\t}\n")
			b.WriteString("\t\treturn s.sendSSEEvent(eventType, message)\n")
		}
		b.WriteString("\tdefault:\n")
		b.WriteString("\t\treturn fmt.Errorf(\"unknown event type: %T\", event)\n")
		b.WriteString("\t}\n")
		b.WriteString("}\n\n")
	}

	if serviceHasErrors(data.Service.Methods) {
		b.WriteString("// SendError sends a JSON-RPC error response.\n")
		fmt.Fprintf(&b, "func (s *%s) SendError(ctx context.Context, id string, err error) error {\n", streamName)
		b.WriteString("\tvar en goa.GoaErrorNamer\n")
		b.WriteString("\tcode := jsonrpc.InternalError\n")
		b.WriteString("\tmessage := err.Error()\n")
		b.WriteString("\tvar data any\n\n")
		b.WriteString("\tif errors.As(err, &en) {\n")
		b.WriteString("\t\tswitch en.GoaErrorName() {\n")
		b.WriteString("\t\tcase \"invalid_params\":\n")
		b.WriteString("\t\t\tcode = jsonrpc.InvalidParams\n")
		b.WriteString("\t\tcase \"method_not_found\":\n")
		b.WriteString("\t\t\tcode = jsonrpc.MethodNotFound\n")
		b.WriteString("\t\tdefault:\n")
		b.WriteString("\t\t\tcode = jsonrpc.InternalError\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t}\n\n")
		b.WriteString("\treturn s.sendError(ctx, id, code, message, data)\n")
		b.WriteString("}\n")
	}

	return codegen.NewRawSection("jsonrpc-server-sse-stream-impl", b.String())
}

func jsonrpcWebSocketServerSections(data *httpcodegen.ServiceData) []codegen.Section {
	sections := []codegen.Section{
		jsonrpcWebSocketServerStructSection(data),
		jsonrpcWebSocketServerWrapperSection(data),
		jsonrpcWebSocketServerSendSection(data),
		jsonrpcWebSocketServerRecvSection(data),
		jsonrpcWebSocketServerCloseSection(data),
	}
	return sections
}

func jsonrpcWebSocketServerStructSection(data *httpcodegen.ServiceData) codegen.Section {
	var b strings.Builder
	streamName := lowerInitial(data.Service.StructName) + "Stream"
	fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("%s implements the Stream interface.", streamName)))
	fmt.Fprintf(&b, "type %s struct {\n", streamName)
	for _, ed := range data.Endpoints {
		fmt.Fprintf(&b, "\t%s\n", codegen.Comment(fmt.Sprintf("%s decodes requests for the %s method", lowerInitial(ed.Method.VarName), ed.Method.Name)))
		fmt.Fprintf(&b, "\t%s func(context.Context, *http.Request, *jsonrpc.RawRequest) (any, error)\n", lowerInitial(ed.Method.VarName))
		if ed.Method.ServerStream != nil && (ed.Method.ServerStream.Kind == expr.ServerStreamKind || ed.Method.ServerStream.Kind == expr.BidirectionalStreamKind) {
			fmt.Fprintf(&b, "\t%s\n", codegen.Comment(fmt.Sprintf("%sEndpoint is the endpoint for the %s method", lowerInitial(ed.Method.VarName), ed.Method.Name)))
			fmt.Fprintf(&b, "\t%sEndpoint goa.Endpoint\n", lowerInitial(ed.Method.VarName))
		}
	}
	b.WriteString("\t" + codegen.Comment("cancel is the context cancellation function which cancels the request context when invoked.") + "\n")
	b.WriteString("\tcancel context.CancelFunc\n")
	b.WriteString("\t" + codegen.Comment("w is the HTTP response writer used in upgrading the connection.") + "\n")
	b.WriteString("\tw http.ResponseWriter\n")
	b.WriteString("\t" + codegen.Comment("r is the HTTP request.") + "\n")
	b.WriteString("\tr *http.Request\n")
	b.WriteString("\t" + codegen.Comment("conn is the underlying websocket connection.") + "\n")
	b.WriteString("\tconn *websocket.Conn\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("jsonrpc-server-websocket-struct", b.String())
}

func jsonrpcWebSocketServerWrapperSection(data *httpcodegen.ServiceData) codegen.Section {
	var b strings.Builder
	for _, ed := range data.Endpoints {
		if ed.Method.ServerStream == nil || (ed.Method.ServerStream.Kind != expr.ServerStreamKind && ed.Method.ServerStream.Kind != expr.BidirectionalStreamKind) {
			continue
		}
		name := lowerInitial(ed.Method.VarName)
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		fmt.Fprintf(&b, "\n// %sStreamWrapper wraps the JSON-RPC stream to provide a method-specific interface.\n", name)
		fmt.Fprintf(&b, "type %sStreamWrapper struct {\n", name)
		fmt.Fprintf(&b, "\tstream *%s\n", streamName)
		b.WriteString("\trequestID any\n")
		b.WriteString("}\n\n")
		fmt.Fprintf(&b, "func (w *%sStreamWrapper) SendNotification(ctx context.Context, res %s) error {\n", name, ed.Result.Ref)
		fmt.Fprintf(&b, "\treturn w.stream.Send%sNotification(ctx, res)\n", ed.Method.VarName)
		b.WriteString("}\n\n")
		fmt.Fprintf(&b, "func (w *%sStreamWrapper) SendResponse(ctx context.Context, res %s) error {\n", name, ed.Result.Ref)
		fmt.Fprintf(&b, "\treturn w.stream.Send%sResponse(ctx, w.requestID, res)\n", ed.Method.VarName)
		b.WriteString("}\n\n")
		fmt.Fprintf(&b, "func (w *%sStreamWrapper) SendError(ctx context.Context, err error) error {\n", name)
		b.WriteString("\treturn w.stream.SendError(ctx, w.requestID, err)\n")
		b.WriteString("}\n\n")
		fmt.Fprintf(&b, "func (w *%sStreamWrapper) Close() error {\n", name)
		b.WriteString("\treturn w.stream.Close()\n")
		b.WriteString("}\n")
	}
	return codegen.NewRawSection("jsonrpc-server-websocket-stream-wrapper", b.String())
}

func jsonrpcWebSocketServerSendSection(data *httpcodegen.ServiceData) codegen.Section {
	var b strings.Builder
	streamName := lowerInitial(data.Service.StructName) + "Stream"
	for _, ed := range data.Endpoints {
		if ed.Result == nil || ed.Result.Ref == "" {
			continue
		}
		fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("Send%sNotification sends a JSON-RPC notification for the %s method.", ed.Method.VarName, ed.Method.Name)))
		fmt.Fprintf(&b, "func (s *%s) Send%sNotification(ctx context.Context, result %s) error {\n", streamName, ed.Method.VarName, ed.Result.Ref)
		fmt.Fprintf(&b, "\t%s\n", streamResultBodyInit("result", ed))
		fmt.Fprintf(&b, "\treturn s.conn.WriteJSON(jsonrpc.MakeNotification(%q, body))\n", ed.Method.Name)
		b.WriteString("}\n\n")
		fmt.Fprintf(&b, "%s\n", codegen.Comment(fmt.Sprintf("Send%sResponse sends a JSON-RPC response for the %s method.", ed.Method.VarName, ed.Method.Name)))
		fmt.Fprintf(&b, "func (s *%s) Send%sResponse(ctx context.Context, id any, result %s) error {\n", streamName, ed.Method.VarName, ed.Result.Ref)
		fmt.Fprintf(&b, "\t%s\n", streamResultBodyInit("result", ed))
		b.WriteString("\treturn s.conn.WriteJSON(jsonrpc.MakeSuccessResponse(id, body))\n")
		b.WriteString("}\n")
	}
	b.WriteString("\n" + codegen.Comment("SendError streams JSON-RPC errors.") + "\n")
	fmt.Fprintf(&b, "func (s *%s) SendError(ctx context.Context, id any, err error) error {\n", streamName)
	b.WriteString(streamErrorDataSwitch("return s.sendError(ctx, id, ", allErrors(data)))
	b.WriteString("}\n\n")
	b.WriteString(codegen.Comment("send writes a JSON-RPC response to the websocket connection.") + "\n")
	fmt.Fprintf(&b, "func (s *%s) send(id any, method string, result any) error {\n", streamName)
	b.WriteString("\tif id == nil || id == \"\" {\n")
	b.WriteString("\t\treturn s.conn.WriteJSON(jsonrpc.MakeNotification(method, result))\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn s.conn.WriteJSON(jsonrpc.MakeSuccessResponse(id, result))\n")
	b.WriteString("}\n\n")
	b.WriteString(codegen.Comment("sendError sends a JSON-RPC error response to the websocket connection.") + "\n")
	fmt.Fprintf(&b, "func (s *%s) sendError(ctx context.Context, id any, code jsonrpc.Code, message string, data any) error {\n", streamName)
	b.WriteString("\tresponse := jsonrpc.MakeErrorResponse(id, code, message, data)\n")
	b.WriteString("\treturn s.conn.WriteJSON(response)\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("jsonrpc-server-websocket-send", b.String())
}

func jsonrpcWebSocketServerRecvSection(data *httpcodegen.ServiceData) codegen.Section {
	var b strings.Builder
	streamName := lowerInitial(data.Service.StructName) + "Stream"
	fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("Recv reads JSON-RPC requests from the %s service stream.", data.Service.Name)))
	fmt.Fprintf(&b, "func (s *%s) Recv(ctx context.Context) error {\n", streamName)
	b.WriteString("\tvar req jsonrpc.RawRequest\n")
	b.WriteString("\tif err := s.conn.ReadJSON(&req); err != nil {\n")
	b.WriteString("\t\tif websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {\n")
	b.WriteString("\t\t\treturn err\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif err := s.sendError(ctx, nil, jsonrpc.ParseError, \"Parse error\", nil); err != nil {\n")
	b.WriteString("\t\t\treturn fmt.Errorf(\"failed to send parse error: %w\", err)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn s.processRequest(ctx, &req)\n")
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func (s *%s) processRequest(ctx context.Context, req *jsonrpc.RawRequest) error {\n", streamName)
	b.WriteString("\tif req.JSONRPC != \"2.0\" {\n")
	b.WriteString("\t\tif req.HasID {\n")
	b.WriteString("\t\t\treturn s.sendError(ctx, req.ID, jsonrpc.InvalidRequest, \"Invalid request\", nil)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tif req.Method == \"\" {\n")
	b.WriteString("\t\tif req.HasID {\n")
	b.WriteString("\t\t\treturn s.sendError(ctx, req.ID, jsonrpc.InvalidRequest, \"Invalid request\", nil)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tswitch req.Method {\n")
	for _, ed := range data.Endpoints {
		fmt.Fprintf(&b, "\tcase %q:\n", ed.Method.Name)
		if ed.Method.ServerStream != nil && (ed.Method.ServerStream.Kind == expr.ServerStreamKind || ed.Method.ServerStream.Kind == expr.BidirectionalStreamKind) {
			if ed.Payload != nil && ed.Payload.Ref != "" {
				fmt.Fprintf(&b, "\t\tpayload, err := s.%s(ctx, s.r, req)\n", lowerInitial(ed.Method.VarName))
			} else {
				fmt.Fprintf(&b, "\t\t_, err := s.%s(ctx, s.r, req)\n", lowerInitial(ed.Method.VarName))
			}
			b.WriteString("\t\tif err != nil {\n")
			fmt.Fprintf(&b, "\t\t\treturn fmt.Errorf(\"handler error for %s: %%w\", err)\n", ed.Method.Name)
			b.WriteString("\t\t}\n")
			fmt.Fprintf(&b, "\t\tstreamWrapper := &%sStreamWrapper{\n", lowerInitial(ed.Method.VarName))
			b.WriteString("\t\t\tstream: s,\n")
			b.WriteString("\t\t\trequestID: req.ID,\n")
			b.WriteString("\t\t}\n")
			fmt.Fprintf(&b, "\t\tendpointInput := &%s.%s{\n", ed.ServicePkgName, ed.Method.ServerStream.EndpointStruct)
			if ed.Payload != nil && ed.Payload.Ref != "" {
				fmt.Fprintf(&b, "\t\t\tPayload: payload.(%s),\n", ed.Payload.Ref)
			}
			b.WriteString("\t\t\tStream: streamWrapper,\n")
			b.WriteString("\t\t}\n")
			fmt.Fprintf(&b, "\t\tif _, err := s.%sEndpoint(ctx, endpointInput); err != nil {\n", lowerInitial(ed.Method.VarName))
			b.WriteString("\t\t\tif req.HasID {\n")
			b.WriteString("\t\t\t\tif sendErr := streamWrapper.SendError(ctx, err); sendErr != nil {\n")
			b.WriteString("\t\t\t\t\treturn fmt.Errorf(\"failed to send error response: %w\", sendErr)\n")
			b.WriteString("\t\t\t\t}\n")
			b.WriteString("\t\t\t\treturn nil\n")
			b.WriteString("\t\t\t}\n")
			b.WriteString("\t\t\treturn nil\n")
			b.WriteString("\t\t}\n")
			b.WriteString("\t\treturn nil\n")
		} else {
			fmt.Fprintf(&b, "\t\tres, err := s.%s(ctx, s.r, req)\n", lowerInitial(ed.Method.VarName))
			b.WriteString("\t\tif err != nil {\n")
			b.WriteString("\t\t\tif req.HasID {\n")
			b.WriteString("\t\t\t\tif sendErr := s.SendError(ctx, req.ID, err); sendErr != nil {\n")
			b.WriteString("\t\t\t\t\treturn fmt.Errorf(\"failed to send error response: %w\", sendErr)\n")
			b.WriteString("\t\t\t\t}\n")
			b.WriteString("\t\t\t}\n")
			b.WriteString("\t\t\treturn nil\n")
			b.WriteString("\t\t}\n")
			b.WriteString("\t\tif req.HasID {\n")
			b.WriteString("\t\t\tif res == nil {\n")
			b.WriteString("\t\t\t\treturn s.sendError(ctx, req.ID, jsonrpc.InternalError, \"Internal error\", nil)\n")
			b.WriteString("\t\t\t}\n")
			fmt.Fprintf(&b, "\t\t\tif r, ok := res.(*%s.%sResult); ok {\n", ed.ServicePkgName, ed.Method.VarName)
			fmt.Fprintf(&b, "\t\t\t\tif err := s.Send%sResponse(ctx, req.ID, r); err != nil {\n", ed.Method.VarName)
			fmt.Fprintf(&b, "\t\t\t\t\treturn fmt.Errorf(\"send response error for %s: %%w\", err)\n", ed.Method.Name)
			b.WriteString("\t\t\t\t}\n")
			b.WriteString("\t\t\t} else {\n")
			b.WriteString("\t\t\t\treturn s.sendError(ctx, req.ID, jsonrpc.InternalError, \"Internal error\", nil)\n")
			b.WriteString("\t\t\t}\n")
			b.WriteString("\t\t}\n")
			b.WriteString("\t\treturn nil\n")
		}
	}
	b.WriteString("\tdefault:\n")
	b.WriteString("\t\tif req.HasID {\n")
	b.WriteString("\t\t\treturn s.sendError(ctx, req.ID, jsonrpc.MethodNotFound, \"Method not found\", nil)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("jsonrpc-server-websocket-recv", b.String())
}

func jsonrpcWebSocketServerCloseSection(data *httpcodegen.ServiceData) codegen.Section {
	var b strings.Builder
	streamName := lowerInitial(data.Service.StructName) + "Stream"
	fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("Close closes the %s service websocket connection.", data.Service.Name)))
	fmt.Fprintf(&b, "func (s *%s) Close() error {\n", streamName)
	b.WriteString("\tvar err error\n")
	b.WriteString("\tif s.conn == nil {\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif err = s.conn.WriteControl(\n")
	b.WriteString("\t\twebsocket.CloseMessage,\n")
	b.WriteString("\t\twebsocket.FormatCloseMessage(websocket.CloseNormalClosure, \"server closing connection\"),\n")
	b.WriteString("\t\ttime.Now().Add(time.Second),\n")
	b.WriteString("\t); err != nil {\n")
	b.WriteString("\t\treturn err\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn s.conn.Close()\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("jsonrpc-server-websocket-close", b.String())
}

func streamResultBodyInit(resultVar string, ed *httpcodegen.EndpointData) string {
	if ed.Result != nil && len(ed.Result.Responses) > 0 && len(ed.Result.Responses[0].ServerBody) > 0 && ed.Result.Responses[0].ServerBody[0].Init != nil {
		return fmt.Sprintf("body := %s(%s)", ed.Result.Responses[0].ServerBody[0].Init.Name, resultVar)
	}
	return fmt.Sprintf("body := %s", resultVar)
}

func streamErrorSwitch(prefix string, groups []*httpcodegen.ErrorGroupData) string {
	var b strings.Builder
	if len(groups) > 0 {
		b.WriteString("\tvar en goa.GoaErrorNamer\n")
		b.WriteString("\tif !errors.As(err, &en) {\n")
		b.WriteString("\t\tcode := jsonrpc.InternalError\n")
		b.WriteString("\t\tif _, ok := err.(*goa.ServiceError); ok {\n")
		b.WriteString("\t\t\tcode = jsonrpc.InvalidParams\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\t" + prefix + "code, goa.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n")
		b.WriteString("\t}\n")
		b.WriteString("\tswitch en.GoaErrorName() {\n")
		for _, gerr := range groups {
			for _, e := range gerr.Errors {
				if e.Response == nil {
					continue
				}
				fmt.Fprintf(&b, "\tcase %q:\n", e.Name)
				fmt.Fprintf(&b, "\t\t%s%d, goa.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n", prefix, e.Response.Code)
			}
		}
		b.WriteString("\tdefault:\n")
		b.WriteString("\t\tcode := jsonrpc.InternalError\n")
		b.WriteString("\t\tif _, ok := err.(*goa.ServiceError); ok {\n")
		b.WriteString("\t\t\tcode = jsonrpc.InvalidParams\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\t" + prefix + "code, goa.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n")
		b.WriteString("\t}\n")
		return b.String()
	}
	b.WriteString("\tcode := jsonrpc.InternalError\n")
	b.WriteString("\tif _, ok := err.(*goa.ServiceError); ok {\n")
	b.WriteString("\t\tcode = jsonrpc.InvalidParams\n")
	b.WriteString("\t}\n")
	b.WriteString("\t" + prefix + "code, goa.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n")
	return b.String()
}

func streamErrorDataSwitch(prefix string, errs []*httpcodegen.ErrorData) string {
	var b strings.Builder
	if len(errs) > 0 {
		b.WriteString("\tvar en goa.GoaErrorNamer\n")
		b.WriteString("\tif !errors.As(err, &en) {\n")
		b.WriteString("\t\tcode := jsonrpc.InternalError\n")
		b.WriteString("\t\tif _, ok := err.(*goa.ServiceError); ok {\n")
		b.WriteString("\t\t\tcode = jsonrpc.InvalidParams\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\t" + prefix + "code, goa.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n")
		b.WriteString("\t}\n")
		b.WriteString("\tswitch en.GoaErrorName() {\n")
		for _, e := range errs {
			if e.Response == nil {
				continue
			}
			fmt.Fprintf(&b, "\tcase %q:\n", e.Name)
			fmt.Fprintf(&b, "\t\t%s%d, goa.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n", prefix, e.Response.Code)
		}
		b.WriteString("\tdefault:\n")
		b.WriteString("\t\tcode := jsonrpc.InternalError\n")
		b.WriteString("\t\tif _, ok := err.(*goa.ServiceError); ok {\n")
		b.WriteString("\t\t\tcode = jsonrpc.InvalidParams\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\t" + prefix + "code, goa.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n")
		b.WriteString("\t}\n")
		return b.String()
	}
	b.WriteString("\tcode := jsonrpc.InternalError\n")
	b.WriteString("\tif _, ok := err.(*goa.ServiceError); ok {\n")
	b.WriteString("\t\tcode = jsonrpc.InvalidParams\n")
	b.WriteString("\t}\n")
	b.WriteString("\t" + prefix + "code, goa.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n")
	return b.String()
}

func serviceHasErrors(methods []*service.MethodData) bool {
	for _, m := range methods {
		if len(m.Errors) > 0 {
			return true
		}
	}
	return false
}

func hasAnyStreamingResults(endpoints []*httpcodegen.EndpointData) bool {
	for _, ed := range endpoints {
		if ed.Method.ServerStream != nil && ed.Method.Result != "" {
			return true
		}
	}
	return false
}

func dedupeSSEEndpoints(endpoints []*httpcodegen.EndpointData) []*httpcodegen.EndpointData {
	seen := make(map[string]struct{})
	out := make([]*httpcodegen.EndpointData, 0, len(endpoints))
	for _, e := range endpoints {
		if e == nil || e.SSE == nil || e.SSE.EventTypeRef == "" {
			continue
		}
		if _, ok := seen[e.SSE.EventTypeRef]; ok {
			continue
		}
		seen[e.SSE.EventTypeRef] = struct{}{}
		out = append(out, e)
	}
	return out
}

func jsonrpcMinimalRequestEncoderSection(ed *httpcodegen.EndpointData) codegen.Section {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("Encode%sRequest returns an encoder for requests sent to the %s service %s JSON-RPC method.", ed.Method.VarName, ed.ServiceName, ed.Method.Name)))
	fmt.Fprintf(&b, "func Encode%sRequest(encoder func(*http.Request) goahttp.Encoder) func(*http.Request, any) error {\n", ed.Method.VarName)
	b.WriteString("\treturn func(req *http.Request, v any) error {\n")
	b.WriteString("\t\tid := uuid.New().String()\n")
	b.WriteString("\t\tbody := &jsonrpc.Request{\n")
	b.WriteString("\t\t\tJSONRPC: \"2.0\",\n")
	fmt.Fprintf(&b, "\t\t\tMethod:  %q,\n", ed.Method.Name)
	b.WriteString("\t\t\tID:      id,\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif err := encoder(req).Encode(body); err != nil {\n")
	fmt.Fprintf(&b, "\t\t\treturn goahttp.ErrEncodingError(%q, %q, err)\n", ed.ServiceName, ed.Method.Name)
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("jsonrpc-minimal-request-encoder", b.String())
}

func jsonrpcClientEndpointInitSection(ed *httpcodegen.EndpointData) codegen.Section {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("%s returns an endpoint that makes JSON-RPC requests to the %s service %s method.", ed.EndpointInit, ed.ServiceName, ed.Method.Name)))
	fmt.Fprintf(&b, "func (c *%s) %s() goa.Endpoint {\n", ed.ClientStruct, ed.EndpointInit)
	if !httpcodegen.IsWebSocketEndpoint(ed) {
		b.WriteString("\tvar (\n")
		if ed.RequestEncoder != "" {
			fmt.Fprintf(&b, "\t\tencodeRequest  = %s(c.encoder)\n", ed.RequestEncoder)
		}
		if !httpcodegen.IsSSEEndpoint(ed) {
			fmt.Fprintf(&b, "\t\tdecodeResponse = %s(c.decoder, c.RestoreResponseBody)\n", ed.ResponseDecoder)
		}
		b.WriteString("\t)\n")
	}
	b.WriteString("\treturn func(ctx context.Context, v any) (any, error) {\n")
	if !httpcodegen.IsWebSocketEndpoint(ed) {
		b.WriteString("\t\treq, err := c." + ed.RequestInit.Name + "(ctx")
		if len(ed.RequestInit.ClientArgs) > 0 {
			b.WriteString(", ")
			for i, arg := range ed.RequestInit.ClientArgs {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(arg.Ref)
			}
		}
		b.WriteString(")\n")
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString("\t\t\treturn nil, err\n")
		b.WriteString("\t\t}\n")
		if ed.RequestEncoder != "" {
			b.WriteString("\t\tif err := encodeRequest(req, v); err != nil {\n")
			b.WriteString("\t\t\treturn nil, err\n")
			b.WriteString("\t\t}\n")
		}
	}
	switch {
	case httpcodegen.IsWebSocketEndpoint(ed):
		if ed.ClientWebSocket != nil && ed.ClientWebSocket.RecvName != "" && ed.ClientWebSocket.RecvTypeRef != "" {
			b.WriteString("\t\tdecodeResponse := c.decoder\n")
		}
		b.WriteString("\t\tws, err := c.getConn(ctx)\n")
		b.WriteString("\t\tif err != nil {\n")
		b.WriteString("\t\t\treturn nil, err\n")
		b.WriteString("\t\t}\n\n")
		b.WriteString("\t\tstreamCtx, cancel := context.WithCancel(ctx)\n")
		fmt.Fprintf(&b, "\t\tstream := &%s{\n", ed.ClientWebSocket.VarName)
		b.WriteString("\t\t\tws:     ws,\n")
		b.WriteString("\t\t\tctx:    streamCtx,\n")
		b.WriteString("\t\t\tcancel: cancel,\n")
		b.WriteString("\t\t\tdone:   make(chan struct{}),\n")
		b.WriteString("\t\t\tconfig: c.streamConfig,\n")
		if ed.ClientWebSocket != nil && ed.ClientWebSocket.RecvName != "" && ed.ClientWebSocket.RecvTypeRef != "" {
			b.WriteString("\t\t\tdecoder: decodeResponse,\n")
		}
		b.WriteString("\t\t}\n")
		b.WriteString("\t\tgo stream.responseHandler()\n")
		b.WriteString("\t\treturn stream, nil\n")
	case httpcodegen.IsSSEEndpoint(ed):
		b.WriteString("\t\tresp, err := c.Doer.Do(req)\n")
		b.WriteString("\t\tif err != nil {\n")
		fmt.Fprintf(&b, "\t\t\treturn nil, goahttp.ErrRequestError(%q, %q, err)\n", ed.ServiceName, ed.Method.Name)
		b.WriteString("\t\t}\n\n")
		b.WriteString("\t\tif resp.StatusCode != http.StatusOK {\n")
		b.WriteString("\t\t\tbody, _ := io.ReadAll(resp.Body)\n")
		b.WriteString("\t\t\tresp.Body.Close()\n")
		fmt.Fprintf(&b, "\t\t\treturn nil, goahttp.ErrInvalidResponse(%q, %q, resp.StatusCode, string(body))\n", ed.ServiceName, ed.Method.Name)
		b.WriteString("\t\t}\n\n")
		b.WriteString("\t\tcontentType := resp.Header.Get(\"Content-Type\")\n")
		b.WriteString("\t\tif contentType != \"\" && !strings.HasPrefix(contentType, \"text/event-stream\") {\n")
		b.WriteString("\t\t\tresp.Body.Close()\n")
		b.WriteString("\t\t\treturn nil, fmt.Errorf(\"unexpected content type: %s (expected text/event-stream)\", contentType)\n")
		b.WriteString("\t\t}\n\n")
		fmt.Fprintf(&b, "\t\tstream := &%sClientStream{\n", ed.Method.VarName)
		b.WriteString("\t\t\tresp:    resp,\n")
		b.WriteString("\t\t\treader:  bufio.NewReader(resp.Body),\n")
		b.WriteString("\t\t\tdecoder: c.decoder,\n")
		b.WriteString("\t\t}\n")
		b.WriteString("\t\treturn stream, nil\n")
	default:
		b.WriteString("\t\tresp, err := c.Doer.Do(req)\n")
		b.WriteString("\t\tif err != nil {\n")
		fmt.Fprintf(&b, "\t\t\treturn nil, goahttp.ErrRequestError(%q, %q, err)\n", ed.ServiceName, ed.Method.Name)
		b.WriteString("\t\t}\n")
		b.WriteString("\t\treturn decodeResponse(resp)\n")
	}
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("jsonrpc-client-endpoint-init", b.String())
}

func jsonrpcWebSocketClientConnSection(data *httpcodegen.ServiceData) codegen.Section {
	var b strings.Builder
	fmt.Fprintf(&b, "\n// getConn returns the current WebSocket connection or creates a new one\n")
	fmt.Fprintf(&b, "func (c *%s) getConn(ctx context.Context) (*websocket.Conn, error) {\n", data.ClientStruct)
	b.WriteString("\tc.connMu.RLock()\n")
	b.WriteString("\tconn := c.conn\n")
	b.WriteString("\tif conn != nil {\n")
	b.WriteString("\t\tif err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err == nil {\n")
	b.WriteString("\t\t\tc.connMu.RUnlock()\n")
	b.WriteString("\t\t\treturn conn, nil\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\tc.connMu.RUnlock()\n\n")
	b.WriteString("\tc.connMu.Lock()\n")
	b.WriteString("\tdefer c.connMu.Unlock()\n\n")
	b.WriteString("\tif c.conn != nil {\n")
	b.WriteString("\t\tif err := c.conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(5*time.Second)); err == nil {\n")
	b.WriteString("\t\t\treturn c.conn, nil\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tc.conn.Close()\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\twsScheme := \"ws\"\n")
	b.WriteString("\tif c.scheme == \"https\" {\n")
	b.WriteString("\t\twsScheme = \"wss\"\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\turl := wsScheme + \"://\" + c.host")
	firstPath := ""
	for _, ed := range data.Endpoints {
		for _, route := range ed.Routes {
			if route.Verb == "GET" && route.Path != "/" {
				firstPath = route.Path
				break
			}
		}
		if firstPath != "" {
			break
		}
	}
	if firstPath != "" {
		fmt.Fprintf(&b, " + %q", firstPath)
	}
	b.WriteString("\n\n")
	b.WriteString("\tws, _, err := c.dialer.DialContext(ctx, url, nil)\n")
	b.WriteString("\tif err != nil {\n")
	fmt.Fprintf(&b, "\t\treturn nil, goahttp.ErrRequestError(%q, %q, err)\n", data.Service.Name, "connect")
	b.WriteString("\t}\n\n")
	b.WriteString("\tif c.configfn != nil {\n")
	b.WriteString("\t\tws = c.configfn(ws, nil)\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tc.conn = ws\n")
	b.WriteString("\treturn c.conn, nil\n")
	b.WriteString("}\n\n")
	b.WriteString("// Close closes the WebSocket connection and marks the client as closed\n")
	fmt.Fprintf(&b, "func (c *%s) Close() error {\n", data.ClientStruct)
	b.WriteString("\tif c.closed.Swap(true) {\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tc.connMu.Lock()\n")
	b.WriteString("\tdefer c.connMu.Unlock()\n\n")
	b.WriteString("\tif c.conn != nil {\n")
	b.WriteString("\t\terr := c.conn.Close()\n")
	b.WriteString("\t\tc.conn = nil\n")
	b.WriteString("\t\treturn err\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n\n")
	b.WriteString("// IsClosed returns true if the client connection has been closed\n")
	fmt.Fprintf(&b, "func (c *%s) IsClosed() bool {\n", data.ClientStruct)
	b.WriteString("\treturn c.closed.Load()\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("jsonrpc-client-websocket-conn", b.String())
}

func jsonrpcWebSocketStreamErrorTypesSection() codegen.Section {
	source := `
// Stream error types for comprehensive error reporting
type StreamErrorType int

const (
	StreamErrorConnection StreamErrorType = iota // WebSocket connection errors
	StreamErrorProtocol                          // Invalid JSON-RPC protocol
	StreamErrorParsing                           // Failed to parse/decode response
	StreamErrorOrphaned                          // Response with no matching request
	StreamErrorTimeout                           // Request timeout
)

// StreamErrorHandler allows users to handle stream errors
type StreamErrorHandler func(ctx context.Context, errorType StreamErrorType, err error, response *jsonrpc.RawResponse)
`
	return codegen.NewRawSection("jsonrpc-websocket-stream-error-types", source)
}
