//nolint:errcheck // Generator helpers write only to in-memory builders.
package service

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func serverInterceptorsInterfaceSection(svc *Data) codegen.Section {
	return codegen.NewJenniferSection("server-interceptors-type", func(stmt *jen.Statement) {
		addInterceptorsInterfaceSection(stmt, svc.ServerInterceptors, true)
	})
}

func clientInterceptorsInterfaceSection(svc *Data) codegen.Section {
	return codegen.NewJenniferSection("client-interceptors-type", func(stmt *jen.Statement) {
		addInterceptorsInterfaceSection(stmt, svc.ClientInterceptors, false)
	})
}

func interceptorTypesSection(interceptors []*InterceptorData) codegen.Section {
	return codegen.NewJenniferSection("interceptor-types", func(stmt *jen.Statement) {
		addInterceptorTypesSection(stmt, interceptors)
	})
}

func endpointWrapperSection(server bool, methodVarName, method string, interceptors []string) codegen.Section {
	name := "client-wrapper"
	if server {
		name = "endpoint-wrapper"
	}
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		addEndpointWrapperSection(stmt, server, methodVarName, method, interceptors)
	})
}

func interceptorsSection(interceptors []*InterceptorData, server bool) codegen.Section {
	return codegen.NewJenniferSection("interceptors", func(stmt *jen.Statement) {
		addInterceptorsSection(stmt, interceptors, server)
	})
}

func streamWrapperTypesSection(name string, streams []*StreamInterceptorData, server bool) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		addStreamWrapperTypesSection(stmt, streams, server)
	})
}

func serverInterceptorWrappersSection(service string, interceptors []*InterceptorData) codegen.Section {
	return codegen.NewJenniferSection("server-interceptor-wrappers", func(stmt *jen.Statement) {
		addServerInterceptorWrappersSection(stmt, service, interceptors)
	})
}

func clientInterceptorWrappersSection(service string, interceptors []*InterceptorData) codegen.Section {
	return codegen.NewJenniferSection("client-interceptor-wrappers", func(stmt *jen.Statement) {
		addClientInterceptorWrappersSection(stmt, service, interceptors)
	})
}

func streamWrappersSection(name string, streams []*StreamInterceptorData, server bool) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		addStreamWrappersSection(stmt, streams, server)
	})
}

func addInterceptorsInterfaceSection(stmt *jen.Statement, interceptors []*InterceptorData, server bool) {
	name := "ClientInterceptors"
	if server {
		name = "ServerInterceptors"
	}
	addInterceptorsInterfaceComment(stmt, server)
	stmt.Type().Id(name).InterfaceFunc(func(group *jen.Group) {
		for _, interceptor := range interceptors {
			addInterceptorInterfaceMethod(group, interceptor)
		}
	})
	stmt.Line()
}

func addInterceptorsInterfaceComment(stmt *jen.Statement, server bool) {
	if server {
		stmt.Comment("ServerInterceptors defines the interface for all server-side interceptors.").Line()
		stmt.Comment("Server interceptors execute after the request is decoded and before the").Line()
		stmt.Comment("payload is sent to the service. The implementation is responsible for calling").Line()
		stmt.Comment("next to complete the request.").Line()
		return
	}
	stmt.Comment("ClientInterceptors defines the interface for all client-side interceptors.").Line()
	stmt.Comment("Client interceptors execute after the payload is encoded and before the request").Line()
	stmt.Comment("is sent to the server. The implementation is responsible for calling next to").Line()
	stmt.Comment("complete the request.").Line()
}

func addInterceptorInterfaceMethod(group *jen.Group, interceptor *InterceptorData) {
	if interceptor.Description != "" {
		for _, line := range strings.Split(codegen.Comment(interceptor.Description), "\n") {
			group.Comment(strings.TrimPrefix(line, "// "))
		}
	}
	group.Id(interceptor.Name).Params(
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("info").Op("*").Id(interceptor.Name+"Info"),
		jen.Id("next").Add(codegen.TypeRef("loom.Endpoint")),
	).Params(jen.Any(), jen.Error())
}

