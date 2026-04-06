package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

//nolint:maintidx // Generator entrypoint intentionally keeps JSON-RPC handler branches together.
func jsonrpcServerHandlerSection(data *httpcodegen.ServiceData, mixed bool) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-server-handler", func(stmt *jen.Statement) {
		if !httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) && !mixed {
			stmt.Comment("ServeHTTP handles JSON-RPC requests.").Line()
			stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
				Id("ServeHTTP").
				Params(
					jen.Id("w").Qual("net/http", "ResponseWriter"),
					jen.Id("r").Op("*").Qual("net/http", "Request"),
				).
				Block(
					jen.Id("s").Dot("handleHTTP").Call(jen.Id("w"), jen.Id("r")),
				)
			stmt.Line()
		}

		stmt.Comment("handleHTTP handles JSON-RPC requests.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("handleHTTP").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			BlockFunc(writeBufferedRequestHandling)
		stmt.Line()

		stmt.Comment("handleSingle handles a single JSON-RPC request.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("handleSingle").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			Block(
				jen.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
				jen.If(
					jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("req")),
					jen.Err().Op("!=").Nil(),
				).BlockFunc(func(g *jen.Group) {
					writeParseErrorResponse(g, jen.Id("r").Dot("Context").Call())
					g.Return()
				}),
				jen.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
			)
		stmt.Line()

		stmt.Comment("handleBatch handles a batch of JSON-RPC requests.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("handleBatch").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			Block(
				jen.Var().Id("reqs").Index().Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
				jen.If(
					jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("reqs")),
					jen.Err().Op("!=").Nil(),
				).BlockFunc(func(g *jen.Group) {
					writeParseErrorResponse(g, jen.Id("r").Dot("Context").Call())
					g.Return()
				}),
				jen.Id("w").Dot("Header").Call().Dot("Set").Call(jen.Lit("Content-Type"), jen.Lit("application/json")),
				jen.Id("writer").Op(":=").Op("&").Id("batchWriter").Values(jen.Dict{
					jen.Id("Writer"): jen.Id("w"),
				}),
				jen.For(
					jen.List(jen.Id("_"), jen.Id("req")).Op(":=").Range().Id("reqs"),
				).Block(
					jen.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("writer")),
				),
				jen.If(jen.Id("writer").Dot("written")).Block(
					jen.Id("writer").Dot("Writer").Dot("Write").Call(jen.Index().Byte().Values(jen.LitByte(']'))),
				),
			)
		stmt.Line()

		stmt.Comment("processRequest processes a single JSON-RPC request.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("processRequest").
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
				jen.Id("req").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
				jen.Id("w").Qual("net/http", "ResponseWriter"),
			).
			Block(
				jen.If(jen.Id("req").Dot("JSONRPC").Op("!=").Lit("2.0")).Block(
					jen.Id("s").Dot("encodeJSONRPCError").Call(
						jen.Id("ctx"),
						jen.Id("w"),
						jen.Id("req"),
						jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"),
						jen.Lit("Invalid request"),
						jen.Nil(),
					),
					jen.Return(),
				),
				jen.If(jen.Id("req").Dot("Method").Op("==").Lit("")).Block(
					jen.Id("s").Dot("encodeJSONRPCError").Call(
						jen.Id("ctx"),
						jen.Id("w"),
						jen.Id("req"),
						jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"),
						jen.Lit("Missing method field"),
						jen.Nil(),
					),
					jen.Return(),
				),
				jen.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(g *jen.Group) {
					writeJSONRPCMethodDispatch(g, data.Endpoints)
					g.Default().Block(
						jen.Id("s").Dot("encodeJSONRPCError").Call(
							jen.Id("ctx"),
							jen.Id("w"),
							jen.Id("req"),
							jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"),
							jen.Lit("Method not found"),
							jen.Nil(),
						),
					)
				}),
			)
		stmt.Line()

		stmt.Comment("batchWriter is a helper type that implements http.ResponseWriter for writing multiple JSON-RPC responses").Line()
		stmt.Type().Id("batchWriter").Struct(
			jen.Qual("io", "Writer"),
			jen.Id("header").Qual("net/http", "Header"),
			jen.Id("statusCode").Int(),
			jen.Id("written").Bool(),
		)
		stmt.Line()
		stmt.Func().Params(jen.Id("rb").Op("*").Id("batchWriter")).
			Id("Header").
			Params().
			Qual("net/http", "Header").
			Block(
				jen.If(jen.Id("rb").Dot("header").Op("==").Nil()).Block(
					jen.Id("rb").Dot("header").Op("=").Make(jen.Qual("net/http", "Header")),
				),
				jen.Return(jen.Id("rb").Dot("header")),
			)
		stmt.Line()
		stmt.Func().Params(jen.Id("rb").Op("*").Id("batchWriter")).
			Id("WriteHeader").
			Params(jen.Id("statusCode").Int()).
			Block(
				jen.If(jen.Id("rb").Dot("written")).Block(
					jen.Return(),
				),
				jen.Id("rb").Dot("statusCode").Op("=").Id("statusCode"),
			)
		stmt.Line()
		stmt.Func().Params(jen.Id("rb").Op("*").Id("batchWriter")).
			Id("Write").
			Params(jen.Id("data").Index().Byte()).
			Params(jen.Int(), jen.Error()).
			Block(
				jen.If(jen.Op("!").Id("rb").Dot("written")).Block(
					jen.Id("rb").Dot("written").Op("=").True(),
					jen.Id("rb").Dot("Writer").Dot("Write").Call(jen.Index().Byte().Values(jen.LitByte('['))),
				).Else().Block(
					jen.Id("rb").Dot("Writer").Dot("Write").Call(jen.Index().Byte().Values(jen.LitByte(','))),
				),
				jen.Return(jen.Id("rb").Dot("Writer").Dot("Write").Call(jen.Id("data"))),
			)
	})
}

