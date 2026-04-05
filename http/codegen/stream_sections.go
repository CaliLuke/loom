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

func renderSSEClientReadEvent(implName string) string {
	var b sourceBuilder
	b.Add("// readEvent reads a single SSE event from the stream, respecting context\n")
	b.Add("// cancellation.  It first checks the internal buffer for a complete event\n")
	b.Add("// (delimited by double newlines). If no complete event is found, it reads from\n")
	b.Add("// the HTTP response body until it either finds an event boundary, reaches EOF,\n")
	b.Add("// or encounters an error. Any data after the event boundary is saved in the\n")
	b.Add("// buffer for the next call.\n")
	b.Addf("func (s *%s) readEvent(ctx context.Context) ([]byte, error) {\n", implName)
	b.Add("\tconst bufSize = 4096 // 4KB buffer size\n\n")
	b.Add("\t// Check for event in existing buffer\n")
	b.Add("\tevent, ok := s.checkBuffer()\n")
	b.Add("\tif ok {\n")
	b.Add("\t\treturn event, nil\n")
	b.Add("\t}\n\n")
	b.Add("\t// Initialize with any data from buffer\n")
	b.Add("\teventData := event\n")
	b.Add("\twasNewline := len(eventData) > 0 && eventData[len(eventData)-1] == '\\n'\n")
	b.Add("\tbuf := make([]byte, bufSize)\n\n")
	b.Add("\t// Read data in chunks until we find an event or hit EOF\n")
	b.Add("\tfor {\n")
	b.Add("\t\t// Check if context is done\n")
	b.Add("\t\tselect {\n")
	b.Add("\t\tcase <-ctx.Done():\n")
	b.Add("\t\t\tif len(eventData) > 0 {\n")
	b.Add("\t\t\t\treturn eventData, nil\n")
	b.Add("\t\t\t}\n")
	b.Add("\t\t\treturn nil, ctx.Err()\n")
	b.Add("\t\tdefault:\n")
	b.Add("\t\t\t// Continue processing\n")
	b.Add("\t\t}\n\n")
	b.Add("\t\t// Check if stream is closed\n")
	b.Add("\t\ts.lock.Lock()\n")
	b.Add("\t\tif s.closed {\n")
	b.Add("\t\t\ts.lock.Unlock()\n")
	b.Add("\t\t\tif len(eventData) > 0 {\n")
	b.Add("\t\t\t\treturn eventData, nil\n")
	b.Add("\t\t\t}\n")
	b.Add("\t\t\treturn nil, io.EOF\n")
	b.Add("\t\t}\n\n")
	b.Add("\t\t// Read next chunk\n")
	b.Add("\t\tn, err := s.resp.Body.Read(buf)\n")
	b.Add("\t\ts.lock.Unlock()\n\n")
	b.Add("\t\t// Handle read errors\n")
	b.Add("\t\tif err != nil && err != io.EOF {\n")
	b.Add("\t\t\treturn nil, err\n")
	b.Add("\t\t}\n\n")
	b.Add("\t\t// Process data if we got any\n")
	b.Add("\t\tif n > 0 {\n")
	b.Add("\t\t\t// Look for event boundary in this chunk\n")
	b.Add("\t\t\tfor i := 0; i < n; i++ {\n")
	b.Add("\t\t\t\tb := buf[i]\n")
	b.Add("\t\t\t\teventData = append(eventData, b)\n\n")
	b.Add("\t\t\t\t// Check for double newlines (event boundary)\n")
	b.Add("\t\t\t\tif b == '\\n' && wasNewline {\n")
	b.Add("\t\t\t\t\t// Save any remaining data for next read\n")
	b.Add("\t\t\t\t\tif i+1 < n {\n")
	b.Add("\t\t\t\t\t\ts.lock.Lock()\n")
	b.Add("\t\t\t\t\t\ts.buffer = append(s.buffer[:0], buf[i+1:n]...)\n")
	b.Add("\t\t\t\t\t\ts.lock.Unlock()\n")
	b.Add("\t\t\t\t\t}\n")
	b.Add("\t\t\t\t\treturn eventData, nil\n")
	b.Add("\t\t\t\t}\n\n")
	b.Add("\t\t\t\t// Update newline tracking\n")
	b.Add("\t\t\t\twasNewline = (b == '\\n')\n")
	b.Add("\t\t\t}\n")
	b.Add("\t\t}\n\n")
	b.Add("\t\t// Return partial data at EOF\n")
	b.Add("\t\tif errors.Is(err, io.EOF) {\n")
	b.Add("\t\t\tif len(eventData) > 0 {\n")
	b.Add("\t\t\t\treturn eventData, nil\n")
	b.Add("\t\t\t}\n")
	b.Add("\t\t\treturn nil, io.EOF\n")
	b.Add("\t\t}\n")
	b.Add("\t}\n")
	b.Add("}\n\n")
	return b.String()
}

