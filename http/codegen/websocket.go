package codegen

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
	// WebSocketData contains the data needed to render struct type that
	// implements the server and client stream interfaces.
	WebSocketData struct {
		// VarName is the name of the struct.
		VarName string
		// Type is type of the stream (server or client).
		Type string
		// Interface is the fully qualified name of the interface that
		// the struct implements.
		Interface string
		// Endpoint is endpoint data that defines streaming
		// payload/result.
		Endpoint *EndpointData
		// Payload is the streaming payload type sent via the stream.
		Payload *TypeData
		// Response is the successful response data for the streaming
		// endpoint.
		Response *ResponseData
		// SendName is the name of the send function.
		SendName string
		// SendDesc is the description for the send function.
		SendDesc string
		// SendWithContextName is the name of the send function with context.
		SendWithContextName string
		// SendWithContextDesc is the description for the send function with context.
		SendWithContextDesc string
		// SendTypeName is the fully qualified type name sent through
		// the stream.
		SendTypeName string
		// SendTypeRef is the fully qualified type ref sent through the
		// stream.
		SendTypeRef string
		// RecvName is the name of the receive function.
		RecvName string
		// RecvDesc is the description for the recv function.
		RecvDesc string
		// RecvWithContextName is the name of the receive function with context.
		RecvWithContextName string
		// RecvWithContextDesc is the description for the recv function with context.
		RecvWithContextDesc string
		// RecvTypeName is the fully qualified type name received from
		// the stream.
		RecvTypeName string
		// RecvTypeRef is the fully qualified type ref received from the
		// stream.
		RecvTypeRef string
		// RecvTypeIsPointer is true if the type received from the stream is a
		// array or map. This is needed so that the code reading the stream can
		// use a pointer reference when needed to check whether anything was
		// read (check against the nil value) and in this case return EOF.
		RecvTypeIsPointer bool
		// MustClose indicates whether to generate the Close() function
		// for the stream.
		MustClose bool
		// PkgName is the service package name.
		PkgName string
		// Kind is the kind of the stream (payload, result or
		// bidirectional).
		Kind expr.StreamKind
	}
)

// initWebSocketData initializes the WebSocket related data in ed.
func (sds *ServicesData) initWebSocketData(ed *EndpointData, endpointIR *transportir.Endpoint, sd *ServiceData) {
	if endpointIR == nil || endpointIR.Stream == nil || !endpointIR.Stream.IsWebSocket {
		return
	}
	md := ed.Method
	svc := sd.Service
	streamDesc := service.BuildStreamDescriptor(svc, md, endpointIR.Stream.RequestPayload, endpointIR.Response.Result)
	stream := buildWebSocketStreamData(sds, endpointIR, sd, streamDesc)
	serverMeta, clientMeta := describeWebSocketDirections(ed, endpointIR, stream.serverRecvTypeName)
	ed.ServerWebSocket = &WebSocketData{
		VarName:             md.ServerStream.VarName,
		Interface:           fmt.Sprintf("%s.%s", svc.PkgName, md.ServerStream.Interface), // nolint: namescope -- svc.PkgName is the exact import alias in the emitted server file
		Endpoint:            ed,
		Payload:             stream.serverPayload,
		Response:            ed.Result.Responses[0],
		PkgName:             svc.PkgName,
		Type:                "server",
		Kind:                md.ServerStream.Kind,
		SendName:            md.ServerStream.SendName,
		SendDesc:            serverMeta.sendDesc,
		SendWithContextName: md.ServerStream.SendWithContextName,
		SendWithContextDesc: serverMeta.sendWithContextDesc,
		SendTypeName:        ed.Result.Name,
		SendTypeRef:         ed.Result.Ref,
		RecvName:            md.ServerStream.RecvName,
		RecvDesc:            serverMeta.recvDesc,
		RecvWithContextName: md.ServerStream.RecvWithContextName,
		RecvWithContextDesc: serverMeta.recvWithContextDesc,
		RecvTypeName:        stream.serverRecvTypeName,
		RecvTypeRef:         stream.serverRecvTypeRef,
		RecvTypeIsPointer:   expr.IsArray(endpointIR.Stream.RequestMessage.Type) || expr.IsMap(endpointIR.Stream.RequestMessage.Type),
		MustClose:           md.ServerStream.MustClose,
	}
	ed.ClientWebSocket = &WebSocketData{
		VarName:             md.ClientStream.VarName,
		Interface:           fmt.Sprintf("%s.%s", svc.PkgName, md.ClientStream.Interface), // nolint: namescope -- svc.PkgName is the exact import alias in the emitted client file
		Endpoint:            ed,
		Payload:             stream.clientPayload,
		Response:            ed.Result.Responses[0],
		PkgName:             svc.PkgName,
		Type:                "client",
		Kind:                md.ClientStream.Kind,
		SendName:            md.ClientStream.SendName,
		SendDesc:            clientMeta.sendDesc,
		SendWithContextName: md.ClientStream.SendWithContextName,
		SendWithContextDesc: clientMeta.sendWithContextDesc,
		SendTypeName:        stream.serverRecvTypeName,
		SendTypeRef:         stream.serverRecvTypeRef,
		RecvName:            md.ClientStream.RecvName,
		RecvDesc:            clientMeta.recvDesc,
		RecvWithContextName: md.ClientStream.RecvWithContextName,
		RecvWithContextDesc: clientMeta.recvWithContextDesc,
		RecvTypeName:        ed.Result.Name,
		RecvTypeRef:         ed.Result.Ref,
		MustClose:           md.ClientStream.MustClose,
	}
}

