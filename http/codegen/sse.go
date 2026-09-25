package codegen

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
	// SSEProjectionData describes one event-discriminator result projection.
	SSEProjectionData struct {
		// EventType is the SSE event discriminator value.
		EventType string
		// View is the result view used for the event data.
		View string
	}

	// SSEData contains the data needed to render struct type that
	// implements the server and client stream interface for SSE.
	SSEData struct {
		// StructName is the name of the generated struct which encapsulates the
		// server implementation.
		StructName string
		// Interface is the fully qualified name of the interface that
		// the struct implements.
		Interface string
		// SendName is the name of the send function.
		SendName string
		// SendDesc is the description for the send function.
		SendDesc string
		// SendWithContextName is the name of the send function with context.
		SendWithContextName string
		// SendWithContextDesc is the description for the send function with context.
		SendWithContextDesc string
		// RecvName is the name of the client method to connect to the SSE endpoint.
		RecvName string
		// RecvDesc is the description for the client method.
		RecvDesc string
		// EventTypeRef is the fully qualified type ref for the event type.
		EventTypeRef string
		// EventTypeName is the name of the event type without package qualifier.
		EventTypeName string
		// EventIsStruct indicates whether the SSE method return type is a struct.
		EventIsStruct bool
		// DataFieldTypeRef is the fully qualified type ref for the data field if any.
		DataFieldTypeRef string
		// DataField is the name of the result type event data attribute if any.
		// If empty, the entire result type is used as the data field.
		DataField string
		// IDField is the name of the result type event ID attribute if any.
		// If empty, no id field is included in the event.
		IDField string
		// IDPointer indicates whether IDField is a pointer, that is an
		// optional attribute without a default value.
		IDPointer bool
		// IDDefault is the Go literal of the default value of an optional
		// IDField. The client assigns it when an event omits its id. It is
		// empty when IDField is required or has no default value.
		IDDefault string
		// EventField is the name of the result type event field if any.
		// If empty, no event field is included in the event.
		EventField string
		// EventPointer indicates whether EventField is a pointer, that is an
		// optional attribute without a default value.
		EventPointer bool
		// EventDefault is the Go literal of the default value of an optional
		// EventField. The client assigns it when an event omits its type. It
		// is empty when EventField is required or has no default value.
		EventDefault string
		// RetryField is the name of the result type event retry field if any.
		// If empty, no retry field is included in the event.
		RetryField string
		// RetryPointer indicates whether RetryField is a pointer, that is an
		// optional attribute without a default value.
		RetryPointer bool
		// RequestIDField is the name of the payload field that maps to the Last-Event-ID header if any.
		// If empty, no last event id is included in the request.
		RequestIDField string
		// NotificationMethod is the JSON-RPC method for intermediate SSE events.
		NotificationMethod string
		// RequestIDPointer indicates whether the RequestIDField is a pointer (i.e., optional primitive).
		RequestIDPointer bool
		// ResponseBody is the server response body whose constructor converts
		// each event before encoding. It is nil when the event value is encoded
		// directly, for example for primitive, collection, or mixed results.
		ResponseBody *TypeData
		// Projections map SSE event discriminator values to result views.
		Projections []*SSEProjectionData
		// ProjectedTypeRef is the generated view projection type reference.
		ProjectedTypeRef string
		// ViewedResultRef is the generated viewed-result wrapper reference.
		ViewedResultRef string
		// ViewedValidateRef validates the viewed-result wrapper.
		ViewedValidateRef string
		// ResultInitRef rebuilds the canonical result from a viewed result.
		ResultInitRef string
	}
)

