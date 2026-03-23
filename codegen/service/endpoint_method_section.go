package service

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
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
	b.WriteString(") loom.Endpoint {\n")
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
			renderSchemeAuth(&b, req, scheme, payload)
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

func renderSchemeAuth(b *strings.Builder, req *RequirementData, scheme *SchemeData, payload string) {
	switch scheme.Type {
	case "Basic":
		renderBasicSchemeAuth(b, req, scheme, payload)
	case "APIKey":
		renderCredentialSchemeAuth(b, req, scheme, payload, "APIKey", "key")
	case "JWT":
		renderCredentialSchemeAuth(b, req, scheme, payload, "JWT", "token")
	case "OAuth2":
		renderOAuth2SchemeAuth(b, req, scheme, payload)
	}
}

func renderBasicSchemeAuth(b *strings.Builder, req *RequirementData, scheme *SchemeData, payload string) {
	renderSchemeHeader(b, "BasicScheme", scheme.SchemeName, scheme.Scopes, req.Scopes)
	renderPointerStringBinding(b, "user", payload, scheme.UsernameField, scheme.UsernamePointer)
	renderPointerStringBinding(b, "pass", payload, scheme.PasswordField, scheme.PasswordPointer)
	userExpr := payload + "." + scheme.UsernameField
	passExpr := payload + "." + scheme.PasswordField
	if scheme.UsernamePointer {
		userExpr = "user"
	}
	if scheme.PasswordPointer {
		passExpr = "pass"
	}
	fmt.Fprintf(b, "\t\t\t\tctx, err = auth%sFn(ctx, %s, %s, &sc)\n", scheme.Type, userExpr, passExpr)
}

func renderCredentialSchemeAuth(b *strings.Builder, req *RequirementData, scheme *SchemeData, payload, schemeStruct, tempVar string) {
	renderSchemeHeader(b, schemeStruct+"Scheme", scheme.SchemeName, scheme.Scopes, req.Scopes)
	renderPointerStringBinding(b, tempVar, payload, scheme.CredField, scheme.CredPointer)
	expr := payload + "." + scheme.CredField
	if scheme.CredPointer {
		expr = tempVar
	}
	fmt.Fprintf(b, "\t\t\t\tctx, err = auth%sFn(ctx, %s, &sc)\n", scheme.Type, expr)
}

func renderOAuth2SchemeAuth(b *strings.Builder, req *RequirementData, scheme *SchemeData, payload string) {
	renderSchemeHeaderStart(b, "OAuth2Scheme", scheme.SchemeName, scheme.Scopes, req.Scopes)
	renderOAuth2Flows(b, scheme)
	b.WriteString("\t\t\t\t}\n")
	renderPointerStringBinding(b, "token", payload, scheme.CredField, scheme.CredPointer)
	expr := payload + "." + scheme.CredField
	if scheme.CredPointer {
		expr = "token"
	}
	fmt.Fprintf(b, "\t\t\t\tctx, err = auth%sFn(ctx, %s, &sc)\n", scheme.Type, expr)
}

func renderSchemeHeader(b *strings.Builder, schemeType, schemeName string, scopes, requiredScopes []string) {
	renderSchemeHeaderStart(b, schemeType, schemeName, scopes, requiredScopes)
	b.WriteString("\t\t\t\t}\n")
}

func renderSchemeHeaderStart(b *strings.Builder, schemeType, schemeName string, scopes, requiredScopes []string) {
	fmt.Fprintf(b, "\t\t\t\tsc := security.%s{\n", schemeType)
	fmt.Fprintf(b, "\t\t\t\t\tName: %q,\n", schemeName)
	renderScopeSlice(b, "\t\t\t\t\tScopes", scopes)
	renderScopeSlice(b, "\t\t\t\t\tRequiredScopes", requiredScopes)
}

func renderScopeSlice(b *strings.Builder, field string, scopes []string) {
	fmt.Fprintf(b, "%s: []string{", field)
	for _, scope := range scopes {
		fmt.Fprintf(b, " %q,", scope)
	}
	b.WriteString(" },\n")
}

func renderPointerStringBinding(b *strings.Builder, tempVar, payload, field string, isPointer bool) {
	if !isPointer {
		return
	}
	fmt.Fprintf(b, "\t\t\t\tvar %s string\n", tempVar)
	fmt.Fprintf(b, "\t\t\t\tif %s.%s != nil {\n", payload, field)
	fmt.Fprintf(b, "\t\t\t\t\t%s = *%s.%s\n", tempVar, payload, field)
	b.WriteString("\t\t\t\t}\n")
}

func renderOAuth2Flows(b *strings.Builder, scheme *SchemeData) {
	if len(scheme.Flows) == 0 {
		return
	}
	b.WriteString("\t\t\t\t\tFlows: []*security.OAuthFlow{\n")
	for _, flow := range scheme.Flows {
		b.WriteString("\t\t\t\t\t\t&security.OAuthFlow{\n")
		fmt.Fprintf(b, "\t\t\t\t\t\t\tType: %q,\n", flow.Type())
		if flow.AuthorizationURL != "" {
			fmt.Fprintf(b, "\t\t\t\t\t\t\tAuthorizationURL: %q,\n", flow.AuthorizationURL)
		}
		if flow.TokenURL != "" {
			fmt.Fprintf(b, "\t\t\t\t\t\t\tTokenURL: %q,\n", flow.TokenURL)
		}
		if flow.RefreshURL != "" {
			fmt.Fprintf(b, "\t\t\t\t\t\t\tRefreshURL: %q,\n", flow.RefreshURL)
		}
		b.WriteString("\t\t\t\t\t\t},\n")
	}
	b.WriteString("\t\t\t\t\t},\n")
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
