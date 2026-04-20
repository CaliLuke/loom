package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/internal/ssecodegen"
)

func jsonrpcSSEServerStreamSection(ed *httpcodegen.EndpointData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-sse-server-stream", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(renderSSEServerStreamSource(ed)))
	})
}

func renderSSEServerStreamSource(ed *httpcodegen.EndpointData) string {
	return fmt.Sprintf(`
%s
type %s struct {
	// once ensures headers are written once
	once sync.Once
	// encoder is the SSE event encoder
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder
	// w is the HTTP response writer
	w http.ResponseWriter
	// r is the HTTP request
	r *http.Request
	// requestID is the JSON-RPC request ID for sending final response
	requestID any
	// closed indicates if the stream has been closed via SendAndClose
	closed bool
	// mu protects the closed flag
	mu sync.Mutex
}

%s
func (s *%s) initSSEHeaders() {
	s.once.Do(func() {
		s.w.Header().Set("Content-Type", "text/event-stream")
		s.w.Header().Set("Cache-Control", "no-cache")
		s.w.Header().Set("Connection", "keep-alive")
		s.w.Header().Set("X-Accel-Buffering", "no")
		s.w.WriteHeader(http.StatusOK)
	})
}

%s
func (s *%s) open() error {
	s.initSSEHeaders()
	return http.NewResponseController(s.w).Flush()
}

%s
%s
%s
%s
%s
`, codegen.Comment(fmt.Sprintf("%s implements the %s.%s interface using Server-Sent Events.", ed.SSE.StructName, ed.ServicePkgName, ed.Method.ServerStream.Interface)),
		ed.SSE.StructName,
		codegen.Comment("initSSEHeaders initializes the SSE response headers."),
		ed.SSE.StructName,
		codegen.Comment("open commits and flushes the SSE headers before the first application event."),
		ed.SSE.StructName,
		renderSSEEndpointStreamSendSource(ed),
		renderSSEEndpointStreamSendAndCloseSource(ed),
		renderSSEEndpointStreamErrorsSource(ed),
		renderSSEEndpointSendSSEEventSource(ed),
		"",
	)
}

func renderSSEEndpointStreamSendSource(ed *httpcodegen.EndpointData) string {
	bodyInit := streamResultBodyInit("result", ed)
	bodyComment := ""
	if bodyInit != "body := result" {
		bodyComment = "\t// Convert to response body type for proper JSON encoding\n"
	}
	return fmt.Sprintf(`%s
%s
func (s *%s) Send(ctx context.Context, event %s.%sEvent) error {
	// Check if stream is closed
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("stream closed")
	}
	s.mu.Unlock()

	// Type assert to the specific result type
	result, ok := event.(%s)
	if !ok {
		return fmt.Errorf("unexpected event type: %%T", event)
	}

%s	%s
	// Send as notification (no ID)
	message := map[string]any{
		"jsonrpc": "2.0",
		"method":  %q,
		"params":  body,
	}

	return s.sendSSEEvent("message", message)
}
`, codegen.Comment("Send sends a JSON-RPC notification to the client."),
		codegen.Comment("Notifications do not expect a response from the client."),
		ed.SSE.StructName,
		ed.ServicePkgName,
		ed.Method.VarName,
		ed.SSE.EventTypeRef,
		bodyComment,
		bodyInit,
		ed.Method.Name)
}

func renderSSEEndpointStreamSendAndCloseSource(ed *httpcodegen.EndpointData) string {
	bodyInit := streamResultBodyInit("result", ed)
	idResolution := renderSSEEndpointResponseIDResolution(ed)
	bodyComment := ""
	if bodyInit != "body := result" {
		bodyComment = "\t// Convert to response body type for proper JSON encoding\n"
	}
	return fmt.Sprintf(`%s
%s
%s
func (s *%s) SendAndClose(ctx context.Context, event %s.%sEvent) error {
	// Check if stream is already closed
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("stream already closed")
	}
	s.closed = true
	s.mu.Unlock()

	// Type assert to the specific result type
	result, ok := event.(%s)
	if !ok {
		return fmt.Errorf("unexpected event type: %%T", event)
	}

	// Determine the ID to use for the response
	var id any = s.requestID
%s%s	%s
	// Send as response with ID
	message := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  body,
	}

	return s.sendSSEEvent("response", message)
}
`, codegen.Comment("SendAndClose sends a final JSON-RPC response to the client and closes the stream."),
		codegen.Comment("The response will include the original request ID unless the result has an ID field populated."),
		codegen.Comment("After calling this method, no more events can be sent on this stream."),
		ed.SSE.StructName,
		ed.ServicePkgName,
		ed.Method.VarName,
		ed.SSE.EventTypeRef,
		idResolution,
		bodyComment,
		bodyInit)
}

