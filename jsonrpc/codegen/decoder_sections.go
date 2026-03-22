package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcResponseDecoderSection(e *httpcodegen.EndpointData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-response-decoder", renderJSONRPCResponseDecoder(e))
}

func renderJSONRPCResponseDecoder(e *httpcodegen.EndpointData) string {
	var b strings.Builder
	comment := fmt.Sprintf("%s returns a decoder for responses returned by the %s service %s JSON-RPC method. restoreBody controls whether the response body should be restored after having been read.", e.ResponseDecoder, e.ServiceName, e.Method.Name)
	b.WriteString("\n")
	b.WriteString(codegen.Comment(comment))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {\n", e.ResponseDecoder)
	b.WriteString("\treturn func(resp *http.Response) (any, error) {\n")
	b.WriteString("\t\tif restoreBody {\n")
	b.WriteString("\t\t\tb, err := io.ReadAll(resp.Body)\n")
	b.WriteString("\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n")
	b.WriteString("\t\t\tresp.Body = io.NopCloser(bytes.NewBuffer(b))\n")
	b.WriteString("\t\t\tdefer func() {\n\t\t\t\tresp.Body = io.NopCloser(bytes.NewBuffer(b))\n\t\t\t}()\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tdefer resp.Body.Close()\n\n")
	b.WriteString("\t\tif resp.StatusCode != http.StatusOK {\n")
	fmt.Fprintf(&b, "\t\t\tbody, _ := io.ReadAll(resp.Body)\n\t\t\treturn nil, goahttp.ErrInvalidResponse(%q, %q, resp.StatusCode, string(body))\n", e.ServiceName, e.Method.Name)
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tvar jresp jsonrpc.RawResponse\n")
	b.WriteString("\t\tif err := decoder(resp).Decode(&jresp); err != nil {\n")
	fmt.Fprintf(&b, "\t\t\treturn nil, goahttp.ErrDecodingError(%q, %q, err)\n", e.ServiceName, e.Method.Name)
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tif jresp.Error != nil {\n")
	b.WriteString("\t\t\tswitch jresp.Error.Code {\n")
	writeJSONRPCErrorDecodeSwitch(&b, e)
	b.WriteString("\t\t\tdefault:\n")
	fmt.Fprintf(&b, "\t\t\t\tbody, _ := io.ReadAll(resp.Body)\n\t\t\t\treturn nil, goahttp.ErrInvalidResponse(%q, %q, resp.StatusCode, string(body))\n", e.ServiceName, e.Method.Name)
	b.WriteString("\t\t\t}\n\t\t}\n\n")
	if e.Result != nil && len(e.Result.Responses) > 0 {
		resp := e.Result.Responses[0]
		b.WriteString("\t\tresp.Body = io.NopCloser(bytes.NewBuffer(jresp.Result))\n")
		renderSingleResponseDecode(&b, resp, e.ServiceName, e.Method)
		switch {
		case resp.ResultInit != nil:
			if resp.ViewedResult != nil {
				b.WriteString("\t\tp := " + resp.ResultInit.Name + "(")
				for i, arg := range resp.ResultInit.ClientArgs {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString(arg.Ref)
				}
				b.WriteString(")\n")
				if resp.TagName != "" {
					fmt.Fprintf(&b, "\t\ttmp := %q\n\t\tp.%s = &tmp\n", resp.TagValue, resp.TagName)
				}
				if e.Method.ViewedResult != nil && e.Method.ViewedResult.ViewName != "" {
					fmt.Fprintf(&b, "\t\tview := %q\n", e.Method.ViewedResult.ViewName)
				} else {
					b.WriteString("\t\tview := resp.Header.Get(\"loom-view\")\n")
				}
				fmt.Fprintf(&b, "\t\tvres := %s%s.%s{Projected: p, View: view}\n", viewedResultPrefix(e.Method.ViewedResult), e.Method.ViewedResult.ViewsPkg, e.Method.ViewedResult.VarName)
				if resp.ClientBody != nil {
					fmt.Fprintf(&b, "\t\tif err = %s.Validate%s(vres); err != nil {\n", e.Method.ViewedResult.ViewsPkg, e.Method.Result)
					fmt.Fprintf(&b, "\t\t\treturn nil, goahttp.ErrValidationError(%q, %q, err)\n", e.ServiceName, e.Method.Name)
					b.WriteString("\t\t}\n")
				}
				fmt.Fprintf(&b, "\t\tres := %s.%s(vres)\n", e.ServicePkgName, e.Method.ViewedResult.ResultInit.Name)
				b.WriteString("\t\treturn res, nil\n")
			}
			b.WriteString("\t\tres := " + resp.ResultInit.Name + "(")
			for i, arg := range resp.ResultInit.ClientArgs {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(arg.Ref)
			}
			b.WriteString(")\n")
			if resp.TagName != "" && !isViewedResponse(resp) {
				if resp.TagPointer {
					fmt.Fprintf(&b, "\t\ttmp := %q\n\t\tres.%s = &tmp\n", resp.TagValue, resp.TagName)
				} else {
					fmt.Fprintf(&b, "\t\tres.%s = %q\n", resp.TagName, resp.TagValue)
				}
			}
			b.WriteString("\t\treturn res, nil\n")
		case resp.ClientBody != nil:
			b.WriteString("\t\treturn body, nil\n")
		case len(resp.Headers) > 0:
			fmt.Fprintf(&b, "\t\treturn %s, nil\n", resp.Headers[0].VarName)
		case len(resp.Cookies) > 0:
			fmt.Fprintf(&b, "\t\treturn %s, nil\n", resp.Cookies[0].VarName)
		default:
			b.WriteString("\t\treturn nil, nil\n")
		}
	} else {
		b.WriteString("\t\treturn nil, nil\n")
	}
	b.WriteString("\t}\n}\n")
	return b.String()
}

func writeJSONRPCErrorDecodeSwitch(b *strings.Builder, e *httpcodegen.EndpointData) {
	for _, group := range e.Errors {
		if len(group.Errors) == 0 || group.Errors[0].Response == nil {
			continue
		}
		fmt.Fprintf(b, "\t\t\tcase %s:\n", group.StatusCode)
		if len(group.Errors) > 1 {
			writeJSONRPCNamedErrorDecode(b, group, e)
			continue
		}
		writeJSONRPCErrorResponseDecode(b, group.Errors[0].Response, e.ServiceName, e.Method)
		writeResultInitReturn(b, group.Errors[0].Response)
	}
}

func writeJSONRPCNamedErrorDecode(b *strings.Builder, group *httpcodegen.ErrorGroupData, e *httpcodegen.EndpointData) {
	b.WriteString("\t\t\t\tvar jerrData jsonrpc.ErrorData\n")
	b.WriteString("\t\t\t\tif len(jresp.Error.Data) > 0 {\n")
	fmt.Fprintf(b, "\t\t\t\t\tif err := json.Unmarshal(jresp.Error.Data, &jerrData); err != nil {\n\t\t\t\t\t\treturn nil, goahttp.ErrDecodingError(%q, %q, err)\n\t\t\t\t\t}\n", e.ServiceName, e.Method.Name)
	b.WriteString("\t\t\t\t}\n")
	b.WriteString("\t\t\t\tswitch jerrData.Name {\n")
	for _, item := range group.Errors {
		if item.Response == nil {
			continue
		}
		fmt.Fprintf(b, "\t\t\t\tcase %q:\n", item.Name)
		writeJSONRPCErrorResponseDecode(b, item.Response, e.ServiceName, e.Method)
		writeResultInitReturn(b, item.Response)
	}
	fmt.Fprintf(b, "\t\t\t\tdefault:\n\t\t\t\t\treturn nil, goahttp.ErrInvalidResponse(%q, %q, resp.StatusCode, string(jresp.Error.Data))\n", e.ServiceName, e.Method.Name)
	b.WriteString("\t\t\t\t}\n")
}

func writeJSONRPCErrorResponseDecode(b *strings.Builder, data *httpcodegen.ResponseData, serviceName string, method *service.MethodData) {
	b.WriteString("\t\t\t\tresp.Body = io.NopCloser(bytes.NewBuffer(jresp.Error.Data))\n")
	renderSingleResponseDecode(b, data, serviceName, method)
}

func writeResultInitReturn(b *strings.Builder, resp *httpcodegen.ResponseData) {
	switch {
	case resp.ResultInit != nil:
		b.WriteString("\t\t\t\treturn nil, " + resp.ResultInit.Name + "(")
		for i, arg := range resp.ResultInit.ClientArgs {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(arg.Ref)
		}
		b.WriteString(")\n")
	case resp.ClientBody != nil:
		b.WriteString("\t\t\t\treturn nil, body\n")
	default:
		b.WriteString("\t\t\t\treturn nil, nil\n")
	}
}

func renderSingleResponseDecode(b *strings.Builder, data *httpcodegen.ResponseData, serviceName string, method *service.MethodData) {
	if data.ClientBody != nil {
		fmt.Fprintf(b, "\t\t\tvar (\n\t\t\t\tbody %s\n\t\t\t\terr error\n\t\t\t)\n", data.ClientBody.VarName)
		fmt.Fprintf(b, "\t\t\terr = decoder(resp).Decode(&body)\n\t\t\tif err != nil {\n\t\t\t\treturn nil, goahttp.ErrDecodingError(%q, %q, err)\n\t\t\t}\n", serviceName, method.Name)
		if data.ClientBody.ValidateRef != "" {
			fmt.Fprintf(b, "\t\t\t%s\n\t\t\tif err != nil {\n\t\t\t\treturn nil, goahttp.ErrValidationError(%q, %q, err)\n\t\t\t}\n", data.ClientBody.ValidateRef, serviceName, method.Name)
		}
	}
	if len(data.Headers) > 0 {
		b.WriteString("\t\t\tvar (\n")
		for _, header := range data.Headers {
			fmt.Fprintf(b, "\t\t\t\t%s %s\n", header.VarName, header.TypeRef)
		}
		if data.ClientBody == nil && data.MustValidate {
			b.WriteString("\t\t\t\terr error\n")
		}
		b.WriteString("\t\t\t)\n")
		for _, header := range data.Headers {
			writeResponseHeaderDecode(b, header)
			if header.Validate != "" {
				fmt.Fprintf(b, "\t\t\t%s\n", header.Validate)
			}
		}
	}
	if len(data.Cookies) > 0 {
		b.WriteString("\t\t\tvar (\n")
		for _, cookie := range data.Cookies {
			fmt.Fprintf(b, "\t\t\t\t%s %s\n", cookie.VarName, cookie.TypeRef)
			fmt.Fprintf(b, "\t\t\t\t%sRaw string\n", cookie.VarName)
		}
		b.WriteString("\t\t\t\tcookies = resp.Cookies()\n")
		if data.ClientBody == nil && data.MustValidate && len(data.Headers) == 0 {
			b.WriteString("\t\t\t\terr error\n")
		}
		b.WriteString("\t\t\t)\n")
		b.WriteString("\t\t\tfor _, c := range cookies {\n\t\t\t\tswitch c.Name {\n")
		for _, cookie := range data.Cookies {
			fmt.Fprintf(b, "\t\t\t\tcase %q:\n\t\t\t\t\t%sRaw = c.Value\n", cookie.HTTPName, cookie.VarName)
		}
		b.WriteString("\t\t\t\t}\n\t\t\t}\n")
		for _, cookie := range data.Cookies {
			writeResponseCookieDecode(b, cookie)
			if cookie.Validate != "" {
				fmt.Fprintf(b, "\t\t\t%s\n", cookie.Validate)
			}
		}
	}
	if data.MustValidate {
		fmt.Fprintf(b, "\t\t\tif err != nil {\n\t\t\t\treturn nil, goahttp.ErrValidationError(%q, %q, err)\n\t\t\t}\n", serviceName, method.Name)
	}
}

func writeResponseHeaderDecode(b *strings.Builder, h *httpcodegen.HeaderData) {
	switch {
	case h.Type.Name() == "string" || h.Type.Name() == "any":
		fmt.Fprintf(b, "\t\t\t%sRaw := resp.Header.Get(%q)\n", h.VarName, h.CanonicalName)
		if h.Required {
			fmt.Fprintf(b, "\t\t\tif %sRaw == \"\" {\n\t\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%q, \"header\"))\n\t\t\t}\n", h.VarName, h.Name)
			fmt.Fprintf(b, "\t\t\t%s = %s%sRaw\n", h.VarName, stringPointerPrefix(h.Type.Name(), h.Pointer), h.VarName)
		} else {
			fmt.Fprintf(b, "\t\t\tif %sRaw != \"\" {\n\t\t\t\t%s = %s%sRaw\n\t\t\t}", h.VarName, h.VarName, stringPointerPrefix(h.Type.Name(), h.Pointer), h.VarName)
			if h.DefaultValue != nil {
				fmt.Fprintf(b, " else {\n\t\t\t\t%s = %s\n\t\t\t}\n", h.VarName, literalValue(h.Type.Name(), h.DefaultValue))
			} else {
				b.WriteString("\n")
			}
		}
	case h.StringSlice:
		fmt.Fprintf(b, "\t\t\t%s = resp.Header[%q]\n", h.VarName, h.CanonicalName)
		if h.Required {
			fmt.Fprintf(b, "\t\t\tif %s == nil {\n\t\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%q, \"header\"))\n\t\t\t}\n", h.VarName, h.Name)
		}
	case h.Slice:
		fmt.Fprintf(b, "\t\t\t{\n\t\t\t\t%sRaw := resp.Header[%q]\n", h.VarName, h.CanonicalName)
		if h.Required {
			fmt.Fprintf(b, "\t\t\t\tif %sRaw == nil {\n\t\t\t\t\treturn nil, goahttp.ErrValidationError(%q, %q, goa.MissingFieldError(%q, \"header\"))\n\t\t\t\t}\n", h.VarName, "", "", h.Name)
		}
		writeElementSliceConversion(b, h.AttributeData)
		b.WriteString("\t\t\t}\n")
	default:
		fmt.Fprintf(b, "\t\t\t{\n\t\t\t\t%sRaw := resp.Header.Get(%q)\n", h.VarName, h.CanonicalName)
		if h.Required {
			fmt.Fprintf(b, "\t\t\t\tif %sRaw == \"\" {\n\t\t\t\t\treturn nil, goahttp.ErrValidationError(%q, %q, goa.MissingFieldError(%q, \"header\"))\n\t\t\t\t}\n", h.VarName, "", "", h.Name)
		}
		writeQueryTypeConversion(b, h.AttributeData)
		b.WriteString("\t\t\t}\n")
	}
}

func writeResponseCookieDecode(b *strings.Builder, c *httpcodegen.CookieData) {
	if c.Type.Name() == "string" || c.Type.Name() == "any" {
		if c.Required {
			fmt.Fprintf(b, "\t\t\tif %sRaw == \"\" {\n\t\t\t\terr = goa.MergeErrors(err, goa.MissingFieldError(%q, \"cookie\"))\n\t\t\t}\n", c.VarName, c.Name)
			fmt.Fprintf(b, "\t\t\t%s = %s%sRaw\n", c.VarName, stringPointerPrefix(c.Type.Name(), c.Pointer), c.VarName)
		} else {
			fmt.Fprintf(b, "\t\t\tif %sRaw != \"\" {\n\t\t\t\t%s = %s%sRaw\n\t\t\t}\n", c.VarName, c.VarName, stringPointerPrefix(c.Type.Name(), c.Pointer), c.VarName)
		}
		return
	}
	fmt.Fprintf(b, "\t\t\t{\n")
	if c.Required {
		fmt.Fprintf(b, "\t\t\t\tif %sRaw == \"\" {\n\t\t\t\t\treturn nil, goahttp.ErrValidationError(%q, %q, goa.MissingFieldError(%q, \"cookie\"))\n\t\t\t\t}\n", c.VarName, "", "", c.Name)
	}
	writeQueryTypeConversion(b, c.AttributeData)
	b.WriteString("\t\t\t}\n")
}

func writeElementSliceConversion(b *strings.Builder, a *httpcodegen.AttributeData) {
	fmt.Fprintf(b, "\t\t\t\t%s = make(%s, len(%sRaw))\n", a.VarName, a.TypeRef, a.VarName)
	fmt.Fprintf(b, "\t\t\t\tfor i, rv := range %sRaw {\n", a.VarName)
	writeSliceItemConversion(b, a)
	b.WriteString("\t\t\t\t}\n")
}

func writeSliceItemConversion(b *strings.Builder, a *httpcodegen.AttributeData) {
	arr := expr.AsArray(a.Type)
	if arr == nil {
		fmt.Fprintf(b, "\t\t\t\t\t// unsupported non-array type for var %s\n", a.VarName)
		return
	}
	elemName := arr.ElemType.Type.Name()
	switch elemName {
	default:
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = rv\n", a.VarName)
	case "bytes":
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = []byte(rv)\n", a.VarName)
	case "int":
		b.WriteString("\t\t\t\t\tv, err2 := strconv.ParseInt(rv, 10, strconv.IntSize)\n")
		fmt.Fprintf(b, "\t\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"array of integers\")) }\n", a.Name, a.VarName)
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = int(v)\n", a.VarName)
	case "int32":
		b.WriteString("\t\t\t\t\tv, err2 := strconv.ParseInt(rv, 10, 32)\n")
		fmt.Fprintf(b, "\t\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"array of integers\")) }\n", a.Name, a.VarName)
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = int32(v)\n", a.VarName)
	case "int64":
		b.WriteString("\t\t\t\t\tv, err2 := strconv.ParseInt(rv, 10, 64)\n")
		fmt.Fprintf(b, "\t\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"array of integers\")) }\n", a.Name, a.VarName)
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = v\n", a.VarName)
	case "uint":
		b.WriteString("\t\t\t\t\tv, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)\n")
		fmt.Fprintf(b, "\t\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"array of unsigned integers\")) }\n", a.Name, a.VarName)
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = uint(v)\n", a.VarName)
	case "uint32":
		b.WriteString("\t\t\t\t\tv, err2 := strconv.ParseUint(rv, 10, 32)\n")
		fmt.Fprintf(b, "\t\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"array of unsigned integers\")) }\n", a.Name, a.VarName)
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = uint32(v)\n", a.VarName)
	case "uint64":
		b.WriteString("\t\t\t\t\tv, err2 := strconv.ParseUint(rv, 10, 64)\n")
		fmt.Fprintf(b, "\t\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"array of unsigned integers\")) }\n", a.Name, a.VarName)
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = v\n", a.VarName)
	case "float32":
		b.WriteString("\t\t\t\t\tv, err2 := strconv.ParseFloat(rv, 32)\n")
		fmt.Fprintf(b, "\t\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"array of floats\")) }\n", a.Name, a.VarName)
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = float32(v)\n", a.VarName)
	case "float64":
		b.WriteString("\t\t\t\t\tv, err2 := strconv.ParseFloat(rv, 64)\n")
		fmt.Fprintf(b, "\t\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"array of floats\")) }\n", a.Name, a.VarName)
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = v\n", a.VarName)
	case "boolean":
		b.WriteString("\t\t\t\t\tv, err2 := strconv.ParseBool(rv)\n")
		fmt.Fprintf(b, "\t\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"array of booleans\")) }\n", a.Name, a.VarName)
		fmt.Fprintf(b, "\t\t\t\t\t%s[i] = v\n", a.VarName)
	}
}

func writeQueryTypeConversion(b *strings.Builder, a *httpcodegen.AttributeData) {
	switch a.Type.Name() {
	case "bytes":
		fmt.Fprintf(b, "\t\t\t\t%s = []byte(%sRaw)\n", a.VarName, a.VarName)
	case "int":
		b.WriteString("\t\t\t\tv, err2 := strconv.ParseInt(" + a.VarName + "Raw, 10, strconv.IntSize)\n")
		fmt.Fprintf(b, "\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"integer\")) }\n", a.Name, a.VarName)
		assignConverted(b, a, "int", "v")
	case "int32":
		b.WriteString("\t\t\t\tv, err2 := strconv.ParseInt(" + a.VarName + "Raw, 10, 32)\n")
		fmt.Fprintf(b, "\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"integer\")) }\n", a.Name, a.VarName)
		assignConverted(b, a, "int32", "v")
	case "int64":
		b.WriteString("\t\t\t\tv, err2 := strconv.ParseInt(" + a.VarName + "Raw, 10, 64)\n")
		fmt.Fprintf(b, "\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"integer\")) }\n", a.Name, a.VarName)
		assignDirectOrCast(b, a, "v", "int64")
	case "uint":
		b.WriteString("\t\t\t\tv, err2 := strconv.ParseUint(" + a.VarName + "Raw, 10, strconv.IntSize)\n")
		fmt.Fprintf(b, "\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"unsigned integer\")) }\n", a.Name, a.VarName)
		assignConverted(b, a, "uint", "v")
	case "uint32":
		b.WriteString("\t\t\t\tv, err2 := strconv.ParseUint(" + a.VarName + "Raw, 10, 32)\n")
		fmt.Fprintf(b, "\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"unsigned integer\")) }\n", a.Name, a.VarName)
		assignConverted(b, a, "uint32", "v")
	case "uint64":
		b.WriteString("\t\t\t\tv, err2 := strconv.ParseUint(" + a.VarName + "Raw, 10, 64)\n")
		fmt.Fprintf(b, "\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"unsigned integer\")) }\n", a.Name, a.VarName)
		assignDirectOrCast(b, a, "v", "uint64")
	case "float32":
		b.WriteString("\t\t\t\tv, err2 := strconv.ParseFloat(" + a.VarName + "Raw, 32)\n")
		fmt.Fprintf(b, "\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"float\")) }\n", a.Name, a.VarName)
		assignConverted(b, a, "float32", "v")
	case "float64":
		b.WriteString("\t\t\t\tv, err2 := strconv.ParseFloat(" + a.VarName + "Raw, 64)\n")
		fmt.Fprintf(b, "\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"float\")) }\n", a.Name, a.VarName)
		assignDirectOrCast(b, a, "v", "float64")
	case "boolean":
		b.WriteString("\t\t\t\tv, err2 := strconv.ParseBool(" + a.VarName + "Raw)\n")
		fmt.Fprintf(b, "\t\t\t\tif err2 != nil { err = goa.MergeErrors(err, goa.InvalidFieldTypeError(%q, %sRaw, \"boolean\")) }\n", a.Name, a.VarName)
		assignDirectOrCast(b, a, "v", "bool")
	default:
		fmt.Fprintf(b, "\t\t\t\t// unsupported type %s for var %s\n", a.Type.Name(), a.VarName)
	}
}

func assignConverted(b *strings.Builder, a *httpcodegen.AttributeData, baseType, source string) {
	targetType := baseType
	if a.TypeRef != "" {
		targetType = strings.TrimPrefix(a.TypeRef, "*")
	}
	if a.Pointer {
		fmt.Fprintf(b, "\t\t\t\tpv := %s(%s)\n", targetType, source)
		fmt.Fprintf(b, "\t\t\t\t%s = &pv\n", a.VarName)
		return
	}
	fmt.Fprintf(b, "\t\t\t\t%s = %s(%s)\n", a.VarName, targetType, source)
}

func assignDirectOrCast(b *strings.Builder, a *httpcodegen.AttributeData, source, builtin string) {
	if a.TypeRef != "" && a.TypeRef != builtin && a.TypeRef != "*"+builtin {
		if a.Pointer {
			fmt.Fprintf(b, "\t\t\t\t%s = (%s)(&%s)\n", a.VarName, a.TypeRef, source)
		} else {
			fmt.Fprintf(b, "\t\t\t\t%s = (%s)(%s)\n", a.VarName, a.TypeRef, source)
		}
		return
	}
	if a.Pointer {
		fmt.Fprintf(b, "\t\t\t\t%s = &%s\n", a.VarName, source)
	} else {
		fmt.Fprintf(b, "\t\t\t\t%s = %s\n", a.VarName, source)
	}
}

func literalValue(typeName string, v any) string {
	if typeName == "string" {
		return fmt.Sprintf("%q", v)
	}
	return fmt.Sprintf("%#v", v)
}

func stringPointerPrefix(typeName string, pointer bool) string {
	if typeName == "string" && pointer {
		return "&"
	}
	return ""
}

func isViewedResponse(resp *httpcodegen.ResponseData) bool {
	return resp.ViewedResult != nil
}

func viewedResultPrefix(v *service.ViewedResultTypeData) string {
	if v == nil || v.IsCollection {
		return ""
	}
	return "&"
}
