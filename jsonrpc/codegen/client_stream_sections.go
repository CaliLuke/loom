package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcSSEClientStreamSection(ed *httpcodegen.EndpointData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-sse-client-stream", func(stmt *jen.Statement) {
		writeJSONRPCSSEClientStreamType(stmt, ed)
		stmt.Line()
		writeJSONRPCSSERecv(stmt, ed)
		if ed.Method.Result != "" {
			stmt.Line()
			writeJSONRPCSSEDecodeResult(stmt, ed)
		}
		stmt.Line()
		writeJSONRPCSSEClose(stmt, ed)
	})
}

func writeJSONRPCSSEClientStreamType(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	codegen.Doc(stmt, fmt.Sprintf("%sClientStream implements the %s.%sClientStream interface using Server-Sent Events.", ed.Method.VarName, ed.ServicePkgName, ed.Method.VarName))
	stmt.Type().Id(ed.Method.VarName+"ClientStream").Struct(
		jen.Id("resp").Op("*").Qual("net/http", "Response"),
		jen.Id("reader").Op("*").Qual("bufio", "Reader"),
		jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")),
		jen.Id("closed").Bool(),
		jen.Id("lock").Qual("sync", "Mutex"),
	)
	stmt.Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName+"ClientStream")).
		Id("readSSEEvent").
		Params().
		Params(jen.Index().Byte(), jen.Error()).
		Block(
			jen.Var().Id("event").Qual("bytes", "Buffer"),
			jen.Line(),
			jen.For().Block(
				jen.List(jen.Id("line"), jen.Err()).Op(":=").Id("s").Dot("reader").Dot("ReadString").Call(jen.LitByte('\n')),
				jen.If(jen.Err().Op("!=").Nil()).Block(
					jen.If(jen.Err().Op("==").Qual("io", "EOF").Op("&&").Id("event").Dot("Len").Call().Op(">").Lit(0)).Block(
						jen.Return(jen.Id("event").Dot("Bytes").Call(), jen.Nil()),
					),
					jen.Return(jen.Nil(), jen.Err()),
				),
				jen.Line(),
				jen.Id("event").Dot("WriteString").Call(jen.Id("line")),
				jen.Line(),
				jen.Id("line").Op("=").Qual("strings", "TrimRight").Call(jen.Id("line"), jen.Lit("\r\n")),
				jen.If(jen.Id("line").Op("==").Lit("")).Block(
					jen.If(jen.Id("event").Dot("Len").Call().Op(">").Lit(0)).Block(
						jen.Return(jen.Id("event").Dot("Bytes").Call(), jen.Nil()),
					),
					jen.Continue(),
				),
			),
		)
}

func writeJSONRPCSSERecv(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	codegen.Doc(stmt, ed.Method.ClientStream.RecvDesc)
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName+"ClientStream")).
		Id(ed.Method.ClientStream.RecvName).
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(codegen.TypeRef(ed.Result.Ref), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Id("s").Dot("lock").Dot("Lock").Call()
			g.Defer().Id("s").Dot("lock").Dot("Unlock").Call()
			g.Line()
			g.Var().Id("zero").Add(codegen.TypeRef(ed.Result.Ref))
			g.Line()
			g.If(jen.Id("s").Dot("closed")).Block(
				jen.Return(jen.Id("zero"), jen.Qual("io", "EOF")),
			)
			g.Line()
			g.For().Block(
				jen.List(jen.Id("rawEvent"), jen.Err()).Op(":=").Id("s").Dot("readSSEEvent").Call(),
				jen.If(jen.Err().Op("!=").Nil()).Block(
					jen.Id("s").Dot("closed").Op("=").True(),
					jen.Return(jen.Id("zero"), jen.Err()),
				),
				jen.Line(),
				jen.List(jen.Id("parsedEvent"), jen.Err()).Op(":=").Add(codegen.Expr("loomhttp.ParseSSEEvent")).Call(jen.Id("rawEvent")),
				jen.If(jen.Err().Op("!=").Nil()).Block(
					jen.Id("s").Dot("closed").Op("=").True(),
					jen.Return(jen.Id("zero"), jen.Err()),
				),
				jen.Line(),
				jen.List(jen.Id("eventType"), jen.Id("data")).Op(":=").List(
					jen.Id("parsedEvent").Dot("Type"),
					jen.Index().Byte().Call(jen.Id("parsedEvent").Dot("Data")),
				),
				jen.Line(),
				jen.Switch(jen.Id("eventType")).BlockFunc(func(sg *jen.Group) {
					sg.Case(jen.Lit("notification")).BlockFunc(func(cg *jen.Group) {
						writeSSEClientNotificationCase(cg, ed)
					})
					sg.Case(jen.Lit("response")).BlockFunc(func(cg *jen.Group) {
						writeSSEClientResponseCase(cg, ed, false)
					})
					sg.Case(jen.Lit("error")).BlockFunc(func(cg *jen.Group) {
						writeSSEClientErrorCase(cg)
					})
					sg.Case(jen.Lit(""), jen.Lit("message")).BlockFunc(func(cg *jen.Group) {
						writeSSEClientMessageCase(cg, ed)
					})
					sg.Default().Block(
						jen.Continue(),
					)
				}),
			)
		})
}

func writeSSEClientNotificationCase(g *jen.Group, ed *httpcodegen.EndpointData) {
	g.Var().Id("notification").Struct(
		jen.Id("JSONRPC").String().Tag(map[string]string{"json": "jsonrpc"}),
		jen.Id("Method").String().Tag(map[string]string{"json": "method"}),
		jen.Id("Params").Qual("encoding/json", "RawMessage").Tag(map[string]string{"json": "params"}),
	)
	g.If(
		jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("notification")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse notification: %w"), jen.Err())),
	)
	g.If(jen.Id("notification").Dot("JSONRPC").Op("!=").Lit("2.0")).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("invalid JSON-RPC version: %s"), jen.Id("notification").Dot("JSONRPC"))),
	)
	g.If(jen.Id("notification").Dot("Method").Op("!=").Lit(ed.Method.Name)).Block(
		jen.Continue(),
	)
	if ed.Method.Result != "" {
		g.List(jen.Id("result"), jen.Err()).Op(":=").Id("s").Dot("decodeResult").Call(jen.Id("notification").Dot("Params"))
		g.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to decode result: %w"), jen.Err())),
		)
		g.Return(jen.Id("result"), jen.Nil())
		return
	}
	g.Return(jen.Id("zero"), jen.Nil())
}

