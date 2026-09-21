package service

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func authorizedEndpointMethodSection(m *EndpointMethodData) codegen.Section {
	return codegen.NewJenniferSection("endpoint-method", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("New%sEndpoint constructs a protected %s endpoint. Middleware runs after access checks; checks repeat before service execution. Missing evaluators panic at construction.", m.VarName, m.Name))
		stmt.Func().Id("New" + m.VarName + "Endpoint").ParamsFunc(func(g *jen.Group) {
			g.Id("s").Id(m.ServiceVarName)
			if params := authorizationCheckParams(m); params != "" {
				g.Add(codegen.Expr(params))
			}
			g.Id("middleware").Op("...").Func().Params(codegen.TypeRef("loom.Endpoint")).Add(codegen.TypeRef("loom.Endpoint"))
		}).Add(codegen.TypeRef("loom.Endpoint")).BlockFunc(func(g *jen.Group) {
			g.Id("check").Op(":=").Id(m.Authorization.checkName).Call(codegen.Expr(authorizationCheckArgs(m, false)))
			g.Id("endpoint").Op(":=").Add(codegen.Expr("security.Protect")).Call(
				jen.Id("check"),
				jen.Func().Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("req").Any()).Params(jen.Any(), jen.Error()).BlockFunc(func(body *jen.Group) {
					switch {
					case m.ServerStream != nil && m.ServerStream.EndpointStruct != "":
						body.Id("ep").Op(":=").Id("req").Assert(jen.Op("*").Id(m.ServerStream.EndpointStruct))
					case m.SkipRequestBodyEncodeDecode:
						body.Id("ep").Op(":=").Id("req").Assert(jen.Op("*").Id(m.RequestStruct))
					case m.PayloadRef != "":
						body.Id("p").Op(":=").Id("req").Assert(codegen.TypeRef(m.PayloadRef))
					}
					buildEndpointInvocation(body, m, payloadVar(m))
				}),
				jen.Id("middleware").Op("..."),
			)
			g.Return(jen.Id("endpoint"))
		})
	})
}

func authorizationEndpointCall(m *EndpointMethodData) *jen.Statement {
	return jen.Id("New" + m.VarName + "Endpoint").CallFunc(func(g *jen.Group) {
		g.Id("s")
		if args := authorizationCheckArgs(m, true); args != "" {
			g.Add(codegen.Expr(args))
		}
		if len(m.ServerInterceptors) > 0 {
			g.Func().Params(jen.Id("endpoint").Add(codegen.TypeRef("loom.Endpoint"))).Add(codegen.TypeRef("loom.Endpoint")).Block(
				jen.Return(jen.Id("Wrap"+m.VarName+"Endpoint").Call(jen.Id("endpoint"), jen.Id("si"))),
			)
		}
	})
}

func authorizationEndpointInit(g *jen.Group, data *EndpointsData) {
	if len(data.Schemes) > 0 {
		g.Id("a").Op(":=").Id("s").Assert(jen.Id("Authorizer"))
	}
	g.Id("endpoints").Op(":=").Op("&").Id(data.VarName).Values()
	for _, m := range data.Methods {
		g.Id("endpoints").Dot(m.VarName).Op("=").Add(newEndpointCall(m))
		if m.Authorization != nil {
			g.Id("endpoints").Dot(m.Authorization.fieldName).Op("=").Id(m.Authorization.checkName).Call(codegen.Expr(authorizationCheckArgs(m, true)))
		} else if len(m.ServerInterceptors) > 0 {
			g.Id("endpoints").Dot(m.VarName).Op("=").Id("Wrap"+m.VarName+"Endpoint").Call(jen.Id("endpoints").Dot(m.VarName), jen.Id("si"))
		}
	}
	g.Return(jen.Id("endpoints"))
}

// HasAccessAuthorizer reports whether generated endpoint constructors require
// the application access evaluator in addition to existing security hooks.
func (d *Data) HasAccessAuthorizer() bool {
	return d.Authorization != nil && len(d.Authorization.requirements) > 0
}

func authorizationEndpointUse(g *jen.Group, m *EndpointMethodData) {
	field := jen.Id("e").Dot(m.VarName)
	if m.Authorization == nil {
		g.Add(field).Op("=").Id("m").Call(field)
		return
	}
	g.Add(field).Op("=").Add(codegen.Expr("security.Protect")).Call(jen.Id("e").Dot(m.Authorization.fieldName), field, jen.Id("m"))
}
