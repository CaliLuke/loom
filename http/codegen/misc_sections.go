package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func requestBuilderSection(endpoint *EndpointData) codegen.Section {
	return codegen.NewRawSection("request-builder", renderRequestBuilderSection(endpoint))
}

func renderRequestBuilderSection(endpoint *EndpointData) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(endpoint.RequestInit.Description))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (c *%s) %s(ctx context.Context", endpoint.ClientStruct, endpoint.RequestInit.Name)
	for _, arg := range endpoint.RequestInit.ClientArgs {
		fmt.Fprintf(&b, ", %s %s", arg.VarName, arg.TypeRef)
	}
	b.WriteString(") (*http.Request, error) {\n")
	b.WriteString(strings.TrimLeft(endpoint.RequestInit.ClientCode, "\n"))
	if !strings.HasSuffix(endpoint.RequestInit.ClientCode, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("}\n")
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
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s returns an encoder to encode the multipart request for the %q service %q endpoint.", data.InitName, data.ServiceName, data.MethodName)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(encoderFn %s) func(r *http.Request) loomhttp.Encoder {\n", data.InitName, data.FuncName)
	b.WriteString("\treturn func(r *http.Request) loomhttp.Encoder {\n")
	b.WriteString("\t\tbody := &bytes.Buffer{}\n")
	b.WriteString("\t\tmw := multipart.NewWriter(body)\n")
	b.WriteString("\t\treturn loomhttp.EncodingFunc(func(v any) error {\n")
	fmt.Fprintf(&b, "\t\t\tp := v.(%s)\n", data.Payload.Ref)
	b.WriteString("\t\t\tif err := encoderFn(mw, p); err != nil {\n")
	b.WriteString("\t\t\t\treturn err\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\tr.Body = io.NopCloser(body)\n")
	b.WriteString("\t\t\tr.Header.Set(\"Content-Type\", mw.FormDataContentType())\n")
	b.WriteString("\t\t\treturn mw.Close()\n")
	b.WriteString("\t\t})\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("multipart-request-encoder", b.String())
}

func multipartRequestDecoderTypeSection(data *MultipartData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s is the type to decode multipart request for the %q service %q endpoint.", data.FuncName, data.ServiceName, data.MethodName)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s func(*multipart.Reader, *%s) error\n", data.FuncName, data.Payload.Ref)
	return codegen.NewRawSection("multipart-request-decoder-type", b.String())
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
	var b strings.Builder
	writeSSETypeSection(&b, ed)
	writeSSESendSection(&b, ed)
	writeSSEInitHeaders(&b, ed)
	writeSSESendWithContextSection(&b, ed)
	writeSSECloseSection(&b, ed)
	return codegen.NewRawSection("server-sse", b.String())
}

func writeSSETypeSection(b *strings.Builder, ed *EndpointData) {
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s implements the %s interface using Server-Sent Events.", ed.SSE.StructName, ed.SSE.Interface)))
	b.WriteString("\n")
	fmt.Fprintf(b, "type %s struct {\n", ed.SSE.StructName)
	b.WriteString("\t" + codegen.Comment("once ensures the headers are written once.") + "\n")
	b.WriteString("\tonce sync.Once\n")
	b.WriteString("\t" + codegen.Comment("w is the HTTP response writer used to send the SSE events.") + "\n")
	b.WriteString("\tw http.ResponseWriter\n")
	b.WriteString("\t" + codegen.Comment("r is the HTTP request.") + "\n")
	b.WriteString("\tr *http.Request\n")
	b.WriteString("}\n\n")
}

func writeSSESendSection(b *strings.Builder, ed *EndpointData) {
	b.WriteString(codegen.Comment(fmt.Sprintf("%s %s", ed.SSE.SendName, ed.SSE.SendDesc)))
	b.WriteString("\n")
	fmt.Fprintf(b, "func (s *%s) %s(v %s) error {\n", ed.SSE.StructName, ed.SSE.SendName, ed.SSE.EventTypeRef)
	fmt.Fprintf(b, "\treturn s.%s(context.Background(), v)\n", ed.SSE.SendWithContextName)
	b.WriteString("}\n\n")
}

