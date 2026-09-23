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
	// closed records that the terminal response (SendAndClose or SendError)
	// has been issued.
	closed bool
	// mu serializes every stream write with the terminal transition so no
	// event can follow the final response.
	mu sync.Mutex
}

%s
func (s *%s) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return loomhttp.ErrSSEStreamClosed
	}
	return s.writer.Open(ctx)
}

%s
func (s *%s) SendComment(ctx context.Context, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return loomhttp.ErrSSEStreamClosed
	}
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
	bodyInit := sseEventBodyInit("result", ed)
	bodyComment := ""
	if bodyInit != "body := result" {
		bodyComment = "\t// Convert to response body type for proper JSON encoding\n"
	}
	notificationMethod := sseNotificationMethod(ed)
	return fmt.Sprintf(`%s
%s
func (s *%s) Send(ctx context.Context, event %s.%sEvent) error {
%s
%s	%s
	// Send as notification (no ID)
	message := map[string]any{
		"jsonrpc": "2.0",
		"method":  %q,
		"params":  body,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return loomhttp.ErrSSEStreamClosed
	}
	return s.sendSSEEvent(ctx, "message", message)
}
`, codegen.Comment("Send sends a JSON-RPC notification to the client."),
		codegen.Comment("Notifications do not expect a response from the client. Send returns loomhttp.ErrSSEStreamClosed after the terminal response."),
		ed.SSE.StructName,
		ed.ServicePkgName,
		ed.Method.VarName,
		sseEventResult(ed),
		bodyComment,
		bodyInit,
		notificationMethod)
}

// sseEventBodyInit renders the statement that converts the event value in
// resultVar to the JSON-RPC params or result body. It uses the SSE response
// body constructor when the event has one, which excludes primitive,
// collection, and mixed-result events, and the event value otherwise.
func sseEventBodyInit(resultVar string, ed *httpcodegen.EndpointData) string {
	if body := ed.SSE.ResponseBody; body != nil && body.Init != nil {
		return fmt.Sprintf("body := %s(%s)", body.Init.Name, resultVar)
	}
	return fmt.Sprintf("body := %s", resultVar)
}

// sseEventResult renders the statements that bind the method event to
// result. An event that aliases its result type is the result itself;
// otherwise the event interface is asserted to the result type.
func sseEventResult(ed *httpcodegen.EndpointData) string {
	if !ed.Method.ServerStream.SendTypeAcceptsMethods {
		return "\tresult := event\n"
	}
	return fmt.Sprintf(`	// Type assert to the specific result type
	result, ok := event.(%s)
	if !ok {
		return fmt.Errorf("unexpected event type: %%T", event)
	}
`, ed.SSE.EventTypeRef)
}

func sseNotificationMethod(ed *httpcodegen.EndpointData) string {
	if ed.SSE.NotificationMethod != "" {
		return ed.SSE.NotificationMethod
	}
	return ed.ServiceName + "/stream.event"
}

func renderSSEEndpointStreamSendAndCloseSource(ed *httpcodegen.EndpointData) string {
	bodyInit := sseEventBodyInit("result", ed)
	bodyComment := ""
	if bodyInit != "body := result" {
		bodyComment = "\t// Convert to response body type for proper JSON encoding\n"
	}
	return fmt.Sprintf(`%s
%s
%s
func (s *%s) SendAndClose(ctx context.Context, event %s.%sEvent) error {
%s
%s	%s
	return s.complete(ctx, func(commit func(*jsonrpc.Response) error) error {
		return jsonrpc.CompleteStream(ctx, s.requestHasID, s.requestID, body, commit)
	})
}
`, codegen.Comment("SendAndClose sends a final JSON-RPC response to the client and closes the stream."),
		codegen.Comment("The response includes the original request ID. ID-less streams (JSON-RPC notifications and raw GET events/stream listeners) are closed without a final response: the value is discarded and a stream_final_response_suppressed transport event is emitted. Implementations serving GET listeners should Send every value and close instead."),
		codegen.Comment("After calling this method, every later stream operation returns loomhttp.ErrSSEStreamClosed."),
		ed.SSE.StructName,
		ed.ServicePkgName,
		ed.Method.VarName,
		sseEventResult(ed),
		bodyComment,
		bodyInit)
}

