package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func jsonrpcClientStructSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-client-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s lists the %s service endpoint HTTP clients.", data.ClientStruct, data.Service.Name))
		stmt.Type().Id(data.ClientStruct).StructFunc(func(g *jen.Group) {
			g.Comment(codegen.Comment(fmt.Sprintf("Doer is the HTTP client used to make requests to the %s service.", data.Service.Name)))
			g.Id("Doer").Add(codegen.TypeRef("loomhttp.Doer"))
			for _, endpoint := range data.Endpoints {
				if !httpcodegen.IsSSEEndpoint(endpoint) {
					continue
				}
				g.Comment(codegen.Comment(fmt.Sprintf("%s Doer is the HTTP client used to make requests to the %s endpoint.", endpoint.Method.VarName, endpoint.Method.Name)))
				g.Id(endpoint.Method.VarName + "Doer").Add(codegen.TypeRef("loomhttp.Doer"))
			}
			g.Comment("RestoreResponseBody controls whether the response bodies are reset after")
			g.Comment("decoding so they can be read again.")
			g.Id("RestoreResponseBody").Bool()
			g.Line()
			g.Id("scheme").String()
			g.Id("host").String()
			g.Id("encoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Encoder"))
			g.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder"))
			if httpcodegen.HasWebSocket(data) {
				g.Id("dialer").Add(codegen.TypeRef("loomhttp.Dialer"))
				g.Id("configfn").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc"))
				g.Line()
				g.Id("connMu").Qual("sync", "RWMutex")
				g.Id("conn").Op("*").Qual("github.com/gorilla/websocket", "Conn")
				g.Id("closed").Qual("sync/atomic", "Bool")
				g.Line()
				g.Comment("Stream configuration (shared by all WebSocket streams)")
				g.Id("streamConfig").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "StreamConfig")
			}
		})
		stmt.Line()
	})
}

func jsonrpcClientInitSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-client-init", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("New%s instantiates HTTP clients for all the %s service servers.", data.ClientStruct, data.Service.Name))
		params := []jen.Code{
			jen.Id("scheme").String(),
			jen.Id("host").String(),
			jen.Id("doer").Add(codegen.TypeRef("loomhttp.Doer")),
			jen.Id("enc").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Encoder")),
			jen.Id("dec").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")),
			jen.Id("restoreBody").Bool(),
		}
		if httpcodegen.HasWebSocket(data) {
			params = append(params,
				jen.Id("dialer").Add(codegen.TypeRef("loomhttp.Dialer")),
				jen.Id("cfn").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc")),
				jen.Id("streamOpts").Op("...").Qual("github.com/CaliLuke/loom/jsonrpc", "StreamConfigOption"),
			)
		}
		stmt.Func().Id("New" + data.ClientStruct).
			Params(params...).
			Op("*").Id(data.ClientStruct).
			BlockFunc(func(g *jen.Group) {
				if httpcodegen.HasWebSocket(data) {
					g.Comment("Create stream configuration from options")
					g.Id("streamConfig").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "NewStreamConfig").Call(jen.Id("streamOpts").Op("..."))
					g.Line()
				}
				dict := jen.Dict{
					jen.Id("Doer"):                jen.Id("doer"),
					jen.Id("RestoreResponseBody"): jen.Id("restoreBody"),
					jen.Id("scheme"):              jen.Id("scheme"),
					jen.Id("host"):                jen.Id("host"),
					jen.Id("decoder"):             jen.Id("dec"),
					jen.Id("encoder"):             jen.Id("enc"),
				}
				for _, endpoint := range data.Endpoints {
					if !httpcodegen.IsSSEEndpoint(endpoint) {
						continue
					}
					dict[jen.Id(endpoint.Method.VarName+"Doer")] = jen.Id("doer")
				}
				if httpcodegen.HasWebSocket(data) {
					dict[jen.Id("dialer")] = jen.Id("dialer")
					dict[jen.Id("configfn")] = jen.Id("cfn")
					dict[jen.Id("streamConfig")] = jen.Id("streamConfig")
				}
				g.Return(jen.Op("&").Id(data.ClientStruct).Values(dict))
			})
		stmt.Line()
	})
}

func jsonrpcServerStructSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s handles JSON-RPC requests for the %s service.", data.ServerStruct, data.Service.Name))
		stmt.Type().Id(data.ServerStruct).StructFunc(func(g *jen.Group) {
			g.Qual("net/http", "Handler")
			if data.CORS != nil && data.CORS.Runtime {
				g.Id("corsPolicy").Add(codegen.TypeRef("loomhttp.RuntimeCORSPolicy"))
			}
			g.Comment("Methods is the list of methods served by this server.")
			g.Id("Methods").Index().String()
			if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
				g.Comment("StreamHandler is the handler for the streaming service.")
				g.Id("StreamHandler").Func().
					Params(jen.Qual("context", "Context"), codegen.TypeRef(data.Service.PkgName+".Stream")).
					Error()
			}
			for _, endpoint := range data.Endpoints {
				if httpcodegen.IsWebSocketEndpoint(endpoint) {
					g.Id(lowerInitial(endpoint.Method.VarName)).
						Func().
						Params(
							jen.Qual("context", "Context"),
							jen.Op("*").Qual("net/http", "Request"),
							jen.Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
						).
						Params(jen.Any(), jen.Error())
					if endpoint.Method.ServerStream != nil && (endpoint.Method.ServerStream.Kind == 3 || endpoint.Method.ServerStream.Kind == 4) {
						g.Id(lowerInitial(endpoint.Method.VarName) + "Endpoint").Add(codegen.TypeRef("loom.Endpoint"))
					}
					continue
				}
				g.Comment(codegen.Comment(fmt.Sprintf("%s is the handler for the %s method.", endpoint.Method.VarName, endpoint.Method.Name)))
				g.Id(endpoint.Method.VarName).
					Func().
					Params(
						jen.Qual("context", "Context"),
						jen.Op("*").Qual("net/http", "Request"),
						jen.Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
						jen.Qual("net/http", "ResponseWriter"),
					).
					Error()
			}
			g.Line()
			g.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Decoder"))
			g.Id("encoder").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter")).Add(codegen.TypeRef("loomhttp.Encoder"))
			g.Id("errhandler").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter"), jen.Error())
			if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
				g.Id("upgrader").Add(codegen.TypeRef("loomhttp.Upgrader"))
				g.Id("configfn").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc"))
			}
		})
		stmt.Line()
	})
}