func addInterceptorTypesSection(stmt *jen.Statement, interceptors []*InterceptorData) {
	stmt.Line()
	stmt.Comment("Access interfaces for interceptor payloads and results").Line()
	stmt.Type().DefsFunc(func(group *jen.Group) {
		for _, interceptor := range interceptors {
			addInterceptorInfoType(group, interceptor)
			addInterceptorAccessorInterface(group, interceptor.Name, "Payload", interceptor.HasPayloadAccess, interceptor.ReadPayload, interceptor.WritePayload)
			addInterceptorAccessorInterface(group, interceptor.Name, "Result", interceptor.HasResultAccess, interceptor.ReadResult, interceptor.WriteResult)
			addInterceptorAccessorInterface(group, interceptor.Name, "StreamingPayload", interceptor.HasStreamingPayloadAccess, interceptor.ReadStreamingPayload, interceptor.WriteStreamingPayload)
			addInterceptorAccessorInterface(group, interceptor.Name, "StreamingResult", interceptor.HasStreamingResultAccess, interceptor.ReadStreamingResult, interceptor.WriteStreamingResult)
		}
	})
	if !hasPrivateImplementationTypes(interceptors) {
		stmt.Line()
		return
	}
	stmt.Line()
	stmt.Comment("Private implementation types").Line()
	stmt.Type().DefsFunc(func(group *jen.Group) {
		addInterceptorMethodStructs(group, interceptors, func(m *MethodInterceptorData) (string, string) { return m.PayloadAccess, m.PayloadRef }, "payload")
		addInterceptorMethodStructs(group, interceptors, func(m *MethodInterceptorData) (string, string) { return m.ResultAccess, m.ResultRef }, "result")
		addInterceptorMethodStructs(group, interceptors, func(m *MethodInterceptorData) (string, string) {
			return m.StreamingPayloadAccess, m.StreamingPayloadRef
		}, "payload")
		addInterceptorMethodStructs(group, interceptors, func(m *MethodInterceptorData) (string, string) { return m.StreamingResultAccess, m.StreamingResultRef }, "result")
	})
	stmt.Line()
}

func addInterceptorInfoType(group *jen.Group, interceptor *InterceptorData) {
	addIndentedGroupComment(group, fmt.Sprintf("%sInfo provides metadata about the current interception.\nIt includes service name, method name, and access to the endpoint.", interceptor.Name))
	group.Id(interceptor.Name+"Info").Struct(
		jen.Id("service").String(),
		jen.Id("method").String(),
		jen.Id("callType").Add(codegen.TypeRef("loom.InterceptorCallType")),
		jen.Id("rawPayload").Any(),
	)
}

func addInterceptorAccessorInterface(group *jen.Group, name, suffix string, enabled bool, readFields, writeFields []*AttributeData) {
	if !enabled {
		return
	}
	group.Line()
	desc := fmt.Sprintf("%s%s provides type-safe access to the method %s.\nIt allows reading and writing specific fields of the %s as defined\nin the design.", name, suffix, interfaceAccessTarget(suffix), interfaceAccessTarget(suffix))
	if strings.HasPrefix(suffix, "Streaming") {
		group.Comment(name + suffix + " provides type-safe access to the method " + interfaceAccessTarget(suffix) + ".")
		group.Comment("It allows reading and writing specific fields of the " + interfaceAccessTarget(suffix) + " as defined")
		group.Comment("in the design.")
	} else {
		addIndentedGroupComment(group, desc)
	}
	group.Id(name + suffix).InterfaceFunc(func(methods *jen.Group) {
		for _, field := range readFields {
			methods.Id(field.Name).Params().Add(codegen.TypeRef(field.TypeRef))
		}
		for _, field := range writeFields {
			methods.Id("Set" + field.Name).Params(codegen.TypeRef(field.TypeRef))
		}
	})
}

