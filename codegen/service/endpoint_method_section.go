package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/v3/codegen"
)

func endpointMethodSection(method *EndpointMethodData) codegen.Section {
	return codegen.NewRawSection("endpoint-method", renderEndpointMethod(method))
}

func renderEndpointMethod(method *EndpointMethodData) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("New%sEndpoint returns an endpoint function that calls the method %q of service %q.", method.VarName, method.Name, method.ServiceName)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func New%sEndpoint(s %s", method.VarName, method.ServiceVarName)
	for _, scheme := range method.Schemes.DedupeByType() {
		fmt.Fprintf(&b, ", auth%sFn security.Auth%sFunc", scheme.Type, scheme.Type)
	}
	b.WriteString(") goa.Endpoint {\n")
	b.WriteString("\treturn func(ctx context.Context, req any) (any, error) {\n")

	switch {
	case method.ServerStream != nil:
		if method.ServerStream.EndpointStruct != "" {
			fmt.Fprintf(&b, "\t\tep := req.(*%s)\n", method.ServerStream.EndpointStruct)
		}
	case method.SkipRequestBodyEncodeDecode:
		fmt.Fprintf(&b, "\t\tep := req.(*%s)\n", method.RequestStruct)
	case method.PayloadRef != "":
		fmt.Fprintf(&b, "\t\tp := req.(%s)\n", method.PayloadRef)
	}

	payload := payloadVar(method)
	if len(method.Requirements) > 0 {
		b.WriteString(renderEndpointAuth(method, payload))
	}

	switch {
	case method.ServerStream != nil:
		b.WriteString(renderStreamingEndpointInvocation(method, payload))
	case method.SkipRequestBodyEncodeDecode:
		b.WriteString(renderSkipRequestEndpointInvocation(method))
	case method.ViewedResult != nil:
		b.WriteString(renderViewedResultEndpointInvocation(method, payload))
	case method.SkipResponseBodyEncodeDecode:
		b.WriteString(renderSkipResponseEndpointInvocation(method, payload))
	default:
		b.WriteString(renderDefaultEndpointInvocation(method, payload))
	}

	b.WriteString("\t}\n}\n")
	return b.String()
}