func writeSSEClientResponseCase(g *jen.Group, ed *httpcodegen.EndpointData, closeOnSuccess bool) {
	g.Var().Id("response").Add(codegen.TypeRef("jsonrpc.Response"))
	g.If(
		jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("response")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse response: %w"), jen.Err())),
	)
	g.If(jen.Id("response").Dot("Error").Op("!=").Nil()).Block(
		func() jen.Code {
			if closeOnSuccess {
				return jen.Block(
					jen.Id("s").Dot("closed").Op("=").True(),
					jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("JSON-RPC error %d: %s"), jen.Id("response").Dot("Error").Dot("Code"), jen.Id("response").Dot("Error").Dot("Message"))),
				)
			}
			return jen.Block(
				jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("JSON-RPC error %d: %s"), jen.Id("response").Dot("Error").Dot("Code"), jen.Id("response").Dot("Error").Dot("Message"))),
			)
		}(),
	)
	if ed.Method.Result != "" {
		g.If(jen.Id("response").Dot("Result").Op("==").Nil()).Block(
			jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("missing result in response"))),
		)
		g.List(jen.Id("resultBytes"), jen.Err()).Op(":=").Qual("encoding/json", "Marshal").Call(jen.Id("response").Dot("Result"))
		g.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to marshal result: %w"), jen.Err())),
		)
		g.List(jen.Id("result"), jen.Err()).Op(":=").Id("s").Dot("decodeResult").Call(jen.Qual("encoding/json", "RawMessage").Call(jen.Id("resultBytes")))
		g.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to decode final result: %w"), jen.Err())),
		)
		if closeOnSuccess {
			g.Id("s").Dot("closed").Op("=").True()
		}
		g.Return(jen.Id("result"), jen.Nil())
		return
	}
	if closeOnSuccess {
		g.Id("s").Dot("closed").Op("=").True()
	}
	g.Return(jen.Id("zero"), jen.Nil())
}

func writeSSEClientErrorCase(g *jen.Group) {
	g.Var().Id("response").Add(codegen.TypeRef("jsonrpc.Response"))
	g.If(
		jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("response")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse error response: %w"), jen.Err())),
	)
	g.Id("s").Dot("closed").Op("=").True()
	g.If(jen.Id("response").Dot("Error").Op("!=").Nil()).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("JSON-RPC error %d: %s"), jen.Id("response").Dot("Error").Dot("Code"), jen.Id("response").Dot("Error").Dot("Message"))),
	)
	g.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("unexpected error response")))
}

func writeSSEClientMessageCase(g *jen.Group, ed *httpcodegen.EndpointData) {
	g.Var().Id("envelope").Map(jen.String()).Qual("encoding/json", "RawMessage")
	g.If(
		jen.Err().Op(":=").Qual("encoding/json", "Unmarshal").Call(jen.Id("data"), jen.Op("&").Id("envelope")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to parse message event: %w"), jen.Err())),
	)
	g.If(
		jen.List(jen.Id("_"), jen.Id("ok")).Op(":=").Id("envelope").Index(jen.Lit("method")),
		jen.Id("ok"),
	).BlockFunc(func(cg *jen.Group) {
		writeSSEClientNotificationCase(cg, ed)
	})
	writeSSEClientResponseCase(g, ed, true)
}

func writeJSONRPCSSEDecodeResult(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName+"ClientStream")).
		Id("decodeResult").
		Params(jen.Id("data").Qual("encoding/json", "RawMessage")).
		Params(codegen.TypeRef(ed.Result.Ref), jen.Error()).
		Block(
			jen.Id("resp").Op(":=").Op("&").Qual("net/http", "Response").Values(jen.Dict{
				jen.Id("StatusCode"): jen.Qual("net/http", "StatusOK"),
				jen.Id("Body"):       jen.Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewReader").Call(jen.Id("data"))),
			}),
			jen.Id("decoder").Op(":=").Id("s").Dot("decoder").Call(jen.Id("resp")),
			jen.Var().Id("result").Add(codegen.TypeRef(ed.Result.Ref)),
			jen.If(
				jen.Err().Op(":=").Id("decoder").Dot("Decode").Call(jen.Op("&").Id("result")),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Return(jen.Id("result"), jen.Err()),
			),
			jen.Return(jen.Id("result"), jen.Nil()),
		)
}

func writeJSONRPCSSEClose(stmt *jen.Statement, ed *httpcodegen.EndpointData) {
	codegen.Doc(stmt, "Close closes the stream.")
	stmt.Func().Params(jen.Id("s").Op("*").Id(ed.Method.VarName+"ClientStream")).
		Id("Close").
		Params().
		Error().
		Block(
			jen.Id("s").Dot("lock").Dot("Lock").Call(),
			jen.Defer().Id("s").Dot("lock").Dot("Unlock").Call(),
			jen.Line(),
			jen.If(jen.Op("!").Id("s").Dot("closed")).Block(
				jen.Id("s").Dot("closed").Op("=").True(),
				jen.If(jen.Id("s").Dot("resp").Op("!=").Nil().Op("&&").Id("s").Dot("resp").Dot("Body").Op("!=").Nil()).Block(
					jen.Return(jen.Id("s").Dot("resp").Dot("Body").Dot("Close").Call()),
				),
			),
			jen.Return(jen.Nil()),
		)
}

func jsonrpcWebSocketClientStreamSection(ws *httpcodegen.WebSocketData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-websocket-client-stream", func(stmt *jen.Statement) {
		hasRecv := ws.RecvName != "" && ws.RecvTypeRef != ""
		hasSend := ws.SendName != ""
		isBidirectional := hasSend && hasRecv

		writeJSONRPCWebSocketClientTypes(stmt, ws, hasRecv)
		stmt.Line()
		writeJSONRPCWebSocketSend(stmt, ws, isBidirectional)
		if hasRecv {
			stmt.Line()
			writeJSONRPCWebSocketRecv(stmt, ws, isBidirectional)
		}
		stmt.Line()
		writeJSONRPCWebSocketClientHelpers(stmt, ws, hasRecv)
	})
}