func renderSSEEndpointResponseIDResolution(ed *httpcodegen.EndpointData) string {
	if ed.Result == nil || ed.Result.IDAttribute == "" {
		return ""
	}
	if ed.Result.IDAttributeRequired {
		return fmt.Sprintf(`	if result.%s != "" {
		// Use the ID from the result if provided
		id = result.%s
		// Clear the ID field so it's not duplicated in the result
		result.%s = ""
	}
`, ed.Result.IDAttribute, ed.Result.IDAttribute, ed.Result.IDAttribute)
	}
	return fmt.Sprintf(`	if result.%s != nil && *result.%s != "" {
		// Use the ID from the result if provided
		id = *result.%s
		// Clear the ID field so it's not duplicated in the result
		result.%s = nil
	}
`, ed.Result.IDAttribute, ed.Result.IDAttribute, ed.Result.IDAttribute, ed.Result.IDAttribute)
}

func renderSSEEndpointStreamErrorsSource(ed *httpcodegen.EndpointData) string {
	noCustomComment := ""
	if len(ed.Errors) == 0 {
		noCustomComment = "\t// No custom errors defined - check if it's a validation error, otherwise use internal error\n"
	}
	return fmt.Sprintf(`%s
func (s *%s) SendError(ctx context.Context, id string, err error) error {
%s%s}

%s
func (s *%s) sendError(ctx context.Context, id any, code jsonrpc.Code, message string, data any) error {
	response := jsonrpc.MakeErrorResponse(id, code, message, data)
	return s.sendSSEEvent("message", response)
}
`, codegen.Comment("SendError sends a JSON-RPC error response."),
		ed.SSE.StructName,
		noCustomComment,
		streamErrorSwitch("	return s.sendError(ctx, id, ", ed.Errors),
		codegen.Comment("sendError sends a JSON-RPC error response via SSE."),
		ed.SSE.StructName)
}

func renderSSEEndpointSendSSEEventSource(ed *httpcodegen.EndpointData) string {
	return fmt.Sprintf(`%s
func (s *%s) sendSSEEvent(eventType string, v any) error {
	s.initSSEHeaders()
	%s
}
`, codegen.Comment("sendSSEEvent sends a single SSE event."),
		ed.SSE.StructName,
		indentGeneratedCode(compactGeneratedCode(ssecodegen.WriteAndFlushSource(`loomhttp.WriteJSONSSEEvent(s.w, loomhttp.SSEMessage{Type: eventType}, v)`, "s.w")), "\t"))
}

func jsonrpcSSEServerImplSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-server-sse-stream-impl", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "SSEStream"
		codegen.Doc(stmt, fmt.Sprintf("%s implements the %s.Stream interface for SSE transport.", streamName, data.Service.PkgName))
		stmt.Type().Id(streamName).Struct(jsonrpcSSEStreamFields()...)
		stmt.Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).Id("initSSEHeaders").Params().Block(jsonrpcSSEInitHeadersBody()...)
		stmt.Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("sendSSEEvent").
			Params(jen.Id("eventType").String(), jen.Id("v").Any()).
			Error().
			Block(append([]jen.Code{
				jen.Id("s").Dot("initSSEHeaders").Call(),
			}, ssecodegen.WriteAndFlushBody(
				jen.Id("loomhttp").Dot("WriteJSONSSEEvent").Call(
					jen.Id("s").Dot("w"),
					jen.Id("loomhttp").Dot("SSEMessage").Values(jen.Dict{jen.Id("Type"): jen.Id("eventType")}),
					jen.Id("v"),
				),
				"s.w",
			)...)...)
		stmt.Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("sendError").
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("id").Any(),
				jen.Id("code").Qual("github.com/CaliLuke/loom/jsonrpc", "Code"),
				jen.Id("message").String(),
				jen.Id("data").Any(),
			).
			Error().
			Block(
				jen.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(jen.Id("id"), jen.Id("code"), jen.Id("message"), jen.Id("data")),
				jen.Return(jen.Id("s").Dot("sendSSEEvent").Call(jen.Lit("message"), jen.Id("response"))),
			)
		stmt.Line()
		writeSSEServiceStreamSend(stmt, data, streamName)
		if serviceHasErrors(data.Service.Methods) {
			stmt.Line()
			writeSSEServiceStreamSendError(stmt, streamName)
		}
	})
}

