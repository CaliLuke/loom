package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	gocodegen "github.com/CaliLuke/loom/codegen"
)

func clientStructSection(data *ServiceData) gocodegen.Section {
	return gocodegen.NewJenniferSection("client-struct", func(stmt *jen.Statement) {
		gocodegen.Doc(stmt, fmt.Sprintf("%s lists the %s service endpoint HTTP clients.", data.ClientStruct, data.Service.Name))
		stmt.Type().Id(data.ClientStruct).StructFunc(func(group *jen.Group) {
			for _, endpoint := range data.Endpoints {
				group.Comment(gocodegen.Comment(fmt.Sprintf("%s Doer is the HTTP client used to make requests to the %s endpoint.", endpoint.Method.VarName, endpoint.Method.Name)))
				group.Id(endpoint.ClientDoer).Id("loomhttp").Dot("Doer")
			}
			for _, opGroup := range data.clientGroups {
				group.Comment(gocodegen.Comment(fmt.Sprintf("%s groups %s operations.", opGroup.FieldName, data.Service.Name)))
				group.Id(opGroup.FieldName).Op("*").Id(opGroup.Name)
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

			if len(data.clientGroups) == 0 {
				group.Return().Op("&").Id(data.ClientStruct).ValuesFunc(func(values *jen.Group) {
					addClientInitValues(values, data)
				})
				return
			}
			group.Id("client").Op(":=").Op("&").Id(data.ClientStruct).ValuesFunc(func(values *jen.Group) {
				addClientInitValues(values, data)
			})
			for _, opGroup := range data.clientGroups {
				group.Id("client").Dot(opGroup.FieldName).Op("=").Op("&").Id(opGroup.Name).Values(jen.Id("client").Op(":").Id("client"))
			}
			group.Return(jen.Id("client"))
		})
	})
}

func addClientInitValues(values *jen.Group, data *ServiceData) {
	for _, endpoint := range data.Endpoints {
		values.Id(endpoint.ClientDoer).Op(":").Id("doer")
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
}

func clientEndpointSections(endpoint *EndpointData) []gocodegen.Section {
	if endpoint.HasMixedResults {
		standard := *endpoint
		standard.SSE = nil

		sseEndpoint := *endpoint
		sseEndpoint.EndpointInit = endpoint.StreamEndpointInit

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
	if endpoint.ResponseDecoder != "" {
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

	group.List(jen.Id("ctx"), jen.Id("cancel")).Op(":=").Qual("context", "WithCancel").Call(jen.Id("ctx"))
	group.Id("wsconn").Op(":=").Id("loomhttp").Dot("NewWebSocketStream").Call(jen.Id("c").Dot("configurer").Dot(endpoint.Method.VarName + "Config"))
	group.If(jen.Err().Op(":=").Id("wsconn").Dot("SetConn").Call(jen.Id("ctx"), jen.Id("conn"), jen.Id("cancel")), jen.Err().Op("!=").Nil()).Block(
		jen.Id("cancel").Call(),
		jen.Return(jen.Nil(), jen.Id("loomhttp").Dot("ErrRequestError").Call(jen.Lit(endpoint.ServiceName), jen.Lit(endpoint.Method.Name), jen.Err())),
	)

	group.Id("stream").Op(":=").Op("&").Id(endpoint.ClientWebSocket.VarName).ValuesFunc(func(values *jen.Group) {
		values.Id("conn").Op(":").Id("wsconn")
	})
	if endpoint.Method.ViewedResult != nil && endpoint.Method.ViewedResult.ViewName == "" {
		group.Id("view").Op(":=").Id("resp").Dot("Header").Dot("Get").Call(jen.Lit("loom-view"))
		group.Id("stream").Dot("SetView").Call(jen.Id("view"))
	}
	group.Return(jen.Id("stream"), jen.Nil())
}

func renderClientSSEEndpoint(group *jen.Group, endpoint *EndpointData) {
	if endpoint.HasMixedResults {
		group.Id("req").Dot("Header").Dot("Set").Call(jen.Lit("Accept"), jen.Lit("text/event-stream"))
	}
	renderClientDo(group, endpoint)
	var b strings.Builder
	fmt.Fprintf(&b, "selected, found := loomhttp.SelectFinal([]loomhttp.StatusSelector{%s}, resp.StatusCode)\n", finalSelectorValues(endpoint))
	b.WriteString("success := false\nif found {\nswitch selected {\n")
	for _, response := range endpoint.Result.Responses {
		fmt.Fprintf(&b, "case %#v:\nsuccess = true\n", responseContractSelectorValue(response.Selector))
	}
	b.WriteString("}\n}\nif !success {\nreturn decodeResponse(resp)\n}\n")
	b.WriteString("return loomhttp.DecodeResponse(resp, false, true, func(resp *http.Response) (any, error) {\nmediaType, _, err := mime.ParseMediaType(resp.Header.Get(\"Content-Type\"))\nif err != nil || mediaType != \"text/event-stream\" {\nreturn nil, fmt.Errorf(\"unexpected content type %q (expected text/event-stream)\", resp.Header.Get(\"Content-Type\"))\n}\n")
	argument := ""
	if endpoint.SSE.HandshakeRef != "" {
		fmt.Fprintf(&b, "metadata, err := %s(selected, resp)\nif err != nil {\nreturn nil, err\n}\n", endpoint.SSE.DecodeHandshake)
		argument = ", metadata"
	}
	fmt.Fprintf(&b, "return New%sStream(ctx, resp, c.decoder%s), nil\n})", endpoint.Method.VarName, argument)
	group.Add(gocodegen.Expr(b.String()))
}

func renderClientDo(group *jen.Group, endpoint *EndpointData) {
	if len(endpoint.Informational) > 0 {
		group.List(jen.Id("req"), jen.Id("finishInformational")).Op(":=").Id("loomhttp").Dot("ObserveInformational").Call(
			jen.Id("req"), jen.Id(endpoint.InformationalObservers).Call(jen.Id("ctx")),
		)
	}
	group.List(jen.Id("resp"), jen.Err()).Op(":=").Id("c").Dot(endpoint.ClientDoer).Dot("Do").Call(jen.Id("req"))
	if len(endpoint.Informational) > 0 {
		group.List(jen.Id("resp"), jen.Err()).Op("=").Id("finishInformational").Call(jen.Id("resp"), jen.Err())
	}
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
}

func renderClientHTTPEndpoint(group *jen.Group, endpoint *EndpointData) {
	renderClientDo(group, endpoint)

	if endpoint.Method.SkipResponseBodyEncodeDecode || endpoint.Method.FileResponse {
		if endpoint.Result.Ref != "" {
			group.List(jen.Id("res"), jen.Err()).Op(":=").Id("decodeResponse").Call(jen.Id("resp"))
		} else {
			group.List(jen.Id("_"), jen.Err()).Op("=").Id("decodeResponse").Call(jen.Id("resp"))
		}
		group.If(jen.Err().Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Err()),
		)

		group.Return(
			jen.Op("&").Id(endpoint.ServicePkgName).Dot(endpoint.Method.ResponseStruct).ValuesFunc(func(values *jen.Group) {
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