func addInterceptorMethodStructs(group *jen.Group, interceptors []*InterceptorData, pick func(*MethodInterceptorData) (string, string), fieldName string) {
	for _, interceptor := range interceptors {
		for _, method := range interceptor.Methods {
			typeName, typeRef := pick(method)
			if typeName == "" {
				continue
			}
			group.Id(typeName).Struct(
				jen.Id(fieldName).Add(codegen.TypeRef(typeRef)),
			)
		}
	}
}

func addIndentedGroupComment(group *jen.Group, text string) {
	for _, line := range strings.Split(codegen.Comment(text), "\n") {
		group.Comment(strings.TrimPrefix(line, "// "))
	}
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

func addEndpointWrapperSection(stmt *jen.Statement, server bool, methodVarName, method string, interceptors []string) {
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
	codegen.Doc(stmt, fmt.Sprintf("%s wraps the %s endpoint with the %s interceptors defined in the design.", wrapName, method, commentTarget))
	stmt.Func().
		Id(wrapName).
		Params(
			jen.Id("endpoint").Add(codegen.TypeRef("loom.Endpoint")),
			jen.Id("i").Id(interfaceName),
		).
		Add(codegen.TypeRef("loom.Endpoint")).
		BlockFunc(func(group *jen.Group) {
			group.If(jen.Id("i").Op("!=").Nil()).BlockFunc(func(block *jen.Group) {
				for _, interceptor := range interceptors {
					block.Id("endpoint").Op("=").Id(callPrefix+interceptor).Call(jen.Id("endpoint"), jen.Id("i"))
				}
			})
			group.Return(jen.Id("endpoint"))
		})
	stmt.Line()
}

func addInterceptorsSection(stmt *jen.Statement, interceptors []*InterceptorData, server bool) {
	stmt.Comment("Public accessor methods for Info types").Line()
	for _, interceptor := range interceptors {
		addInterceptorAccessors(stmt, interceptor, server)
	}
	if hasPrivateImplementationTypes(interceptors) {
		stmt.Line()
		stmt.Comment("Private implementation methods").Line()
		first := true
		for _, interceptor := range interceptors {
			for _, method := range interceptor.Methods {
				first = addAccessorMethods(stmt, method.PayloadAccess, "payload", interceptor.ReadPayload, interceptor.WritePayload, first)
				first = addAccessorMethods(stmt, method.ResultAccess, "result", interceptor.ReadResult, interceptor.WriteResult, first)
				first = addAccessorMethods(stmt, method.StreamingPayloadAccess, "payload", interceptor.ReadStreamingPayload, interceptor.WriteStreamingPayload, first)
				first = addAccessorMethods(stmt, method.StreamingResultAccess, "result", interceptor.ReadStreamingResult, interceptor.WriteStreamingResult, first)
			}
		}
	}
}

func addInterceptorAccessors(stmt *jen.Statement, interceptor *InterceptorData, server bool) {
	stmt.Line()
	addInfoAccessorMethod(stmt, interceptor, "Service", "Service returns the name of the service handling the request.", jen.String(), "info.service")
	addInfoAccessorMethod(stmt, interceptor, "Method", "Method returns the name of the method handling the request.", jen.String(), "info.method")
	addInfoAccessorMethod(stmt, interceptor, "CallType", "CallType returns the type of call the interceptor is handling.", codegen.TypeRef("loom.InterceptorCallType"), "info.callType")
	addInfoAccessorMethod(stmt, interceptor, "RawPayload", "RawPayload returns the raw payload of the request.", jen.Any(), "info.rawPayload")
	addPayloadAccessor(stmt, interceptor, server)
	addResultAccessor(stmt, interceptor)
	addStreamingPayloadAccessor(stmt, interceptor, true)
	addStreamingResultAccessor(stmt, interceptor, true)
	addStreamingPayloadAccessor(stmt, interceptor, false)
	addStreamingResultAccessor(stmt, interceptor, false)
}

func addPayloadAccessor(stmt *jen.Statement, interceptor *InterceptorData, server bool) {
	if !interceptor.HasPayloadAccess {
		return
	}
	stmt.Line()
	codegen.Doc(stmt, "Payload returns a type-safe accessor for the method payload.")
	stmt.Func().
		Params(jen.Id("info").Op("*").Id(interceptor.Name + "Info")).
		Id("Payload").
		Params().
		Id(interceptor.Name + "Payload").
		BlockFunc(func(group *jen.Group) {
			addRawGroup(group, renderPayloadAccessSwitch(interceptor, server))
		})
	stmt.Line()
}

func addResultAccessor(stmt *jen.Statement, interceptor *InterceptorData) {
	if !interceptor.HasResultAccess {
		return
	}
	stmt.Line()
	codegen.Doc(stmt, "Result returns a type-safe accessor for the method result.")
	stmt.Func().
		Params(jen.Id("info").Op("*").Id(interceptor.Name + "Info")).
		Id("Result").
		Params(jen.Id("res").Any()).
		Id(interceptor.Name + "Result").
		BlockFunc(func(group *jen.Group) {
			addRawGroup(group, renderResultAccessSwitch(interceptor))
		})
	stmt.Line()
}

func addStreamingPayloadAccessor(stmt *jen.Statement, interceptor *InterceptorData, client bool) {
	if !interceptor.HasStreamingPayloadAccess {
		return
	}
	methodName := "ServerStreamingPayload"
	doc := "ServerStreamingPayload returns a type-safe accessor for the method streaming payload for a server-side interceptor."
	params := []jen.Code{jen.Id("pay").Any()}
	if client {
		methodName = "ClientStreamingPayload"
		doc = "ClientStreamingPayload returns a type-safe accessor for the method streaming payload for a client-side interceptor."
		params = nil
	}
	stmt.Line()
	stmt.Comment(doc).Line()
	stmt.Func().
		Params(jen.Id("info").Op("*").Id(interceptor.Name + "Info")).
		Id(methodName).
		Params(params...).
		Id(interceptor.Name + "StreamingPayload").
		BlockFunc(func(group *jen.Group) {
			addRawGroup(group, renderStreamingPayloadAccess(interceptor, client))
		})
	stmt.Line()
}

func addStreamingResultAccessor(stmt *jen.Statement, interceptor *InterceptorData, client bool) {
	if !interceptor.HasStreamingResultAccess {
		return
	}
	methodName := "ServerStreamingResult"
	doc := "ServerStreamingResult returns a type-safe accessor for the method streaming result for a server-side interceptor."
	params := []jen.Code{}
	if client {
		methodName = "ClientStreamingResult"
		doc = "ClientStreamingResult returns a type-safe accessor for the method streaming result for a client-side interceptor."
		params = []jen.Code{jen.Id("res").Any()}
	}
	stmt.Line()
	stmt.Comment(doc).Line()
	stmt.Func().
		Params(jen.Id("info").Op("*").Id(interceptor.Name + "Info")).
		Id(methodName).
		Params(params...).
		Id(interceptor.Name + "StreamingResult").
		BlockFunc(func(group *jen.Group) {
			addRawGroup(group, renderStreamingResultAccess(interceptor, client))
		})
	stmt.Line()
}

func addInfoAccessorMethod(stmt *jen.Statement, interceptor *InterceptorData, methodName, doc string, returnType *jen.Statement, expr string) {
	codegen.Doc(stmt, doc)
	stmt.Func().
		Params(jen.Id("info").Op("*").Id(interceptor.Name + "Info")).
		Id(methodName).
		Params().
		Add(returnType).
		BlockFunc(func(group *jen.Group) {
			addRawGroup(group, "return "+expr)
		})
	stmt.Line()
}

func addAccessorMethods(stmt *jen.Statement, accessName, fieldName string, readers, writers []*AttributeData, first bool) bool {
	if accessName == "" {
		return first
	}
	receiver := "p"
	if fieldName == "result" {
		receiver = "r"
	}
	if len(readers) > 0 || len(writers) > 0 {
		if first {
			stmt.Line()
			first = false
		}
	}
	for _, field := range readers {
		stmt.Func().
			Params(jen.Id(receiver).Op("*").Id(accessName)).
			Id(field.Name).
			Params().
			Add(codegen.TypeRef(field.TypeRef)).
			BlockFunc(func(group *jen.Group) {
				if field.Pointer {
					addRawGroup(group, "if "+receiver+"."+fieldName+"."+field.Name+" == nil {\n\tvar zero "+field.TypeRef+"\n\treturn zero\n}\nreturn *"+receiver+"."+fieldName+"."+field.Name)
				} else {
					addRawGroup(group, "return "+receiver+"."+fieldName+"."+field.Name)
				}
			})
		stmt.Line()
	}
	for _, field := range writers {
		stmt.Func().
			Params(jen.Id(receiver).Op("*").Id(accessName)).
			Id("Set" + field.Name).
			Params(jen.Id("v").Add(codegen.TypeRef(field.TypeRef))).
			BlockFunc(func(group *jen.Group) {
				if field.Pointer {
					addRawGroup(group, receiver+"."+fieldName+"."+field.Name+" = &v")
				} else {
					addRawGroup(group, receiver+"."+fieldName+"."+field.Name+" = v")
				}
			})
		stmt.Line()
	}
	return first
}

func addStreamWrapperTypesSection(stmt *jen.Statement, streams []*StreamInterceptorData, server bool) {
	for i, stream := range streams {
		stmt.Line()
		if i == 0 {
			stmt.Line()
		}
		target := "client"
		if server {
			target = "server"
		}
		codegen.Doc(stmt, fmt.Sprintf("wrapped%s is a %s interceptor wrapper for the %s stream.", stream.Interface, target, stream.Interface))
		stmt.Type().Id("wrapped" + stream.Interface).StructFunc(func(group *jen.Group) {
			group.Id("ctx").Qual("context", "Context")
			if stream.SendTypeRef != "" {
				group.Id("sendWithContext").Func().Params(
					jen.Qual("context", "Context"),
					codegen.TypeRef(stream.SendTypeRef),
				).Error()
			}
			if stream.RecvTypeRef != "" {
				group.Id("recvWithContext").Func().Params(
					jen.Qual("context", "Context"),
				).Params(codegen.TypeRef(stream.RecvTypeRef), jen.Error())
			}
			group.Id("stream").Add(codegen.TypeRef(stream.Interface))
		})
	}
	stmt.Line()
}

func addServerInterceptorWrappersSection(stmt *jen.Statement, service string, interceptors []*InterceptorData) {
	addInterceptorWrappersSection(stmt, service, interceptors, false)
}

func addClientInterceptorWrappersSection(stmt *jen.Statement, service string, interceptors []*InterceptorData) {
	addInterceptorWrappersSection(stmt, service, interceptors, true)
}

func addInterceptorWrappersSection(stmt *jen.Statement, service string, interceptors []*InterceptorData, client bool) {
	for _, interceptor := range interceptors {
		for _, method := range interceptor.Methods {
			name, iface, doc, body := interceptorWrapperMeta(service, interceptor, method, client)
			stmt.Line()
			stmt.Line()
			codegen.Doc(stmt, doc)
			stmt.Func().
				Id(name).
				Params(
					jen.Id("endpoint").Add(codegen.TypeRef("loom.Endpoint")),
					jen.Id("i").Id(iface),
				).
				Add(codegen.TypeRef("loom.Endpoint")).
				BlockFunc(func(group *jen.Group) {
					group.Return(
						jen.Func().
							Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("req").Any()).
							Params(jen.Any(), jen.Error()).
							BlockFunc(func(inner *jen.Group) {
								addRawGroup(inner, body)
							}),
					)
				})
		}
	}
	stmt.Line()
}

