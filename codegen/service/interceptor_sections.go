//nolint:errcheck // Generator helpers write only to in-memory builders.
package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func serverInterceptorsInterfaceSection(svc *Data) codegen.Section {
	return codegen.MustRenderSection("server-interceptors-type", func() string {
		return renderInterceptorsInterface(svc.ServerInterceptors, true)
	})
}

func clientInterceptorsInterfaceSection(svc *Data) codegen.Section {
	return codegen.MustRenderSection("client-interceptors-type", func() string {
		return renderInterceptorsInterface(svc.ClientInterceptors, false)
	})
}

func interceptorTypesSection(interceptors []*InterceptorData) codegen.Section {
	return codegen.MustRenderSection("interceptor-types", func() string {
		return renderInterceptorTypes(interceptors)
	})
}

func endpointWrapperSection(server bool, methodVarName, method string, interceptors []string) codegen.Section {
	name := "client-wrapper"
	if server {
		name = "endpoint-wrapper"
	}
	return codegen.MustRenderSection(name, func() string {
		return renderEndpointWrapper(server, methodVarName, method, interceptors)
	})
}

func interceptorsSection(interceptors []*InterceptorData, server bool) codegen.Section {
	return codegen.MustRenderSection("interceptors", func() string {
		return renderInterceptors(interceptors, server)
	})
}

func streamWrapperTypesSection(name string, streams []*StreamInterceptorData, server bool) codegen.Section {
	return codegen.MustRenderSection(name, func() string {
		return renderStreamWrapperTypes(streams, server)
	})
}

func serverInterceptorWrappersSection(service string, interceptors []*InterceptorData) codegen.Section {
	return codegen.MustRenderSection("server-interceptor-wrappers", func() string {
		return renderServerInterceptorWrappers(service, interceptors)
	})
}

func clientInterceptorWrappersSection(service string, interceptors []*InterceptorData) codegen.Section {
	return codegen.MustRenderSection("client-interceptor-wrappers", func() string {
		return renderClientInterceptorWrappers(service, interceptors)
	})
}

func streamWrappersSection(name string, streams []*StreamInterceptorData, server bool) codegen.Section {
	return codegen.MustRenderSection(name, func() string {
		return renderStreamWrappers(streams, server)
	})
}

func renderInterceptorsInterface(interceptors []*InterceptorData, server bool) string {
	var b sourceBuilder
	if server {
		b.Add("// ServerInterceptors defines the interface for all server-side interceptors.\n")
		b.Add("// Server interceptors execute after the request is decoded and before the\n")
		b.Add("// payload is sent to the service. The implementation is responsible for calling\n")
		b.Add("// next to complete the request.\n")
		b.Add("type ServerInterceptors interface {\n")
		for _, interceptor := range interceptors {
			if interceptor.Description != "" {
				b.Add(codegen.Indent(codegen.Comment(interceptor.Description), "\t"))
				b.Add("\n")
			}
			fmt.Fprintf(&b, "\t%s(ctx context.Context, info *%sInfo, next loom.Endpoint) (any, error)\n", interceptor.Name, interceptor.Name)
		}
		b.Add("}\n")
		return b.String()
	}
	b.Add("// ClientInterceptors defines the interface for all client-side interceptors.\n")
	b.Add("// Client interceptors execute after the payload is encoded and before the request\n")
	b.Add("// is sent to the server. The implementation is responsible for calling next to\n")
	b.Add("// complete the request.\n")
	b.Add("type ClientInterceptors interface {\n")
	for _, interceptor := range interceptors {
		if interceptor.Description != "" {
			b.Add(codegen.Indent(codegen.Comment(interceptor.Description), "\t"))
			b.Add("\n")
		}
		fmt.Fprintf(&b, "\t%s(ctx context.Context, info *%sInfo, next loom.Endpoint) (any, error)\n", interceptor.Name, interceptor.Name)
	}
	b.Add("}\n")
	return b.String()
}

func renderInterceptorTypes(interceptors []*InterceptorData) string {
	var b sourceBuilder
	writeInterceptorAccessTypes(&b, interceptors)

	if !hasPrivateImplementationTypes(interceptors) {
		return b.String()
	}
	writeInterceptorPrivateTypes(&b, interceptors)
	return b.String()
}

