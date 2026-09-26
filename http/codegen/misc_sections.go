package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func requestBuilderSection(endpoint *EndpointData) codegen.Section {
	return codegen.NewJenniferSection("request-builder", func(stmt *jen.Statement) {
		codegen.Doc(stmt, endpoint.RequestInit.Description)
		stmt.Func().
			Params(jen.Id("c").Op("*").Id(endpoint.ClientStruct)).
			Id(endpoint.RequestInit.Name).
			ParamsFunc(func(group *jen.Group) {
				group.Id("ctx").Add(codegen.TypeRef("context.Context"))
				for _, arg := range endpoint.RequestInit.ClientArgs {
					group.Id(arg.VarName).Add(codegen.TypeRef(arg.TypeRef))
				}
			}).
			Params(jen.Op("*").Qual("net/http", "Request"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				appendRawBlock(group, endpoint.RequestInit.ClientCode)
			})
		stmt.Line()
	})
}

func transformHelperSection(name string, data *codegen.TransformFunctionData) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s builds a value of type %s from a value of type %s.", data.Name, data.ResultTypeRef, data.ParamTypeRef))
		stmt.Func().
			Id(data.Name).
			Params(jen.Id("v").Add(codegen.TypeRef(data.ParamTypeRef))).
			Add(codegen.TypeRef(data.ResultTypeRef)).
			BlockFunc(func(group *jen.Group) {
				appendRawBlock(group, data.Code)
				group.Line()
				group.Return(jen.Id("res"))
			})
		stmt.Line()
	})
}

func appendRawBlock(group *jen.Group, code string) {
	if trimmed := strings.TrimSpace(code); trimmed != "" {
		group.Add(codegen.Expr(trimmed))
	}
}

func multipartRequestEncoderSection(data *MultipartData) codegen.Section {
	return codegen.NewJenniferSection("multipart-request-encoder", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s returns an encoder to encode the multipart request for the %q service %q endpoint.", data.InitName, data.ServiceName, data.MethodName))
		stmt.Func().
			Id(data.InitName).
			Params(jen.Id("encoderFn").Id(data.FuncName)).
			Func().
			Params(jen.Id("r").Op("*").Qual("net/http", "Request")).
			Add(codegen.TypeRef("loomhttp.Encoder")).
			Block(
				jen.Return(
					jen.Func().
						Params(jen.Id("r").Op("*").Qual("net/http", "Request")).
						Add(codegen.TypeRef("loomhttp.Encoder")).
						Block(
							jen.Id("body").Op(":=").Op("&").Qual("bytes", "Buffer").Values(),
							jen.Id("mw").Op(":=").Qual("mime/multipart", "NewWriter").Call(jen.Id("body")),
							jen.Return(
								codegen.Expr("loomhttp.EncodingFunc").Call(
									jen.Func().
										Params(jen.Id("v").Any()).
										Error().
										BlockFunc(func(group *jen.Group) {
											addRawWebSocketGroup(group, "p := v.("+data.Payload.Ref+")\nif err := encoderFn(mw, p); err != nil {\n\treturn err\n}\nr.Body = io.NopCloser(body)\nr.Header.Set(\"Content-Type\", mw.FormDataContentType())\nreturn mw.Close()")
										}),
								),
							),
						),
				),
			)
		stmt.Line()
	})
}

func multipartRequestDecoderTypeSection(data *MultipartData) codegen.Section {
	return codegen.NewJenniferSection("multipart-request-decoder-type", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s is the type to decode multipart request for the %q service %q endpoint.", data.FuncName, data.ServiceName, data.MethodName))
		stmt.Type().Id(data.FuncName).Func().Params(
			jen.Op("*").Qual("mime/multipart", "Reader"),
			jen.Op("*").Add(codegen.TypeRef(data.Payload.Ref)),
		).Error()
		stmt.Line()
	})
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
	return codegen.NewJenniferSection("server-sse", func(stmt *jen.Statement) {
		addServerSSESection(stmt, ed)
	})
}

func writeSSEResultSetup(b *sourceBuilder, ed *EndpointData) {
	if ed.Method.ViewedResult != nil {
		if len(ed.SSE.Projections) > 0 {
			b.Add("\tvar view string\n")
			b.Addf("\tswitch v.%s {\n", ed.SSE.EventField)
			for _, projection := range ed.SSE.Projections {
				b.Addf("\tcase %q:\n\t\tview = %q\n", projection.EventType, projection.View)
			}
			b.Addf("\tdefault:\n\t\treturn fmt.Errorf(\"invalid SSE projection discriminator %%q\", v.%s)\n\t}\n", ed.SSE.EventField)
			b.Addf("\tres, err := %s.%s(v, view)\n", ed.ServicePkgName, ed.Method.ViewedResult.Init.Name)
			b.Add("\tif err != nil {\n\t\treturn err\n\t}\n")
			return
		}
		viewName := ed.Method.ViewedResult.ViewName
		if viewName == "" {
			b.Addf("\tres, err := %s.%s(v, s.view)\n", ed.ServicePkgName, ed.Method.ViewedResult.Init.Name)
			b.Add("\tif err != nil {\n\t\treturn err\n\t}\n")
			return
		}
		b.Addf("\tres, err := %s.%s(v, %q)\n", ed.ServicePkgName, ed.Method.ViewedResult.Init.Name, viewName)
		b.Add("\tif err != nil {\n\t\treturn err\n\t}\n")
		return
	}
	b.Add("\tres := v\n")
}

