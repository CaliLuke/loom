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