func writeJSONRPCWebSocketClientTypes(stmt *jen.Statement, ws *httpcodegen.WebSocketData, hasRecv bool) {
	codegen.Doc(stmt, fmt.Sprintf("%s implements the %s client stream with direct WebSocket handling.", ws.VarName, ws.Endpoint.Method.Name))
	stmt.Type().Id(ws.VarName).StructFunc(func(g *jen.Group) {
		g.Id("ws").Op("*").Qual("github.com/gorilla/websocket", "Conn")
		g.Id("writeMu").Qual("sync", "Mutex")
		g.Id("pending").Qual("sync", "Map")
		g.Id("idGenerator").Qual("sync/atomic", "Uint64")
		g.Id("ctx").Qual("context", "Context")
		g.Id("cancel").Qual("context", "CancelFunc")
		g.Id("done").Chan().Struct()
		g.Id("closeOnce").Qual("sync", "Once")
		g.Id("errorOnce").Qual("sync", "Once")
		g.Id("lastError").Qual("sync/atomic", "Value")
		g.Id("config").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "StreamConfig")
		if hasRecv {
			g.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder"))
		}
	})
	stmt.Line()
	stmt.Type().Id(ws.VarName+"PendingRequest").Struct(
		jen.Id("userID").String(),
		jen.Id("resultChan").Chan().Id(ws.VarName+"StreamResult"),
		jen.Id("timeout").Op("*").Qual("time", "Timer"),
	)
	stmt.Line()
	stmt.Type().Id(ws.VarName + "StreamResult").StructFunc(func(g *jen.Group) {
		if hasRecv {
			g.Id("result").Add(codegen.TypeRef(ws.RecvTypeRef))
		}
		g.Id("err").Error()
	})
}

func writeJSONRPCWebSocketSend(stmt *jen.Statement, ws *httpcodegen.WebSocketData, isBidirectional bool) {
	if ws.SendName == "" {
		return
	}
	codegen.Doc(stmt, fmt.Sprintf("%s sends streaming data to the %s endpoint with dual ID correlation.", ws.SendName, ws.Endpoint.Method.Name))
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendName).
		Params(jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		Block(
			jen.Return(jen.Id("s").Dot(ws.SendWithContextName).Call(jen.Id("s").Dot("ctx"), jen.Id("v"))),
		)
	stmt.Line()
	codegen.Doc(stmt, fmt.Sprintf("%s sends streaming data to the %s endpoint with context.", ws.SendWithContextName, ws.Endpoint.Method.Name))
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		BlockFunc(func(g *jen.Group) {
			g.If(
				jen.Err().Op(":=").Id("s").Dot("getError").Call(),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Return(jen.Err()),
			)
			if isBidirectional {
				writeJSONRPCWebSocketBidirectionalSend(g, ws)
				return
			}
			writeJSONRPCWebSocketSimpleSend(g, ws)
		})
}

func writeJSONRPCWebSocketBidirectionalSend(g *jen.Group, ws *httpcodegen.WebSocketData) {
	g.Id("userID").Op(":=").Id("s").Dot("generateUserID").Call()
	g.Id("jsonrpcID").Op(":=").Qual("strconv", "FormatUint").Call(jen.Id("s").Dot("idGenerator").Dot("Add").Call(jen.Lit(1)), jen.Lit(10))
	g.Id("pending").Op(":=").Op("&").Id(ws.VarName + "PendingRequest").Values(jen.Dict{
		jen.Id("userID"):     jen.Id("userID"),
		jen.Id("resultChan"): jen.Make(jen.Chan().Id(ws.VarName+"StreamResult"), jen.Id("s").Dot("config").Dot("ResultChannelBuffer")),
		jen.Id("timeout"):    jen.Qual("time", "NewTimer").Call(jen.Id("s").Dot("config").Dot("RequestTimeout")),
	})
	g.Id("s").Dot("pending").Dot("Store").Call(jen.Id("jsonrpcID"), jen.Id("pending"))
	writeJSONRPCWriteRequest(g, ws, true, "v")
	g.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Id("s").Dot("pending").Dot("Delete").Call(jen.Id("jsonrpcID")),
		jen.Id("pending").Dot("timeout").Dot("Stop").Call(),
		jen.Id("s").Dot("setError").Call(jen.Id("err")),
		jen.Id("s").Dot("handleError").Call(codegen.Expr("jsonrpc.StreamErrorConnection"), jen.Id("err"), jen.Nil()),
		jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send request: %w"), jen.Id("err"))),
	)
	g.Return(jen.Nil())
}

func writeJSONRPCWebSocketSimpleSend(g *jen.Group, ws *httpcodegen.WebSocketData) {
	writeJSONRPCWriteRequest(g, ws, false, "v")
	g.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Id("s").Dot("setError").Call(jen.Id("err")),
		jen.Id("s").Dot("handleError").Call(codegen.Expr("jsonrpc.StreamErrorConnection"), jen.Id("err"), jen.Nil()),
		jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send request: %w"), jen.Id("err"))),
	)
	g.Return(jen.Nil())
}

func writeJSONRPCWriteRequest(g *jen.Group, ws *httpcodegen.WebSocketData, includeID bool, paramsExpr string) {
	dict := jen.Dict{
		jen.Id("JSONRPC"): jen.Lit("2.0"),
		jen.Id("Method"):  jen.Lit(ws.Endpoint.Method.Name),
		jen.Id("Params"):  codegen.Expr(paramsExpr),
	}
	if includeID {
		dict[jen.Id("ID")] = jen.Op("&").Id("jsonrpcID")
	}
	g.Id("request").Op(":=").Op("&").Add(codegen.TypeRef("jsonrpc.Request")).Values(dict)
	g.Id("s").Dot("writeMu").Dot("Lock").Call()
	g.Id("err").Op(":=").Id("s").Dot("ws").Dot("WriteJSON").Call(jen.Id("request"))
	g.Id("s").Dot("writeMu").Dot("Unlock").Call()
}

