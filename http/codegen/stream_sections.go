package codegen

import (
	"fmt"
	"strings"

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
	return codegen.NewRawSection("client-sse", renderSSEClientSection(ed))
}

func renderSSEClientSection(ed *EndpointData) string {
	var b strings.Builder
	streamName := ed.Method.VarName + "ClientStream"
	implName := ed.Method.VarName + "StreamImpl"

	b.WriteString(renderSSEClientTypes(streamName, implName, ed))
	b.WriteString(renderSSEClientConstructor(streamName, implName, ed))
	b.WriteString(renderSSEClientRecv(implName, ed))
	b.WriteString(renderSSEClientReadEvent(implName))
	b.WriteString(renderSSEClientCheckBuffer(implName))
	b.WriteString(renderSSEClientClose(implName))
	b.WriteString(renderSSEClientProcessEvent(implName, ed))
	return b.String()
}

func renderSSEClientTypes(streamName, implName string, ed *EndpointData) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(streamName + " is the interface for reading Server-Sent Events."))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s interface {\n", streamName)
	b.WriteString("\t" + codegen.Comment("Recv reads and returns the next event from the SSE stream.") + "\n")
	fmt.Fprintf(&b, "\tRecv(context.Context) (%s, error)\n", ed.SSE.EventTypeRef)
	b.WriteString("\t" + codegen.Comment("Close closes the SSE stream and releases resources.") + "\n")
	b.WriteString("\tClose() error\n")
	b.WriteString("}\n\n")
	b.WriteString("type (\n")
	fmt.Fprintf(&b, "\t%s\n", codegen.Comment(implName+" implements the "+streamName+" interface."))
	fmt.Fprintf(&b, "\t%s struct {\n", implName)
	b.WriteString("\t\tresp *http.Response\n")
	b.WriteString("\t\tdecoder func(*http.Response) loomhttp.Decoder\n")
	b.WriteString("\t\tbuffer []byte // Buffer for unprocessed data\n")
	b.WriteString("\t\tlock sync.Mutex\n")
	b.WriteString("\t\tclosed bool\n")
	b.WriteString("\t}\n")
	b.WriteString(")\n\n")
	fmt.Fprintf(&b, "%s\n", codegen.Comment(implName+" implements the "+streamName+" interface."))
	fmt.Fprintf(&b, "var _ %s = (*%s)(nil)\n\n", streamName, implName)
	return b.String()
}