func writeInterceptorAccessTypes(b *sourceBuilder, interceptors []*InterceptorData) {
	b.Add("\n// Access interfaces for interceptor payloads and results\n")
	b.Add("type (\n")
	for _, interceptor := range interceptors {
		writeInterceptorInfoType(b, interceptor)
		writeInterceptorAccessorInterface(b, interceptor.Name, "Payload", interceptor.HasPayloadAccess, interceptor.ReadPayload, interceptor.WritePayload)
		writeInterceptorAccessorInterface(b, interceptor.Name, "Result", interceptor.HasResultAccess, interceptor.ReadResult, interceptor.WriteResult)
		writeInterceptorAccessorInterface(b, interceptor.Name, "StreamingPayload", interceptor.HasStreamingPayloadAccess, interceptor.ReadStreamingPayload, interceptor.WriteStreamingPayload)
		writeInterceptorAccessorInterface(b, interceptor.Name, "StreamingResult", interceptor.HasStreamingResultAccess, interceptor.ReadStreamingResult, interceptor.WriteStreamingResult)
	}
	b.Add(")\n")
}

func writeInterceptorInfoType(b *sourceBuilder, interceptor *InterceptorData) {
	b.Add(codegen.Indent(codegen.Comment(fmt.Sprintf("%sInfo provides metadata about the current interception.\nIt includes service name, method name, and access to the endpoint.", interceptor.Name)), "\t"))
	b.Add("\n")
	fmt.Fprintf(b, "\t%sInfo struct {\n\t\tservice    string\n\t\tmethod     string\n\t\tcallType   loom.InterceptorCallType\n\t\trawPayload any\n\t}\n", interceptor.Name)
}

func writeInterceptorAccessorInterface(b *sourceBuilder, name, suffix string, enabled bool, readFields, writeFields []*AttributeData) {
	if !enabled {
		return
	}
	b.Add("\n")
	desc := fmt.Sprintf("%s%s provides type-safe access to the method %s.\nIt allows reading and writing specific fields of the %s as defined\nin the design.", name, suffix, interfaceAccessTarget(suffix), interfaceAccessTarget(suffix))
	if strings.HasPrefix(suffix, "Streaming") {
		b.Add("\t// " + name + suffix + " provides type-safe access to the method " + interfaceAccessTarget(suffix) + ".\n")
		b.Add("\t// It allows reading and writing specific fields of the " + interfaceAccessTarget(suffix) + " as defined\n")
		b.Add("\t// in the design.\n")
	} else {
		b.Add(codegen.Indent(codegen.Comment(desc), "\t"))
		b.Add("\n")
	}
	fmt.Fprintf(b, "\t%s%s interface {\n", name, suffix)
	for _, field := range readFields {
		fmt.Fprintf(b, "\t\t%s() %s\n", field.Name, field.TypeRef)
	}
	for _, field := range writeFields {
		fmt.Fprintf(b, "\t\tSet%s(%s)\n", field.Name, field.TypeRef)
	}
	b.Add("\t}\n")
}

func interfaceAccessTarget(suffix string) string {
	switch suffix {
	case "Payload":
		return "payload"
	case "Result":
		return "result"
	case "StreamingPayload":
		return "streaming payload"
	case "StreamingResult":
		return "streaming result"
	default:
		return "value"
	}
}

func writeInterceptorPrivateTypes(b *sourceBuilder, interceptors []*InterceptorData) {
	b.Add("\n// Private implementation types\n")
	b.Add("type (\n")
	writeInterceptorMethodStructs(b, interceptors, func(m *MethodInterceptorData) (string, string) { return m.PayloadAccess, m.PayloadRef }, "payload")
	writeInterceptorMethodStructs(b, interceptors, func(m *MethodInterceptorData) (string, string) { return m.ResultAccess, m.ResultRef }, "result")
	writeInterceptorMethodStructs(b, interceptors, func(m *MethodInterceptorData) (string, string) {
		return m.StreamingPayloadAccess, m.StreamingPayloadRef
	}, "payload")
	writeInterceptorMethodStructs(b, interceptors, func(m *MethodInterceptorData) (string, string) { return m.StreamingResultAccess, m.StreamingResultRef }, "result")
	b.Add(")\n")
}

