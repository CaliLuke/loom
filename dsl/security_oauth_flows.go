package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// Scope has two uses: in JWTSecurity or OAuth2Security it defines a scope
// supported by the scheme. In Security it lists required scopes.
//
// Scope must appear in Security, BasicSecurity, APIKeySecurity, JWTSecurity or OAuth2Security.
//
// Scope accepts one or two arguments: the first argument is the scope name and
// when used in JWTSecurity or OAuth2Security the second argument is a
// description.
//
// Example:
//
//	var JWT = JWTSecurity("JWT", func() {
//	    Scope("api:read", "Read access") // Defines a scope
//	    Scope("api:write", "Write access")
//	})
//
//	Method("secured", func() {
//	    Security(JWT, func() {
//	        Scope("api:read") // Required scope for auth
//	    })
//	})
func Scope(name string, desc ...string) {
	switch current := eval.Current().(type) {
	case *expr.SecurityExpr:
		if len(desc) >= 1 {
			eval.TooManyArgError()
			return
		}
		current.Scopes = append(current.Scopes, name)
	case *expr.SchemeExpr:
		if len(desc) > 1 {
			eval.TooManyArgError()
			return
		}
		d := "no description"
		if len(desc) == 1 {
			d = desc[0]
		}
		current.Scopes = append(current.Scopes,
			&expr.ScopeExpr{Name: name, Description: d})
	default:
		eval.IncompatibleDSL()
	}
}

// AuthorizationCodeFlow defines an authorizationCode OAuth2 flow as described
// in section 1.3.1 of RFC 6749.
//
// AuthorizationCodeFlow must be used in OAuth2Security.
//
// AuthorizationCodeFlow accepts three arguments: the authorization, token and
// refresh URLs.
func AuthorizationCodeFlow(authorizationURL, tokenURL, refreshURL string) {
	current, ok := eval.Current().(*expr.SchemeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if current.Kind != expr.OAuth2Kind {
		eval.ReportError("cannot specify flow for non-oauth2 security scheme.")
		return
	}
	current.Flows = append(current.Flows, &expr.FlowExpr{
		Kind:             expr.AuthorizationCodeFlowKind,
		AuthorizationURL: authorizationURL,
		TokenURL:         tokenURL,
		RefreshURL:       refreshURL,
	})
}

// ImplicitFlow defines an implicit OAuth2 flow as described in section 1.3.2
// of RFC 6749.
//
// ImplicitFlow must be used in OAuth2Security.
//
// ImplicitFlow accepts two arguments: the authorization and refresh URLs.
func ImplicitFlow(authorizationURL, refreshURL string) {
	current, ok := eval.Current().(*expr.SchemeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if current.Kind != expr.OAuth2Kind {
		eval.ReportError("cannot specify flow for non-oauth2 security scheme.")
		return
	}
	current.Flows = append(current.Flows, &expr.FlowExpr{
		Kind:             expr.ImplicitFlowKind,
		AuthorizationURL: authorizationURL,
		RefreshURL:       refreshURL,
	})
}

// PasswordFlow defines an Resource Owner Password Credentials OAuth2 flow as
// described in section 1.3.3 of RFC 6749.
//
// PasswordFlow must be used in OAuth2Security.
//
// PasswordFlow accepts two arguments: the token and refresh URLs.
func PasswordFlow(tokenURL, refreshURL string) {
	current, ok := eval.Current().(*expr.SchemeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if current.Kind != expr.OAuth2Kind {
		eval.ReportError("cannot specify flow for non-oauth2 security scheme.")
		return
	}
	current.Flows = append(current.Flows, &expr.FlowExpr{
		Kind:       expr.PasswordFlowKind,
		TokenURL:   tokenURL,
		RefreshURL: refreshURL,
	})
}

// ClientCredentialsFlow defines an clientCredentials OAuth2 flow as described
// in section 1.3.4 of RFC 6749.
//
// ClientCredentialsFlow must be used in OAuth2Security.
//
// ClientCredentialsFlow accepts two arguments: the token and refresh URLs.
func ClientCredentialsFlow(tokenURL, refreshURL string) {
	current, ok := eval.Current().(*expr.SchemeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if current.Kind != expr.OAuth2Kind {
		eval.ReportError("cannot specify flow for non-oauth2 security scheme.")
		return
	}
	current.Flows = append(current.Flows, &expr.FlowExpr{
		Kind:       expr.ClientCredentialsFlowKind,
		TokenURL:   tokenURL,
		RefreshURL: refreshURL,
	})
}

// DeviceAuthorizationFlow defines an OAuth 2 device authorization flow as
// described by RFC 8628. It must be used in OAuth2Security.
func DeviceAuthorizationFlow(deviceAuthorizationURL, tokenURL, refreshURL string) {
	current, ok := eval.Current().(*expr.SchemeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if current.Kind != expr.OAuth2Kind {
		eval.ReportError("cannot specify flow for non-oauth2 security scheme.")
		return
	}
	current.Flows = append(current.Flows, &expr.FlowExpr{
		Kind:                   expr.DeviceAuthorizationFlowKind,
		DeviceAuthorizationURL: deviceAuthorizationURL,
		TokenURL:               tokenURL,
		RefreshURL:             refreshURL,
	})
}