func renderSSEClientConstructor(streamName, implName string, ed *EndpointData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment("New"+ed.Method.VarName+"Stream creates a new "+streamName+"."))
	fmt.Fprintf(&b, "func New%sStream(resp *http.Response, decoder func(*http.Response) loomhttp.Decoder) %s {\n", ed.Method.VarName, streamName)
	fmt.Fprintf(&b, "\treturn &%s{\n", implName)
	b.WriteString("\t\tresp: resp,\n")
	b.WriteString("\t\tdecoder: decoder,\n")
	b.WriteString("\t\tbuffer: make([]byte, 0, 4096), // Pre-allocate buffer\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderSSEClientRecv(implName string, ed *EndpointData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment("Recv reads and returns the next event from the SSE stream, respecting context cancellation."))
	fmt.Fprintf(&b, "func (s *%s) Recv(ctx context.Context) (event %s, err error) {\n", implName, ed.SSE.EventTypeRef)
	b.WriteString("\tvar byts []byte\n")
	b.WriteString("\tbyts, err = s.readEvent(ctx)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\tif errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {\n")
	b.WriteString("\t\t\t// Clean up on EOF or context cancellation\n")
	b.WriteString("\t\t\ts.Close()\n")
	b.WriteString("\t\t\tif errors.Is(err, io.EOF) {\n")
	b.WriteString("\t\t\t\terr = nil\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\treturn\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn s.processEvent(byts)\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderSSEClientReadEvent(implName string) string {
	var b strings.Builder
	b.WriteString("// readEvent reads a single SSE event from the stream, respecting context\n")
	b.WriteString("// cancellation.  It first checks the internal buffer for a complete event\n")
	b.WriteString("// (delimited by double newlines). If no complete event is found, it reads from\n")
	b.WriteString("// the HTTP response body until it either finds an event boundary, reaches EOF,\n")
	b.WriteString("// or encounters an error. Any data after the event boundary is saved in the\n")
	b.WriteString("// buffer for the next call.\n")
	fmt.Fprintf(&b, "func (s *%s) readEvent(ctx context.Context) ([]byte, error) {\n", implName)
	b.WriteString("\tconst bufSize = 4096 // 4KB buffer size\n\n")
	b.WriteString("\t// Check for event in existing buffer\n")
	b.WriteString("\tevent, ok := s.checkBuffer()\n")
	b.WriteString("\tif ok {\n")
	b.WriteString("\t\treturn event, nil\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\t// Initialize with any data from buffer\n")
	b.WriteString("\teventData := event\n")
	b.WriteString("\twasNewline := len(eventData) > 0 && eventData[len(eventData)-1] == '\\n'\n")
	b.WriteString("\tbuf := make([]byte, bufSize)\n\n")
	b.WriteString("\t// Read data in chunks until we find an event or hit EOF\n")
	b.WriteString("\tfor {\n")
	b.WriteString("\t\t// Check if context is done\n")
	b.WriteString("\t\tselect {\n")
	b.WriteString("\t\tcase <-ctx.Done():\n")
	b.WriteString("\t\t\tif len(eventData) > 0 {\n")
	b.WriteString("\t\t\t\treturn eventData, nil\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\treturn nil, ctx.Err()\n")
	b.WriteString("\t\tdefault:\n")
	b.WriteString("\t\t\t// Continue processing\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\t// Check if stream is closed\n")
	b.WriteString("\t\ts.lock.Lock()\n")
	b.WriteString("\t\tif s.closed {\n")
	b.WriteString("\t\t\ts.lock.Unlock()\n")
	b.WriteString("\t\t\tif len(eventData) > 0 {\n")
	b.WriteString("\t\t\t\treturn eventData, nil\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\treturn nil, io.EOF\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\t// Read next chunk\n")
	b.WriteString("\t\tn, err := s.resp.Body.Read(buf)\n")
	b.WriteString("\t\ts.lock.Unlock()\n\n")
	b.WriteString("\t\t// Handle read errors\n")
	b.WriteString("\t\tif err != nil && err != io.EOF {\n")
	b.WriteString("\t\t\treturn nil, err\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\t// Process data if we got any\n")
	b.WriteString("\t\tif n > 0 {\n")
	b.WriteString("\t\t\t// Look for event boundary in this chunk\n")
	b.WriteString("\t\t\tfor i := 0; i < n; i++ {\n")
	b.WriteString("\t\t\t\tb := buf[i]\n")
	b.WriteString("\t\t\t\teventData = append(eventData, b)\n\n")
	b.WriteString("\t\t\t\t// Check for double newlines (event boundary)\n")
	b.WriteString("\t\t\t\tif b == '\\n' && wasNewline {\n")
	b.WriteString("\t\t\t\t\t// Save any remaining data for next read\n")
	b.WriteString("\t\t\t\t\tif i+1 < n {\n")
	b.WriteString("\t\t\t\t\t\ts.lock.Lock()\n")
	b.WriteString("\t\t\t\t\t\ts.buffer = append(s.buffer[:0], buf[i+1:n]...)\n")
	b.WriteString("\t\t\t\t\t\ts.lock.Unlock()\n")
	b.WriteString("\t\t\t\t\t}\n")
	b.WriteString("\t\t\t\t\treturn eventData, nil\n")
	b.WriteString("\t\t\t\t}\n\n")
	b.WriteString("\t\t\t\t// Update newline tracking\n")
	b.WriteString("\t\t\t\twasNewline = (b == '\\n')\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\t// Return partial data at EOF\n")
	b.WriteString("\t\tif errors.Is(err, io.EOF) {\n")
	b.WriteString("\t\t\tif len(eventData) > 0 {\n")
	b.WriteString("\t\t\t\treturn eventData, nil\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\treturn nil, io.EOF\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderSSEClientCheckBuffer(implName string) string {
	var b strings.Builder
	b.WriteString("// checkBuffer examines the internal buffer for a complete SSE event (delimited\n")
	b.WriteString("// by double newlines).  It returns two values: the event data (or all buffer\n")
	b.WriteString("// contents if no complete event is found), and a boolean indicating whether a\n")
	b.WriteString("// complete event was found. If a complete event is found, any remaining data\n")
	b.WriteString("// after the event is kept in the buffer for the next call.\n")
	fmt.Fprintf(&b, "func (s *%s) checkBuffer() ([]byte, bool) {\n", implName)
	b.WriteString("\ts.lock.Lock()\n")
	b.WriteString("\tdefer s.lock.Unlock()\n\n")
	b.WriteString("\t// Quick return if buffer is empty\n")
	b.WriteString("\tif len(s.buffer) == 0 {\n")
	b.WriteString("\t\treturn nil, false\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\t// Look for double newline in buffer\n")
	b.WriteString("\tfor i := 0; i < len(s.buffer)-1; i++ {\n")
	b.WriteString("\t\tif s.buffer[i] == '\\n' && s.buffer[i+1] == '\\n' {\n")
	b.WriteString("\t\t\t// Found complete event\n")
	b.WriteString("\t\t\teventEnd := i + 2 // Include both newlines\n")
	b.WriteString("\t\t\teventData := s.buffer[:eventEnd]\n\n")
	b.WriteString("\t\t\t// Save remaining data for next time\n")
	b.WriteString("\t\t\tif eventEnd < len(s.buffer) {\n")
	b.WriteString("\t\t\t\ts.buffer = append(s.buffer[:0], s.buffer[eventEnd:]...)\n")
	b.WriteString("\t\t\t} else {\n")
	b.WriteString("\t\t\t\ts.buffer = s.buffer[:0]\n")
	b.WriteString("\t\t\t}\n\n")
	b.WriteString("\t\t\treturn eventData, true\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\t// No complete event found, return buffer contents\n")
	b.WriteString("\teventData := s.buffer\n")
	b.WriteString("\ts.buffer = s.buffer[:0] // Clear buffer but keep capacity\n")
	b.WriteString("\treturn eventData, false\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderSSEClientClose(implName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment("Close closes the SSE stream and releases any associated resources."))
	fmt.Fprintf(&b, "func (s *%s) Close() error {\n", implName)
	b.WriteString("\ts.lock.Lock()\n")
	b.WriteString("\tdefer s.lock.Unlock()\n")
	b.WriteString("\tif s.closed {\n")
	b.WriteString("\t\treturn nil\n")
	b.WriteString("\t}\n")
	b.WriteString("\ts.closed = true\n")
	b.WriteString("\treturn s.resp.Body.Close()\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderSSEClientProcessEvent(implName string, ed *EndpointData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// processEvent processes a raw SSE event into the expected type\nfunc (s *%s) processEvent(eventData []byte) (event %s, err error) {\n", implName, ed.SSE.EventTypeRef)
	b.WriteString("\tparsed, err := loomhttp.ParseSSEEvent(eventData)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn event, err\n")
	b.WriteString("\t}\n")
	if ed.SSE.EventIsStruct {
		fmt.Fprintf(&b, "\tevent = new(%s)\n", strings.TrimPrefix(ed.SSE.EventTypeRef, "*"))
	}
	if ed.SSE.IDField != "" {
		fmt.Fprintf(&b, "\tevent.%s = parsed.ID\n", ed.SSE.IDField)
	}
	if ed.SSE.EventField != "" {
		fmt.Fprintf(&b, "\tevent.%s = parsed.Type\n", ed.SSE.EventField)
	}
	b.WriteString("\tdataContent := parsed.Data\n")
	switch {
	case ed.SSE.DataField != "":
		b.WriteString(renderSSEParseAssignment("event."+ed.SSE.DataField, ed.SSE.DataFieldTypeRef))
	case ed.SSE.EventIsStruct:
		b.WriteString("\t// Decode JSON into the struct pointer directly\n")
		b.WriteString("\trespBody := &http.Response{\n")
		b.WriteString("\t\tStatusCode: http.StatusOK,\n")
		b.WriteString("\t\tBody:       io.NopCloser(bytes.NewReader([]byte(dataContent))),\n")
		b.WriteString("\t}\n")
		b.WriteString("\terr = s.decoder(respBody).Decode(event)\n")
		b.WriteString("\tif err != nil {\n")
		b.WriteString("\t\treturn\n")
		b.WriteString("\t}\n")
	default:
		b.WriteString(renderSSEParseAssignment("event", ed.SSE.EventTypeRef))
	}
	b.WriteString("\treturn\n")
	b.WriteString("}\n")
	return b.String()
}

func renderSSEParseAssignment(target, typeRef string) string {
	var b strings.Builder
	switch typeRef {
	case "string":
		fmt.Fprintf(&b, "\t%s = dataContent\n", target)
	case "[]byte":
		fmt.Fprintf(&b, "\t%s = []byte(dataContent)\n", target)
	case "int":
		fmt.Fprintf(&b, "\tv, parseErr := strconv.Atoi(dataContent)\n")
		b.WriteString("\tif parseErr != nil {\n")
		b.WriteString("\t\terr = parseErr\n")
		b.WriteString("\t\treturn\n")
		b.WriteString("\t}\n")
		fmt.Fprintf(&b, "\t%s = v\n", target)
	default:
		fmt.Fprintf(&b, "\trespBody := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(dataContent))}\n")
		fmt.Fprintf(&b, "\tif err = s.decoder(respBody).Decode(&%s); err != nil {\n", target)
		b.WriteString("\t\treturn\n")
		b.WriteString("\t}\n")
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
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("ConnConfigurer holds the websocket connection configurer functions for the streaming endpoints in %q service.", data.Service.Name)))
	b.WriteString("\n")
	b.WriteString("type ConnConfigurer struct {\n")
	for _, endpoint := range data.Endpoints {
		if IsWebSocketEndpoint(endpoint) {
			fmt.Fprintf(&b, "\t%sFn loomhttp.ConnConfigureFunc\n", endpoint.Method.VarName)
		}
	}
	b.WriteString("}\n")
	prefix := "server"
	if client {
		prefix = "client"
	}
	return codegen.NewRawSection(prefix+"-websocket-conn-configurer-struct", b.String())
}

func websocketConnConfigurerInitSection(data *ServiceData, client bool) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("NewConnConfigurer initializes the websocket connection configurer function with fn for all the streaming endpoints in %q service.", data.Service.Name)))
	b.WriteString("\n")
	b.WriteString("func NewConnConfigurer(fn loomhttp.ConnConfigureFunc) *ConnConfigurer {\n")
	b.WriteString("\treturn &ConnConfigurer{\n")
	for _, endpoint := range data.Endpoints {
		if IsWebSocketEndpoint(endpoint) {
			fmt.Fprintf(&b, "\t\t%sFn: fn,\n", endpoint.Method.VarName)
		}
	}
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	prefix := "server"
	if client {
		prefix = "client"
	}
	return codegen.NewRawSection(prefix+"-websocket-conn-configurer-struct-init", b.String())
}

