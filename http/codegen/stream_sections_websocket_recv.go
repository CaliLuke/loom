package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func websocketRecvSection(ws *WebSocketData) codegen.Section {
	return codegen.NewJenniferSection(ws.Type+"-websocket-recv", func(stmt *jen.Statement) {
		addWebsocketRecvSection(stmt, ws)
	})
}

func writeWebsocketRecvVars(b *strings.Builder, ws *WebSocketData) {
	b.WriteString("\tvar (\n")
	fmt.Fprintf(b, "\t\trv %s\n", ws.RecvTypeRef)
	if ws.Type == "server" {
		if ws.RecvTypeIsPointer {
			fmt.Fprintf(b, "\t\tbody %s\n", ws.Payload.VarName)
		} else {
			fmt.Fprintf(b, "\t\tmsg *%s\n", ws.Payload.VarName)
		}
	} else {
		bodyTypeRef := ws.RecvTypeRef
		if ws.Response != nil && ws.Response.ClientBody != nil {
			bodyTypeRef = ws.Response.ClientBody.ValueRef
		}
		fmt.Fprintf(b, "\t\tbody %s\n", bodyTypeRef)
	}
	b.WriteString("\t\terr error\n")
	b.WriteString("\t)\n")
}

func writeServerWebsocketRecvBody(b *strings.Builder, ws *WebSocketData, withContext bool) {
	b.WriteString(renderWebsocketUpgrade(ws.Endpoint, ws.RecvName, true, withContext))
	if ws.RecvTypeIsPointer {
		b.WriteString("\tif err = s.conn.ReadJSON(ctx, &body); err != nil {\n")
	} else {
		b.WriteString("\tif err = s.conn.ReadJSON(ctx, &msg); err != nil {\n")
	}
	b.WriteString("\t\treturn rv, err\n")
	b.WriteString("\t}\n")
	if ws.RecvTypeIsPointer {
		b.WriteString("\tif body == nil {\n")
	} else {
		b.WriteString("\tif msg == nil {\n")
	}
	b.WriteString("\t\treturn rv, io.EOF\n")
	b.WriteString("\t}\n")
	writeServerWebsocketRecvValidation(b, ws)
	writeServerWebsocketRecvReturn(b, ws)
}

func writeServerWebsocketRecvValidation(b *strings.Builder, ws *WebSocketData) {
	validate := serverWebSocketPayloadValidation(ws)
	if validate == "" {
		return
	}
	if !ws.RecvTypeIsPointer {
		b.WriteString("\tbody := *msg\n")
	}
	fmt.Fprintf(b, "\t%s\n", validate)
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn rv, err\n")
	b.WriteString("\t}\n")
}

func serverWebSocketPayloadValidation(ws *WebSocketData) string {
	if ws.Payload != nil && ws.Payload.ValidateRef != "" {
		return ws.Payload.ValidateRef
	}
	if ws.Payload == nil || ws.Payload.Init == nil {
		return ""
	}
	for _, arg := range ws.Payload.Init.ServerArgs {
		if arg.AttributeData != nil && arg.AttributeData.Validate != "" {
			return arg.AttributeData.Validate
		}
	}
	return ""
}

// writeServerWebsocketRecvReturn writes the statement that returns the
// received payload. The constructor of a payload takes a primitive body, such
// as the body of a named primitive type, by value.
func writeServerWebsocketRecvReturn(b *strings.Builder, ws *WebSocketData) {
	switch {
	case ws.Payload != nil && ws.Payload.Init != nil:
		switch {
		case ws.RecvTypeIsPointer:
			fmt.Fprintf(b, "\treturn %s(body), nil\n", ws.Payload.Init.Name)
		case websocketPayloadIsPrimitive(ws):
			fmt.Fprintf(b, "\treturn %s(*msg), nil\n", ws.Payload.Init.Name)
		default:
			fmt.Fprintf(b, "\treturn %s(msg), nil\n", ws.Payload.Init.Name)
		}
	case ws.RecvTypeIsPointer:
		b.WriteString("\treturn body, nil\n")
	default:
		b.WriteString("\treturn *msg, nil\n")
	}
}

// websocketPayloadIsPrimitive reports whether the payload constructor of the
// server stream ws takes a primitive body.
func websocketPayloadIsPrimitive(ws *WebSocketData) bool {
	args := ws.Payload.Init.ServerArgs
	return len(args) == 1 && args[0].AttributeData != nil && expr.IsPrimitive(args[0].AttributeData.Type)
}

