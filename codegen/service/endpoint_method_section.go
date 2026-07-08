package service

import (
	"fmt"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func endpointMethodSection(method *EndpointMethodData) codegen.Section {
	return codegen.MustJenniferSection("endpoint-method", func(stmt *jen.Statement) {
		stmt.Line()
		codegen.Doc(stmt, fmt.Sprintf("New%sEndpoint returns an endpoint function that calls the method %q of service %q.", method.VarName, method.Name, method.ServiceName))
		stmt.Func().Id("New" + method.VarName + "Endpoint").ParamsFunc(func(group *jen.Group) {
			group.Id("s").Id(method.ServiceVarName)
			for _, scheme := range method.Schemes.DedupeByType() {
				group.Id("auth" + scheme.Type + "Fn").Add(codegen.Expr("security.Auth" + scheme.Type + "Func"))
			}
		}).Add(codegen.Expr("loom.Endpoint")).Block(
			jen.Return(
				jen.Func().Params(
					jen.Id("ctx").Qual("context", "Context"),
					jen.Id("req").Any(),
				).Params(
					jen.Any(),
					jen.Error(),
				).BlockFunc(func(group *jen.Group) {
					switch {
					case method.ServerStream != nil:
						if method.ServerStream.EndpointStruct != "" {
							group.Id("ep").Op(":=").Id("req").Assert(jen.Op("*").Id(method.ServerStream.EndpointStruct))
						} else if len(method.Requirements) > 0 && method.PayloadRef != "" {
							group.Id("p").Op(":=").Id("req").Assert(codegen.TypeRef(method.PayloadRef))
						}
					case method.SkipRequestBodyEncodeDecode:
						group.Id("ep").Op(":=").Id("req").Assert(jen.Op("*").Id(method.RequestStruct))
					case method.PayloadRef != "":
						group.Id("p").Op(":=").Id("req").Assert(codegen.TypeRef(method.PayloadRef))
					}

					payload := payloadVar(method)
					if len(method.Requirements) > 0 {
						buildEndpointAuth(group, method, payload)
					}

					switch {
					case method.ServerStream != nil:
						buildStreamingEndpointInvocation(group, method, payload)
					case method.SkipRequestBodyEncodeDecode:
						buildSkipRequestEndpointInvocation(group, method)
					case method.ViewedResult != nil:
						buildViewedResultEndpointInvocation(group, method, payload)
					case method.SkipResponseBodyEncodeDecode:
						buildSkipResponseEndpointInvocation(group, method, payload)
					default:
						buildDefaultEndpointInvocation(group, method, payload)
					}
				}),
			),
		)
		stmt.Line()
	})
}

func buildEndpointAuth(group *jen.Group, method *EndpointMethodData, payload string) {
	group.Var().Id("err").Error()
	for ridx, req := range method.Requirements {
		if ridx != 0 {
			group.If(jen.Id("err").Op("!=").Nil()).BlockFunc(func(nested *jen.Group) {
				buildRequirementSchemes(nested, req, payload)
			})
			continue
		}
		buildRequirementSchemes(group, req, payload)
	}
	group.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Id("err")),
	)
}

func buildRequirementSchemes(group *jen.Group, req *RequirementData, payload string) {
	for sidx, scheme := range req.Schemes {
		if sidx != 0 {
			group.If(jen.Id("err").Op("==").Nil()).BlockFunc(func(nested *jen.Group) {
				buildSchemeAuth(nested, req, scheme, payload)
			})
			continue
		}
		buildSchemeAuth(group, req, scheme, payload)
	}
}

func buildSchemeAuth(group *jen.Group, req *RequirementData, scheme *SchemeData, payload string) {
	switch scheme.Type {
	case "Basic":
		buildBasicSchemeAuth(group, req, scheme, payload)
	case "APIKey":
		buildCredentialSchemeAuth(group, req, scheme, payload, "APIKey", "key")
	case "JWT":
		buildCredentialSchemeAuth(group, req, scheme, payload, "JWT", "token")
	case "OAuth2":
		buildOAuth2SchemeAuth(group, req, scheme, payload)
	}
}

