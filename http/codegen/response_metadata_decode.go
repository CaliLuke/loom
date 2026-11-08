package codegen

import (
	"fmt"
	"strings"
)

func responseMetadataDecode(response *ResponseData, endpoint *EndpointData, data *ServiceData, cookies string) (string, string, string) {
	if len(response.Cookies) > 0 && cookies == "" {
		panic("response metadata cookies require an actual response source")
	}
	vars := newResponseVarScope(data)
	for _, header := range response.Headers {
		local := header.Locals
		vars.reserve(header.VarName, local.Raw, local.Present, local.Val, local.Slice, local.Encoded, local.EncodedRaw)
	}
	for _, cookie := range response.Cookies {
		local := cookie.Locals
		vars.reserve(cookie.VarName, local.Raw, local.Present, local.Val, local.Slice, local.Encoded, local.EncodedRaw)
	}
	status, headers := vars.names.Unique("status"), vars.names.Unique("headers")
	code, err := renderResponseMetadataFields(response, endpoint.ServiceName, endpoint.Method, data, headers, cookies)
	if err != nil {
		panic(err)
	}
	var out strings.Builder
	out.WriteString(code)
	out.WriteString("\n")
	if response.MustValidate {
		fmt.Fprintf(&out, "if err != nil {\nreturn nil, loomhttp.ErrValidationError(%q, %q, err)\n}\n", endpoint.ServiceName, endpoint.Method.Name)
	}
	init := response.ResultInit
	if init == nil {
		panic("response metadata has no native constructor")
	}
	args := strings.Join(responseConstructorArgRefs(init.ClientArgs, status), ", ")
	result, failure := vars.names.Unique("decoded"), vars.names.Unique("decodeErr")
	validation := responseConstructedStatusValidation(response, result)
	switch {
	case init.ClientErrorAware:
		fmt.Fprintf(&out, "%s, %s := %s(%s)\nif %s != nil {\nreturn nil, loomhttp.ErrValidationError(%q, %q, %s)\n}\n", result, failure, init.Name, args, failure, endpoint.ServiceName, endpoint.Method.Name, failure)
	case validation != "":
		fmt.Fprintf(&out, "%s := %s(%s)\n", result, init.Name, args)
	default:
		fmt.Fprintf(&out, "return %s(%s), nil\n", init.Name, args)
		return status, headers, out.String()
	}
	if validation != "" {
		fmt.Fprintf(&out, "if %s := func() (err error) {\n%s\nreturn err\n}(); %s != nil {\nreturn nil, loomhttp.ErrValidationError(%q, %q, %s)\n}\n", failure, validation, failure, endpoint.ServiceName, endpoint.Method.Name, failure)
	}
	fmt.Fprintf(&out, "return %s, nil\n", result)
	return status, headers, out.String()
}
