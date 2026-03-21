package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	gocodegen "goa.design/goa/v3/codegen"
)

func clientStructSection(data *ServiceData) *gocodegen.SectionTemplate {
	return gocodegen.MustJenniferSection("client-struct", func(stmt *jen.Statement) {
		stmt.Comment(gocodegen.Comment(fmt.Sprintf("%s lists the %s service endpoint HTTP clients.", data.ClientStruct, data.Service.Name))).Line()
		stmt.Type().Id(data.ClientStruct).StructFunc(func(group *jen.Group) {
			for _, endpoint := range data.Endpoints {
				group.Comment(gocodegen.Comment(fmt.Sprintf("%s Doer is the HTTP client used to make requests to the %s endpoint.", endpoint.Method.VarName, endpoint.Method.Name)))
				group.Id(endpoint.Method.VarName + "Doer").Id("goahttp").Dot("Doer")
			}

			group.Comment(gocodegen.Comment("RestoreResponseBody controls whether the response bodies are reset after\ndecoding so they can be read again."))
			group.Id("RestoreResponseBody").Bool()
			group.Line()

			group.Id("scheme").String()
			group.Id("host").String()
			group.Id("encoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Id("goahttp").Dot("Encoder")
			group.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Id("goahttp").Dot("Decoder")
			if HasWebSocket(data) {
				group.Id("dialer").Id("goahttp").Dot("Dialer")
				group.Id("configurer").Op("*").Id("ConnConfigurer")
			}
		})
	})
}

func clientInitSection(data *ServiceData) *gocodegen.SectionTemplate {
	return gocodegen.MustJenniferSection("http-client-init", func(stmt *jen.Statement) {
		stmt.Comment(gocodegen.Comment(fmt.Sprintf("New%s instantiates HTTP clients for all the %s service servers.", data.ClientStruct, data.Service.Name))).Line()

		fn := stmt.Func().Id("New" + data.ClientStruct).ParamsFunc(func(args *jen.Group) {
			args.Id("scheme").String()
			args.Id("host").String()
			args.Id("doer").Id("goahttp").Dot("Doer")
			args.Id("enc").Func().Params(jen.Op("*").Qual("net/http", "Request")).Id("goahttp").Dot("Encoder")
			args.Id("dec").Func().Params(jen.Op("*").Qual("net/http", "Response")).Id("goahttp").Dot("Decoder")
			args.Id("restoreBody").Bool()
			if HasWebSocket(data) {
				args.Id("dialer").Id("goahttp").Dot("Dialer")
				args.Id("cfn").Op("*").Id("ConnConfigurer")
			}
		}).Op("*").Id(data.ClientStruct)

		fn.BlockFunc(func(group *jen.Group) {
			if HasWebSocket(data) {
				group.If(jen.Id("cfn").Op("==").Nil()).Block(
					jen.Id("cfn").Op("=").Op("&").Id("ConnConfigurer").Values(),
				)
			}

			group.Return().Op("&").Id(data.ClientStruct).ValuesFunc(func(values *jen.Group) {
				for _, endpoint := range data.Endpoints {
					values.Id(endpoint.Method.VarName + "Doer").Op(":").Id("doer")
				}
				values.Id("RestoreResponseBody").Op(":").Id("restoreBody")
				values.Id("scheme").Op(":").Id("scheme")
				values.Id("host").Op(":").Id("host")
				values.Id("decoder").Op(":").Id("dec")
				values.Id("encoder").Op(":").Id("enc")
				if HasWebSocket(data) {
					values.Id("dialer").Op(":").Id("dialer")
					values.Id("configurer").Op(":").Id("cfn")
				}
			})
		})
	})
}

func clientEndpointSections(endpoint *EndpointData) []*gocodegen.SectionTemplate {
	if endpoint.HasMixedResults {
		standard := *endpoint
		standard.SSE = nil

		sseEndpoint := *endpoint
		sseEndpoint.EndpointInit = endpoint.EndpointInit + "Stream"

		return []*gocodegen.SectionTemplate{
			clientEndpointSection(&standard),
			clientEndpointSection(&sseEndpoint),
		}
	}

	return []*gocodegen.SectionTemplate{clientEndpointSection(endpoint)}
}

func clientEndpointSection(endpoint *EndpointData) *gocodegen.SectionTemplate {
	return gocodegen.MustJenniferSection("client-endpoint-init", func(stmt *jen.Statement) {
		stmt.Comment(gocodegen.Comment(fmt.Sprintf("%s returns an endpoint that makes HTTP requests to the %s service %s server.", endpoint.EndpointInit, endpoint.ServiceName, endpoint.Method.Name))).Line()

		fn := stmt.Func().Params(jen.Id("c").Op("*").Id(endpoint.ClientStruct)).Id(endpoint.EndpointInit)
		if endpoint.MultipartRequestEncoder != nil {
			fn.Params(jen.Id(endpoint.MultipartRequestEncoder.VarName).Id(endpoint.MultipartRequestEncoder.FuncName))
		} else {
			fn.Params()
		}
		fn.Id("goa").Dot("Endpoint").BlockFunc(func(group *jen.Group) {
			var defs []jen.Code
			if endpoint.RequestEncoder != "" {
				var requestEncoderArg jen.Code = jen.Id("c").Dot("encoder")
				if endpoint.MultipartRequestEncoder != nil {
					requestEncoderArg = jen.Id(endpoint.MultipartRequestEncoder.InitName).Call(jen.Id(endpoint.MultipartRequestEncoder.VarName))
				}
				defs = append(defs,
					jen.Id("encodeRequest").Op("=").Id(endpoint.RequestEncoder).Call(requestEncoderArg),
				)
			}
			if !IsSSEEndpoint(endpoint) {
				defs = append(defs,
					jen.Id("decodeResponse").Op("=").Id(endpoint.ResponseDecoder).Call(
						jen.Id("c").Dot("decoder"),
						jen.Id("c").Dot("RestoreResponseBody"),
					),
				)
			}
			if len(defs) > 0 {
				group.Var().Defs(defs...)
			}

			group.Return().Func().
				Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Any()).
				Params(jen.Any(), jen.Error()).
				BlockFunc(func(body *jen.Group) {
					body.List(jen.Id("req"), jen.Err()).Op(":=").Id("c").Dot(endpoint.RequestInit.Name).CallFunc(func(args *jen.Group) {
						args.Id("ctx")
						for _, arg := range endpoint.RequestInit.ClientArgs {
							args.Id(arg.Ref)
						}
					})
					body.If(jen.Err().Op("!=").Nil()).Block(
						jen.Return(jen.Nil(), jen.Err()),
					)

					if endpoint.RequestEncoder != "" {
						body.Err().Op("=").Id("encodeRequest").Call(jen.Id("req"), jen.Id("v"))
						body.If(jen.Err().Op("!=").Nil()).Block(
							jen.Return(jen.Nil(), jen.Err()),
						)
					}

					switch {
					case IsWebSocketEndpoint(endpoint):
						renderClientWebSocketEndpoint(body, endpoint)
					case IsSSEEndpoint(endpoint):
						renderClientSSEEndpoint(body, endpoint)
					default:
						renderClientHTTPEndpoint(body, endpoint)
					}
				})
		})
	})
}