//nolint:maintidx // Server constructor wiring intentionally aggregates transport-setup branches.
func jsonrpcServerInitSection(data *httpcodegen.ServiceData, hasSSE, hasMixed bool) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-init", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s creates a JSON-RPC server which loads HTTP requests and calls the %q service methods.", data.ServerInit, data.Service.Name))
		params := []jen.Code{}
		if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
			params = append(params, jen.Id("streamHandler").Func().Params(jen.Qual("context", "Context"), codegen.TypeRef(data.Service.PkgName+".Stream")).Error())
		}
		params = append(params,
			jen.Id("endpoints").Op("*").Qual(data.Service.PkgName, "Endpoints"),
			jen.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer")),
			jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Decoder")),
			jen.Id("encoder").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter")).Add(codegen.TypeRef("loomhttp.Encoder")),
			jen.Id("errhandler").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter"), jen.Error()),
		)
		if data.CORS != nil && data.CORS.Runtime {
			params = append(params, jen.Id("corsPolicy").Add(codegen.TypeRef("loomhttp.RuntimeCORSPolicy")))
		}
		if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
			params = append(params,
				jen.Id("upgrader").Add(codegen.TypeRef("loomhttp.Upgrader")),
				jen.Id("configfn").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc")),
			)
		}
		stmt.Func().Id(data.ServerInit).
			Params(params...).
			Op("*").Id(data.ServerStruct).
			BlockFunc(func(g *jen.Group) {
				dict := jen.Dict{
					jen.Id("Methods"): jen.Index().String().ValuesFunc(func(values *jen.Group) {
						for _, endpoint := range data.Endpoints {
							values.Lit(endpoint.Method.Name)
						}
					}),
					jen.Id("decoder"):    jen.Id("decoder"),
					jen.Id("encoder"):    jen.Id("encoder"),
					jen.Id("errhandler"): jen.Id("errhandler"),
				}
				if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
					dict[jen.Id("StreamHandler")] = jen.Id("streamHandler")
					dict[jen.Id("upgrader")] = jen.Id("upgrader")
					dict[jen.Id("configfn")] = jen.Id("configfn")
				}
				if data.CORS != nil && data.CORS.Runtime {
					dict[jen.Id("corsPolicy")] = jen.Id("corsPolicy")
				}
				for _, endpoint := range data.Endpoints {
					if httpcodegen.IsWebSocketEndpoint(endpoint) {
						dict[jen.Id(lowerInitial(endpoint.Method.VarName))] = jen.Id(endpoint.HandlerInit).Call(
							jen.Id("endpoints").Dot(endpoint.Method.VarName),
							jen.Id("mux"),
							jen.Id("decoder"),
						)
						if endpoint.Method.ServerStream != nil && (endpoint.Method.ServerStream.Kind == 3 || endpoint.Method.ServerStream.Kind == 4) {
							dict[jen.Id(lowerInitial(endpoint.Method.VarName)+"Endpoint")] = jen.Id("endpoints").Dot(endpoint.Method.VarName)
						}
						continue
					}
					dict[jen.Id(endpoint.Method.VarName)] = jen.Id(endpoint.HandlerInit).Call(
						jen.Id("endpoints").Dot(endpoint.Method.VarName),
						jen.Id("mux"),
						jen.Id("decoder"),
						jen.Id("encoder"),
						jen.Id("errhandler"),
					)
				}
				g.Id("s").Op(":=").Op("&").Id(data.ServerStruct).Values(dict)
				var handler *jen.Statement
				switch {
				case httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]):
					g.Comment("WebSocket services implement ServeHTTP for upgrade")
					handler = jen.Qual("net/http", "HandlerFunc").Call(jen.Id("s").Dot("ServeHTTP"))
				case hasMixed:
					g.Comment("Mixed HTTP/SSE services negotiate transports in ServeHTTP")
					handler = jen.Qual("net/http", "HandlerFunc").Call(jen.Id("s").Dot("ServeHTTP"))
				case hasSSE:
					g.Comment("SSE-only services route via handleSSE")
					handler = jen.Qual("net/http", "HandlerFunc").Call(jen.Id("s").Dot("handleSSE"))
				default:
					g.Comment("Plain HTTP JSON-RPC")
					handler = jen.Qual("net/http", "HandlerFunc").Call(jen.Id("s").Dot("ServeHTTP"))
				}
				switch {
				case data.CORS != nil && data.CORS.Runtime:
					g.Id("s").Dot("Handler").Op("=").Id("corsPolicy").Dot("Handler").Call(handler)
				case data.CORS != nil:
					g.Id("s").Dot("Handler").Op("=").Add(codegen.Expr("loomhttp.CORSHandler")).Call(
						renderJSONRPCCORSPolicy(data.CORS),
						handler,
					)
				default:
					g.Id("s").Dot("Handler").Op("=").Qual("net/http", "NewCrossOriginProtection").Call().Dot("Handler").Call(
						handler,
					)
				}
				g.Return(jen.Id("s"))
			})
		stmt.Line()
	})
}

func jsonrpcServerServiceSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-service", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s returns the name of the service served.", data.ServerService))
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id(data.ServerService).
			Params().
			String().
			Block(
				jen.Return(jen.Lit(data.Service.Name)),
			)
		stmt.Line()
	})
}

func jsonrpcServerUseSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-use", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "Use wraps the server handlers with the given middleware.")
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("Use").
			Params(jen.Id("m").Func().Params(jen.Qual("net/http", "Handler")).Qual("net/http", "Handler")).
			Block(
				jen.Id("s").Dot("Handler").Op("=").Id("m").Call(jen.Id("s").Dot("Handler")),
			)
		stmt.Line()
	})
}

func jsonrpcServerMethodNamesSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-method-names", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "MethodNames returns the methods served.")
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("MethodNames").
			Params().
			Index().String().
			Block(
				jen.Return(codegen.Expr(data.Service.PkgName + ".MethodNames[:]")),
			)
		stmt.Line()
	})
}