func buildBasicSchemeAuth(group *jen.Group, req *RequirementData, scheme *SchemeData, payload string) {
	buildSchemeStruct(group, "BasicScheme", scheme.SchemeName, scheme.Scopes, req.Scopes, nil)
	buildPointerStringBinding(group, "user", payload, scheme.UsernameField, scheme.UsernamePointer)
	buildPointerStringBinding(group, "pass", payload, scheme.PasswordField, scheme.PasswordPointer)
	userExpr := payloadFieldExpr(payload, scheme.UsernameField, scheme.UsernamePointer, "user")
	passExpr := payloadFieldExpr(payload, scheme.PasswordField, scheme.PasswordPointer, "pass")
	group.List(jen.Id("ctx"), jen.Id("err")).Op("=").Id("auth"+scheme.Type+"Fn").Call(
		jen.Id("ctx"),
		userExpr,
		passExpr,
		jen.Op("&").Id("sc"),
	)
}

func buildCredentialSchemeAuth(group *jen.Group, req *RequirementData, scheme *SchemeData, payload, schemeStruct, tempVar string) {
	buildSchemeStruct(group, schemeStruct+"Scheme", scheme.SchemeName, scheme.Scopes, req.Scopes, nil)
	if scheme.TransportOwned {
		group.List(jen.Id("ctx"), jen.Id("err")).Op("=").Id("auth"+scheme.Type+"Fn").Call(
			jen.Id("ctx"),
			jen.Lit(""),
			jen.Op("&").Id("sc"),
		)
		return
	}
	buildPointerStringBinding(group, tempVar, payload, scheme.CredField, scheme.CredPointer)
	expr := payloadFieldExpr(payload, scheme.CredField, scheme.CredPointer, tempVar)
	group.List(jen.Id("ctx"), jen.Id("err")).Op("=").Id("auth"+scheme.Type+"Fn").Call(
		jen.Id("ctx"),
		expr,
		jen.Op("&").Id("sc"),
	)
}

func buildOAuth2SchemeAuth(group *jen.Group, req *RequirementData, scheme *SchemeData, payload string) {
	buildSchemeStruct(group, "OAuth2Scheme", scheme.SchemeName, scheme.Scopes, req.Scopes, func(values *jen.Group) {
		if len(scheme.Flows) == 0 {
			return
		}
		values.Id("Flows").Op(":").Index().Op("*").Add(codegen.Expr("security.OAuthFlow")).CustomFunc(multilineValues, func(flows *jen.Group) {
			for _, flow := range scheme.Flows {
				flows.Op("&").Add(codegen.Expr("security.OAuthFlow")).CustomFunc(multilineValues, func(fields *jen.Group) {
					fields.Id("Type").Op(":").Lit(flow.Type())
					if flow.AuthorizationURL != "" {
						fields.Id("AuthorizationURL").Op(":").Lit(flow.AuthorizationURL)
					}
					if flow.TokenURL != "" {
						fields.Id("TokenURL").Op(":").Lit(flow.TokenURL)
					}
					if flow.RefreshURL != "" {
						fields.Id("RefreshURL").Op(":").Lit(flow.RefreshURL)
					}
				})
			}
		})
	})
	buildPointerStringBinding(group, "token", payload, scheme.CredField, scheme.CredPointer)
	expr := payloadFieldExpr(payload, scheme.CredField, scheme.CredPointer, "token")
	group.List(jen.Id("ctx"), jen.Id("err")).Op("=").Id("auth"+scheme.Type+"Fn").Call(
		jen.Id("ctx"),
		expr,
		jen.Op("&").Id("sc"),
	)
}

func buildSchemeStruct(group *jen.Group, schemeType, schemeName string, scopes, requiredScopes []string, extra func(*jen.Group)) {
	group.Id("sc").Op(":=").Add(codegen.Expr("security."+schemeType)).CustomFunc(multilineValues, func(values *jen.Group) {
		values.Id("Name").Op(":").Lit(schemeName)
		values.Id("Scopes").Op(":").Index().String().ValuesFunc(func(items *jen.Group) {
			for _, scope := range scopes {
				items.Lit(scope)
			}
		})
		values.Id("RequiredScopes").Op(":").Index().String().ValuesFunc(func(items *jen.Group) {
			for _, scope := range requiredScopes {
				items.Lit(scope)
			}
		})
		if extra != nil {
			extra(values)
		}
	})
}

func buildPointerStringBinding(group *jen.Group, tempVar, payload, field string, isPointer bool) {
	if !isPointer {
		return
	}
	group.Var().Id(tempVar).String()
	group.If(jen.Add(codegen.Expr(payload)).Dot(field).Op("!=").Nil()).Block(
		jen.Id(tempVar).Op("=").Op("*").Add(codegen.Expr(payload)).Dot(field),
	)
}