func renderSSEClientCheckBuffer(implName string) string {
	var b sourceBuilder
	b.Add("// checkBuffer examines the internal buffer for a complete SSE event (delimited\n")
	b.Add("// by double newlines).  It returns two values: the event data (or all buffer\n")
	b.Add("// contents if no complete event is found), and a boolean indicating whether a\n")
	b.Add("// complete event was found. If a complete event is found, any remaining data\n")
	b.Add("// after the event is kept in the buffer for the next call.\n")
	b.Addf("func (s *%s) checkBuffer() ([]byte, bool) {\n", implName)
	b.Add("\ts.lock.Lock()\n")
	b.Add("\tdefer s.lock.Unlock()\n\n")
	b.Add("\t// Quick return if buffer is empty\n")
	b.Add("\tif len(s.buffer) == 0 {\n")
	b.Add("\t\treturn nil, false\n")
	b.Add("\t}\n\n")
	b.Add("\t// Look for double newline in buffer\n")
	b.Add("\tfor i := 0; i < len(s.buffer)-1; i++ {\n")
	b.Add("\t\tif s.buffer[i] == '\\n' && s.buffer[i+1] == '\\n' {\n")
	b.Add("\t\t\t// Found complete event\n")
	b.Add("\t\t\teventEnd := i + 2 // Include both newlines\n")
	b.Add("\t\t\teventData := s.buffer[:eventEnd]\n\n")
	b.Add("\t\t\t// Save remaining data for next time\n")
	b.Add("\t\t\tif eventEnd < len(s.buffer) {\n")
	b.Add("\t\t\t\ts.buffer = append(s.buffer[:0], s.buffer[eventEnd:]...)\n")
	b.Add("\t\t\t} else {\n")
	b.Add("\t\t\t\ts.buffer = s.buffer[:0]\n")
	b.Add("\t\t\t}\n\n")
	b.Add("\t\t\treturn eventData, true\n")
	b.Add("\t\t}\n")
	b.Add("\t}\n\n")
	b.Add("\t// No complete event found, return buffer contents\n")
	b.Add("\teventData := s.buffer\n")
	b.Add("\ts.buffer = s.buffer[:0] // Clear buffer but keep capacity\n")
	b.Add("\treturn eventData, false\n")
	b.Add("}\n\n")
	return b.String()
}

func renderSSEClientProcessEvent(implName string, ed *EndpointData) string {
	var b sourceBuilder
	b.Addf("// processEvent processes a raw SSE event into the expected type\nfunc (s *%s) processEvent(eventData []byte) (event %s, err error) {\n", implName, ed.SSE.EventTypeRef)
	b.Add("\tparsed, err := loomhttp.ParseSSEEvent(eventData)\n")
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn event, err\n")
	b.Add("\t}\n")
	if ed.SSE.EventIsStruct {
		b.Addf("\tevent = new(%s)\n", strings.TrimPrefix(ed.SSE.EventTypeRef, "*"))
	}
	if ed.SSE.IDField != "" {
		b.Addf("\tevent.%s = parsed.ID\n", ed.SSE.IDField)
	}
	if ed.SSE.EventField != "" {
		b.Addf("\tevent.%s = parsed.Type\n", ed.SSE.EventField)
	}
	b.Add("\tdataContent := parsed.Data\n")
	switch {
	case ed.SSE.DataField != "":
		b.Add(renderSSEParseAssignment("event."+ed.SSE.DataField, ed.SSE.DataFieldTypeRef))
	case ed.SSE.EventIsStruct:
		b.Add("\t// Decode JSON into the struct pointer directly\n")
		b.Add("\trespBody := &http.Response{\n")
		b.Add("\t\tStatusCode: http.StatusOK,\n")
		b.Add("\t\tBody:       io.NopCloser(bytes.NewReader([]byte(dataContent))),\n")
		b.Add("\t}\n")
		b.Add("\terr = s.decoder(respBody).Decode(event)\n")
		b.Add("\tif err != nil {\n")
		b.Add("\t\treturn\n")
		b.Add("\t}\n")
	default:
		b.Add(renderSSEParseAssignment("event", ed.SSE.EventTypeRef))
	}
	b.Add("\treturn\n")
	b.Add("}\n")
	return b.String()
}