func writeInterceptorMethodStructs(b *sourceBuilder, interceptors []*InterceptorData, pick func(*MethodInterceptorData) (string, string), fieldName string) {
	for _, interceptor := range interceptors {
		for _, method := range interceptor.Methods {
			typeName, typeRef := pick(method)
			if typeName == "" {
				continue
			}
			fmt.Fprintf(b, "\t%s struct {\n\t\t%s %s\n\t}\n", typeName, fieldName, typeRef)
		}
	}
}

func renderEndpointWrapper(server bool, methodVarName, method string, interceptors []string) string {
	var b sourceBuilder
	wrapName := "Wrap" + methodVarName + "Endpoint"
	commentTarget := "server-side"
	callPrefix := "wrap" + methodVarName
	interfaceName := "ServerInterceptors"
	if !server {
		wrapName = "Wrap" + methodVarName + "ClientEndpoint"
		commentTarget = "client"
		callPrefix = "wrapClient" + methodVarName
		interfaceName = "ClientInterceptors"
	}
	b.Add(codegen.Comment(fmt.Sprintf("%s wraps the %s endpoint with the %s interceptors defined in the design.", wrapName, method, commentTarget)))
	b.Add("\n")
	fmt.Fprintf(&b, "func %s(endpoint loom.Endpoint, i %s) loom.Endpoint {\n", wrapName, interfaceName)
	b.Add("\tif i != nil {\n")
	for _, interceptor := range interceptors {
		fmt.Fprintf(&b, "\t\tendpoint = %s%s(endpoint, i)\n", callPrefix, interceptor)
	}
	b.Add("\t}\n\treturn endpoint\n}\n")
	return b.String()
}

func renderInterceptors(interceptors []*InterceptorData, server bool) string {
	var b sourceBuilder
	b.Add("// Public accessor methods for Info types\n")
	for _, interceptor := range interceptors {
		b.Add("\n")
		fmt.Fprintf(&b, "// Service returns the name of the service handling the request.\nfunc (info *%sInfo) Service() string {\n\treturn info.service\n}\n\n", interceptor.Name)
		fmt.Fprintf(&b, "// Method returns the name of the method handling the request.\nfunc (info *%sInfo) Method() string {\n\treturn info.method\n}\n\n", interceptor.Name)
		fmt.Fprintf(&b, "// CallType returns the type of call the interceptor is handling.\nfunc (info *%sInfo) CallType() loom.InterceptorCallType {\n\treturn info.callType\n}\n\n", interceptor.Name)
		fmt.Fprintf(&b, "// RawPayload returns the raw payload of the request.\nfunc (info *%sInfo) RawPayload() any {\n\treturn info.rawPayload\n}\n", interceptor.Name)
		if interceptor.HasPayloadAccess {
			b.Add("\n")
			fmt.Fprintf(&b, "// Payload returns a type-safe accessor for the method payload.\nfunc (info *%sInfo) Payload() %sPayload {\n", interceptor.Name, interceptor.Name)
			b.Add(renderPayloadAccessSwitch(interceptor, server))
			b.Add("}\n")
		}
		if interceptor.HasResultAccess {
			b.Add("\n")
			fmt.Fprintf(&b, "// Result returns a type-safe accessor for the method result.\nfunc (info *%sInfo) Result(res any) %sResult {\n", interceptor.Name, interceptor.Name)
			b.Add(renderResultAccessSwitch(interceptor))
			b.Add("}\n")
		}
		if interceptor.HasStreamingPayloadAccess {
			b.Add("\n")
			fmt.Fprintf(&b, "// ClientStreamingPayload returns a type-safe accessor for the method streaming payload for a client-side interceptor.\nfunc (info *%sInfo) ClientStreamingPayload() %sStreamingPayload {\n", interceptor.Name, interceptor.Name)
			b.Add(renderStreamingPayloadAccess(interceptor, true))
			b.Add("}\n")
		}
		if interceptor.HasStreamingResultAccess {
			b.Add("\n")
			fmt.Fprintf(&b, "// ClientStreamingResult returns a type-safe accessor for the method streaming result for a client-side interceptor.\nfunc (info *%sInfo) ClientStreamingResult(res any) %sStreamingResult {\n", interceptor.Name, interceptor.Name)
			b.Add(renderStreamingResultAccess(interceptor, true))
			b.Add("}\n")
		}
		if interceptor.HasStreamingPayloadAccess {
			b.Add("\n")
			fmt.Fprintf(&b, "// ServerStreamingPayload returns a type-safe accessor for the method streaming payload for a server-side interceptor.\nfunc (info *%sInfo) ServerStreamingPayload(pay any) %sStreamingPayload {\n", interceptor.Name, interceptor.Name)
			b.Add(renderStreamingPayloadAccess(interceptor, false))
			b.Add("}\n")
		}
		if interceptor.HasStreamingResultAccess {
			b.Add("\n")
			fmt.Fprintf(&b, "// ServerStreamingResult returns a type-safe accessor for the method streaming result for a server-side interceptor.\nfunc (info *%sInfo) ServerStreamingResult() %sStreamingResult {\n", interceptor.Name, interceptor.Name)
			b.Add(renderStreamingResultAccess(interceptor, false))
			b.Add("}\n")
		}
	}
	if hasPrivateImplementationTypes(interceptors) {
		b.Add("\n// Private implementation methods\n")
		first := true
		for _, interceptor := range interceptors {
			for _, method := range interceptor.Methods {
				first = renderAccessorMethods(&b, method.PayloadAccess, "payload", interceptor.ReadPayload, interceptor.WritePayload, first)
				first = renderAccessorMethods(&b, method.ResultAccess, "result", interceptor.ReadResult, interceptor.WriteResult, first)
				first = renderAccessorMethods(&b, method.StreamingPayloadAccess, "payload", interceptor.ReadStreamingPayload, interceptor.WriteStreamingPayload, first)
				first = renderAccessorMethods(&b, method.StreamingResultAccess, "result", interceptor.ReadStreamingResult, interceptor.WriteStreamingResult, first)
			}
		}
	}
	return b.String()
}