func payloadFieldExpr(payload, field string, isPointer bool, tempVar string) *jen.Statement {
	if isPointer {
		return jen.Id(tempVar)
	}
	return jen.Add(codegen.Expr(payload)).Dot(field)
}

func buildStreamingEndpointInvocation(group *jen.Group, method *EndpointMethodData, payload string) {
	if method.ServerStream.EndpointStruct != "" {
		buildEndpointStructStreamingInvocation(group, method, payload)
		return
	}
	buildDirectStreamingInvocation(group, method)
}

func buildEndpointStructStreamingInvocation(group *jen.Group, method *EndpointMethodData, payload string) {
	if method.HasMixedResults {
		buildMixedStreamingEndpointInvocation(group, method, payload)
		return
	}
	group.Return(
		jen.Nil(),
		jen.Id("s").Dot(method.VarName).CallFunc(func(args *jen.Group) {
			args.Id("ctx")
			if method.PayloadRef != "" {
				args.Add(codegen.Expr(payload))
			}
			args.Id("ep").Dot("Stream")
		}),
	)
}

func buildMixedStreamingEndpointInvocation(group *jen.Group, method *EndpointMethodData, payload string) {
	lhs := []jen.Code{}
	if method.ResultRef != "" {
		lhs = append(lhs, jen.Id("res"))
	}
	if method.ViewedResult != nil && method.ViewedResult.ViewName == "" {
		lhs = append(lhs, jen.Id("view"))
	}
	lhs = append(lhs, jen.Id("err"))
	group.List(lhs...).Op(":=").Id("s").Dot(method.VarName).CallFunc(func(args *jen.Group) {
		args.Id("ctx")
		if method.PayloadRef != "" {
			args.Add(codegen.Expr(payload))
		}
		args.Id("ep").Dot("Stream")
	})
	group.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Id("err")),
	)
	if method.ViewedResult != nil {
		viewExpr := jen.Lit(method.ViewedResult.ViewName)
		if method.ViewedResult.ViewName == "" {
			viewExpr = jen.Id("view")
		}
		group.List(jen.Id("vres"), jen.Id("err")).Op(":=").Id(method.ViewedResult.Init.Name).Call(
			jen.Id("res"),
			viewExpr,
		)
		group.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Return(jen.Nil(), jen.Id("err")),
		)
		group.Return(jen.Id("vres"), jen.Nil())
		return
	}
	group.Return(jen.Id("res"), jen.Nil())
}

func buildDirectStreamingInvocation(group *jen.Group, method *EndpointMethodData) {
	if method.PayloadRef != "" {
		if len(method.Requirements) == 0 {
			group.Id("p").Op(":=").Id("req").Assert(codegen.TypeRef(method.PayloadRef))
		}
		if method.ResultRef != "" {
			group.Return(jen.Id("s").Dot(method.VarName).Call(jen.Id("ctx"), jen.Id("p")))
			return
		}
		group.Return(jen.Nil(), jen.Id("s").Dot(method.VarName).Call(jen.Id("ctx"), jen.Id("p")))
		return
	}
	if method.ResultRef != "" {
		group.Return(jen.Id("s").Dot(method.VarName).Call(jen.Id("ctx")))
		return
	}
	group.Return(jen.Nil(), jen.Id("s").Dot(method.VarName).Call(jen.Id("ctx")))
}

func buildSkipRequestEndpointInvocation(group *jen.Group, method *EndpointMethodData) {
	if method.SkipResponseBodyEncodeDecode {
		buildSkipRequestSkipResponseEndpointInvocation(group, method)
		return
	}
	if method.ViewedResult != nil {
		buildSkipRequestViewedResultEndpointInvocation(group, method)
		return
	}
	returnExprs := []jen.Code{}
	if method.ResultRef == "" {
		returnExprs = append(returnExprs, jen.Nil())
	}
	returnExprs = append(returnExprs, jen.Id("s").Dot(method.VarName).CallFunc(func(args *jen.Group) {
		args.Id("ctx")
		if method.PayloadRef != "" {
			args.Id("ep").Dot("Payload")
		}
		args.Id("ep").Dot("Body")
	}))
	group.Return(returnExprs...)
}