func interceptorWrapperMeta(service string, interceptor *InterceptorData, method *MethodInterceptorData, client bool) (string, string, string, string) {
	if client {
		return "wrapClient" + method.MethodName + interceptor.Name,
			"ClientInterceptors",
			fmt.Sprintf("wrapClient%s%s applies the %s client interceptor to endpoints.", interceptor.Name, method.MethodName, interceptor.DesignName),
			renderClientInterceptorWrapperBody(service, interceptor, method)
	}
	return "wrap" + method.MethodName + interceptor.Name,
		"ServerInterceptors",
		fmt.Sprintf("wrap%s%s applies the %s server interceptor to endpoints.", interceptor.Name, method.MethodName, interceptor.DesignName),
		renderServerInterceptorWrapperBody(service, interceptor, method)
}

func renderServerInterceptorWrapperBody(service string, interceptor *InterceptorData, method *MethodInterceptorData) string {
	var b sourceBuilder
	if interceptor.HasStreamingPayloadAccess || interceptor.HasStreamingResultAccess {
		fmt.Fprintf(&b, "stream := req.(*%s).Stream\n", method.ServerStream.EndpointStruct)
		fmt.Fprintf(&b, "req.(*%s).Stream = &wrapped%s{\n\tctx:     ctx,\n", method.ServerStream.EndpointStruct, method.ServerStream.Interface)
		if interceptor.HasStreamingResultAccess {
			fmt.Fprintf(&b, "\tsendWithContext: func(ctx context.Context, req %s) error {\n", method.ServerStream.SendTypeRef)
			fmt.Fprintf(&b, "\t\tinfo := &%sInfo{\n\t\t\tservice:    %q,\n\t\t\tmethod:     %q,\n\t\t\tcallType:   loom.InterceptorStreamingSend,\n\t\t\trawPayload: req,\n\t\t}\n", interceptor.Name, service, method.MethodName)
			fmt.Fprintf(&b, "\t\t_, err := i.%s(ctx, info, func(ctx context.Context, req any) (any, error) {\n", interceptor.Name)
			fmt.Fprintf(&b, "\t\t\tcastReq, _ := req.(%s)\n\t\t\treturn nil, stream.%s(ctx, castReq)\n\t\t})\n\t\treturn err\n\t},\n", method.ServerStream.SendTypeRef, method.ServerStream.SendWithContextName)
		}
		if interceptor.HasStreamingPayloadAccess {
			fmt.Fprintf(&b, "\trecvWithContext: func(ctx context.Context) (%s, error) {\n", method.ServerStream.RecvTypeRef)
			fmt.Fprintf(&b, "\t\tinfo := &%sInfo{\n\t\t\tservice:  %q,\n\t\t\tmethod:   %q,\n\t\t\tcallType: loom.InterceptorStreamingRecv,\n\t\t}\n", interceptor.Name, service, method.MethodName)
			fmt.Fprintf(&b, "\t\tres, err := i.%s(ctx, info, func(ctx context.Context, _ any) (any, error) {\n\t\t\treturn stream.%s(ctx)\n\t\t})\n", interceptor.Name, method.ServerStream.RecvWithContextName)
			fmt.Fprintf(&b, "\t\tcastRes, _ := res.(%s)\n\t\treturn castRes, err\n\t},\n", method.ServerStream.RecvTypeRef)
		}
		fmt.Fprintf(&b, "\tstream: stream,\n}\n")
		if interceptor.HasPayloadAccess {
			fmt.Fprintf(&b, "info := &%sInfo{\n\tservice:    %q,\n\tmethod:     %q,\n\tcallType:   loom.InterceptorUnary,\n\trawPayload: req,\n}\n", interceptor.Name, service, method.MethodName)
			fmt.Fprintf(&b, "return i.%s(ctx, info, endpoint)", interceptor.Name)
		} else {
			b.Add("return endpoint(ctx, req)")
		}
		return b.String()
	}
	fmt.Fprintf(&b, "info := &%sInfo{\n\tservice:    %q,\n\tmethod:     %q,\n\tcallType:   loom.InterceptorUnary,\n\trawPayload: req,\n}\n", interceptor.Name, service, method.MethodName)
	fmt.Fprintf(&b, "return i.%s(ctx, info, endpoint)", interceptor.Name)
	return b.String()
}