func jsonrpcServerResponseCaptureSection() codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-response-capture", func(stmt *jen.Statement) {
		stmt.Type().Id("jsonrpcResponseCapture").Struct(
			jen.Id("header").Qual("net/http", "Header"),
			jen.Id("body").Qual("bytes", "Buffer"),
			jen.Id("statusCode").Int(),
		)
		stmt.Line()
		stmt.Func().Params(jen.Id("c").Op("*").Id("jsonrpcResponseCapture")).
			Id("Header").
			Params().
			Qual("net/http", "Header").
			Block(
				jen.If(jen.Id("c").Dot("header").Op("==").Nil()).Block(
					jen.Id("c").Dot("header").Op("=").Make(jen.Qual("net/http", "Header")),
				),
				jen.Return(jen.Id("c").Dot("header")),
			)
		stmt.Line()
		stmt.Func().Params(jen.Id("c").Op("*").Id("jsonrpcResponseCapture")).
			Id("Write").
			Params(jen.Id("data").Index().Byte()).
			Params(jen.Int(), jen.Error()).
			Block(
				jen.If(jen.Id("c").Dot("statusCode").Op("==").Lit(0)).Block(
					jen.Id("c").Dot("statusCode").Op("=").Qual("net/http", "StatusOK"),
				),
				jen.Return(jen.Id("c").Dot("body").Dot("Write").Call(jen.Id("data"))),
			)
		stmt.Line()
		stmt.Func().Params(jen.Id("c").Op("*").Id("jsonrpcResponseCapture")).
			Id("WriteHeader").
			Params(jen.Id("statusCode").Int()).
			Block(
				jen.If(jen.Id("c").Dot("statusCode").Op("!=").Lit(0)).Block(
					jen.Return(),
				),
				jen.Id("c").Dot("statusCode").Op("=").Id("statusCode"),
			)
		stmt.Line()
		stmt.Func().Id("copyJSONRPCResponseMetadata").
			Params(
				jen.Id("dst").Qual("net/http", "ResponseWriter"),
				jen.Id("src").Op("*").Id("jsonrpcResponseCapture"),
			).
			Block(
				jen.For(
					jen.List(jen.Id("key"), jen.Id("vals")).Op(":=").Range().Id("src").Dot("Header").Call(),
				).Block(
					jen.Switch(jen.Qual("net/http", "CanonicalHeaderKey").Call(jen.Id("key"))).Block(
						jen.Case(jen.Lit("Content-Length"), jen.Lit("Content-Type"), jen.Lit("Transfer-Encoding")).Block(
							jen.Continue(),
						),
					),
					jen.For(
						jen.List(jen.Id("_"), jen.Id("val")).Op(":=").Range().Id("vals"),
					).Block(
						jen.Id("dst").Dot("Header").Call().Dot("Add").Call(jen.Id("key"), jen.Id("val")),
					),
				),
			)
		stmt.Line()
	})
}