func writeSSEInitHeaders(b *strings.Builder, ed *EndpointData) {
	fmt.Fprintf(b, "func (s *%s) initHeaders() {\n", ed.SSE.StructName)
	b.WriteString("\ts.once.Do(func() {\n")
	b.WriteString("\t\theader := s.w.Header()\n")
	b.WriteString("\t\tif header.Get(\"Content-Type\") == \"\" {\n")
	b.WriteString("\t\t\theader.Set(\"Content-Type\", \"text/event-stream\")\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif header.Get(\"Cache-Control\") == \"\" {\n")
	b.WriteString("\t\t\theader.Set(\"Cache-Control\", \"no-cache\")\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif header.Get(\"Connection\") == \"\" {\n")
	b.WriteString("\t\t\theader.Set(\"Connection\", \"keep-alive\")\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\ts.w.WriteHeader(http.StatusOK)\n")
	b.WriteString("\t})\n")
	b.WriteString("}\n\n")
}

func writeSSESendWithContextSection(b *strings.Builder, ed *EndpointData) {
	b.WriteString(codegen.Comment(fmt.Sprintf("%s %s", ed.SSE.SendWithContextName, ed.SSE.SendWithContextDesc)))
	b.WriteString("\n")
	fmt.Fprintf(b, "func (s *%s) %s(ctx context.Context, v %s) error {\n", ed.SSE.StructName, ed.SSE.SendWithContextName, ed.SSE.EventTypeRef)
	b.WriteString("\ts.initHeaders()\n")
	writeSSEResultSetup(b, ed)
	writeSSEPayloadSetup(b, ed)
	writeSSEPayloadEncoding(b)
	writeSSEMessageSetup(b, ed)
	b.WriteString("\tif err := loomhttp.WriteSSEEvent(s.w, msg); err != nil {\n\t\treturn err\n\t}\n\n")
	b.WriteString("\treturn http.NewResponseController(s.w).Flush()\n")
	b.WriteString("}\n\n")
}

func writeSSEResultSetup(b *strings.Builder, ed *EndpointData) {
	if ed.Method.ViewedResult != nil {
		viewName := ed.Method.ViewedResult.ViewName
		if viewName == "" {
			viewName = "default"
		}
		fmt.Fprintf(b, "\tres := %s.%s(v, %q)\n", ed.ServicePkgName, ed.Method.ViewedResult.Init.Name, viewName)
		return
	}
	b.WriteString("\tres := v\n")
}

func writeSSEPayloadSetup(b *strings.Builder, ed *EndpointData) {
	b.WriteString("\n\tvar data string\n\tvar payload any\n")
	if ed.SSE.HasResponseBody {
		fmt.Fprintf(b, "\tbody := New%sResponseBody(res)\n", codegen.Goify(ed.Method.Name, true))
		if ed.SSE.DataField != "" {
			fmt.Fprintf(b, "\tpayload = body.%s\n", ed.SSE.DataField)
			return
		}
		b.WriteString("\tpayload = body\n")
		return
	}
	if ed.SSE.DataField != "" {
		fmt.Fprintf(b, "\tpayload = res.%s\n", ed.SSE.DataField)
		return
	}
	b.WriteString("\tpayload = res\n")
}

func writeSSEPayloadEncoding(b *strings.Builder) {
	b.WriteString("\tswitch v := payload.(type) {\n")
	b.WriteString("\tcase nil:\n\t\tdata = \"null\"\n")
	b.WriteString("\tcase string:\n\t\tdata = v\n")
	b.WriteString("\tcase []byte:\n\t\tdata = string(v)\n")
	b.WriteString("\tcase bool:\n\t\tif v {\n\t\t\tdata = \"true\"\n\t\t} else {\n\t\t\tdata = \"false\"\n\t\t}\n")
	for _, t := range []string{"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64"} {
		fmt.Fprintf(b, "\tcase %s:\n\t\tdata = fmt.Sprintf(\"%%d\", v)\n", t)
	}
	for _, t := range []string{"float32", "float64"} {
		fmt.Fprintf(b, "\tcase %s:\n\t\tdata = fmt.Sprintf(\"%%g\", v)\n", t)
	}
	b.WriteString("\tdefault:\n")
	b.WriteString("\t\tbyts, err := json.Marshal(payload)\n")
	b.WriteString("\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n")
	b.WriteString("\t\tdata = string(byts)\n")
	b.WriteString("\t}\n\n")
}

