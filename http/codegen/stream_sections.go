package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func sseClientSections(data *ServiceData) []codegen.Section {
	sections := make([]codegen.Section, 0)
	for _, ed := range data.Endpoints {
		if ed.SSE == nil {
			continue
		}
		sections = append(sections, sseClientSection(ed))
	}
	return sections
}

func sseClientSection(ed *EndpointData) codegen.Section {
	return codegen.MustJenniferSection("client-sse", func(stmt *jen.Statement) {
		addSSEClientSection(stmt, ed)
	})
}

func websocketStructSections(data *ServiceData, client bool) []codegen.Section {
	sections := []codegen.Section{websocketConnConfigurerStructSection(data, client)}
	for _, e := range data.Endpoints {
		var ws *WebSocketData
		if client {
			ws = e.ClientWebSocket
		} else {
			ws = e.ServerWebSocket
		}
		if ws != nil {
			sections = append(sections, websocketStructTypeSection(ws))
		}
	}
	return sections
}

func websocketCodeSections(data *ServiceData, client bool) []codegen.Section {
	sections := []codegen.Section{websocketConnConfigurerInitSection(data, client)}
	for _, e := range data.Endpoints {
		var ws *WebSocketData
		if client {
			ws = e.ClientWebSocket
		} else {
			ws = e.ServerWebSocket
		}
		if ws == nil {
			continue
		}
		if client {
			if ws.RecvTypeRef != "" {
				sections = append(sections, websocketRecvSection(ws))
			}
			switch ws.Kind {
			case expr.ClientStreamKind, expr.BidirectionalStreamKind:
				sections = append(sections, websocketSendSection(ws))
			}
		} else {
			if ws.SendTypeRef != "" {
				sections = append(sections, websocketSendSection(ws))
			}
			switch ws.Kind {
			case expr.ClientStreamKind, expr.BidirectionalStreamKind:
				sections = append(sections, websocketRecvSection(ws))
			}
		}
		if ws.MustClose {
			sections = append(sections, websocketCloseSection(ws))
		}
		if ws.Endpoint.Method.ViewedResult != nil && ws.Endpoint.Method.ViewedResult.ViewName == "" {
			sections = append(sections, websocketSetViewSection(ws))
		}
	}
	return sections
}

func websocketConnConfigurerStructSection(data *ServiceData, client bool) codegen.Section {
	prefix := "server"
	if client {
		prefix = "client"
	}
	return codegen.MustJenniferSection(prefix+"-websocket-conn-configurer-struct", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("ConnConfigurer holds the websocket connection configurer functions for the streaming endpoints in %q service.", data.Service.Name))
		stmt.Type().Id("ConnConfigurer").StructFunc(func(group *jen.Group) {
			for _, endpoint := range data.Endpoints {
				if IsWebSocketEndpoint(endpoint) {
					group.Id(endpoint.Method.VarName + "Fn").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc"))
				}
			}
		})
		stmt.Line()
	})
}

func websocketConnConfigurerInitSection(data *ServiceData, client bool) codegen.Section {
	prefix := "server"
	if client {
		prefix = "client"
	}
	return codegen.MustJenniferSection(prefix+"-websocket-conn-configurer-struct-init", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("NewConnConfigurer initializes the websocket connection configurer function with fn for all the streaming endpoints in %q service.", data.Service.Name))
		stmt.Func().
			Id("NewConnConfigurer").
			Params(jen.Id("fn").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc"))).
			Op("*").Id("ConnConfigurer").
			BlockFunc(func(group *jen.Group) {
				var b sourceBuilder
				b.Add("return &ConnConfigurer{\n")
				for _, endpoint := range data.Endpoints {
					if IsWebSocketEndpoint(endpoint) {
						b.Addf("\t%sFn: fn,\n", endpoint.Method.VarName)
					}
				}
				b.Add("}")
				appendHTTPRawBlock(group, b.String())
			})
		stmt.Line()
	})
}