func jsonrpcWebSocketServerSections(data *httpcodegen.ServiceData) []codegen.Section {
	return []codegen.Section{
		jsonrpcWebSocketServerStructSection(data),
		jsonrpcWebSocketServerWrapperSection(data),
		jsonrpcWebSocketServerSendSection(data),
		jsonrpcWebSocketServerRecvSection(data),
		jsonrpcWebSocketServerCloseSection(data),
	}
}

func jsonrpcWebSocketServerStructSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-server-websocket-struct", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		codegen.Doc(stmt, fmt.Sprintf("%s implements the Stream interface.", streamName))
		stmt.Type().Id(streamName).StructFunc(func(g *jen.Group) {
			for _, ed := range data.Endpoints {
				g.Comment(codegen.Comment(fmt.Sprintf("%s decodes requests for the %s method", lowerInitial(ed.Method.VarName), ed.Method.Name)))
				g.Id(lowerInitial(ed.Method.VarName)).Func().
					Params(jen.Qual("context", "Context"), jen.Op("*").Qual("net/http", "Request"), jen.Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest")).
					Params(jen.Any(), jen.Error())
				if ed.Method.ServerStream != nil && (ed.Method.ServerStream.Kind == expr.ServerStreamKind || ed.Method.ServerStream.Kind == expr.BidirectionalStreamKind) {
					g.Comment(codegen.Comment(fmt.Sprintf("%sEndpoint is the endpoint for the %s method", lowerInitial(ed.Method.VarName), ed.Method.Name)))
					g.Id(lowerInitial(ed.Method.VarName) + "Endpoint").Add(codegen.TypeRef("loom.Endpoint"))
				}
			}
			g.Comment("cancel is the context cancellation function which cancels the request context when invoked.")
			g.Id("cancel").Qual("context", "CancelFunc")
			g.Comment("w is the HTTP response writer used in upgrading the connection.")
			g.Id("w").Qual("net/http", "ResponseWriter")
			g.Comment("r is the HTTP request.")
			g.Id("r").Op("*").Qual("net/http", "Request")
			g.Comment("conn is the underlying websocket connection.")
			g.Id("conn").Op("*").Qual("github.com/gorilla/websocket", "Conn")
		})
	})
}

func jsonrpcWebSocketServerWrapperSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-server-websocket-stream-wrapper", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		first := true
		for _, ed := range data.Endpoints {
			if ed.Method.ServerStream == nil || (ed.Method.ServerStream.Kind != expr.ServerStreamKind && ed.Method.ServerStream.Kind != expr.BidirectionalStreamKind) {
				continue
			}
			if !first {
				stmt.Line()
			}
			first = false
			name := lowerInitial(ed.Method.VarName)
			stmt.Comment(fmt.Sprintf("%sStreamWrapper wraps the JSON-RPC stream to provide a method-specific interface.", name)).Line()
			stmt.Type().Id(name+"StreamWrapper").Struct(
				jen.Id("stream").Op("*").Id(streamName),
				jen.Id("requestID").Any(),
			)
			stmt.Line()
			stmt.Func().Params(jen.Id("w").Op("*").Id(name+"StreamWrapper")).
				Id("SendNotification").
				Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("res").Add(codegen.TypeRef(ed.Result.Ref))).
				Error().
				Block(
					jen.Return(jen.Id("w").Dot("stream").Dot("Send"+ed.Method.VarName+"Notification").Call(jen.Id("ctx"), jen.Id("res"))),
				)
			stmt.Line()
			stmt.Func().Params(jen.Id("w").Op("*").Id(name+"StreamWrapper")).
				Id("SendResponse").
				Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("res").Add(codegen.TypeRef(ed.Result.Ref))).
				Error().
				Block(
					jen.Return(jen.Id("w").Dot("stream").Dot("Send"+ed.Method.VarName+"Response").Call(jen.Id("ctx"), jen.Id("w").Dot("requestID"), jen.Id("res"))),
				)
			stmt.Line()
			stmt.Func().Params(jen.Id("w").Op("*").Id(name+"StreamWrapper")).
				Id("SendError").
				Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("err").Error()).
				Error().
				Block(
					jen.Return(jen.Id("w").Dot("stream").Dot("SendError").Call(jen.Id("ctx"), jen.Id("w").Dot("requestID"), jen.Id("err"))),
				)
			stmt.Line()
			stmt.Func().Params(jen.Id("w").Op("*").Id(name + "StreamWrapper")).
				Id("Close").
				Params().
				Error().
				Block(
					jen.Return(jen.Id("w").Dot("stream").Dot("Close").Call()),
				)
		}
	})
}

func jsonrpcWebSocketServerSendSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-server-websocket-send", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		for _, ed := range data.Endpoints {
			if ed.Result == nil || ed.Result.Ref == "" {
				continue
			}
			addJSONRPCWebSocketResultSendMethods(stmt, streamName, ed)
		}
		sendErrorDecl := &jen.Statement{}
		codegen.Doc(sendErrorDecl, "SendError streams JSON-RPC errors.")
		sendErrorDecl.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("SendError").
			Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("id").Any(), jen.Id("err").Error()).
			Error().
			BlockFunc(func(g *jen.Group) {
				writeStreamErrorDataSwitch(g, allErrors(data), jen.Id("id"))
			})
		stmt.Add(sendErrorDecl)
		stmt.Line()
		sendDecl := &jen.Statement{}
		codegen.Doc(sendDecl, "send writes a JSON-RPC response to the websocket connection.")
		sendDecl.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("send").
			Params(jen.Id("id").Any(), jen.Id("method").String(), jen.Id("result").Any()).
			Error().
			Block(
				jen.If(jen.Id("id").Op("==").Nil().Op("||").Id("id").Op("==").Lit("")).Block(
					jen.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeNotification").Call(jen.Id("method"), jen.Id("result")))),
				),
				jen.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("id"), jen.Id("result")))),
			)
		stmt.Add(sendDecl)
		stmt.Line()
		sendErrorResponseDecl := &jen.Statement{}
		codegen.Doc(sendErrorResponseDecl, "sendError sends a JSON-RPC error response to the websocket connection.")
		sendErrorResponseDecl.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("sendError").
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("id").Any(),
				jen.Id("code").Qual("github.com/CaliLuke/loom/jsonrpc", "Code"),
				jen.Id("message").String(),
				jen.Id("data").Any(),
			).
			Error().
			Block(
				jen.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(jen.Id("id"), jen.Id("code"), jen.Id("message"), jen.Id("data")),
				jen.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(jen.Id("response"))),
			)
		stmt.Add(sendErrorResponseDecl)
	})
}

func jsonrpcSSEStreamFields() []jen.Code {
	return []jen.Code{
		jen.Comment("once ensures the headers are written once."),
		jen.Id("once").Qual("sync", "Once"),
		jen.Comment("w is the HTTP response writer used to send the SSE events."),
		jen.Id("w").Qual("net/http", "ResponseWriter"),
		jen.Comment("r is the HTTP request."),
		jen.Id("r").Op("*").Qual("net/http", "Request"),
		jen.Comment("encoder is the response encoder."),
		jen.Id("encoder").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter")).Add(codegen.TypeRef("loomhttp.Encoder")),
		jen.Comment("decoder is the request decoder."),
		jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Decoder")),
	}
}

func jsonrpcSSEInitHeadersBody() []jen.Code {
	return ssecodegen.InitHeadersBody("s.w", ssecodegen.HeaderOptions{IncludeAccelBuffering: true})
}

func indentGeneratedCode(code, indent string) string {
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}
		lines[i] = indent + line
	}
	return strings.Join(lines, "\n")
}

func compactGeneratedCode(code string) string {
	return strings.Replace(code, "\n\nreturn", "\nreturn", 1)
}

func addJSONRPCWebSocketResultSendMethods(stmt *jen.Statement, streamName string, ed *httpcodegen.EndpointData) {
	addJSONRPCWebSocketSendMethod(stmt, streamName, ed, true)
	stmt.Line()
	addJSONRPCWebSocketSendMethod(stmt, streamName, ed, false)
	stmt.Line()
}

func addJSONRPCWebSocketSendMethod(stmt *jen.Statement, streamName string, ed *httpcodegen.EndpointData, notification bool) {
	methodName, doc := jsonrpcWebSocketSendMethodMeta(ed, notification)
	decl := &jen.Statement{}
	codegen.Doc(decl, doc)
	if notification {
		decl.
			Func().
			Params(jen.Id("s").Op("*").Id(streamName)).
			Id(methodName).
			Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("result").Add(codegen.TypeRef(ed.Result.Ref))).
			Error().
			BlockFunc(func(g *jen.Group) {
				writeStreamResultBodyInit(g, "body", "result", ed)
				g.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(
					jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeNotification").Call(jen.Lit(ed.Method.Name), jen.Id("body")),
				))
			})
		stmt.Add(decl)
		return
	}
	decl.
		Func().
		Params(jen.Id("s").Op("*").Id(streamName)).
		Id(methodName).
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("id").Any(),
			jen.Id("result").Add(codegen.TypeRef(ed.Result.Ref)),
		).
		Error().
		BlockFunc(func(g *jen.Group) {
			writeStreamResultBodyInit(g, "body", "result", ed)
			g.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(
				jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("id"), jen.Id("body")),
			))
		})
	stmt.Add(decl)
}