func writeJSONRPCWebSocketRecv(stmt *jen.Statement, ws *httpcodegen.WebSocketData, isBidirectional bool) {
	codegen.Doc(stmt, fmt.Sprintf("%s receives streaming data from the %s endpoint.", ws.RecvName, ws.Endpoint.Method.Name))
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.RecvName).
		Params().
		Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
		Block(
			jen.Return(jen.Id("s").Dot(ws.RecvWithContextName).Call(jen.Id("s").Dot("ctx"))),
		)
	stmt.Line()
	codegen.Doc(stmt, fmt.Sprintf("%s receives streaming data from the %s endpoint with context.", ws.RecvWithContextName, ws.Endpoint.Method.Name))
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.RecvWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Var().Id("zero").Add(codegen.TypeRef(ws.RecvTypeRef))
			g.If(
				jen.Err().Op(":=").Id("s").Dot("getError").Call(),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Return(jen.Id("zero"), jen.Err()),
			)
			if isBidirectional {
				writeJSONRPCWebSocketBidirectionalRecv(g, ws)
				return
			}
			writeJSONRPCWebSocketSimpleRecv(g, ws)
		})
}

func writeJSONRPCWebSocketBidirectionalRecv(g *jen.Group, ws *httpcodegen.WebSocketData) {
	g.Var().Id("oldestPending").Op("*").Id(ws.VarName + "PendingRequest")
	g.Var().Id("oldestKey").String()
	g.Id("s").Dot("pending").Dot("Range").Call(
		jen.Func().Params(jen.Id("key"), jen.Id("value").Any()).Bool().Block(
			jen.Id("pending").Op(":=").Id("value").Assert(jen.Op("*").Id(ws.VarName+"PendingRequest")),
			jen.If(jen.Id("oldestPending").Op("==").Nil()).Block(
				jen.Id("oldestPending").Op("=").Id("pending"),
				jen.Id("oldestKey").Op("=").Id("key").Assert(jen.String()),
			),
			jen.Return(jen.False()),
		),
	)
	g.If(jen.Id("oldestPending").Op("==").Nil()).Block(
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit(fmt.Sprintf("no pending requests - call %s() first", ws.SendName)))),
	)
	g.Switch().Block(
		jen.Case(jen.Id("result").Op(":=").Op("<-").Id("oldestPending").Dot("resultChan")).Block(
			jen.Id("s").Dot("pending").Dot("Delete").Call(jen.Id("oldestKey")),
			jen.Id("oldestPending").Dot("timeout").Dot("Stop").Call(),
			jen.Return(jen.Id("result").Dot("result"), jen.Id("result").Dot("err")),
		),
	)
	g.Add(codegen.Expr(`select {
case result := <-oldestPending.resultChan:
	s.pending.Delete(oldestKey)
	oldestPending.timeout.Stop()
	return result.result, result.err
case <-oldestPending.timeout.C:
	s.pending.Delete(oldestKey)
	timeoutErr := fmt.Errorf("request timeout after %v", s.config.RequestTimeout)
	s.handleError(jsonrpc.StreamErrorTimeout, timeoutErr, nil)
	return zero, timeoutErr
case <-ctx.Done():
	return zero, ctx.Err()
case <-s.done:
	if err := s.getError(); err != nil {
		return zero, err
	}
	return zero, fmt.Errorf("stream closed")
}`))
}

func writeJSONRPCWebSocketSimpleRecv(g *jen.Group, ws *httpcodegen.WebSocketData) {
	g.Id("jsonrpcID").Op(":=").Qual("strconv", "FormatUint").Call(jen.Id("s").Dot("idGenerator").Dot("Add").Call(jen.Lit(1)), jen.Lit(10))
	g.Id("resultChan").Op(":=").Make(jen.Chan().Id(ws.VarName+"StreamResult"), jen.Id("s").Dot("config").Dot("ResultChannelBuffer"))
	g.Id("pending").Op(":=").Op("&").Id(ws.VarName + "PendingRequest").Values(jen.Dict{
		jen.Id("userID"):     jen.Id("jsonrpcID"),
		jen.Id("resultChan"): jen.Id("resultChan"),
		jen.Id("timeout"):    jen.Qual("time", "NewTimer").Call(jen.Id("s").Dot("config").Dot("RequestTimeout")),
	})
	g.Id("s").Dot("pending").Dot("Store").Call(jen.Id("jsonrpcID"), jen.Id("pending"))
	g.Defer().Func().Params().Block(
		jen.Id("s").Dot("pending").Dot("Delete").Call(jen.Id("jsonrpcID")),
		jen.Id("pending").Dot("timeout").Dot("Stop").Call(),
	).Call()
	writeJSONRPCWriteRequest(g, ws, true, "nil")
	g.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Id("s").Dot("setError").Call(jen.Id("err")),
		jen.Id("s").Dot("handleError").Call(codegen.Expr("jsonrpc.StreamErrorConnection"), jen.Id("err"), jen.Nil()),
		jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send request: %w"), jen.Id("err"))),
	)
	g.Add(codegen.Expr(`select {
case result := <-resultChan:
	return result.result, result.err
case <-pending.timeout.C:
	timeoutErr := fmt.Errorf("request timeout after %v", s.config.RequestTimeout)
	s.handleError(jsonrpc.StreamErrorTimeout, timeoutErr, nil)
	return zero, timeoutErr
case <-ctx.Done():
	return zero, ctx.Err()
case <-s.done:
	if err := s.getError(); err != nil {
		return zero, err
	}
	return zero, fmt.Errorf("stream closed")
}`))
}

