package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func serviceDefinitionSection(data *Data) codegen.Section {
	return codegen.MustJenniferSection("service", func(stmt *jen.Statement) {
		stmt.Line()
		buildServiceInterface(stmt, data)
		buildAuthorizerInterface(stmt, data)
		buildServiceConstants(stmt, data)
		buildMethodStreams(stmt, data)
		buildJSONRPCStreamSupport(stmt, data)
	})
}

func buildServiceInterface(stmt *jen.Statement, data *Data) {
	codegen.Doc(stmt, data.Description)
	stmt.Type().Id("Service").InterfaceFunc(func(group *jen.Group) {
		if isJSONRPCWebSocketService(data) {
			groupDoc(group, "HandleStream handles the JSON-RPC WebSocket streaming connection. Calling Recv() on the stream will dispatch requests to the appropriate methods below.")
			group.Id("HandleStream").Params(
				jen.Qual("context", "Context"),
				jen.Id("Stream"),
			).Error()
		}
		for _, method := range data.Methods {
			buildServiceMethod(group, method)
		}
	})
	stmt.Line()
}

func buildServiceMethod(group *jen.Group, method *MethodData) {
	groupDoc(group, method.Description)
	if method.SkipResponseBodyEncodeDecode {
		groupDoc(group, "If body implements [io.WriterTo], that implementation will be used instead. Consider [github.com/CaliLuke/loom/pkg.SkipResponseWriter] to adapt existing implementations.")
	}
	addViewedResultComment(group, method)
	group.Id(method.VarName).
		ParamsFunc(func(params *jen.Group) { addServiceMethodParams(params, method) }).
		ParamsFunc(func(results *jen.Group) { addServiceMethodResults(results, method) })
}

func addServiceMethodParams(params *jen.Group, method *MethodData) {
	params.Qual("context", "Context")
	if method.Payload != "" {
		params.Add(codegen.TypeRef(method.PayloadRef))
	}
	if method.ServerStream != nil {
		addStreamingServiceMethodParams(params, method)
		return
	}
	if method.SkipRequestBodyEncodeDecode {
		params.Qual("io", "ReadCloser")
	}
}

func addStreamingServiceMethodParams(params *jen.Group, method *MethodData) {
	switch {
	case method.IsJSONRPC && !method.IsJSONRPCSSE && method.ServerStream.Kind == 2:
		return
	case method.HasMixedResults:
		params.Add(codegen.TypeRef(method.ServerStream.Interface))
	case method.IsJSONRPC && !method.IsJSONRPCSSE && method.ServerStream.Kind == 3 && method.PayloadRef != "":
		params.Add(codegen.TypeRef(method.PayloadRef))
		params.Add(codegen.TypeRef(method.ServerStream.Interface))
	default:
		params.Add(codegen.TypeRef(method.ServerStream.Interface))
	}
}

func addServiceMethodResults(results *jen.Group, method *MethodData) {
	if method.ServerStream != nil {
		addStreamingServiceMethodResults(results, method)
		return
	}
	addServiceMethodResultValue(results, method)
	if method.SkipResponseBodyEncodeDecode {
		results.Id("body").Qual("io", "ReadCloser")
	}
	addServiceMethodViewResult(results, method)
	results.Id("err").Error()
}

func addStreamingServiceMethodResults(results *jen.Group, method *MethodData) {
	switch {
	case method.IsJSONRPC && !method.IsJSONRPCSSE && method.ServerStream.Kind == 2:
		addServiceMethodResultValue(results, method)
		results.Id("err").Error()
	case method.HasMixedResults:
		addServiceMethodResultValue(results, method)
		addServiceMethodViewResult(results, method)
		results.Id("err").Error()
	default:
		results.Id("err").Error()
	}
}

func addServiceMethodResultValue(results *jen.Group, method *MethodData) {
	if method.Result != "" {
		results.Id("res").Add(codegen.TypeRef(method.ResultRef))
	}
}

func addServiceMethodViewResult(results *jen.Group, method *MethodData) {
	if method.Result != "" && method.ViewedResult != nil && method.ViewedResult.ViewName == "" {
		results.Id("view").String()
	}
}

