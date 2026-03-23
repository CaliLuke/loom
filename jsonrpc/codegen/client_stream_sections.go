package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcSSEClientStreamSection(ed *httpcodegen.EndpointData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-sse-client-stream", renderJSONRPCSSEClientStream(ed))
}

func renderJSONRPCSSEClientStream(ed *httpcodegen.EndpointData) string {
	var b strings.Builder
	b.WriteString(renderJSONRPCSSEClientStreamType(ed))
	b.WriteString(renderJSONRPCSSERecv(ed))
	if ed.Method.Result != "" {
		b.WriteString(renderJSONRPCSSEDecodeResult(ed))
	}
	b.WriteString(renderJSONRPCSSEClose(ed))
	return b.String()
}

func renderJSONRPCSSEClientStreamType(ed *httpcodegen.EndpointData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("%sClientStream implements the %s.%sClientStream interface using Server-Sent Events.", ed.Method.VarName, ed.ServicePkgName, ed.Method.VarName)))
	fmt.Fprintf(&b, "type %sClientStream struct {\n", ed.Method.VarName)
	b.WriteString("\tresp    *http.Response\n")
	b.WriteString("\treader  *bufio.Reader\n")
	b.WriteString("\tdecoder func(*http.Response) loomhttp.Decoder\n")
	b.WriteString("\tclosed  bool\n")
	b.WriteString("\tlock    sync.Mutex\n")
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "func (s *%sClientStream) readSSEEvent() ([]byte, error) {\n", ed.Method.VarName)
	b.WriteString("\tvar event bytes.Buffer\n\n")
	b.WriteString("\tfor {\n")
	b.WriteString("\t\tline, err := s.reader.ReadString('\\n')\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\tif err == io.EOF && event.Len() > 0 {\n")
	b.WriteString("\t\t\t\treturn event.Bytes(), nil\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\treturn nil, err\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tevent.WriteString(line)\n\n")
	b.WriteString("\t\tline = strings.TrimRight(line, \"\\r\\n\")\n")
	b.WriteString("\t\tif line == \"\" {\n")
	b.WriteString("\t\t\tif event.Len() > 0 {\n")
	b.WriteString("\t\t\t\treturn event.Bytes(), nil\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\tcontinue\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderJSONRPCSSERecv(ed *httpcodegen.EndpointData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment(ed.Method.ClientStream.RecvDesc))
	fmt.Fprintf(&b, "func (s *%sClientStream) %s(ctx context.Context) (%s, error) {\n", ed.Method.VarName, ed.Method.ClientStream.RecvName, ed.Result.Ref)
	b.WriteString("\ts.lock.Lock()\n")
	b.WriteString("\tdefer s.lock.Unlock()\n\n")
	fmt.Fprintf(&b, "\tvar zero %s\n\n", ed.Result.Ref)
	b.WriteString("\tif s.closed {\n")
	b.WriteString("\t\treturn zero, io.EOF\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tfor {\n")
	b.WriteString("\t\trawEvent, err := s.readSSEEvent()\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\ts.closed = true\n")
	b.WriteString("\t\t\treturn zero, err\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\tparsedEvent, err := loomhttp.ParseSSEEvent(rawEvent)\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\ts.closed = true\n")
	b.WriteString("\t\t\treturn zero, err\n")
	b.WriteString("\t\t}\n\n")
	b.WriteString("\t\teventType, data := parsedEvent.Type, []byte(parsedEvent.Data)\n\n")
	b.WriteString("\t\tswitch eventType {\n")
	b.WriteString("\t\tcase \"notification\":\n")
	b.WriteString("\t\t\tvar notification struct {\n")
	b.WriteString("\t\t\t\tJSONRPC string          `json:\"jsonrpc\"`\n")
	b.WriteString("\t\t\t\tMethod  string          `json:\"method\"`\n")
	b.WriteString("\t\t\t\tParams  json.RawMessage `json:\"params\"`\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\tif err := json.Unmarshal(data, &notification); err != nil {\n")
	b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"failed to parse notification: %w\", err)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\tif notification.JSONRPC != \"2.0\" {\n")
	b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"invalid JSON-RPC version: %s\", notification.JSONRPC)\n")
	b.WriteString("\t\t\t}\n")
	fmt.Fprintf(&b, "\t\t\tif notification.Method != %q {\n", ed.Method.Name)
	b.WriteString("\t\t\t\tcontinue\n")
	b.WriteString("\t\t\t}\n")
	if ed.Method.Result != "" {
		b.WriteString("\t\t\tresult, err := s.decodeResult(notification.Params)\n")
		b.WriteString("\t\t\tif err != nil {\n")
		b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"failed to decode result: %w\", err)\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t\treturn result, nil\n")
	} else {
		b.WriteString("\t\t\treturn zero, nil\n")
	}
	b.WriteString("\t\tcase \"response\":\n")
	b.WriteString("\t\t\tvar response jsonrpc.Response\n")
	b.WriteString("\t\t\tif err := json.Unmarshal(data, &response); err != nil {\n")
	b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"failed to parse response: %w\", err)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\tif response.Error != nil {\n")
	b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"JSON-RPC error %d: %s\", response.Error.Code, response.Error.Message)\n")
	b.WriteString("\t\t\t}\n")
	if ed.Method.Result != "" {
		b.WriteString("\t\t\tif response.Result == nil {\n")
		b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"missing result in response\")\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t\tresultBytes, err := json.Marshal(response.Result)\n")
		b.WriteString("\t\t\tif err != nil {\n")
		b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"failed to marshal result: %w\", err)\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t\tresult, err := s.decodeResult(json.RawMessage(resultBytes))\n")
		b.WriteString("\t\t\tif err != nil {\n")
		b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"failed to decode final result: %w\", err)\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t\ts.closed = true\n")
		b.WriteString("\t\t\treturn result, nil\n")
	} else {
		b.WriteString("\t\t\ts.closed = true\n")
		b.WriteString("\t\t\treturn zero, nil\n")
	}
	b.WriteString("\t\tcase \"error\":\n")
	b.WriteString("\t\t\tvar response jsonrpc.Response\n")
	b.WriteString("\t\t\tif err := json.Unmarshal(data, &response); err != nil {\n")
	b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"failed to parse error response: %w\", err)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\ts.closed = true\n")
	b.WriteString("\t\t\tif response.Error != nil {\n")
	b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"JSON-RPC error %d: %s\", response.Error.Code, response.Error.Message)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\treturn zero, fmt.Errorf(\"unexpected error response\")\n")
	b.WriteString("\t\tcase \"\", \"message\":\n")
	b.WriteString(renderJSONRPCSSEMessageCase(ed))
	b.WriteString("\t\tdefault:\n")
	b.WriteString("\t\t\tcontinue\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderJSONRPCSSEMessageCase(ed *httpcodegen.EndpointData) string {
	var b strings.Builder
	b.WriteString("\t\t\tvar envelope map[string]json.RawMessage\n")
	b.WriteString("\t\t\tif err := json.Unmarshal(data, &envelope); err != nil {\n")
	b.WriteString("\t\t\t\treturn zero, fmt.Errorf(\"failed to parse message event: %w\", err)\n")
	b.WriteString("\t\t\t}\n")
	b.WriteString("\t\t\tif _, ok := envelope[\"method\"]; ok {\n")
	b.WriteString(renderJSONRPCSSENotificationBranch(ed, "\t\t\t\t", true))
	b.WriteString("\t\t\t}\n")
	b.WriteString(renderJSONRPCSSEResponseBranch(ed, "\t\t\t", true))
	return b.String()
}