//nolint:maintidx // Mixed HTTP/SSE negotiation is intentionally centralized here.
func jsonrpcMixedServerHandlerSection(data *httpcodegen.ServiceData) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-mixed-server-handler", func(stmt *jen.Statement) {
		stmt.Comment("ServeHTTP handles JSON-RPC requests with content negotiation for mixed HTTP/SSE transports.").Line()
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("ServeHTTP").
			Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).
			BlockFunc(func(g *jen.Group) {
				g.Switch(jen.Id("r").Dot("Method")).BlockFunc(func(sg *jen.Group) {
					sg.Case(jen.Qual("net/http", "MethodGet")).BlockFunc(func(getg *jen.Group) {
						getg.Id("req").Op(":=").Add(codegen.Expr(`&jsonrpc.RawRequest{JSONRPC: "2.0", Method: "events/stream"}`))
						getg.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(dispatch *jen.Group) {
							for _, endpoint := range data.Endpoints {
								if endpoint.SSE == nil || endpoint.Method.Name != "events/stream" {
									continue
								}
								dispatch.Case(jen.Lit(endpoint.Method.Name)).Block(
									jen.If(
										jen.Err().Op(":=").Id("s").Dot(endpoint.Method.VarName).Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Id("req"), jen.Id("w")),
										jen.Err().Op("!=").Nil(),
									).Block(
										jen.Id("s").Dot("errhandler").Call(
											jen.Id("r").Dot("Context").Call(),
											jen.Id("w"),
											jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for "+endpoint.Method.Name+": %w"), jen.Err()),
										),
									),
									jen.Return(),
								)
							}
							dispatch.Default().Block(
								jen.Qual("net/http", "NotFound").Call(jen.Id("w"), jen.Id("r")),
								jen.Return(),
							)
						})
					})
					sg.Case(jen.Qual("net/http", "MethodPost")).BlockFunc(func(postg *jen.Group) {
						postg.Id("accept").Op(":=").Id("r").Dot("Header").Dot("Get").Call(jen.Lit("Accept"))
						postg.If(
							jen.Op("!").Qual("strings", "Contains").Call(jen.Id("accept"), jen.Lit("text/event-stream")),
						).Block(
							jen.Id("s").Dot("handleHTTP").Call(jen.Id("w"), jen.Id("r")),
							jen.Return(),
						)
						postg.Line()
						postg.Id("reader").Op(":=").Qual("bufio", "NewReader").Call(jen.Id("r").Dot("Body"))
						postg.Const().Id("maxNegotiationWhitespace").Op("=").Lit(4096)
						postg.Var().Id("first").Byte()
						postg.Id("sniffed").Op(":=").Lit(0)
						postg.For(jen.Id("sniffed").Op("<").Id("maxNegotiationWhitespace")).Block(
							jen.List(jen.Id("peek"), jen.Err()).Op(":=").Id("reader").Dot("Peek").Call(jen.Lit(1)),
							jen.If(
								jen.Err().Op("!=").Nil().Op("&&").Err().Op("!=").Qual("io", "EOF"),
							).Block(
								jen.Id("s").Dot("errhandler").Call(
									jen.Id("r").Dot("Context").Call(),
									jen.Id("w"),
									jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to read request body: %w"), jen.Err()),
								),
								jen.Return(),
							),
							jen.If(jen.Len(jen.Id("peek")).Op("==").Lit(0)).Block(
								jen.Break(),
							),
							jen.Id("first").Op("=").Id("peek").Index(jen.Lit(0)),
							jen.If(
								jen.Id("first").Op("!=").LitByte(' ').
									Op("&&").Id("first").Op("!=").LitByte('\t').
									Op("&&").Id("first").Op("!=").LitByte('\r').
									Op("&&").Id("first").Op("!=").LitByte('\n'),
							).Block(
								jen.Break(),
							),
							jen.If(
								jen.List(jen.Id("_"), jen.Err()).Op(":=").Id("reader").Dot("Discard").Call(jen.Lit(1)),
								jen.Err().Op("!=").Nil(),
							).Block(
								jen.Id("s").Dot("errhandler").Call(
									jen.Id("r").Dot("Context").Call(),
									jen.Id("w"),
									jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to read request body: %w"), jen.Err()),
								),
								jen.Return(),
							),
							jen.Id("sniffed").Op("++"),
						)
						postg.Line()
						postg.Id("r").Dot("Body").Op("=").Struct(
							jen.Qual("io", "Reader"),
							jen.Qual("io", "Closer"),
						).Values(jen.Dict{
							jen.Id("Reader"): jen.Id("reader"),
							jen.Id("Closer"): jen.Id("r").Dot("Body"),
						})
						postg.Line()
						postg.If(
							jen.Id("first").Op("==").Lit(0).Op("||").Id("first").Op("==").LitByte('[').Op("||").Id("sniffed").Op(">=").Id("maxNegotiationWhitespace"),
						).Block(
							jen.Id("s").Dot("handleHTTP").Call(jen.Id("w"), jen.Id("r")),
							jen.Return(),
						)
						postg.Line()
						postg.Var().Id("req").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest")
						postg.If(
							jen.Err().Op(":=").Id("s").Dot("decoder").Call(jen.Id("r")).Dot("Decode").Call(jen.Op("&").Id("req")),
							jen.Err().Op("!=").Nil(),
						).Block(
							jen.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(
								jen.Nil(),
								jen.Qual("github.com/CaliLuke/loom/jsonrpc", "ParseError"),
								jen.Lit("Parse error"),
								jen.Nil(),
							),
							jen.If(
								jen.Id("encErr").Op(":=").Id("s").Dot("encoder").Call(jen.Id("r").Dot("Context").Call(), jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
								jen.Id("encErr").Op("!=").Nil(),
							).Block(
								jen.Id("s").Dot("errhandler").Call(
									jen.Id("r").Dot("Context").Call(),
									jen.Id("w"),
									jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode parse error response: %w"), jen.Id("encErr")),
								),
							),
							jen.Return(),
						)
						postg.Line()
						postg.If(jen.Id("req").Dot("Invalid")).Block(
							jen.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
							jen.Return(),
						)
						postg.Line()
						postg.Switch(jen.Id("req").Dot("Method")).BlockFunc(func(dispatch *jen.Group) {
							for _, endpoint := range data.Endpoints {
								if endpoint.SSE == nil {
									continue
								}
								dispatch.Case(jen.Lit(endpoint.Method.Name)).Block(
									jen.If(
										jen.Err().Op(":=").Id("s").Dot(endpoint.Method.VarName).Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
										jen.Err().Op("!=").Nil(),
									).Block(
										jen.Id("s").Dot("errhandler").Call(
											jen.Id("r").Dot("Context").Call(),
											jen.Id("w"),
											jen.Qual("fmt", "Errorf").Call(jen.Lit("handler error for "+endpoint.Method.Name+": %w"), jen.Err()),
										),
									),
								)
							}
							dispatch.Default().Block(
								jen.Id("s").Dot("processRequest").Call(jen.Id("r").Dot("Context").Call(), jen.Id("r"), jen.Op("&").Id("req"), jen.Id("w")),
							)
						})
					})
					sg.Default().Block(
						jen.Qual("net/http", "NotFound").Call(jen.Id("w"), jen.Id("r")),
					)
				})
			})
		stmt.Line()
	})
}