func renderSSEParseAssignment(target, typeRef string) string {
	var b sourceBuilder
	switch typeRef {
	case "string":
		b.Addf("\t%s = dataContent\n", target)
	case "[]byte":
		b.Addf("\t%s = []byte(dataContent)\n", target)
	case "int":
		b.Addf("\tv, parseErr := strconv.Atoi(dataContent)\n")
		b.Add("\tif parseErr != nil {\n")
		b.Add("\t\terr = parseErr\n")
		b.Add("\t\treturn\n")
		b.Add("\t}\n")
		b.Addf("\t%s = v\n", target)
	default:
		b.Addf("\trespBody := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dataContent))}\n")
		b.Addf("\tif err = s.decoder(respBody).Decode(&%s); err != nil {\n", target)
		b.Add("\t\treturn\n")
		b.Add("\t}\n")
	}
	return b.String()
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
	b.Add(codegen.Comment(fmt.Sprintf("Upgrade the HTTP connection to a websocket connection only once. Connection upgrade is done here so that authorization logic in the endpoint is executed before calling the actual service method which may call %s().", function)))
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

func addSSEClientSection(stmt *jen.Statement, ed *EndpointData) {
	streamName := ed.Method.VarName + "ClientStream"
	implName := ed.Method.VarName + "StreamImpl"

	stmt.Line()
	codegen.Doc(stmt, streamName+" is the interface for reading Server-Sent Events.")
	stmt.Type().Id(streamName).Interface(
		jen.Comment("Recv reads and returns the next event from the SSE stream."),
		jen.Id("Recv").Params(jen.Qual("context", "Context")).Params(codegen.TypeRef(ed.SSE.EventTypeRef), jen.Error()),
		jen.Comment("Close closes the SSE stream and releases resources."),
		jen.Id("Close").Params().Error(),
	)
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
			addRawWebSocketGroup(group, fmt.Sprintf("return &%s{\n\tresp: resp,\n\tdecoder: decoder,\n\tbuffer: make([]byte, 0, 4096), // Pre-allocate buffer\n}", implName))
		})
	stmt.Line()
	codegen.Doc(stmt, "Recv reads and returns the next event from the SSE stream, respecting context cancellation.")
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(implName)).
		Id("Recv").
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(jen.Id("event").Add(codegen.TypeRef(ed.SSE.EventTypeRef)), jen.Id("err").Error()).
		BlockFunc(func(group *jen.Group) {
			addRawWebSocketGroup(group, renderSSEClientRecvBody())
		})
	stmt.Line()
	stmt.Add(codegen.Expr(strings.TrimSpace(renderSSEClientReadEvent(implName))))
	stmt.Line()
	stmt.Add(codegen.Expr(strings.TrimSpace(renderSSEClientCheckBuffer(implName))))
	stmt.Line()
	codegen.Doc(stmt, "Close closes the SSE stream and releases any associated resources.")
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(implName)).
		Id("Close").
		Params().
		Error().
		BlockFunc(func(group *jen.Group) {
			addRawWebSocketGroup(group, renderSSEClientCloseBody())
		})
	stmt.Line()
	stmt.Add(codegen.Expr(strings.TrimSpace(renderSSEClientProcessEvent(implName, ed))))
	stmt.Line()
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