func renderJSONRPCSSENotificationBranch(ed *httpcodegen.EndpointData, indent string, continueOnMismatch bool) string {
	var b strings.Builder
	b.WriteString(indent + "var notification struct {\n")
	b.WriteString(indent + "\tJSONRPC string          `json:\"jsonrpc\"`\n")
	b.WriteString(indent + "\tMethod  string          `json:\"method\"`\n")
	b.WriteString(indent + "\tParams  json.RawMessage `json:\"params\"`\n")
	b.WriteString(indent + "}\n")
	b.WriteString(indent + "if err := json.Unmarshal(data, &notification); err != nil {\n")
	b.WriteString(indent + "\treturn zero, fmt.Errorf(\"failed to parse notification: %w\", err)\n")
	b.WriteString(indent + "}\n")
	b.WriteString(indent + "if notification.JSONRPC != \"2.0\" {\n")
	b.WriteString(indent + "\treturn zero, fmt.Errorf(\"invalid JSON-RPC version: %s\", notification.JSONRPC)\n")
	b.WriteString(indent + "}\n")
	fmt.Fprintf(&b, "%sif notification.Method != %q {\n", indent, ed.Method.Name)
	if continueOnMismatch {
		b.WriteString(indent + "\tcontinue\n")
	} else {
		b.WriteString(indent + "\treturn zero, nil\n")
	}
	b.WriteString(indent + "}\n")
	if ed.Method.Result != "" {
		b.WriteString(indent + "result, err := s.decodeResult(notification.Params)\n")
		b.WriteString(indent + "if err != nil {\n")
		b.WriteString(indent + "\treturn zero, fmt.Errorf(\"failed to decode result: %w\", err)\n")
		b.WriteString(indent + "}\n")
		b.WriteString(indent + "return result, nil\n")
	} else {
		b.WriteString(indent + "return zero, nil\n")
	}
	return b.String()
}