func websocketStructTypeSection(ws *WebSocketData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s implements the %s interface.", ws.VarName, ws.Interface)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s struct {\n", ws.VarName)
	if ws.Type == "server" {
		b.WriteString("\tonce sync.Once\n")
		b.WriteString("\t" + codegen.Comment("upgradeErr is the error returned by the websocket upgrade attempt.") + "\n")
		b.WriteString("\tupgradeErr error\n")
		b.WriteString("\t" + codegen.Comment("upgrader is the websocket connection upgrader.") + "\n")
		b.WriteString("\tupgrader loomhttp.Upgrader\n")
		b.WriteString("\t" + codegen.Comment("configurer is the websocket connection configurer.") + "\n")
		b.WriteString("\tconfigurer loomhttp.ConnConfigureFunc\n")
		b.WriteString("\t" + codegen.Comment("cancel is the context cancellation function which cancels the request context when invoked.") + "\n")
		b.WriteString("\tcancel context.CancelFunc\n")
		b.WriteString("\t" + codegen.Comment("w is the HTTP response writer used in upgrading the connection.") + "\n")
		b.WriteString("\tw http.ResponseWriter\n")
		b.WriteString("\t" + codegen.Comment("r is the HTTP request.") + "\n")
		b.WriteString("\tr *http.Request\n")
	}
	b.WriteString("\t" + codegen.Comment("conn is the underlying websocket connection.") + "\n")
	b.WriteString("\tconn *websocket.Conn\n")
	if ws.Endpoint.Method.ViewedResult != nil && ws.Endpoint.Method.ViewedResult.ViewName == "" {
		fmt.Fprintf(&b, "\t%s\n", codegen.Comment(fmt.Sprintf("view is the view to render %s result type before sending to the websocket connection.", ws.SendTypeName)))
		b.WriteString("\tview string\n")
	}
	b.WriteString("}\n")
	prefix := ws.Type
	return codegen.NewRawSection(prefix+"-websocket-struct-type", b.String())
}