func writeJSONRPCWebSocketClientHelpers(stmt *jen.Statement, ws *httpcodegen.WebSocketData, hasRecv bool) {
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id("responseHandler").
		Params().
		Block(
			jen.Defer().Close(jen.Id("s").Dot("done")),
			codegen.Expr(`for {
		select {
		case <-s.ctx.Done():
			s.cleanupPendingRequests(s.ctx.Err())
			return
		default:
			var response jsonrpc.RawResponse
			if err := s.ws.ReadJSON(&response); err != nil {
				connectionErr := fmt.Errorf("failed to read response: %w", err)
				s.setError(connectionErr)
				s.handleError(jsonrpc.StreamErrorConnection, connectionErr, nil)
				s.cleanupPendingRequests(connectionErr)
				return
			}
			s.handleResponse(&response)
		}
	}`),
		)
	stmt.Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id("handleResponse").
		Params(jen.Id("response").Op("*").Add(codegen.TypeRef("jsonrpc.RawResponse"))).
		Block(
			codegen.Expr(`if response.ID == nil {
	if s.config.ErrorHandler != nil {
		s.config.ErrorHandler(s.ctx, jsonrpc.StreamErrorNotification, fmt.Errorf("received server notification"), response)
	}
	return
}`),
			jen.Id("jsonrpcID").Op(":=").Id("response").Dot("ID"),
			jen.List(jen.Id("pendingInterface"), jen.Id("exists")).Op(":=").Id("s").Dot("pending").Dot("LoadAndDelete").Call(jen.Id("jsonrpcID")),
			jen.If(jen.Op("!").Id("exists")).Block(
				jen.Id("s").Dot("handleError").Call(codegen.Expr("jsonrpc.StreamErrorOrphaned"), jen.Qual("fmt", "Errorf").Call(jen.Lit("received response for unknown ID: %s"), jen.Id("jsonrpcID")), jen.Id("response")),
				jen.Return(),
			),
			jen.Id("pending").Op(":=").Id("pendingInterface").Assert(jen.Op("*").Id(ws.VarName+"PendingRequest")),
			jen.Id("pending").Dot("timeout").Dot("Stop").Call(),
			jen.Var().Id("result").Id(ws.VarName+"StreamResult"),
			jen.If(jen.Id("response").Dot("Error").Op("!=").Nil()).Block(
				jen.Id("result").Dot("err").Op("=").Id("response").Dot("Error"),
				jen.Id("s").Dot("handleError").Call(codegen.Expr("jsonrpc.StreamErrorProtocol"), jen.Id("response").Dot("Error"), jen.Id("response")),
			).Else().BlockFunc(func(g *jen.Group) {
				if hasRecv {
					writeJSONRPCWebSocketDecodeResponseSuccess(g, ws)
				}
			}),
			codegen.Expr(`select {
case pending.resultChan <- result:
default:
}`),
		)
	stmt.Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id("generateUserID").
		Params().
		String().
		Block(
			jen.Return(jen.Qual("fmt", "Sprintf").Call(jen.Lit("user-%d-%d"), jen.Qual("time", "Now").Call().Dot("UnixNano").Call(), jen.Id("s").Dot("idGenerator").Dot("Load").Call())),
		)
	stmt.Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id("handleError").
		Params(
			jen.Id("errorType").Add(codegen.TypeRef("jsonrpc.StreamErrorType")),
			jen.Id("err").Error(),
			jen.Id("response").Op("*").Add(codegen.TypeRef("jsonrpc.RawResponse")),
		).
		Block(
			jen.If(jen.Id("s").Dot("config").Dot("ErrorHandler").Op("!=").Nil()).Block(
				jen.Id("s").Dot("config").Dot("ErrorHandler").Call(jen.Id("s").Dot("ctx"), jen.Id("errorType"), jen.Id("err"), jen.Id("response")),
			),
		)
	if hasRecv {
		stmt.Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
			Id("decodeResponse").
			Params(jen.Id("data").Qual("encoding/json", "RawMessage")).
			Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
			Block(
				jen.Id("resp").Op(":=").Op("&").Qual("net/http", "Response").Values(jen.Dict{
					jen.Id("StatusCode"): jen.Qual("net/http", "StatusOK"),
					jen.Id("Body"):       jen.Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewReader").Call(jen.Id("data"))),
				}),
				jen.Id("dec").Op(":=").Id("s").Dot("decoder").Call(jen.Id("resp")),
				jen.Var().Id("out").Add(codegen.TypeRef(ws.RecvTypeRef)),
				jen.If(
					jen.Err().Op(":=").Id("dec").Dot("Decode").Call(jen.Op("&").Id("out")),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.Return(jen.Nil(), jen.Err()),
				),
				jen.Return(jen.Id("out"), jen.Nil()),
			)
	}
	stmt.Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id("setError").
		Params(jen.Id("err").Error()).
		Block(
			jen.Id("s").Dot("errorOnce").Dot("Do").Call(
				jen.Func().Params().Block(
					jen.Id("s").Dot("lastError").Dot("Store").Call(jen.Id("err")),
					jen.Id("s").Dot("cancel").Call(),
				),
			),
		)
	stmt.Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id("getError").
		Params().
		Error().
		Block(
			jen.If(
				jen.List(jen.Id("err"), jen.Id("ok")).Op(":=").Id("s").Dot("lastError").Dot("Load").Call().Assert(jen.Error()),
				jen.Id("ok"),
			).Block(
				jen.Return(jen.Id("err")),
			),
			jen.Return(jen.Nil()),
		)
	stmt.Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id("cleanupPendingRequests").
		Params(jen.Id("err").Error()).
		Block(
			jen.Id("s").Dot("pending").Dot("Range").Call(
				jen.Func().Params(jen.Id("key"), jen.Id("value").Any()).Bool().Block(
					jen.Id("pending").Op(":=").Id("value").Assert(jen.Op("*").Id(ws.VarName+"PendingRequest")),
					jen.Id("pending").Dot("timeout").Dot("Stop").Call(),
					codegen.Expr(fmt.Sprintf(`select {
case pending.resultChan <- %sStreamResult{err: err}:
default:
}`, ws.VarName)),
					jen.Id("s").Dot("pending").Dot("Delete").Call(jen.Id("key")),
					jen.Return(jen.True()),
				),
			),
		)
	stmt.Line()
	codegen.Doc(stmt, "Close closes the stream and cleans up resources.")
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id("Close").
		Params().
		Error().
		Block(
			jen.Var().Id("err").Error(),
			jen.Id("s").Dot("closeOnce").Dot("Do").Call(
				jen.Func().Params().Block(
					jen.Id("s").Dot("cancel").Call(),
					codegen.Expr(`select {
case <-s.done:
case <-time.After(s.config.CloseTimeout):
}`),
					jen.Id("s").Dot("cleanupPendingRequests").Call(jen.Qual("fmt", "Errorf").Call(jen.Lit("stream closed"))),
					jen.If(jen.Id("s").Dot("ws").Op("!=").Nil()).Block(
						jen.Id("err").Op("=").Id("s").Dot("ws").Dot("Close").Call(),
					),
				),
			),
			jen.Return(jen.Id("err")),
		)
}

