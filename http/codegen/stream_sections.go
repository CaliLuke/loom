package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func sseClientSections(data *ServiceData) []codegen.Section {
	sections := make([]codegen.Section, 0)
	for _, ed := range data.Endpoints {
		if ed.SSE == nil {
			continue
		}
		sections = append(sections, sseClientSection(ed))
	}
	return sections
}

func sseClientSection(ed *EndpointData) codegen.Section {
	return codegen.NewJenniferSection("client-sse", func(stmt *jen.Statement) {
		addSSEClientSection(stmt, ed)
	})
}

func websocketCloseSection(ws *WebSocketData) codegen.Section {
	return codegen.NewJenniferSection(ws.Type+"-websocket-close", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("Close closes the %q endpoint websocket connection.", ws.Endpoint.Method.Name))
		stmt.Func().
			Params(jen.Id("s").Op("*").Id(ws.VarName)).
			Id("Close").
			Params().
			Error().
			BlockFunc(func(group *jen.Group) {
				addRawWebSocketGroup(group, renderWebSocketCloseBody(ws))
			})
		stmt.Line()
	})
}

func websocketSetViewSection(ws *WebSocketData) codegen.Section {
	return codegen.NewJenniferSection(ws.Type+"-websocket-set-view", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("SetView sets the view to render the %s type before sending to the %q endpoint websocket connection.", ws.SendTypeName, ws.Endpoint.Method.Name))
		stmt.Func().
			Params(jen.Id("s").Op("*").Id(ws.VarName)).
			Id("SetView").
			Params(jen.Id("view").String()).
			Block(
				jen.Id("s").Dot("view").Op("=").Id("view"),
			)
		stmt.Line()
	})
}

func buildStreamRequestSection(endpoint *EndpointData) codegen.Section {
	return codegen.NewJenniferSection("build-stream-request", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s creates a streaming endpoint request payload from the method payload and the path to the file to be streamed", endpoint.BuildStreamPayload))
		stmt.Func().
			Id(endpoint.BuildStreamPayload).
			ParamsFunc(func(group *jen.Group) {
				if endpoint.Payload.Ref != "" {
					group.Id("payload").Any()
				}
				group.Id("fpath").String()
			}).
			Params(jen.Op("*").Id(requestStructPkg(endpoint.Method, endpoint.ServicePkgName)).Dot(endpoint.Method.RequestStruct), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				addRawWebSocketGroup(group, renderBuildStreamRequestBody(endpoint))
			})
		stmt.Line()
	})
}

func multipartRequestEncoderTypeSection(data *MultipartData) codegen.Section {
	return codegen.NewJenniferSection("multipart-request-encoder-type", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s is the type to encode multipart request for the %q service %q endpoint.", data.FuncName, data.ServiceName, data.MethodName))
		stmt.Type().Id(data.FuncName).Func().Params(
			jen.Op("*").Qual("mime/multipart", "Writer"),
			codegen.TypeRef(data.Payload.Ref),
		).Error()
		stmt.Line()
	})
}

func renderWebSocketCloseBody(ws *WebSocketData) string {
	var b sourceBuilder
	b.Add("var err error\n")
	if ws.Type == "client" && ws.SendName == "" {
		b.Add("s.closeOnce.Do(func() {\n\tif s.done != nil {\n\t\tclose(s.done)\n\t}\n})\n")
	}
	if ws.Type == "server" {
		b.Add("if s.conn == nil {\n\treturn nil\n}\n")
		b.Add("if err = s.conn.WriteControl(\n")
		b.Add("\twebsocket.CloseMessage,\n")
		b.Add("\twebsocket.FormatCloseMessage(websocket.CloseNormalClosure, \"server closing connection\"),\n")
		b.Add("\ttime.Now().Add(time.Second),\n")
		b.Add("); err != nil {\n\treturn err\n}\n")
	} else {
		b.Add("// Send a nil payload to the server implying client closing connection.\n")
		b.Add("if err = s.conn.WriteJSON(nil); err != nil {\n\treturn err\n}\n")
	}
	b.Add("return s.conn.Close()")
	return b.String()
}