func renderJSONRPCSSEResponseBranch(ed *httpcodegen.EndpointData, indent string, closeOnSuccess bool) string {
	var b strings.Builder
	b.WriteString(indent + "var response jsonrpc.Response\n")
	b.WriteString(indent + "if err := json.Unmarshal(data, &response); err != nil {\n")
	b.WriteString(indent + "\treturn zero, fmt.Errorf(\"failed to parse response: %w\", err)\n")
	b.WriteString(indent + "}\n")
	b.WriteString(indent + "if response.Error != nil {\n")
	if closeOnSuccess {
		b.WriteString(indent + "\ts.closed = true\n")
	}
	b.WriteString(indent + "\treturn zero, fmt.Errorf(\"JSON-RPC error %d: %s\", response.Error.Code, response.Error.Message)\n")
	b.WriteString(indent + "}\n")
	if ed.Method.Result != "" {
		b.WriteString(indent + "if response.Result == nil {\n")
		b.WriteString(indent + "\treturn zero, fmt.Errorf(\"missing result in response\")\n")
		b.WriteString(indent + "}\n")
		b.WriteString(indent + "resultBytes, err := json.Marshal(response.Result)\n")
		b.WriteString(indent + "if err != nil {\n")
		b.WriteString(indent + "\treturn zero, fmt.Errorf(\"failed to marshal result: %w\", err)\n")
		b.WriteString(indent + "}\n")
		b.WriteString(indent + "result, err := s.decodeResult(json.RawMessage(resultBytes))\n")
		b.WriteString(indent + "if err != nil {\n")
		b.WriteString(indent + "\treturn zero, fmt.Errorf(\"failed to decode final result: %w\", err)\n")
		b.WriteString(indent + "}\n")
	}
	if closeOnSuccess {
		b.WriteString(indent + "s.closed = true\n")
	}
	if ed.Method.Result != "" {
		b.WriteString(indent + "return result, nil\n")
	} else {
		b.WriteString(indent + "return zero, nil\n")
	}
	return b.String()
}

func renderJSONRPCSSEDecodeResult(ed *httpcodegen.EndpointData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "func (s *%sClientStream) decodeResult(data json.RawMessage) (%s, error) {\n", ed.Method.VarName, ed.Result.Ref)
	b.WriteString("\tresp := &http.Response{\n")
	b.WriteString("\t\tStatusCode: http.StatusOK,\n")
	b.WriteString("\t\tBody:       io.NopCloser(bytes.NewReader(data)),\n")
	b.WriteString("\t}\n")
	b.WriteString("\tdecoder := s.decoder(resp)\n")
	fmt.Fprintf(&b, "\tvar result %s\n", ed.Result.Ref)
	b.WriteString("\tif err := decoder.Decode(&result); err != nil {\n")
	b.WriteString("\t\treturn result, err\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn result, nil\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderJSONRPCSSEClose(ed *httpcodegen.EndpointData) string {
	var b strings.Builder
	b.WriteString(codegen.Comment("Close closes the stream."))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%sClientStream) Close() error {\n", ed.Method.VarName)
	b.WriteString("\ts.lock.Lock()\n")
	b.WriteString("\tdefer s.lock.Unlock()\n\n")
	b.WriteString("\tif !s.closed {\n")
	b.WriteString("\t\ts.closed = true\n")
	b.WriteString("\t\tif s.resp != nil && s.resp.Body != nil {\n")
	b.WriteString("\t\t\treturn s.resp.Body.Close()\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn nil\n")
	b.WriteString("}\n")
	return b.String()
}

