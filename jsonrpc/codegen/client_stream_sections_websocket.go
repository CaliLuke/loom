package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcWebSocketClientStreamSection(ws *httpcodegen.WebSocketData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-websocket-client-stream", func(stmt *jen.Statement) {
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