func writeBufferedRequestHandling(g *jen.Group) {
	g.Comment("Peek at the first byte to determine request type")
	g.Id("bufReader").Op(":=").Qual("bufio", "NewReader").Call(jen.Id("r").Dot("Body"))
	g.List(jen.Id("peek"), jen.Err()).Op(":=").Id("bufReader").Dot("Peek").Call(jen.Lit(1))
	g.If(
		jen.Err().Op("!=").Nil().Op("&&").Err().Op("!=").Qual("io", "EOF"),
	).Block(
		jen.Id("r").Dot("Body").Dot("Close").Call(),
		jen.Id("s").Dot("errhandler").Call(
			jen.Id("r").Dot("Context").Call(),
			jen.Id("w"),
			jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to read request body: %w"), jen.Err()),
		),
		jen.Return(),
	)
	g.Line()
	g.Comment("Wrap the buffered reader with the original closer")
	g.Id("r").Dot("Body").Op("=").Struct(
		jen.Qual("io", "Reader"),
		jen.Qual("io", "Closer"),
	).Values(jen.Dict{
		jen.Id("Reader"): jen.Id("bufReader"),
		jen.Id("Closer"): jen.Id("r").Dot("Body"),
	})
	g.Defer().Func().Params(jen.Id("r").Op("*").Qual("net/http", "Request")).Block(
		jen.If(
			jen.Err().Op(":=").Id("r").Dot("Body").Dot("Close").Call(),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("s").Dot("errhandler").Call(
				jen.Id("r").Dot("Context").Call(),
				jen.Id("w"),
				jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to close request body: %w"), jen.Err()),
			),
		),
	).Call(jen.Id("r"))
	g.Line()
	g.Comment("Route to appropriate handler")
	g.If(
		jen.Len(jen.Id("peek")).Op(">").Lit(0).Op("&&").Id("peek").Index(jen.Lit(0)).Op("==").LitByte('['),
	).Block(
		jen.Id("s").Dot("handleBatch").Call(jen.Id("w"), jen.Id("r")),
		jen.Return(),
	)
	g.Id("s").Dot("handleSingle").Call(jen.Id("w"), jen.Id("r"))
}