func writeJSONRPCWebSocketDecodeResponseSuccess(g *jen.Group, ws *httpcodegen.WebSocketData) {
	g.List(jen.Id("parsedResult"), jen.Id("err")).Op(":=").Id("s").Dot("decodeResponse").Call(jen.Id("response").Dot("Result"))
	g.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Id("result").Dot("err").Op("=").Qual("fmt", "Errorf").Call(jen.Lit("failed to decode response: %w"), jen.Id("err")),
		jen.Id("s").Dot("handleError").Call(codegen.Expr("jsonrpc.StreamErrorParsing"), jen.Id("err"), jen.Id("response")),
	).Else().BlockFunc(func(eg *jen.Group) {
		if ws.Endpoint.Result.IDAttribute != "" {
			if ws.Endpoint.Result.IDAttributeRequired {
				eg.If(jen.Id("parsedResult").Dot(ws.Endpoint.Result.IDAttribute).Op("==").Lit("")).Block(
					jen.Id("parsedResult").Dot(ws.Endpoint.Result.IDAttribute).Op("=").Add(codegen.Expr("jsonrpc.IDToString(response.ID)")),
				)
			} else {
				eg.If(
					jen.Id("parsedResult").Dot(ws.Endpoint.Result.IDAttribute).Op("==").Nil().Op("||").Op("*").Id("parsedResult").Dot(ws.Endpoint.Result.IDAttribute).Op("==").Lit(""),
				).Block(
					jen.Id("idCopy").Op(":=").Add(codegen.Expr("jsonrpc.IDToString(response.ID)")),
					jen.Id("parsedResult").Dot(ws.Endpoint.Result.IDAttribute).Op("=").Op("&").Id("idCopy"),
				)
			}
		}
		eg.Id("result").Dot("result").Op("=").Id("parsedResult")
	})
}

func jsonrpcMinimalRequestEncoderSection(ed *httpcodegen.EndpointData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-minimal-request-encoder", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("Encode%sRequest returns an encoder for requests sent to the %s service %s JSON-RPC method.", ed.Method.VarName, ed.ServiceName, ed.Method.Name))
		stmt.Func().
			Id("Encode" + ed.Method.VarName + "Request").
			Params(
				jen.Id("encoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Encoder")),
			).
			Params(
				jen.Func().Params(jen.Op("*").Qual("net/http", "Request"), jen.Any()).Error(),
			).
			Block(
				jen.Return(
					jen.Func().
						Params(jen.Id("req").Op("*").Qual("net/http", "Request"), jen.Id("v").Any()).
						Error().
						Block(
							jen.Id("id").Op(":=").Qual("github.com/google/uuid", "New").Call().Dot("String").Call(),
							jen.Id("body").Op(":=").Op("&").Qual("github.com/CaliLuke/loom/jsonrpc", "Request").Values(jen.Dict{
								jen.Id("JSONRPC"): jen.Lit("2.0"),
								jen.Id("Method"):  jen.Lit(ed.Method.Name),
								jen.Id("ID"):      jen.Id("id"),
							}),
							jen.If(
								jen.Err().Op(":=").Id("encoder").Call(jen.Id("req")).Dot("Encode").Call(jen.Id("body")),
								jen.Err().Op("!=").Nil(),
							).Block(
								jen.Return(
									jen.Qual("github.com/CaliLuke/loom/http", "ErrEncodingError").Call(
										jen.Lit(ed.ServiceName),
										jen.Lit(ed.Method.Name),
										jen.Err(),
									),
								),
							),
							jen.Return(jen.Nil()),
						),
				),
			)
	})
}

func jsonrpcClientEndpointInitSection(ed *httpcodegen.EndpointData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-client-endpoint-init", func(stmt *jen.Statement) {
		requestEncoder := jsonrpcRequestEncoderName(ed)
		codegen.Doc(stmt, fmt.Sprintf("%s returns an endpoint that makes JSON-RPC requests to the %s service %s method.", ed.EndpointInit, ed.ServiceName, ed.Method.Name))
		stmt.Func().
			Params(jen.Id("c").Op("*").Id(ed.ClientStruct)).
			Id(ed.EndpointInit).
			Params().
			Add(codegen.TypeRef("loom.Endpoint")).
			BlockFunc(func(g *jen.Group) {
				writeJSONRPCClientEndpointLocals(g, ed, requestEncoder)
				g.Return(
					jen.Func().
						Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Any()).
						Params(jen.Any(), jen.Error()).
						BlockFunc(func(eg *jen.Group) {
							writeJSONRPCClientRequestSetup(eg, ed, requestEncoder)
							switch {
							case httpcodegen.IsWebSocketEndpoint(ed):
								writeJSONRPCWebSocketEndpointBody(eg, ed)
							case httpcodegen.IsSSEEndpoint(ed):
								writeJSONRPCSSEEndpointBody(eg, ed)
							default:
								writeJSONRPCUnaryEndpointBody(eg, ed)
							}
						}),
				)
			})
	})
}

func jsonrpcRequestEncoderName(ed *httpcodegen.EndpointData) string {
	if ed.RequestEncoder != "" || httpcodegen.IsWebSocketEndpoint(ed) {
		return ed.RequestEncoder
	}
	return fmt.Sprintf("Encode%sRequest", ed.Method.VarName)
}