func jsonrpcWebSocketSendMethodMeta(ed *httpcodegen.EndpointData, notification bool) (string, string) {
	if notification {
		return "Send" + ed.Method.VarName + "Notification",
			fmt.Sprintf("Send%sNotification sends a JSON-RPC notification for the %s method.", ed.Method.VarName, ed.Method.Name)
	}
	return "Send" + ed.Method.VarName + "Response",
		fmt.Sprintf("Send%sResponse sends a JSON-RPC response for the %s method.", ed.Method.VarName, ed.Method.Name)
}

func jsonrpcWebSocketServerRecvSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-server-websocket-recv", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		codegen.Doc(stmt, fmt.Sprintf("Recv reads JSON-RPC requests from the %s service stream.", data.Service.Name))
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("Recv").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Error().
			Block(
				jen.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
				jen.If(
					jen.Err().Op(":=").Id("s").Dot("conn").Dot("ReadJSON").Call(jen.Op("&").Id("req")),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.If(
						jen.Qual("github.com/gorilla/websocket", "IsUnexpectedCloseError").Call(
							jen.Err(),
							jen.Qual("github.com/gorilla/websocket", "CloseGoingAway"),
							jen.Qual("github.com/gorilla/websocket", "CloseAbnormalClosure"),
						),
					).Block(
						jen.Return(jen.Err()),
					),
					jen.If(
						jen.Err().Op(":=").Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Nil(), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "ParseError"), jen.Lit("Parse error"), jen.Nil()),
						jen.Err().Op("!=").Nil(),
					).Block(
						jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send parse error: %w"), jen.Err())),
					),
					jen.Return(jen.Nil()),
				),
				jen.Return(jen.Id("s").Dot("processRequest").Call(jen.Id("ctx"), jen.Op("&").Id("req"))),
			)
		stmt.Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("processRequest").
			Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("req").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest")).
			Error().
			BlockFunc(func(g *jen.Group) {
				writeWebSocketRequestValidation(g)
				g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(sg *jen.Group) {
					for _, ed := range data.Endpoints {
						writeWebSocketRequestCase(sg, ed)
					}
					sg.Default().Block(
						jen.If(jen.Id("req").Dot("HasID")).Block(
							jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"), jen.Lit("Method not found"), jen.Nil())),
						),
						jen.Return(jen.Nil()),
					)
				})
			})
	})
}

func jsonrpcWebSocketServerCloseSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-server-websocket-close", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		codegen.Doc(stmt, fmt.Sprintf("Close closes the %s service websocket connection.", data.Service.Name))
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("Close").
			Params().
			Error().
			Block(
				jen.Var().Id("err").Error(),
				jen.If(jen.Id("s").Dot("conn").Op("==").Nil()).Block(
					jen.Return(jen.Nil()),
				),
				jen.If(
					jen.Id("err").Op("=").Id("s").Dot("conn").Dot("WriteControl").Call(
						jen.Qual("github.com/gorilla/websocket", "CloseMessage"),
						jen.Qual("github.com/gorilla/websocket", "FormatCloseMessage").Call(
							jen.Qual("github.com/gorilla/websocket", "CloseNormalClosure"),
							jen.Lit("server closing connection"),
						),
						jen.Qual("time", "Now").Call().Dot("Add").Call(jen.Qual("time", "Second")),
					),
					jen.Id("err").Op("!=").Nil(),
				).Block(
					jen.Return(jen.Id("err")),
				),
				jen.Return(jen.Id("s").Dot("conn").Dot("Close").Call()),
			)
	})
}

func streamResultBodyInit(resultVar string, ed *httpcodegen.EndpointData) string {
	if ed.Result != nil && len(ed.Result.Responses) > 0 && len(ed.Result.Responses[0].ServerBody) > 0 && ed.Result.Responses[0].ServerBody[0].Init != nil {
		return fmt.Sprintf("body := %s(%s)", ed.Result.Responses[0].ServerBody[0].Init.Name, resultVar)
	}
	return fmt.Sprintf("body := %s", resultVar)
}

