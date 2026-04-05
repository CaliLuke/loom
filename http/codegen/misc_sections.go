package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func requestBuilderSection(endpoint *EndpointData) codegen.Section {
	return codegen.MustRenderSection("request-builder", func() string {
		return renderRequestBuilderSection(endpoint)
	})
}

func renderRequestBuilderSection(endpoint *EndpointData) string {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(endpoint.RequestInit.Description))
	b.Add("\n")
	b.Addf("func (c *%s) %s(ctx context.Context", endpoint.ClientStruct, endpoint.RequestInit.Name)
	for _, arg := range endpoint.RequestInit.ClientArgs {
		b.Addf(", %s %s", arg.VarName, arg.TypeRef)
	}
	b.Add(") (*http.Request, error) {\n")
	b.Add(strings.TrimLeft(endpoint.RequestInit.ClientCode, "\n"))
	if !strings.HasSuffix(endpoint.RequestInit.ClientCode, "\n") {
		b.Add("\n")
	}
	b.Add("}\n")
	return b.String()
}

func transformHelperSection(name string, data *codegen.TransformFunctionData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s builds a value of type %s from a value of type %s.", data.Name, data.ResultTypeRef, data.ParamTypeRef))
		stmt.Add(codegen.Expr("func " + data.Name + "(v " + data.ParamTypeRef + ") " + data.ResultTypeRef + " {\n" + data.Code + "\nreturn res\n}"))
		stmt.Line()
	})
}

func multipartRequestEncoderSection(data *MultipartData) codegen.Section {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(fmt.Sprintf("%s returns an encoder to encode the multipart request for the %q service %q endpoint.", data.InitName, data.ServiceName, data.MethodName)))
	b.Add("\n")
	b.Addf("func %s(encoderFn %s) func(r *http.Request) loomhttp.Encoder {\n", data.InitName, data.FuncName)
	b.Add("\treturn func(r *http.Request) loomhttp.Encoder {\n")
	b.Add("\t\tbody := &bytes.Buffer{}\n")
	b.Add("\t\tmw := multipart.NewWriter(body)\n")
	b.Add("\t\treturn loomhttp.EncodingFunc(func(v any) error {\n")
	b.Addf("\t\t\tp := v.(%s)\n", data.Payload.Ref)
	b.Add("\t\t\tif err := encoderFn(mw, p); err != nil {\n")
	b.Add("\t\t\t\treturn err\n")
	b.Add("\t\t\t}\n")
	b.Add("\t\t\tr.Body = io.NopCloser(body)\n")
	b.Add("\t\t\tr.Header.Set(\"Content-Type\", mw.FormDataContentType())\n")
	b.Add("\t\t\treturn mw.Close()\n")
	b.Add("\t\t})\n")
	b.Add("\t}\n")
	b.Add("}\n")
	return codegen.MustRenderSection("multipart-request-encoder", b.String)
}

func multipartRequestDecoderTypeSection(data *MultipartData) codegen.Section {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(fmt.Sprintf("%s is the type to decode multipart request for the %q service %q endpoint.", data.FuncName, data.ServiceName, data.MethodName)))
	b.Add("\n")
	b.Addf("type %s func(*multipart.Reader, *%s) error\n", data.FuncName, data.Payload.Ref)
	return codegen.MustRenderSection("multipart-request-decoder-type", b.String)
}

func serverSSESections(data *ServiceData) []codegen.Section {
	var sections []codegen.Section
	for _, ed := range data.Endpoints {
		if ed.SSE != nil {
			sections = append(sections, serverSSESection(ed))
		}
	}
	return sections
}

func serverSSESection(ed *EndpointData) codegen.Section {
	var b sourceBuilder
	writeSSETypeSection(&b, ed)
	writeSSESendSection(&b, ed)
	writeSSEInitHeaders(&b, ed)
	writeSSESendWithContextSection(&b, ed)
	writeSSECloseSection(&b, ed)
	return codegen.MustRenderSection("server-sse", b.String)
}

func writeSSETypeSection(b *sourceBuilder, ed *EndpointData) {
	b.Add("\n")
	b.Add(codegen.Comment(fmt.Sprintf("%s implements the %s interface using Server-Sent Events.", ed.SSE.StructName, ed.SSE.Interface)))
	b.Add("\n")
	b.Addf("type %s struct {\n", ed.SSE.StructName)
	b.Add("\t" + codegen.Comment("once ensures the headers are written once.") + "\n")
	b.Add("\tonce sync.Once\n")
	b.Add("\t" + codegen.Comment("w is the HTTP response writer used to send the SSE events.") + "\n")
	b.Add("\tw http.ResponseWriter\n")
	b.Add("\t" + codegen.Comment("r is the HTTP request.") + "\n")
	b.Add("\tr *http.Request\n")
	b.Add("}\n\n")
}