func writeParseErrorResponse(g *jen.Group, ctx jen.Code) {
	g.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(
		jen.Nil(),
		jen.Qual("github.com/CaliLuke/loom/jsonrpc", "ParseError"),
		jen.Lit("Parse error"),
		jen.Nil(),
	)
	g.If(
		jen.Id("encErr").Op(":=").Id("s").Dot("encoder").Call(ctx, jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
		jen.Id("encErr").Op("!=").Nil(),
	).Block(
		jen.Id("s").Dot("errhandler").Call(
			ctx,
			jen.Id("w"),
			jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode parse error response: %w"), jen.Id("encErr")),
		),
	)
}

func writeJSONRPCMethodDispatch(g *jen.Group, endpoints []*httpcodegen.EndpointData) {
	for _, endpoint := range endpoints {
		g.Case(jen.Lit(endpoint.Method.Name)).Block(
			jen.If(
				jen.Err().Op(":=").Id("s").Dot(endpoint.Method.VarName).Call(jen.Id("ctx"), jen.Id("r"), jen.Id("req"), jen.Id("w")),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Id("s").Dot("errhandler").Call(
					jen.Id("ctx"),
					jen.Id("w"),
					jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for "+endpoint.Method.Name+": %w"), jen.Err()),
				),
			),
		)
	}
}

//nolint:maintidx // Generator entrypoint intentionally keeps SSE routing branches together.
func jsonrpcSSEServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-sse-server-handler", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "SSEStream"

		stmt.Comment("handleSSE handles JSON-RPC SSE requests by dispatching to the appropriate method.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("handleSSE").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			BlockFunc(func(g *jen.Group) {
				g.Id("ctx").Op(":=").Id("r").Dot("Context").Call()
				g.Line()
				g.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest")
				g.If(
					jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("req")),
					jen.Err().Op("!=").Nil(),
				).BlockFunc(func(eg *jen.Group) {
					writeSSEErrorStreamInit(eg, streamName)
					eg.Id("stream").Dot("sendError").Call(
						jen.Id("ctx"),
						jen.Nil(),
						jen.Qual("github.com/CaliLuke/loom/jsonrpc", "ParseError"),
						jen.Lit("Parse error"),
						jen.Nil(),
					)
					eg.Return()
				})
				g.Line()
				writeSSERequestValidation(g, streamName)
				g.Var().Id("handler").Func().Params(
					jen.Qual("context", "Context"),
					jen.Op("*").Qual("net/http", "Request"),
					jen.Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
					jen.Qual("net/http", "ResponseWriter"),
				).Error()
				g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(sg *jen.Group) {
					for _, endpoint := range data.Endpoints {
						if endpoint.SSE == nil {
							continue
						}
						sg.Case(jen.Lit(endpoint.Method.Name)).Block(
							jen.Id("handler").Op("=").Id("s").Dot(endpoint.Method.VarName),
						)
					}
					sg.Default().BlockFunc(func(dg *jen.Group) {
						dg.If(
							jen.Id("req").Dot("ID").Op("==").Nil().Op("||").Id("req").Dot("ID").Op("==").Lit(""),
						).Block(
							jen.Id("w").Dot("WriteHeader").Call(jen.Qual("net/http", "StatusNoContent")),
							jen.Return(),
						)
						dg.Id("stream").Op(":=").Op("&").Id(streamName).Values(jen.Dict{
							jen.Id("w"):       jen.Id("w"),
							jen.Id("r"):       jen.Id("r"),
							jen.Id("encoder"): jen.Id("s").Dot("encoder"),
							jen.Id("decoder"): jen.Id("s").Dot("decoder"),
						})
						dg.Id("stream").Dot("sendError").Call(
							jen.Id("ctx"),
							jen.Id("req").Dot("ID"),
							jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"),
							jen.Lit("Method not found"),
							jen.Nil(),
						)
						dg.Return()
					})
				})
				g.Line()
				g.If(
					jen.Err().Op(":=").Id("handler").Call(jen.Id("ctx"), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.Id("s").Dot("errhandler").Call(
						jen.Id("ctx"),
						jen.Id("w"),
						jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for %s: %w"), jen.Id("req").Dot("Method"), jen.Err()),
					),
					jen.Return(),
				)
				g.Line()
				g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(sg *jen.Group) {
					for _, endpoint := range data.Endpoints {
						if endpoint.SSE == nil || endpoint.Method.ServerStream != nil {
							continue
						}
						sg.Case(jen.Lit(endpoint.Method.Name)).Block(
							jen.If(jen.Id("req").Dot("ID").Op("==").Nil()).Block(
								jen.Id("w").Dot("WriteHeader").Call(jen.Qual("net/http", "StatusNoContent")),
							),
						)
					}
				})
			})
	})
}

func writeSSEErrorStreamInit(g *jen.Group, streamName string) {
	g.Id("stream").Op(":=").Op("&").Id(streamName).Values(jen.Dict{
		jen.Id("w"):       jen.Id("w"),
		jen.Id("r"):       jen.Id("r"),
		jen.Id("encoder"): jen.Id("s").Dot("encoder"),
		jen.Id("decoder"): jen.Id("s").Dot("decoder"),
	})
}

func writeSSEValidationError(g *jen.Group, streamName, message string) {
	g.If(
		jen.Id("req").Dot("ID").Op("==").Nil().Op("||").Id("req").Dot("ID").Op("==").Lit(""),
	).Block(
		jen.Id("w").Dot("WriteHeader").Call(jen.Qual("net/http", "StatusNoContent")),
		jen.Return(),
	)
	writeSSEErrorStreamInit(g, streamName)
	g.Id("stream").Dot("sendError").Call(
		jen.Id("ctx"),
		jen.Id("req").Dot("ID"),
		jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest"),
		jen.Lit(message),
		jen.Nil(),
	)
	g.Return()
}

func writeSSERequestValidation(g *jen.Group, streamName string) {
	g.If(jen.Id("req").Dot("JSONRPC").Op("!=").Lit("2.0")).BlockFunc(func(eg *jen.Group) {
		writeSSEValidationError(eg, streamName, "Invalid request")
	})
	g.Line()
	g.If(jen.Id("req").Dot("Method").Op("==").Lit("")).BlockFunc(func(eg *jen.Group) {
		writeSSEValidationError(eg, streamName, "Invalid request")
	})
	g.Line()
}

func jsonrpcWebSocketServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-websocket-server-handler", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"

		stmt.Comment("ServeHTTP handles WebSocket JSON-RPC requests.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("ServeHTTP").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			BlockFunc(func(g *jen.Group) {
				g.List(jen.Id("ctx"), jen.Id("cancel")).Op(":=").Qual("context", "WithCancel").Call(jen.Id("r").Dot("Context").Call())
				g.List(jen.Id("conn"), jen.Err()).Op(":=").Id("s").Dot("upgrader").Dot("Upgrade").Call(jen.Id("w"), jen.Id("r"), jen.Nil())
				g.If(jen.Err().Op("!=").Nil()).Block(
					jen.Id("s").Dot("errhandler").Call(
						jen.Id("r").Dot("Context").Call(),
						jen.Id("w"),
						jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to upgrade to WebSocket: %w"), jen.Err()),
					),
					jen.Id("cancel").Call(),
					jen.Return(),
				)
				g.If(jen.Id("s").Dot("configfn").Op("!=").Nil()).Block(
					jen.Id("conn").Op("=").Id("s").Dot("configfn").Call(jen.Id("conn"), jen.Id("cancel")),
				)
				g.Defer().Id("conn").Dot("Close").Call()
				g.Line()
				streamDict := jen.Dict{
					jen.Id("r"):      jen.Id("r"),
					jen.Id("w"):      jen.Id("w"),
					jen.Id("conn"):   jen.Id("conn"),
					jen.Id("cancel"): jen.Id("cancel"),
				}
				for _, endpoint := range data.Endpoints {
					streamDict[jen.Id(lowerInitial(endpoint.Method.VarName))] = jen.Id("s").Dot(lowerInitial(endpoint.Method.VarName))
					if endpoint.Method.ServerStream != nil && (endpoint.Method.ServerStream.Kind == 3 || endpoint.Method.ServerStream.Kind == 4) {
						streamDict[jen.Id(lowerInitial(endpoint.Method.VarName)+"Endpoint")] = jen.Id("s").Dot(lowerInitial(endpoint.Method.VarName) + "Endpoint")
					}
				}
				g.Id("stream").Op(":=").Op("&").Id(streamName).Values(streamDict)
				g.Id("s").Dot("StreamHandler").Call(jen.Id("ctx"), jen.Id("stream"))
			})
	})
}

func jsonrpcServerHandlerInitSection(e *httpcodegen.EndpointData) codegen.Section {
	return codegen.MustJenniferSection("jsonrpc-server-handler-init", func(stmt *jen.Statement) {
		comment := fmt.Sprintf("%s creates a JSON-RPC handler which calls the %q service %q endpoint.", e.HandlerInit, e.ServiceName, e.Method.Name)
		codegen.Doc(stmt, comment)
		stmt.Func().Id(e.HandlerInit).
			Params(jsonrpcHandlerInitParams(e)...).
			Add(jsonrpcHandlerInitType(e)).
			BlockFunc(func(g *jen.Group) {
				if !httpcodegen.IsSSEEndpoint(e) && e.Payload != nil && e.Payload.Ref != "" {
					if !(httpcodegen.IsWebSocketEndpoint(e) && e.Method.ServerStream != nil && (e.Method.ServerStream.Kind == 3 || e.Method.ServerStream.Kind == 4)) {
						g.Id("decodeParams").Op(":=").Id(e.RequestDecoder).Call(jen.Id("mux"), jen.Id("decoder"))
					}
				}
				if !httpcodegen.IsWebSocketEndpoint(e) && needsJSONRPCResponseCapture(e) {
					g.Id("encodeResponse").Op(":=").Id(e.ResponseEncoder).Call(jen.Id("encoder"))
				}
				g.Return(
					jen.Func().
						Params(jsonrpcHandlerClosureParams(e)...).
						Params(jsonrpcHandlerClosureReturns(e)...).
						BlockFunc(func(cg *jen.Group) {
							cg.Id("ctx").Op("=").Qual("context", "WithValue").Call(jen.Id("ctx"), codegen.Expr("loom.MethodKey"), jen.Lit(e.Method.Name))
							cg.Id("ctx").Op("=").Qual("context", "WithValue").Call(jen.Id("ctx"), codegen.Expr("loom.ServiceKey"), jen.Lit(e.ServiceName))
							cg.Line()
							if httpcodegen.IsSSEEndpoint(e) {
								writeSSEHandlerInitBody(cg, e)
								return
							}
							writeJSONRPCStandardHandlerInitBody(cg, e)
						}),
				)
			})
	})
}

func jsonrpcHandlerInitParams(e *httpcodegen.EndpointData) []jen.Code {
	params := []jen.Code{
		jen.Id("endpoint").Add(codegen.TypeRef("loom.Endpoint")),
		jen.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer")),
		jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Decoder")),
	}
	if !httpcodegen.IsWebSocketEndpoint(e) {
		params = append(params,
			jen.Id("encoder").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter")).Add(codegen.TypeRef("loomhttp.Encoder")),
			jen.Id("errhandler").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter"), jen.Error()),
		)
	}
	return params
}

func jsonrpcHandlerInitType(e *httpcodegen.EndpointData) *jen.Statement {
	return jen.Func().
		Params(jsonrpcHandlerClosureParams(e)...).
		Params(jsonrpcHandlerClosureReturns(e)...)
}

