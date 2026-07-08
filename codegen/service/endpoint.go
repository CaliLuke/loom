package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// EndpointsData contains the data necessary to render the
	// service endpoints struct template.
	EndpointsData struct {
		// Name is the service name.
		Name string
		// Description is the service description.
		Description string
		// VarName is the endpoint struct name.
		VarName string
		// ClientVarName is the client struct name.
		ClientVarName string
		// ServiceVarName is the service interface name.
		ServiceVarName string
		// Methods lists the endpoint struct methods.
		Methods []*EndpointMethodData
		// ClientInitArgs lists the arguments needed to instantiate the client.
		ClientInitArgs string
		// Schemes contains the security schemes types used by the
		// all the endpoints.
		Schemes SchemesData
		// HasServerInterceptors indicates that the service has server-side
		// interceptors.
		HasServerInterceptors bool
		// HasClientInterceptors indicates that the service has client-side
		// interceptors.
		HasClientInterceptors bool
	}

	// EndpointMethodData describes a single endpoint method.
	EndpointMethodData struct {
		*MethodData
		// ArgName is the name of the argument used to initialize the client
		// struct method field.
		ArgName string
		// StreamArgName is the name of the argument used to initialize the client
		// struct stream endpoint field when the method defines mixed results.
		//
		// It is only set when HasMixedResults is true.
		StreamArgName string
		// ClientVarName is the corresponding client struct field name.
		ClientVarName string
		// ServiceName is the name of the owner service.
		ServiceName string
		// ServiceVarName is the name of the owner service Go interface.
		ServiceVarName string
	}
)

const (
	// endpointsStructName is the name of the generated endpoints data
	// structure.
	endpointsStructName = "Endpoints"

	// serviceInterfaceName is the name of the generated service interface.
	serviceInterfaceName = "Service"
)

// EndpointFile returns the endpoint file for the given service.
func EndpointFile(genpkg string, service *expr.ServiceExpr, services *ServicesData) *codegen.File {
	svc := services.Get(service.Name)
	svcName := svc.PathName
	path := filepath.Join(codegen.Gendir, svcName, "endpoints.go")
	data := endpointData(svc)
	var (
		sections []codegen.Section
	)
	{
		imports := []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "io"},
			{Path: "fmt"},
			codegen.LoomImport(""),
			codegen.LoomImport("security"),
			{Path: genpkg + "/" + svcName + "/" + "views", Name: svc.ViewsPkg},
		}
		header := codegen.Header(service.Name+" endpoints", svc.PkgName, imports)
		def := endpointsStructSection(data)
		sections = []codegen.Section{header, def}
		for _, m := range data.Methods {
			if m.ServerStream != nil {
				// Generate endpoint input struct for streaming methods
				// For JSON-RPC WebSocket with StreamingResult: generate struct (needed for stream handle)
				// For JSON-RPC WebSocket without StreamingResult (client streaming only): no struct needed
				// For JSON-RPC SSE: always generate struct (methods have stream params)
				// For HTTP/gRPC: always generate endpoint input struct
				isJSONRPCWebSocket := m.IsJSONRPC && !isJSONRPCSSE(services, service)
				if !isJSONRPCWebSocket || (isJSONRPCWebSocket && m.ServerStream.EndpointStruct != "") {
					sections = append(sections, endpointStreamStructSection(m))
				}
			}
			if m.SkipRequestBodyEncodeDecode {
				sections = append(sections, requestBodyStructSection(m))
			}
			if m.SkipResponseBodyEncodeDecode {
				sections = append(sections, responseBodyStructSection(m))
			}
		}
		sections = append(sections, endpointsInitSection(data), endpointsUseSection(data))
		for _, m := range data.Methods {
			sections = append(sections, endpointMethodSection(m))
		}
	}

	return &codegen.File{Path: path, Sections: sections}
}

func endpointData(svc *Data) *EndpointsData {
	methods := make([]*EndpointMethodData, len(svc.Methods))
	argScope := codegen.NewNameScope()
	names := make([]string, 0, len(svc.Methods)*2)
	for i, m := range svc.Methods {
		argName := argScope.Unique(codegen.Goify(m.VarName, false), "")
		names = append(names, argName)
		streamArgName := ""
		if m.HasMixedResults {
			streamArgName = argScope.Unique(argName+"Stream", "")
			names = append(names, streamArgName)
		}
		methods[i] = &EndpointMethodData{
			MethodData:     m,
			ArgName:        argName,
			StreamArgName:  streamArgName,
			ServiceName:    svc.Name,
			ServiceVarName: serviceInterfaceName,
			ClientVarName:  clientStructName,
		}
	}
	desc := fmt.Sprintf("%s wraps the %q service endpoints.", endpointsStructName, svc.Name)
	return &EndpointsData{
		Name:                  svc.Name,
		Description:           desc,
		VarName:               endpointsStructName,
		ClientVarName:         clientStructName,
		ServiceVarName:        serviceInterfaceName,
		ClientInitArgs:        strings.Join(names, ", "),
		Methods:               methods,
		Schemes:               svc.Schemes,
		HasServerInterceptors: len(svc.ServerInterceptors) > 0,
		HasClientInterceptors: len(svc.ClientInterceptors) > 0,
	}
}

func payloadVar(e *EndpointMethodData) string {
	if e.ServerStream != nil {
		if e.ServerStream.EndpointStruct != "" {
			return "ep.Payload"
		}
		if e.PayloadRef != "" {
			return "p"
		}
		// JSON-RPC WebSocket has no payload for server streaming.
		return ""
	}
	if e.SkipRequestBodyEncodeDecode {
		return "ep.Payload"
	}
	return "p"
}