func jsonrpcWebSocketClientStreamSection(ws *httpcodegen.WebSocketData) codegen.Section {
	return codegen.NewRawSection("jsonrpc-websocket-client-stream", renderJSONRPCWebSocketClientStream(ws))
}

func renderJSONRPCWebSocketClientStream(ws *httpcodegen.WebSocketData) string {
	var b strings.Builder
	hasRecv := ws.RecvName != "" && ws.RecvTypeRef != ""
	hasSend := ws.SendName != ""
	isBidirectional := hasSend && hasRecv
	b.WriteString(renderJSONRPCWebSocketClientTypes(ws, hasRecv))
	b.WriteString(renderJSONRPCWebSocketSend(ws, isBidirectional))
	b.WriteString(renderJSONRPCWebSocketRecv(ws, hasRecv, isBidirectional))
	b.WriteString(renderJSONRPCWebSocketClientHelpers(ws, hasRecv))
	return b.String()
}

func renderJSONRPCWebSocketClientTypes(ws *httpcodegen.WebSocketData, hasRecv bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n", codegen.Comment(fmt.Sprintf("%s implements the %s client stream with direct WebSocket handling.", ws.VarName, ws.Endpoint.Method.Name)))
	fmt.Fprintf(&b, "type %s struct {\n", ws.VarName)
	b.WriteString("\tws          *websocket.Conn\n")
	b.WriteString("\twriteMu     sync.Mutex\n")
	b.WriteString("\tpending     sync.Map\n")
	b.WriteString("\tidGenerator atomic.Uint64\n")
	b.WriteString("\tctx         context.Context\n")
	b.WriteString("\tcancel      context.CancelFunc\n")
	b.WriteString("\tdone        chan struct{}\n")
	b.WriteString("\tcloseOnce   sync.Once\n")
	b.WriteString("\terrorOnce   sync.Once\n")
	b.WriteString("\tlastError   atomic.Value\n")
	b.WriteString("\tconfig *jsonrpc.StreamConfig\n")
	if hasRecv {
		b.WriteString("\tdecoder        func(*http.Response) loomhttp.Decoder\n")
	}
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "type %sPendingRequest struct {\n", ws.VarName)
	b.WriteString("\tuserID      string\n")
	fmt.Fprintf(&b, "\tresultChan  chan %sStreamResult\n", ws.VarName)
	b.WriteString("\ttimeout     *time.Timer\n")
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "type %sStreamResult struct {\n", ws.VarName)
	if hasRecv {
		fmt.Fprintf(&b, "\tresult      %s\n", ws.RecvTypeRef)
	}
	b.WriteString("\terr         error\n")
	b.WriteString("}\n\n")
	return b.String()
}

func renderJSONRPCWebSocketSend(ws *httpcodegen.WebSocketData, isBidirectional bool) string {
	if ws.SendName == "" {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment(fmt.Sprintf("%s sends streaming data to the %s endpoint with dual ID correlation.", ws.SendName, ws.Endpoint.Method.Name)))
	fmt.Fprintf(&b, "func (s *%s) %s(v %s) error {\n", ws.VarName, ws.SendName, ws.SendTypeRef)
	fmt.Fprintf(&b, "\treturn s.%s(s.ctx, v)\n", ws.SendWithContextName)
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "%s\n", codegen.Comment(fmt.Sprintf("%s sends streaming data to the %s endpoint with context.", ws.SendWithContextName, ws.Endpoint.Method.Name)))
	fmt.Fprintf(&b, "func (s *%s) %s(ctx context.Context, v %s) error {\n", ws.VarName, ws.SendWithContextName, ws.SendTypeRef)
	b.WriteString("\tif err := s.getError(); err != nil {\n\t\treturn err\n\t}\n")
	if isBidirectional {
		b.WriteString(renderJSONRPCWebSocketBidirectionalSend(ws))
	} else {
		b.WriteString(renderJSONRPCWebSocketSimpleSend(ws))
	}
	b.WriteString("}\n\n")
	return b.String()
}