func renderPayloadAccessSwitch(interceptor *InterceptorData, server bool) string {
	if len(interceptor.Methods) == 1 {
		method := interceptor.Methods[0]
		if server && hasEndpointStruct(true)(method) {
			return fmt.Sprintf("\tswitch pay := info.RawPayload().(type) {\n\tcase *%s:\n\t\treturn &%s{payload: pay.Payload}\n\tdefault:\n\t\treturn &%s{payload: pay.(%s)}\n\t}\n", method.ServerStream.EndpointStruct, method.PayloadAccess, method.PayloadAccess, method.PayloadRef)
		}
		return fmt.Sprintf("\treturn &%s{payload: info.RawPayload().(%s)}\n", method.PayloadAccess, method.PayloadRef)
	}
	var b sourceBuilder
	b.Add("\tswitch info.Method() {\n")
	for _, method := range interceptor.Methods {
		fmt.Fprintf(&b, "\tcase %q:\n", method.MethodName)
		if server && hasEndpointStruct(true)(method) {
			fmt.Fprintf(&b, "\t\tswitch pay := info.RawPayload().(type) {\n\t\tcase *%s:\n\t\t\treturn &%s{payload: pay.Payload}\n\t\tdefault:\n\t\t\treturn &%s{payload: pay.(%s)}\n\t\t}\n", method.ServerStream.EndpointStruct, method.PayloadAccess, method.PayloadAccess, method.PayloadRef)
		} else {
			fmt.Fprintf(&b, "\t\treturn &%s{payload: info.RawPayload().(%s)}\n", method.PayloadAccess, method.PayloadRef)
		}
	}
	b.Add("\tdefault:\n\t\treturn nil\n\t}\n")
	return b.String()
}

func renderResultAccessSwitch(interceptor *InterceptorData) string {
	if len(interceptor.Methods) == 1 {
		method := interceptor.Methods[0]
		return fmt.Sprintf("\treturn &%s{result: res.(%s)}\n", method.ResultAccess, method.ResultRef)
	}
	var b sourceBuilder
	b.Add("\tswitch info.Method() {\n")
	for _, method := range interceptor.Methods {
		fmt.Fprintf(&b, "\tcase %q:\n\t\treturn &%s{result: res.(%s)}\n", method.MethodName, method.ResultAccess, method.ResultRef)
	}
	b.Add("\tdefault:\n\t\treturn nil\n\t}\n")
	return b.String()
}

