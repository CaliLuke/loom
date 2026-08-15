package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcServerHandlerSection(data *httpcodegen.ServiceData, mixed bool) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-handler", func(stmt *jen.Statement) {
		addJSONRPCServeHTTPSection(stmt, data, mixed)
		addJSONRPCHandleHTTPSection(stmt, data)
		addJSONRPCHandleSingleSection(stmt, data)
		addJSONRPCHandleBatchSection(stmt, data)
		addJSONRPCProcessRequestSection(stmt, data)
		addJSONRPCBatchWriterSection(stmt)
	})
}

func jsonrpcWebSocketServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-websocket-server-handler", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"

		stmt.Comment("serveHTTP handles WebSocket JSON-RPC requests before server middleware.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("serveHTTP").
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
				g.Id("wsconn").Op(":=").Id("loomhttp").Dot("NewWebSocketStream").Call(jen.Id("conn"), jen.Id("s").Dot("streamWritePolicy"))
				g.Line()
				streamDict := jen.Dict{
					jen.Id("r"):      jen.Id("r"),
					jen.Id("w"):      jen.Id("w"),
					jen.Id("conn"):   jen.Id("wsconn"),
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
	return codegen.NewJenniferSection("jsonrpc-server-handler-init", func(stmt *jen.Statement) {
		comment := fmt.Sprintf("%s creates a JSON-RPC handler which calls the %q service %q endpoint.", e.HandlerInit, e.ServiceName, e.Method.Name)
		codegen.Doc(stmt, comment)
		stmt.Func().Id(e.HandlerInit).
			Params(jsonrpcHandlerInitParams(e)...).
			Add(jsonrpcHandlerInitType(e)).
			BlockFunc(func(g *jen.Group) {
				if e.Payload != nil && e.Payload.Ref != "" {
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
	if httpcodegen.IsSSEEndpoint(e) {
		params = append(params, jen.Id("streamWritePolicy").Add(codegen.TypeRef("loomhttp.StreamWritePolicy")))
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
		jen.Id("w"):            jen.Id("w"),
		jen.Id("r"):            jen.Id("r"),
		jen.Id("writer"):       jen.Id("loomhttp.NewSSEStreamWriter").Call(jen.Id("w"), jen.Id("r").Dot("Context").Call(), jen.Id("loomtransport.TransportJSONRPC"), jen.Id("streamWritePolicy")),
		jen.Id("encoder"):      jen.Id("encoder"),
		jen.Id("requestID"):    jen.Id("req").Dot("ID"),
		jen.Id("requestHasID"): jen.Id("req").Dot("HasID"),
	})
	if e.Method.Name == "events/stream" {
		g.If(
			jen.Id("r").Dot("Method").Op("==").Qual("net/http", "MethodGet").Op("&&").Id("req").Dot("Method").Op("==").Lit("events/stream"),
		).Block(
			jen.If(
				jen.Err().Op(":=").Id("strm").Dot("Open").Call(jen.Id("r").Dot("Context").Call()),
				jen.Err().Op("!=").Nil(),
			).Block(
				jen.Return(jen.Err()),
			),
		)
	}
	if e.Payload != nil && e.Payload.Ref != "" {
		g.List(jen.Id("params"), jen.Id("err")).Op(":=").Id("decodeParams").Call(jen.Id("r"), jen.Id("req"))
		g.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.If(jen.Id("req").Dot("HasID")).BlockFunc(func(eg *jen.Group) {
				eg.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")
				eg.Var().Id("serviceError").Op("*").Add(codegen.TypeRef("loom.ServiceError"))
				eg.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("serviceError"))).Block(
					jen.Id("code").Op("=").Id("jsonrpcErrorCodeForServiceError").Call(jen.Id("serviceError")),
				)
				eg.Return(jen.Id("strm").Dot("sendError").Call(
					jen.Id("ctx"),
					jen.Id("req").Dot("ID"),
					jen.Id("code"),
					codegen.Expr("loom.ErrorSafeMessage(err)"),
					codegen.Expr("jsonrpc.NewErrorData(err)"),
				))
			}),
			jen.Return(jen.Nil()),
		)
		writePayloadIDInjection(g, e.Payload)
	}
	if e.SSE.RequestIDField != "" {
		g.If(
			jen.Id("lastEventID").Op(":=").Id("r").Dot("Header").Dot("Get").Call(jen.Lit("Last-Event-ID")),
			jen.Id("lastEventID").Op("!=").Lit(""),
		).BlockFunc(func(eg *jen.Group) {
			eg.Id("ctx").Op("=").Qual("context", "WithValue").Call(
				jen.Id("ctx"),
				codegen.TypeRef("loomhttp.LastEventIDKey"),
				jen.Id("lastEventID"),
			)
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
		eg.If(jen.Id("req").Dot("HasID")).BlockFunc(func(idg *jen.Group) {
			idg.Var().Id("en").Add(codegen.TypeRef("loom.LoomErrorNamer"))
			idg.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("en"))).Block(
				jen.Switch(jen.Id("en").Dot("LoomErrorName").Call()).Block(
					jen.Case(jen.Lit("invalid_params")).Block(
						jen.Return(jen.Id("strm").Dot("sendError").Call(
							jen.Id("ctx"),
							jen.Id("req").Dot("ID"),
							jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidParams"),
							codegen.Expr("loom.ErrorSafeMessage(err)"),
							codegen.Expr("jsonrpc.NewErrorData(err)"),
						)),
					),
					jen.Case(jen.Lit("method_not_found")).Block(
						jen.Return(jen.Id("strm").Dot("sendError").Call(
							jen.Id("ctx"),
							jen.Id("req").Dot("ID"),
							jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"),
							codegen.Expr("loom.ErrorSafeMessage(err)"),
							codegen.Expr("jsonrpc.NewErrorData(err)"),
						)),
					),
				),
			)
			idg.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")
			idg.Var().Id("serviceError").Op("*").Add(codegen.TypeRef("loom.ServiceError"))
			idg.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("serviceError"))).Block(
				jen.Id("code").Op("=").Id("jsonrpcErrorCodeForServiceError").Call(jen.Id("serviceError")),
			)
			idg.Return(jen.Id("strm").Dot("sendError").Call(
				jen.Id("ctx"),
				jen.Id("req").Dot("ID"),
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
		eg.Add(loomtransportRef("RequestObserverFromContext")).Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef("ReasonInvalidJSONRPCParams"))
		if httpcodegen.IsWebSocketEndpoint(e) {
			eg.Return(jen.Nil(), jen.Id("err"))
			return
		}
		eg.If(jen.Id("req").Dot("HasID")).Block(
			jen.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError"),
			jen.Var().Id("serviceError").Op("*").Add(codegen.TypeRef("loom.ServiceError")),
			jen.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("serviceError"))).Block(
				jen.Id("code").Op("=").Id("jsonrpcErrorCodeForServiceError").Call(jen.Id("serviceError")),
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
		eg.Add(loomtransportRef("RequestObserverFromContext")).Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef("ReasonHandlerError"))
		eg.If(jen.Id("req").Dot("HasID")).BlockFunc(func(idg *jen.Group) {
			idg.Var().Id("en").Add(codegen.TypeRef("loom.LoomErrorNamer"))
			idg.If(jen.Op("!").Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("en"))).Block(
				jen.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError"),
				jen.Var().Id("serviceError").Op("*").Add(codegen.TypeRef("loom.ServiceError")),
				jen.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("serviceError"))).Block(
					jen.Id("code").Op("=").Id("jsonrpcErrorCodeForServiceError").Call(jen.Id("serviceError")),
				),
				writeJSONRPCEncodeErrorCall(jen.Id("code")),
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
			g.Case(jen.Lit(item.Name)).BlockFunc(func(cg *jen.Group) {
				writeJSONRPCMappedError(cg, item)
			})
		}
	}
}

func writeJSONRPCEncodeErrorCall(code jen.Code) jen.Code {
	return writeJSONRPCEncodeErrorCallWithData(code, codegen.Expr("jsonrpc.NewErrorData(err)"))
}

func writeJSONRPCEncodeErrorCallWithData(code, data jen.Code) jen.Code {
	return jen.Id("encodeJSONRPCError").Call(
		jen.Id("ctx"),
		jen.Id("w"),
		jen.Id("req"),
		code,
		codegen.Expr("loom.ErrorSafeMessage(err)"),
		data,
		jen.Id("encoder"),
		jen.Id("errhandler"),
	)
}

func writeJSONRPCMappedError(g *jen.Group, item *httpcodegen.ErrorData) {
	response := item.Response
	if response.EncodePlan == nil || !response.EncodePlan.HasBody {
		g.Add(writeJSONRPCEncodeErrorCall(jen.Lit(response.Code)))
		return
	}

	g.Var().Id("res").Add(codegen.TypeRef(item.Ref))
	g.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("res"))).BlockFunc(func(cg *jen.Group) {
		body := response.EncodePlan.FirstBody
		data := jen.Code(jen.Id("res"))
		if body.Init != nil {
			args := make([]jen.Code, 0, len(body.Init.ServerArgs))
			for _, arg := range body.Init.ServerArgs {
				args = append(args, codegen.Expr(arg.Ref))
			}
			cg.Id("data").Op(":=").Id(body.Init.Name).Call(args...)
			data = jen.Id("data")
		} else if response.ResultAttr != "" {
			data = jen.Id("res").Dot(response.ResultAttr)
		}
		cg.Add(writeJSONRPCEncodeErrorCallWithData(jen.Lit(response.Code), data))
		cg.Break()
	})
	g.Add(writeJSONRPCEncodeErrorCall(jen.Lit(response.Code)))
}

