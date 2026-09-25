package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// jsonrpcWebSocketClientStreamSection writes the typed client stream of a
// WebSocket endpoint. The stream adapts the typed payloads and results to a
// jsonrpc.WebSocketClientStream, which owns the requests, their routing on
// the shared connection and the stream lifecycle.
func jsonrpcWebSocketClientStreamSection(ws *httpcodegen.WebSocketData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-websocket-client-stream", func(stmt *jen.Statement) {
		hasRecv := ws.RecvName != "" && ws.RecvTypeRef != ""
		hasSend := ws.SendName != ""
		isBidirectional := hasSend && hasRecv

		writeJSONRPCWebSocketClientType(stmt, ws, hasRecv, isBidirectional)
		if hasSend {
			stmt.Line()
			writeJSONRPCWebSocketSend(stmt, ws, isBidirectional)
		}
		if hasRecv {
			stmt.Line()
			writeJSONRPCWebSocketRecv(stmt, ws, isBidirectional)
			stmt.Line()
			writeJSONRPCWebSocketDecodeResponse(stmt, ws)
		}
		stmt.Line()
		codegen.Doc(stmt, "Close ends the stream: the requests it still waits for fail, and the other streams on the connection are not affected. It closes the connection when the stream is the last one using it and returns the result of that close. Later calls return the same result.")
		stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
			Id("Close").
			Params().
			Error().
			Block(
				jen.Return(jen.Id("s").Dot("stream").Dot("Close").Call()),
			)
	})
}

func writeJSONRPCWebSocketClientType(stmt *jen.Statement, ws *httpcodegen.WebSocketData, hasRecv, isBidirectional bool) {
	doc := fmt.Sprintf("%s implements the %s client stream on the WebSocket connection shared by the streams of the client.", ws.VarName, ws.Endpoint.Method.Name)
	if isBidirectional {
		doc += fmt.Sprintf(" Every request sent with %s gets its own response, which %s returns in send order. A response that arrives before %s is called is held until %s returns it or the stream ends.", ws.SendName, ws.RecvName, ws.RecvName, ws.RecvName)
	}
	doc += " Canceling the context the stream was opened with ends the stream as Close does."
	codegen.Doc(stmt, doc)
	stmt.Type().Id(ws.VarName).StructFunc(func(g *jen.Group) {
		g.Id("stream").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "WebSocketClientStream")
		if hasRecv {
			g.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder"))
		}
	})
}

func writeJSONRPCWebSocketSend(stmt *jen.Statement, ws *httpcodegen.WebSocketData, isBidirectional bool) {
	codegen.Doc(stmt, fmt.Sprintf("%s sends streaming data to the %s endpoint.", ws.SendName, ws.Endpoint.Method.Name))
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendName).
		Params(jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		Block(
			jen.Return(jen.Id("s").Dot(ws.SendWithContextName).Call(jen.Id("s").Dot("stream").Dot("Context").Call(), jen.Id("v"))),
		)
	stmt.Line()
	send := "Notify"
	if isBidirectional {
		send = "Send"
	}
	codegen.Doc(stmt, fmt.Sprintf("%s sends streaming data to the %s endpoint with context.", ws.SendWithContextName, ws.Endpoint.Method.Name))
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		Block(
			jen.Return(jen.Id("s").Dot("stream").Dot(send).Call(jen.Id("ctx"), jen.Id("v"))),
		)
}

func writeJSONRPCWebSocketRecv(stmt *jen.Statement, ws *httpcodegen.WebSocketData, isBidirectional bool) {
	codegen.Doc(stmt, fmt.Sprintf("%s receives streaming data from the %s endpoint.", ws.RecvName, ws.Endpoint.Method.Name))
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.RecvName).
		Params().
		Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
		Block(
			jen.Return(jen.Id("s").Dot(ws.RecvWithContextName).Call(jen.Id("s").Dot("stream").Dot("Context").Call())),
		)
	stmt.Line()
	receive := jen.Id("s").Dot("stream").Dot("Call").Call(jen.Id("ctx"), jen.Nil())
	doc := fmt.Sprintf("%s requests the next result of the %s endpoint and waits for it.", ws.RecvWithContextName, ws.Endpoint.Method.Name)
	if isBidirectional {
		receive = jen.Id("s").Dot("stream").Dot("Recv").Call(jen.Id("ctx"))
		doc = fmt.Sprintf("%s receives the response to the oldest request sent with %s and not received yet. When ctx is done first, that request stays the oldest one.", ws.RecvWithContextName, ws.SendName)
	}
	codegen.Doc(stmt, doc)
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.RecvWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
		Block(
			// zero is declared before the locals that could shadow the
			// package of the result type.
			jen.Var().Id("zero").Add(codegen.TypeRef(ws.RecvTypeRef)),
			jen.List(jen.Id("response"), jen.Err()).Op(":=").Add(receive),
			jen.If(jen.Err().Op("!=").Nil()).Block(
				jen.Return(jen.Id("zero"), jen.Err()),
			),
			jen.Return(jen.Id("s").Dot("decodeResponse").Call(jen.Id("response"))),
		)
}

// writeJSONRPCWebSocketDecodeResponse writes the helper that decodes the
// result of a response and reports decoding failures to the error handler.
func writeJSONRPCWebSocketDecodeResponse(stmt *jen.Statement, ws *httpcodegen.WebSocketData) {
	codegen.Doc(stmt, "decodeResponse decodes the result of response.")
	stmt.Func().Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id("decodeResponse").
		Params(jen.Id("response").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawResponse")).
		Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
		BlockFunc(func(g *jen.Group) {
			g.Id("resp").Op(":=").Op("&").Qual("net/http", "Response").Values(jen.Dict{
				jen.Id("StatusCode"): jen.Qual("net/http", "StatusOK"),
				jen.Id("Body"):       jen.Qual("io", "NopCloser").Call(jen.Qual("bytes", "NewReader").Call(jen.Id("response").Dot("Result"))),
			})
			// out and zero share one declaration, so the result type is
			// resolved before either name can shadow its package.
			g.Var().List(jen.Id("out"), jen.Id("zero")).Add(codegen.TypeRef(ws.RecvTypeRef))
			g.If(
				jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("resp")).Dot("Decode").Call(jen.Op("&").Id("out")),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Id("s").Dot("stream").Dot("ReportError").Call(codegen.Expr("jsonrpc.StreamErrorParsing"), jen.Err(), jen.Id("response")),
				jen.Return(jen.Id("zero"), jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to decode response: %w"), jen.Err())),
			)
			if attr := ws.Endpoint.Result.IDAttribute; attr != "" {
				if ws.Endpoint.Result.IDAttributeRequired {
					g.If(jen.Id("out").Dot(attr).Op("==").Lit("")).Block(
						jen.Id("out").Dot(attr).Op("=").Add(codegen.Expr("jsonrpc.IDToString(response.ID)")),
					)
				} else {
					g.If(
						jen.Id("out").Dot(attr).Op("==").Nil().Op("||").Op("*").Id("out").Dot(attr).Op("==").Lit(""),
					).Block(
						jen.Id("idCopy").Op(":=").Add(codegen.Expr("jsonrpc.IDToString(response.ID)")),
						jen.Id("out").Dot(attr).Op("=").Op("&").Id("idCopy"),
					)
				}
			}
			g.Return(jen.Id("out"), jen.Nil())
		})
}