type websocketInitData struct {
	serverRecvTypeName string
	serverRecvTypeRef  string
	serverPayload      *TypeData
	clientPayload      *TypeData
}

type websocketDirectionDescriptions struct {
	sendDesc            string
	sendWithContextDesc string
	recvDesc            string
	recvWithContextDesc string
}

func buildWebSocketStreamData(sds *ServicesData, endpointIR *transportir.Endpoint, sd *ServiceData, streamDesc service.StreamDescriptor) *websocketInitData {
	data := &websocketInitData{}
	if !streamDesc.HasPayload {
		return data
	}
	data.serverRecvTypeName = streamDesc.Payload.Name
	data.serverRecvTypeRef = streamDesc.Payload.Ref
	data.serverPayload = sds.buildRequestBodyType(endpointIR.Request.StreamingBody, endpointIR.Stream.RequestPayload, endpointIR.Name, false, true, sd)
	if needInit(endpointIR.Stream.RequestPayload) {
		initWebSocketPayloadConstructor(data.serverPayload, sds, endpointIR, sd)
	}
	data.clientPayload = sds.buildRequestBodyType(endpointIR.Request.StreamingBody, endpointIR.Stream.RequestPayload, endpointIR.Name, false, false, sd)
	if data.clientPayload != nil {
		sd.ClientTypeNames[data.clientPayload.Name] = false
		sd.ServerTypeNames[data.clientPayload.Name] = false
	}
	return data
}