func buildSkipRequestSkipResponseEndpointInvocation(group *jen.Group, method *EndpointMethodData) {
	lhs := []jen.Code{}
	if method.ResultRef != "" {
		lhs = append(lhs, jen.Id("res"))
	}
	lhs = append(lhs, jen.Id("body"), jen.Id("err"))
	group.List(lhs...).Op(":=").Id("s").Dot(method.VarName).CallFunc(func(args *jen.Group) {
		args.Id("ctx")
		if method.PayloadRef != "" {
			args.Id("ep").Dot("Payload")
		}
		args.Id("ep").Dot("Body")
	})
	group.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Id("err")),
	)
	group.Return(
		jen.Op("&").Id(method.ResponseStruct).ValuesFunc(func(values *jen.Group) {
			if method.ResultRef != "" {
				values.Id("Result").Op(":").Id("res")
			}
			values.Id("Body").Op(":").Id("body")
		}),
		jen.Nil(),
	)
}

func buildSkipRequestViewedResultEndpointInvocation(group *jen.Group, method *EndpointMethodData) {
	lhs := []jen.Code{jen.Id("res")}
	if method.ViewedResult.ViewName == "" {
		lhs = append(lhs, jen.Id("view"))
	}
	lhs = append(lhs, jen.Id("err"))
	group.List(lhs...).Op(":=").Id("s").Dot(method.VarName).CallFunc(func(args *jen.Group) {
		args.Id("ctx")
		if method.PayloadRef != "" {
			args.Id("ep").Dot("Payload")
		}
		args.Id("ep").Dot("Body")
	})
	group.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Id("err")),
	)
	viewExpr := jen.Lit(method.ViewedResult.ViewName)
	if method.ViewedResult.ViewName == "" {
		viewExpr = jen.Id("view")
	}
	group.List(jen.Id("vres"), jen.Id("err")).Op(":=").Id(method.ViewedResult.Init.Name).Call(jen.Id("res"), viewExpr)
	group.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Id("err")),
	)
	group.Return(jen.Id("vres"), jen.Nil())
}

func buildViewedResultEndpointInvocation(group *jen.Group, method *EndpointMethodData, payload string) {
	lhs := []jen.Code{jen.Id("res")}
	if method.ViewedResult.ViewName == "" {
		lhs = append(lhs, jen.Id("view"))
	}
	lhs = append(lhs, jen.Id("err"))
	group.List(lhs...).Op(":=").Id("s").Dot(method.VarName).CallFunc(func(args *jen.Group) {
		args.Id("ctx")
		if method.PayloadRef != "" {
			args.Add(codegen.Expr(payload))
		}
	})
	group.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Id("err")),
	)
	viewExpr := jen.Lit(method.ViewedResult.ViewName)
	if method.ViewedResult.ViewName == "" {
		viewExpr = jen.Id("view")
	}
	group.List(jen.Id("vres"), jen.Id("err")).Op(":=").Id(method.ViewedResult.Init.Name).Call(jen.Id("res"), viewExpr)
	group.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Id("err")),
	)
	group.Return(jen.Id("vres"), jen.Nil())
}

func buildSkipResponseEndpointInvocation(group *jen.Group, method *EndpointMethodData, payload string) {
	lhs := []jen.Code{}
	if method.ResultRef != "" {
		lhs = append(lhs, jen.Id("res"))
	}
	lhs = append(lhs, jen.Id("body"), jen.Id("err"))
	group.List(lhs...).Op(":=").Id("s").Dot(method.VarName).CallFunc(func(args *jen.Group) {
		args.Id("ctx")
		if method.PayloadRef != "" {
			args.Add(codegen.Expr(payload))
		}
	})
	group.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Return(jen.Nil(), jen.Id("err")),
	)
	group.Return(
		jen.Op("&").Id(method.ResponseStruct).ValuesFunc(func(values *jen.Group) {
			if method.ResultRef != "" {
				values.Id("Result").Op(":").Id("res")
			}
			values.Id("Body").Op(":").Id("body")
		}),
		jen.Nil(),
	)
}

func buildDefaultEndpointInvocation(group *jen.Group, method *EndpointMethodData, payload string) {
	returnExprs := []jen.Code{}
	if method.ResultRef == "" {
		returnExprs = append(returnExprs, jen.Nil())
	}
	returnExprs = append(returnExprs, jen.Id("s").Dot(method.VarName).CallFunc(func(args *jen.Group) {
		args.Id("ctx")
		if method.PayloadRef != "" {
			args.Add(codegen.Expr(payload))
		}
	}))
	group.Return(returnExprs...)
}
