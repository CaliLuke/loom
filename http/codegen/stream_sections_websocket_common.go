package codegen

import (
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func renderWebsocketUpgrade(endpoint *EndpointData, function string, recv bool, withContext bool) string {
	var b sourceBuilder
	b.Add("\t")
	b.Add(codegen.Comment("Upgrade the HTTP connection to a websocket connection only once. Connection upgrade is done here so that authorization logic in the endpoint is executed before calling the actual service method which may call " + function + "()."))
	b.Add("\n")
	b.Add("\ts.once.Do(func() {\n")
	if endpoint.Method.ViewedResult != nil && function == "Send" && endpoint.Method.ViewedResult.ViewName == "" {
		b.Add("\t\trespHdr := make(http.Header)\n")
		b.Add("\t\trespHdr.Add(\"loom-view\", s.view)\n")
	}
	b.Add("\t\tvar conn *websocket.Conn\n")
	if function == "Send" && endpoint.Method.ViewedResult != nil && endpoint.Method.ViewedResult.ViewName == "" {
		b.Add("\t\tconn, err = s.upgrader.Upgrade(s.w, s.r, respHdr)\n")
	} else {
		b.Add("\t\tconn, err = s.upgrader.Upgrade(s.w, s.r, nil)\n")
	}
	b.Add("\t\tif err != nil {\n")
	b.Add("\t\t\ts.upgradeErr = err\n")
	b.Add("\t\t\treturn\n")
	b.Add("\t\t}\n")
	b.Add("\t\tif s.configurer != nil {\n")
	b.Add("\t\t\tconn = s.configurer(conn, s.cancel)\n")
	b.Add("\t\t}\n")
	b.Add("\t\ts.conn.SetConn(conn)\n")
	if withContext {
		b.Add("\t\tif err = ctx.Err(); err != nil {\n")
		b.Add("\t\t\tif closeErr := s.conn.Close(); closeErr != nil {\n")
		b.Add("\t\t\t\ts.upgradeErr = closeErr\n")
		b.Add("\t\t\t\treturn\n")
		b.Add("\t\t\t}\n")
		b.Add("\t\t\ts.upgradeErr = err\n")
		b.Add("\t\t\treturn\n")
		b.Add("\t\t}\n")
	}
	b.Add("\t})\n")
	b.Add("\tif s.upgradeErr != nil {\n")
	if recv {
		b.Add("\t\treturn rv, s.upgradeErr\n")
	} else {
		b.Add("\t\treturn s.upgradeErr\n")
	}
	b.Add("\t}\n")
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