func websocketSendSection(ws *WebSocketData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(ws.SendDesc))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) %s(v %s) error {\n", ws.VarName, ws.SendName, ws.SendTypeRef)
	if ws.Type != "server" {
		writeClientWebSocketSend(&b, ws)
		return websocketSendWithContextSection(&b, ws)
	}
	writeServerWebSocketSend(&b, ws)
	return websocketSendWithContextSection(&b, ws)
}

func writeClientWebSocketSend(b *strings.Builder, ws *WebSocketData) {
	if ws.Payload != nil && ws.Payload.Init != nil {
		fmt.Fprintf(b, "\tbody := %s(v)\n", ws.Payload.Init.Name)
		b.WriteString("\treturn s.conn.WriteJSON(body)\n")
	} else {
		b.WriteString("\treturn s.conn.WriteJSON(v)\n")
	}
}

func writeServerWebSocketSend(b *strings.Builder, ws *WebSocketData) {
	writeServerWebSocketSendPreamble(b, ws)
	writeServerWebSocketSendResult(b, ws)
	if !writeServerWebSocketResponseBody(b, ws) {
		b.WriteString("\treturn s.conn.WriteJSON(res)\n")
	}
}

func writeServerWebSocketSendPreamble(b *strings.Builder, ws *WebSocketData) {
	if ws.SendName == "Send" {
		b.WriteString("\tvar err error\n")
		b.WriteString(renderWebsocketUpgrade(ws.Endpoint, ws.SendName, false))
		return
	}
	b.WriteString("\tdefer s.conn.Close()\n")
}

