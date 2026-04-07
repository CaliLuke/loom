package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

//nolint:maintidx // Stream helper generation intentionally centralizes protocol-handling branches.
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
					jen.Select().Block(
						jen.Case(
							jen.Id("pending").Dot("resultChan").Op("<-").Id(ws.VarName+"StreamResult").ValuesFunc(func(values *jen.Group) {
								values.Id("err").Op(":").Id("err")
							}),
						).Block(),
						jen.Default().Block(),
					),
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
									jen.Id("loomhttp").Dot("ErrEncodingError").Call(
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
			jen.Id("loomhttp").Dot("ErrInvalidResponse").Call(
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
			jen.Id("loomhttp").Dot("ErrRequestError").Call(
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

//nolint:maintidx // Connection bootstrap and reconnection logic are intentionally emitted together.
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
							jen.Qual("time", "Now").Call().Dot("Add").Call(jen.Lit(5).Op("*").Qual("time", "Second")),
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
							jen.Qual("time", "Now").Call().Dot("Add").Call(jen.Lit(5).Op("*").Qual("time", "Second")),
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
						jen.Id("loomhttp").Dot("ErrRequestError").Call(
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