func jsonrpcHandlerClosureParams(e *httpcodegen.EndpointData) []jen.Code {
	params := []jen.Code{
		jen.Id("ctx").Qual("context", "Context"),
		jen.Id("r").Op("*").Qual("net/http", "Request"),
		jen.Id("req").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
	}
	if !httpcodegen.IsWebSocketEndpoint(e) {
		params = append(params, jen.Id("w").Qual("net/http", "ResponseWriter"))
	}
	return params
}

func jsonrpcHandlerClosureReturns(e *httpcodegen.EndpointData) []jen.Code {
	if httpcodegen.IsWebSocketEndpoint(e) {
		return []jen.Code{jen.Any(), jen.Error()}
	}
	return []jen.Code{jen.Error()}
}

//nolint:maintidx // Transport initialization must encode several protocol branches in one place.
func writeSSEHandlerInitBody(g *jen.Group, e *httpcodegen.EndpointData) {
	g.Id("strm").Op(":=").Op("&").Id(e.SSE.StructName).Values(jen.Dict{
		jen.Id("w"):         jen.Id("w"),
		jen.Id("r"):         jen.Id("r"),
		jen.Id("encoder"):   jen.Id("encoder"),
		jen.Id("requestID"): jen.Id("req").Dot("ID"),
	})
	g.If(
		jen.Id("r").Dot("Method").Op("==").Qual("net/http", "MethodGet").Op("&&").Id("req").Dot("Method").Op("==").Lit("events/stream"),
	).Block(
		jen.If(
			jen.Err().Op(":=").Id("strm").Dot("open").Call(),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Return(jen.Err()),
		),
	)
	if e.Payload != nil && e.Payload.Ref != "" {
		g.Id("decodeParams").Op(":=").Id(e.RequestDecoder).Call(jen.Id("mux"), jen.Id("decoder"))
		g.List(jen.Id("params"), jen.Id("err")).Op(":=").Id("decodeParams").Call(jen.Id("r"), jen.Id("req"))
		g.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.If(jen.Id("req").Dot("ID").Op("!=").Nil().Op("&&").Id("req").Dot("ID").Op("!=").Lit("")).Block(
				jen.Id("strm").Dot("SendError").Call(jen.Id("ctx"), codegen.Expr("jsonrpc.IDToString(req.ID)"), jen.Id("err")),
			),
			jen.Return(jen.Nil()),
		)
		writePayloadIDInjection(g, e.Payload)
	}
	if e.SSE.RequestIDField != "" {
		g.If(
			jen.Id("lastEventID").Op(":=").Id("r").Dot("Header").Dot("Get").Call(jen.Lit("Last-Event-ID")),
			jen.Id("lastEventID").Op("!=").Lit(""),
		).BlockFunc(func(eg *jen.Group) {
			eg.Id("ctx").Op("=").Qual("context", "WithValue").Call(jen.Id("ctx"), jen.Lit("last-event-id"), jen.Id("lastEventID"))
			if e.Payload != nil && e.Payload.Ref != "" && e.Payload.Request != nil && e.Payload.Request.PayloadType != nil && e.Payload.Request.PayloadType.Name() == "Object" {
				eg.Id("params").Dot(e.SSE.RequestIDField).Op("=").Id("lastEventID")
			}
		})
	}
	vDict := jen.Dict{
		jen.Id("Stream"): jen.Id("strm"),
	}
	if e.Payload != nil && e.Payload.Ref != "" {
		vDict[jen.Id("Payload")] = jen.Id("params")
	}
	g.Id("v").Op(":=").Op("&").Qual(e.ServicePkgName, e.Method.ServerStream.EndpointStruct).Values(vDict)
	g.If(
		jen.List(jen.Id("_"), jen.Id("err")).Op(":=").Id("endpoint").Call(jen.Id("ctx"), jen.Id("v")),
		jen.Id("err").Op("!=").Nil(),
	).BlockFunc(func(eg *jen.Group) {
		eg.If(jen.Id("req").Dot("ID").Op("!=").Nil().Op("&&").Id("req").Dot("ID").Op("!=").Lit("")).BlockFunc(func(idg *jen.Group) {
			idg.Var().Id("en").Add(codegen.TypeRef("loom.LoomErrorNamer"))
			idg.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("en"))).Block(
				jen.Switch(jen.Id("en").Dot("LoomErrorName").Call()).Block(
					jen.Case(jen.Lit("invalid_params")).Block(
						jen.Return(jen.Id("strm").Dot("sendError").Call(
							jen.Id("ctx"),
							codegen.Expr("jsonrpc.IDToString(req.ID)"),
							jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidParams"),
							codegen.Expr("loom.ErrorSafeMessage(err)"),
							codegen.Expr("jsonrpc.NewErrorData(err)"),
						)),
					),
					jen.Case(jen.Lit("method_not_found")).Block(
						jen.Return(jen.Id("strm").Dot("sendError").Call(
							jen.Id("ctx"),
							codegen.Expr("jsonrpc.IDToString(req.ID)"),
							jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"),
							codegen.Expr("loom.ErrorSafeMessage(err)"),
							codegen.Expr("jsonrpc.NewErrorData(err)"),
						)),
					),
				),
			)
			idg.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")
			idg.If(
				jen.List(jen.Id("_"), jen.Id("ok")).Op(":=").Id("err").Assert(jen.Op("*").Add(codegen.TypeRef("loom.ServiceError"))),
				jen.Id("ok"),
			).Block(
				jen.Id("code").Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidParams"),
			)
			idg.Return(jen.Id("strm").Dot("sendError").Call(
				jen.Id("ctx"),
				codegen.Expr("jsonrpc.IDToString(req.ID)"),
				jen.Id("code"),
				codegen.Expr("loom.ErrorSafeMessage(err)"),
				codegen.Expr("jsonrpc.NewErrorData(err)"),
			))
		})
		eg.Return(jen.Nil())
	})
	g.Return(jen.Nil())
}