func writeServerWebSocketSendResult(b *strings.Builder, ws *WebSocketData) {
	if ws.Endpoint.Method.ViewedResult == nil {
		b.WriteString("\tres := v\n")
		return
	}
	if ws.Endpoint.Method.ViewedResult.ViewName != "" {
		fmt.Fprintf(b, "\tres := %s.%s(v, %q)\n", ws.PkgName, ws.Endpoint.Method.ViewedResult.Init.Name, ws.Endpoint.Method.ViewedResult.ViewName)
		return
	}
	fmt.Fprintf(b, "\tres := %s.%s(v, s.view)\n", ws.PkgName, ws.Endpoint.Method.ViewedResult.Init.Name)
}

func writeServerWebSocketResponseBody(b *strings.Builder, ws *WebSocketData) bool {
	if len(ws.Response.ServerBody) == 0 {
		return false
	}
	body := ws.Response.ServerBody[0]
	if body.Init == nil {
		return false
	}
	writeServerWebSocketBodyInit(b, ws, body)
	b.WriteString("\treturn s.conn.WriteJSON(body)\n")
	return true
}

func writeServerWebSocketBodyInit(b *strings.Builder, ws *WebSocketData, body *TypeData) {
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
	b.WriteString("\tvar body any\n")
	b.WriteString("\tswitch s.view {\n")
	for _, view := range ws.Endpoint.Method.ViewedResult.Views {
		writeViewedServerBodyCase(b, ws, view.Name)
	}
	b.WriteString("\t}\n")
}

func writeViewedServerBodyCase(b *strings.Builder, ws *WebSocketData, viewName string) {
	if viewName == "default" {
		fmt.Fprintf(b, "\tcase %q, \"\":\n", viewName)
	} else {
		fmt.Fprintf(b, "\tcase %q:\n", viewName)
	}
	if vsb := viewedServerBody(ws.Response.ServerBody, viewName); vsb != nil {
		writeServerBodyInitCall(b, vsb, "\t\tbody = ")
	}
}

func writeServerBodyInitCall(b *strings.Builder, body *TypeData, prefix string) {
	fmt.Fprintf(b, "%s%s(", prefix, body.Init.Name)
	for _, arg := range body.Init.ServerArgs {
		fmt.Fprintf(b, "%s, ", arg.Ref)
	}
	b.WriteString(")\n")
}