func writeSSEPayloadSetup(b *sourceBuilder, ed *EndpointData) {
	b.Add("\n\tvar payload any\n")
	if len(ed.SSE.Projections) > 0 {
		response := sseProjectionResponse(ed)
		b.Add("\tswitch view {\n")
		for _, projection := range ed.SSE.Projections {
			b.Addf("\tcase %q:\n", projection.View)
			body := viewedServerBody(response.ServerBody, projection.View)
			writeServerBodyInitCall(b, body, "\t\tpayload = ")
		}
		b.Add("\t}\n")
		return
	}
	if ed.SSE.ResponseBody != nil {
		writeServerBodyInitCall(b, ed.SSE.ResponseBody, "\tbody := ")
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

func sseProjectionResponse(ed *EndpointData) *ResponseData {
	for _, response := range ed.Result.Responses {
		if len(response.ServerBody) > 0 {
			return response
		}
	}
	panic("SSE projections require a generated response body")
}

// writeSSEPayloadEncoding encodes the event data. Whole-event primitive and
// collection results are always JSON so that strings and bytes survive SSE
// framing exactly; struct results already encode as JSON, and field-level data
// mappings keep their raw-text encoding.
func writeSSEPayloadEncoding(b *sourceBuilder, ed *EndpointData) {
	if ed.SSE.DataField == "" && !ed.SSE.EventIsStruct {
		b.Add("\tdata, err := loomhttp.EncodeSSEJSONData(payload)\n")
		b.Add("\tif err != nil {\n\t\treturn err\n\t}\n\n")
		return
	}
	b.Add("\tdata, err := loomhttp.EncodeSSEData(payload)\n")
	b.Add("\tif err != nil {\n\t\treturn err\n\t}\n\n")
}

func writeSSEMessageSetup(b *sourceBuilder, ed *EndpointData) {
	b.Add("\tmsg := loomhttp.SSEMessage{Data: data}\n")
	resultVar := "res"
	if ed.Method.ViewedResult != nil {
		resultVar = "v"
	}
	if ed.SSE.IDField != "" {
		b.Add("\n")
		writeSSEMessageField(b, "id", resultVar+"."+ed.SSE.IDField, ed.SSE.IDPointer, `%s != ""`, "msg.ID = %s")
	}
	if ed.SSE.EventField != "" {
		writeSSEMessageField(b, "event", resultVar+"."+ed.SSE.EventField, ed.SSE.EventPointer, `%s != ""`, "msg.Type = %s")
	}
	if ed.SSE.RetryField != "" {
		writeSSEMessageField(b, "retry", resultVar+"."+ed.SSE.RetryField, ed.SSE.RetryPointer, "%s > 0", "msg.RetryMillis = int64(%s)")
	}
	b.Add("\n")
}

// writeSSEMessageField copies the result field source into the SSE message
// when it holds a value that the SSE wire format can carry. An empty id or
// event and a non-positive retry are omitted. A pointer field is also omitted
// when nil and dereferenced otherwise. cond and assign are format strings
// applied to the field value expression.
func writeSSEMessageField(b *sourceBuilder, local, source string, pointer bool, cond, assign string) {
	value := local
	check := fmt.Sprintf(cond, value)
	if pointer {
		value = "*" + local
		check = local + " != nil && " + fmt.Sprintf(cond, value)
	}
	b.Addf("\tif %s := %s; %s {\n\t\t%s\n\t}\n", local, source, check, fmt.Sprintf(assign, value))
}

func addServerSSESection(stmt *jen.Statement, ed *EndpointData) {
	stmt.Line()
	addServerSSEType(stmt, ed)
	stmt.Line()
	codegen.Doc(stmt, ed.SSE.SendDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ed.SSE.StructName)).
		Id(ed.SSE.SendName).
		Params(jen.Id("v").Add(codegen.TypeRef(ed.SSE.EventTypeRef))).
		Error().
		Block(
			jen.Return(jen.Id("s").Dot(ed.SSE.SendWithContextName).Call(jen.Id("s").Dot("writer").Dot("Context").Call(), jen.Id("v"))),
		)
	stmt.Line()
	stmt.Line()
	addServerSSESetViewMethod(stmt, ed)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ed.SSE.StructName)).
		Id("started").
		Params().
		Bool().
		Block(jen.Return(jen.Id("s").Dot("writer").Dot("Started").Call()))
	stmt.Line()
	codegen.Doc(stmt, "Open commits and flushes the successful SSE response before the first event.")
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ed.SSE.StructName)).
		Id("Open").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Error().
		Block(jen.Return(jen.Id("s").Dot("writer").Dot("Open").Call(jen.Id("ctx"))))
	stmt.Line()
	codegen.Doc(stmt, "SendComment writes and flushes an SSE heartbeat comment.")
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ed.SSE.StructName)).
		Id("SendComment").
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("text").String()).
		Error().
		Block(jen.Return(jen.Id("s").Dot("writer").Dot("SendComment").Call(jen.Id("ctx"), jen.Id("text"))))
	stmt.Line()
	codegen.Doc(stmt, ed.SSE.SendWithContextDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ed.SSE.StructName)).
		Id(ed.SSE.SendWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Add(codegen.TypeRef(ed.SSE.EventTypeRef))).
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawWebSocketGroup(group, renderServerSSESendWithContextBody(ed))
		})
	stmt.Line()
	codegen.Doc(stmt, "Close prevents later SSE control or event writes.")
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ed.SSE.StructName)).
		Id("Close").
		Params().
		Error().
		Block(jen.Return(jen.Id("s").Dot("writer").Dot("Close").Call()))
	stmt.Line()
}