func writeJSONRPCStandardHandlerInitBody(g *jen.Group, e *httpcodegen.EndpointData) {
	if e.Payload != nil && e.Payload.Ref != "" {
		writeJSONRPCParamsDecode(g, e)
		writePayloadIDInjection(g, e.Payload)
	}

	if writeJSONRPCWebSocketInitReturn(g, e) {
		return
	}

	writeJSONRPCEndpointInvoke(g, e)

	if httpcodegen.IsWebSocketEndpoint(e) {
		g.Return(jen.Id("res"), jen.Id("err"))
		return
	}

	writeJSONRPCEndpointErrorHandling(g, e)

	if e.Result == nil || e.Result.Ref == "" {
		writeJSONRPCNoResultSuccess(g, e)
		return
	}

	writeJSONRPCResultSuccess(g, e)
}

func writeJSONRPCEndpointInvoke(g *jen.Group, e *httpcodegen.EndpointData) {
	callArgs := []jen.Code{jen.Id("ctx")}
	if e.Payload != nil && e.Payload.Ref != "" {
		callArgs = append(callArgs, jen.Id("params"))
	} else {
		callArgs = append(callArgs, jen.Nil())
	}

	switch {
	case httpcodegen.IsWebSocketEndpoint(e):
		g.List(jen.Id("res"), jen.Id("err")).Op(":=").Id("endpoint").Call(callArgs...)
	case e.Result == nil || e.Result.Ref == "":
		if needsJSONRPCResponseCapture(e) {
			g.List(jen.Id("res"), jen.Id("err")).Op(":=").Id("endpoint").Call(callArgs...)
			return
		}
		if e.Payload != nil && e.Payload.Ref != "" {
			g.List(jen.Id("_"), jen.Id("err")).Op("=").Id("endpoint").Call(callArgs...)
			return
		}
		g.List(jen.Id("_"), jen.Id("err")).Op(":=").Id("endpoint").Call(callArgs...)
	default:
		g.List(jen.Id("res"), jen.Id("err")).Op(":=").Id("endpoint").Call(callArgs...)
	}
}

func writeJSONRPCParamsDecode(g *jen.Group, e *httpcodegen.EndpointData) {
	if httpcodegen.IsWebSocketEndpoint(e) && e.Method.ServerStream != nil && (e.Method.ServerStream.Kind == 3 || e.Method.ServerStream.Kind == 4) {
		g.Id("decodeParams").Op(":=").Id(e.RequestDecoder).Call(jen.Id("mux"), jen.Id("decoder"))
	}
	g.List(jen.Id("params"), jen.Id("err")).Op(":=").Id("decodeParams").Call(jen.Id("r"), jen.Id("req"))
	g.If(jen.Id("err").Op("!=").Nil()).BlockFunc(func(eg *jen.Group) {
		if httpcodegen.IsWebSocketEndpoint(e) {
			eg.Return(jen.Nil(), jen.Id("err"))
			return
		}
		eg.If(jen.Id("req").Dot("ID").Op("!=").Nil().Op("&&").Id("req").Dot("ID").Op("!=").Lit("")).Block(
			jen.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError"),
			jen.If(
				jen.List(jen.Id("_"), jen.Id("ok")).Op(":=").Id("err").Assert(jen.Op("*").Add(codegen.TypeRef("loom.ServiceError"))),
				jen.Id("ok"),
			).Block(
				jen.Id("code").Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidParams"),
			),
			jen.Id("encodeJSONRPCError").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Id("req"),
				jen.Id("code"),
				codegen.Expr("loom.ErrorSafeMessage(err)"),
				codegen.Expr("jsonrpc.NewErrorData(err)"),
				jen.Id("encoder"),
				jen.Id("errhandler"),
			),
		).Else().Block(
			jen.Id("errhandler").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to decode parameters: %w"), jen.Id("err")),
			),
		)
		eg.Return(jen.Nil())
	})
}

