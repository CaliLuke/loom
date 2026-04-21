package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func websocketSendSection(ws *WebSocketData) codegen.Section {
	return codegen.MustJenniferSection(ws.Type+"-websocket-send", func(stmt *jen.Statement) {
		addWebsocketSendSection(stmt, ws)
	})
}

func writeClientWebSocketSend(b *sourceBuilder, ws *WebSocketData) {
	if ws.Payload != nil && ws.Payload.Init != nil {
		b.Addf("\tbody := %s(v)\n", ws.Payload.Init.Name)
		b.Add("\treturn s.conn.WriteJSON(body)\n")
	} else {
		b.Add("\treturn s.conn.WriteJSON(v)\n")
	}
}

func writeServerWebSocketSend(b *sourceBuilder, ws *WebSocketData) {
	writeServerWebSocketSendPreamble(b, ws)
	writeServerWebSocketSendResult(b, ws)
	if !writeServerWebSocketResponseBody(b, ws) {
		b.Add("\treturn s.conn.WriteJSON(res)\n")
	}
}

func writeServerWebSocketSendPreamble(b *sourceBuilder, ws *WebSocketData) {
	if ws.SendName == "Send" {
		b.Add("\tvar err error\n")
		b.Add(renderWebsocketUpgrade(ws.Endpoint, ws.SendName, false))
		return
	}
	b.Add("\tdefer s.conn.Close()\n")
}

func writeServerWebSocketSendResult(b *sourceBuilder, ws *WebSocketData) {
	if ws.Endpoint.Method.ViewedResult == nil {
		b.Add("\tres := v\n")
		return
	}
	if ws.Endpoint.Method.ViewedResult.ViewName != "" {
		b.Addf("\tres := %s.%s(v, %q)\n", ws.PkgName, ws.Endpoint.Method.ViewedResult.Init.Name, ws.Endpoint.Method.ViewedResult.ViewName)
		return
	}
	b.Addf("\tres := %s.%s(v, s.view)\n", ws.PkgName, ws.Endpoint.Method.ViewedResult.Init.Name)
}

func writeServerWebSocketResponseBody(b *sourceBuilder, ws *WebSocketData) bool {
	if len(ws.Response.ServerBody) == 0 {
		return false
	}
	body := ws.Response.ServerBody[0]
	if body.Init == nil {
		return false
	}
	writeServerWebSocketBodyInit(b, ws, body)
	b.Add("\treturn s.conn.WriteJSON(body)\n")
	return true
}

func writeServerWebSocketBodyInit(b *sourceBuilder, ws *WebSocketData, body *TypeData) {
	if ws.Endpoint.Method.ViewedResult == nil {
		writeServerBodyInitCall(b, body, "\tbody := ")
		return
	}
	if ws.Endpoint.Method.ViewedResult.ViewName != "" {
		if vsb := viewedServerBody(ws.Response.ServerBody, ws.Endpoint.Method.ViewedResult.ViewName); vsb != nil {
			writeServerBodyInitCall(b, vsb, "\tbody := ")
		}
		return
	}
	b.Add("\tvar body any\n")
	b.Add("\tswitch s.view {\n")
	for _, view := range ws.Endpoint.Method.ViewedResult.Views {
		writeViewedServerBodyCase(b, ws, view.Name)
	}
	b.Add("\t}\n")
}

func writeViewedServerBodyCase(b *sourceBuilder, ws *WebSocketData, viewName string) {
	if viewName == "default" {
		b.Addf("\tcase %q, \"\":\n", viewName)
	} else {
		b.Addf("\tcase %q:\n", viewName)
	}
	if vsb := viewedServerBody(ws.Response.ServerBody, viewName); vsb != nil {
		writeServerBodyInitCall(b, vsb, "\t\tbody = ")
	}
}

func writeServerBodyInitCall(b *sourceBuilder, body *TypeData, prefix string) {
	b.Addf("%s%s(", prefix, body.Init.Name)
	for _, arg := range body.Init.ServerArgs {
		b.Addf("%s, ", arg.Ref)
	}
	b.Add(")\n")
}

func addWebsocketSendSection(stmt *jen.Statement, ws *WebSocketData) {
	// Emit SendWithContext with the real body, and Send as a thin forwarder
	// to keep the no-context convenience method available without duplicating
	// the send logic in two places. The ctx parameter is currently unused by
	// the body; this layout means future work to honor it only touches
	// SendWithContext.
	stmt.Line()
	codegen.Doc(stmt, ws.SendWithContextDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		BlockFunc(func(group *jen.Group) {
			if ws.Type != "server" {
				var b sourceBuilder
				writeClientWebSocketSend(&b, ws)
				addRawWebSocketGroup(group, b.String())
				return
			}
			var b sourceBuilder
			writeServerWebSocketSend(&b, ws)
			addRawWebSocketGroup(group, b.String())
		})
	stmt.Line()
	codegen.Doc(stmt, ws.SendDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.SendName).
		Params(jen.Id("v").Add(codegen.TypeRef(ws.SendTypeRef))).
		Error().
		Block(
			jen.Return(jen.Id("s").Dot(ws.SendWithContextName).Call(jen.Qual("context", "Background").Call(), jen.Id("v"))),
		)
	stmt.Line()
}