func websocketStructTypeSection(ws *WebSocketData) codegen.Section {
	prefix := ws.Type
	return codegen.MustJenniferSection(prefix+"-websocket-struct-type", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s implements the %s interface.", ws.VarName, ws.Interface))
		stmt.Type().Id(ws.VarName).StructFunc(func(group *jen.Group) {
			if ws.Type == "server" {
				group.Id("once").Qual("sync", "Once")
				group.Comment("upgradeErr is the error returned by the websocket upgrade attempt.")
				group.Id("upgradeErr").Error()
				group.Comment("upgrader is the websocket connection upgrader.")
				group.Id("upgrader").Add(codegen.TypeRef("loomhttp.Upgrader"))
				group.Comment("configurer is the websocket connection configurer.")
				group.Id("configurer").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc"))
				addWrappedGroupComment(group, "cancel is the context cancellation function which cancels the request context when invoked.")
				group.Id("cancel").Qual("context", "CancelFunc")
				group.Comment("w is the HTTP response writer used in upgrading the connection.")
				group.Id("w").Qual("net/http", "ResponseWriter")
				group.Comment("r is the HTTP request.")
				group.Id("r").Op("*").Qual("net/http", "Request")
			}
			group.Comment("conn is the underlying websocket connection.")
			group.Id("conn").Op("*").Qual("github.com/gorilla/websocket", "Conn")
			if ws.Endpoint.Method.ViewedResult != nil && ws.Endpoint.Method.ViewedResult.ViewName == "" {
				addWrappedGroupComment(group, fmt.Sprintf("view is the view to render %s result type before sending to the websocket connection.", ws.SendTypeName))
				group.Id("view").String()
			}
		})
		stmt.Line()
	})
}

func addWrappedGroupComment(group *jen.Group, text string) {
	for _, line := range strings.Split(codegen.Comment(text), "\n") {
		group.Comment(strings.TrimPrefix(line, "// "))
	}
}

func websocketSendSection(ws *WebSocketData) codegen.Section {
	return codegen.MustJenniferSection(ws.Type+"-websocket-send", func(stmt *jen.Statement) {
		addWebsocketSendSection(stmt, ws)
	})
}

func writeClientWebSocketSend(b *sourceBuilder, ws *WebSocketData) {
	if ws.Payload != nil && ws.Payload.Init != nil {
		b.Addf("\tbody := %s(v)\n", ws.Payload.Init.Name)
		b.Add("\treturn s.conn.WriteJSON(body)\n")
	} else {
		b.Add("\treturn s.conn.WriteJSON(v)\n")
	}
}

func writeServerWebSocketSend(b *sourceBuilder, ws *WebSocketData) {
	writeServerWebSocketSendPreamble(b, ws)
	writeServerWebSocketSendResult(b, ws)
	if !writeServerWebSocketResponseBody(b, ws) {
		b.Add("\treturn s.conn.WriteJSON(res)\n")
	}
}

func writeServerWebSocketSendPreamble(b *sourceBuilder, ws *WebSocketData) {
	if ws.SendName == "Send" {
		b.Add("\tvar err error\n")
		b.Add(renderWebsocketUpgrade(ws.Endpoint, ws.SendName, false))
		return
	}
	b.Add("\tdefer s.conn.Close()\n")
}

func writeServerWebSocketSendResult(b *sourceBuilder, ws *WebSocketData) {
	if ws.Endpoint.Method.ViewedResult == nil {
		b.Add("\tres := v\n")
		return
	}
	if ws.Endpoint.Method.ViewedResult.ViewName != "" {
		b.Addf("\tres := %s.%s(v, %q)\n", ws.PkgName, ws.Endpoint.Method.ViewedResult.Init.Name, ws.Endpoint.Method.ViewedResult.ViewName)
		return
	}
	b.Addf("\tres := %s.%s(v, s.view)\n", ws.PkgName, ws.Endpoint.Method.ViewedResult.Init.Name)
}

func writeServerWebSocketResponseBody(b *sourceBuilder, ws *WebSocketData) bool {
	if len(ws.Response.ServerBody) == 0 {
		return false
	}
	body := ws.Response.ServerBody[0]
	if body.Init == nil {
		return false
	}
	writeServerWebSocketBodyInit(b, ws, body)
	b.Add("\treturn s.conn.WriteJSON(body)\n")
	return true
}

func writeServerWebSocketBodyInit(b *sourceBuilder, ws *WebSocketData, body *TypeData) {
	if ws.Endpoint.Method.ViewedResult == nil {
		writeServerBodyInitCall(b, body, "\tbody := ")
		return
	}
	if ws.Endpoint.Method.ViewedResult.ViewName != "" {
		if vsb := viewedServerBody(ws.Response.ServerBody, ws.Endpoint.Method.ViewedResult.ViewName); vsb != nil {
			writeServerBodyInitCall(b, vsb, "\tbody := ")
		}
		return
	}
	b.Add("\tvar body any\n")
	b.Add("\tswitch s.view {\n")
	for _, view := range ws.Endpoint.Method.ViewedResult.Views {
		writeViewedServerBodyCase(b, ws, view.Name)
	}
	b.Add("\t}\n")
}

