package service

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

// AuthenticationData names the shared streaming authentication owner.
type AuthenticationData struct {
	// FactoryName constructs a bound authentication check.
	FactoryName string
	// FieldName stores the bound check on the endpoint aggregate.
	FieldName string
	// MethodName is the authentication-only admission entrypoint.
	MethodName string
}

func assignAuthenticationFields(methods []*MethodData, scope *codegen.NameScope) {
	members := codegen.NewNameScope()
	members.Unique("Use")
	for _, method := range methods {
		members.Unique(method.VarName)
	}
	for _, method := range methods {
		if method.ServerStream == nil || len(method.Schemes) == 0 {
			continue
		}
		method.Authentication = &AuthenticationData{
			FactoryName: scope.Unique("new" + method.VarName + "Authentication"),
			FieldName:   members.Unique("authenticate" + method.VarName),
			MethodName:  members.Unique("Authenticate" + method.VarName),
		}
	}
}

func authenticationFactoryCall(m *EndpointMethodData, aggregate bool) *jen.Statement {
	return jen.Id(m.Authentication.FactoryName).CallFunc(func(g *jen.Group) {
		for _, scheme := range m.Schemes.DedupeByType() {
			if aggregate {
				g.Id("a").Dot(scheme.Type + "Auth")
			} else {
				g.Id("auth" + scheme.Type + "Fn")
			}
		}
	})
}

func authenticationBinding(m *EndpointMethodData) *jen.Statement {
	if m.Authentication == nil {
		return jen.Null()
	}
	return jen.Id("authenticate").Op(":=").Add(authenticationFactoryCall(m, false))
}

func buildMethodAuthentication(g *jen.Group, m *EndpointMethodData, payload string) {
	if m.Authentication == nil {
		buildEndpointAuth(g, m, payload)
		return
	}
	g.List(jen.Id("authenticated"), jen.Err()).Op(":=").Add(codegen.Expr("security.RunCheck")).Call(jen.Id("ctx"), codegen.Expr(payload), jen.Id("authenticate"))
	g.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Nil(), jen.Err()))
	g.Id("ctx").Op("=").Id("authenticated")
}

func authenticationSection(m *EndpointMethodData) codegen.Section {
	return codegen.NewJenniferSection("endpoint-authentication", func(s *jen.Statement) {
		s.Func().Id(m.Authentication.FactoryName).ParamsFunc(func(g *jen.Group) {
			for _, scheme := range m.Schemes.DedupeByType() {
				g.Id("auth" + scheme.Type + "Fn").Add(codegen.TypeRef("security.Auth" + scheme.Type + "Func"))
			}
		}).Add(codegen.TypeRef("security.EndpointCheck")).BlockFunc(func(g *jen.Group) {
			g.Return(jen.Func().Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("req").Any()).Params(jen.Qual("context", "Context"), jen.Error()).BlockFunc(func(b *jen.Group) {
				if authenticationUsesPayload(m) {
					b.Id("p").Op(":=").Id("req").Assert(codegen.TypeRef(m.PayloadRef))
				}
				buildEndpointAuth(b, m, "p")
				b.Return(jen.Id("ctx"), jen.Nil())
			}))
		})
		s.Line()
		codegen.Doc(s, fmt.Sprintf("%s authenticates metadata-derived payload credentials before streaming request decoding. It does not validate the complete payload or evaluate application access. The normal endpoint checks security again after decoding.", m.Authentication.MethodName))
		s.Func().Params(jen.Id("e").Op("*").Id("Endpoints")).Id(m.Authentication.MethodName).Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("payload").Any()).Params(jen.Qual("context", "Context"), jen.Error()).Block(jen.Return(codegen.Expr("security.RunCheck").Call(jen.Id("ctx"), jen.Id("payload"), jen.Id("e").Dot(m.Authentication.FieldName))))
	})
}

func authenticationUsesPayload(m *EndpointMethodData) bool {
	for _, scheme := range m.Schemes {
		if !scheme.TransportOwned {
			return true
		}
	}
	return false
}