func addViewedResultComment(group *jen.Group, method *MethodData) {
	if method.ViewedResult == nil || method.ViewedResult.ViewName != "" {
		return
	}
	groupDoc(group, `The "view" return value must have one of the following views`)
	for _, view := range method.ViewedResult.Views {
		line := fmt.Sprintf("- %q", view.Name)
		if view.Description != "" {
			line += ": " + view.Description
		}
		group.Comment("\t" + line)
	}
}

func buildAuthorizerInterface(stmt *jen.Statement, data *Data) {
	if len(data.Schemes) == 0 {
		return
	}
	stmt.Comment("Authorizer defines the authorization functions to be implemented by the service.").Line()
	stmt.Type().Id("Authorizer").InterfaceFunc(func(group *jen.Group) {
		for _, scheme := range data.Schemes.DedupeByType() {
			groupDoc(group, fmt.Sprintf("%sAuth implements the authorization logic for the %s security scheme.", scheme.Type, scheme.Type))
			group.Id(scheme.Type+"Auth").ParamsFunc(func(params *jen.Group) {
				params.Id("ctx").Qual("context", "Context")
				for _, name := range authorizerArgList(scheme.Type) {
					params.Id(name).String()
				}
				params.Id("schema").Op("*").Add(codegen.Expr("security." + scheme.Type + "Scheme"))
			}).Params(
				jen.Qual("context", "Context"),
				jen.Error(),
			)
		}
	})
	stmt.Line()
}

func authorizerArgList(schemeType string) []string {
	switch schemeType {
	case "Basic":
		return []string{"user", "pass"}
	case "APIKey":
		return []string{"key"}
	default:
		return []string{"token"}
	}
}

func buildServiceConstants(stmt *jen.Statement, data *Data) {
	stmt.Comment("APIName is the name of the API as defined in the design.").Line()
	stmt.Const().Id("APIName").Op("=").Lit(data.APIName)
	stmt.Line()
	stmt.Comment("APIVersion is the version of the API as defined in the design.").Line()
	stmt.Const().Id("APIVersion").Op("=").Lit(data.APIVersion)
	stmt.Line()
	stmt.Comment("ServiceName is the name of the service as defined in the design. This is the").Line()
	stmt.Comment("same value that is set in the endpoint request contexts under the ServiceKey").Line()
	stmt.Comment("key.").Line()
	stmt.Const().Id("ServiceName").Op("=").Lit(data.Name)
	stmt.Line()
	stmt.Comment("MethodNames lists the service method names as defined in the design. These").Line()
	stmt.Comment("are the same values that are set in the endpoint request contexts under the").Line()
	stmt.Comment("MethodKey key.").Line()
	stmt.Var().Id("MethodNames").Op("=").Index(jen.Lit(len(data.Methods))).String().ValuesFunc(func(values *jen.Group) {
		for _, method := range data.Methods {
			values.Lit(method.Name)
		}
	})
	stmt.Line()
}

func buildMethodStreams(stmt *jen.Statement, data *Data) {
	for _, method := range data.Methods {
		if method.ServerStream == nil {
			continue
		}
		buildStreamInterface(stmt, streamInterfaceData("server", method, method.ServerStream))
		if method.ClientStream == nil {
			continue
		}
		buildStreamInterface(stmt, streamInterfaceData("client", method, method.ClientStream))
	}
}