func renderClientWebSocketEndpoint(group *jen.Group, endpoint *EndpointData) {
	group.List(jen.Id("conn"), jen.Id("resp"), jen.Err()).Op(":=").Id("c").Dot("dialer").Dot("DialContext").Call(
		jen.Id("ctx"),
		jen.Id("req").Dot("URL").Dot("String").Call(),
		jen.Id("req").Dot("Header"),
	)
	group.If(jen.Err().Op("!=").Nil()).BlockFunc(func(failure *jen.Group) {
		failure.If(jen.Id("resp").Op("!=").Nil()).Block(
			jen.Return(jen.Id("decodeResponse").Call(jen.Id("resp"))),
		)
		failure.Return(
			jen.Nil(),
			jen.Id("goahttp").Dot("ErrRequestError").Call(
				jen.Lit(endpoint.ServiceName),
				jen.Lit(endpoint.Method.Name),
				jen.Err(),
			),
		)
	})

	group.If(jen.Id("c").Dot("configurer").Dot(endpoint.Method.VarName + "Fn").Op("!=").Nil()).BlockFunc(func(configure *jen.Group) {
		if endpoint.ClientWebSocket.SendName == "" {
			configure.Var().Id("cancel").Qual("context", "CancelFunc")
			configure.List(jen.Id("ctx"), jen.Id("cancel")).Op("=").Qual("context", "WithCancel").Call(jen.Id("ctx"))
			configure.Id("conn").Op("=").Id("c").Dot("configurer").Dot(endpoint.Method.VarName+"Fn").Call(jen.Id("conn"), jen.Id("cancel"))
		} else {
			configure.Id("conn").Op("=").Id("c").Dot("configurer").Dot(endpoint.Method.VarName+"Fn").Call(jen.Id("conn"), jen.Nil())
		}
	})

	if endpoint.ClientWebSocket.SendName == "" {
		group.Go().Func().Params().Block(
			jen.Op("<-").Id("ctx").Dot("Done").Call(),
			jen.Id("conn").Dot("WriteControl").Call(
				jen.Id("websocket").Dot("CloseMessage"),
				jen.Id("websocket").Dot("FormatCloseMessage").Call(
					jen.Id("websocket").Dot("CloseNormalClosure"),
					jen.Lit("client closing connection"),
				),
				jen.Id("time").Dot("Now").Call().Dot("Add").Call(jen.Id("time").Dot("Second")),
			),
			jen.Id("conn").Dot("Close").Call(),
		).Call()
	}

	group.Id("stream").Op(":=").Op("&").Id(endpoint.ClientWebSocket.VarName).Values(jen.Id("conn").Op(":").Id("conn"))
	if endpoint.Method.ViewedResult != nil && endpoint.Method.ViewedResult.ViewName == "" {
		group.Id("view").Op(":=").Id("resp").Dot("Header").Dot("Get").Call(jen.Lit("goa-view"))
		group.Id("stream").Dot("SetView").Call(jen.Id("view"))
	}
	group.Return(jen.Id("stream"), jen.Nil())
}

