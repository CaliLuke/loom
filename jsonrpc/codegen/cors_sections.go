package codegen

import (
	"sort"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

func renderJSONRPCCORSPolicy(cors *httpcodegen.CORSData) jen.Code {
	origins := make([]jen.Code, 0, len(cors.Origins))
	for _, origin := range cors.Origins {
		fields := jen.Dict{jen.Id("Pattern"): jen.Lit(origin.Pattern)}
		if origin.Regex {
			fields[jen.Id("Regex")] = jen.True()
		}
		if len(origin.Methods) > 0 {
			fields[jen.Id("Methods")] = stringSliceLiteral(origin.Methods)
		}
		if len(origin.Headers) > 0 {
			fields[jen.Id("Headers")] = stringSliceLiteral(origin.Headers)
		}
		if len(origin.Expose) > 0 {
			fields[jen.Id("Expose")] = stringSliceLiteral(origin.Expose)
		}
		if origin.MaxAge > 0 {
			fields[jen.Id("MaxAge")] = jen.Lit(origin.MaxAge)
		}
		if origin.Credentials {
			fields[jen.Id("Credentials")] = jen.True()
		}
		origins = append(origins, jen.Values(fields))
	}
	return codegen.TypeRef("loomhttp.CORSPolicy").Values(jen.Dict{
		jen.Id("Origins"): jen.Index().Add(codegen.TypeRef("loomhttp.CORSOrigin")).Values(origins...),
	})
}

func writeJSONRPCCORSMounts(g *jen.Group, data *httpcodegen.ServiceData, hasSSE, hasMixed bool) {
	if data.CORS == nil {
		return
	}
	routes := jsonrpcCORSRoutes(data, hasSSE, hasMixed)
	paths := make([]string, 0, len(routes))
	for path := range routes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		methods := routes[path]
		var handle jen.Code
		if data.CORS.Runtime {
			handle = jen.Id("h").Dot("corsPolicy").Dot("HandlePreflight").Call(
				jen.Id("w"),
				jen.Id("r"),
				stringSliceLiteral(methods),
			)
		} else {
			handle = codegen.Expr("loomhttp.HandleCORSPreflight").Call(
				jen.Id("w"),
				jen.Id("r"),
				renderJSONRPCCORSPolicy(data.CORS),
				stringSliceLiteral(methods),
			)
		}
		g.Id("mux").Dot("Handle").Call(
			jen.Lit("OPTIONS"),
			jen.Lit(path),
			jen.Func().Params(
				jen.Id("w").Qual("net/http", "ResponseWriter"),
				jen.Id("r").Op("*").Qual("net/http", "Request"),
			).Block(handle),
		)
	}
}

func jsonrpcCORSRoutes(data *httpcodegen.ServiceData, hasSSE, hasMixed bool) map[string][]string {
	methodsByPath := make(map[string]map[string]struct{})
	add := func(path, method string) {
		if methodsByPath[path] == nil {
			methodsByPath[path] = make(map[string]struct{})
		}
		methodsByPath[path][method] = struct{}{}
	}
	for _, endpoint := range data.Endpoints {
		for _, route := range endpoint.Routes {
			add(route.Path, route.Verb)
			if hasMixed || (hasSSE && endpoint.Method.Name == "events/stream") {
				add(route.Path, "GET")
			}
		}
		if !hasSSE && !hasMixed {
			break
		}
	}
	out := make(map[string][]string, len(methodsByPath))
	for path, set := range methodsByPath {
		methods := make([]string, 0, len(set))
		for method := range set {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		out[path] = methods
	}
	return out
}

func stringSliceLiteral(values []string) jen.Code {
	items := make([]jen.Code, len(values))
	for i, value := range values {
		items[i] = jen.Lit(value)
	}
	return jen.Index().String().Values(items...)
}
