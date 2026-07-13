package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcWebSocketServerSections(data *httpcodegen.ServiceData) []codegen.Section {
	return []codegen.Section{
		jsonrpcWebSocketServerStructSection(data),
		jsonrpcWebSocketServerWrapperSection(data),
		jsonrpcWebSocketServerSendSection(data),
		jsonrpcWebSocketServerRecvSection(data),
		jsonrpcWebSocketServerCloseSection(data),
		jsonrpcWebSocketServerServiceErrorClassifierSection(),
	}
}

func jsonrpcWebSocketServerServiceErrorClassifierSection() codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-websocket-service-error-classifier", func(stmt *jen.Statement) {
		writeJSONRPCServiceErrorClassifier(stmt)
	})
}

func jsonrpcWebSocketServerStructSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-websocket-struct", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		codegen.Doc(stmt, fmt.Sprintf("%s implements the Stream interface.", streamName))
		stmt.Type().Id(streamName).StructFunc(func(g *jen.Group) {
			for _, ed := range data.Endpoints {
				g.Comment(codegen.Comment(fmt.Sprintf("%s decodes requests for the %s method", lowerInitial(ed.Method.VarName), ed.Method.Name)))
				g.Id(lowerInitial(ed.Method.VarName)).Func().
					Params(jen.Qual("context", "Context"), jen.Op("*").Qual("net/http", "Request"), jen.Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest")).
					Params(jen.Any(), jen.Error())
				if ed.Method.ServerStream != nil && (ed.Method.ServerStream.Kind == expr.ServerStreamKind || ed.Method.ServerStream.Kind == expr.BidirectionalStreamKind) {
					g.Comment(codegen.Comment(fmt.Sprintf("%sEndpoint is the endpoint for the %s method", lowerInitial(ed.Method.VarName), ed.Method.Name)))
					g.Id(lowerInitial(ed.Method.VarName) + "Endpoint").Add(codegen.TypeRef("loom.Endpoint"))
				}
			}
			g.Comment("cancel is the context cancellation function which cancels the request context when invoked.")
			g.Id("cancel").Qual("context", "CancelFunc")
			g.Comment("w is the HTTP response writer used in upgrading the connection.")
			g.Id("w").Qual("net/http", "ResponseWriter")
			g.Comment("r is the HTTP request.")
			g.Id("r").Op("*").Qual("net/http", "Request")
			g.Comment("conn owns the websocket connection lifecycle.")
			g.Id("conn").Op("*").Id("loomhttp.WebSocketStream")
		})
	})
}

func jsonrpcWebSocketServerWrapperSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-websocket-stream-wrapper", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		first := true
		for _, ed := range data.Endpoints {
			if ed.Method.ServerStream == nil || (ed.Method.ServerStream.Kind != expr.ServerStreamKind && ed.Method.ServerStream.Kind != expr.BidirectionalStreamKind) {
				continue
			}
			if !first {
				stmt.Line()
			}
			first = false
			name := lowerInitial(ed.Method.VarName)
			stmt.Comment(fmt.Sprintf("%sStreamWrapper wraps the JSON-RPC stream to provide a method-specific interface.", name)).Line()
			stmt.Type().Id(name+"StreamWrapper").Struct(
				jen.Id("stream").Op("*").Id(streamName),
				jen.Id("requestID").Any(),
			)
			stmt.Line()
			stmt.Func().Params(jen.Id("w").Op("*").Id(name+"StreamWrapper")).
				Id("SendNotification").
				Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("res").Add(codegen.TypeRef(ed.Result.Ref))).
				Error().
				Block(
					jen.Return(jen.Id("w").Dot("stream").Dot("Send"+ed.Method.VarName+"Notification").Call(jen.Id("ctx"), jen.Id("res"))),
				)
			stmt.Line()
			stmt.Func().Params(jen.Id("w").Op("*").Id(name+"StreamWrapper")).
				Id("SendResponse").
				Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("res").Add(codegen.TypeRef(ed.Result.Ref))).
				Error().
				Block(
					jen.Return(jen.Id("w").Dot("stream").Dot("Send"+ed.Method.VarName+"Response").Call(jen.Id("ctx"), jen.Id("w").Dot("requestID"), jen.Id("res"))),
				)
			stmt.Line()
			stmt.Func().Params(jen.Id("w").Op("*").Id(name+"StreamWrapper")).
				Id("SendError").
				Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("err").Error()).
				Error().
				Block(
					jen.Return(jen.Id("w").Dot("stream").Dot("SendError").Call(jen.Id("ctx"), jen.Id("w").Dot("requestID"), jen.Id("err"))),
				)
			stmt.Line()
			stmt.Func().Params(jen.Id("w").Op("*").Id(name + "StreamWrapper")).
				Id("Close").
				Params().
				Error().
				Block(
					jen.Return(jen.Id("w").Dot("stream").Dot("Close").Call()),
				)
		}
	})
}

func jsonrpcWebSocketServerSendSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-websocket-send", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		for _, ed := range data.Endpoints {
			if ed.Result == nil || ed.Result.Ref == "" {
				continue
			}
			addJSONRPCWebSocketResultSendMethods(stmt, streamName, ed)
		}
		sendErrorDecl := &jen.Statement{}
		codegen.Doc(sendErrorDecl, "SendError streams JSON-RPC errors.")
		sendErrorDecl.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("SendError").
			Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("id").Any(), jen.Id("err").Error()).
			Error().
			BlockFunc(func(g *jen.Group) {
				writeStreamErrorDataSwitch(g, allErrors(data), jen.Id("id"))
			})
		stmt.Add(sendErrorDecl)
		stmt.Line()
		sendDecl := &jen.Statement{}
		codegen.Doc(sendDecl, "send writes a JSON-RPC response to the websocket connection.")
		sendDecl.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("send").
			Params(jen.Id("id").Any(), jen.Id("method").String(), jen.Id("result").Any()).
			Error().
			Block(
				jen.If(jen.Id("id").Op("==").Nil().Op("||").Id("id").Op("==").Lit("")).Block(
					jen.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(jen.Id("s").Dot("r").Dot("Context").Call(), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeNotification").Call(jen.Id("method"), jen.Id("result")))),
				),
				jen.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(jen.Id("s").Dot("r").Dot("Context").Call(), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("id"), jen.Id("result")))),
			)
		stmt.Add(sendDecl)
		stmt.Line()
		sendErrorResponseDecl := &jen.Statement{}
		codegen.Doc(sendErrorResponseDecl, "sendError sends a JSON-RPC error response to the websocket connection.")
		sendErrorResponseDecl.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("sendError").
			Params(
				jen.Id("ctx").Qual("context", "Context"),
				jen.Id("id").Any(),
				jen.Id("code").Qual("github.com/CaliLuke/loom/jsonrpc", "Code"),
				jen.Id("message").String(),
				jen.Id("data").Any(),
			).
			Error().
			Block(
				jen.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(jen.Id("id"), jen.Id("code"), jen.Id("message"), jen.Id("data")),
				jen.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(jen.Id("ctx"), jen.Id("response"))),
			)
		stmt.Add(sendErrorResponseDecl)
	})
}

func jsonrpcSSEStreamFields() []jen.Code {
	return []jen.Code{
		jen.Comment("writer owns the serialized SSE response lifecycle."),
		jen.Id("writer").Op("*").Id("loomhttp.SSEStreamWriter"),
		jen.Comment("w is the HTTP response writer used to send the SSE events."),
		jen.Id("w").Qual("net/http", "ResponseWriter"),
		jen.Comment("r is the HTTP request."),
		jen.Id("r").Op("*").Qual("net/http", "Request"),
		jen.Comment("encoder is the response encoder."),
		jen.Id("encoder").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter")).Add(codegen.TypeRef("loomhttp.Encoder")),
		jen.Comment("decoder is the request decoder."),
		jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Decoder")),
	}
}

