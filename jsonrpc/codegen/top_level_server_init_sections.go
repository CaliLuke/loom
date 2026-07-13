package codegen

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

//nolint:maintidx // Server constructor wiring intentionally aggregates transport-setup branches.
func jsonrpcServerInitSection(data *httpcodegen.ServiceData, hasSSE, hasMixed bool) codegen.Section {
	return codegen.NewJenniferSection("jsonrpc-server-init", func(stmt *jen.Statement) {
		comment := fmt.Sprintf("%s creates a JSON-RPC server which loads HTTP requests and calls the %q service methods.", data.ServerInit, data.Service.Name)
		if hasJSONRPCServerStream(data) {
			comment += " An optional streamWritePolicy bounds each server-stream network write and flush. Construct policies with loomhttp.NewStreamWritePolicy."
		}
		codegen.Doc(stmt, comment)
		params := jsonrpcServerInitParams(data)
		stmt.Func().Id(data.ServerInit).
			Params(params...).
			Op("*").Id(data.ServerStruct).
			BlockFunc(func(g *jen.Group) {
				writeJSONRPCServerInitBody(g, data, hasSSE, hasMixed)
			})
		stmt.Line()
	})
}

func jsonrpcServerInitParams(data *httpcodegen.ServiceData) []jen.Code {
	params := []jen.Code{}
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		params = append(params, jen.Id("streamHandler").Func().Params(jen.Qual("context", "Context"), codegen.TypeRef(data.Service.PkgName+".Stream")).Error())
	}
	params = append(params,
		jen.Id("endpoints").Op("*").Qual(data.Service.PkgName, "Endpoints"),
		jen.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer")),
		jen.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Decoder")),
		jen.Id("encoder").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter")).Add(codegen.TypeRef("loomhttp.Encoder")),
		jen.Id("errhandler").Func().Params(jen.Qual("context", "Context"), jen.Qual("net/http", "ResponseWriter"), jen.Error()),
	)
	if data.CORS != nil && data.CORS.Runtime {
		params = append(params, jen.Id("corsPolicy").Add(codegen.TypeRef("loomhttp.RuntimeCORSPolicy")))
	}
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		params = append(params,
			jen.Id("upgrader").Add(codegen.TypeRef("loomhttp.Upgrader")),
			jen.Id("configfn").Add(codegen.TypeRef("loomhttp.ConnConfigureFunc")),
		)
	}
	if hasJSONRPCServerStream(data) {
		params = append(params, jen.Id("streamWritePolicies").Op("...").Add(codegen.TypeRef("loomhttp.StreamWritePolicy")))
	}
	return params
}

func writeJSONRPCServerInitBody(g *jen.Group, data *httpcodegen.ServiceData, hasSSE, hasMixed bool) {
	if hasJSONRPCServerStream(data) {
		g.Var().Id("streamWritePolicy").Add(codegen.TypeRef("loomhttp.StreamWritePolicy"))
		g.If(jen.Len(jen.Id("streamWritePolicies")).Op(">").Lit(0)).Block(
			jen.Id("streamWritePolicy").Op("=").Id("streamWritePolicies").Index(jen.Lit(0)),
		)
	}
	dict := jsonrpcServerInitFields(data)
	g.Id("s").Op(":=").Op("&").Id(data.ServerStruct).Values(dict)
	handler := jsonrpcServerHandler(g, data, hasSSE, hasMixed)
	switch {
	case data.CORS != nil && data.CORS.Runtime:
		g.Id("s").Dot("Handler").Op("=").Id("corsPolicy").Dot("Handler").Call(handler)
	case data.CORS != nil:
		g.Id("s").Dot("Handler").Op("=").Add(codegen.Expr("loomhttp.CORSHandler")).Call(
			renderJSONRPCCORSPolicy(data.CORS),
			handler,
		)
	default:
		g.Id("s").Dot("Handler").Op("=").Qual("net/http", "NewCrossOriginProtection").Call().Dot("Handler").Call(handler)
	}
	g.Return(jen.Id("s"))
}

func jsonrpcServerInitFields(data *httpcodegen.ServiceData) jen.Dict {
	dict := jen.Dict{
		jen.Id("Methods"): jen.Index().String().ValuesFunc(func(values *jen.Group) {
			for _, endpoint := range data.Endpoints {
				values.Lit(endpoint.Method.Name)
			}
		}),
		jen.Id("decoder"):    jen.Id("decoder"),
		jen.Id("encoder"):    jen.Id("encoder"),
		jen.Id("errhandler"): jen.Id("errhandler"),
	}
	if httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]) {
		dict[jen.Id("StreamHandler")] = jen.Id("streamHandler")
		dict[jen.Id("upgrader")] = jen.Id("upgrader")
		dict[jen.Id("configfn")] = jen.Id("configfn")
	}
	if hasJSONRPCServerStream(data) {
		dict[jen.Id("streamWritePolicy")] = jen.Id("streamWritePolicy")
	}
	if data.CORS != nil && data.CORS.Runtime {
		dict[jen.Id("corsPolicy")] = jen.Id("corsPolicy")
	}
	for _, endpoint := range data.Endpoints {
		if httpcodegen.IsWebSocketEndpoint(endpoint) {
			dict[jen.Id(lowerInitial(endpoint.Method.VarName))] = jen.Id(endpoint.HandlerInit).Call(
				jen.Id("endpoints").Dot(endpoint.Method.VarName),
				jen.Id("mux"),
				jen.Id("decoder"),
			)
			if endpoint.Method.ServerStream != nil && (endpoint.Method.ServerStream.Kind == 3 || endpoint.Method.ServerStream.Kind == 4) {
				dict[jen.Id(lowerInitial(endpoint.Method.VarName)+"Endpoint")] = jen.Id("endpoints").Dot(endpoint.Method.VarName)
			}
			continue
		}
		args := []jen.Code{
			jen.Id("endpoints").Dot(endpoint.Method.VarName),
			jen.Id("mux"),
			jen.Id("decoder"),
			jen.Id("encoder"),
			jen.Id("errhandler"),
		}
		if httpcodegen.IsSSEEndpoint(endpoint) {
			args = append(args, jen.Id("streamWritePolicy"))
		}
		dict[jen.Id(endpoint.Method.VarName)] = jen.Id(endpoint.HandlerInit).Call(args...)
	}
	return dict
}

func jsonrpcServerHandler(
	g *jen.Group,
	data *httpcodegen.ServiceData,
	hasSSE bool,
	hasMixed bool,
) *jen.Statement {
	switch {
	case httpcodegen.IsWebSocketEndpoint(data.Endpoints[0]):
		g.Comment("WebSocket services upgrade through the internal dispatcher")
		return jen.Qual("net/http", "HandlerFunc").Call(jen.Id("s").Dot("serveHTTP"))
	case hasMixed:
		g.Comment("Mixed HTTP/SSE services negotiate transports through the internal dispatcher")
		return jen.Qual("net/http", "HandlerFunc").Call(jen.Id("s").Dot("serveHTTP"))
	case hasSSE:
		g.Comment("SSE-only services route via handleSSE")
		return jen.Qual("net/http", "HandlerFunc").Call(jen.Id("s").Dot("handleSSE"))
	default:
		g.Comment("Plain HTTP JSON-RPC")
		return jen.Qual("net/http", "HandlerFunc").Call(jen.Id("s").Dot("serveHTTP"))
	}
}
