package codegen

import (
	"fmt"
	"path/filepath"

	"slices"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

type (
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
		// EventField is the name of the result type event field if any.
		// If empty, no event field is included in the event.
		EventField string
		// RetryField is the name of the result type event retry field if any.
		// If empty, no retry field is included in the event.
		RetryField string
		// RequestIDField is the name of the payload field that maps to the Last-Event-ID header if any.
		// If empty, no last event id is included in the request.
		RequestIDField string
		// RequestIDPointer indicates whether the RequestIDField is a pointer (i.e., optional primitive).
		RequestIDPointer bool
		// HasResponseBody indicates whether an HTTP response body converter exists for this endpoint.
		HasResponseBody bool
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
	dataFieldVar, dataFieldTypeRef, idFieldVar, eventFieldVar, retryFieldVar := sseEventFieldData(endpointIR, sd, svc, eventAttr)

	ed.SSE = &SSEData{
		StructName:          md.ServerStream.VarName,
		Interface:           fmt.Sprintf("%s.%s", svc.PkgName, md.ServerStream.Interface),
		SendName:            md.ServerStream.SendName,
		SendDesc:            sseSendDescription(md.ServerStream.SendName, eventType.Name, md.Name),
		SendWithContextName: md.ServerStream.SendWithContextName,
		SendWithContextDesc: sseSendWithContextDescription(md.ServerStream.SendWithContextName, eventType.Name, md.Name),
		RecvName:            md.ClientStream.RecvName,
		RecvDesc:            sseRecvDescription(md.ClientStream.RecvName, md.Name),
		EventTypeRef:        eventType.Ref,
		EventTypeName:       eventType.Name,
		EventIsStruct:       eventType.IsStruct,
		DataFieldTypeRef:    dataFieldTypeRef,
		DataField:           dataFieldVar,
		IDField:             idFieldVar,
		EventField:          eventFieldVar,
		RetryField:          retryFieldVar,
		RequestIDField:      endpointIR.Stream.SSE.RequestIDField,
		RequestIDPointer:    endpointIR.Stream.SSE.RequestIDPointer,
	}

	// Mixed results SSE uses the streaming result type for events, not the unary
	// HTTP response body type. Disable HTTP response body conversion in the SSE
	// stream implementation and marshal the event value directly.
	if caps.HasMixedResults {
		ed.SSE.HasResponseBody = false
		return
	}

	if ed.Result != nil {
		for _, resp := range ed.Result.Responses {
			if len(resp.ServerBody) > 0 {
				ed.SSE.HasResponseBody = true
				break
			}
		}
	}
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

func sseEventFieldData(endpointIR *transportir.Endpoint, sd *ServiceData, svc *service.Data, eventAttr *expr.AttributeExpr) (string, string, string, string, string) {
	var dataFieldVar, dataFieldTypeRef, idFieldVar, eventFieldVar, retryFieldVar string
	if obj := expr.AsObject(eventAttr.Type); obj != nil {
		for _, nat := range *obj {
			switch nat.Name {
			case endpointIR.Stream.SSE.IDField:
				idFieldVar = codegen.GoifyAtt(nat.Attribute, nat.Name, true)
			case endpointIR.Stream.SSE.EventField:
				eventFieldVar = codegen.GoifyAtt(nat.Attribute, nat.Name, true)
			case endpointIR.Stream.SSE.RetryField:
				retryFieldVar = codegen.GoifyAtt(nat.Attribute, nat.Name, true)
			case endpointIR.Stream.SSE.DataField:
				dataFieldVar = codegen.GoifyAtt(nat.Attribute, nat.Name, true)
				dataFieldTypeRef = sd.Service.Scope.GoFullTypeRef(nat.Attribute, svc.PkgName)
			}
		}
	}
	return dataFieldVar, dataFieldTypeRef, idFieldVar, eventFieldVar, retryFieldVar
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

	path := filepath.Join(codegen.Gendir, "http", codegen.SnakeCase(svc.Name()), "server", "sse.go")
	sseSections := serverSSESections(data)
	sections := make([]codegen.Section, 0, 1+len(sseSections))
	sections = append(sections,
		codegen.Header(
			"sse",
			"server",
			[]*codegen.ImportSpec{
				{Path: "context"},
				{Path: "io"},
				{Path: "net/http"},
				{Path: "sync"},
				{Path: "time"},
				{Path: "encoding/json"},
				{Path: "fmt"},
				{Path: "github.com/CaliLuke/loom/http", Name: "loomhttp"},
				{Path: genpkg + "/" + codegen.SnakeCase(svc.Name()), Name: data.Service.PkgName},
				{Path: genpkg + "/" + codegen.SnakeCase(svc.Name()) + "/views", Name: data.Service.ViewsPkg},
			},
		),
	)
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

// HasSSE returns true if at least one endpoint in the service uses SSE.
func HasSSE(data *ServiceData) bool {
	return slices.ContainsFunc(data.Endpoints, IsSSEEndpoint)
}