func writeViewedServerBodyCase(b *sourceBuilder, ws *WebSocketData, viewName string) {
	if viewName == "default" {
		b.Addf("\tcase %q, \"\":\n", viewName)
	} else {
		b.Addf("\tcase %q:\n", viewName)
	}
	if vsb := viewedServerBody(ws.Response.ServerBody, viewName); vsb != nil {
		writeServerBodyInitCall(b, vsb, "\t\tbody = ")
	}
}

func writeServerBodyInitCall(b *sourceBuilder, body *TypeData, prefix string) {
	b.Addf("%s%s(", prefix, body.Init.Name)
	for _, arg := range body.Init.ServerArgs {
		b.Addf("%s, ", arg.Ref)
	}
	b.Add(")\n")
}

func addWebsocketSendSection(stmt *jen.Statement, ws *WebSocketData) {
	stmt.Line()
	codegen.Doc(stmt, ws.SendDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendName).
		Params(jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		BlockFunc(func(group *jen.Group) {
			if ws.Type != "server" {
				var b sourceBuilder
				writeClientWebSocketSend(&b, ws)
				addRawWebSocketGroup(group, b.String())
				return
			}
			var b sourceBuilder
			writeServerWebSocketSend(&b, ws)
			addRawWebSocketGroup(group, b.String())
		})
	stmt.Line()
	codegen.Doc(stmt, ws.SendWithContextDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		Block(
			jen.Return(jen.Id("s").Dot(ws.SendName).Call(jen.Id("v"))),
		)
	stmt.Line()
}

func websocketRecvSection(ws *WebSocketData) codegen.Section {
	return codegen.MustJenniferSection(ws.Type+"-websocket-recv", func(stmt *jen.Statement) {
		addWebsocketRecvSection(stmt, ws)
	})
}

func writeWebsocketRecvVars(b *sourceBuilder, ws *WebSocketData) {
	b.Add("\tvar (\n")
	b.Addf("\t\trv %s\n", ws.RecvTypeRef)
	if ws.Type == "server" {
		if ws.RecvTypeIsPointer {
			b.Addf("\t\tbody %s\n", ws.Payload.VarName)
		} else {
			b.Addf("\t\tmsg *%s\n", ws.Payload.VarName)
		}
	} else {
		bodyTypeRef := ws.RecvTypeRef
		if ws.Response != nil && ws.Response.ClientBody != nil {
			bodyTypeRef = ws.Response.ClientBody.VarName
		}
		b.Addf("\t\tbody %s\n", bodyTypeRef)
	}
	b.Add("\t\terr error\n")
	b.Add("\t)\n")
}

func writeServerWebsocketRecvBody(b *sourceBuilder, ws *WebSocketData) {
	b.Add(renderWebsocketUpgrade(ws.Endpoint, ws.RecvName, true))
	if ws.RecvTypeIsPointer {
		b.Add("\tif err = s.conn.ReadJSON(&body); err != nil {\n")
	} else {
		b.Add("\tif err = s.conn.ReadJSON(&msg); err != nil {\n")
	}
	b.Add("\t\treturn rv, err\n")
	b.Add("\t}\n")
	if ws.RecvTypeIsPointer {
		b.Add("\tif body == nil {\n")
	} else {
		b.Add("\tif msg == nil {\n")
	}
	b.Add("\t\treturn rv, io.EOF\n")
	b.Add("\t}\n")
	writeServerWebsocketRecvValidation(b, ws)
	writeServerWebsocketRecvReturn(b, ws)
}

func writeServerWebsocketRecvValidation(b *sourceBuilder, ws *WebSocketData) {
	if ws.Payload == nil || ws.Payload.ValidateRef == "" {
		return
	}
	if !ws.RecvTypeIsPointer {
		b.Add("\tbody := *msg\n")
	}
	b.Addf("\t%s\n", ws.Payload.ValidateRef)
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn rv, err\n")
	b.Add("\t}\n")
}

func writeServerWebsocketRecvReturn(b *sourceBuilder, ws *WebSocketData) {
	switch {
	case ws.Payload != nil && ws.Payload.Init != nil:
		if ws.RecvTypeIsPointer {
			b.Addf("\treturn %s(body), nil\n", ws.Payload.Init.Name)
		} else {
			b.Addf("\treturn %s(msg), nil\n", ws.Payload.Init.Name)
		}
	case ws.RecvTypeIsPointer:
		b.Add("\treturn body, nil\n")
	default:
		b.Add("\treturn *msg, nil\n")
	}
}

func writeClientWebsocketRecvBody(b *sourceBuilder, ws *WebSocketData) {
	if ws.RecvName == "CloseAndRecv" {
		b.Add("\tdefer s.conn.Close()\n")
		b.Add("\t// Send a nil payload to the server implying end of message\n")
		b.Add("\tif err = s.conn.WriteJSON(nil); err != nil {\n")
		b.Add("\t\treturn rv, err\n")
		b.Add("\t}\n")
	}
	b.Add("\terr = s.conn.ReadJSON(&body)\n")
	b.Add("\tif websocket.IsCloseError(err, websocket.CloseNormalClosure) {\n")
	if !ws.MustClose {
		b.Add("\t\ts.conn.Close()\n")
	}
	b.Add("\t\treturn rv, io.EOF\n")
	b.Add("\t}\n")
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn rv, err\n")
	b.Add("\t}\n")
	writeClientWebsocketRecvValidation(b, ws)
	writeClientWebsocketRecvReturn(b, ws)
}

func writeClientWebsocketRecvValidation(b *sourceBuilder, ws *WebSocketData) {
	if ws.Response.ClientBody == nil || ws.Response.ClientBody.ValidateRef == "" || ws.Endpoint.Method.ViewedResult != nil {
		return
	}
	b.Addf("\t%s\n", ws.Response.ClientBody.ValidateRef)
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn rv, err\n")
	b.Add("\t}\n")
}

func writeClientWebsocketRecvReturn(b *sourceBuilder, ws *WebSocketData) {
	if ws.Response.ResultInit == nil {
		b.Add("\treturn body, nil\n")
		return
	}
	b.Add("\tres := ")
	b.Addf("%s(", ws.Response.ResultInit.Name)
	for _, arg := range ws.Response.ResultInit.ClientArgs {
		b.Addf("%s,", arg.Ref)
	}
	b.Add(")\n")
	if ws.Endpoint.Method.ViewedResult == nil {
		b.Add("\treturn res, nil\n")
		return
	}
	writeClientWebsocketViewedResultReturn(b, ws)
}

func writeClientWebsocketViewedResultReturn(b *sourceBuilder, ws *WebSocketData) {
	view := ws.Endpoint.Method.ViewedResult
	prefix := ""
	if !view.IsCollection {
		prefix = "&"
	}
	viewArg := fmt.Sprintf("%q", view.ViewName)
	if view.ViewName == "" {
		viewArg = "s.view"
	}
	b.Addf("\tvres := %s%s.%s{res, %s }\n", prefix, view.ViewsPkg, view.VarName, viewArg)
	b.Addf("\tif err := %s.Validate%s(vres); err != nil {\n", view.ViewsPkg, ws.Endpoint.Method.Result)
	b.Addf("\t\treturn rv, loomhttp.ErrValidationError(%q, %q, err)\n", ws.Endpoint.ServiceName, ws.Endpoint.Method.Name)
	b.Add("\t}\n")
	b.Addf("\treturn %s.%s(vres), nil\n", ws.PkgName, view.ResultInit.Name)
}

func websocketCloseSection(ws *WebSocketData) codegen.Section {
	return codegen.MustJenniferSection(ws.Type+"-websocket-close", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("Close closes the %q endpoint websocket connection.", ws.Endpoint.Method.Name))
		stmt.Func().
			Params(jen.Id("s").Op("*").Id(ws.VarName)).
			Id("Close").
			Params().
			Error().
			BlockFunc(func(group *jen.Group) {
				addRawWebSocketGroup(group, renderWebSocketCloseBody(ws))
			})
		stmt.Line()
	})
}

