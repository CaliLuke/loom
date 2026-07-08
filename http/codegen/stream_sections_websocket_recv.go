package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func websocketRecvSection(ws *WebSocketData) codegen.Section {
	return codegen.MustJenniferSection(ws.Type+"-websocket-recv", func(stmt *jen.Statement) {
		addWebsocketRecvSection(stmt, ws)
	})
}

func writeWebsocketRecvVars(b *sourceBuilder, ws *WebSocketData) {
	b.Add("\tvar (\n")
	b.Addf("\t\trv %s\n", ws.RecvTypeRef)
	if ws.Type == "server" {
		if ws.RecvTypeIsPointer {
			b.Addf("\t\tbody %s\n", ws.Payload.VarName)
		} else {
			b.Addf("\t\tmsg *%s\n", ws.Payload.VarName)
		}
	} else {
		bodyTypeRef := ws.RecvTypeRef
		if ws.Response != nil && ws.Response.ClientBody != nil {
			bodyTypeRef = ws.Response.ClientBody.VarName
		}
		b.Addf("\t\tbody %s\n", bodyTypeRef)
	}
	b.Add("\t\terr error\n")
	b.Add("\t)\n")
}

func writeServerWebsocketRecvBody(b *sourceBuilder, ws *WebSocketData, withContext bool) {
	b.Add(renderWebsocketUpgrade(ws.Endpoint, ws.RecvName, true, withContext))
	if ws.RecvTypeIsPointer {
		b.Add("\tif err = s.conn.ReadJSON(&body); err != nil {\n")
	} else {
		b.Add("\tif err = s.conn.ReadJSON(&msg); err != nil {\n")
	}
	if withContext {
		b.Add("\t\tif ctxErr := ctx.Err(); ctxErr != nil {\n")
		b.Add("\t\t\treturn rv, ctxErr\n")
		b.Add("\t\t}\n")
	}
	b.Add("\t\treturn rv, err\n")
	b.Add("\t}\n")
	if ws.RecvTypeIsPointer {
		b.Add("\tif body == nil {\n")
	} else {
		b.Add("\tif msg == nil {\n")
	}
	b.Add("\t\treturn rv, io.EOF\n")
	b.Add("\t}\n")
	writeServerWebsocketRecvValidation(b, ws)
	writeServerWebsocketRecvReturn(b, ws)
}

func writeServerWebsocketRecvValidation(b *sourceBuilder, ws *WebSocketData) {
	if ws.Payload == nil || ws.Payload.ValidateRef == "" {
		return
	}
	if !ws.RecvTypeIsPointer {
		b.Add("\tbody := *msg\n")
	}
	b.Addf("\t%s\n", ws.Payload.ValidateRef)
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn rv, err\n")
	b.Add("\t}\n")
}

func writeServerWebsocketRecvReturn(b *sourceBuilder, ws *WebSocketData) {
	switch {
	case ws.Payload != nil && ws.Payload.Init != nil:
		if ws.RecvTypeIsPointer {
			b.Addf("\treturn %s(body), nil\n", ws.Payload.Init.Name)
		} else {
			b.Addf("\treturn %s(msg), nil\n", ws.Payload.Init.Name)
		}
	case ws.RecvTypeIsPointer:
		b.Add("\treturn body, nil\n")
	default:
		b.Add("\treturn *msg, nil\n")
	}
}

func writeClientWebsocketRecvBody(b *sourceBuilder, ws *WebSocketData) {
	if ws.RecvName == "CloseAndRecv" {
		b.Add("\tdefer s.conn.Close()\n")
		b.Add("\t// Send a nil payload to the server implying end of message\n")
		b.Add("\tif err = s.conn.WriteJSON(nil); err != nil {\n")
		b.Add("\t\treturn rv, err\n")
		b.Add("\t}\n")
	}
	b.Add("\terr = s.conn.ReadJSON(&body)\n")
	b.Add("\tif websocket.IsCloseError(err, websocket.CloseNormalClosure) {\n")
	if !ws.MustClose {
		b.Add("\t\ts.conn.Close()\n")
	}
	b.Add("\t\treturn rv, io.EOF\n")
	b.Add("\t}\n")
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn rv, err\n")
	b.Add("\t}\n")
	writeClientWebsocketRecvValidation(b, ws)
	writeClientWebsocketRecvReturn(b, ws)
}

func writeClientWebsocketRecvValidation(b *sourceBuilder, ws *WebSocketData) {
	if ws.Response.ClientBody == nil || ws.Response.ClientBody.ValidateRef == "" || ws.Endpoint.Method.ViewedResult != nil {
		return
	}
	b.Addf("\t%s\n", ws.Response.ClientBody.ValidateRef)
	b.Add("\tif err != nil {\n")
	b.Add("\t\treturn rv, err\n")
	b.Add("\t}\n")
}