// initSSEData initializes the SSE related data in ed.
func initSSEData(ed *EndpointData, endpointIR *transportir.Endpoint, sd *ServiceData) {
	if endpointIR == nil || endpointIR.Stream == nil || !endpointIR.Stream.IsSSE {
		return
	}
	md := ed.Method
	svc := sd.Service
	caps := service.DescribeMethodCapabilities(md)

	eventAttr, eventType := sseEventType(ed, endpointIR, sd, caps)

	ed.SSE = &SSEData{
		StructName:          md.ServerStream.VarName,
		Interface:           fmt.Sprintf("%s.%s", svc.PkgName, md.ServerStream.Interface), // nolint: namescope -- svc.PkgName is the exact import alias used in the emitted server file
		SendName:            md.ServerStream.SendName,
		SendDesc:            sseSendDescription(md.ServerStream.SendName, eventType.Name, md.Name),
		SendWithContextName: md.ServerStream.SendWithContextName,
		SendWithContextDesc: sseSendWithContextDescription(md.ServerStream.SendWithContextName, eventType.Name, md.Name),
		RecvName:            md.ClientStream.RecvName,
		RecvDesc:            sseRecvDescription(md.ClientStream.RecvName, md.Name),
		EventTypeRef:        eventType.Ref,
		EventTypeName:       eventType.Name,
		EventIsStruct:       eventType.IsStruct,
		RequestIDField:      endpointIR.Stream.SSE.RequestIDField,
		NotificationMethod:  endpointIR.Stream.SSE.NotificationMethod,
		RequestIDPointer:    endpointIR.Stream.SSE.RequestIDPointer,
	}
	setSSEEventFields(ed.SSE, endpointIR.Stream.SSE, sd, eventAttr)
	for _, projection := range endpointIR.Stream.SSE.Projections {
		ed.SSE.Projections = append(ed.SSE.Projections, &SSEProjectionData{
			EventType: projection.EventType,
			View:      projection.View,
		})
	}
	if len(ed.SSE.Projections) > 0 && md.ViewedResult != nil {
		projected := expr.AsObject(md.ViewedResult.Type.Attribute().Type).Attribute("projected")
		ed.SSE.ProjectedTypeRef = sd.Service.ViewScope.GoFullTypeRef(projected, svc.ViewsPkg)
		ed.SSE.ViewedResultRef = md.ViewedResult.FullRef
		ed.SSE.ViewedValidateRef = svc.ViewsPkg + "." + md.ViewedResult.Validate.Name
		ed.SSE.ResultInitRef = svc.PkgName + "." + md.ViewedResult.ResultInit.Name
	}

	// Mixed results SSE uses the streaming result type for events, not the unary
	// HTTP response body type. Disable HTTP response body conversion in the SSE
	// stream implementation and marshal the event value directly.
	if caps.HasMixedResults {
		return
	}
	ed.SSE.ResponseBody = sseResponseBody(ed)
}

// sseResponseBody returns the server response body used to convert each event,
// or nil when the event value must be encoded directly. Primitive and
// collection results share the service type and have no body constructor.
func sseResponseBody(ed *EndpointData) *TypeData {
	if ed.Result == nil {
		return nil
	}
	for _, resp := range ed.Result.Responses {
		if len(resp.ServerBody) == 0 {
			continue
		}
		body := resp.ServerBody[0]
		if body.Init == nil {
			return nil
		}
		return body
	}
	return nil
}

func sseEventType(ed *EndpointData, endpointIR *transportir.Endpoint, sd *ServiceData, caps service.MethodCapabilityDescriptor) (*expr.AttributeExpr, *ResultData) {
	eventAttr := endpointIR.Response.Result
	if caps.HasMixedResults && endpointIR.Response.StreamingResult != nil {
		eventAttr = endpointIR.Response.StreamingResult
	}
	if !caps.HasMixedResults {
		return eventAttr, ed.Result
	}
	streamDesc := service.BuildStreamDescriptor(sd.Service, ed.Method, nil, eventAttr)
	return eventAttr, &ResultData{
		Name:     streamDesc.Result.Declared.Name,
		Ref:      streamDesc.Result.Declared.Ref,
		IsStruct: expr.IsObject(eventAttr.Type),
	}
}

