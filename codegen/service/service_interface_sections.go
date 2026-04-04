package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func serviceDefinitionSection(data *Data) codegen.Section {
	return codegen.NewRawSection("service", renderServiceDefinition(data))
}

func renderServiceDefinition(data *Data) string {
	var b strings.Builder
	writeServiceInterface(&b, data)
	writeAuthorizerInterface(&b, data)
	writeServiceConstants(&b, data)
	writeMethodStreams(&b, data)
	writeJSONRPCStreamSupport(&b, data)
	return b.String()
}

func writeServiceInterface(b *strings.Builder, data *Data) {
	b.WriteString("\n")
	b.WriteString(codegen.Comment(data.Description))
	b.WriteString("\n")
	b.WriteString("type Service interface {\n")
	if isJSONRPCWebSocketService(data) {
		b.WriteString(codegen.Indent(codegen.Comment("HandleStream handles the JSON-RPC WebSocket streaming connection. Calling Recv() on the stream will dispatch requests to the appropriate methods below."), "\t"))
		b.WriteString("\n")
		b.WriteString("\tHandleStream(context.Context, Stream) error\n")
	}
	for _, method := range data.Methods {
		writeServiceMethod(b, method)
	}
	b.WriteString("}\n")
}

func writeServiceMethod(b *strings.Builder, method *MethodData) {
	b.WriteString(codegen.Indent(codegen.Comment(method.Description), "\t"))
	b.WriteString("\n")
	if method.SkipResponseBodyEncodeDecode {
		b.WriteString(codegen.Indent(codegen.Comment("\nIf body implements [io.WriterTo], that implementation will be used instead. Consider [github.com/CaliLuke/loom/pkg.SkipResponseWriter] to adapt existing implementations."), "\t"))
		b.WriteString("\n")
	}
	writeViewedResultComment(b, method)
	fmt.Fprintf(b, "\t%s\n", renderServiceMethodSignature(method))
}

func writeViewedResultComment(b *strings.Builder, method *MethodData) {
	if method.ViewedResult == nil || method.ViewedResult.ViewName != "" {
		return
	}
	b.WriteString(codegen.Indent(codegen.Comment("The \"view\" return value must have one of the following views"), "\t"))
	b.WriteString("\n")
	for _, view := range method.ViewedResult.Views {
		if view.Description != "" {
			fmt.Fprintf(b, "\t//\t- %q: %s\n", view.Name, view.Description)
			continue
		}
		fmt.Fprintf(b, "\t//\t- %q\n", view.Name)
	}
}

func writeAuthorizerInterface(b *strings.Builder, data *Data) {
	if len(data.Schemes) == 0 {
		return
	}
	b.WriteString("\n// Authorizer defines the authorization functions to be implemented by the service.\n")
	b.WriteString("type Authorizer interface {\n")
	for _, scheme := range data.Schemes.DedupeByType() {
		b.WriteString(codegen.Indent(codegen.Comment(fmt.Sprintf("%sAuth implements the authorization logic for the %s security scheme.", scheme.Type, scheme.Type)), "\t"))
		b.WriteString("\n")
		fmt.Fprintf(b, "\t%sAuth(ctx context.Context, %s string, schema *security.%sScheme) (context.Context, error)\n", scheme.Type, authorizerArgNames(scheme.Type), scheme.Type)
	}
	b.WriteString("}\n")
}

func authorizerArgNames(schemeType string) string {
	switch schemeType {
	case "Basic":
		return "user, pass"
	case "APIKey":
		return "key"
	default:
		return "token"
	}
}