func renderJSONRPCWebSocketBidirectionalSend(ws *httpcodegen.WebSocketData) string {
	var b strings.Builder
	b.WriteString("\tuserID := s.generateUserID()\n")
	b.WriteString("\tjsonrpcID := strconv.FormatUint(s.idGenerator.Add(1), 10)\n")
	fmt.Fprintf(&b, "\tpending := &%sPendingRequest{\n", ws.VarName)
	b.WriteString("\t\tuserID:     userID,\n")
	fmt.Fprintf(&b, "\t\tresultChan: make(chan %sStreamResult, s.config.ResultChannelBuffer),\n", ws.VarName)
	b.WriteString("\t\ttimeout:    time.NewTimer(s.config.RequestTimeout),\n")
	b.WriteString("\t}\n")
	b.WriteString("\ts.pending.Store(jsonrpcID, pending)\n")
	b.WriteString(renderJSONRPCWriteRequest(ws, true, "v"))
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\ts.pending.Delete(jsonrpcID)\n")
	b.WriteString("\t\tpending.timeout.Stop()\n")
	b.WriteString("\t\ts.setError(err)\n")
	b.WriteString("\t\ts.handleError(jsonrpc.StreamErrorConnection, err, nil)\n")
	b.WriteString("\t\treturn fmt.Errorf(\"failed to send request: %w\", err)\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn nil\n")
	return b.String()
}

func renderJSONRPCWebSocketSimpleSend(ws *httpcodegen.WebSocketData) string {
	var b strings.Builder
	b.WriteString(renderJSONRPCWriteRequest(ws, false, "v"))
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\ts.setError(err)\n")
	b.WriteString("\t\ts.handleError(jsonrpc.StreamErrorConnection, err, nil)\n")
	b.WriteString("\t\treturn fmt.Errorf(\"failed to send request: %w\", err)\n")
	b.WriteString("\t}\n")
	b.WriteString("\treturn nil\n")
	return b.String()
}

func renderJSONRPCWriteRequest(ws *httpcodegen.WebSocketData, includeID bool, paramsExpr string) string {
	var b strings.Builder
	b.WriteString("\trequest := &jsonrpc.Request{\n")
	b.WriteString("\t\tJSONRPC: \"2.0\",\n")
	fmt.Fprintf(&b, "\t\tMethod:  %q,\n", ws.Endpoint.Method.Name)
	fmt.Fprintf(&b, "\t\tParams:  %s,\n", paramsExpr)
	if includeID {
		b.WriteString("\t\tID:      &jsonrpcID,\n")
	}
	b.WriteString("\t}\n")
	b.WriteString("\ts.writeMu.Lock()\n")
	b.WriteString("\terr := s.ws.WriteJSON(request)\n")
	b.WriteString("\ts.writeMu.Unlock()\n")
	return b.String()
}