func websocketSendWithContextSection(b *strings.Builder, ws *WebSocketData) codegen.Section {
	b.WriteString("}\n\n")
	b.WriteString(codegen.Comment(ws.SendWithContextDesc))
	b.WriteString("\n")
	fmt.Fprintf(b, "func (s *%s) %s(ctx context.Context, v %s) error {\n", ws.VarName, ws.SendWithContextName, ws.SendTypeRef)
	fmt.Fprintf(b, "\treturn s.%s(v)\n", ws.SendName)
	b.WriteString("}\n")
	return codegen.NewRawSection(ws.Type+"-websocket-send", b.String())
}

func websocketRecvSection(ws *WebSocketData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(ws.RecvDesc))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) %s() (%s, error) {\n", ws.VarName, ws.RecvName, ws.RecvTypeRef)
	writeWebsocketRecvVars(&b, ws)
	if ws.Type == "server" {
		writeServerWebsocketRecvBody(&b, ws)
	} else {
		writeClientWebsocketRecvBody(&b, ws)
	}
	b.WriteString("}\n\n")
	b.WriteString(codegen.Comment(ws.RecvWithContextDesc))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) %s(ctx context.Context) (%s, error) {\n", ws.VarName, ws.RecvWithContextName, ws.RecvTypeRef)
	fmt.Fprintf(&b, "\treturn s.%s()\n", ws.RecvName)
	b.WriteString("}\n")
	return codegen.NewRawSection(ws.Type+"-websocket-recv", b.String())
}

func writeWebsocketRecvVars(b *strings.Builder, ws *WebSocketData) {
	b.WriteString("\tvar (\n")
	fmt.Fprintf(b, "\t\trv %s\n", ws.RecvTypeRef)
	if ws.Type == "server" {
		if ws.RecvTypeIsPointer {
			fmt.Fprintf(b, "\t\tbody %s\n", ws.Payload.VarName)
		} else {
			fmt.Fprintf(b, "\t\tmsg *%s\n", ws.Payload.VarName)
		}
	} else {
		bodyTypeRef := ws.RecvTypeRef
		if ws.Response != nil && ws.Response.ClientBody != nil {
			bodyTypeRef = ws.Response.ClientBody.VarName
		}
		fmt.Fprintf(b, "\t\tbody %s\n", bodyTypeRef)
	}
	b.WriteString("\t\terr error\n")
	b.WriteString("\t)\n")
}

func writeServerWebsocketRecvBody(b *strings.Builder, ws *WebSocketData) {
	b.WriteString(renderWebsocketUpgrade(ws.Endpoint, ws.RecvName, true))
	if ws.RecvTypeIsPointer {
		b.WriteString("\tif err = s.conn.ReadJSON(&body); err != nil {\n")
	} else {
		b.WriteString("\tif err = s.conn.ReadJSON(&msg); err != nil {\n")
	}
	b.WriteString("\t\treturn rv, err\n")
	b.WriteString("\t}\n")
	if ws.RecvTypeIsPointer {
		b.WriteString("\tif body == nil {\n")
	} else {
		b.WriteString("\tif msg == nil {\n")
	}
	b.WriteString("\t\treturn rv, io.EOF\n")
	b.WriteString("\t}\n")
	writeServerWebsocketRecvValidation(b, ws)
	writeServerWebsocketRecvReturn(b, ws)
}

func writeServerWebsocketRecvValidation(b *strings.Builder, ws *WebSocketData) {
	if ws.Payload == nil || ws.Payload.ValidateRef == "" {
		return
	}
	if !ws.RecvTypeIsPointer {
		b.WriteString("\tbody := *msg\n")
	}
	fmt.Fprintf(b, "\t%s\n", ws.Payload.ValidateRef)
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn rv, err\n")
	b.WriteString("\t}\n")
}

func writeServerWebsocketRecvReturn(b *strings.Builder, ws *WebSocketData) {
	switch {
	case ws.Payload != nil && ws.Payload.Init != nil:
		if ws.RecvTypeIsPointer {
			fmt.Fprintf(b, "\treturn %s(body), nil\n", ws.Payload.Init.Name)
		} else {
			fmt.Fprintf(b, "\treturn %s(msg), nil\n", ws.Payload.Init.Name)
		}
	case ws.RecvTypeIsPointer:
		b.WriteString("\treturn body, nil\n")
	default:
		b.WriteString("\treturn *msg, nil\n")
	}
}

