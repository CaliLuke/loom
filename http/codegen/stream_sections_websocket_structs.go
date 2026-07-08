package codegen

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

func websocketStructSections(data *ServiceData, client bool) []codegen.Section {
	sections := []codegen.Section{websocketConnConfigurerStructSection(data, client)}
	for _, e := range data.Endpoints {
		var ws *WebSocketData
		if client {
			ws = e.ClientWebSocket
		} else {
			ws = e.ServerWebSocket
		}
		if ws != nil {
			sections = append(sections, websocketStructTypeSection(ws))
		}
	}
	return sections
}

func websocketCodeSections(data *ServiceData, client bool) []codegen.Section {
	sections := []codegen.Section{websocketConnConfigurerInitSection(data, client)}
	for _, e := range data.Endpoints {
		var ws *WebSocketData
		if client {
			ws = e.ClientWebSocket
		} else {
			ws = e.ServerWebSocket
		}
		if ws == nil {
			continue
		}
		if client {
			if ws.RecvTypeRef != "" {
				sections = append(sections, websocketRecvSection(ws))
			}
			switch ws.Kind {
			case expr.ClientStreamKind, expr.BidirectionalStreamKind:
				sections = append(sections, websocketSendSection(ws))
			}
		} else {
			if ws.SendTypeRef != "" {
				sections = append(sections, websocketSendSection(ws))
			}
			switch ws.Kind {
			case expr.ClientStreamKind, expr.BidirectionalStreamKind:
				sections = append(sections, websocketRecvSection(ws))
			}
		}
		if ws.MustClose {
			sections = append(sections, websocketCloseSection(ws))
		}
		if ws.Endpoint.Method.ViewedResult != nil && ws.Endpoint.Method.ViewedResult.ViewName == "" {
			sections = append(sections, websocketSetViewSection(ws))
		}
	}
	return sections
}

func websocketConnConfigurerStructSection(data *ServiceData, client bool) codegen.Section {
	prefix := "server"
	if client {
		prefix = "client"
	}
	return codegen.NewJenniferSection(prefix+"-websocket-conn-configurer-struct", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("ConnConfigurer holds the websocket connection configurer functions for the streaming endpoints in %q service.", data.Service.Name))
		stmt.Type().Id("ConnConfigurer").StructFunc(func(group *jen.Group) {
			for _, endpoint := range data.Endpoints {
				if IsWebSocketEndpoint(endpoint) {
					group.Id(endpoint.Method.VarName + "Fn").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc"))
				}
			}
		})
		stmt.Line()
	})
}

func websocketConnConfigurerInitSection(data *ServiceData, client bool) codegen.Section {
	prefix := "server"
	if client {
		prefix = "client"
	}
	return codegen.NewJenniferSection(prefix+"-websocket-conn-configurer-struct-init", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("NewConnConfigurer initializes the websocket connection configurer function with fn for all the streaming endpoints in %q service.", data.Service.Name))
		stmt.Func().
			Id("NewConnConfigurer").
			Params(jen.Id("fn").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc"))).
			Op("*").Id("ConnConfigurer").
			BlockFunc(func(group *jen.Group) {
				var b sourceBuilder
				b.Add("return &ConnConfigurer{\n")
				for _, endpoint := range data.Endpoints {
					if IsWebSocketEndpoint(endpoint) {
						b.Addf("\t%sFn: fn,\n", endpoint.Method.VarName)
					}
				}
				b.Add("}")
				appendHTTPRawBlock(group, b.String())
			})
		stmt.Line()
	})
}

func websocketStructTypeSection(ws *WebSocketData) codegen.Section {
	prefix := ws.Type
	return codegen.NewJenniferSection(prefix+"-websocket-struct-type", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("%s implements the %s interface.", ws.VarName, ws.Interface))
		stmt.Type().Id(ws.VarName).StructFunc(func(group *jen.Group) {
			if ws.Type == "server" {
				group.Id("once").Qual("sync", "Once")
				group.Comment("upgradeErr is the error returned by the websocket upgrade attempt.")
				group.Id("upgradeErr").Error()
				group.Comment("upgrader is the websocket connection upgrader.")
				group.Id("upgrader").Add(codegen.TypeRef("loomhttp.Upgrader"))
				group.Comment("configurer is the websocket connection configurer.")
				group.Id("configurer").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc"))
				addWrappedGroupComment(group, "cancel is the context cancellation function which cancels the request context when invoked.")
				group.Id("cancel").Qual("context", "CancelFunc")
				group.Comment("w is the HTTP response writer used in upgrading the connection.")
				group.Id("w").Qual("net/http", "ResponseWriter")
				group.Comment("r is the HTTP request.")
				group.Id("r").Op("*").Qual("net/http", "Request")
			}
			group.Comment("conn owns the websocket connection lifecycle.")
			group.Id("conn").Op("*").Add(codegen.TypeRef("loomhttp.WebSocketStream"))
			if ws.Type == "client" && ws.SendName == "" {
				group.Comment("done is closed when the client stream is closed.")
				group.Id("done").Chan().Struct()
				group.Comment("closeOnce closes done at most once.")
				group.Id("closeOnce").Qual("sync", "Once")
			}
			if ws.Endpoint.Method.ViewedResult != nil && ws.Endpoint.Method.ViewedResult.ViewName == "" {
				addWrappedGroupComment(group, fmt.Sprintf("view is the view to render %s result type before sending to the websocket connection.", ws.SendTypeName))
				group.Id("view").String()
			}
		})
		stmt.Line()
	})
}

func addWrappedGroupComment(group *jen.Group, text string) {
	for _, line := range strings.Split(codegen.Comment(text), "\n") {
		group.Comment(strings.TrimPrefix(line, "// "))
	}
}
