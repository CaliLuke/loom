package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/v3/codegen"
)

func endpointsStructSection(data *EndpointsData) codegen.Section {
	return codegen.NewRawSection("endpoints-struct", renderEndpointsStruct(data))
}

func endpointStreamStructSection(method *EndpointMethodData) codegen.Section {
	return codegen.NewRawSection("endpoint-input-struct", renderEndpointStreamStruct(method))
}

func requestBodyStructSection(method *EndpointMethodData) codegen.Section {
	return codegen.NewRawSection("request-body-struct", renderRequestBodyStruct(method))
}

func responseBodyStructSection(method *EndpointMethodData) codegen.Section {
	return codegen.NewRawSection("response-body-struct", renderResponseBodyStruct(method))
}

func endpointsInitSection(data *EndpointsData) codegen.Section {
	return codegen.NewRawSection("endpoints-init", renderEndpointsInit(data))
}

func endpointsUseSection(data *EndpointsData) codegen.Section {
	return codegen.NewRawSection("endpoints-use", renderEndpointsUse(data))
}

func renderEndpointsStruct(data *EndpointsData) string {
	var b strings.Builder
	b.WriteString(codegen.Comment(data.Description))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s struct {\n", data.VarName)
	for _, method := range data.Methods {
		fmt.Fprintf(&b, "\t%s goa.Endpoint\n", method.VarName)
	}
	b.WriteString("}\n")
	return b.String()
}

func renderEndpointsInit(data *EndpointsData) string {
	var b strings.Builder
	b.WriteString(codegen.Comment(fmt.Sprintf("New%s wraps the methods of the %q service with endpoints.", data.VarName, data.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func New%s(s %s", data.VarName, data.ServiceVarName)
	if data.HasServerInterceptors {
		b.WriteString(", si ServerInterceptors")
	}
	fmt.Fprintf(&b, ") *%s {\n", data.VarName)
	if len(data.Schemes) > 0 {
		b.WriteString("\t// Casting service to Auther interface\n")
		b.WriteString("\ta := s.(Auther)\n")
	}
	if data.HasServerInterceptors {
		fmt.Fprintf(&b, "\tendpoints := &%s{\n", data.VarName)
	} else {
		fmt.Fprintf(&b, "\treturn &%s{\n", data.VarName)
	}
	for _, method := range data.Methods {
		fmt.Fprintf(&b, "\t\t%s: New%sEndpoint(s", method.VarName, method.VarName)
		for _, scheme := range method.Schemes.DedupeByType() {
			fmt.Fprintf(&b, ", a.%sAuth", scheme.Type)
		}
		b.WriteString("),\n")
	}
	b.WriteString("\t}\n")
	if data.HasServerInterceptors {
		for _, method := range data.Methods {
			if len(method.ServerInterceptors) == 0 {
				continue
			}
			fmt.Fprintf(&b, "\tendpoints.%s = Wrap%sEndpoint(endpoints.%s, si)\n", method.VarName, method.VarName, method.VarName)
		}
		b.WriteString("\treturn endpoints\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func renderEndpointsUse(data *EndpointsData) string {
	var b strings.Builder
	b.WriteString(codegen.Comment(fmt.Sprintf("Use applies the given middleware to all the %q service endpoints.", data.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (e *%s) Use(m func(goa.Endpoint) goa.Endpoint) {\n", data.VarName)
	for _, method := range data.Methods {
		fmt.Fprintf(&b, "\te.%s = m(e.%s)\n", method.VarName, method.VarName)
	}
	b.WriteString("}\n")
	return b.String()
}

func renderEndpointStreamStruct(method *EndpointMethodData) string {
	var b strings.Builder
	b.WriteString(codegen.Comment(fmt.Sprintf("%s holds both the payload and the server stream of the %q method.", method.ServerStream.EndpointStruct, method.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s struct {\n", method.ServerStream.EndpointStruct)
	if method.PayloadRef != "" {
		b.WriteString(codegen.Indent(codegen.Comment("Payload is the method payload."), "\t"))
		b.WriteString("\n")
		fmt.Fprintf(&b, "\tPayload %s\n", method.PayloadRef)
	}
	if method.IsJSONRPC {
		b.WriteString(codegen.Indent(codegen.Comment("RequestID is the JSON-RPC request ID (available for JSON-RPC transports)."), "\t"))
		b.WriteString("\n")
		b.WriteString("\tRequestID any\n")
	}
	b.WriteString(codegen.Indent(codegen.Comment(fmt.Sprintf("Stream is the server stream used by the %q method to send data.", method.Name)), "\t"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "\tStream %s\n", method.ServerStream.Interface)
	b.WriteString("}\n")
	return b.String()
}

func renderRequestBodyStruct(method *EndpointMethodData) string {
	var b strings.Builder
	b.WriteString(codegen.Comment(fmt.Sprintf("%s holds both the payload and the HTTP request body reader of the %q method.", method.RequestStruct, method.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s struct {\n", method.RequestStruct)
	if method.PayloadRef != "" {
		b.WriteString(codegen.Indent(codegen.Comment("Payload is the method payload."), "\t"))
		b.WriteString("\n")
		fmt.Fprintf(&b, "\tPayload %s\n", method.PayloadRef)
	}
	b.WriteString(codegen.Indent(codegen.Comment("Body streams the HTTP request body."), "\t"))
	b.WriteString("\n")
	b.WriteString("\tBody io.ReadCloser\n")
	b.WriteString("}\n")
	return b.String()
}

func renderResponseBodyStruct(method *EndpointMethodData) string {
	var b strings.Builder
	b.WriteString(codegen.Comment(fmt.Sprintf("%s holds both the result and the HTTP response body reader of the %q method.", method.ResponseStruct, method.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s struct {\n", method.ResponseStruct)
	if method.ResultRef != "" {
		b.WriteString(codegen.Indent(codegen.Comment("Result is the method result."), "\t"))
		b.WriteString("\n")
		fmt.Fprintf(&b, "\tResult %s\n", method.ResultRef)
	}
	b.WriteString(codegen.Indent(codegen.Comment("Body streams the HTTP response body."), "\t"))
	b.WriteString("\n")
	b.WriteString("\tBody io.ReadCloser\n")
	b.WriteString("}\n")
	return b.String()
}
