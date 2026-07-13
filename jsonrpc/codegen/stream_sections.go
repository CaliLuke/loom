package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcSSEServerStreamSection(ed *httpcodegen.EndpointData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-sse-server-stream", func(stmt *jen.Statement) {
		stmt.Add(codegen.Expr(renderSSEServerStreamSource(ed)))
	})
}

func renderSSEServerStreamSource(ed *httpcodegen.EndpointData) string {
	return fmt.Sprintf(`
%s
type %s struct {
	// writer owns the serialized SSE response lifecycle
	writer *loomhttp.SSEStreamWriter
	// encoder is the SSE event encoder
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder
	// w is the HTTP response writer
	w http.ResponseWriter
	// r is the HTTP request
	r *http.Request
	// requestID is the JSON-RPC request ID for sending final response
	requestID any
	// requestHasID records whether the JSON-RPC request included an ID.
	requestHasID bool
	// closed indicates if the stream has been closed via SendAndClose
	closed bool
	// mu protects the closed flag
	mu sync.Mutex
}

%s
func (s *%s) Open(ctx context.Context) error {
	return s.writer.Open(ctx)
}

%s
func (s *%s) SendComment(ctx context.Context, text string) error {
	return s.writer.SendComment(ctx, text)
}

%s
%s
%s
%s
%s
`, codegen.Comment(fmt.Sprintf("%s implements the %s.%s interface using Server-Sent Events.", ed.SSE.StructName, ed.ServicePkgName, ed.Method.ServerStream.Interface)),
		ed.SSE.StructName,
		codegen.Comment("Open commits and flushes the SSE headers before the first application event."),
		ed.SSE.StructName,
		codegen.Comment("SendComment writes and flushes an SSE heartbeat comment."),
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
	notificationMethod := sseNotificationMethod(ed)
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
		notificationMethod)
}

func sseNotificationMethod(ed *httpcodegen.EndpointData) string {
	if ed.SSE.NotificationMethod != "" {
		return ed.SSE.NotificationMethod
	}
	return ed.ServiceName + "/stream.event"
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
	if !s.requestHasID {
		// ID-less streams (JSON-RPC notifications and raw GET events/stream
		// listeners) must not receive a final response, so the value is
		// discarded; emit an observability event so the suppression is
		// visible to implementations that call SendAndClose with data.
		loomtransport.Observe(s.r.Context(), loomtransport.Event{Kind: loomtransport.EventKindStreamClose, Reason: loomtransport.ReasonStreamFinalResponseSuppressed, Transport: loomtransport.TransportJSONRPC})
		return nil
	}
%s%s	%s
	// Send as a JSON-RPC response message with ID
	message := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  body,
	}

	return s.sendSSEEvent("message", message)
}
`, codegen.Comment("SendAndClose sends a final JSON-RPC response to the client and closes the stream."),
		codegen.Comment("The response includes the original request ID. ID-less streams (JSON-RPC notifications and raw GET events/stream listeners) are closed without a final response: the value is discarded and a stream_final_response_suppressed transport event is emitted. Implementations serving GET listeners should Send every value and close instead."),
		codegen.Comment("After calling this method, no more events can be sent on this stream."),
		ed.SSE.StructName,
		ed.ServicePkgName,
		ed.Method.VarName,
		ed.SSE.EventTypeRef,
		idResolution,
		bodyComment,
		bodyInit)
}

func renderSSEEndpointResponseIDResolution(_ *httpcodegen.EndpointData) string {
	return ""
}

func renderSSEEndpointStreamErrorsSource(ed *httpcodegen.EndpointData) string {
	noCustomComment := ""
	if len(ed.Errors) == 0 {
		noCustomComment = "\t// No custom errors defined - check if it's a validation error, otherwise use internal error\n"
	}
	return fmt.Sprintf(`%s
func (s *%s) SendError(ctx context.Context, id any, err error) error {
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
	return s.writer.WriteEvent(s.r.Context(), func(w io.Writer) error {
		return loomhttp.WriteJSONSSEEvent(w, loomhttp.SSEMessage{Type: eventType}, v)
	})
}
`, codegen.Comment("sendSSEEvent sends a single SSE event."),
		ed.SSE.StructName)
}

func jsonrpcSSEServerImplSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-sse-stream-impl", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "SSEStream"
		codegen.Doc(stmt, fmt.Sprintf("%s implements the %s.Stream interface for SSE transport.", streamName, data.Service.PkgName))
		stmt.Type().Id(streamName).Struct(jsonrpcSSEStreamFields()...)
		stmt.Line()
		codegen.Doc(stmt, "Open commits and flushes the successful SSE response before the first event.")
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).Id("Open").Params(jen.Id("ctx").Qual("context", "Context")).Error().Block(
			jen.Return(jen.Id("s").Dot("writer").Dot("Open").Call(jen.Id("ctx"))),
		)
		stmt.Line()
		codegen.Doc(stmt, "SendComment writes and flushes an SSE heartbeat comment.")
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).Id("SendComment").Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("text").String()).Error().Block(
			jen.Return(jen.Id("s").Dot("writer").Dot("SendComment").Call(jen.Id("ctx"), jen.Id("text"))),
		)
		stmt.Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("sendSSEEvent").
			Params(jen.Id("eventType").String(), jen.Id("v").Any()).
			Error().
			Block(
				jen.Return(jen.Id("s").Dot("writer").Dot("WriteEvent").Call(
					jen.Id("s").Dot("r").Dot("Context").Call(),
					jen.Func().Params(jen.Id("w").Qual("io", "Writer")).Error().Block(
						jen.Return(jen.Id("loomhttp").Dot("WriteJSONSSEEvent").Call(
							jen.Id("w"),
							jen.Id("loomhttp").Dot("SSEMessage").Values(jen.Dict{jen.Id("Type"): jen.Id("eventType")}),
							jen.Id("v"),
						)),
					),
				)),
			)
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
			writeSSEServiceStreamSendError(stmt, data, streamName)
		}
	})
}