func writeServiceConstants(b *strings.Builder, data *Data) {
	fmt.Fprintf(b, "\n// APIName is the name of the API as defined in the design.\nconst APIName = %q\n", data.APIName)
	fmt.Fprintf(b, "\n// APIVersion is the version of the API as defined in the design.\nconst APIVersion = %q\n", data.APIVersion)
	b.WriteString("\n// ServiceName is the name of the service as defined in the design. This is the\n// same value that is set in the endpoint request contexts under the ServiceKey\n// key.\n")
	fmt.Fprintf(b, "const ServiceName = %q\n", data.Name)
	b.WriteString("\n// MethodNames lists the service method names as defined in the design. These\n// are the same values that are set in the endpoint request contexts under the\n// MethodKey key.\n")
	b.WriteString("var MethodNames = [")
	fmt.Fprintf(b, "%d]string{ ", len(data.Methods))
	for _, method := range data.Methods {
		fmt.Fprintf(b, "%q, ", method.Name)
	}
	b.WriteString("}\n")
}

func writeMethodStreams(b *strings.Builder, data *Data) {
	for _, method := range data.Methods {
		if method.ServerStream == nil {
			continue
		}
		b.WriteString("\n")
		b.WriteString(renderStreamInterface(streamInterfaceData("server", method, method.ServerStream)))
		if method.ClientStream == nil {
			continue
		}
		b.WriteString("\n")
		b.WriteString(renderStreamInterface(streamInterfaceData("client", method, method.ClientStream)))
	}
}

func writeJSONRPCStreamSupport(b *strings.Builder, data *Data) {
	if !hasJSONRPCStreamingData(data) {
		return
	}
	b.WriteString("\n")
	if isJSONRPCWebSocketService(data) {
		b.WriteString(renderJSONRPCWebSocketStream(data))
		return
	}
	b.WriteString(renderJSONRPCSSEStream(data))
}

type serviceStreamInterfaceData struct {
	Type               string
	Endpoint           string
	Stream             *StreamData
	MethodVarName      string
	IsJSONRPCSSE       bool
	IsJSONRPCWebSocket bool
	IsViewedResult     bool
}

func streamInterfaceData(typ string, method *MethodData, stream *StreamData) *serviceStreamInterfaceData {
	return &serviceStreamInterfaceData{
		Type:               typ,
		Endpoint:           method.Name,
		Stream:             stream,
		MethodVarName:      method.VarName,
		IsJSONRPCSSE:       method.IsJSONRPCSSE && typ == "server",
		IsJSONRPCWebSocket: method.IsJSONRPCWebSocket,
		IsViewedResult:     method.ViewedResult != nil && method.ViewedResult.ViewName == "",
	}
}

func renderServiceMethodSignature(method *MethodData) string {
	var b strings.Builder
	b.WriteString(method.VarName)
	b.WriteString("(context.Context")
	if method.Payload != "" {
		b.WriteString(", ")
		b.WriteString(method.PayloadRef)
	}
	if method.ServerStream != nil {
		switch {
		case method.IsJSONRPC && !method.IsJSONRPCSSE && method.ServerStream.Kind == 2:
			b.WriteString(")")
			b.WriteString(" (")
			if method.Result != "" {
				b.WriteString("res ")
				b.WriteString(method.ResultRef)
				b.WriteString(", ")
			}
			b.WriteString("err error)")
		case method.HasMixedResults:
			b.WriteString(", ")
			b.WriteString(method.ServerStream.Interface)
			b.WriteString(") (")
			if method.Result != "" {
				b.WriteString("res ")
				b.WriteString(method.ResultRef)
				b.WriteString(", ")
			}
			if method.ViewedResult != nil && method.ViewedResult.ViewName == "" {
				b.WriteString("view string, ")
			}
			b.WriteString("err error)")
		case method.IsJSONRPC && !method.IsJSONRPCSSE && method.ServerStream.Kind == 3 && method.PayloadRef != "":
			b.WriteString(", ")
			b.WriteString(method.PayloadRef)
			b.WriteString(", ")
			b.WriteString(method.ServerStream.Interface)
			b.WriteString(") (err error)")
		default:
			b.WriteString(", ")
			b.WriteString(method.ServerStream.Interface)
			b.WriteString(") (err error)")
		}
		return b.String()
	}
	if method.SkipRequestBodyEncodeDecode {
		b.WriteString(", io.ReadCloser")
	}
	b.WriteString(") (")
	if method.Result != "" {
		b.WriteString("res ")
		b.WriteString(method.ResultRef)
		b.WriteString(", ")
	}
	if method.SkipResponseBodyEncodeDecode {
		b.WriteString("body io.ReadCloser, ")
	}
	if method.Result != "" && method.ViewedResult != nil && method.ViewedResult.ViewName == "" {
		b.WriteString("view string, ")
	}
	b.WriteString("err error)")
	return b.String()
}