func jsonrpcServerMountSection(data *httpcodegen.ServiceData, hasSSE, hasMixed bool) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-mount", func(stmt *jen.Statement) {
		comment := fmt.Sprintf("%s configures the mux to serve the JSON-RPC %s service methods.", data.MountServer, data.Service.Name)
		codegen.Doc(stmt, comment)
		stmt.Func().Id(data.MountServer).
			Params(
				jen.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer")),
				jen.Id("h").Op("*").Id(data.ServerStruct),
			).
			BlockFunc(func(g *jen.Group) {
				switch {
				case hasMixed:
					g.Comment("Mixed transports: mount unified handler that negotiates HTTP vs SSE by Accept header and JSON-RPC method")
					seen := map[string]struct{}{}
					for _, route := range data.Endpoints[0].Routes {
						for _, verb := range []string{route.Verb, "GET"} {
							key := verb + " " + route.Path
							if _, ok := seen[key]; ok {
								continue
							}
							seen[key] = struct{}{}
							g.Id("mux").Dot("Handle").Call(jen.Lit(verb), jen.Lit(route.Path), jen.Id("h").Dot("Handler").Dot("ServeHTTP"))
						}
					}
				case hasSSE:
					g.Comment("SSE only: mount SSE handler")
					seen := map[string]struct{}{}
					for _, endpoint := range data.Endpoints {
						for _, route := range endpoint.Routes {
							verbs := []string{route.Verb}
							if endpoint.Method.Name == "events/stream" {
								verbs = append(verbs, "GET")
							}
							for _, verb := range verbs {
								key := verb + " " + route.Path
								if _, ok := seen[key]; ok {
									continue
								}
								seen[key] = struct{}{}
								g.Id("mux").Dot("Handle").Call(jen.Lit(verb), jen.Lit(route.Path), jen.Id("h").Dot("Handler").Dot("ServeHTTP"))
							}
						}
					}
				default:
					g.Comment("HTTP only")
					for _, route := range data.Endpoints[0].Routes {
						g.Id("mux").Dot("Handle").Call(jen.Lit(route.Verb), jen.Lit(route.Path), jen.Id("h").Dot("Handler").Dot("ServeHTTP"))
					}
				}
				writeJSONRPCCORSMounts(g, data, hasSSE, hasMixed)
			})
		stmt.Line()
		codegen.Doc(stmt, comment)
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id(data.MountServer).
			Params(jen.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer"))).
			Block(
				jen.Id(data.MountServer).Call(jen.Id("mux"), jen.Id("s")),
			)
		stmt.Line()
	})
}

func jsonrpcServerEncodeErrorSection(serverStruct string) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-encode-error", func(stmt *jen.Statement) {
		writeJSONRPCEncodeErrorMethod(stmt, serverStruct)
		writeJSONRPCEncodeErrorFunction(stmt)
		writeJSONRPCServiceErrorClassifier(stmt)
	})
}