func writeJSONRPCWebSocketInitReturn(g *jen.Group, e *httpcodegen.EndpointData) bool {
	if !httpcodegen.IsWebSocketEndpoint(e) || e.Method.ServerStream == nil || (e.Method.ServerStream.Kind != 3 && e.Method.ServerStream.Kind != 4) {
		return false
	}
	if e.Payload != nil && e.Payload.Ref != "" {
		g.Return(jen.Id("params"), jen.Nil())
	} else {
		g.Return(jen.Nil(), jen.Nil())
	}
	return true
}

func writeJSONRPCEndpointErrorHandling(g *jen.Group, e *httpcodegen.EndpointData) {
	g.If(jen.Id("err").Op("!=").Nil()).BlockFunc(func(eg *jen.Group) {
		eg.If(jen.Id("req").Dot("ID").Op("!=").Nil().Op("&&").Id("req").Dot("ID").Op("!=").Lit("")).BlockFunc(func(idg *jen.Group) {
			idg.Var().Id("en").Add(codegen.TypeRef("loom.LoomErrorNamer"))
			idg.If(jen.Op("!").Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("en"))).Block(
				writeJSONRPCEncodeErrorCall(jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")),
				jen.Return(jen.Nil()),
			)
			idg.Switch(jen.Id("en").Dot("LoomErrorName").Call()).BlockFunc(func(sg *jen.Group) {
				writeJSONRPCKnownErrorCases(sg, e)
				sg.Case(jen.Lit("invalid_params")).Block(
					writeJSONRPCEncodeErrorCall(jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidParams")),
				)
				sg.Case(jen.Lit("method_not_found")).Block(
					writeJSONRPCEncodeErrorCall(jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound")),
				)
				sg.Default().BlockFunc(writeJSONRPCDefaultEndpointError)
			})
		}).Else().Block(
			jen.Id("errhandler").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Qual("fmt", "Errorf").Call(jen.Lit("endpoint error: %w"), jen.Id("err")),
			),
		)
		eg.Return(jen.Nil())
	})
	g.Line()
}

func writeJSONRPCKnownErrorCases(g *jen.Group, e *httpcodegen.EndpointData) {
	for _, gerr := range e.Errors {
		for _, item := range gerr.Errors {
			if item.Response == nil {
				continue
			}
			g.Case(jen.Lit(item.Name)).Block(
				writeJSONRPCEncodeErrorCall(jen.Lit(item.Response.Code)),
			)
		}
	}
}

func writeJSONRPCEncodeErrorCall(code jen.Code) jen.Code {
	return jen.Id("encodeJSONRPCError").Call(
		jen.Id("ctx"),
		jen.Id("w"),
		jen.Id("req"),
		code,
		codegen.Expr("loom.ErrorSafeMessage(err)"),
		codegen.Expr("jsonrpc.NewErrorData(err)"),
		jen.Id("encoder"),
		jen.Id("errhandler"),
	)
}

func writeJSONRPCDefaultEndpointError(g *jen.Group) {
	g.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")
	g.If(
		jen.List(jen.Id("_"), jen.Id("ok")).Op(":=").Id("err").Assert(jen.Op("*").Add(codegen.TypeRef("loom.ServiceError"))),
		jen.Id("ok"),
	).Block(
		jen.Id("code").Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidParams"),
	)
	g.Add(writeJSONRPCEncodeErrorCall(jen.Id("code")))
}

func writeJSONRPCNoResultSuccess(g *jen.Group, e *httpcodegen.EndpointData) {
	g.If(
		jen.Id("req").Dot("ID").Op("==").Nil().Op("||").Id("req").Dot("ID").Op("==").Lit(""),
	).Block(
		jen.Return(jen.Nil()),
	)
	if needsJSONRPCResponseCapture(e) {
		g.Id("capture").Op(":=").Op("&").Id("jsonrpcResponseCapture").Values()
		g.If(
			jen.Err().Op(":=").Id("encodeResponse").Call(jen.Id("ctx"), jen.Id("capture"), jen.Id("res")),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("errhandler").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode transport response: %w"), jen.Err()),
			),
			jen.Return(jen.Nil()),
		)
		g.Id("copyJSONRPCResponseMetadata").Call(jen.Id("w"), jen.Id("capture"))
	}
	g.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("req").Dot("ID"), jen.Nil())
	g.If(
		jen.Err().Op(":=").Id("encoder").Call(jen.Id("ctx"), jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Id("errhandler").Call(
			jen.Id("ctx"),
			jen.Id("w"),
			jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode JSON-RPC response: %w"), jen.Err()),
		),
	)
	g.Return(jen.Nil())
}