func writeSSESendSection(b *sourceBuilder, ed *EndpointData) {
	b.Add(codegen.Comment(fmt.Sprintf("%s %s", ed.SSE.SendName, ed.SSE.SendDesc)))
	b.Add("\n")
	b.Addf("func (s *%s) %s(v %s) error {\n", ed.SSE.StructName, ed.SSE.SendName, ed.SSE.EventTypeRef)
	b.Addf("\treturn s.%s(context.Background(), v)\n", ed.SSE.SendWithContextName)
	b.Add("}\n\n")
}

func writeSSEInitHeaders(b *sourceBuilder, ed *EndpointData) {
	b.Addf("func (s *%s) initHeaders() {\n", ed.SSE.StructName)
	b.Add("\ts.once.Do(func() {\n")
	b.Add("\t\theader := s.w.Header()\n")
	b.Add("\t\tif header.Get(\"Content-Type\") == \"\" {\n")
	b.Add("\t\t\theader.Set(\"Content-Type\", \"text/event-stream\")\n")
	b.Add("\t\t}\n")
	b.Add("\t\tif header.Get(\"Cache-Control\") == \"\" {\n")
	b.Add("\t\t\theader.Set(\"Cache-Control\", \"no-cache\")\n")
	b.Add("\t\t}\n")
	b.Add("\t\tif header.Get(\"Connection\") == \"\" {\n")
	b.Add("\t\t\theader.Set(\"Connection\", \"keep-alive\")\n")
	b.Add("\t\t}\n")
	b.Add("\t\ts.w.WriteHeader(http.StatusOK)\n")
	b.Add("\t})\n")
	b.Add("}\n\n")
}

func writeSSESendWithContextSection(b *sourceBuilder, ed *EndpointData) {
	b.Add(codegen.Comment(fmt.Sprintf("%s %s", ed.SSE.SendWithContextName, ed.SSE.SendWithContextDesc)))
	b.Add("\n")
	b.Addf("func (s *%s) %s(ctx context.Context, v %s) error {\n", ed.SSE.StructName, ed.SSE.SendWithContextName, ed.SSE.EventTypeRef)
	b.Add("\ts.initHeaders()\n")
	writeSSEResultSetup(b, ed)
	writeSSEPayloadSetup(b, ed)
	writeSSEPayloadEncoding(b)
	writeSSEMessageSetup(b, ed)
	b.Add("\tif err := loomhttp.WriteSSEEvent(s.w, msg); err != nil {\n\t\treturn err\n\t}\n\n")
	b.Add("\treturn http.NewResponseController(s.w).Flush()\n")
	b.Add("}\n\n")
}

func writeSSEResultSetup(b *sourceBuilder, ed *EndpointData) {
	if ed.Method.ViewedResult != nil {
		viewName := ed.Method.ViewedResult.ViewName
		if viewName == "" {
			viewName = "default"
		}
		b.Addf("\tres := %s.%s(v, %q)\n", ed.ServicePkgName, ed.Method.ViewedResult.Init.Name, viewName)
		return
	}
	b.Add("\tres := v\n")
}

func writeSSEPayloadSetup(b *sourceBuilder, ed *EndpointData) {
	b.Add("\n\tvar data string\n\tvar payload any\n")
	if ed.SSE.HasResponseBody {
		b.Addf("\tbody := New%sResponseBody(res)\n", codegen.Goify(ed.Method.Name, true))
		if ed.SSE.DataField != "" {
			b.Addf("\tpayload = body.%s\n", ed.SSE.DataField)
			return
		}
		b.Add("\tpayload = body\n")
		return
	}
	if ed.SSE.DataField != "" {
		b.Addf("\tpayload = res.%s\n", ed.SSE.DataField)
		return
	}
	b.Add("\tpayload = res\n")
}

func writeSSEPayloadEncoding(b *sourceBuilder) {
	b.Add("\tswitch v := payload.(type) {\n")
	b.Add("\tcase nil:\n\t\tdata = \"null\"\n")
	b.Add("\tcase string:\n\t\tdata = v\n")
	b.Add("\tcase []byte:\n\t\tdata = string(v)\n")
	b.Add("\tcase bool:\n\t\tif v {\n\t\t\tdata = \"true\"\n\t\t} else {\n\t\t\tdata = \"false\"\n\t\t}\n")
	for _, t := range []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64"} {
		b.Addf("\tcase %s:\n\t\tdata = fmt.Sprintf(\"%%d\", v)\n", t)
	}
	for _, t := range []string{"float32", "float64"} {
		b.Addf("\tcase %s:\n\t\tdata = fmt.Sprintf(\"%%g\", v)\n", t)
	}
	b.Add("\tdefault:\n")
	b.Add("\t\tbyts, err := json.Marshal(payload)\n")
	b.Add("\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n")
	b.Add("\t\tdata = string(byts)\n")
	b.Add("\t}\n\n")
}