func renderEndpointAuth(method *EndpointMethodData, payload string) string {
	var b strings.Builder
	b.WriteString("\t\tvar err error\n")
	for ridx, req := range method.Requirements {
		if ridx != 0 {
			b.WriteString("\t\tif err != nil {\n")
		}
		for sidx, scheme := range req.Schemes {
			if sidx != 0 {
				b.WriteString("\t\t\tif err == nil {\n")
			}
			switch scheme.Type {
			case "Basic":
				b.WriteString("\t\t\t\tsc := security.BasicScheme{\n")
				fmt.Fprintf(&b, "\t\t\t\t\tName: %q,\n", scheme.SchemeName)
				b.WriteString("\t\t\t\t\tScopes: []string{")
				for _, scope := range scheme.Scopes {
					fmt.Fprintf(&b, " %q,", scope)
				}
				b.WriteString(" },\n")
				b.WriteString("\t\t\t\t\tRequiredScopes: []string{")
				for _, scope := range req.Scopes {
					fmt.Fprintf(&b, " %q,", scope)
				}
				b.WriteString(" },\n\t\t\t\t}\n")
				if scheme.UsernamePointer {
					b.WriteString("\t\t\t\tvar user string\n")
					fmt.Fprintf(&b, "\t\t\t\tif %s.%s != nil {\n\t\t\t\t\tuser = *%s.%s\n\t\t\t\t}\n", payload, scheme.UsernameField, payload, scheme.UsernameField)
				}
				if scheme.PasswordPointer {
					b.WriteString("\t\t\t\tvar pass string\n")
					fmt.Fprintf(&b, "\t\t\t\tif %s.%s != nil {\n\t\t\t\t\tpass = *%s.%s\n\t\t\t\t}\n", payload, scheme.PasswordField, payload, scheme.PasswordField)
				}
				userExpr := payload + "." + scheme.UsernameField
				passExpr := payload + "." + scheme.PasswordField
				if scheme.UsernamePointer {
					userExpr = "user"
				}
				if scheme.PasswordPointer {
					passExpr = "pass"
				}
				fmt.Fprintf(&b, "\t\t\t\tctx, err = auth%sFn(ctx, %s, %s, &sc)\n", scheme.Type, userExpr, passExpr)
			case "APIKey":
				b.WriteString("\t\t\t\tsc := security.APIKeyScheme{\n")
				fmt.Fprintf(&b, "\t\t\t\t\tName: %q,\n", scheme.SchemeName)
				b.WriteString("\t\t\t\t\tScopes: []string{")
				for _, scope := range scheme.Scopes {
					fmt.Fprintf(&b, " %q,", scope)
				}
				b.WriteString(" },\n")
				b.WriteString("\t\t\t\t\tRequiredScopes: []string{")
				for _, scope := range req.Scopes {
					fmt.Fprintf(&b, " %q,", scope)
				}
				b.WriteString(" },\n\t\t\t\t}\n")
				expr := payload + "." + scheme.CredField
				if scheme.CredPointer {
					b.WriteString("\t\t\t\tvar key string\n")
					fmt.Fprintf(&b, "\t\t\t\tif %s.%s != nil {\n\t\t\t\t\tkey = *%s.%s\n\t\t\t\t}\n", payload, scheme.CredField, payload, scheme.CredField)
					expr = "key"
				}
				fmt.Fprintf(&b, "\t\t\t\tctx, err = auth%sFn(ctx, %s, &sc)\n", scheme.Type, expr)
			case "JWT":
				b.WriteString("\t\t\t\tsc := security.JWTScheme{\n")
				fmt.Fprintf(&b, "\t\t\t\t\tName: %q,\n", scheme.SchemeName)
				b.WriteString("\t\t\t\t\tScopes: []string{")
				for _, scope := range scheme.Scopes {
					fmt.Fprintf(&b, " %q,", scope)
				}
				b.WriteString(" },\n")
				b.WriteString("\t\t\t\t\tRequiredScopes: []string{")
				for _, scope := range req.Scopes {
					fmt.Fprintf(&b, " %q,", scope)
				}
				b.WriteString(" },\n\t\t\t\t}\n")
				expr := payload + "." + scheme.CredField
				if scheme.CredPointer {
					b.WriteString("\t\t\t\tvar token string\n")
					fmt.Fprintf(&b, "\t\t\t\tif %s.%s != nil {\n\t\t\t\t\ttoken = *%s.%s\n\t\t\t\t}\n", payload, scheme.CredField, payload, scheme.CredField)
					expr = "token"
				}
				fmt.Fprintf(&b, "\t\t\t\tctx, err = auth%sFn(ctx, %s, &sc)\n", scheme.Type, expr)
			case "OAuth2":
				b.WriteString("\t\t\t\tsc := security.OAuth2Scheme{\n")
				fmt.Fprintf(&b, "\t\t\t\t\tName: %q,\n", scheme.SchemeName)
				b.WriteString("\t\t\t\t\tScopes: []string{")
				for _, scope := range scheme.Scopes {
					fmt.Fprintf(&b, " %q,", scope)
				}
				b.WriteString(" },\n")
				b.WriteString("\t\t\t\t\tRequiredScopes: []string{")
				for _, scope := range req.Scopes {
					fmt.Fprintf(&b, " %q,", scope)
				}
				b.WriteString(" },\n")
				if len(scheme.Flows) > 0 {
					b.WriteString("\t\t\t\t\tFlows: []*security.OAuthFlow{\n")
					for _, flow := range scheme.Flows {
						b.WriteString("\t\t\t\t\t\t&security.OAuthFlow{\n")
						fmt.Fprintf(&b, "\t\t\t\t\t\t\tType: %q,\n", flow.Type())
						if flow.AuthorizationURL != "" {
							fmt.Fprintf(&b, "\t\t\t\t\t\t\tAuthorizationURL: %q,\n", flow.AuthorizationURL)
						}
						if flow.TokenURL != "" {
							fmt.Fprintf(&b, "\t\t\t\t\t\t\tTokenURL: %q,\n", flow.TokenURL)
						}
						if flow.RefreshURL != "" {
							fmt.Fprintf(&b, "\t\t\t\t\t\t\tRefreshURL: %q,\n", flow.RefreshURL)
						}
						b.WriteString("\t\t\t\t\t\t},\n")
					}
					b.WriteString("\t\t\t\t\t},\n")
				}
				b.WriteString("\t\t\t\t}\n")
				expr := payload + "." + scheme.CredField
				if scheme.CredPointer {
					b.WriteString("\t\t\t\tvar token string\n")
					fmt.Fprintf(&b, "\t\t\t\tif %s.%s != nil {\n\t\t\t\t\ttoken = *%s.%s\n\t\t\t\t}\n", payload, scheme.CredField, payload, scheme.CredField)
					expr = "token"
				}
				fmt.Fprintf(&b, "\t\t\t\tctx, err = auth%sFn(ctx, %s, &sc)\n", scheme.Type, expr)
			}
			if sidx != 0 {
				b.WriteString("\t\t\t}\n")
			}
		}
		if ridx != 0 {
			b.WriteString("\t\t}\n")
		}
	}
	b.WriteString("\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
	return b.String()
}

func renderStreamingEndpointInvocation(method *EndpointMethodData, payload string) string {
	var b strings.Builder
	if method.ServerStream.EndpointStruct != "" {
		if method.HasMixedResults {
			b.WriteString("\t\t")
			if method.ResultRef != "" {
				b.WriteString("res, ")
			}
			if method.ViewedResult != nil && method.ViewedResult.ViewName == "" {
				b.WriteString("view, ")
			}
			b.WriteString("err := s." + method.VarName + "(ctx")
			if method.PayloadRef != "" {
				b.WriteString(", " + payload)
			}
			b.WriteString(", ep.Stream)\n")
			b.WriteString("\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
			if method.ViewedResult != nil {
				viewExpr := fmt.Sprintf("%q", method.ViewedResult.ViewName)
				if method.ViewedResult.ViewName == "" {
					viewExpr = "view"
				}
				fmt.Fprintf(&b, "\t\tvres := %s(res, %s)\n\t\treturn vres, nil\n", method.ViewedResult.Init.Name, viewExpr)
			} else {
				b.WriteString("\t\treturn res, nil\n")
			}
			return b.String()
		}
		b.WriteString("\t\treturn nil, s." + method.VarName + "(ctx")
		if method.PayloadRef != "" {
			b.WriteString(", " + payload)
		}
		b.WriteString(", ep.Stream)\n")
		return b.String()
	}
	if method.PayloadRef != "" {
		fmt.Fprintf(&b, "\t\tp := req.(%s)\n", method.PayloadRef)
		if method.ResultRef != "" {
			fmt.Fprintf(&b, "\t\treturn s.%s(ctx, p)\n", method.VarName)
		} else {
			fmt.Fprintf(&b, "\t\treturn nil, s.%s(ctx, p)\n", method.VarName)
		}
		return b.String()
	}
	if method.ResultRef != "" {
		fmt.Fprintf(&b, "\t\treturn s.%s(ctx)\n", method.VarName)
	} else {
		fmt.Fprintf(&b, "\t\treturn nil, s.%s(ctx)\n", method.VarName)
	}
	return b.String()
}

func renderSkipRequestEndpointInvocation(method *EndpointMethodData) string {
	var b strings.Builder
	if method.SkipResponseBodyEncodeDecode {
		b.WriteString("\t\t")
		if method.ResultRef != "" {
			b.WriteString("res, ")
		}
		b.WriteString("body, err := s." + method.VarName + "(ctx")
		if method.PayloadRef != "" {
			b.WriteString(", ep.Payload")
		}
		b.WriteString(", ep.Body)\n")
		b.WriteString("\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
		fmt.Fprintf(&b, "\t\treturn &%s{ ", method.ResponseStruct)
		if method.ResultRef != "" {
			b.WriteString("Result: res, ")
		}
		b.WriteString("Body: body }, nil\n")
		return b.String()
	}
	if method.ViewedResult != nil {
		b.WriteString("\t\tres, ")
		if method.ViewedResult.ViewName == "" {
			b.WriteString("view, ")
		}
		b.WriteString("err := s." + method.VarName + "(ctx")
		if method.PayloadRef != "" {
			b.WriteString(", ep.Payload")
		}
		b.WriteString(", ep.Body)\n")
		b.WriteString("\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
		viewExpr := fmt.Sprintf("%q", method.ViewedResult.ViewName)
		if method.ViewedResult.ViewName == "" {
			viewExpr = "view"
		}
		fmt.Fprintf(&b, "\t\tvres := %s(res, %s)\n\t\treturn vres, nil\n", method.ViewedResult.Init.Name, viewExpr)
		return b.String()
	}
	b.WriteString("\t\treturn ")
	if method.ResultRef == "" {
		b.WriteString("nil, ")
	}
	b.WriteString("s." + method.VarName + "(ctx")
	if method.PayloadRef != "" {
		b.WriteString(", ep.Payload")
	}
	b.WriteString(", ep.Body)\n")
	return b.String()
}

func renderViewedResultEndpointInvocation(method *EndpointMethodData, payload string) string {
	var b strings.Builder
	b.WriteString("\t\tres, ")
	if method.ViewedResult.ViewName == "" {
		b.WriteString("view, ")
	}
	b.WriteString("err := s." + method.VarName + "(ctx")
	if method.PayloadRef != "" {
		b.WriteString(", " + payload)
	}
	b.WriteString(")\n")
	b.WriteString("\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
	viewExpr := fmt.Sprintf("%q", method.ViewedResult.ViewName)
	if method.ViewedResult.ViewName == "" {
		viewExpr = "view"
	}
	fmt.Fprintf(&b, "\t\tvres := %s(res, %s)\n\t\treturn vres, nil\n", method.ViewedResult.Init.Name, viewExpr)
	return b.String()
}

func renderSkipResponseEndpointInvocation(method *EndpointMethodData, payload string) string {
	var b strings.Builder
	b.WriteString("\t\t")
	if method.ResultRef != "" {
		b.WriteString("res, ")
	}
	b.WriteString("body, err := s." + method.VarName + "(ctx")
	if method.PayloadRef != "" {
		b.WriteString(", " + payload)
	}
	b.WriteString(")\n")
	b.WriteString("\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n")
	fmt.Fprintf(&b, "\t\treturn &%s{ ", method.ResponseStruct)
	if method.ResultRef != "" {
		b.WriteString("Result: res, ")
	}
	b.WriteString("Body: body }, nil\n")
	return b.String()
}

func renderDefaultEndpointInvocation(method *EndpointMethodData, payload string) string {
	var b strings.Builder
	b.WriteString("\t\treturn ")
	if method.ResultRef == "" {
		b.WriteString("nil, ")
	}
	b.WriteString("s." + method.VarName + "(ctx")
	if method.PayloadRef != "" {
		b.WriteString(", " + payload)
	}
	b.WriteString(")\n")
	return b.String()
}
