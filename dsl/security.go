package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

const (
	authUnauthorizedErrorName = "unauthorized"
	authForbiddenErrorName    = "forbidden"
)

// BasicAuthSecurity defines a basic authentication security scheme.
//
// BasicAuthSecurity is a top level DSL.
//
// BasicAuthSecurity takes a name as first argument and an optional DSL as
// second argument.
//
// Example:
//
//	var Basic = BasicAuthSecurity("basicauth", func() {
//	    Description("Use your own password!")
//	})
func BasicAuthSecurity(name string, fn ...func()) *expr.SchemeExpr {
	if _, ok := eval.Current().(eval.TopExpr); !ok {
		eval.IncompatibleDSL()
		return nil
	}

	if securitySchemeRedefined(name) {
		return nil
	}

	e := &expr.SchemeExpr{
		Kind:       expr.BasicAuthKind,
		SchemeName: name,
	}

	if len(fn) != 0 {
		if !eval.Execute(fn[0], e) {
			return nil
		}
	}

	expr.Root.Schemes = append(expr.Root.Schemes, e)

	return e
}

// APIKeySecurity defines an API key security scheme where a key must be
// provided by the client to perform authorization.
//
// APIKeySecurity is a top level DSL.
//
// APIKeySecurity takes a name as first argument and an optional DSL as
// second argument.
//
// Example:
//
//	var APIKey = APIKeySecurity("key", func() {
//	      Description("Shared secret")
//	})
func APIKeySecurity(name string, fn ...func()) *expr.SchemeExpr {
	if _, ok := eval.Current().(eval.TopExpr); !ok {
		eval.IncompatibleDSL()
		return nil
	}

	if securitySchemeRedefined(name) {
		return nil
	}

	e := &expr.SchemeExpr{
		Kind:       expr.APIKeyKind,
		SchemeName: name,
	}

	if len(fn) != 0 {
		if !eval.Execute(fn[0], e) {
			return nil
		}
	}

	expr.Root.Schemes = append(expr.Root.Schemes, e)

	return e
}

// OAuth2Security defines an OAuth2 security scheme. The DSL provided as second
// argument defines the specific flows supported by the scheme. The supported
// flow types are ImplicitFlow, PasswordFlow, ClientCredentialsFlow, and
// AuthorizationCodeFlow. The DSL also defines the scopes that may be
// associated with the incoming request tokens.
//
// OAuth2Security is a top level DSL.
//
// OAuth2Security takes a name as first argument and a DSL as second argument.
//
// Example:
//
//	var OAuth2 = OAuth2Security("googauth", func() {
//	    ImplicitFlow("/authorization")
//
//	    Scope("api:write", "Write acess")
//	    Scope("api:read", "Read access")
//	})
func OAuth2Security(name string, fn ...func()) *expr.SchemeExpr {
	if _, ok := eval.Current().(eval.TopExpr); !ok {
		eval.IncompatibleDSL()
		return nil
	}

	if securitySchemeRedefined(name) {
		return nil
	}

	e := &expr.SchemeExpr{
		SchemeName: name,
		Kind:       expr.OAuth2Kind,
	}

	if len(fn) != 0 {
		if !eval.Execute(fn[0], e) {
			return nil
		}
	}

	expr.Root.Schemes = append(expr.Root.Schemes, e)

	return e
}

// JWTSecurity defines an HTTP security scheme where a JWT is passed in the
// request Authorization header as a bearer token to perform auth. This scheme
// supports defining scopes that endpoint may require to authorize the request.
// The scheme also supports specifying a token URL used to retrieve token
// values.
//
// JWTSecurity is a top level DSL.
//
// JWTSecurity takes a name as first argument and an optional DSL as second
// argument.
//
// Example:
//
//	var JWT = JWTSecurity("jwt", func() {
//	    Scope("system:write", "Write to the system")
//	    Scope("system:read", "Read anything in there")
//	})
func JWTSecurity(name string, fn ...func()) *expr.SchemeExpr {
	if _, ok := eval.Current().(eval.TopExpr); !ok {
		eval.IncompatibleDSL()
		return nil
	}

	if securitySchemeRedefined(name) {
		return nil
	}

	e := &expr.SchemeExpr{
		SchemeName: name,
		Kind:       expr.JWTKind,
		In:         "header",
		Name:       "Authorization",
	}

	if len(fn) != 0 {
		if !eval.Execute(fn[0], e) {
			return nil
		}
	}

	expr.Root.Schemes = append(expr.Root.Schemes, e)

	return e
}

// SessionAuth defines a logical auth contract backed by one or more transport
// alternatives.
//
// SessionAuth is a top level DSL.
func SessionAuth(name string, fn ...func()) *expr.SessionAuthExpr {
	if _, ok := eval.Current().(eval.TopExpr); !ok {
		eval.IncompatibleDSL()
		return nil
	}
	if sessionAuthRedefined(name) {
		return nil
	}
	e := &expr.SessionAuthExpr{Name: name}
	if len(fn) != 0 {
		if !eval.Execute(fn[0], e) {
			return nil
		}
	}
	expr.Root.SessionAuths = append(expr.Root.SessionAuths, e)
	return e
}