func renderBuildStreamRequestBody(endpoint *EndpointData) string {
	var b sourceBuilder
	b.Add("f, err := os.Open(fpath)\n")
	b.Add("if err != nil {\n\treturn nil, err\n}\n")
	b.Addf("return &%s.%s{\n", requestStructPkg(endpoint.Method, endpoint.ServicePkgName), endpoint.Method.RequestStruct)
	if endpoint.Payload.Ref != "" {
		b.Addf("\tPayload: payload.(%s),\n", endpoint.Payload.Ref)
	}
	b.Add("\tBody: f,\n")
	b.Add("}, nil")
	return b.String()
}

func addSSEClientInterface(stmt *jen.Statement, ed *EndpointData, streamName string) {
	stmt.Line()
	codegen.Doc(stmt, streamName+" is the interface for reading Server-Sent Events.")
	stmt.Type().Id(streamName).Interface(
		jen.Comment("Recv reads and returns the next event from the SSE stream."),
		jen.Id("Recv").Params(jen.Qual("context", "Context")).Params(codegen.TypeRef(ed.SSE.EventTypeRef), jen.Error()),
		jen.Comment("Close closes the SSE stream and releases resources."),
		jen.Id("Close").Params().Error(),
	)
}

func addSSEClientImplStruct(stmt *jen.Statement, ed *EndpointData, streamName, implName string) {
	stmt.Line()
	stmt.Type().DefsFunc(func(group *jen.Group) {
		group.Comment(implName + " implements the " + streamName + " interface.")
		fields := []jen.Code{
			jen.Id("resp").Op("*").Qual("net/http", "Response"),
			jen.Id("buffer").Index().Byte().Comment("Buffer for unprocessed data"),
			jen.Id("readLock").Qual("sync", "Mutex"),
			jen.Id("lock").Qual("sync", "Mutex"),
			jen.Id("closed").Bool(),
		}
		if sseClientNeedsDecoder(ed) {
			fields = append(fields, jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")))
		}
		group.Id(implName).Struct(fields...)
	})
	stmt.Line()
	codegen.Doc(stmt, implName+" implements the "+streamName+" interface.")
	stmt.Var().Id("_").Id(streamName).Op("=").Parens(jen.Op("*").Id(implName)).Call(jen.Nil())
}

func addSSEClientConstructor(stmt *jen.Statement, ed *EndpointData, streamName, implName string) {
	stmt.Line()
	codegen.Doc(stmt, "New"+ed.Method.VarName+"Stream creates a new "+streamName+".")
	stmt.Func().
		Id("New"+ed.Method.VarName+"Stream").
		Params(
			jen.Id("resp").Op("*").Qual("net/http", "Response"),
			jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Response")).Add(codegen.TypeRef("loomhttp.Decoder")),
		).
		Id(streamName).
		BlockFunc(func(group *jen.Group) {
			values := jen.Dict{
				jen.Id("resp"):   jen.Id("resp"),
				jen.Id("buffer"): jen.Make(jen.Index().Byte(), jen.Lit(0), jen.Lit(4096)),
			}
			if sseClientNeedsDecoder(ed) {
				values[jen.Id("decoder")] = jen.Id("decoder")
			}
			group.Return(
				jen.Op("&").Id(implName).Values(values),
			)
		})
}

func renderSSEClientRecvBody() string {
	var b sourceBuilder
	b.Add("var byts []byte\n")
	b.Add("byts, err = s.readEvent(ctx)\n")
	b.Add("if err != nil {\n")
	b.Add("\tif errors.Is(err, io.EOF) {\n")
	b.Add("\t\ts.Close()\n")
	b.Add("\t\treturn event, io.EOF\n")
	b.Add("\t}\n")
	b.Add("\tif errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {\n")
	b.Add("\t\ts.Close()\n")
	b.Add("\t}\n")
	b.Add("\treturn\n")
	b.Add("}\n")
	b.Add("return s.processEvent(byts)")
	return b.String()
}

func renderSSEClientCloseBody() string {
	var b sourceBuilder
	b.Add("s.lock.Lock()\n")
	b.Add("if s.closed {\n\ts.lock.Unlock()\n\treturn nil\n}\n")
	b.Add("s.closed = true\n")
	b.Add("body := s.resp.Body\n")
	b.Add("s.lock.Unlock()\n\n")
	b.Add("return body.Close()")
	return b.String()
}