func renderStreamingPayloadAccess(interceptor *InterceptorData, client bool) string {
	if len(interceptor.Methods) == 1 {
		method := interceptor.Methods[0]
		arg := "info.RawPayload()"
		if !client {
			arg = "pay"
		}
		return fmt.Sprintf("\treturn &%s{payload: %s.(%s)}\n", method.StreamingPayloadAccess, arg, method.StreamingPayloadRef)
	}
	var b sourceBuilder
	b.Add("\tswitch info.Method() {\n")
	for _, method := range interceptor.Methods {
		arg := "info.RawPayload()"
		if !client {
			arg = "pay"
		}
		fmt.Fprintf(&b, "\tcase %q:\n\t\treturn &%s{payload: %s.(%s)}\n", method.MethodName, method.StreamingPayloadAccess, arg, method.StreamingPayloadRef)
	}
	b.Add("\tdefault:\n\t\treturn nil\n\t}\n")
	return b.String()
}

func renderStreamingResultAccess(interceptor *InterceptorData, client bool) string {
	if len(interceptor.Methods) == 1 {
		method := interceptor.Methods[0]
		if client {
			return fmt.Sprintf("\treturn &%s{result: res.(%s)}\n", method.StreamingResultAccess, method.StreamingResultRef)
		}
		return fmt.Sprintf("\treturn &%s{result: info.RawPayload().(%s)}\n", method.StreamingResultAccess, method.StreamingResultRef)
	}
	var b sourceBuilder
	b.Add("\tswitch info.Method() {\n")
	for _, method := range interceptor.Methods {
		if client {
			fmt.Fprintf(&b, "\tcase %q:\n\t\treturn &%s{result: res.(%s)}\n", method.MethodName, method.StreamingResultAccess, method.StreamingResultRef)
		} else {
			fmt.Fprintf(&b, "\tcase %q:\n\t\treturn &%s{result: info.RawPayload().(%s)}\n", method.MethodName, method.StreamingResultAccess, method.StreamingResultRef)
		}
	}
	b.Add("\tdefault:\n\t\treturn nil\n\t}\n")
	return b.String()
}

func renderAccessorMethods(b *sourceBuilder, accessName, fieldName string, readers, writers []*AttributeData, first bool) bool {
	if accessName == "" {
		return first
	}
	receiver := "p"
	if fieldName == "result" {
		receiver = "r"
	}
	if len(readers) > 0 || len(writers) > 0 {
		if first {
			b.Add("\n")
			first = false
		}
	}
	for _, field := range readers {
		fmt.Fprintf(b, "func (%s *%s) %s() %s {\n", receiver, accessName, field.Name, field.TypeRef)
		if field.Pointer {
			fmt.Fprintf(b, "\tif %s.%s.%s == nil {\n\t\tvar zero %s\n\t\treturn zero\n\t}\n\treturn *%s.%s.%s\n", receiver, fieldName, field.Name, field.TypeRef, receiver, fieldName, field.Name)
		} else {
			fmt.Fprintf(b, "\treturn %s.%s.%s\n", receiver, fieldName, field.Name)
		}
		b.Add("}\n")
	}
	for _, field := range writers {
		fmt.Fprintf(b, "func (%s *%s) Set%s(v %s) {\n", receiver, accessName, field.Name, field.TypeRef)
		if field.Pointer {
			fmt.Fprintf(b, "\t%s.%s.%s = &v\n", receiver, fieldName, field.Name)
		} else {
			fmt.Fprintf(b, "\t%s.%s.%s = v\n", receiver, fieldName, field.Name)
		}
		b.Add("}\n")
	}
	return first
}

func renderStreamWrapperTypes(streams []*StreamInterceptorData, server bool) string {
	var b sourceBuilder
	for i, stream := range streams {
		b.Add("\n")
		if i == 0 {
			b.Add("\n")
		}
		target := "client"
		if server {
			target = "server"
		}
		b.Add(codegen.Comment(fmt.Sprintf("wrapped%s is a %s interceptor wrapper for the %s stream.", stream.Interface, target, stream.Interface)))
		b.Add("\n")
		fmt.Fprintf(&b, "type wrapped%s struct {\n\tctx context.Context\n", stream.Interface)
		if stream.SendTypeRef != "" {
			fmt.Fprintf(&b, "\tsendWithContext func(context.Context, %s) error\n", stream.SendTypeRef)
		}
		if stream.RecvTypeRef != "" {
			fmt.Fprintf(&b, "\trecvWithContext func(context.Context) (%s, error)\n", stream.RecvTypeRef)
		}
		fmt.Fprintf(&b, "\tstream %s\n}\n", stream.Interface)
	}
	return b.String()
}

