//nolint:errcheck // Generator helpers write only to in-memory builders.
package service

import (
	"fmt"
)

func renderPayloadAccessSwitch(interceptor *InterceptorData, server bool) string {
	if len(interceptor.Methods) == 1 {
		method := interceptor.Methods[0]
		if server && hasEndpointStruct(true)(method) {
			return "\tswitch pay := info.RawPayload().(type) {\n\tcase *" + method.ServerStream.EndpointStruct + ":\n\t\treturn &" + method.PayloadAccess + "{payload: pay.Payload}\n\tdefault:\n\t\treturn &" + method.PayloadAccess + "{payload: pay.(" + method.PayloadRef + ")}\n\t}\n"
		}
		return "\treturn &" + method.PayloadAccess + "{payload: info.RawPayload().(" + method.PayloadRef + ")}\n"
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
		return "\treturn &" + method.ResultAccess + "{result: res.(" + method.ResultRef + ")}\n"
	}
	var b sourceBuilder
	b.Add("\tswitch info.Method() {\n")
	for _, method := range interceptor.Methods {
		fmt.Fprintf(&b, "\tcase %q:\n\t\treturn &%s{result: res.(%s)}\n", method.MethodName, method.ResultAccess, method.ResultRef)
	}
	b.Add("\tdefault:\n\t\treturn nil\n\t}\n")
	return b.String()
}

// renderViewedResultInterception renders the end of the server wrapper of an
// interceptor that accesses the result of a method whose endpoint returns the
// viewed result. The interceptor reads and writes the result through the
// accessor of the result type, so the wrapper converts the viewed result that
// the endpoint returns to the result type and projects the result that the
// interceptor returns with the view of the endpoint result, or with the view
// of the design when the interceptor does not call the endpoint. A value of
// another type or a nil pointer fails with a fault error instead of a panic.
func renderViewedResultInterception(interceptor *InterceptorData, method *MethodInterceptorData) string {
	viewed := method.ViewedResult
	var b sourceBuilder
	fmt.Fprintf(&b, "view := %q\n", viewed.ViewName)
	fmt.Fprintf(&b, "res, err := i.%s(ctx, info, func(ctx context.Context, req any) (any, error) {\n", interceptor.Name)
	b.Add("\tres, err := endpoint(ctx, req)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n")
	fmt.Fprintf(&b, "\tvres, ok := res.(%s)\n\tif !ok || vres == nil {\n\t\treturn nil, %s\n\t}\n", viewed.FullRef, invalidResultFault(viewed.FullRef))
	fmt.Fprintf(&b, "\tview = vres.View\n\treturn %s(vres)\n})\n", viewed.ResultInit.Name)
	b.Add("if err != nil {\n\treturn nil, err\n}\n")
	// Interceptors access only object results, which are pointers.
	fmt.Fprintf(&b, "result, ok := res.(%s)\nif !ok || result == nil {\n\treturn nil, %s\n}\n", method.ResultRef, invalidResultFault(method.ResultRef))
	fmt.Fprintf(&b, "return %s(result, view)", viewed.Init.Name)
	return b.String()
}

// invalidResultFault returns the expression of the fault error that a server
// interceptor wrapper returns when a result does not have the type typeRef.
func invalidResultFault(typeRef string) string {
	return fmt.Sprintf("loom.Fault(%q, res)", "invalid value expected "+typeRef+", got %v")
}

func renderStreamingPayloadAccess(interceptor *InterceptorData, client bool) string {
	if len(interceptor.Methods) == 1 {
		method := interceptor.Methods[0]
		arg := "info.RawPayload()"
		if !client {
			arg = "pay"
		}
		return "\treturn &" + method.StreamingPayloadAccess + "{payload: " + arg + ".(" + method.StreamingPayloadRef + ")}\n"
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
			return "\treturn &" + method.StreamingResultAccess + "{result: res.(" + method.StreamingResultRef + ")}\n"
		}
		return "\treturn &" + method.StreamingResultAccess + "{result: info.RawPayload().(" + method.StreamingResultRef + ")}\n"
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