func streamErrorSwitch(prefix string, groups []*httpcodegen.ErrorGroupData) string {
	parts := make([]string, 0, len(groups)+8)
	if len(groups) > 0 {
		parts = append(parts,
			"\tvar en loom.LoomErrorNamer\n",
			"\tif !errors.As(err, &en) {\n",
			"\t\tcode := jsonrpc.InternalError\n",
			"\t\tif _, ok := err.(*loom.ServiceError); ok {\n",
			"\t\t\tcode = jsonrpc.InvalidParams\n",
			"\t\t}\n",
			"\t\t"+prefix+"code, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n",
			"\t}\n",
			"\tswitch en.LoomErrorName() {\n",
		)
		for _, gerr := range groups {
			for _, e := range gerr.Errors {
				if e.Response == nil {
					continue
				}
				parts = append(parts,
					fmt.Sprintf("\tcase %q:\n", e.Name),
					fmt.Sprintf("\t\t%s%d, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n", prefix, e.Response.Code),
				)
			}
		}
		parts = append(parts,
			"\tdefault:\n",
			"\t\tcode := jsonrpc.InternalError\n",
			"\t\tif _, ok := err.(*loom.ServiceError); ok {\n",
			"\t\t\tcode = jsonrpc.InvalidParams\n",
			"\t\t}\n",
			"\t\t"+prefix+"code, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n",
			"\t}\n",
		)
		return strings.Join(parts, "")
	}
	parts = append(parts,
		"\tcode := jsonrpc.InternalError\n",
		"\tif _, ok := err.(*loom.ServiceError); ok {\n",
		"\t\tcode = jsonrpc.InvalidParams\n",
		"\t}\n",
		"\t"+prefix+"code, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n",
	)
	return strings.Join(parts, "")
}

func writeStreamResultBodyInit(g *jen.Group, targetName, resultVar string, ed *httpcodegen.EndpointData) {
	if ed.Result != nil && len(ed.Result.Responses) > 0 && len(ed.Result.Responses[0].ServerBody) > 0 && ed.Result.Responses[0].ServerBody[0].Init != nil {
		g.Id(targetName).Op(":=").Id(ed.Result.Responses[0].ServerBody[0].Init.Name).Call(jen.Id(resultVar))
		return
	}
	g.Id(targetName).Op(":=").Id(resultVar)
}

func writeStreamErrorDataSwitch(g *jen.Group, errs []*httpcodegen.ErrorData, targetID jen.Code) {
	writeDefault := func(group *jen.Group) {
		group.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")
		group.If(jen.List(jen.Id("_"), jen.Id("ok")).Op(":=").Id("err").Assert(jen.Op("*").Id("loom").Dot("ServiceError")), jen.Id("ok")).Block(
			jen.Id("code").Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidParams"),
		)
		group.Return(
			jen.Id("s").Dot("sendError").Call(
				jen.Id("ctx"),
				targetID,
				jen.Id("code"),
				jen.Id("loom").Dot("ErrorSafeMessage").Call(jen.Id("err")),
				jen.Qual("github.com/CaliLuke/loom/jsonrpc", "NewErrorData").Call(jen.Id("err")),
			),
		)
	}
	if len(errs) == 0 {
		writeDefault(g)
		return
	}
	g.Var().Id("en").Add(codegen.TypeRef("loom.LoomErrorNamer"))
	g.If(jen.Op("!").Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("en"))).BlockFunc(writeDefault)
	g.Switch(jen.Id("en").Dot("LoomErrorName").Call()).BlockFunc(func(sg *jen.Group) {
		for _, e := range errs {
			if e.Response == nil {
				continue
			}
			sg.Case(jen.Lit(e.Name)).Block(
				jen.Return(
					jen.Id("s").Dot("sendError").Call(
						jen.Id("ctx"),
						targetID,
						jen.Lit(e.Response.Code),
						jen.Id("loom").Dot("ErrorSafeMessage").Call(jen.Id("err")),
						jen.Qual("github.com/CaliLuke/loom/jsonrpc", "NewErrorData").Call(jen.Id("err")),
					),
				),
			)
		}
		sg.Default().BlockFunc(writeDefault)
	})
}