func writeSSEMessageSetup(b *sourceBuilder, ed *EndpointData) {
	b.Add("\tmsg := loomhttp.SSEMessage{Data: data}\n")
	if ed.SSE.IDField != "" {
		b.Addf("\n\tif id := res.%s; id != \"\" {\n\t\tmsg.ID = id\n\t}\n", ed.SSE.IDField)
	}
	if ed.SSE.EventField != "" {
		b.Addf("\tif event := res.%s; event != \"\" {\n\t\tmsg.Type = event\n\t}\n", ed.SSE.EventField)
	}
	if ed.SSE.RetryField != "" {
		b.Addf("\tif retry := res.%s; retry > 0 {\n\t\tmsg.RetryMillis = int64(retry)\n\t}\n", ed.SSE.RetryField)
	}
	b.Add("\n")
}

func writeSSECloseSection(b *sourceBuilder, ed *EndpointData) {
	b.Add(codegen.Comment("Close is a no-op for SSE. We keep the method for compatibility with other stream types."))
	b.Add("\n")
	b.Addf("func (s *%s) Close() error {\n\treturn nil\n}\n", ed.SSE.StructName)
}

func renderPathInitCode(args []*InitArgData, pathParams *expr.Object, pathFormat string) string {
	var b sourceBuilder
	if len(args) > 0 {
		for i, arg := range args {
			typ := (*pathParams)[i].Attribute.Type
			if typ.Name() == "array" {
				b.Addf("\t%sSlice := make([]string, len(%s))\n", arg.VarName, arg.VarName)
				b.Addf("\tfor i, v := range %s {\n", arg.VarName)
				b.Addf("\t\t%sSlice[i] = %s\n", arg.VarName, renderQuerySliceConversion(expr.AsArray(typ).ElemType.Type))
				b.Add("\t}\n")
			}
		}
		b.Addf("\treturn fmt.Sprintf(%q, ", pathFormat)
		for i, arg := range args {
			typ := (*pathParams)[i].Attribute.Type
			if typ.Name() == "array" {
				b.Addf("strings.Join(%sSlice, \",\")", arg.VarName)
			} else {
				b.Add(arg.VarName)
			}
			b.Add(", ")
		}
		b.Add(")\n")
		return b.String()
	}
	return fmt.Sprintf("\treturn %q\n", pathFormat)
}

func renderQuerySliceConversion(dt expr.DataType) string {
	switch dt.Name() {
	case "string":
		return "url.QueryEscape(v)"
	case "int", "int32":
		return "strconv.FormatInt(int64(v), 10)"
	case "int64":
		return "strconv.FormatInt(v, 10)"
	case "uint", "uint32":
		return "strconv.FormatUint(uint64(v), 10)"
	case "uint64":
		return "strconv.FormatUint(v, 10)"
	case "float32":
		return "strconv.FormatFloat(float64(v), 'f', -1, 32)"
	case "float64":
		return "strconv.FormatFloat(v, 'f', -1, 64)"
	case "boolean":
		return "strconv.FormatBool(v)"
	case "bytes":
		return "url.QueryEscape(string(v))"
	default:
		return "url.QueryEscape(fmt.Sprintf(\"%v\", v))"
	}
}

func renderRequestInitCode(payloadRef string, hasFields bool, serviceName, endpointName string, args []*InitArgData, pathInit *InitData, verb string, isWebSocket bool, requestStruct string) string {
	var b sourceBuilder
	renderRequestInitVars(&b, args, requestStruct)
	renderRequestPayloadSetup(&b, payloadRef, hasFields, serviceName, endpointName, args, requestStruct)
	renderRequestURLSetup(&b, pathInit, args, isWebSocket)
	renderRequestCreation(&b, serviceName, endpointName, requestStruct, verb)
	renderRequestContextBinding(&b)
	renderRequestReturn(&b)
	return b.String()
}

func renderRequestPayloadSetup(b *sourceBuilder, payloadRef string, hasFields bool, serviceName, endpointName string, args []*InitArgData, requestStruct string) {
	if payloadRef != "" && len(args) > 0 {
		renderPayloadExtraction(b, payloadRef, hasFields, serviceName, endpointName, args, requestStruct)
		return
	}
	if requestStruct == "" {
		return
	}
	b.Addf("\trd, ok := v.(*%s)\n", requestStruct)
	ifTypeErr(b, serviceName, endpointName, requestStruct)
	b.Add("\tbody = rd.Body\n")
}