// BearerTransport defines a bearer transport for a session auth contract.
func BearerTransport(scheme any, fieldName string, fn ...func()) {
	sessionTransport(expr.SessionBearerTransportKind, scheme, fieldName, fn...)
}

// CookieTransport defines a cookie transport for a session auth contract.
//
// An empty field name makes the cookie transport transport-owned: Loom still
// infers HTTP cookie decoding and OpenAPI security, but does not inject a
// payload credential field.
func CookieTransport(scheme any, fieldName string, fn ...func()) {
	sessionTransport(expr.SessionCookieTransportKind, scheme, fieldName, fn...)
}

// CookieName sets the inferred HTTP cookie name of a session cookie transport.
func CookieName(name string) {
	current, ok := eval.Current().(*expr.SessionTransportExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if current.Kind != expr.SessionCookieTransportKind {
		eval.ReportError("CookieName must be used with a session cookie transport")
		return
	}
	if name == "" {
		eval.ReportError("cookie name cannot be empty")
		return
	}
	current.HTTPName = name
}

// Security defines authentication requirements to access an entire API, service
// or individual service method.
//
// The requirement refers to one or more OAuth2Security, BasicAuthSecurity,
// APIKeySecurity or JWTSecurity security scheme. If the schemes include a
// OAuth2Security or JWTSecurity scheme then required scopes may be listed by
// name in the Security DSL. All the listed schemes must be validated by the
// client for the request to be authorized. Security may appear multiple times
// in the same scope in which case the client may validate any one of the
// requirements for the request to be authorized.
//
// Security must appear in an API, Service or Method expression.
//
// Security accepts an arbitrary number of security schemes as argument
// specified by name or by reference and an optional DSL function as last
// argument.
//
// Examples:
//
//	var _ = Service("calculator", func() {
//	    // Override default API security requirements. Accept either basic
//	    // auth or OAuth2 access token with "api:read" scope.
//	    Security(BasicAuth)
//	    Security("oauth2", func() {
//	        Scope("api:read")
//	    })
//
//	    Method("add", func() {
//	        Description("Add two operands")
//
//	        // Override default service security requirements. Require
//	        // both basic auth and OAuth2 access token with "api:write"
//	        // scope.
//	        Security(BasicAuth, "oauth2", func() {
//	            Scope("api:write")
//	        })
//
//	        Payload(Operands)
//	        Error(ErrBadRequest, ErrorResult)
//	    })
//
//	    Method("health-check", func() {
//	        Description("Check health")
//
//	        // Remove need for authorization for this endpoint.
//	        NoSecurity()
//
//	        Payload(Operands)
//	        Error(ErrBadRequest, ErrorResult)
//	    })
//	})
func Security(args ...any) {
	var dsl func()
	if d, ok := args[len(args)-1].(func()); ok {
		args = args[:len(args)-1]
		dsl = d
	}

	schemes := make([]*expr.SchemeExpr, len(args))
	for i, arg := range args {
		switch val := arg.(type) {
		case string:
			for _, s := range expr.Root.Schemes {
				if s.SchemeName == val {
					schemes[i] = expr.DupScheme(s)
					break
				}
			}
			if schemes[i] == nil {
				eval.ReportError("security scheme %q not found", val)
				return
			}
		case *expr.SchemeExpr:
			if val == nil {
				eval.InvalidArgError("security scheme", val)
				return
			}
			schemes[i] = expr.DupScheme(val)
		default:
			eval.InvalidArgError("security scheme or security scheme name", val)
			return
		}
	}

	security := &expr.SecurityExpr{Schemes: schemes}
	if dsl != nil {
		if !eval.Execute(dsl, security) {
			return
		}
	}

	current := eval.Current()
	switch actual := current.(type) {
	case *expr.MethodExpr:
		actual.Requirements = append(actual.Requirements, security)
	case *expr.ServiceExpr:
		actual.Requirements = append(actual.Requirements, security)
	case *expr.APIExpr:
		actual.Requirements = append(actual.Requirements, security)
	case expr.SecurityHolder:
		actual.AddSecurityRequirement(security)
	default:
		eval.IncompatibleDSL()
		return
	}
}

// SessionSecurity defines authentication requirements using a named
// multi-transport session auth contract.
func SessionSecurity(arg any) {
	sessionAuth := lookupSessionAuth(arg)
	if sessionAuth == nil {
		return
	}
	current := eval.Current()
	switch actual := current.(type) {
	case *expr.MethodExpr:
		actual.SessionAuths = append(actual.SessionAuths, sessionAuth)
	case *expr.ServiceExpr:
		actual.SessionAuths = append(actual.SessionAuths, sessionAuth)
	case *expr.APIExpr:
		actual.SessionAuths = append(actual.SessionAuths, sessionAuth)
	default:
		eval.IncompatibleDSL()
		return
	}
}

// AuthErrorResponses defines standard HTTP auth error responses for secured
// endpoints.
//
// AuthErrorResponses must appear in an API, service or method HTTP expression.
// The helper ensures the corresponding "unauthorized" and "forbidden" errors
// exist in the matching API, service or method scope and adds the HTTP 401 and
// 403 response mappings if they are not already defined.
func AuthErrorResponses() {
	switch actual := eval.Current().(type) {
	case *expr.RootExpr:
		ensureAuthError(actual, authUnauthorizedErrorName)
		ensureAuthError(actual, authForbiddenErrorName)
		ensureHTTPAuthError(&actual.API.HTTP.Errors, actual, authUnauthorizedErrorName, expr.StatusUnauthorized, "Authentication is required.")
		ensureHTTPAuthError(&actual.API.HTTP.Errors, actual, authForbiddenErrorName, expr.StatusForbidden, "Access is forbidden.")
	case *expr.HTTPServiceExpr:
		ensureAuthError(actual.ServiceExpr, authUnauthorizedErrorName)
		ensureAuthError(actual.ServiceExpr, authForbiddenErrorName)
		ensureHTTPAuthError(&actual.HTTPErrors, actual, authUnauthorizedErrorName, expr.StatusUnauthorized, "Authentication is required.")
		ensureHTTPAuthError(&actual.HTTPErrors, actual, authForbiddenErrorName, expr.StatusForbidden, "Access is forbidden.")
	case *expr.HTTPEndpointExpr:
		ensureAuthError(actual.MethodExpr, authUnauthorizedErrorName)
		ensureAuthError(actual.MethodExpr, authForbiddenErrorName)
		ensureHTTPAuthError(&actual.HTTPErrors, actual, authUnauthorizedErrorName, expr.StatusUnauthorized, "Authentication is required.")
		ensureHTTPAuthError(&actual.HTTPErrors, actual, authForbiddenErrorName, expr.StatusForbidden, "Access is forbidden.")
	default:
		eval.IncompatibleDSL()
	}
}

// NoSecurity removes the need for an endpoint to perform authorization.
//
// NoSecurity must appear in Method.
func NoSecurity() {
	security := &expr.SecurityExpr{
		Schemes: []*expr.SchemeExpr{{Kind: expr.NoKind}},
	}

	current := eval.Current()
	switch actual := current.(type) {
	case *expr.MethodExpr:
		if actual.Meta == nil {
			actual.Meta = expr.MetaExpr{}
		}
		actual.Meta["security:no"] = []string{}
		actual.Requirements = append(actual.Requirements, security)
	default:
		eval.IncompatibleDSL()
		return
	}
}

func securitySchemeRedefined(name string) bool {
	for _, s := range expr.Root.Schemes {
		if s.SchemeName == name {
			eval.ReportError("cannot redefine security scheme with name %q", name)
			return true
		}
	}
	return false
}

func sessionAuthRedefined(name string) bool {
	for _, s := range expr.Root.SessionAuths {
		if s.Name == name {
			eval.ReportError("cannot redefine session auth with name %q", name)
			return true
		}
	}
	return false
}

func lookupSessionAuth(arg any) *expr.SessionAuthExpr {
	switch val := arg.(type) {
	case string:
		for _, sessionAuth := range expr.Root.SessionAuths {
			if sessionAuth.Name == val {
				return sessionAuth
			}
		}
		eval.ReportError("session auth %q not found", val)
		return nil
	case *expr.SessionAuthExpr:
		if val == nil {
			eval.InvalidArgError("session auth", val)
			return nil
		}
		return val
	default:
		eval.InvalidArgError("session auth or session auth name", val)
		return nil
	}
}

func lookupSessionTransportScheme(arg any) *expr.SchemeExpr {
	switch val := arg.(type) {
	case string:
		for _, scheme := range expr.Root.Schemes {
			if scheme.SchemeName == val {
				return scheme
			}
		}
		eval.ReportError("security scheme %q not found", val)
		return nil
	case *expr.SchemeExpr:
		if val == nil {
			eval.InvalidArgError("security scheme", val)
			return nil
		}
		return val
	default:
		eval.InvalidArgError("security scheme or security scheme name", val)
		return nil
	}
}

func sessionTransport(kind expr.SessionTransportKind, scheme any, fieldName string, fn ...func()) {
	current, ok := eval.Current().(*expr.SessionAuthExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	sch := lookupSessionTransportScheme(scheme)
	if sch == nil {
		return
	}
	transport := &expr.SessionTransportExpr{
		Kind:      kind,
		Scheme:    expr.DupScheme(sch),
		FieldName: fieldName,
	}
	if len(fn) != 0 {
		if !eval.Execute(fn[0], transport) {
			return
		}
	}
	current.Transports = append(current.Transports, transport)
}

// useDSL modifies the Attribute function to use the given function as DSL,
// merging it with any pre-existing DSL.
func useDSL(args []any, d func()) []any {
	if len(args) == 0 {
		return []any{d}
	}
	ds, ok := args[len(args)-1].(func())
	if ok {
		newdsl := func() { ds(); d() }
		args = append(args[:len(args)-1], newdsl)
	} else {
		args = append(args, d)
	}
	return args
}
