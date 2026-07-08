package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	gocodegen "github.com/CaliLuke/loom/codegen"
)

func clientStructSection(data *ServiceData) gocodegen.Section {
	return gocodegen.NewJenniferSection("client-struct", func(stmt *jen.Statement) {
		gocodegen.Doc(stmt, fmt.Sprintf("%s lists the %s service endpoint HTTP clients.", data.ClientStruct, data.Service.Name))
		stmt.Type().Id(data.ClientStruct).StructFunc(func(group *jen.Group) {
			for _, endpoint := range data.Endpoints {
				group.Comment(gocodegen.Comment(fmt.Sprintf("%s Doer is the HTTP client used to make requests to the %s endpoint.", endpoint.Method.VarName, endpoint.Method.Name)))
				group.Id(endpoint.Method.VarName + "Doer").Id("loomhttp").Dot("Doer")
			}

			group.Comment(gocodegen.Comment("RestoreResponseBody controls whether the response bodies are reset after\ndecoding so they can be read again."))
			group.Id("RestoreResponseBody").Bool()
			group.Line()

			group.Id("scheme").String()
			group.Id("host").String()
			group.Id("encoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Id("loomhttp").Dot("Encoder")
			group.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Id("loomhttp").Dot("Decoder")
			if HasWebSocket(data) {
				group.Id("dialer").Id("loomhttp").Dot("Dialer")
				group.Id("configurer").Op("*").Id("ConnConfigurer")
			}
		})
	})
}

func clientInitSection(data *ServiceData) gocodegen.Section {
	return gocodegen.NewJenniferSection("http-client-init", func(stmt *jen.Statement) {
		gocodegen.Doc(stmt, fmt.Sprintf("New%s instantiates HTTP clients for all the %s service servers.", data.ClientStruct, data.Service.Name))

		fn := stmt.Func().Id("New" + data.ClientStruct).ParamsFunc(func(args *jen.Group) {
			args.Id("scheme").String()
			args.Id("host").String()
			args.Id("doer").Id("loomhttp").Dot("Doer")
			args.Id("enc").Func().Params(jen.Op("*").Qual("net/http", "Request")).Id("loomhttp").Dot("Encoder")
			args.Id("dec").Func().Params(jen.Op("*").Qual("net/http", "Response")).Id("loomhttp").Dot("Decoder")
			args.Id("restoreBody").Bool()
			if HasWebSocket(data) {
				args.Id("dialer").Id("loomhttp").Dot("Dialer")
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

func clientEndpointSections(endpoint *EndpointData) []gocodegen.Section {
	if endpoint.HasMixedResults {
		standard := *endpoint
		standard.SSE = nil

		sseEndpoint := *endpoint
		sseEndpoint.EndpointInit = endpoint.EndpointInit + "Stream"

		return []gocodegen.Section{
			clientEndpointSection(&standard),
			clientEndpointSection(&sseEndpoint),
		}
	}

	return []gocodegen.Section{clientEndpointSection(endpoint)}
}

func clientEndpointSection(endpoint *EndpointData) gocodegen.Section {
	return gocodegen.NewJenniferSection("client-endpoint-init", func(stmt *jen.Statement) {
		gocodegen.Doc(stmt, fmt.Sprintf("%s returns an endpoint that makes HTTP requests to the %s service %s server.", endpoint.EndpointInit, endpoint.ServiceName, endpoint.Method.Name))

		fn := stmt.Func().Params(jen.Id("c").Op("*").Id(endpoint.ClientStruct)).Id(endpoint.EndpointInit)
		if endpoint.MultipartRequestEncoder != nil {
			fn.Params(jen.Id(endpoint.MultipartRequestEncoder.VarName).Id(endpoint.MultipartRequestEncoder.FuncName))
		} else {
			fn.Params()
		}
		fn.Id("loom").Dot("Endpoint").BlockFunc(func(group *jen.Group) {
			writeClientEndpointDefinitions(group, endpoint)
			group.Return().Func().
				Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Any()).
				Params(jen.Any(), jen.Error()).
				BlockFunc(func(body *jen.Group) { writeClientEndpointBody(body, endpoint) })
		})
	})
}

func writeClientEndpointDefinitions(group *jen.Group, endpoint *EndpointData) {
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
	if !IsSSEEndpoint(endpoint) || len(endpoint.Errors) > 0 {
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
}

func writeClientEndpointBody(body *jen.Group, endpoint *EndpointData) {
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
			jen.Id("loomhttp").Dot("ErrRequestError").Call(
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
		group.Id("done").Op(":=").Make(jen.Chan().Struct())
		addRawWebSocketGroup(group, `go func() {
	select {
	case <-ctx.Done():
		if err := conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client closing connection"),
			time.Now().Add(time.Second),
		); err != nil {
			return
		}
		if err := conn.Close(); err != nil {
			return
		}
	case <-done:
	}
}()`)
	}

	group.Id("stream").Op(":=").Op("&").Id(endpoint.ClientWebSocket.VarName).ValuesFunc(func(values *jen.Group) {
		values.Id("conn").Op(":").Id("conn")
		if endpoint.ClientWebSocket.SendName == "" {
			values.Id("done").Op(":").Id("done")
		}
	})
	if endpoint.Method.ViewedResult != nil && endpoint.Method.ViewedResult.ViewName == "" {
		group.Id("view").Op(":=").Id("resp").Dot("Header").Dot("Get").Call(jen.Lit("loom-view"))
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
			jen.Id("loomhttp").Dot("ErrRequestError").Call(
				jen.Lit(endpoint.ServiceName),
				jen.Lit(endpoint.Method.Name),
				jen.Err(),
			),
		),
	)

	group.If(jen.Id("resp").Dot("StatusCode").Op("!=").Id("http").Dot("StatusOK")).BlockFunc(func(status *jen.Group) {
		if len(endpoint.Errors) > 0 {
			status.Return(jen.Id("decodeResponse").Call(jen.Id("resp")))
			return
		}
		status.Id("resp").Dot("Body").Dot("Close").Call()
		status.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
			jen.Lit("unexpected status from SSE endpoint: %d"),
			jen.Id("resp").Dot("StatusCode"),
		))
	})

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
			jen.Id("loomhttp").Dot("ErrRequestError").Call(
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