func renderJSONRPCWebSocketRecv(ws *httpcodegen.WebSocketData, hasRecv, isBidirectional bool) string {
	if !hasRecv {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", codegen.Comment(fmt.Sprintf("%s receives streaming data from the %s endpoint.", ws.RecvName, ws.Endpoint.Method.Name)))
	fmt.Fprintf(&b, "func (s *%s) %s() (%s, error) {\n", ws.VarName, ws.RecvName, ws.RecvTypeRef)
	fmt.Fprintf(&b, "\treturn s.%s(s.ctx)\n", ws.RecvWithContextName)
	b.WriteString("}\n\n")
	fmt.Fprintf(&b, "%s\n", codegen.Comment(fmt.Sprintf("%s receives streaming data from the %s endpoint with context.", ws.RecvWithContextName, ws.Endpoint.Method.Name)))
	fmt.Fprintf(&b, "func (s *%s) %s(ctx context.Context) (%s, error) {\n", ws.VarName, ws.RecvWithContextName, ws.RecvTypeRef)
	fmt.Fprintf(&b, "\tvar zero %s\n", ws.RecvTypeRef)
	b.WriteString("\tif err := s.getError(); err != nil {\n\t\treturn zero, err\n\t}\n")
	if isBidirectional {
		b.WriteString(renderJSONRPCWebSocketBidirectionalRecv(ws))
	} else {
		b.WriteString(renderJSONRPCWebSocketSimpleRecv(ws))
	}
	b.WriteString("}\n\n")
	return b.String()
}

func renderJSONRPCWebSocketBidirectionalRecv(ws *httpcodegen.WebSocketData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\tvar oldestPending *%sPendingRequest\n", ws.VarName)
	b.WriteString("\tvar oldestKey string\n")
	b.WriteString("\ts.pending.Range(func(key, value any) bool {\n")
	fmt.Fprintf(&b, "\t\tpending := value.(*%sPendingRequest)\n", ws.VarName)
	b.WriteString("\t\tif oldestPending == nil {\n\t\t\toldestPending = pending\n\t\t\toldestKey = key.(string)\n\t\t}\n\t\treturn false\n\t})\n")
	b.WriteString("\tif oldestPending == nil {\n")
	fmt.Fprintf(&b, "\t\treturn zero, fmt.Errorf(%q)\n", fmt.Sprintf("no pending requests - call %s() first", ws.SendName))
	b.WriteString("\t}\n")
	b.WriteString("\tselect {\n")
	b.WriteString("\tcase result := <-oldestPending.resultChan:\n")
	b.WriteString("\t\ts.pending.Delete(oldestKey)\n\t\toldestPending.timeout.Stop()\n\t\treturn result.result, result.err\n")
	b.WriteString("\tcase <-oldestPending.timeout.C:\n")
	b.WriteString("\t\ts.pending.Delete(oldestKey)\n")
	b.WriteString("\t\ttimeoutErr := fmt.Errorf(\"request timeout after %v\", s.config.RequestTimeout)\n")
	b.WriteString("\t\ts.handleError(jsonrpc.StreamErrorTimeout, timeoutErr, nil)\n\t\treturn zero, timeoutErr\n")
	b.WriteString("\tcase <-ctx.Done():\n\t\treturn zero, ctx.Err()\n")
	b.WriteString("\tcase <-s.done:\n")
	b.WriteString("\t\tif err := s.getError(); err != nil {\n\t\t\treturn zero, err\n\t\t}\n")
	b.WriteString("\t\treturn zero, fmt.Errorf(\"stream closed\")\n")
	b.WriteString("\t}\n")
	return b.String()
}

func renderJSONRPCWebSocketSimpleRecv(ws *httpcodegen.WebSocketData) string {
	var b strings.Builder
	b.WriteString("\tjsonrpcID := strconv.FormatUint(s.idGenerator.Add(1), 10)\n")
	fmt.Fprintf(&b, "\tresultChan := make(chan %sStreamResult, s.config.ResultChannelBuffer)\n", ws.VarName)
	fmt.Fprintf(&b, "\tpending := &%sPendingRequest{\n", ws.VarName)
	b.WriteString("\t\tuserID:     jsonrpcID,\n\t\tresultChan: resultChan,\n\t\ttimeout:    time.NewTimer(s.config.RequestTimeout),\n\t}\n")
	b.WriteString("\ts.pending.Store(jsonrpcID, pending)\n")
	b.WriteString("\tdefer func() {\n\t\ts.pending.Delete(jsonrpcID)\n\t\tpending.timeout.Stop()\n\t}()\n")
	b.WriteString(renderJSONRPCWriteRequest(ws, true, "nil"))
	b.WriteString("\tif err != nil {\n\t\ts.setError(err)\n\t\ts.handleError(jsonrpc.StreamErrorConnection, err, nil)\n\t\treturn zero, fmt.Errorf(\"failed to send request: %w\", err)\n\t}\n")
	b.WriteString("\tselect {\n")
	b.WriteString("\tcase result := <-resultChan:\n\t\treturn result.result, result.err\n")
	b.WriteString("\tcase <-pending.timeout.C:\n")
	b.WriteString("\t\ttimeoutErr := fmt.Errorf(\"request timeout after %v\", s.config.RequestTimeout)\n")
	b.WriteString("\t\ts.handleError(jsonrpc.StreamErrorTimeout, timeoutErr, nil)\n\t\treturn zero, timeoutErr\n")
	b.WriteString("\tcase <-ctx.Done():\n\t\treturn zero, ctx.Err()\n")
	b.WriteString("\tcase <-s.done:\n")
	b.WriteString("\t\tif err := s.getError(); err != nil {\n\t\t\treturn zero, err\n\t\t}\n")
	b.WriteString("\t\treturn zero, fmt.Errorf(\"stream closed\")\n")
	b.WriteString("\t}\n")
	return b.String()
}

func renderJSONRPCWebSocketClientHelpers(ws *httpcodegen.WebSocketData, hasRecv bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "func (s *%s) responseHandler() {\n", ws.VarName)
	b.WriteString("\tdefer close(s.done)\n\tfor {\n\t\tselect {\n\t\tcase <-s.ctx.Done():\n")
	b.WriteString("\t\t\ts.cleanupPendingRequests(s.ctx.Err())\n\t\t\treturn\n\t\tdefault:\n")
	b.WriteString("\t\t\tvar response jsonrpc.RawResponse\n")
	b.WriteString("\t\t\tif err := s.ws.ReadJSON(&response); err != nil {\n")
	b.WriteString("\t\t\t\tconnectionErr := fmt.Errorf(\"failed to read response: %w\", err)\n")
	b.WriteString("\t\t\t\ts.setError(connectionErr)\n")
	b.WriteString("\t\t\t\ts.handleError(jsonrpc.StreamErrorConnection, connectionErr, nil)\n")
	b.WriteString("\t\t\t\ts.cleanupPendingRequests(connectionErr)\n\t\t\t\treturn\n\t\t\t}\n")
	b.WriteString("\t\t\ts.handleResponse(&response)\n\t\t}\n\t}\n}\n\n")
	fmt.Fprintf(&b, "func (s *%s) handleResponse(response *jsonrpc.RawResponse) {\n", ws.VarName)
	b.WriteString("\tif response.ID == nil {\n")
	b.WriteString("\t\tif s.config.ErrorHandler != nil {\n")
	b.WriteString("\t\t\ts.config.ErrorHandler(s.ctx, jsonrpc.StreamErrorNotification, fmt.Errorf(\"received server notification\"), response)\n")
	b.WriteString("\t\t}\n\t\treturn\n\t}\n")
	b.WriteString("\tjsonrpcID := response.ID\n")
	b.WriteString("\tpendingInterface, exists := s.pending.LoadAndDelete(jsonrpcID)\n")
	b.WriteString("\tif !exists {\n\t\ts.handleError(jsonrpc.StreamErrorOrphaned, fmt.Errorf(\"received response for unknown ID: %s\", jsonrpcID), response)\n\t\treturn\n\t}\n")
	fmt.Fprintf(&b, "\tpending := pendingInterface.(*%sPendingRequest)\n", ws.VarName)
	b.WriteString("\tpending.timeout.Stop()\n")
	fmt.Fprintf(&b, "\tvar result %sStreamResult\n", ws.VarName)
	b.WriteString("\tif response.Error != nil {\n")
	b.WriteString("\t\tresult.err = response.Error\n")
	b.WriteString("\t\ts.handleError(jsonrpc.StreamErrorProtocol, response.Error, response)\n")
	b.WriteString("\t} else {\n")
	if hasRecv {
		b.WriteString(renderJSONRPCWebSocketDecodeResponseSuccess(ws))
	}
	b.WriteString("\t}\n")
	b.WriteString("\tselect {\n\tcase pending.resultChan <- result:\n\tdefault:\n\t}\n}\n\n")
	fmt.Fprintf(&b, "func (s *%s) generateUserID() string {\n", ws.VarName)
	b.WriteString("\treturn fmt.Sprintf(\"user-%d-%d\", time.Now().UnixNano(), s.idGenerator.Load())\n}\n\n")
	fmt.Fprintf(&b, "func (s *%s) handleError(errorType jsonrpc.StreamErrorType, err error, response *jsonrpc.RawResponse) {\n", ws.VarName)
	b.WriteString("\tif s.config.ErrorHandler != nil {\n\t\ts.config.ErrorHandler(s.ctx, errorType, err, response)\n\t}\n}\n\n")
	if hasRecv {
		fmt.Fprintf(&b, "func (s *%s) decodeResponse(data json.RawMessage) (%s, error) {\n", ws.VarName, ws.RecvTypeRef)
		b.WriteString("\tresp := &http.Response{\n\t\tStatusCode: http.StatusOK,\n\t\tBody: io.NopCloser(bytes.NewReader(data)),\n\t}\n")
		b.WriteString("\tdec := s.decoder(resp)\n")
		fmt.Fprintf(&b, "\tvar out %s\n", ws.RecvTypeRef)
		b.WriteString("\tif err := dec.Decode(&out); err != nil {\n\t\treturn nil, err\n\t}\n")
		b.WriteString("\treturn out, nil\n}\n\n")
	}
	fmt.Fprintf(&b, "func (s *%s) setError(err error) {\n", ws.VarName)
	b.WriteString("\ts.errorOnce.Do(func() {\n\t\ts.lastError.Store(err)\n\t\ts.cancel()\n\t})\n}\n\n")
	fmt.Fprintf(&b, "func (s *%s) getError() error {\n", ws.VarName)
	b.WriteString("\tif err, ok := s.lastError.Load().(error); ok {\n\t\treturn err\n\t}\n\treturn nil\n}\n\n")
	fmt.Fprintf(&b, "func (s *%s) cleanupPendingRequests(err error) {\n", ws.VarName)
	b.WriteString("\ts.pending.Range(func(key, value any) bool {\n")
	fmt.Fprintf(&b, "\t\tpending := value.(*%sPendingRequest)\n", ws.VarName)
	b.WriteString("\t\tpending.timeout.Stop()\n")
	fmt.Fprintf(&b, "\t\tselect {\n\t\tcase pending.resultChan <- %sStreamResult{err: err}:\n\t\tdefault:\n\t\t}\n", ws.VarName)
	b.WriteString("\t\ts.pending.Delete(key)\n\t\treturn true\n\t})\n}\n\n")
	b.WriteString(codegen.Comment("Close closes the stream and cleans up resources."))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) Close() error {\n", ws.VarName)
	b.WriteString("\tvar err error\n")
	b.WriteString("\ts.closeOnce.Do(func() {\n\t\ts.cancel()\n")
	b.WriteString("\t\tselect {\n\t\tcase <-s.done:\n\t\tcase <-time.After(s.config.CloseTimeout):\n\t\t}\n")
	b.WriteString("\t\ts.cleanupPendingRequests(fmt.Errorf(\"stream closed\"))\n")
	b.WriteString("\t\tif s.ws != nil {\n\t\t\terr = s.ws.Close()\n\t\t}\n\t})\n\treturn err\n}\n")
	return b.String()
}