func renderSSEEndpointStreamErrorsSource(ed *httpcodegen.EndpointData) string {
	noCustomComment := ""
	if len(ed.Errors) == 0 {
		noCustomComment = "\t// No custom errors defined - check if it's a validation error, otherwise use internal error\n"
	}
	return fmt.Sprintf(`%s
func (s *%s) SendError(ctx context.Context, id any, err error) error {
	return s.complete(ctx, func(commit func(*jsonrpc.Response) error) error {
		return jsonrpc.CompleteStreamError(ctx, s.requestHasID, func() error {
			return commit(s.mappedErrorResponse(id, err))
		})
	})
}

%s
func (s *%s) mappedErrorResponse(id any, err error) *jsonrpc.Response {
%s%s}

%s
func (s *%s) sendError(ctx context.Context, id any, code jsonrpc.Code, message string, data any) error {
	return s.complete(ctx, func(commit func(*jsonrpc.Response) error) error {
		return commit(jsonrpc.MakeErrorResponse(id, code, message, data))
	})
}

%s
func (s *%s) complete(ctx context.Context, run func(commit func(*jsonrpc.Response) error) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return loomhttp.ErrSSEStreamClosed
	}
	committed := false
	err := run(func(response *jsonrpc.Response) error {
		data, err := loomhttp.EncodeSSEData(response)
		if err != nil {
			return err
		}
		committed = true
		return s.writeSSEData(ctx, "message", data)
	})
	if err != nil && !committed {
		return err
	}
	s.closed = true
	if closeErr := s.writer.Close(); closeErr != nil && err == nil {
		return closeErr
	}
	return err
}
	`, codegen.Comment("SendError sends a terminal JSON-RPC error response and closes the stream. ID-less streams are closed without a response. After calling this method, every later stream operation returns loomhttp.ErrSSEStreamClosed."),
		ed.SSE.StructName,
		codegen.Comment("mappedErrorResponse maps err to the designed JSON-RPC error response."),
		ed.SSE.StructName,
		noCustomComment,
		streamErrorSwitch("	return jsonrpc.MakeErrorResponse(id, ", ed.Errors),
		codegen.Comment("sendError sends a terminal JSON-RPC error response via SSE and closes the stream."),
		ed.SSE.StructName,
		codegen.Comment("complete issues the terminal response at most once while holding s.mu. run passes the response to commit, or returns without calling commit to close without a frame. commit encodes the response before the stream becomes terminal, so an encoding failure returns the error and leaves the stream open for a later error response. Once encoding succeeds, the stream is terminal even if the write fails, and the writer is closed so no later event, comment, or open can follow."),
		ed.SSE.StructName)
}

func renderSSEEndpointSendSSEEventSource(ed *httpcodegen.EndpointData) string {
	return fmt.Sprintf(`%s
func (s *%s) sendSSEEvent(ctx context.Context, eventType string, v any) error {
	data, err := loomhttp.EncodeSSEData(v)
	if err != nil {
		return err
	}
	return s.writeSSEData(ctx, eventType, data)
}

%s
func (s *%s) writeSSEData(ctx context.Context, eventType string, data string) error {
	return s.writer.WriteEvent(ctx, func(w io.Writer) error {
		return loomhttp.WriteSSEEvent(w, loomhttp.SSEMessage{Type: eventType, Data: data})
	})
}
`, codegen.Comment("sendSSEEvent encodes v and sends it as a single SSE event."),
		ed.SSE.StructName,
		codegen.Comment("writeSSEData writes one pre-encoded SSE event, checking ctx and the request context first."),
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
			Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("eventType").String(), jen.Id("v").Any()).
			Error().
			Block(
				jen.Return(jen.Id("s").Dot("writer").Dot("WriteEvent").Call(
					jen.Id("ctx"),
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
				jen.Return(jen.Id("s").Dot("sendSSEEvent").Call(jen.Id("ctx"), jen.Lit("message"), jen.Id("response"))),
			)
		stmt.Line()
		writeSSEServiceStreamSend(stmt, data, streamName)
		if serviceHasErrors(data.Service.Methods) {
			stmt.Line()
			writeSSEServiceStreamSendError(stmt, data, streamName)
		}
	})
}