func writeSSEServiceStreamSend(stmt *jen.Statement, data *httpcodegen.ServiceData, streamName string) {
	if !hasAnyStreamingResults(data.Endpoints) {
		return
	}
	codegen.Doc(stmt, "Send sends an event (notification or response) to the client.")
	stmt.Comment("For notifications, the result should not have an ID field.").Line()
	stmt.Comment("For responses, the result must have an ID field.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
		Id("Send").
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("event").Add(codegen.TypeRef(data.Service.PkgName+".Event"))).
		Error().
		BlockFunc(func(g *jen.Group) {
			g.Switch(jen.Id("v").Op(":=").Id("event").Assert(jen.Type())).BlockFunc(func(sg *jen.Group) {
				for _, ed := range dedupeSSEEndpoints(data.Endpoints) {
					if ed.Method.ServerStream == nil || ed.Method.Result == "" {
						continue
					}
					sg.Case(codegen.Expr(ed.SSE.EventTypeRef)).BlockFunc(func(cg *jen.Group) {
						writeStreamResultBodyInit(cg, "body", "v", ed)
						cg.Var().Id("id").String()
						cg.Var().Id("isResponse").Bool()
						writeSSEServiceResponseIDResolution(cg, ed)
						cg.Var().Id("message").Map(jen.String()).Any()
						cg.Var().Id("eventType").String()
						cg.If(jen.Id("isResponse")).Block(
							jen.Id("resp").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("id"), jen.Id("body")),
							jen.Id("message").Op("=").Map(jen.String()).Any().Values(jen.Dict{
								jen.Lit("jsonrpc"): jen.Id("resp").Dot("JSONRPC"),
								jen.Lit("id"):      jen.Id("resp").Dot("ID"),
								jen.Lit("result"):  jen.Id("resp").Dot("Result"),
							}),
							jen.Id("eventType").Op("=").Lit("response"),
						).Else().Block(
							jen.Id("message").Op("=").Map(jen.String()).Any().Values(jen.Dict{
								jen.Lit("jsonrpc"): jen.Lit("2.0"),
								jen.Lit("method"):  jen.Lit(ed.Method.Name),
								jen.Lit("params"):  jen.Id("body"),
							}),
							jen.Id("eventType").Op("=").Lit("message"),
						)
						cg.Return(jen.Id("s").Dot("sendSSEEvent").Call(jen.Id("eventType"), jen.Id("message")))
					})
				}
				sg.Default().Block(
					jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("unknown event type: %T"), jen.Id("event"))),
				)
			})
		})
}

func writeSSEServiceResponseIDResolution(g *jen.Group, ed *httpcodegen.EndpointData) {
	if ed.Result == nil || ed.Result.IDAttribute == "" {
		return
	}
	if ed.Result.IDAttributeRequired {
		g.If(jen.Id("v").Dot(ed.Result.IDAttribute).Op("!=").Lit("")).Block(
			jen.Id("id").Op("=").Id("v").Dot(ed.Result.IDAttribute),
			jen.Id("isResponse").Op("=").True(),
			jen.Id("v").Dot(ed.Result.IDAttribute).Op("=").Lit(""),
		)
		return
	}
	g.If(
		jen.Id("v").Dot(ed.Result.IDAttribute).Op("!=").Nil().Op("&&").
			Op("*").Id("v").Dot(ed.Result.IDAttribute).Op("!=").Lit(""),
	).Block(
		jen.Id("id").Op("=").Op("*").Id("v").Dot(ed.Result.IDAttribute),
		jen.Id("isResponse").Op("=").True(),
		jen.Id("v").Dot(ed.Result.IDAttribute).Op("=").Nil(),
	)
}

func writeSSEServiceStreamSendError(stmt *jen.Statement, streamName string) {
	codegen.Doc(stmt, "SendError sends a JSON-RPC error response.")
	stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
		Id("SendError").
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("id").String(), jen.Id("err").Error()).
		Error().
		Block(
			jen.Var().Id("en").Add(codegen.TypeRef("loom.LoomErrorNamer")),
			jen.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError"),
			jen.Id("message").Op(":=").Id("err").Dot("Error").Call(),
			jen.Var().Id("data").Any(),
			jen.Line(),
			jen.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("en"))).Block(
				jen.Switch(jen.Id("en").Dot("LoomErrorName").Call()).Block(
					jen.Case(jen.Lit("invalid_params")).Block(
						jen.Id("code").Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidParams"),
					),
					jen.Case(jen.Lit("method_not_found")).Block(
						jen.Id("code").Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"),
					),
					jen.Default().Block(
						jen.Id("code").Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError"),
					),
				),
			),
			jen.Line(),
			jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("id"), jen.Id("code"), jen.Id("message"), jen.Id("data"))),
		)
}

func writeWebSocketRequestValidation(g *jen.Group) {
	g.If(jen.Id("req").Dot("JSONRPC").Op("!=").Lit("2.0")).Block(
		jen.If(jen.Id("req").Dot("HasID")).Block(
			jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"), jen.Lit("Invalid request"), jen.Nil())),
		),
		jen.Return(jen.Nil()),
	)
	g.Line()
	g.If(jen.Id("req").Dot("Method").Op("==").Lit("")).Block(
		jen.If(jen.Id("req").Dot("HasID")).Block(
			jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"), jen.Lit("Invalid request"), jen.Nil())),
		),
		jen.Return(jen.Nil()),
	)
	g.Line()
}