func renderRequestURLSetup(b *sourceBuilder, pathInit *InitData, args []*InitArgData, isWebSocket bool) {
	renderRequestScheme(b, isWebSocket)
	renderRequestURLPrefix(b, isWebSocket)
	b.Addf("%s(", pathInit.Name)
	for _, arg := range args {
		b.Addf("%s, ", arg.Ref)
	}
	b.Add(")}\n")
}

func renderRequestScheme(b *sourceBuilder, isWebSocket bool) {
	if !isWebSocket {
		return
	}
	b.Add("\tscheme := c.scheme\n")
	b.Add("\tswitch c.scheme {\n")
	b.Add("\tcase \"http\":\n\t\tscheme = \"ws\"\n")
	b.Add("\tcase \"https\":\n\t\tscheme = \"wss\"\n")
	b.Add("\t}\n")
}

func renderRequestURLPrefix(b *sourceBuilder, isWebSocket bool) {
	if isWebSocket {
		b.Add("\tu := &url.URL{Scheme: scheme, Host: c.host, Path: ")
		return
	}
	b.Add("\tu := &url.URL{Scheme: c.scheme, Host: c.host, Path: ")
}

func renderRequestCreation(b *sourceBuilder, serviceName, endpointName, requestStruct, verb string) {
	bodyRef := "nil"
	if requestStruct != "" {
		bodyRef = "body"
	}
	b.Addf("\treq, err := http.NewRequest(%q, u.String(), %s)\n", verb, bodyRef)
	b.Add("\tif err != nil {\n")
	b.Addf("\t\treturn nil, loomhttp.ErrInvalidURL(%q, %q, u.String(), err)\n", serviceName, endpointName)
	b.Add("\t}\n")
}

func renderRequestContextBinding(b *sourceBuilder) {
	b.Add("\tif ctx != nil {\n\t\treq = req.WithContext(ctx)\n\t}\n\n")
}

func renderRequestReturn(b *sourceBuilder) {
	b.Add("\treturn req, nil\n")
}

func renderRequestInitVars(b *sourceBuilder, args []*InitArgData, requestStruct string) {
	if len(args) == 0 && requestStruct == "" {
		return
	}
	b.Add("\tvar (\n")
	for _, arg := range args {
		b.Addf("\t\t%s %s\n", arg.VarName, arg.TypeRef)
	}
	if requestStruct != "" {
		b.Add("\t\tbody io.Reader\n")
	}
	b.Add("\t)\n")
}

func renderPayloadExtraction(b *sourceBuilder, payloadRef string, hasFields bool, serviceName, endpointName string, args []*InitArgData, requestStruct string) {
	b.Add("\t{\n")
	if requestStruct != "" {
		b.Addf("\t\trd, ok := v.(*%s)\n", requestStruct)
		ifTypeErr(b, serviceName, endpointName, requestStruct)
		b.Add("\t\tp := rd.Payload\n")
		b.Add("\t\tbody = rd.Body\n")
	} else {
		b.Addf("\t\tp, ok := v.(%s)\n", payloadRef)
		ifTypeErr(b, serviceName, endpointName, payloadRef)
	}
	for _, arg := range args {
		renderPayloadAssignment(b, hasFields, arg)
	}
	b.Add("\t}\n")
}

func renderPayloadAssignment(b *sourceBuilder, hasFields bool, arg *InitArgData) {
	if arg.Pointer {
		if hasFields {
			b.Addf("\t\tif p.%s != nil {\n", arg.FieldName)
		} else {
			b.Add("\t\tif p != nil {\n")
		}
	}
	if arg.IsAliased {
		renderAliasedPayloadAssignment(b, hasFields, arg)
	} else {
		renderDirectPayloadAssignment(b, hasFields, arg)
	}
	if arg.Pointer {
		b.Add("\t\t}\n")
	}
}

func renderAliasedPayloadAssignment(b *sourceBuilder, hasFields bool, arg *InitArgData) {
	b.Addf("\t\t\t%s = %s(", arg.VarName, arg.ServiceTypeRef)
	if arg.Pointer {
		b.Add("*")
	}
	if hasFields {
		b.Addf("p.%s)\n", arg.FieldName)
		return
	}
	b.Add("p)\n")
}

func renderDirectPayloadAssignment(b *sourceBuilder, hasFields bool, arg *InitArgData) {
	b.Addf("\t\t\t%s = ", arg.VarName)
	if arg.Pointer {
		b.Add("*")
	}
	if hasFields {
		b.Addf("p.%s\n", arg.FieldName)
		return
	}
	b.Add("p\n")
}

func ifTypeErr(b *sourceBuilder, serviceName, endpointName, typeRef string) {
	b.Add("\t\tif !ok {\n")
	b.Addf("\t\t\treturn nil, loomhttp.ErrInvalidType(%q, %q, %q, v)\n", serviceName, endpointName, typeRef)
	b.Add("\t\t}\n")
}