func websocketSetViewSection(ws *WebSocketData) codegen.Section {
	return codegen.MustJenniferSection(ws.Type+"-websocket-set-view", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("SetView sets the view to render the %s type before sending to the %q endpoint websocket connection.", ws.SendTypeName, ws.Endpoint.Method.Name))
		stmt.Func().
			Params(jen.Id("s").Op("*").Id(ws.VarName)).
			Id("SetView").
			Params(jen.Id("view").String()).
			Block(
				jen.Id("s").Dot("view").Op("=").Id("view"),
			)
		stmt.Line()
	})
}

func renderWebsocketUpgrade(endpoint *EndpointData, function string, recv bool) string {
	var b sourceBuilder
	b.Add("\t")
	b.Add(codegen.Comment("Upgrade the HTTP connection to a websocket connection only once. Connection upgrade is done here so that authorization logic in the endpoint is executed before calling the actual service method which may call " + function + "()."))
	b.Add("\n")
	b.Add("\ts.once.Do(func() {\n")
	if endpoint.Method.ViewedResult != nil && function == "Send" && endpoint.Method.ViewedResult.ViewName == "" {
		b.Add("\t\trespHdr := make(http.Header)\n")
		b.Add("\t\trespHdr.Add(\"loom-view\", s.view)\n")
	}
	b.Add("\t\tvar conn *websocket.Conn\n")
	if function == "Send" && endpoint.Method.ViewedResult != nil && endpoint.Method.ViewedResult.ViewName == "" {
		b.Add("\t\tconn, err = s.upgrader.Upgrade(s.w, s.r, respHdr)\n")
	} else {
		b.Add("\t\tconn, err = s.upgrader.Upgrade(s.w, s.r, nil)\n")
	}
	b.Add("\t\tif err != nil {\n")
	b.Add("\t\t\ts.upgradeErr = err\n")
	b.Add("\t\t\treturn\n")
	b.Add("\t\t}\n")
	b.Add("\t\tif s.configurer != nil {\n")
	b.Add("\t\t\tconn = s.configurer(conn, s.cancel)\n")
	b.Add("\t\t}\n")
	b.Add("\t\ts.conn = conn\n")
	b.Add("\t})\n")
	b.Add("\tif s.upgradeErr != nil {\n")
	if recv {
		b.Add("\t\treturn rv, s.upgradeErr\n")
	} else {
		b.Add("\t\treturn s.upgradeErr\n")
	}
	b.Add("\t}\n")
	return b.String()
}