func renderServerInterceptorWrappers(service string, interceptors []*InterceptorData) string {
	var b sourceBuilder
	for _, interceptor := range interceptors {
		for _, method := range interceptor.Methods {
			b.Add("\n")
			b.Add("\n")
			b.Add(codegen.Comment(fmt.Sprintf("wrap%s%s applies the %s server interceptor to endpoints.", interceptor.Name, method.MethodName, interceptor.DesignName)))
			b.Add("\n")
			fmt.Fprintf(&b, "func wrap%s%s(endpoint loom.Endpoint, i ServerInterceptors) loom.Endpoint {\n", method.MethodName, interceptor.Name)
			b.Add("\treturn func(ctx context.Context, req any) (any, error) {\n")
			if interceptor.HasStreamingPayloadAccess || interceptor.HasStreamingResultAccess {
				fmt.Fprintf(&b, "\t\tstream := req.(*%s).Stream\n", method.ServerStream.EndpointStruct)
				fmt.Fprintf(&b, "\t\treq.(*%s).Stream = &wrapped%s{\n\t\t\tctx:     ctx,\n", method.ServerStream.EndpointStruct, method.ServerStream.Interface)
				if interceptor.HasStreamingResultAccess {
					fmt.Fprintf(&b, "\t\t\tsendWithContext: func(ctx context.Context, req %s) error {\n", method.ServerStream.SendTypeRef)
					fmt.Fprintf(&b, "\t\t\t\tinfo := &%sInfo{\n\t\t\t\t\tservice:    %q,\n\t\t\t\t\tmethod:     %q,\n\t\t\t\t\tcallType:   loom.InterceptorStreamingSend,\n\t\t\t\t\trawPayload: req,\n\t\t\t\t}\n", interceptor.Name, service, method.MethodName)
					fmt.Fprintf(&b, "\t\t\t\t_, err := i.%s(ctx, info, func(ctx context.Context, req any) (any, error) {\n", interceptor.Name)
					fmt.Fprintf(&b, "\t\t\t\t\tcastReq, _ := req.(%s)\n\t\t\t\t\treturn nil, stream.%s(ctx, castReq)\n\t\t\t\t})\n\t\t\t\treturn err\n\t\t\t},\n", method.ServerStream.SendTypeRef, method.ServerStream.SendWithContextName)
				}
				if interceptor.HasStreamingPayloadAccess {
					fmt.Fprintf(&b, "\t\t\trecvWithContext: func(ctx context.Context) (%s, error) {\n", method.ServerStream.RecvTypeRef)
					fmt.Fprintf(&b, "\t\t\t\tinfo := &%sInfo{\n\t\t\t\t\tservice:  %q,\n\t\t\t\t\tmethod:   %q,\n\t\t\t\t\tcallType: loom.InterceptorStreamingRecv,\n\t\t\t\t}\n", interceptor.Name, service, method.MethodName)
					fmt.Fprintf(&b, "\t\t\t\tres, err := i.%s(ctx, info, func(ctx context.Context, _ any) (any, error) {\n\t\t\t\t\treturn stream.%s(ctx)\n\t\t\t\t})\n", interceptor.Name, method.ServerStream.RecvWithContextName)
					fmt.Fprintf(&b, "\t\t\t\tcastRes, _ := res.(%s)\n\t\t\t\treturn castRes, err\n\t\t\t},\n", method.ServerStream.RecvTypeRef)
				}
				fmt.Fprintf(&b, "\t\t\tstream: stream,\n\t\t}\n")
				if interceptor.HasPayloadAccess {
					fmt.Fprintf(&b, "\t\tinfo := &%sInfo{\n\t\t\tservice:    %q,\n\t\t\tmethod:     %q,\n\t\t\tcallType:   loom.InterceptorUnary,\n\t\t\trawPayload: req,\n\t\t}\n", interceptor.Name, service, method.MethodName)
					fmt.Fprintf(&b, "\t\treturn i.%s(ctx, info, endpoint)\n", interceptor.Name)
				} else {
					b.Add("\t\treturn endpoint(ctx, req)\n")
				}
			} else {
				fmt.Fprintf(&b, "\t\tinfo := &%sInfo{\n\t\t\tservice:    %q,\n\t\t\tmethod:     %q,\n\t\t\tcallType:   loom.InterceptorUnary,\n\t\t\trawPayload: req,\n\t\t}\n", interceptor.Name, service, method.MethodName)
				fmt.Fprintf(&b, "\t\treturn i.%s(ctx, info, endpoint)\n", interceptor.Name)
			}
			b.Add("\t}\n}\n")
		}
	}
	return b.String()
}