func renderStreamInterface(data *serviceStreamInterfaceData) string {
	var b strings.Builder
	stream := data.Stream
	if data.IsJSONRPCSSE && data.Type == "server" {
		writeJSONRPCSSEStreamInterface(&b, data, stream)
		return b.String()
	}

	elemType := stream.SendTypeRef
	if elemType == "" {
		elemType = stream.RecvTypeRef
	}
	b.WriteString(codegen.Comment(fmt.Sprintf("%s allows streaming instances of %s to the client.", stream.Interface, elemType)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s interface {\n", stream.Interface)
	if stream.SendTypeRef != "" {
		writeStreamSendMethods(&b, data, stream)
	}
	if stream.RecvTypeRef != "" && !data.IsJSONRPCWebSocket {
		writeStreamRecvMethods(&b, stream)
	}
	writeOptionalStreamMethods(&b, data, stream)
	b.WriteString("}\n")
	return b.String()
}

func writeJSONRPCSSEStreamInterface(b *strings.Builder, data *serviceStreamInterfaceData, stream *StreamData) {
	b.WriteString(codegen.Comment(fmt.Sprintf("%sEvent is the interface implemented by the result type for the %s method.", data.MethodVarName, data.Endpoint)))
	b.WriteString("\n")
	fmt.Fprintf(b, "type %sEvent interface {\n\tis%sEvent()\n}\n\n", data.MethodVarName, data.MethodVarName)
	b.WriteString(codegen.Comment(fmt.Sprintf("is%sEvent implements the %sEvent interface.", data.MethodVarName, data.MethodVarName)))
	b.WriteString("\n")
	fmt.Fprintf(b, "func (%s) is%sEvent() {}\n\n", stream.SendTypeRef, data.MethodVarName)
	b.WriteString(codegen.Comment(fmt.Sprintf("%s allows streaming instances of %s over SSE.", stream.Interface, stream.SendTypeRef)))
	b.WriteString("\n")
	fmt.Fprintf(b, "type %s interface {\n", stream.Interface)
	if stream.SendTypeRef != "" {
		writeJSONRPCSSESendMethods(b, data, stream)
	}
	b.WriteString(codegen.Indent(codegen.Comment("SendError sends a JSON-RPC error response."), "\t"))
	b.WriteString("\n")
	b.WriteString("\tSendError(ctx context.Context, id string, err error) error\n}\n")
}

func writeJSONRPCSSESendMethods(b *strings.Builder, data *serviceStreamInterfaceData, stream *StreamData) {
	b.WriteString(codegen.Indent(codegen.Comment(stream.SendDesc), "\t"))
	b.WriteString("\n")
	b.WriteString(codegen.Indent(codegen.Comment("IMPORTANT: Send only sends JSON-RPC notifications. Use SendAndClose to send a final response."), "\t"))
	b.WriteString("\n")
	fmt.Fprintf(b, "\tSend(ctx context.Context, event %sEvent) error\n", data.MethodVarName)
	if stream.SendAndCloseName == "" {
		return
	}
	b.WriteString(codegen.Indent(codegen.Comment(stream.SendAndCloseDesc), "\t"))
	b.WriteString("\n")
	b.WriteString(codegen.Indent(codegen.Comment("The result will be sent as a JSON-RPC response with the original request ID."), "\t"))
	b.WriteString("\n")
	b.WriteString(codegen.Indent(codegen.Comment("If the result has an ID field populated, that ID will be used instead of the request ID."), "\t"))
	b.WriteString("\n")
	fmt.Fprintf(b, "\t%s(ctx context.Context, event %sEvent) error\n", stream.SendAndCloseName, data.MethodVarName)
}

func writeStreamSendMethods(b *strings.Builder, data *serviceStreamInterfaceData, stream *StreamData) {
	if data.IsJSONRPCWebSocket {
		writeJSONRPCWebSocketSendMethods(b, stream)
		return
	}
	b.WriteString(codegen.Indent(codegen.Comment(stream.SendDesc), "\t"))
	b.WriteString("\n")
	fmt.Fprintf(b, "\t%s(%s) error\n", stream.SendName, stream.SendTypeRef)
	b.WriteString(codegen.Indent(codegen.Comment(stream.SendWithContextDesc), "\t"))
	b.WriteString("\n")
	fmt.Fprintf(b, "\t%s(context.Context, %s) error\n", stream.SendWithContextName, stream.SendTypeRef)
}

func writeJSONRPCWebSocketSendMethods(b *strings.Builder, stream *StreamData) {
	b.WriteString(codegen.Indent(codegen.Comment("SendNotification sends a JSON-RPC notification (no response expected)."), "\t"))
	b.WriteString("\n")
	fmt.Fprintf(b, "\tSendNotification(context.Context, %s) error\n", stream.SendTypeRef)
	b.WriteString(codegen.Indent(codegen.Comment("SendResponse sends a JSON-RPC response with the original request ID."), "\t"))
	b.WriteString("\n")
	fmt.Fprintf(b, "\tSendResponse(context.Context, %s) error\n", stream.SendTypeRef)
	b.WriteString(codegen.Indent(codegen.Comment("SendError sends a JSON-RPC error response."), "\t"))
	b.WriteString("\n")
	b.WriteString("\tSendError(context.Context, error) error\n")
}

func writeStreamRecvMethods(b *strings.Builder, stream *StreamData) {
	b.WriteString(codegen.Indent(codegen.Comment(stream.RecvDesc), "\t"))
	b.WriteString("\n")
	fmt.Fprintf(b, "\t%s() (%s, error)\n", stream.RecvName, stream.RecvTypeRef)
	b.WriteString(codegen.Indent(codegen.Comment(stream.RecvWithContextDesc), "\t"))
	b.WriteString("\n")
	fmt.Fprintf(b, "\t%s(context.Context) (%s, error)\n", stream.RecvWithContextName, stream.RecvTypeRef)
}

func writeOptionalStreamMethods(b *strings.Builder, data *serviceStreamInterfaceData, stream *StreamData) {
	if data.IsJSONRPCWebSocket || stream.MustClose {
		b.WriteString(codegen.Indent(codegen.Comment("Close closes the stream."), "\t"))
		b.WriteString("\n")
		b.WriteString("\tClose() error\n")
	}
	if data.IsViewedResult && data.Type == "server" {
		b.WriteString(codegen.Indent(codegen.Comment("SetView sets the view used to render the result before streaming."), "\t"))
		b.WriteString("\n")
		b.WriteString("\tSetView(view string)\n")
	}
}

func renderJSONRPCWebSocketStream(data *Data) string {
	var b strings.Builder
	b.WriteString(codegen.Comment(fmt.Sprintf("Stream defines the interface for managing a WebSocket streaming connection in the %s server. It allows sending results, sending errors, receiving requests, and closing the connection. This interface is used by the service to interact with clients over WebSocket using JSON-RPC.", data.Name)))
	b.WriteString("\n")
	b.WriteString("type Stream interface {\n")
	for _, method := range data.Methods {
		if method.Result == "" {
			continue
		}
		b.WriteString(codegen.Indent(codegen.Comment(fmt.Sprintf("Send%sNotification sends a JSON-RPC notification for the %s method (no response expected).", method.VarName, method.Name)), "\t"))
		b.WriteString("\n")
		fmt.Fprintf(&b, "\tSend%sNotification(ctx context.Context, result %s) error\n", method.VarName, method.ResultRef)
		b.WriteString(codegen.Indent(codegen.Comment(fmt.Sprintf("Send%sResponse sends a JSON-RPC response for the %s method with the given ID.", method.VarName, method.Name)), "\t"))
		b.WriteString("\n")
		fmt.Fprintf(&b, "\tSend%sResponse(ctx context.Context, id any, result %s) error\n", method.VarName, method.ResultRef)
	}
	b.WriteString(codegen.Indent(codegen.Comment("SendError sends a JSON-RPC error response."), "\t"))
	b.WriteString("\n\tSendError(ctx context.Context, id any, err error) error\n")
	b.WriteString(codegen.Indent(codegen.Comment(fmt.Sprintf("Recv reads JSON-RPC requests from the %s service WebSocket stream and dispatches them to the appropriate method.", data.Name)), "\t"))
	b.WriteString("\n\tRecv(ctx context.Context) error\n")
	b.WriteString(codegen.Indent(codegen.Comment("Close closes the stream."), "\t"))
	b.WriteString("\n\tClose() error\n}\n")
	return b.String()
}

func renderJSONRPCSSEStream(data *Data) string {
	var b strings.Builder
	var resultTypes []string
	hasErrors := false
	for _, method := range dedupeByResult(data.Methods) {
		if method.Result != "" {
			resultTypes = append(resultTypes, method.ResultRef)
		}
	}
	for _, method := range data.Methods {
		if len(method.Errors) > 0 {
			hasErrors = true
			break
		}
	}
	b.WriteString(codegen.Comment(fmt.Sprintf("Stream defines the interface for managing an SSE streaming connection in the %s server. It allows sending notifications and final responses. This interface is used by the service to interact with clients over SSE using JSON-RPC.", data.Name)))
	b.WriteString("\n")
	b.WriteString("type Stream interface {\n")
	if len(resultTypes) > 0 {
		b.WriteString(codegen.Indent(codegen.Comment("Send sends an event (notification or response) to the client."), "\t"))
		b.WriteString("\n")
		b.WriteString(codegen.Indent(codegen.Comment("For notifications, the result should not have an ID field."), "\t"))
		b.WriteString("\n")
		b.WriteString(codegen.Indent(codegen.Comment("For responses, the result must have an ID field."), "\t"))
		b.WriteString("\n")
		b.WriteString(codegen.Indent(codegen.Comment(fmt.Sprintf("Accepted types: %s", strings.Join(resultTypes, ", "))), "\t"))
		b.WriteString("\n\tSend(ctx context.Context, event Event) error\n")
	}
	if hasErrors {
		b.WriteString(codegen.Indent(codegen.Comment("SendError sends a JSON-RPC error response."), "\t"))
		b.WriteString("\n\tSendError(ctx context.Context, id string, err error) error\n")
	}
	b.WriteString("}\n")
	if len(resultTypes) == 0 {
		return b.String()
	}
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("Event is the interface implemented by all result types that can be sent via the %s Stream.", data.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type Event interface {\n\tis%sEvent()\n}\n", data.VarName)
	for _, method := range dedupeByResult(data.Methods) {
		if method.Result == "" {
			continue
		}
		b.WriteString("\n")
		b.WriteString(codegen.Comment(fmt.Sprintf("is%sEvent implements the Event interface.", data.VarName)))
		b.WriteString("\n")
		fmt.Fprintf(&b, "func (%s) is%sEvent() {}\n", method.ResultRef, data.VarName)
	}
	return b.String()
}

func hasJSONRPCStreamingData(data *Data) bool {
	for _, method := range data.Methods {
		if method.IsJSONRPC && method.ServerStream != nil {
			return true
		}
	}
	return false
}

func hasJSONRPCStreaming(data *Data) bool {
	return hasJSONRPCStreamingData(data)
}

func isJSONRPCWebSocketService(data *Data) bool {
	for _, method := range data.Methods {
		if method.IsJSONRPCWebSocket {
			return true
		}
	}
	return false
}