func buildStreamRequestSection(endpoint *EndpointData) codegen.Section {
	return codegen.MustJenniferSection("build-stream-request", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s creates a streaming endpoint request payload from the method payload and the path to the file to be streamed", endpoint.BuildStreamPayload))
		stmt.Func().
			Id(endpoint.BuildStreamPayload).
			ParamsFunc(func(group *jen.Group) {
				if endpoint.Payload.Ref != "" {
					group.Id("payload").Any()
				}
				group.Id("fpath").String()
			}).
			Params(jen.Op("*").Id(requestStructPkg(endpoint.Method, endpoint.ServicePkgName)).Dot(endpoint.Method.RequestStruct), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				addRawWebSocketGroup(group, renderBuildStreamRequestBody(endpoint))
			})
		stmt.Line()
	})
}

func multipartRequestEncoderTypeSection(data *MultipartData) codegen.Section {
	return codegen.MustJenniferSection("multipart-request-encoder-type", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s is the type to encode multipart request for the %q service %q endpoint.", data.FuncName, data.ServiceName, data.MethodName))
		stmt.Type().Id(data.FuncName).Func().Params(
			jen.Op("*").Qual("mime/multipart", "Writer"),
			codegen.TypeRef(data.Payload.Ref),
		).Error()
		stmt.Line()
	})
}

func renderWebSocketCloseBody(ws *WebSocketData) string {
	var b sourceBuilder
	b.Add("var err error\n")
	if ws.Type == "server" {
		b.Add("if s.conn == nil {\n\treturn nil\n}\n")
		b.Add("if err = s.conn.WriteControl(\n")
		b.Add("\twebsocket.CloseMessage,\n")
		b.Add("\twebsocket.FormatCloseMessage(websocket.CloseNormalClosure, \"server closing connection\"),\n")
		b.Add("\ttime.Now().Add(time.Second),\n")
		b.Add("); err != nil {\n\treturn err\n}\n")
	} else {
		b.Add("// Send a nil payload to the server implying client closing connection.\n")
		b.Add("if err = s.conn.WriteJSON(nil); err != nil {\n\treturn err\n}\n")
	}
	b.Add("return s.conn.Close()")
	return b.String()
}

func renderBuildStreamRequestBody(endpoint *EndpointData) string {
	var b sourceBuilder
	b.Add("f, err := os.Open(fpath)\n")
	b.Add("if err != nil {\n\treturn nil, err\n}\n")
	b.Addf("return &%s.%s{\n", requestStructPkg(endpoint.Method, endpoint.ServicePkgName), endpoint.Method.RequestStruct)
	if endpoint.Payload.Ref != "" {
		b.Addf("\tPayload: payload.(%s),\n", endpoint.Payload.Ref)
	}
	b.Add("\tBody: f,\n")
	b.Add("}, nil")
	return b.String()
}