func writeJSONRPCEncodeErrorMethod(stmt *jen.Statement, serverStruct string) {
	stmt.Comment("encodeJSONRPCError creates and sends a JSON-RPC error response (handles nil ID gracefully)").Line()
	stmt.Func().Params(jen.Id("s").Op("*").Id(serverStruct)).
		Id("encodeJSONRPCError").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("req").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
			jen.Id("code").Qual("github.com/CaliLuke/loom/jsonrpc", "Code"),
			jen.Id("message").String(),
			jen.Id("data").Any(),
		).
		Block(
			jen.Id("encodeJSONRPCError").Call(
				jen.Id("ctx"),
				jen.Id("w"),
				jen.Id("req"),
				jen.Id("code"),
				jen.Id("message"),
				jen.Id("data"),
				jen.Id("s").Dot("encoder"),
				jen.Id("s").Dot("errhandler"),
			),
		)
	stmt.Line()
}

func writeJSONRPCEncodeErrorFunction(stmt *jen.Statement) {
	stmt.Comment("encodeJSONRPCError creates and sends a JSON-RPC error response (handles nil ID gracefully)").Line()
	stmt.Func().Id("encodeJSONRPCError").
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("w").Qual("net/http", "ResponseWriter"),
			jen.Id("req").Op("*").Qual("github.com/CaliLuke/loom/jsonrpc", "RawRequest"),
			jen.Id("code").Qual("github.com/CaliLuke/loom/jsonrpc", "Code"),
			jen.Id("message").String(),
			jen.Id("data").Any(),
			jen.Id("encoder").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter")).Add(codegen.TypeRef("loomhttp.Encoder")),
			jen.Id("errhandler").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter"), jen.Error()),
		).
		Block(
			jen.If(jen.Id("req").Dot("HasID").Op("||").Id("code").Op("==").Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidRequest")).Block(
				jen.Id("id").Op(":=").Id("req").Dot("ID"),
				jen.If(jen.Op("!").Id("req").Dot("HasID")).Block(
					jen.Id("id").Op("=").Nil(),
				),
				jen.Id("response").Op(":=").Qual("github.com/CaliLuke/loom/jsonrpc", "MakeErrorResponse").Call(
					jen.Id("id"),
					jen.Id("code"),
					jen.Id("message"),
					jen.Id("data"),
				),
				jen.If(
					jen.Err().Op(":=").Id("encoder").Call(jen.Id("ctx"), jen.Id("w")).Dot("Encode").Call(jen.Id("response")),
					jen.Err().Op("!=").Nil(),
				).Block(
					jen.Id("errhandler").Call(
						jen.Id("ctx"),
						jen.Id("w"),
						jen.Qual("fmt", "Errorf").Call(jen.Lit("failed to encode JSON-RPC response: %w"), jen.Err()),
					),
				),
			),
		)
	stmt.Line()
}

func writeJSONRPCServiceErrorClassifier(stmt *jen.Statement) {
	stmt.Comment("jsonrpcErrorCodeForServiceError classifies framework validation errors as invalid params and all other service errors as internal errors.").Line()
	stmt.Func().Id("jsonrpcErrorCodeForServiceError").
		Params(jen.Id("err").Op("*").Add(codegen.TypeRef("loom.ServiceError"))).
		Qual("github.com/CaliLuke/loom/jsonrpc", "Code").
		Block(
			jen.If(jen.Id("err").Op("==").Nil()).Block(
				jen.Return(jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")),
			),
			jen.Switch(jen.Id("err").Dot("Name")).Block(
				jen.Case(
					codegen.Expr("loom.InvalidFieldType"),
					codegen.Expr("loom.MissingField"),
					codegen.Expr("loom.InvalidEnumValue"),
					codegen.Expr("loom.InvalidFormat"),
					codegen.Expr("loom.InvalidPattern"),
					codegen.Expr("loom.InvalidRange"),
					codegen.Expr("loom.InvalidLength"),
					codegen.Expr("loom.DecodePayload"),
					codegen.Expr("loom.MissingPayload"),
				).Block(
					jen.Return(jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InvalidParams")),
				),
				jen.Default().Block(
					jen.Return(jen.Qual("github.com/CaliLuke/loom/jsonrpc", "InternalError")),
				),
			),
		)
	stmt.Line()
}