// setSSEEventFields records the result attributes mapped to the SSE data, id,
// event, and retry fields. The pointer and default flags follow the service
// type field semantics so that generated code dereferences optional fields and
// applies defaults to omitted ones.
func setSSEEventFields(data *SSEData, mapping *transportir.SSE, sd *ServiceData, eventAttr *expr.AttributeExpr) {
	obj := expr.AsObject(eventAttr.Type)
	if obj == nil {
		return
	}
	for _, nat := range *obj {
		switch nat.Name {
		case mapping.IDField:
			data.IDField = codegen.GoifyAtt(nat.Attribute, nat.Name, true)
			data.IDPointer = eventAttr.IsPrimitivePointer(nat.Name, true)
			data.IDDefault = sseStringDefault(eventAttr, nat.Name)
		case mapping.EventField:
			data.EventField = codegen.GoifyAtt(nat.Attribute, nat.Name, true)
			data.EventPointer = eventAttr.IsPrimitivePointer(nat.Name, true)
			data.EventDefault = sseStringDefault(eventAttr, nat.Name)
		case mapping.RetryField:
			data.RetryField = codegen.GoifyAtt(nat.Attribute, nat.Name, true)
			data.RetryPointer = eventAttr.IsPrimitivePointer(nat.Name, true)
		case mapping.DataField:
			data.DataField = codegen.GoifyAtt(nat.Attribute, nat.Name, true)
			data.DataFieldTypeRef = sd.Service.Scope.GoFullTypeRef(nat.Attribute, sd.Service.PkgName)
		}
	}
}

// sseStringDefault returns the Go literal of the default value of the optional
// String attribute name of eventAttr, or "" when the attribute is required or
// has no default value.
func sseStringDefault(eventAttr *expr.AttributeExpr, name string) string {
	if eventAttr.IsRequired(name) {
		return ""
	}
	def, ok := eventAttr.GetDefault(name).(string)
	if !ok {
		return ""
	}
	return strconv.Quote(def)
}

func sseSendDescription(sendName, eventTypeName, methodName string) string {
	return fmt.Sprintf("%s streams instances of %q to the %q endpoint SSE connection.", sendName, eventTypeName, methodName)
}

func sseSendWithContextDescription(sendName, eventTypeName, methodName string) string {
	return fmt.Sprintf("%s streams instances of %q to the %q endpoint SSE connection with context.", sendName, eventTypeName, methodName)
}

func sseRecvDescription(recvName, methodName string) string {
	return fmt.Sprintf("%s connects to the %q SSE endpoint and streams events.", recvName, methodName)
}

// sseServerFile returns the file implementing the SSE server
// streaming implementation if any.
func sseServerFile(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	if data == nil {
		return nil
	}

	// Check if any endpoint has SSE
	hasSSE := false
	for _, ed := range data.Endpoints {
		if ed.SSE != nil {
			hasSSE = true
			break
		}
	}
	if !hasSSE {
		return nil
	}

	path := filepath.Join(codegen.Gendir, "http", data.Service.PathName, "server", "sse.go")
	data, imports := services.FileData(svc.Name(), append([]*codegen.ImportSpec{
		{Path: "context"},
		{Path: "io"},
		{Path: "net/http"},
		{Path: "sync"},
		{Path: "time"},
		{Path: "encoding/json/v2", Name: "json"},
		{Path: "fmt"},
		{Path: "github.com/CaliLuke/loom/http", Name: "loomhttp"},
		codegen.LoomNamedImport("observability/transport", "loomtransport"),
		codegen.LoomImport(""),
		{Path: genpkg + "/" + data.Service.PathName, Name: data.Service.PkgName},
		{Path: genpkg + "/" + data.Service.PathName + "/views", Name: data.Service.ViewsPkg},
	}, data.Service.UserTypeImports...))
	sseSections := serverSSESections(data)
	sections := make([]codegen.Section, 0, 1+len(sseSections))
	sections = append(sections, codegen.Header("sse", "server", imports))
	for _, section := range sseSections {
		sections = append(sections, section)
	}
	return &codegen.File{Path: path, Sections: sections}
}

// IsSSEEndpoint returns true if the endpoint defines a streaming result
// with SSE.
func IsSSEEndpoint(ed *EndpointData) bool {
	return ed.SSE != nil
}