func renderClientInterceptorWrappers(service string, interceptors []*InterceptorData) string {
	var b sourceBuilder
	for _, interceptor := range interceptors {
		for _, method := range interceptor.Methods {
			b.Add("\n")
			b.Add("\n")
			b.Add(codegen.Comment(fmt.Sprintf("wrapClient%s%s applies the %s client interceptor to endpoints.", interceptor.Name, method.MethodName, interceptor.DesignName)))
			b.Add("\n")
			fmt.Fprintf(&b, "func wrapClient%s%s(endpoint loom.Endpoint, i ClientInterceptors) loom.Endpoint {\n", method.MethodName, interceptor.Name)
			b.Add("\treturn func(ctx context.Context, req any) (any, error) {\n")
			if interceptor.HasStreamingPayloadAccess || interceptor.HasStreamingResultAccess {
				if interceptor.HasPayloadAccess {
					fmt.Fprintf(&b, "\t\tinfo := &%sInfo{\n\t\t\tservice:    %q,\n\t\t\tmethod:     %q,\n\t\t\tcallType:   loom.InterceptorUnary,\n\t\t\trawPayload: req,\n\t\t}\n", interceptor.Name, service, method.MethodName)
					fmt.Fprintf(&b, "\t\tres, err := i.%s(ctx, info, endpoint)\n", interceptor.Name)
				} else {
					b.Add("\t\tres, err := endpoint(ctx, req)\n")
				}
				b.Add("\t\tif err != nil {\n\t\t\treturn res, err\n\t\t}\n")
				fmt.Fprintf(&b, "\t\tstream := res.(%s)\n", method.ClientStream.Interface)
				fmt.Fprintf(&b, "\t\treturn &wrapped%s{\n\t\t\tctx: ctx,\n", method.ClientStream.Interface)
				if interceptor.HasStreamingPayloadAccess {
					fmt.Fprintf(&b, "\t\t\tsendWithContext: func(ctx context.Context, req %s) error {\n", method.ClientStream.SendTypeRef)
					fmt.Fprintf(&b, "\t\t\t\tinfo := &%sInfo{\n\t\t\t\t\tservice:    %q,\n\t\t\t\t\tmethod:     %q,\n\t\t\t\t\tcallType:   loom.InterceptorStreamingSend,\n\t\t\t\t\trawPayload: req,\n\t\t\t\t}\n", interceptor.Name, service, method.MethodName)
					fmt.Fprintf(&b, "\t\t\t\t_, err := i.%s(ctx, info, func(ctx context.Context, req any) (any, error) {\n", interceptor.Name)
					fmt.Fprintf(&b, "\t\t\t\t\tcastReq, _ := req.(%s)\n\t\t\t\t\treturn nil, stream.%s(ctx, castReq)\n\t\t\t\t})\n\t\t\t\treturn err\n\t\t\t},\n", method.ClientStream.SendTypeRef, method.ClientStream.SendWithContextName)
				}
				if interceptor.HasStreamingResultAccess {
					fmt.Fprintf(&b, "\t\t\trecvWithContext: func(ctx context.Context) (%s, error) {\n", method.ClientStream.RecvTypeRef)
					fmt.Fprintf(&b, "\t\t\t\tinfo := &%sInfo{\n\t\t\t\t\tservice:  %q,\n\t\t\t\t\tmethod:   %q,\n\t\t\t\t\tcallType: loom.InterceptorStreamingRecv,\n\t\t\t\t}\n", interceptor.Name, service, method.MethodName)
					fmt.Fprintf(&b, "\t\t\t\tres, err := i.%s(ctx, info, func(ctx context.Context, _ any) (any, error) {\n\t\t\t\t\treturn stream.%s(ctx)\n\t\t\t\t})\n", interceptor.Name, method.ClientStream.RecvWithContextName)
					fmt.Fprintf(&b, "\t\t\t\tcastRes, _ := res.(%s)\n\t\t\t\treturn castRes, err\n\t\t\t},\n", method.ClientStream.RecvTypeRef)
				}
				fmt.Fprintf(&b, "\t\t\tstream: stream,\n\t\t}, nil\n")
			} else {
				fmt.Fprintf(&b, "\t\tinfo := &%sInfo{\n\t\t\tservice:    %q,\n\t\t\tmethod:     %q,\n\t\t\tcallType:   loom.InterceptorUnary,\n\t\t\trawPayload: req,\n\t\t}\n", interceptor.Name, service, method.MethodName)
				fmt.Fprintf(&b, "\t\treturn i.%s(ctx, info, endpoint)\n", interceptor.Name)
			}
			b.Add("\t}\n}\n")
		}
	}
	return b.String()
}

