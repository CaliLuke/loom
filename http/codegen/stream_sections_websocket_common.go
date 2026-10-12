package codegen

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func renderWebsocketUpgrade(endpoint *EndpointData, function string, recv bool, withContext bool) string {
	var b strings.Builder
	b.WriteString("\t")
	b.WriteString(codegen.Comment("Upgrade the HTTP connection to a websocket connection only once. Connection upgrade is done here so that authorization logic in the endpoint is executed before calling the actual service method which may call " + function + "()."))
	b.WriteString("\n")
	b.WriteString("\ts.once.Do(func() {\n")
	if endpoint.Method.ViewedResult != nil && function == "Send" && endpoint.Method.ViewedResult.ViewName == "" {
		b.WriteString("\t\trespHdr := make(http.Header)\n")
		b.WriteString("\t\trespHdr.Add(\"loom-view\", s.view)\n")
	}
	b.WriteString("\t\tvar conn *websocket.Conn\n")
	if function == "Send" && endpoint.Method.ViewedResult != nil && endpoint.Method.ViewedResult.ViewName == "" {
		b.WriteString("\t\tconn, err = s.upgrader.Upgrade(s.w, s.r, respHdr)\n")
	} else {
		b.WriteString("\t\tconn, err = s.upgrader.Upgrade(s.w, s.r, nil)\n")
	}
	b.WriteString("\t\tif err != nil {\n")
	b.WriteString("\t\t\ts.upgradeErr = err\n")
	b.WriteString("\t\t\treturn\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tif err = s.conn.SetConn(s.r.Context(), conn, s.cancel); err != nil {\n\t\t\ts.upgradeErr = err\n\t\t\treturn\n\t\t}\n")
	if withContext {
		b.WriteString("\t\tif err = ctx.Err(); err != nil {\n")
		b.WriteString("\t\t\tif closeErr := s.conn.Close(); closeErr != nil {\n")
		b.WriteString("\t\t\t\ts.upgradeErr = closeErr\n")
		b.WriteString("\t\t\t\treturn\n")
		b.WriteString("\t\t\t}\n")
		b.WriteString("\t\t\ts.upgradeErr = err\n")
		b.WriteString("\t\t\treturn\n")
		b.WriteString("\t\t}\n")
	}
	b.WriteString("\t})\n")
	b.WriteString("\tif s.upgradeErr != nil {\n")
	if recv {
		b.WriteString("\t\treturn rv, s.upgradeErr\n")
	} else {
		b.WriteString("\t\treturn s.upgradeErr\n")
	}
	b.WriteString("\t}\n")
	return b.String()
}

func addRawWebSocketGroup(group *jen.Group, code string) {
	if strings.TrimSpace(code) == "" {
		return
	}
	if strings.HasPrefix(code, "\n") {
		group.Line()
	}
	group.Add(codegen.Expr(strings.TrimRight(code, "\n")))
}