func renderJSONRPCWebSocketDecodeResponseSuccess(ws *httpcodegen.WebSocketData) string {
	var b strings.Builder
	b.WriteString("\t\tparsedResult, err := s.decodeResponse(response.Result)\n")
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\tresult.err = fmt.Errorf(\"failed to decode response: %w\", err)\n")
	b.WriteString("\t\t\ts.handleError(jsonrpc.StreamErrorParsing, err, response)\n")
	b.WriteString("\t\t} else {\n")
	if ws.Endpoint.Result.IDAttribute != "" {
		if ws.Endpoint.Result.IDAttributeRequired {
			fmt.Fprintf(&b, "\t\t\tif parsedResult.%s == \"\" {\n", ws.Endpoint.Result.IDAttribute)
			fmt.Fprintf(&b, "\t\t\t\tparsedResult.%s = jsonrpc.IDToString(response.ID)\n", ws.Endpoint.Result.IDAttribute)
			b.WriteString("\t\t\t}\n")
		} else {
			fmt.Fprintf(&b, "\t\t\tif parsedResult.%s == nil || *parsedResult.%s == \"\" {\n", ws.Endpoint.Result.IDAttribute, ws.Endpoint.Result.IDAttribute)
			b.WriteString("\t\t\t\tidCopy := jsonrpc.IDToString(response.ID)\n")
			fmt.Fprintf(&b, "\t\t\t\tparsedResult.%s = &idCopy\n", ws.Endpoint.Result.IDAttribute)
			b.WriteString("\t\t\t}\n")
		}
	}
	b.WriteString("\t\t\tresult.result = parsedResult\n")
	b.WriteString("\t\t}\n")
	return b.String()
}