func addJSONRPCWebSocketResultSendMethods(stmt *jen.Statement, streamName string, ed *httpcodegen.EndpointData) {
	addJSONRPCWebSocketSendMethod(stmt, streamName, ed, true)
	stmt.Line()
	addJSONRPCWebSocketSendMethod(stmt, streamName, ed, false)
	stmt.Line()
}

func addJSONRPCWebSocketSendMethod(stmt *jen.Statement, streamName string, ed *httpcodegen.EndpointData, notification bool) {
	methodName, doc := jsonrpcWebSocketSendMethodMeta(ed, notification)
	decl := &jen.Statement{}
	codegen.Doc(decl, doc)
	if notification {
		decl.
			Func().
			Params(jen.Id("s").Op("*").Id(streamName)).
			Id(methodName).
			Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("result").Add(codegen.TypeRef(ed.Result.Ref))).
			Error().
			BlockFunc(func(g *jen.Group) {
				writeStreamResultBodyInit(g, "body", "result", ed)
				g.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(
					jen.Id("ctx"),
					jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeNotification").Call(jen.Lit(ed.Method.Name), jen.Id("body")),
				))
			})
		stmt.Add(decl)
		return
	}
	decl.
		Func().
		Params(jen.Id("s").Op("*").Id(streamName)).
		Id(methodName).
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("id").Any(),
			jen.Id("result").Add(codegen.TypeRef(ed.Result.Ref)),
		).
		Error().
		BlockFunc(func(g *jen.Group) {
			writeStreamResultBodyInit(g, "body", "result", ed)
			g.Return(jen.Id("s").Dot("conn").Dot("WriteJSON").Call(
				jen.Id("ctx"),
				jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("id"), jen.Id("body")),
			))
		})
	stmt.Add(decl)
}

func jsonrpcWebSocketSendMethodMeta(ed *httpcodegen.EndpointData, notification bool) (string, string) {
	if notification {
		return "Send" + ed.Method.VarName + "Notification",
			fmt.Sprintf("Send%sNotification sends a JSON-RPC notification for the %s method.", ed.Method.VarName, ed.Method.Name)
	}
	return "Send" + ed.Method.VarName + "Response",
		fmt.Sprintf("Send%sResponse sends a JSON-RPC response for the %s method.", ed.Method.VarName, ed.Method.Name)
}

func jsonrpcWebSocketServerRecvSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-websocket-recv", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		codegen.Doc(stmt, fmt.Sprintf("Recv reads JSON-RPC requests from the %s service stream.", data.Service.Name))
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("Recv").
			Params(jen.Id("ctx").Qual("context", "Context")).
			Error().
			Block(
				jen.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
				jen.If(
					jen.Err().Op(":=").Id("s").Dot("conn").Dot("ReadJSON").Call(jen.Id("ctx"), jen.Op("&").Id("req")),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.If(
						jen.Qual("github.com/gorilla/websocket", "IsUnexpectedCloseError").Call(
							jen.Err(),
							jen.Qual("github.com/gorilla/websocket", "CloseGoingAway"),
							jen.Qual("github.com/gorilla/websocket", "CloseAbnormalClosure"),
						),
					).Block(
						jen.Return(jen.Err()),
					),
					jen.If(
						jen.Err().Op(":=").Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Nil(), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "ParseError"), jen.Lit("Parse error"), jen.Nil()),
						jen.Err().Op("!=").Nil(),
					).Block(
						jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to send parse error: %w"), jen.Err())),
					),
					jen.Return(jen.Nil()),
				),
				jen.Return(jen.Id("s").Dot("processRequest").Call(jen.Id("ctx"), jen.Op("&").Id("req"))),
			)
		stmt.Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("processRequest").
			Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("req").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest")).
			Error().
			BlockFunc(func(g *jen.Group) {
				writeWebSocketRequestValidation(g)
				g.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(sg *jen.Group) {
					for _, ed := range data.Endpoints {
						writeWebSocketRequestCase(sg, ed)
					}
					sg.Default().Block(
						jen.If(jen.Id("req").Dot("HasID")).Block(
							jen.Return(jen.Id("s").Dot("sendError").Call(jen.Id("ctx"), jen.Id("req").Dot("ID"), jen.Qual("github.com/CaliLuke/loom/jsonrpc", "MethodNotFound"), jen.Lit("Method not found"), jen.Nil())),
						),
						jen.Return(jen.Nil()),
					)
				})
			})
	})
}

func jsonrpcWebSocketServerCloseSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-websocket-close", func(stmt *jen.Statement) {
		streamName := lowerInitial(data.Service.StructName) + "Stream"
		codegen.Doc(stmt, fmt.Sprintf("Close closes the %s service websocket connection.", data.Service.Name))
		stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
			Id("Close").
			Params().
			Error().
			Block(
				jen.Var().Id("err").Error(),
				jen.If(jen.Id("s").Dot("conn").Op("==").Nil()).Block(
					jen.Return(jen.Nil()),
				),
				jen.If(
					jen.Id("err").Op("=").Id("s").Dot("conn").Dot("WriteClose").Call(jen.Lit("server closing connection")),
					jen.Id("err").Op("!=").Nil(),
				).Block(
					jen.Return(jen.Id("err")),
				),
				jen.Return(jen.Id("s").Dot("conn").Dot("Close").Call()),
			)
	})
}

func streamResultBodyInit(resultVar string, ed *httpcodegen.EndpointData) string {
	if ed.Result != nil && len(ed.Result.Responses) > 0 && len(ed.Result.Responses[0].ServerBody) > 0 && ed.Result.Responses[0].ServerBody[0].Init != nil {
		return fmt.Sprintf("body := %s(%s)", ed.Result.Responses[0].ServerBody[0].Init.Name, resultVar)
	}
	return fmt.Sprintf("body := %s", resultVar)
}

func streamErrorSwitch(prefix string, groups []*httpcodegen.ErrorGroupData) string {
	parts := make([]string, 0, len(groups)+8)
	if len(groups) > 0 {
		parts = append(parts,
			"\tvar en loom.LoomErrorNamer\n",
			"\tif !errors.As(err, &en) {\n",
			"\t\tcode := jsonrpc.InternalError\n",
			"\t\tvar serviceError *loom.ServiceError\n",
			"\t\tif errors.As(err, &serviceError) {\n",
			"\t\t\tcode = jsonrpcErrorCodeForServiceError(serviceError)\n",
			"\t\t}\n",
			"\t\t"+prefix+"code, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n",
			"\t}\n",
			"\tswitch en.LoomErrorName() {\n",
		)
		for _, gerr := range groups {
			for _, e := range gerr.Errors {
				if e.Response == nil {
					continue
				}
				parts = append(parts,
					fmt.Sprintf("\tcase %q:\n", e.Name),
					fmt.Sprintf("\t\t%s%d, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n", prefix, e.Response.Code),
				)
			}
		}
		parts = append(parts,
			"\tdefault:\n",
			"\t\tcode := jsonrpc.InternalError\n",
			"\t\tvar serviceError *loom.ServiceError\n",
			"\t\tif errors.As(err, &serviceError) {\n",
			"\t\t\tcode = jsonrpcErrorCodeForServiceError(serviceError)\n",
			"\t\t}\n",
			"\t\t"+prefix+"code, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n",
			"\t}\n",
		)
		return strings.Join(parts, "")
	}
	parts = append(parts,
		"\tcode := jsonrpc.InternalError\n",
		"\tvar serviceError *loom.ServiceError\n",
		"\tif errors.As(err, &serviceError) {\n",
		"\t\tcode = jsonrpcErrorCodeForServiceError(serviceError)\n",
		"\t}\n",
		"\t"+prefix+"code, loom.ErrorSafeMessage(err), jsonrpc.NewErrorData(err))\n",
	)
	return strings.Join(parts, "")
}