//nolint:maintidx // Generated websocket request dispatch is intentionally centralized.
func writeWebSocketRequestCase(g *jen.Group, ed *httpcodegen.EndpointData) {
	if ed.Method.ServerStream != nil && (ed.Method.ServerStream.Kind == expr.ServerStreamKind || ed.Method.ServerStream.Kind == expr.BidirectionalStreamKind) {
		g.Case(jen.Lit(ed.Method.Name)).BlockFunc(func(cg *jen.Group) {
			if ed.Payload != nil && ed.Payload.Ref != "" {
				cg.List(jen.Id("payload"), jen.Err()).Op(":=").Id("s").Dot(lowerInitial(ed.Method.VarName)).Call(jen.Id("ctx"), jen.Id("s").Dot("r"), jen.Id("req"))
			} else {
				cg.List(jen.Id("_"), jen.Err()).Op(":=").Id("s").Dot(lowerInitial(ed.Method.VarName)).Call(jen.Id("ctx"), jen.Id("s").Dot("r"), jen.Id("req"))
			}
			cg.If(jen.Err().Op("!=").Nil()).Block(
				jen.If(jen.Id("req").Dot("HasID")).Block(
					jen.If(
						jen.Id("sendErr").Op(":=").Id("s").Dot("SendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Err()),
						jen.Id("sendErr").Op("!=").Nil(),
					).Block(
						jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send error response: %w"), jen.Id("sendErr"))),
					),
					jen.Return(jen.Nil()),
				),
				jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for "+ed.Method.Name+": %w"), jen.Err())),
			)
			cg.Id("streamWrapper").Op(":=").Op("&").Id(lowerInitial(ed.Method.VarName) + "StreamWrapper").Values(jen.Dict{
				jen.Id("stream"):    jen.Id("s"),
				jen.Id("requestID"): jen.Id("req").Dot("ID"),
			})
			fields := jen.Dict{
				jen.Id("Stream"): jen.Id("streamWrapper"),
			}
			if ed.Payload != nil && ed.Payload.Ref != "" {
				fields[jen.Id("Payload")] = jen.Id("payload").Assert(codegen.TypeRef(ed.Payload.Ref))
			}
			cg.Id("endpointInput").Op(":=").Op("&").Qual(ed.ServicePkgName, ed.Method.ServerStream.EndpointStruct).Values(fields)
			cg.If(
				jen.List(jen.Id("_"), jen.Err()).Op(":=").Id("s").Dot(lowerInitial(ed.Method.VarName)+"Endpoint").Call(jen.Id("ctx"), jen.Id("endpointInput")),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.If(jen.Id("req").Dot("HasID")).Block(
					jen.If(
						jen.Id("sendErr").Op(":=").Id("streamWrapper").Dot("SendError").Call(jen.Id("ctx"), jen.Err()),
						jen.Id("sendErr").Op("!=").Nil(),
					).Block(
						jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send error response: %w"), jen.Id("sendErr"))),
					),
					jen.Return(jen.Nil()),
				),
				jen.Return(jen.Nil()),
			)
			cg.Return(jen.Nil())
		})
		return
	}
	g.Case(jen.Lit(ed.Method.Name)).BlockFunc(func(cg *jen.Group) {
		cg.List(jen.Id("res"), jen.Err()).Op(":=").Id("s").Dot(lowerInitial(ed.Method.VarName)).Call(jen.Id("ctx"), jen.Id("s").Dot("r"), jen.Id("req"))
		cg.If(jen.Err().Op("!=").Nil()).Block(
			jen.If(jen.Id("req").Dot("HasID")).Block(
				jen.If(
					jen.Id("sendErr").Op(":=").Id("s").Dot("SendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Err()),
					jen.Id("sendErr").Op("!=").Nil(),
				).Block(
					jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send error response: %w"), jen.Id("sendErr"))),
				),
			),
			jen.Return(jen.Nil()),
		)
		cg.If(jen.Id("req").Dot("HasID")).Block(
			jen.If(jen.Id("res").Op("==").Nil()).Block(
				jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError"), jen.Lit("Internal error"), jen.Nil())),
			),
			jen.If(
				jen.List(jen.Id("r"), jen.Id("ok")).Op(":=").Id("res").Assert(jen.Op("*").Qual(ed.ServicePkgName, ed.Method.VarName+"Result")),
				jen.Id("ok"),
			).Block(
				jen.If(
					jen.Err().Op(":=").Id("s").Dot("Send"+ed.Method.VarName+"Response").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Id("r")),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("send response error for "+ed.Method.Name+": %w"), jen.Err())),
				),
			).Else().Block(
				jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError"), jen.Lit("Internal error"), jen.Nil())),
			),
		)
		cg.Return(jen.Nil())
	})
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