func initWebSocketPayloadConstructor(payload *TypeData, sds *ServicesData, endpointIR *transportir.Endpoint, sd *ServiceData) {
	body := endpointIR.Request.StreamingBody.Type
	name := websocketPayloadInitName(endpointIR.MethodName, payload.Name)
	desc := fmt.Sprintf("%s builds a %s service %s endpoint payload.", name, sd.Service.Name, endpointIR.MethodName)
	serverArgs := websocketPayloadInitArgs(sds, endpointIR.Request.StreamingBody, sd, body)
	serverCode := ""
	if body != expr.Empty {
		var (
			helpers []*codegen.TransformFunctionData
			err     error
		)
		httpctx := httpContext(sd.Scope, true, true)
		streamBody := makeHTTPType(endpointIR.Request.StreamingBody)
		serverCode, helpers, err = marshal(streamBody, endpointIR.Stream.RequestPayload, "body", "v", httpctx, serviceContext(sd.Service.PkgName, sd.Service.Scope))
		if err == nil {
			sd.ServerTransformHelpers = codegen.AppendHelpers(sd.ServerTransformHelpers, helpers)
		} else {
			panic(codegen.NewError(sds.Ctx, endpointIR.Request.StreamingBody, fmt.Errorf("build WebSocket payload transform for %s: %w", endpointIR.MethodName, err)))
		}
	}
	payload.Init = &InitData{
		Name:           name,
		Description:    desc,
		ServerArgs:     serverArgs,
		ReturnTypeName: sd.Service.Scope.GoFullTypeName(endpointIR.Stream.RequestPayload, sd.Service.PkgName),
		ReturnTypeRef:  sd.Service.Scope.GoFullTypeRef(endpointIR.Stream.RequestPayload, sd.Service.PkgName),
		ReturnIsStruct: expr.IsObject(endpointIR.Stream.RequestPayload.Type),
		ReturnTypePkg:  sd.Service.PkgName,
		ServerCode:     serverCode,
	}
}

func websocketPayloadInitName(methodName, payloadName string) string {
	n := codegen.Goify(methodName, true)
	p := codegen.Goify(payloadName, true)
	if strings.HasPrefix(p, n) {
		return fmt.Sprintf("New%s", p)
	}
	return fmt.Sprintf("New%s%s", n, p)
}

func websocketPayloadInitArgs(sds *ServicesData, streamingBody *expr.AttributeExpr, sd *ServiceData, body expr.DataType) []*InitArgData {
	if body == expr.Empty {
		return nil
	}
	ref := "body"
	if expr.IsObject(body) {
		ref = "&body"
	}
	return []*InitArgData{{
		Ref: ref,
		AttributeData: &AttributeData{
			Name:     "payload",
			VarName:  "body",
			TypeName: sd.Scope.GoTypeName(streamingBody),
			TypeRef:  sd.Scope.GoTypeRef(streamingBody),
			Type:     streamingBody.Type,
			Required: true,
			Example:  streamingBody.Example(sds.Root.API.ExampleGenerator),
			Validate: websocketPayloadValidationCode(body, sd),
		},
	}}
}

func websocketPayloadValidationCode(body expr.DataType, sd *ServiceData) string {
	ut, ok := body.(expr.UserType)
	if !ok || ut.Attribute().Validation == nil {
		return ""
	}
	httpctx := httpContext(sd.Scope, true, true)
	return codegen.ValidationCode(ut.Attribute(), ut, httpctx, true, expr.IsAlias(ut), false, "body")
}

