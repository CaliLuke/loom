package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// Scope defines a supported scope in a security scheme, or a default for
// OAuth flows. In an OAuth flow block it defines
// that flow's complete inventory. In Security it lists required scopes.
//
// Scope must appear in Security, a security scheme, or an OAuth flow block.
//
// Scope accepts one or two arguments: the first argument is the scope name and
// when used in a scheme or flow the optional second argument is a description.
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
	if len(desc) > 1 {
		eval.TooManyArgError()
		return
	}
	description := "no description"
	if len(desc) == 1 {
		description = desc[0]
	}
	switch current := eval.Current().(type) {
	case *expr.SecurityExpr:
		if len(desc) > 0 {
			eval.TooManyArgError()
			return
		}
		current.Scopes = append(current.Scopes, name)
	case *expr.SchemeExpr:
		current.Scopes = append(current.Scopes, &expr.ScopeExpr{Name: name, Description: description})
	case *expr.FlowExpr:
		current.Scopes = append(current.Scopes, &expr.ScopeExpr{Name: name, Description: description})
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
// refresh URLs, followed by an optional flow-local Scope block. A supplied
// block replaces scheme-level scope defaults, including when empty.
func AuthorizationCodeFlow(authorizationURL, tokenURL, refreshURL string, fn ...func()) {
	oauthFlow(&expr.FlowExpr{
		Kind:             expr.AuthorizationCodeFlowKind,
		AuthorizationURL: authorizationURL,
		TokenURL:         tokenURL,
		RefreshURL:       refreshURL,
	}, fn)
}

// ImplicitFlow defines an implicit OAuth2 flow as described in section 1.3.2
// of RFC 6749.
//
// ImplicitFlow must be used in OAuth2Security.
//
// ImplicitFlow accepts the authorization and refresh URLs and an optional
// flow-local Scope block. A supplied block replaces scheme-level defaults.
func ImplicitFlow(authorizationURL, refreshURL string, fn ...func()) {
	oauthFlow(&expr.FlowExpr{
		Kind:             expr.ImplicitFlowKind,
		AuthorizationURL: authorizationURL,
		RefreshURL:       refreshURL,
	}, fn)
}

// PasswordFlow defines an Resource Owner Password Credentials OAuth2 flow as
// described in section 1.3.3 of RFC 6749.
//
// PasswordFlow must be used in OAuth2Security.
//
// PasswordFlow accepts the token and refresh URLs and an optional flow-local
// Scope block. A supplied block replaces scheme-level defaults.
func PasswordFlow(tokenURL, refreshURL string, fn ...func()) {
	oauthFlow(&expr.FlowExpr{
		Kind:       expr.PasswordFlowKind,
		TokenURL:   tokenURL,
		RefreshURL: refreshURL,
	}, fn)
}

// ClientCredentialsFlow defines an clientCredentials OAuth2 flow as described
// in section 1.3.4 of RFC 6749.
//
// ClientCredentialsFlow must be used in OAuth2Security.
//
// ClientCredentialsFlow accepts the token and refresh URLs and an optional
// flow-local Scope block. A supplied block replaces scheme-level defaults.
func ClientCredentialsFlow(tokenURL, refreshURL string, fn ...func()) {
	oauthFlow(&expr.FlowExpr{
		Kind:       expr.ClientCredentialsFlowKind,
		TokenURL:   tokenURL,
		RefreshURL: refreshURL,
	}, fn)
}

// DeviceAuthorizationFlow defines an OAuth 2 device authorization flow as
// described by RFC 8628. It must be used in OAuth2Security. An optional Scope
// block supplies the complete flow inventory instead of scheme-level defaults.
func DeviceAuthorizationFlow(deviceAuthorizationURL, tokenURL, refreshURL string, fn ...func()) {
	oauthFlow(&expr.FlowExpr{
		Kind:                   expr.DeviceAuthorizationFlowKind,
		DeviceAuthorizationURL: deviceAuthorizationURL,
		TokenURL:               tokenURL,
		RefreshURL:             refreshURL,
	}, fn)
}

func oauthFlow(flow *expr.FlowExpr, fn []func()) {
	current, ok := eval.Current().(*expr.SchemeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if current.Kind != expr.OAuth2Kind {
		eval.ReportError("cannot specify flow for non-oauth2 security scheme.")
		return
	}
	if len(fn) > 1 {
		eval.TooManyArgError()
		return
	}
	if len(fn) == 1 {
		flow.Scopes = []*expr.ScopeExpr{}
		if fn[0] == nil || !eval.Execute(fn[0], flow) {
			if fn[0] == nil {
				eval.ReportError("OAuth flow scope block must not be nil")
			}
			return
		}
	}
	current.Flows = append(current.Flows, flow)
}