func writeJSONRPCClientEndpointLocals(g *jen.Group, ed *httpcodegen.EndpointData, requestEncoder string) {
	if httpcodegen.IsWebSocketEndpoint(ed) {
		return
	}
	g.Var().DefsFunc(func(defs *jen.Group) {
		if requestEncoder != "" {
			defs.Id("encodeRequest").Op("=").Id(requestEncoder).Call(jen.Id("c").Dot("encoder"))
		}
		if !httpcodegen.IsSSEEndpoint(ed) {
			defs.Id("decodeResponse").Op("=").Id(ed.ResponseDecoder).Call(jen.Id("c").Dot("decoder"), jen.Id("c").Dot("RestoreResponseBody"))
		}
	})
	g.Line()
}

func writeJSONRPCClientRequestSetup(g *jen.Group, ed *httpcodegen.EndpointData, requestEncoder string) {
	if httpcodegen.IsWebSocketEndpoint(ed) {
		return
	}
	args := []jen.Code{jen.Id("ctx")}
	for _, arg := range ed.RequestInit.ClientArgs {
		args = append(args, codegen.Expr(arg.Ref))
	}
	g.List(jen.Id("req"), jen.Err()).Op(":=").Id("c").Dot(ed.RequestInit.Name).Call(args...)
	g.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Err()),
	)
	if requestEncoder == "" {
		return
	}
	g.If(jen.Err().Op(":=").Id("encodeRequest").Call(jen.Id("req"), jen.Id("v")), jen.Err().Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Err()),
	)
}

func writeJSONRPCWebSocketEndpointBody(g *jen.Group, ed *httpcodegen.EndpointData) {
	if ed.ClientWebSocket != nil && ed.ClientWebSocket.RecvName != "" && ed.ClientWebSocket.RecvTypeRef != "" {
		g.Id("decodeResponse").Op(":=").Id("c").Dot("decoder")
	}
	g.List(jen.Id("ws"), jen.Err()).Op(":=").Id("c").Dot("getConn").Call(jen.Id("ctx"))
	g.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Err()),
	)
	g.Line()
	g.List(jen.Id("streamCtx"), jen.Id("cancel")).Op(":=").Qual("context", "WithCancel").Call(jen.Id("ctx"))
	dict := jen.Dict{
		jen.Id("ws"):     jen.Id("ws"),
		jen.Id("ctx"):    jen.Id("streamCtx"),
		jen.Id("cancel"): jen.Id("cancel"),
		jen.Id("done"):   jen.Make(jen.Chan().Struct()),
		jen.Id("config"): jen.Id("c").Dot("streamConfig"),
	}
	if ed.ClientWebSocket != nil && ed.ClientWebSocket.RecvName != "" && ed.ClientWebSocket.RecvTypeRef != "" {
		dict[jen.Id("decoder")] = jen.Id("decodeResponse")
	}
	g.Id("stream").Op(":=").Op("&").Id(ed.ClientWebSocket.VarName).Values(dict)
	g.Go().Id("stream").Dot("responseHandler").Call()
	g.Return(jen.Id("stream"), jen.Nil())
}

func writeJSONRPCSSEEndpointBody(g *jen.Group, ed *httpcodegen.EndpointData) {
	writeJSONRPCDoRequest(g, ed)
	g.If(jen.Id("resp").Dot("StatusCode").Op("!=").Qual("net/http", "StatusOK")).Block(
		jen.List(jen.Id("body"), jen.Id("_")).Op(":=").Qual("io", "ReadAll").Call(jen.Id("resp").Dot("Body")),
		jen.Id("resp").Dot("Body").Dot("Close").Call(),
		jen.Return(
			jen.Nil(),
			jen.Qual("github.com/CaliLuke/loom/http", "ErrInvalidResponse").Call(
				jen.Lit(ed.ServiceName),
				jen.Lit(ed.Method.Name),
				jen.Id("resp").Dot("StatusCode"),
				jen.String().Call(jen.Id("body")),
			),
		),
	)
	g.Line()
	g.Id("contentType").Op(":=").Id("resp").Dot("Header").Dot("Get").Call(jen.Lit("Content-Type"))
	g.If(
		jen.Id("contentType").Op("!=").Lit("").Op("&&").
			Op("!").Qual("strings", "HasPrefix").Call(jen.Id("contentType"), jen.Lit("text/event-stream")),
	).Block(
		jen.Id("resp").Dot("Body").Dot("Close").Call(),
		jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(jen.Lit("unexpected content type: %s (expected text/event-stream)"), jen.Id("contentType"))),
	)
	g.Line()
	g.Id("stream").Op(":=").Op("&").Id(ed.Method.VarName + "ClientStream").Values(jen.Dict{
		jen.Id("resp"):    jen.Id("resp"),
		jen.Id("reader"):  jen.Qual("bufio", "NewReader").Call(jen.Id("resp").Dot("Body")),
		jen.Id("decoder"): jen.Id("c").Dot("decoder"),
	})
	g.Return(jen.Id("stream"), jen.Nil())
}

func writeJSONRPCUnaryEndpointBody(g *jen.Group, ed *httpcodegen.EndpointData) {
	writeJSONRPCDoRequest(g, ed)
	g.Return(jen.Id("decodeResponse").Call(jen.Id("resp")))
}

func writeJSONRPCDoRequest(g *jen.Group, ed *httpcodegen.EndpointData) {
	g.List(jen.Id("resp"), jen.Err()).Op(":=").Id("c").Dot("Doer").Dot("Do").Call(jen.Id("req"))
	g.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(
			jen.Nil(),
			jen.Qual("github.com/CaliLuke/loom/http", "ErrRequestError").Call(
				jen.Lit(ed.ServiceName),
				jen.Lit(ed.Method.Name),
				jen.Err(),
			),
		),
	)
	if httpcodegen.IsSSEEndpoint(ed) {
		g.Line()
	}
}

func jsonrpcWebSocketClientConnSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-client-websocket-conn", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "getConn returns the current WebSocket connection or creates a new one.")
		stmt.Func().
			Params(jen.Id("c").Op("*").Id(data.ClientStruct)).
			Id("getConn").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Params(jen.Op("*").Qual("github.com/gorilla/websocket", "Conn"), jen.Error()).
			BlockFunc(func(g *jen.Group) {
				g.Id("c").Dot("connMu").Dot("RLock").Call()
				g.Id("conn").Op(":=").Id("c").Dot("conn")
				g.If(jen.Id("conn").Op("!=").Nil()).Block(
					jen.If(
						jen.Err().Op(":=").Id("conn").Dot("WriteControl").Call(
							jen.Qual("github.com/gorilla/websocket", "PingMessage"),
							jen.Index().Byte().Values(),
							jen.Qual("time", "Now").Call().Add(jen.Lit(5).Op("*").Qual("time", "Second")),
						),
						jen.Err().Op("==").Nil(),
					).Block(
						jen.Id("c").Dot("connMu").Dot("RUnlock").Call(),
						jen.Return(jen.Id("conn"), jen.Nil()),
					),
				)
				g.Id("c").Dot("connMu").Dot("RUnlock").Call()
				g.Line()
				g.Id("c").Dot("connMu").Dot("Lock").Call()
				g.Defer().Id("c").Dot("connMu").Dot("Unlock").Call()
				g.Line()
				g.If(jen.Id("c").Dot("conn").Op("!=").Nil()).Block(
					jen.If(
						jen.Err().Op(":=").Id("c").Dot("conn").Dot("WriteControl").Call(
							jen.Qual("github.com/gorilla/websocket", "PingMessage"),
							jen.Index().Byte().Values(),
							jen.Qual("time", "Now").Call().Add(jen.Lit(5).Op("*").Qual("time", "Second")),
						),
						jen.Err().Op("==").Nil(),
					).Block(
						jen.Return(jen.Id("c").Dot("conn"), jen.Nil()),
					),
					jen.Id("c").Dot("conn").Dot("Close").Call(),
				)
				g.Line()
				g.Id("wsScheme").Op(":=").Lit("ws")
				g.If(jen.Id("c").Dot("scheme").Op("==").Lit("https")).Block(
					jen.Id("wsScheme").Op("=").Lit("wss"),
				)
				g.Line()
				g.Id("url").Op(":=").Id("wsScheme").Op("+").Lit("://").Op("+").Id("c").Dot("host")
				if path := jsonrpcWebSocketClientPath(data); path != "" {
					g.Id("url").Op("=").Id("url").Op("+").Lit(path)
				}
				g.Line()
				g.List(jen.Id("ws"), jen.Id("_"), jen.Err()).Op(":=").Id("c").Dot("dialer").Dot("DialContext").Call(jen.Id("ctx"), jen.Id("url"), jen.Nil())
				g.If(jen.Err().Op("!=").Nil()).Block(
					jen.Return(
						jen.Nil(),
						jen.Qual("github.com/CaliLuke/loom/http", "ErrRequestError").Call(
							jen.Lit(data.Service.Name),
							jen.Lit("connect"),
							jen.Err(),
						),
					),
				)
				g.Line()
				g.If(jen.Id("c").Dot("configfn").Op("!=").Nil()).Block(
					jen.Id("ws").Op("=").Id("c").Dot("configfn").Call(jen.Id("ws"), jen.Nil()),
				)
				g.Line()
				g.Id("c").Dot("conn").Op("=").Id("ws")
				g.Return(jen.Id("c").Dot("conn"), jen.Nil())
			})
		stmt.Line()
		codegen.Doc(stmt, "Close closes the WebSocket connection and marks the client as closed.")
		stmt.Func().
			Params(jen.Id("c").Op("*").Id(data.ClientStruct)).
			Id("Close").
			Params().
			Error().
			Block(
				jen.If(jen.Id("c").Dot("closed").Dot("Swap").Call(jen.True())).Block(
					jen.Return(jen.Nil()),
				),
				jen.Line(),
				jen.Id("c").Dot("connMu").Dot("Lock").Call(),
				jen.Defer().Id("c").Dot("connMu").Dot("Unlock").Call(),
				jen.Line(),
				jen.If(jen.Id("c").Dot("conn").Op("!=").Nil()).Block(
					jen.Id("err").Op(":=").Id("c").Dot("conn").Dot("Close").Call(),
					jen.Id("c").Dot("conn").Op("=").Nil(),
					jen.Return(jen.Id("err")),
				),
				jen.Return(jen.Nil()),
			)
		stmt.Line()
		codegen.Doc(stmt, "IsClosed returns true if the client connection has been closed.")
		stmt.Func().
			Params(jen.Id("c").Op("*").Id(data.ClientStruct)).
			Id("IsClosed").
			Params().
			Bool().
			Block(
				jen.Return(jen.Id("c").Dot("closed").Dot("Load").Call()),
			)
	})
}

func jsonrpcWebSocketStreamErrorTypesSection() codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-websocket-stream-error-types", func(stmt *jen.Statement) {
		stmt.Comment("Stream error types for comprehensive error reporting.").Line()
		stmt.Type().Id("StreamErrorType").Int()
		stmt.Line()
		stmt.Const().Defs(
			jen.Id("StreamErrorConnection").Id("StreamErrorType").Op("=").Iota().Comment("WebSocket connection errors"),
			jen.Id("StreamErrorProtocol").Comment("Invalid JSON-RPC protocol"),
			jen.Id("StreamErrorParsing").Comment("Failed to parse/decode response"),
			jen.Id("StreamErrorOrphaned").Comment("Response with no matching request"),
			jen.Id("StreamErrorTimeout").Comment("Request timeout"),
		)
		stmt.Line()
		codegen.Doc(stmt, "StreamErrorHandler allows users to handle stream errors.")
		stmt.Type().Id("StreamErrorHandler").Func().
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("errorType").Id("StreamErrorType"),
				jen.Id("err").Error(),
				jen.Id("response").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawResponse"),
			)
	})
}

func jsonrpcWebSocketClientPath(data *httpcodegen.ServiceData) string {
	for _, ed := range data.Endpoints {
		for _, route := range ed.Routes {
			if route.Verb == "GET" && route.Path != "/" {
				return route.Path
			}
		}
	}
	return ""
}