func renderStreamWrappers(streams []*StreamInterceptorData, server bool) string {
	var b sourceBuilder
	for _, stream := range streams {
		b.Add("\n")
		if server || stream.SendTypeRef != "" {
			b.Add(codegen.Comment("Unwrap returns the underlying stream type."))
			b.Add("\n")
			fmt.Fprintf(&b, "func (w *wrapped%s) Unwrap() any {\n\treturn w.stream\n}\n", stream.Interface)
		}
		if stream.SendTypeRef != "" {
			b.Add("\n")
			b.Add(codegen.Comment(fmt.Sprintf("%s streams instances of %q after executing the applied interceptor.", stream.SendName, stream.Interface)))
			b.Add("\n")
			fmt.Fprintf(&b, "func (w *wrapped%s) %s(v %s) error {\n\treturn w.%s(w.ctx, v)\n}\n\n", stream.Interface, stream.SendName, stream.SendTypeRef, stream.SendWithContextName)
			b.Add(codegen.Comment(fmt.Sprintf("%s streams instances of %q after executing the applied interceptor with context.", stream.SendWithContextName, stream.Interface)))
			b.Add("\n")
			fmt.Fprintf(&b, "func (w *wrapped%s) %s(ctx context.Context, v %s) error {\n", stream.Interface, stream.SendWithContextName, stream.SendTypeRef)
			fmt.Fprintf(&b, "\tif w.sendWithContext == nil {\n\t\treturn w.stream.%s(ctx, v)\n\t}\n\treturn w.sendWithContext(ctx, v)\n}\n", stream.SendWithContextName)
		}
		if stream.RecvTypeRef != "" {
			b.Add("\n")
			b.Add(codegen.Comment(fmt.Sprintf("%s reads instances of %q from the stream after executing the applied interceptor.", stream.RecvName, stream.Interface)))
			b.Add("\n")
			fmt.Fprintf(&b, "func (w *wrapped%s) %s() (%s, error) {\n\treturn w.%s(w.ctx)\n}\n\n", stream.Interface, stream.RecvName, stream.RecvTypeRef, stream.RecvWithContextName)
			b.Add(codegen.Comment(fmt.Sprintf("%s reads instances of %q from the stream after executing the applied interceptor with context.", stream.RecvWithContextName, stream.Interface)))
			b.Add("\n")
			fmt.Fprintf(&b, "func (w *wrapped%s) %s(ctx context.Context) (%s, error) {\n", stream.Interface, stream.RecvWithContextName, stream.RecvTypeRef)
			fmt.Fprintf(&b, "\tif w.recvWithContext == nil {\n\t\treturn w.stream.%s(ctx)\n\t}\n\treturn w.recvWithContext(ctx)\n}\n", stream.RecvWithContextName)
		}
		if stream.MustClose {
			b.Add("\n// Close closes the stream.\n")
			fmt.Fprintf(&b, "func (w *wrapped%s) Close() error {\n\treturn w.stream.Close()\n}\n", stream.Interface)
		}
	}
	return b.String()
}
