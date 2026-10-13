package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func websocketSendSection(ws *WebSocketData) codegen.Section {
	return codegen.NewJenniferSection(ws.Type+"-websocket-send", func(stmt *jen.Statement) {
		addWebsocketSendSection(stmt, ws)
	})
}

func writeClientWebSocketSend(b *strings.Builder, ws *WebSocketData) {
	if ws.Payload != nil && ws.Payload.Init != nil {
		fmt.Fprintf(b, "\tbody := %s(v)\n", ws.Payload.Init.Name)
		b.WriteString("\treturn s.conn.WriteJSON(ctx, body)\n")
	} else {
		b.WriteString("\treturn s.conn.WriteJSON(ctx, v)\n")
	}
}

func writeServerWebSocketSend(b *strings.Builder, ws *WebSocketData) {
	writeServerWebSocketSendPreamble(b, ws)
	writeServerWebSocketSendResult(b, ws)
	if !writeServerWebSocketResponseBody(b, ws) {
		b.WriteString("\treturn s.conn.WriteJSON(ctx, res)\n")
	}
}

func writeServerWebSocketSendPreamble(b *strings.Builder, ws *WebSocketData) {
	if ws.SendName == "Send" {
		b.WriteString("\tvar err error\n")
		b.WriteString(renderWebsocketUpgrade(ws.Endpoint, ws.SendName, false, true))
		return
	}
	b.WriteString("\tdefer s.conn.Close()\n")
}

func writeServerWebSocketSendResult(b *strings.Builder, ws *WebSocketData) {
	if ws.Endpoint.Method.ViewedResult == nil {
		b.WriteString("\tres := v\n")
		return
	}
	if ws.Endpoint.Method.ViewedResult.ViewName != "" {
		fmt.Fprintf(b, "\tres, err := %s.%s(v, %q)\n", ws.PkgName, ws.Endpoint.Method.ViewedResult.Init.Name, ws.Endpoint.Method.ViewedResult.ViewName)
		b.WriteString("\tif err != nil {\n\t\treturn err\n\t}\n")
		return
	}
	fmt.Fprintf(b, "\tres, err := %s.%s(v, s.view)\n", ws.PkgName, ws.Endpoint.Method.ViewedResult.Init.Name)
	b.WriteString("\tif err != nil {\n\t\treturn err\n\t}\n")
}

func writeServerWebSocketResponseBody(b *strings.Builder, ws *WebSocketData) bool {
	if len(ws.Response.ServerBody) == 0 {
		return false
	}
	prefix := "\tbody := "
	viewed := ws.Endpoint.Method.ViewedResult
	if viewed != nil && viewed.ViewName == "" {
		b.WriteString("\tvar body any\n")
		prefix = "\t\tbody = "
	}
	writeServerStreamBodySelection(b, ws.Response.ServerBody, viewed, func(body *TypeData) {
		fmt.Fprintf(b, "%s%s\n", prefix, ws.Response.EncodePlan.BodyValues[body])
	})
	b.WriteString("\treturn s.conn.WriteJSON(ctx, body)\n")
	return true
}

func addWebsocketSendSection(stmt *jen.Statement, ws *WebSocketData) {
	// Emit SendWithContext with the real body, and Send as a thin forwarder
	// to keep the no-context convenience method available without duplicating
	// the send logic in two places.
	stmt.Line()
	codegen.Doc(stmt, ws.SendWithContextDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		BlockFunc(func(group *jen.Group) {
			var b strings.Builder
			writeWebSocketContextGuard(&b, "")
			b.WriteString("\terr := func() error {\n")
			if ws.Type != "server" {
				writeClientWebSocketSend(&b, ws)
				b.WriteString("\t}()\n")
				b.WriteString("\tif err != nil {\n")
				b.WriteString("\t\tif ctxErr := ctx.Err(); ctxErr != nil {\n")
				b.WriteString("\t\t\treturn ctxErr\n")
				b.WriteString("\t\t}\n")
				b.WriteString("\t}\n")
				b.WriteString("\treturn err\n")
				addRawWebSocketGroup(group, b.String())
				return
			}
			writeServerWebSocketSend(&b, ws)
			b.WriteString("\t}()\n")
			b.WriteString("\tif err != nil {\n")
			b.WriteString("\t\tif ctxErr := ctx.Err(); ctxErr != nil {\n")
			b.WriteString("\t\t\treturn ctxErr\n")
			b.WriteString("\t\t}\n")
			b.WriteString("\t}\n")
			b.WriteString("\treturn err\n")
			addRawWebSocketGroup(group, b.String())
		})
	stmt.Line()
	codegen.Doc(stmt, ws.SendDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendName).
		Params(jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		BlockFunc(func(group *jen.Group) {
			ctx := jen.Qual("context", "Background").Call()
			if ws.Type == "server" {
				ctx = jen.Id("s").Dot("r").Dot("Context").Call()
			}
			group.Return(jen.Id("s").Dot(ws.SendWithContextName).Call(ctx, jen.Id("v")))
		})
	stmt.Line()
}