func writeClientWebsocketRecvBody(b *strings.Builder, ws *WebSocketData) {
	if ws.RecvName == "CloseAndRecv" {
		b.WriteString("\tdefer s.conn.Close()\n")
		b.WriteString("\t// Send a nil payload to the server implying end of message\n")
		b.WriteString("\tif err = s.conn.WriteJSON(nil); err != nil {\n")
		b.WriteString("\t\treturn rv, err\n")
		b.WriteString("\t}\n")
	}
	b.WriteString("\terr = s.conn.ReadJSON(&body)\n")
	b.WriteString("\tif websocket.IsCloseError(err, websocket.CloseNormalClosure) {\n")
	if !ws.MustClose {
		b.WriteString("\t\ts.conn.Close()\n")
	}
	b.WriteString("\t\treturn rv, io.EOF\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn rv, err\n")
	b.WriteString("\t}\n")
	writeClientWebsocketRecvValidation(b, ws)
	writeClientWebsocketRecvReturn(b, ws)
}

func writeClientWebsocketRecvValidation(b *strings.Builder, ws *WebSocketData) {
	if ws.Response.ClientBody == nil || ws.Response.ClientBody.ValidateRef == "" || ws.Endpoint.Method.ViewedResult != nil {
		return
	}
	fmt.Fprintf(b, "\t%s\n", ws.Response.ClientBody.ValidateRef)
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn rv, err\n")
	b.WriteString("\t}\n")
}

func writeClientWebsocketRecvReturn(b *strings.Builder, ws *WebSocketData) {
	if ws.Response.ResultInit == nil {
		b.WriteString("\treturn body, nil\n")
		return
	}
	b.WriteString("\tres := ")
	fmt.Fprintf(b, "%s(", ws.Response.ResultInit.Name)
	for _, arg := range ws.Response.ResultInit.ClientArgs {
		fmt.Fprintf(b, "%s,", arg.Ref)
	}
	b.WriteString(")\n")
	if ws.Endpoint.Method.ViewedResult == nil {
		b.WriteString("\treturn res, nil\n")
		return
	}
	writeClientWebsocketViewedResultReturn(b, ws)
}

func writeClientWebsocketViewedResultReturn(b *strings.Builder, ws *WebSocketData) {
	view := ws.Endpoint.Method.ViewedResult
	prefix := ""
	if !view.IsCollection {
		prefix = "&"
	}
	viewArg := fmt.Sprintf("%q", view.ViewName)
	if view.ViewName == "" {
		viewArg = "s.view"
	}
	fmt.Fprintf(b, "\tvres := %s%s.%s{res, %s }\n", prefix, view.ViewsPkg, view.VarName, viewArg)
	fmt.Fprintf(b, "\tif err := %s.Validate%s(vres); err != nil {\n", view.ViewsPkg, ws.Endpoint.Method.Result)
	fmt.Fprintf(b, "\t\treturn rv, loomhttp.ErrValidationError(%q, %q, err)\n", ws.Endpoint.ServiceName, ws.Endpoint.Method.Name)
	b.WriteString("\t}\n")
	fmt.Fprintf(b, "\treturn %s.%s(vres), nil\n", ws.PkgName, view.ResultInit.Name)
}