//nolint:maintidx // Result encoding path is branch-heavy by generated transport shape.
func writeJSONRPCResultSuccess(g *jen.Group, e *httpcodegen.EndpointData) {
	g.Var().Id("id").Any()
	if e.Result.IDAttribute != "" {
		g.Id("actual").Op(":=").Id("res").Assert(codegen.TypeRef(e.Result.Ref))
		if e.Result.IDAttributeRequired {
			g.If(jen.Id("actual").Dot(e.Result.IDAttribute).Op("!=").Lit("")).Block(
				jen.Id("id").Op("=").Id("actual").Dot(e.Result.IDAttribute),
			).Else().Block(
				jen.Id("id").Op("=").Id("req").Dot("ID"),
			)
		} else {
			g.If(
				jen.Id("actual").Dot(e.Result.IDAttribute).Op("!=").Nil().Op("&&").Op("*").Id("actual").Dot(e.Result.IDAttribute).Op("!=").Lit(""),
			).Block(
				jen.Id("id").Op("=").Op("*").Id("actual").Dot(e.Result.IDAttribute),
			).Else().Block(
				jen.Id("id").Op("=").Id("req").Dot("ID"),
			)
		}
	} else {
		g.Id("id").Op("=").Id("req").Dot("ID")
	}
	g.If(
		jen.Id("id").Op("==").Nil().Op("||").Id("id").Op("==").Lit(""),
	).Block(
		jen.Return(jen.Nil()),
	)
	if needsJSONRPCResponseCapture(e) {
		g.Id("capture").Op(":=").Op("&").Id("jsonrpcResponseCapture").Values()
		g.If(
			jen.Err().Op(":=").Id("encodeResponse").Call(jen.Id("ctx"), jen.Id("capture"), jen.Id("res")),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("errhandler").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode transport response: %w"), jen.Err()),
			),
			jen.Return(jen.Nil()),
		)
		g.Id("copyJSONRPCResponseMetadata").Call(jen.Id("w"), jen.Id("capture"))
		g.Var().Id("result").Any()
		g.If(jen.Id("capture").Dot("body").Dot("Len").Call().Op(">").Lit(0)).Block(
			jen.Id("result").Op("=").Qual("encoding/json", "RawMessage").Call(jen.Id("capture").Dot("body").Dot("Bytes").Call()),
		)
		g.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("id"), jen.Id("result"))
		g.If(
			jen.Err().Op(":=").Id("encoder").Call(jen.Id("ctx"), jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("errhandler").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode JSON-RPC response: %w"), jen.Err()),
			),
		)
		g.Return(jen.Nil())
		return
	}
	success := e.Result.Responses[0]
	if success != nil && len(success.ServerBody) > 0 && success.ServerBody[0].Init != nil {
		g.Comment("Convert result to response body with proper JSON tags")
		if e.Method.ViewedResult != nil {
			g.Id("viewedRes").Op(":=").Id("res").Assert(codegen.TypeRef(e.Method.ViewedResult.FullRef))
			g.Id("body").Op(":=").Id(success.ServerBody[0].Init.Name).Call(jen.Id("viewedRes").Dot("Projected"))
		} else {
			g.Id("body").Op(":=").Id(success.ServerBody[0].Init.Name).Call(jen.Id("res").Assert(codegen.TypeRef(e.Result.Ref)))
		}
		g.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("id"), jen.Id("body"))
		g.If(
			jen.Err().Op(":=").Id("encoder").Call(jen.Id("ctx"), jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
			jen.Err().Op("!=").Nil(),
		).Block(
			jen.Id("errhandler").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode JSON-RPC response: %w"), jen.Err()),
			),
		)
		g.Return(jen.Nil())
		return
	}
	g.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("id"), jen.Id("res"))
	g.If(
		jen.Err().Op(":=").Id("encoder").Call(jen.Id("ctx"), jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
		jen.Err().Op("!=").Nil(),
	).Block(
		jen.Id("errhandler").Call(
			jen.Id("ctx"),
			jen.Id("w"),
			jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode JSON-RPC response: %w"), jen.Err()),
		),
	)
	g.Return(jen.Nil())
}

func needsJSONRPCResponseCapture(e *httpcodegen.EndpointData) bool {
	if e == nil || e.Result == nil || len(e.Result.Responses) == 0 {
		return false
	}
	success := e.Result.Responses[0]
	if success == nil {
		return false
	}
	return len(success.Headers) > 0 || len(success.Cookies) > 0
}

func writePayloadIDInjection(g *jen.Group, payload *httpcodegen.PayloadData) {
	if payload.IDAttribute == "" {
		return
	}
	if payload.IDAttributeRequired {
		g.If(jen.Id("req").Dot("ID").Op("!=").Nil()).Block(
			jen.Id("params").Dot(payload.IDAttribute).Op("=").Qual("github.com/CaliLuke/loom/jsonrpc", "IDToString").Call(jen.Id("req").Dot("ID")),
		)
		return
	}
	g.If(jen.Id("req").Dot("ID").Op("!=").Nil()).Block(
		jen.Id("idStr").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "IDToString").Call(jen.Id("req").Dot("ID")),
		jen.Id("params").Dot(payload.IDAttribute).Op("=").Op("&").Id("idStr"),
	)
}