func writeClientWebsocketRecvBody(b *strings.Builder, ws *WebSocketData, ctxExpr string) {
	if ws.RecvName == "CloseAndRecv" {
		b.WriteString("\tdefer s.conn.Close()\n")
		b.WriteString("\t// Send a nil payload to the server implying end of message\n")
		fmt.Fprintf(b, "\tif err = s.conn.WriteJSON(%s, nil); err != nil {\n", ctxExpr)
		b.WriteString("\t\treturn rv, err\n")
		b.WriteString("\t}\n")
	}
	fmt.Fprintf(b, "\terr = s.conn.ReadJSON(%s, &body)\n", ctxExpr)
	b.WriteString("\tif websocket.IsCloseError(err, websocket.CloseNormalClosure) {\n")
	if ws.Type == "client" && ws.SendName == "" {
		b.WriteString("\t\ts.closeOnce.Do(func() {\n\t\t\tif s.done != nil {\n\t\t\t\tclose(s.done)\n\t\t\t}\n\t\t})\n")
		b.WriteString("\t\tif closeErr := s.conn.Close(); closeErr != nil {\n\t\t\treturn rv, closeErr\n\t\t}\n")
	} else if !ws.MustClose {
		b.WriteString("\t\ts.conn.Close()\n")
	}
	b.WriteString("\t\treturn rv, io.EOF\n")
	b.WriteString("\t}\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn rv, err\n")
	b.WriteString("\t}\n")
	writeClientWebsocketRecvValidation(b, ws)
	writeClientWebsocketRecvReturn(b, ws)
}

func writeClientWebsocketRecvValidation(b *strings.Builder, ws *WebSocketData) {
	if ws.Response.ClientBody == nil || ws.Response.ClientBody.ValidateRef == "" || ws.Endpoint.Method.ViewedResult != nil {
		return
	}
	fmt.Fprintf(b, "\t%s\n", ws.Response.ClientBody.ValidateRef)
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\treturn rv, err\n")
	b.WriteString("\t}\n")
}

func writeClientWebsocketRecvReturn(b *strings.Builder, ws *WebSocketData) {
	if ws.Response.ResultInit == nil {
		b.WriteString("\treturn body, nil\n")
		return
	}
	b.WriteString("\tres := ")
	fmt.Fprintf(b, "%s(", ws.Response.ResultInit.Name)
	for _, arg := range ws.Response.ResultInit.ClientArgs {
		fmt.Fprintf(b, "%s,", arg.Ref)
	}
	b.WriteString(")\n")
	if ws.Endpoint.Method.ViewedResult == nil {
		b.WriteString("\treturn res, nil\n")
		return
	}
	writeClientWebsocketViewedResultReturn(b, ws)
}

func writeClientWebsocketViewedResultReturn(b *strings.Builder, ws *WebSocketData) {
	view := ws.Endpoint.Method.ViewedResult
	prefix := ""
	if !view.IsCollection {
		prefix = "&"
	}
	viewArg := fmt.Sprintf("%q", view.ViewName)
	if view.ViewName == "" {
		viewArg = "s.view"
	}
	fmt.Fprintf(b, "\tvres := %s%s{Projected: res, View: %s}\n", prefix, view.FullName, viewArg)
	fmt.Fprintf(b, "\tif err := %s.Validate%s(vres); err != nil {\n", view.ViewsPkg, ws.Endpoint.Method.Result)
	fmt.Fprintf(b, "\t\treturn rv, loomhttp.ErrValidationError(%q, %q, err)\n", ws.Endpoint.ServiceName, ws.Endpoint.Method.Name)
	b.WriteString("\t}\n")
	fmt.Fprintf(b, "\tresult, err := %s.%s(vres)\n", ws.PkgName, view.ResultInit.Name)
	b.WriteString("\tif err != nil {\n")
	fmt.Fprintf(b, "\t\treturn rv, loomhttp.ErrValidationError(%q, %q, err)\n", ws.Endpoint.ServiceName, ws.Endpoint.Method.Name)
	b.WriteString("\t}\n")
	b.WriteString("\treturn result, nil\n")
}

func writeWebSocketContextGuard(b *strings.Builder, returnValue string) {
	b.WriteString("\tif err := ctx.Err(); err != nil {\n")
	if returnValue != "" {
		fmt.Fprintf(b, "\t\treturn %s, err\n", returnValue)
	} else {
		b.WriteString("\t\treturn err\n")
	}
	b.WriteString("\t}\n")
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
			var b strings.Builder
			writeWebsocketRecvVars(&b, ws)
			writeClientWebsocketRecvBody(&b, ws, "context.Background()")
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
			var b strings.Builder
			if ws.Type == "server" {
				writeWebsocketRecvVars(&b, ws)
				writeWebSocketContextGuard(&b, "rv")
				writeServerWebsocketRecvBody(&b, ws, true)
			} else {
				writeWebsocketRecvVars(&b, ws)
				writeWebSocketContextGuard(&b, "rv")
				writeClientWebsocketRecvBody(&b, ws, "ctx")
			}
			addRawWebSocketGroup(group, b.String())
		})
	stmt.Line()
}