func websocketCloseSection(ws *WebSocketData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("Close closes the %q endpoint websocket connection.", ws.Endpoint.Method.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) Close() error {\n", ws.VarName)
	b.WriteString("\tvar err error\n")
	if ws.Type == "server" {
		b.WriteString("\tif s.conn == nil {\n")
		b.WriteString("\t\treturn nil\n")
		b.WriteString("\t}\n")
		b.WriteString("\tif err = s.conn.WriteControl(\n")
		b.WriteString("\t\twebsocket.CloseMessage,\n")
		b.WriteString("\t\twebsocket.FormatCloseMessage(websocket.CloseNormalClosure, \"server closing connection\"),\n")
		b.WriteString("\t\ttime.Now().Add(time.Second),\n")
		b.WriteString("\t); err != nil {\n")
		b.WriteString("\t\treturn err\n")
		b.WriteString("\t}\n")
	} else {
		b.WriteString("\t// Send a nil payload to the server implying client closing connection.\n")
		b.WriteString("\tif err = s.conn.WriteJSON(nil); err != nil {\n")
		b.WriteString("\t\treturn err\n")
		b.WriteString("\t}\n")
	}
	b.WriteString("\treturn s.conn.Close()\n")
	b.WriteString("}\n")
	return codegen.NewRawSection(ws.Type+"-websocket-close", b.String())
}

func websocketSetViewSection(ws *WebSocketData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("SetView sets the view to render the %s type before sending to the %q endpoint websocket connection.", ws.SendTypeName, ws.Endpoint.Method.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) SetView(view string) {\n", ws.VarName)
	b.WriteString("\ts.view = view\n")
	b.WriteString("}\n")
	return codegen.NewRawSection(ws.Type+"-websocket-set-view", b.String())
}

func renderWebsocketUpgrade(endpoint *EndpointData, function string, recv bool) string {
	var b strings.Builder
	b.WriteString("\t")
	b.WriteString(codegen.Comment(fmt.Sprintf("Upgrade the HTTP connection to a websocket connection only once. Connection upgrade is done here so that authorization logic in the endpoint is executed before calling the actual service method which may call %s().", function)))
	b.WriteString("\n")
	b.WriteString("\ts.once.Do(func() {\n")
	if endpoint.Method.ViewedResult != nil && function == "Send" && endpoint.Method.ViewedResult.ViewName == "" {
		b.WriteString("\t\trespHdr := make(http.Header)\n")
		b.WriteString("\t\trespHdr.Add(\"loom-view\", s.view)\n")
	}
	b.WriteString("\t\tvar conn *websocket.Conn\n")
	if function == "Send" && endpoint.Method.ViewedResult != nil && endpoint.Method.ViewedResult.ViewName == "" {
		b.WriteString("\t\tconn, err = s.upgrader.Upgrade(s.w, s.r, respHdr)\n")
	} else {
		b.WriteString("\t\tconn, err = s.upgrader.Upgrade(s.w, s.r, nil)\n")
	}
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\ts.upgradeErr = err\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif s.configurer != nil {\n")
	b.WriteString("\t\t\tconn = s.configurer(conn, s.cancel)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\ts.conn = conn\n")
	b.WriteString("\t})\n")
	b.WriteString("\tif s.upgradeErr != nil {\n")
	if recv {
		b.WriteString("\t\treturn rv, s.upgradeErr\n")
	} else {
		b.WriteString("\t\treturn s.upgradeErr\n")
	}
	b.WriteString("\t}\n")
	return b.String()
}

func buildStreamRequestSection(endpoint *EndpointData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s creates a streaming endpoint request payload from the method payload and the path to the file to be streamed", endpoint.BuildStreamPayload)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(", endpoint.BuildStreamPayload)
	if endpoint.Payload.Ref != "" {
		b.WriteString("payload any, ")
	}
	fmt.Fprintf(&b, "fpath string) (*%s.%s, error) {\n", requestStructPkg(endpoint.Method, endpoint.ServicePkgName), endpoint.Method.RequestStruct)
	b.WriteString("\tf, err := os.Open(fpath)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn nil, err\n")
	b.WriteString("\t}\n")
	fmt.Fprintf(&b, "\treturn &%s.%s{\n", requestStructPkg(endpoint.Method, endpoint.ServicePkgName), endpoint.Method.RequestStruct)
	if endpoint.Payload.Ref != "" {
		fmt.Fprintf(&b, "\t\tPayload: payload.(%s),\n", endpoint.Payload.Ref)
	}
	b.WriteString("\t\tBody: f,\n")
	b.WriteString("\t}, nil\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("build-stream-request", b.String())
}

func multipartRequestEncoderTypeSection(data *MultipartData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s is the type to encode multipart request for the %q service %q endpoint.", data.FuncName, data.ServiceName, data.MethodName)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s func(*multipart.Writer, %s) error\n", data.FuncName, data.Payload.Ref)
	return codegen.NewRawSection("multipart-request-encoder-type", b.String())
}
