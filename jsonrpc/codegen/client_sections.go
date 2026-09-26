package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcMinimalRequestEncoderSection(ed *httpcodegen.EndpointData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-minimal-request-encoder", func(stmt *jen.Statement) {
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
							jen.List(jen.Id("id"), jen.Err()).Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "NewRequestID").Call(),
							jen.If(jen.Err().Op("!=").Nil()).Block(
								jen.Return(
									jen.Id("loomhttp").Dot("ErrEncodingError").Call(
										jen.Lit(ed.ServiceName),
										jen.Lit(ed.Method.Name),
										jen.Err(),
									),
								),
							),
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
	return codegen.NewJenniferSection("jsonrpc-client-endpoint-init", func(stmt *jen.Statement) {
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
	if httpcodegen.IsSSEEndpoint(ed) {
		g.Id("req").Dot("Header").Dot("Set").Call(jen.Lit("Accept"), jen.Lit("text/event-stream"))
	}
}

func writeJSONRPCWebSocketEndpointBody(g *jen.Group, ed *httpcodegen.EndpointData) {
	if ed.ClientWebSocket != nil && ed.ClientWebSocket.RecvName != "" && ed.ClientWebSocket.RecvTypeRef != "" {
		g.Id("decodeResponse").Op(":=").Id("c").Dot("decoder")
	}
	g.List(jen.Id("conn"), jen.Err()).Op(":=").Id("c").Dot("getConn").Call(jen.Id("ctx"))
	g.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Err()),
	)
	g.Line()
	dict := jen.Dict{
		jen.Id("stream"): jen.Qual("github.com/CaliLuke/loom/jsonrpc", "NewWebSocketClientStream").Call(
			jen.Id("ctx"), jen.Id("conn"), jen.Lit(ed.Method.Name), jen.Id("c").Dot("streamConfig"),
		),
	}
	if ed.ClientWebSocket != nil && ed.ClientWebSocket.RecvName != "" && ed.ClientWebSocket.RecvTypeRef != "" {
		dict[jen.Id("decoder")] = jen.Id("decodeResponse")
	}
	g.Return(jen.Op("&").Id(ed.ClientWebSocket.VarName).Values(dict), jen.Nil())
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
	writeJSONRPCNotificationResponse(g, ed)
	g.Return(jen.Id("decodeResponse").Call(jen.Id("resp")))
}

// writeJSONRPCNotificationResponse writes the statements that return the zero
// result of ed when the request is a notification, which the request encoder
// sends when the payload ID is empty. The server sends no response to a
// notification, so jsonrpc.DecodeNotificationResponse checks the HTTP
// response instead of the response decoder. A payload without an ID
// attribute is always sent with a generated ID.
func writeJSONRPCNotificationResponse(g *jen.Group, ed *httpcodegen.EndpointData) {
	if ed.Payload == nil || ed.Payload.IDAttribute == "" {
		return
	}
	id := jen.Id("p").Dot(ed.Payload.IDAttribute)
	empty := id.Clone().Op("==").Lit("")
	if !ed.Payload.IDAttributeRequired {
		empty = id.Clone().Op("==").Nil().Op("||").Op("*").Add(id.Clone()).Op("==").Lit("")
	}
	check := jen.Qual("github.com/CaliLuke/loom/jsonrpc", "DecodeNotificationResponse").Call(jen.Lit(ed.ServiceName), jen.Lit(ed.Method.Name), jen.Id("resp"))
	// The request encoder has already checked the type of v. The locals are
	// names that service imports avoid (transportGeneratedLocalNames).
	g.If(
		jen.Id("p").Op(":=").Id("v").Assert(codegen.TypeRef(ed.Payload.Ref)),
		empty,
	).BlockFunc(func(b *jen.Group) {
		if ed.Result == nil || ed.Result.Ref == "" {
			b.Return(jen.Nil(), check)
			return
		}
		// The client method asserts the type of the result.
		b.Var().Id("res").Add(codegen.TypeRef(ed.Result.Ref))
		b.Return(jen.Id("res"), check)
	})
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
	return codegen.NewJenniferSection("jsonrpc-client-websocket-conn", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "getConn returns the WebSocket connection shared by the streams of the client, with a reference for a new stream, and dials a new connection when the current one is closed. It returns an error after the client is closed. The stream releases the reference when it ends, and the connection closes with its last reference or with the client.")
		stmt.Func().
			Params(jen.Id("c").Op("*").Id(data.ClientStruct)).
			Id("getConn").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Params(jen.Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "WebSocketClientConn"), jen.Error()).
			BlockFunc(func(g *jen.Group) {
				g.Id("c").Dot("connMu").Dot("Lock").Call()
				g.Defer().Id("c").Dot("connMu").Dot("Unlock").Call()
				g.Line()
				g.If(jen.Id("c").Dot("closed").Dot("Load").Call()).Block(
					jen.Return(jen.Nil(), jen.Id("loomhttp").Dot("ErrRequestError").Call(
						jen.Lit(data.Service.Name), jen.Lit("connect"), jen.Qual("fmt", "Errorf").Call(jen.Lit("client is closed")),
					)),
				)
				g.Line()
				g.If(jen.Id("c").Dot("conn").Op("!=").Nil().Op("&&").Id("c").Dot("conn").Dot("Acquire").Call()).Block(
					jen.Return(jen.Id("c").Dot("conn"), jen.Nil()),
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
					jen.Id("configured").Op(":=").Id("c").Dot("configfn").Call(jen.Id("ws"), jen.Nil()),
					jen.If(jen.Id("configured").Op("==").Nil()).Block(
						jen.Id("err").Op(":=").Qual("fmt", "Errorf").Call(jen.Lit("connection configure function returned a nil connection")),
						jen.If(jen.Id("closeErr").Op(":=").Id("ws").Dot("Close").Call(), jen.Id("closeErr").Op("!=").Nil()).Block(
							jen.Id("err").Op("=").Qual("fmt", "Errorf").Call(jen.Lit("%w; closing the dialed connection: %w"), jen.Id("err"), jen.Id("closeErr")),
						),
						jen.Return(
							jen.Nil(),
							jen.Id("loomhttp").Dot("ErrRequestError").Call(
								jen.Lit(data.Service.Name),
								jen.Lit("connect"),
								jen.Id("err"),
							),
						),
					),
					jen.Id("ws").Op("=").Id("configured"),
				)
				g.Line()
				g.Id("conn").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "NewWebSocketClientConn").Call(
					jen.Id("loomhttp").Dot("NewWebSocketStream").Call(jen.Id("ws")),
					jen.Id("c").Dot("streamConfig").Dot("ErrorHandler"),
				)
				g.If(jen.Op("!").Id("conn").Dot("Acquire").Call()).Block(
					jen.Return(
						jen.Nil(),
						jen.Id("loomhttp").Dot("ErrRequestError").Call(
							jen.Lit(data.Service.Name),
							jen.Lit("connect"),
							jen.Id("conn").Dot("Err").Call(),
						),
					),
				)
				g.Id("c").Dot("conn").Op("=").Id("conn")
				g.Return(jen.Id("conn"), jen.Nil())
			})
		stmt.Line()
		codegen.Doc(stmt, "Close closes the WebSocket connection shared by the streams of the client, failing the requests they still wait for, and marks the client as closed. Later attempts to open a stream fail without dialing. When the last stream already closed the connection, Close returns the result of that close.")
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
	return codegen.NewJenniferSection("jsonrpc-websocket-stream-error-types", func(stmt *jen.Statement) {
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