func writeClientWebsocketRecvReturn(b *sourceBuilder, ws *WebSocketData) {
	if ws.Response.ResultInit == nil {
		b.Add("\treturn body, nil\n")
		return
	}
	b.Add("\tres := ")
	b.Addf("%s(", ws.Response.ResultInit.Name)
	for _, arg := range ws.Response.ResultInit.ClientArgs {
		b.Addf("%s,", arg.Ref)
	}
	b.Add(")\n")
	if ws.Endpoint.Method.ViewedResult == nil {
		b.Add("\treturn res, nil\n")
		return
	}
	writeClientWebsocketViewedResultReturn(b, ws)
}

func writeClientWebsocketViewedResultReturn(b *sourceBuilder, ws *WebSocketData) {
	view := ws.Endpoint.Method.ViewedResult
	prefix := ""
	if !view.IsCollection {
		prefix = "&"
	}
	viewArg := fmt.Sprintf("%q", view.ViewName)
	if view.ViewName == "" {
		viewArg = "s.view"
	}
	b.Addf("\tvres := %s%s.%s{res, %s }\n", prefix, view.ViewsPkg, view.VarName, viewArg)
	b.Addf("\tif err := %s.Validate%s(vres); err != nil {\n", view.ViewsPkg, ws.Endpoint.Method.Result)
	b.Addf("\t\treturn rv, loomhttp.ErrValidationError(%q, %q, err)\n", ws.Endpoint.ServiceName, ws.Endpoint.Method.Name)
	b.Add("\t}\n")
	b.Addf("\treturn %s.%s(vres), nil\n", ws.PkgName, view.ResultInit.Name)
}

func writeWebSocketContextGuard(b *sourceBuilder, returnValue string) {
	b.Add("\tif err := ctx.Err(); err != nil {\n")
	if returnValue != "" {
		b.Addf("\t\treturn %s, err\n", returnValue)
	} else {
		b.Add("\t\treturn err\n")
	}
	b.Add("\t}\n")
	b.Add("\tstopContextWatch := context.AfterFunc(ctx, func() {\n")
	b.Add("\t\tif s.conn == nil {\n")
	b.Add("\t\t\treturn\n")
	b.Add("\t\t}\n")
	b.Add("\t\tif closeErr := s.conn.Close(); closeErr != nil {\n")
	b.Add("\t\t\treturn\n")
	b.Add("\t\t}\n")
	b.Add("\t})\n")
	b.Add("\tdefer stopContextWatch()\n")
}

func addWebsocketRecvSection(stmt *jen.Statement, ws *WebSocketData) {
	stmt.Line()
	codegen.Doc(stmt, ws.RecvDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.RecvName).
		Params().
		Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			if ws.Type == "server" {
				group.Return(jen.Id("s").Dot(ws.RecvWithContextName).Call(jen.Id("s").Dot("r").Dot("Context").Call()))
				return
			}
			var b sourceBuilder
			writeWebsocketRecvVars(&b, ws)
			writeClientWebsocketRecvBody(&b, ws)
			addRawWebSocketGroup(group, b.String())
		})
	stmt.Line()
	codegen.Doc(stmt, ws.RecvWithContextDesc)
	stmt.Func().
		Params(jen.Id("s").Op("*").Id(ws.VarName)).
		Id(ws.RecvWithContextName).
		Params(jen.Id("ctx").Qual("context", "Context")).
		Params(codegen.TypeRef(ws.RecvTypeRef), jen.Error()).
		BlockFunc(func(group *jen.Group) {
			var b sourceBuilder
			if ws.Type == "server" {
				writeWebsocketRecvVars(&b, ws)
				writeWebSocketContextGuard(&b, "rv")
				writeServerWebsocketRecvBody(&b, ws, true)
			} else {
				b.Addf("\tvar rv %s\n", ws.RecvTypeRef)
				writeWebSocketContextGuard(&b, "rv")
				b.Addf("\tv, err := s.%s()\n", ws.RecvName)
				b.Add("\tif err != nil {\n")
				b.Add("\t\tif ctxErr := ctx.Err(); ctxErr != nil {\n")
				b.Add("\t\t\treturn rv, ctxErr\n")
				b.Add("\t\t}\n")
				b.Add("\t}\n")
				b.Add("\treturn v, err\n")
			}
			addRawWebSocketGroup(group, b.String())
		})
	stmt.Line()
}