func writeStreamResultBodyInit(g *jen.Group, targetName, resultVar string, ed *httpcodegen.EndpointData) {
	if ed.Result != nil && len(ed.Result.Responses) > 0 && len(ed.Result.Responses[0].ServerBody) > 0 && ed.Result.Responses[0].ServerBody[0].Init != nil {
		g.Id(targetName).Op(":=").Id(ed.Result.Responses[0].ServerBody[0].Init.Name).Call(jen.Id(resultVar))
		return
	}
	g.Id(targetName).Op(":=").Id(resultVar)
}

func writeStreamErrorDataSwitch(g *jen.Group, errs []*httpcodegen.ErrorData, targetID jen.Code) {
	writeDefault := func(group *jen.Group) {
		group.Id("code").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")
		group.Var().Id("serviceError").Op("*").Id("loom").Dot("ServiceError")
		group.If(jen.Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("serviceError"))).Block(
			jen.Id("code").Op("=").Id("jsonrpcErrorCodeForServiceError").Call(jen.Id("serviceError")),
		)
		group.Return(
			jen.Id("s").Dot("sendError").Call(
				jen.Id("ctx"),
				targetID,
				jen.Id("code"),
				jen.Id("loom").Dot("ErrorSafeMessage").Call(jen.Id("err")),
				jen.Qual("github.com/CaliLuke/loom/jsonrpc", "NewErrorData").Call(jen.Id("err")),
			),
		)
	}
	if len(errs) == 0 {
		writeDefault(g)
		return
	}
	g.Var().Id("en").Add(codegen.TypeRef("loom.LoomErrorNamer"))
	g.If(jen.Op("!").Qual("errors", "As").Call(jen.Id("err"), jen.Op("&").Id("en"))).BlockFunc(writeDefault)
	g.Switch(jen.Id("en").Dot("LoomErrorName").Call()).BlockFunc(func(sg *jen.Group) {
		for _, e := range errs {
			if e.Response == nil {
				continue
			}
			sg.Case(jen.Lit(e.Name)).Block(
				jen.Return(
					jen.Id("s").Dot("sendError").Call(
						jen.Id("ctx"),
						targetID,
						jen.Lit(e.Response.Code),
						jen.Id("loom").Dot("ErrorSafeMessage").Call(jen.Id("err")),
						jen.Qual("github.com/CaliLuke/loom/jsonrpc", "NewErrorData").Call(jen.Id("err")),
					),
				),
			)
		}
		sg.Default().BlockFunc(writeDefault)
	})
}

func writeSSEServiceStreamSend(stmt *jen.Statement, data *httpcodegen.ServiceData, streamName string) {
	if !hasAnyStreamingResults(data.Endpoints) {
		return
	}
	codegen.Doc(stmt, "Send sends an event (notification or response) to the client.")
	stmt.Comment("For notifications, the result should not have an ID field.").Line()
	stmt.Comment("For responses, the result must have an ID field.").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
		Id("Send").
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("event").Add(codegen.TypeRef(data.Service.PkgName+".Event"))).
		Error().
		BlockFunc(func(g *jen.Group) {
			g.Switch(jen.Id("v").Op(":=").Id("event").Assert(jen.Type())).BlockFunc(func(sg *jen.Group) {
				for _, ed := range dedupeSSEEndpoints(data.Endpoints) {
					if ed.Method.ServerStream == nil || ed.Method.Result == "" {
						continue
					}
					sg.Case(codegen.Expr(ed.SSE.EventTypeRef)).BlockFunc(func(cg *jen.Group) {
						writeStreamResultBodyInit(cg, "body", "v", ed)
						cg.Var().Id("message").Map(jen.String()).Any()
						if sseEventCanBeResponse(ed) {
							cg.Var().Id("id").String()
							cg.Var().Id("isResponse").Bool()
							writeSSEServiceResponseIDResolution(cg, ed)
							cg.If(jen.Id("isResponse")).Block(
								jen.Id("resp").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeSuccessResponse").Call(jen.Id("id"), jen.Id("body")),
								jen.Id("message").Op("=").Map(jen.String()).Any().Values(jen.Dict{
									jen.Lit("jsonrpc"): jen.Id("resp").Dot("JSONRPC"),
									jen.Lit("id"):      jen.Id("resp").Dot("ID"),
									jen.Lit("result"):  jen.Id("resp").Dot("Result"),
								}),
							).Else().Block(
								jen.Id("message").Op("=").Map(jen.String()).Any().Values(jen.Dict{
									jen.Lit("jsonrpc"): jen.Lit("2.0"),
									jen.Lit("method"):  jen.Lit(sseNotificationMethod(ed)),
									jen.Lit("params"):  jen.Id("body"),
								}),
							)
						} else {
							cg.Id("message").Op("=").Map(jen.String()).Any().Values(jen.Dict{
								jen.Lit("jsonrpc"): jen.Lit("2.0"),
								jen.Lit("method"):  jen.Lit(sseNotificationMethod(ed)),
								jen.Lit("params"):  jen.Id("body"),
							})
						}
						cg.Return(jen.Id("s").Dot("sendSSEEvent").Call(jen.Lit("message"), jen.Id("message")))
					})
				}
				sg.Default().Block(
					jen.Return(jen.Qual("fmt", "Errorf").Call(jen.Lit("unknown event type: %T"), jen.Id("event"))),
				)
			})
		})
}