func renderClientSSEEndpoint(group *jen.Group, endpoint *EndpointData) {
	group.Comment("For SSE endpoints, connect and return a stream")
	if endpoint.HasMixedResults {
		group.Comment("Set Accept header for content negotiation")
		group.Id("req").Dot("Header").Dot("Set").Call(jen.Lit("Accept"), jen.Lit("text/event-stream"))
	}

	group.List(jen.Id("resp"), jen.Err()).Op(":=").Id("c").Dot(endpoint.Method.VarName + "Doer").Dot("Do").Call(jen.Id("req"))
	group.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(
			jen.Nil(),
			jen.Id("goahttp").Dot("ErrRequestError").Call(
				jen.Lit(endpoint.ServiceName),
				jen.Lit(endpoint.Method.Name),
				jen.Err(),
			),
		),
	)

	group.If(jen.Id("resp").Dot("StatusCode").Op("!=").Id("http").Dot("StatusOK")).Block(
		jen.Id("resp").Dot("Body").Dot("Close").Call(),
		jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
			jen.Lit("unexpected status from SSE endpoint: %d"),
			jen.Id("resp").Dot("StatusCode"),
		)),
	)

	group.Id("contentType").Op(":=").Id("resp").Dot("Header").Dot("Get").Call(jen.Lit("Content-Type"))
	group.If(
		jen.Id("contentType").Op("!=").Lit("").
			Op("&&").
			Op("!").Qual("strings", "HasPrefix").Call(jen.Id("contentType"), jen.Lit("text/event-stream")),
	).Block(
		jen.Id("resp").Dot("Body").Dot("Close").Call(),
		jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
			jen.Lit("unexpected content type: %s (expected text/event-stream)"),
			jen.Id("contentType"),
		)),
	)

	group.Return(
		jen.Id("New"+endpoint.Method.VarName+"Stream").Call(jen.Id("resp"), jen.Id("c").Dot("decoder")),
		jen.Nil(),
	)
}

func renderClientHTTPEndpoint(group *jen.Group, endpoint *EndpointData) {
	group.List(jen.Id("resp"), jen.Err()).Op(":=").Id("c").Dot(endpoint.Method.VarName + "Doer").Dot("Do").Call(jen.Id("req"))
	group.If(jen.Err().Op("!=").Nil()).Block(
		jen.Return(
			jen.Nil(),
			jen.Id("goahttp").Dot("ErrRequestError").Call(
				jen.Lit(endpoint.ServiceName),
				jen.Lit(endpoint.Method.Name),
				jen.Err(),
			),
		),
	)

	if endpoint.Method.SkipResponseBodyEncodeDecode {
		if endpoint.Result.Ref != "" {
			group.List(jen.Id("res"), jen.Err()).Op(":=").Id("decodeResponse").Call(jen.Id("resp"))
		} else {
			group.List(jen.Id("_"), jen.Err()).Op("=").Id("decodeResponse").Call(jen.Id("resp"))
		}
		group.If(jen.Err().Op("!=").Nil()).Block(
			jen.Id("resp").Dot("Body").Dot("Close").Call(),
			jen.Return(jen.Nil(), jen.Err()),
		)

		group.Return(
			jen.Op("&").Id(responseStructPkg(endpoint.Method, endpoint.ServicePkgName)).Dot(endpoint.Method.ResponseStruct).ValuesFunc(func(values *jen.Group) {
				if endpoint.Result.Ref != "" {
					values.Id("Result").Op(":").Id("res").Assert(jen.Id(endpoint.Result.Ref))
				}
				values.Id("Body").Op(":").Id("resp").Dot("Body")
			}),
			jen.Nil(),
		)
		return
	}

	group.Return(jen.Id("decodeResponse").Call(jen.Id("resp")))
}