func describeWebSocketDirections(ed *EndpointData, endpointIR *transportir.Endpoint, serverRecvTypeName string) (*websocketDirectionDescriptions, *websocketDirectionDescriptions) {
	md := ed.Method
	resultTypeName := ed.Result.Name
	server := &websocketDirectionDescriptions{
		sendDesc:            fmt.Sprintf("%s streams instances of %q to the %q endpoint websocket connection.", md.ServerStream.SendName, resultTypeName, md.Name),
		sendWithContextDesc: fmt.Sprintf("%s streams instances of %q to the %q endpoint websocket connection with context.", md.ServerStream.SendWithContextName, resultTypeName, md.Name),
		recvDesc:            fmt.Sprintf("%s reads instances of %q from the %q endpoint websocket connection.", md.ServerStream.RecvName, serverRecvTypeName, md.Name),
		recvWithContextDesc: fmt.Sprintf("%s reads instances of %q from the %q endpoint websocket connection with context.", md.ServerStream.RecvWithContextName, serverRecvTypeName, md.Name),
	}
	client := &websocketDirectionDescriptions{
		sendDesc:            fmt.Sprintf("%s streams instances of %q to the %q endpoint websocket connection.", md.ClientStream.SendName, serverRecvTypeName, md.Name),
		sendWithContextDesc: fmt.Sprintf("%s streams instances of %q to the %q endpoint websocket connection with context.", md.ClientStream.SendWithContextName, serverRecvTypeName, md.Name),
		recvDesc:            fmt.Sprintf("%s reads instances of %q from the %q endpoint websocket connection.", md.ClientStream.RecvName, resultTypeName, md.Name),
		recvWithContextDesc: fmt.Sprintf("%s reads instances of %q from the %q endpoint websocket connection with context.", md.ClientStream.RecvWithContextName, resultTypeName, md.Name),
	}
	if endpointIR.Stream.Kind == expr.ClientStreamKind {
		server.sendDesc = fmt.Sprintf("%s streams instances of %q to the %q endpoint websocket connection and closes the connection.", md.ServerStream.SendName, resultTypeName, md.Name)
		server.sendWithContextDesc = fmt.Sprintf("%s streams instances of %q to the %q endpoint websocket connection with context and closes the connection.", md.ServerStream.SendWithContextName, resultTypeName, md.Name)
		client.recvDesc = fmt.Sprintf("%s stops sending messages to the %q endpoint websocket connection and reads instances of %q from the connection.", md.ClientStream.RecvName, md.Name, resultTypeName)
		client.recvWithContextDesc = fmt.Sprintf("%s stops sending messages to the %q endpoint websocket connection and reads instances of %q from the connection with context.", md.ClientStream.RecvWithContextName, md.Name, resultTypeName)
	}
	return server, client
}

// websocketServerFile returns the file implementing the WebSocket server
// streaming implementation if any.
func websocketServerFile(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	if !HasWebSocket(data) {
		return nil
	}
	svcName := data.Service.PathName
	title := fmt.Sprintf("%s WebSocket server streaming", svc.Name())
	imports := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "io"},
		{Path: "net/http"},
		{Path: "sync"},
		{Path: "time"},
		{Path: "github.com/gorilla/websocket"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "loomhttp"),
		{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
	}
	structSections := websocketStructSections(data, false)
	wsSections := websocketCodeSections(data, false)
	sections := make([]codegen.Section, 0, 1+len(structSections)+len(wsSections))
	sections = append(sections, codegen.Header(title, "server", imports))
	for _, section := range structSections {
		sections = append(sections, section)
	}
	for _, section := range wsSections {
		sections = append(sections, section)
	}

	return &codegen.File{
		Path:     filepath.Join(codegen.Gendir, "http", svcName, "server", "websocket.go"),
		Sections: sections,
	}
}

// websocketClientFile returns the file implementing the WebSocket client
// streaming implementation if any.
func websocketClientFile(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	if !HasWebSocket(data) {
		return nil
	}
	svcName := data.Service.PathName
	title := fmt.Sprintf("%s WebSocket client streaming", svc.Name())
	imports := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "io"},
		{Path: "net/http"},
		{Path: "sync"},
		{Path: "time"},
		{Path: "github.com/gorilla/websocket"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "loomhttp"),
		{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
		{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
	}
	structSections := websocketStructSections(data, true)
	wsSections := websocketCodeSections(data, true)
	sections := make([]codegen.Section, 0, 1+len(structSections)+len(wsSections))
	sections = append(sections, codegen.Header(title, "client", imports))
	for _, section := range structSections {
		sections = append(sections, section)
	}
	for _, section := range wsSections {
		sections = append(sections, section)
	}

	return &codegen.File{
		Path:     filepath.Join(codegen.Gendir, "http", svcName, "client", "websocket.go"),
		Sections: sections,
	}
}

// HasWebSocket returns true if at least one of the endpoints in the service
// defines a streaming payload or result.
func HasWebSocket(sd *ServiceData) bool {
	return slices.ContainsFunc(sd.Endpoints, IsWebSocketEndpoint)
}

// IsWebSocketEndpoint returns true if the endpoint defines a streaming payload
// or result.
func IsWebSocketEndpoint(ed *EndpointData) bool {
	return ed.ServerWebSocket != nil || ed.ClientWebSocket != nil
}