func writeSSEMessageSetup(b *strings.Builder, ed *EndpointData) {
	b.WriteString("\tmsg := loomhttp.SSEMessage{Data: data}\n")
	if ed.SSE.IDField != "" {
		fmt.Fprintf(b, "\n\tif id := res.%s; id != \"\" {\n\t\tmsg.ID = id\n\t}\n", ed.SSE.IDField)
	}
	if ed.SSE.EventField != "" {
		fmt.Fprintf(b, "\tif event := res.%s; event != \"\" {\n\t\tmsg.Type = event\n\t}\n", ed.SSE.EventField)
	}
	if ed.SSE.RetryField != "" {
		fmt.Fprintf(b, "\tif retry := res.%s; retry > 0 {\n\t\tmsg.RetryMillis = int64(retry)\n\t}\n", ed.SSE.RetryField)
	}
	b.WriteString("\n")
}

func writeSSECloseSection(b *strings.Builder, ed *EndpointData) {
	b.WriteString(codegen.Comment("Close is a no-op for SSE. We keep the method for compatibility with other stream types."))
	b.WriteString("\n")
	fmt.Fprintf(b, "func (s *%s) Close() error {\n\treturn nil\n}\n", ed.SSE.StructName)
}

func renderPathInitCode(args []*InitArgData, pathParams *expr.Object, pathFormat string) string {
	var b strings.Builder
	if len(args) > 0 {
		for i, arg := range args {
			typ := (*pathParams)[i].Attribute.Type
			if typ.Name() == "array" {
				fmt.Fprintf(&b, "\t%sSlice := make([]string, len(%s))\n", arg.VarName, arg.VarName)
				fmt.Fprintf(&b, "\tfor i, v := range %s {\n", arg.VarName)
				fmt.Fprintf(&b, "\t\t%sSlice[i] = %s\n", arg.VarName, renderQuerySliceConversion(expr.AsArray(typ).ElemType.Type))
				b.WriteString("\t}\n")
			}
		}
		fmt.Fprintf(&b, "\treturn fmt.Sprintf(%q, ", pathFormat)
		for i, arg := range args {
			typ := (*pathParams)[i].Attribute.Type
			if typ.Name() == "array" {
				fmt.Fprintf(&b, "strings.Join(%sSlice, \",\")", arg.VarName)
			} else {
				b.WriteString(arg.VarName)
			}
			b.WriteString(", ")
		}
		b.WriteString(")\n")
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
	var b strings.Builder
	renderRequestInitVars(&b, args, requestStruct)
	renderRequestPayloadSetup(&b, payloadRef, hasFields, serviceName, endpointName, args, requestStruct)
	renderRequestURLSetup(&b, pathInit, args, isWebSocket)
	renderRequestCreation(&b, serviceName, endpointName, requestStruct, verb)
	renderRequestContextBinding(&b)
	renderRequestReturn(&b)
	return b.String()
}

func renderRequestPayloadSetup(b *strings.Builder, payloadRef string, hasFields bool, serviceName, endpointName string, args []*InitArgData, requestStruct string) {
	if payloadRef != "" && len(args) > 0 {
		renderPayloadExtraction(b, payloadRef, hasFields, serviceName, endpointName, args, requestStruct)
		return
	}
	if requestStruct == "" {
		return
	}
	fmt.Fprintf(b, "\trd, ok := v.(*%s)\n", requestStruct)
	ifTypeErr(b, serviceName, endpointName, requestStruct)
	b.WriteString("\tbody = rd.Body\n")
}

func renderRequestURLSetup(b *strings.Builder, pathInit *InitData, args []*InitArgData, isWebSocket bool) {
	renderRequestScheme(b, isWebSocket)
	renderRequestURLPrefix(b, isWebSocket)
	fmt.Fprintf(b, "%s(", pathInit.Name)
	for _, arg := range args {
		fmt.Fprintf(b, "%s, ", arg.Ref)
	}
	b.WriteString(")}\n")
}

func renderRequestScheme(b *strings.Builder, isWebSocket bool) {
	if !isWebSocket {
		return
	}
	b.WriteString("\tscheme := c.scheme\n")
	b.WriteString("\tswitch c.scheme {\n")
	b.WriteString("\tcase \"http\":\n\t\tscheme = \"ws\"\n")
	b.WriteString("\tcase \"https\":\n\t\tscheme = \"wss\"\n")
	b.WriteString("\t}\n")
}

func renderRequestURLPrefix(b *strings.Builder, isWebSocket bool) {
	if isWebSocket {
		b.WriteString("\tu := &url.URL{Scheme: scheme, Host: c.host, Path: ")
		return
	}
	b.WriteString("\tu := &url.URL{Scheme: c.scheme, Host: c.host, Path: ")
}

func renderRequestCreation(b *strings.Builder, serviceName, endpointName, requestStruct, verb string) {
	bodyRef := "nil"
	if requestStruct != "" {
		bodyRef = "body"
	}
	fmt.Fprintf(b, "\treq, err := http.NewRequest(%q, u.String(), %s)\n", verb, bodyRef)
	b.WriteString("\tif err != nil {\n")
	fmt.Fprintf(b, "\t\treturn nil, loomhttp.ErrInvalidURL(%q, %q, u.String(), err)\n", serviceName, endpointName)
	b.WriteString("\t}\n")
}

func renderRequestContextBinding(b *strings.Builder) {
	b.WriteString("\tif ctx != nil {\n\t\treq = req.WithContext(ctx)\n\t}\n\n")
}

func renderRequestReturn(b *strings.Builder) {
	b.WriteString("\treturn req, nil\n")
}

func renderRequestInitVars(b *strings.Builder, args []*InitArgData, requestStruct string) {
	if len(args) == 0 && requestStruct == "" {
		return
	}
	b.WriteString("\tvar (\n")
	for _, arg := range args {
		fmt.Fprintf(b, "\t\t%s %s\n", arg.VarName, arg.TypeRef)
	}
	if requestStruct != "" {
		b.WriteString("\t\tbody io.Reader\n")
	}
	b.WriteString("\t)\n")
}

func renderPayloadExtraction(b *strings.Builder, payloadRef string, hasFields bool, serviceName, endpointName string, args []*InitArgData, requestStruct string) {
	b.WriteString("\t{\n")
	if requestStruct != "" {
		fmt.Fprintf(b, "\t\trd, ok := v.(*%s)\n", requestStruct)
		ifTypeErr(b, serviceName, endpointName, requestStruct)
		b.WriteString("\t\tp := rd.Payload\n")
		b.WriteString("\t\tbody = rd.Body\n")
	} else {
		fmt.Fprintf(b, "\t\tp, ok := v.(%s)\n", payloadRef)
		ifTypeErr(b, serviceName, endpointName, payloadRef)
	}
	for _, arg := range args {
		renderPayloadAssignment(b, hasFields, arg)
	}
	b.WriteString("\t}\n")
}

func renderPayloadAssignment(b *strings.Builder, hasFields bool, arg *InitArgData) {
	if arg.Pointer {
		if hasFields {
			fmt.Fprintf(b, "\t\tif p.%s != nil {\n", arg.FieldName)
		} else {
			b.WriteString("\t\tif p != nil {\n")
		}
	}
	if arg.IsAliased {
		renderAliasedPayloadAssignment(b, hasFields, arg)
	} else {
		renderDirectPayloadAssignment(b, hasFields, arg)
	}
	if arg.Pointer {
		b.WriteString("\t\t}\n")
	}
}

func renderAliasedPayloadAssignment(b *strings.Builder, hasFields bool, arg *InitArgData) {
	fmt.Fprintf(b, "\t\t\t%s = %s(", arg.VarName, arg.ServiceTypeRef)
	if arg.Pointer {
		b.WriteString("*")
	}
	if hasFields {
		fmt.Fprintf(b, "p.%s)\n", arg.FieldName)
		return
	}
	b.WriteString("p)\n")
}

func renderDirectPayloadAssignment(b *strings.Builder, hasFields bool, arg *InitArgData) {
	fmt.Fprintf(b, "\t\t\t%s = ", arg.VarName)
	if arg.Pointer {
		b.WriteString("*")
	}
	if hasFields {
		fmt.Fprintf(b, "p.%s\n", arg.FieldName)
		return
	}
	b.WriteString("p\n")
}

func ifTypeErr(b *strings.Builder, serviceName, endpointName, typeRef string) {
	b.WriteString("\t\tif !ok {\n")
	fmt.Fprintf(b, "\t\t\treturn nil, loomhttp.ErrInvalidType(%q, %q, %q, v)\n", serviceName, endpointName, typeRef)
	b.WriteString("\t\t}\n")
}