func writeSSEServiceResponseIDResolution(g *jen.Group, ed *httpcodegen.EndpointData) {
	if ed.Result == nil || ed.Result.IDAttribute == "" {
		return
	}
	if ed.Result.IDAttributeRequired {
		g.If(jen.Id("v").Dot(ed.Result.IDAttribute).Op("!=").Lit("")).Block(
			jen.Id("id").Op("=").Id("v").Dot(ed.Result.IDAttribute),
			jen.Id("isResponse").Op("=").True(),
			jen.Id("v").Dot(ed.Result.IDAttribute).Op("=").Lit(""),
		)
		return
	}
	g.If(
		jen.Id("v").Dot(ed.Result.IDAttribute).Op("!=").Nil().Op("&&").
			Op("*").Id("v").Dot(ed.Result.IDAttribute).Op("!=").Lit(""),
	).Block(
		jen.Id("id").Op("=").Op("*").Id("v").Dot(ed.Result.IDAttribute),
		jen.Id("isResponse").Op("=").True(),
		jen.Id("v").Dot(ed.Result.IDAttribute).Op("=").Nil(),
	)
}

func sseEventCanBeResponse(ed *httpcodegen.EndpointData) bool {
	return ed.Result != nil && ed.Result.IDAttribute != ""
}

func writeSSEServiceStreamSendError(stmt *jen.Statement, data *httpcodegen.ServiceData, streamName string) {
	codegen.Doc(stmt, "SendError sends a JSON-RPC error response.")
	stmt.Func().Params(jen.Id("s").Op("*").Id(streamName)).
		Id("SendError").
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("id").Any(), jen.Id("err").Error()).
		Error().
		BlockFunc(func(g *jen.Group) {
			writeStreamErrorDataSwitch(g, allErrors(data), jen.Id("id"))
		})
}

func serviceHasErrors(methods []*service.MethodData) bool {
	for _, m := range methods {
		if len(m.Errors) > 0 {
			return true
		}
	}
	return false
}

func hasAnyStreamingResults(endpoints []*httpcodegen.EndpointData) bool {
	for _, ed := range endpoints {
		if ed.Method.ServerStream != nil && ed.Method.Result != "" {
			return true
		}
	}
	return false
}

func dedupeSSEEndpoints(endpoints []*httpcodegen.EndpointData) []*httpcodegen.EndpointData {
	seen := make(map[string]struct{})
	out := make([]*httpcodegen.EndpointData, 0, len(endpoints))
	for _, e := range endpoints {
		if e == nil || e.SSE == nil || e.SSE.EventTypeRef == "" {
			continue
		}
		if _, ok := seen[e.SSE.EventTypeRef]; ok {
			continue
		}
		seen[e.SSE.EventTypeRef] = struct{}{}
		out = append(out, e)
	}
	return out
}