func writeJSONRPCDefaultEndpointError(g *jen.Group) {
	g.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")
	g.Var().Id("serviceError").Op("*").Add(codegen.TypeRef("loom.ServiceError"))
	g.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("serviceError"))).Block(
		jen.Id("code").Op("=").Id("jsonrpcErrorCodeForServiceError").Call(jen.Id("serviceError")),
	)
	g.Add(writeJSONRPCEncodeErrorCall(jen.Id("code")))
}

func writeJSONRPCNoResultSuccess(g *jen.Group, e *httpcodegen.EndpointData) {
	g.If(
		jen.Op("!").Id("req").Dot("HasID"),
	).Block(
		jen.Return(jen.Nil()),
	)
	if needsJSONRPCResponseCapture(e) {
		g.Id("capture").Op(":=").Op("&").Id("jsonrpcResponseCapture").Values()
		g.If(
			jen.Err().Op(":=").Id("encodeResponse").Call(jen.Id("ctx"), jen.Id("capture"), jen.Id("res")),
			jen.Err().Op("!=").Nil(),
		).Block(
			loomtransportRef("RequestObserverFromContext").Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef("ReasonResponseWriteFailed")),
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
	g.Id("id").Op(":=").Id("req").Dot("ID")
	g.If(
		jen.Op("!").Id("req").Dot("HasID"),
	).Block(
		jen.Return(jen.Nil()),
	)
	if needsJSONRPCResponseCapture(e) {
		g.Id("capture").Op(":=").Op("&").Id("jsonrpcResponseCapture").Values()
		g.If(
			jen.Err().Op(":=").Id("encodeResponse").Call(jen.Id("ctx"), jen.Id("capture"), jen.Id("res")),
			jen.Err().Op("!=").Nil(),
		).Block(
			loomtransportRef("RequestObserverFromContext").Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef("ReasonResponseWriteFailed")),
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
			loomtransportRef("RequestObserverFromContext").Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef("ReasonResponseWriteFailed")),
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
			loomtransportRef("RequestObserverFromContext").Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef("ReasonResponseWriteFailed")),
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
		loomtransportRef("RequestObserverFromContext").Call(jen.Id("ctx")).Dot("Fail").Call(loomtransportRef("ReasonResponseWriteFailed")),
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

func serviceNeedsJSONRPCResponseCapture(data *httpcodegen.ServiceData) bool {
	for _, endpoint := range data.Endpoints {
		if needsJSONRPCResponseCapture(endpoint) {
			return true
		}
	}
	return false
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