func addServerSSEType(stmt *jen.Statement, ed *EndpointData) {
	codegen.Doc(stmt, fmt.Sprintf("%s implements the %s interface using Server-Sent Events.", ed.SSE.StructName, ed.SSE.Interface))
	stmt.Type().Id(ed.SSE.StructName).StructFunc(func(group *jen.Group) {
		group.Comment("writer owns the serialized SSE response lifecycle.")
		group.Id("writer").Op("*").Id("loomhttp").Dot("SSEStreamWriter")
		if serverSSEUsesDynamicView(ed) {
			group.Id("view").String()
		}
	})
	stmt.Line()
}

func addServerSSESetViewMethod(stmt *jen.Statement, ed *EndpointData) {
	if !serverSSEUsesDynamicView(ed) {
		return
	}
	codegen.Doc(stmt, "SetView sets the result view used by subsequent sends when no discriminator projection is configured.")
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ed.SSE.StructName)).
		Id("SetView").
		Params(jen.Id("view").String()).
		Block(jen.Id("s").Dot("view").Op("=").Id("view"))
	stmt.Line()
}

func serverSSEUsesDynamicView(ed *EndpointData) bool {
	return ed.Method.ViewedResult != nil && ed.Method.ViewedResult.ViewName == ""
}

func renderServerSSESendWithContextBody(ed *EndpointData) string {
	var b sourceBuilder
	b.Add("if err := ctx.Err(); err != nil {\n\treturn err\n}\n")
	writeSSEResultSetup(&b, ed)
	writeSSEPayloadSetup(&b, ed)
	writeSSEPayloadEncoding(&b, ed)
	writeSSEMessageSetup(&b, ed)
	b.Add("return s.writer.WriteEvent(ctx, func(w io.Writer) error {\n\treturn loomhttp.WriteSSEEvent(w, msg)\n})")
	return b.String()
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
	case "any":
		return "url.QueryEscape(loom.JSONValueString(v))"
	default:
		return "url.QueryEscape(" + renderJen(jen.Qual("fmt", "Sprintf").Call(jen.Lit("%v"), jen.Id("v"))) + ")"
	}
}

func renderRequestInitCode(payloadRef string, hasFields bool, serviceName, endpointName string, args []*InitArgData, route *RouteData, isWebSocket bool, requestStruct string) string {
	var b sourceBuilder
	renderRequestInitVars(&b, args, requestStruct)
	renderRequestPayloadSetup(&b, payloadRef, hasFields, serviceName, endpointName, args, requestStruct)
	renderRequestURLSetup(&b, serviceName, endpointName, route, args, isWebSocket)
	renderRequestCreation(&b, serviceName, endpointName, requestStruct, route.Verb)
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

// renderRequestURLSetup writes the statements that declare the URL u of the
// request to the route. The path builder of a route with wildcards, or with
// literal parts that need escaping, returns an escaped path, which
// loomhttp.RequestURL keeps so that an escaped "/" stays inside its segment.
func renderRequestURLSetup(b *sourceBuilder, serviceName, endpointName string, route *RouteData, args []*InitArgData, isWebSocket bool) {
	renderRequestScheme(b, isWebSocket)
	scheme := "c.scheme"
	if isWebSocket {
		scheme = "scheme"
	}
	call := route.PathInit.Name + "("
	for _, arg := range args {
		call += arg.Ref + ", "
	}
	call += ")"
	if len(route.PathInit.ClientArgs) == 0 && escapeRouteLiteral(route.Path) == route.Path {
		b.Addf("\tu := &url.URL{Scheme: %s, Host: c.host, Path: %s}\n", scheme, call)
		return
	}
	b.Addf("\tu, err := loomhttp.RequestURL(%s, c.host, %s)\n", scheme, call)
	b.Add("\tif err != nil {\n")
	b.Addf("\t\treturn nil, loomhttp.ErrInvalidURL(%q, %q, u.String(), err)\n", serviceName, endpointName)
	b.Add("\t}\n")
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