func buildJSONRPCStreamSupport(stmt *jen.Statement, data *Data) {
	if !hasJSONRPCStreamingData(data) {
		return
	}
	if isJSONRPCWebSocketService(data) {
		buildJSONRPCWebSocketStream(stmt, data)
		return
	}
	buildJSONRPCSSEStream(stmt, data)
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

func buildStreamInterface(stmt *jen.Statement, data *serviceStreamInterfaceData) {
	stream := data.Stream
	if data.IsJSONRPCSSE && data.Type == "server" {
		buildJSONRPCSSEMethodStream(stmt, data, stream)
		return
	}
	elemType := stream.SendTypeRef
	if elemType == "" {
		elemType = stream.RecvTypeRef
	}
	codegen.Doc(stmt, fmt.Sprintf("%s allows streaming instances of %s to the client.", stream.Interface, elemType))
	stmt.Type().Id(stream.Interface).InterfaceFunc(func(group *jen.Group) {
		if stream.SendTypeRef != "" {
			buildStreamSendMethods(group, data, stream)
		}
		if stream.RecvTypeRef != "" && !data.IsJSONRPCWebSocket {
			buildStreamRecvMethods(group, stream)
		}
		buildOptionalStreamMethods(group, data, stream)
	})
	stmt.Line()
}

func buildJSONRPCSSEMethodStream(stmt *jen.Statement, data *serviceStreamInterfaceData, stream *StreamData) {
	codegen.Doc(stmt, fmt.Sprintf("%sEvent is the interface implemented by the result type for the %s method.", data.MethodVarName, data.Endpoint))
	stmt.Type().Id(data.MethodVarName + "Event").Interface(
		jen.Id("is" + data.MethodVarName + "Event").Params(),
	)
	stmt.Line()
	codegen.Doc(stmt, fmt.Sprintf("is%sEvent implements the %sEvent interface.", data.MethodVarName, data.MethodVarName))
	stmt.Func().Params(codegen.TypeRef(stream.SendTypeRef)).Id("is" + data.MethodVarName + "Event").Params().Block()
	stmt.Line()
	codegen.Doc(stmt, fmt.Sprintf("%s allows streaming instances of %s over SSE.", stream.Interface, stream.SendTypeRef))
	stmt.Type().Id(stream.Interface).InterfaceFunc(func(group *jen.Group) {
		if stream.SendTypeRef != "" {
			buildJSONRPCSSESendMethods(group, data, stream)
		}
		groupDoc(group, "SendError sends a JSON-RPC error response.")
		group.Id("SendError").Params(
			jen.Qual("context", "Context"),
			jen.Any(),
			jen.Error(),
		).Error()
	})
	stmt.Line()
}

func buildJSONRPCSSESendMethods(group *jen.Group, data *serviceStreamInterfaceData, stream *StreamData) {
	groupDoc(group, stream.SendDesc)
	groupDoc(group, "IMPORTANT: Send only sends JSON-RPC notifications. Use SendAndClose to send a final response.")
	group.Id("Send").Params(
		jen.Qual("context", "Context"),
		jen.Id(data.MethodVarName+"Event"),
	).Error()
	if stream.SendAndCloseName == "" {
		return
	}
	groupDoc(group, stream.SendAndCloseDesc)
	groupDoc(group, "The result will be sent as a JSON-RPC response with the original request ID.")
	groupDoc(group, "If the result has an ID field populated, that ID will be used instead of the request ID.")
	group.Id(stream.SendAndCloseName).Params(
		jen.Qual("context", "Context"),
		jen.Id(data.MethodVarName+"Event"),
	).Error()
}

func buildStreamSendMethods(group *jen.Group, data *serviceStreamInterfaceData, stream *StreamData) {
	if data.IsJSONRPCWebSocket {
		buildJSONRPCWebSocketSendMethods(group, stream)
		return
	}
	groupDoc(group, stream.SendDesc)
	group.Id(stream.SendName).Params(codegen.TypeRef(stream.SendTypeRef)).Error()
	groupDoc(group, stream.SendWithContextDesc)
	group.Id(stream.SendWithContextName).Params(
		jen.Qual("context", "Context"),
		codegen.TypeRef(stream.SendTypeRef),
	).Error()
}

func buildJSONRPCWebSocketSendMethods(group *jen.Group, stream *StreamData) {
	groupDoc(group, "SendNotification sends a JSON-RPC notification (no response expected).")
	group.Id("SendNotification").Params(
		jen.Qual("context", "Context"),
		codegen.TypeRef(stream.SendTypeRef),
	).Error()
	groupDoc(group, "SendResponse sends a JSON-RPC response with the original request ID.")
	group.Id("SendResponse").Params(
		jen.Qual("context", "Context"),
		codegen.TypeRef(stream.SendTypeRef),
	).Error()
	groupDoc(group, "SendError sends a JSON-RPC error response.")
	group.Id("SendError").Params(
		jen.Qual("context", "Context"),
		jen.Error(),
	).Error()
}

func buildStreamRecvMethods(group *jen.Group, stream *StreamData) {
	groupDoc(group, stream.RecvDesc)
	group.Id(stream.RecvName).Params().Params(
		codegen.TypeRef(stream.RecvTypeRef),
		jen.Error(),
	)
	groupDoc(group, stream.RecvWithContextDesc)
	group.Id(stream.RecvWithContextName).Params(jen.Qual("context", "Context")).Params(
		codegen.TypeRef(stream.RecvTypeRef),
		jen.Error(),
	)
}

func buildOptionalStreamMethods(group *jen.Group, data *serviceStreamInterfaceData, stream *StreamData) {
	if data.IsJSONRPCWebSocket || stream.MustClose {
		groupDoc(group, "Close closes the stream.")
		group.Id("Close").Params().Error()
	}
	if data.IsViewedResult && data.Type == "server" {
		groupDoc(group, "SetView sets the view used to render the result before streaming.")
		group.Id("SetView").Params(jen.Id("view").String())
	}
}

func buildJSONRPCWebSocketStream(stmt *jen.Statement, data *Data) {
	codegen.Doc(stmt, fmt.Sprintf("Stream defines the interface for managing a WebSocket streaming connection in the %s server. It allows sending results, sending errors, receiving requests, and closing the connection. This interface is used by the service to interact with clients over WebSocket using JSON-RPC.", data.Name))
	stmt.Type().Id("Stream").InterfaceFunc(func(group *jen.Group) {
		for _, method := range data.Methods {
			if method.Result == "" {
				continue
			}
			groupDoc(group, fmt.Sprintf("Send%sNotification sends a JSON-RPC notification for the %s method (no response expected).", method.VarName, method.Name))
			group.Id("Send"+method.VarName+"Notification").Params(
				jen.Qual("context", "Context"),
				codegen.TypeRef(method.ResultRef),
			).Error()
			groupDoc(group, fmt.Sprintf("Send%sResponse sends a JSON-RPC response for the %s method with the given ID.", method.VarName, method.Name))
			group.Id("Send"+method.VarName+"Response").Params(
				jen.Qual("context", "Context"),
				jen.Any(),
				codegen.TypeRef(method.ResultRef),
			).Error()
		}
		groupDoc(group, "SendError sends a JSON-RPC error response.")
		group.Id("SendError").Params(
			jen.Qual("context", "Context"),
			jen.Any(),
			jen.Error(),
		).Error()
		groupDoc(group, fmt.Sprintf("Recv reads JSON-RPC requests from the %s service WebSocket stream and dispatches them to the appropriate method.", data.Name))
		group.Id("Recv").Params(jen.Qual("context", "Context")).Error()
		groupDoc(group, "Close closes the stream.")
		group.Id("Close").Params().Error()
	})
	stmt.Line()
}

func buildJSONRPCSSEStream(stmt *jen.Statement, data *Data) {
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
	codegen.Doc(stmt, fmt.Sprintf("Stream defines the interface for managing an SSE streaming connection in the %s server. It allows sending notifications and final responses. This interface is used by the service to interact with clients over SSE using JSON-RPC.", data.Name))
	stmt.Type().Id("Stream").InterfaceFunc(func(group *jen.Group) {
		if len(resultTypes) > 0 {
			groupDoc(group, "Send sends an event (notification or response) to the client.")
			groupDoc(group, "For notifications, the result should not have an ID field.")
			groupDoc(group, "For responses, the result must have an ID field.")
			groupDoc(group, fmt.Sprintf("Accepted types: %s", strings.Join(resultTypes, ", ")))
			group.Id("Send").Params(
				jen.Qual("context", "Context"),
				jen.Id("Event"),
			).Error()
		}
		if hasErrors {
			groupDoc(group, "SendError sends a JSON-RPC error response.")
			group.Id("SendError").Params(
				jen.Qual("context", "Context"),
				jen.Any(),
				jen.Error(),
			).Error()
		}
	})
	stmt.Line()
	if len(resultTypes) == 0 {
		return
	}
	codegen.Doc(stmt, fmt.Sprintf("Event is the interface implemented by all result types that can be sent via the %s Stream.", data.Name))
	stmt.Type().Id("Event").Interface(
		jen.Id("is" + data.VarName + "Event").Params(),
	)
	stmt.Line()
	for _, method := range dedupeByResult(data.Methods) {
		if method.Result == "" {
			continue
		}
		codegen.Doc(stmt, fmt.Sprintf("is%sEvent implements the Event interface.", data.VarName))
		stmt.Func().Params(codegen.TypeRef(method.ResultRef)).Id("is" + data.VarName + "Event").Params().Block()
		stmt.Line()
	}
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