func addRawWebSocketGroup(group *jen.Group, code string) {
	if strings.TrimSpace(code) == "" {
		return
	}
	if strings.HasPrefix(code, "\n") {
		group.Line()
	}
	group.Add(codegen.Expr(strings.TrimRight(code, "\n")))
}

func addSSEClientInterface(stmt *jen.Statement, ed *EndpointData, streamName string) {
	stmt.Line()
	codegen.Doc(stmt, streamName+" is the interface for reading Server-Sent Events.")
	stmt.Type().Id(streamName).Interface(
		jen.Comment("Recv reads and returns the next event from the SSE stream."),
		jen.Id("Recv").Params(jen.Qual("context", "Context")).Params(codegen.TypeRef(ed.SSE.EventTypeRef), jen.Error()),
		jen.Comment("Close closes the SSE stream and releases resources."),
		jen.Id("Close").Params().Error(),
	)
}

func addSSEClientImplStruct(stmt *jen.Statement, streamName, implName string) {
	stmt.Line()
	stmt.Type().DefsFunc(func(group *jen.Group) {
		group.Comment(implName + " implements the " + streamName + " interface.")
		group.Id(implName).Struct(
			jen.Id("resp").Op("*").Qual("net/http", "Response"),
			jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")),
			jen.Id("buffer").Index().Byte().Comment("Buffer for unprocessed data"),
			jen.Id("lock").Qual("sync", "Mutex"),
			jen.Id("closed").Bool(),
		)
	})
	stmt.Line()
	codegen.Doc(stmt, implName+" implements the "+streamName+" interface.")
	stmt.Var().Id("_").Id(streamName).Op("=").Parens(jen.Op("*").Id(implName)).Call(jen.Nil())
}

func addSSEClientConstructor(stmt *jen.Statement, ed *EndpointData, streamName, implName string) {
	stmt.Line()
	codegen.Doc(stmt, "New"+ed.Method.VarName+"Stream creates a new "+streamName+".")
	stmt.Func().
		Id("New"+ed.Method.VarName+"Stream").
		Params(
			jen.Id("resp").Op("*").Qual("net/http", "Response"),
			jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")),
		).
		Id(streamName).
		BlockFunc(func(group *jen.Group) {
			group.Return(
				jen.Op("&").Id(implName).Values(jen.Dict{
					jen.Id("resp"):    jen.Id("resp"),
					jen.Id("decoder"): jen.Id("decoder"),
					jen.Id("buffer"):  jen.Make(jen.Index().Byte(), jen.Lit(0), jen.Lit(4096)),
				}),
			)
		})
}

func renderSSEClientRecvBody() string {
	var b sourceBuilder
	b.Add("var byts []byte\n")
	b.Add("byts, err = s.readEvent(ctx)\n")
	b.Add("if err != nil {\n")
	b.Add("\tif errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {\n")
	b.Add("\t\t// Clean up on EOF or context cancellation\n")
	b.Add("\t\ts.Close()\n")
	b.Add("\t\tif errors.Is(err, io.EOF) {\n")
	b.Add("\t\t\terr = nil\n")
	b.Add("\t\t}\n")
	b.Add("\t}\n")
	b.Add("\treturn\n")
	b.Add("}\n")
	b.Add("return s.processEvent(byts)")
	return b.String()
}

func renderSSEClientCloseBody() string {
	var b sourceBuilder
	b.Add("s.lock.Lock()\n")
	b.Add("defer s.lock.Unlock()\n")
	b.Add("if s.closed {\n\treturn nil\n}\n")
	b.Add("s.closed = true\n")
	b.Add("return s.resp.Body.Close()")
	return b.String()
}

func addWebsocketRecvSection(stmt *jen.Statement, ws *WebSocketData) {
	stmt.Line()
	codegen.Doc(stmt, ws.RecvDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.RecvName).
		Params().
		Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			var b sourceBuilder
			writeWebsocketRecvVars(&b, ws)
			if ws.Type == "server" {
				writeServerWebsocketRecvBody(&b, ws)
			} else {
				writeClientWebsocketRecvBody(&b, ws)
			}
			addRawWebSocketGroup(group, b.String())
		})
	stmt.Line()
	codegen.Doc(stmt, ws.RecvWithContextDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.RecvWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
		Block(
			jen.Return(jen.Id("s").Dot(ws.RecvName).Call()),
		)
	stmt.Line()
}