func renderClientInterceptorWrapperBody(service string, interceptor *InterceptorData, method *MethodInterceptorData) string {
	var b sourceBuilder
	if interceptor.HasStreamingPayloadAccess || interceptor.HasStreamingResultAccess {
		if interceptor.HasPayloadAccess {
			fmt.Fprintf(&b, "info := &%sInfo{\n\tservice:    %q,\n\tmethod:     %q,\n\tcallType:   loom.InterceptorUnary,\n\trawPayload: req,\n}\n", interceptor.Name, service, method.MethodName)
			fmt.Fprintf(&b, "res, err := i.%s(ctx, info, endpoint)\n", interceptor.Name)
		} else {
			b.Add("res, err := endpoint(ctx, req)\n")
		}
		b.Add("if err != nil {\n\treturn res, err\n}\n")
		fmt.Fprintf(&b, "stream := res.(%s)\n", method.ClientStream.Interface)
		fmt.Fprintf(&b, "return &wrapped%s{\n\tctx: ctx,\n", method.ClientStream.Interface)
		if interceptor.HasStreamingPayloadAccess {
			fmt.Fprintf(&b, "\tsendWithContext: func(ctx context.Context, req %s) error {\n", method.ClientStream.SendTypeRef)
			fmt.Fprintf(&b, "\t\tinfo := &%sInfo{\n\t\t\tservice:    %q,\n\t\t\tmethod:     %q,\n\t\t\tcallType:   loom.InterceptorStreamingSend,\n\t\t\trawPayload: req,\n\t\t}\n", interceptor.Name, service, method.MethodName)
			fmt.Fprintf(&b, "\t\t_, err := i.%s(ctx, info, func(ctx context.Context, req any) (any, error) {\n", interceptor.Name)
			fmt.Fprintf(&b, "\t\t\tcastReq, _ := req.(%s)\n\t\t\treturn nil, stream.%s(ctx, castReq)\n\t\t})\n\t\treturn err\n\t},\n", method.ClientStream.SendTypeRef, method.ClientStream.SendWithContextName)
		}
		if interceptor.HasStreamingResultAccess {
			fmt.Fprintf(&b, "\trecvWithContext: func(ctx context.Context) (%s, error) {\n", method.ClientStream.RecvTypeRef)
			fmt.Fprintf(&b, "\t\tinfo := &%sInfo{\n\t\t\tservice:  %q,\n\t\t\tmethod:   %q,\n\t\t\tcallType: loom.InterceptorStreamingRecv,\n\t\t}\n", interceptor.Name, service, method.MethodName)
			fmt.Fprintf(&b, "\t\tres, err := i.%s(ctx, info, func(ctx context.Context, _ any) (any, error) {\n\t\t\treturn stream.%s(ctx)\n\t\t})\n", interceptor.Name, method.ClientStream.RecvWithContextName)
			fmt.Fprintf(&b, "\t\tcastRes, _ := res.(%s)\n\t\treturn castRes, err\n\t},\n", method.ClientStream.RecvTypeRef)
		}
		fmt.Fprintf(&b, "\tstream: stream,\n}, nil")
		return b.String()
	}
	fmt.Fprintf(&b, "info := &%sInfo{\n\tservice:    %q,\n\tmethod:     %q,\n\tcallType:   loom.InterceptorUnary,\n\trawPayload: req,\n}\n", interceptor.Name, service, method.MethodName)
	fmt.Fprintf(&b, "return i.%s(ctx, info, endpoint)", interceptor.Name)
	return b.String()
}
