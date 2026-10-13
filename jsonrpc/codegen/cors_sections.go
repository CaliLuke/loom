package codegen

import (
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

func writeJSONRPCCORSMounts(g *jen.Group, data *httpcodegen.ServiceData) {
	if data.CORS == nil {
		return
	}
	for _, route := range data.MountPlan.Preflights {
		path, methods := route.Path, route.Methods
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

func stringSliceLiteral(values []string) jen.Code {
	items := make([]jen.Code, len(values))
	for i, value := range values {
		items[i] = jen.Lit(value)
	}
	return jen.Index().String().Values(items...)
}
