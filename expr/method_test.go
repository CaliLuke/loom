package expr_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestMethodExprValidate(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Error string
	}{
		{"valid-security-schemes-extend", testdata.ValidSecuritySchemesExtendDSL, ""},
		{"invalid-security-schemes", testdata.InvalidSecuritySchemesDSL,
			`service "InvalidSecuritySchemesService" method "SecureMethod": payload of method "SecureMethod" of service "InvalidSecuritySchemesService" does not define a username attribute, use Username to define one
service "InvalidSecuritySchemesService" method "SecureMethod": payload of method "SecureMethod" of service "InvalidSecuritySchemesService" does not define a password attribute, use Password to define one
service "InvalidSecuritySchemesService" method "SecureMethod": payload of method "SecureMethod" of service "InvalidSecuritySchemesService" does not define a JWT attribute, use Token to define one
service "InvalidSecuritySchemesService" method "SecureMethod": security scope "not:found" not found in any of the security schemes.
flow authorization_code: invalid token URL "^example:/token<>": parse "^example:/token<>": first path segment in URL cannot contain colon
flow authorization_code: invalid authorization URL "http://^authorization": parse "http://^authorization": invalid character "^" in host name
flow authorization_code: invalid refresh URL "http://refresh^": parse "http://refresh^": invalid character "^" in host name
service "InvalidSecuritySchemesService" method "InheritedSecureMethod": payload of method "InheritedSecureMethod" of service "InvalidSecuritySchemesService" does not define a OAuth2 access token attribute, use AccessToken to define one
service "InvalidSecuritySchemesService" method "InheritedSecureMethod": payload of method "InheritedSecureMethod" of service "InvalidSecuritySchemesService" does not define an API key attribute, use APIKey to define one
service "InvalidSecuritySchemesService" method "InheritedSecureMethod": security scope "not:found" not found in any of the security schemes.
service "AnotherInvalidSecuritySchemesService" method "Method": payload of method "Method" of service "AnotherInvalidSecuritySchemesService" defines a username attribute, but no basic auth security scheme exist
service "AnotherInvalidSecuritySchemesService" method "Method": payload of method "Method" of service "AnotherInvalidSecuritySchemesService" defines a password attribute, but no basic auth security scheme exist
service "AnotherInvalidSecuritySchemesService" method "Method": payload of method "Method" of service "AnotherInvalidSecuritySchemesService" defines an API key attribute, but no APIKey security scheme exist
service "AnotherInvalidSecuritySchemesService" method "Method": payload of method "Method" of service "AnotherInvalidSecuritySchemesService" defines a JWT token attribute, but no JWT auth security scheme exist
service "AnotherInvalidSecuritySchemesService" method "Method": payload of method "Method" of service "AnotherInvalidSecuritySchemesService" defines a OAuth2 access token attribute, but no OAuth2 security scheme exist`,
		},
		{"union-branch-security-attribute", testdata.UnionBranchSecurityAttributeDSL,
			`service "UnionBranchSecurityService" method "SecureMethod": payload of method "SecureMethod" of service "UnionBranchSecurityService" does not define a JWT attribute, use Token to define one`,
		},
		{"constructor-union-result-view", testdata.ConstructorUnionResultViewDSL,
			`service "ConstructorUnionResultViewService" method "Show": result -  uses view "tiny" but "AOrB" is not a result type`,
		},
		{"valid-session-security", testdata.ValidSessionSecurityDSL, ""},
		{"invalid-session-security-transport", testdata.InvalidSessionSecurityTransportDSL,
			`session auth "invalid_session": cookie transport must use an API key security scheme`,
		},
		{"invalid-session-security-duplicate-transport", testdata.InvalidSessionSecurityDuplicateTransportDSL,
			`session auth "duplicate_transport_session": session auth "duplicate_transport_session" defines duplicate bearer transport`,
		},
		{"valid-session-security-auto-payload", testdata.ValidSessionSecurityAutoPayloadDSL, ""},
		{"invalid-session-security-payload-conflict", testdata.InvalidSessionSecurityPayloadConflictDSL,
			`service "InvalidSessionSecurityPayloadConflictService" method "SecureMethod": payload of method "SecureMethod" of service "InvalidSessionSecurityPayloadConflictService" defines field "auth" which conflicts with session auth "conflict_session" bearer transport`,
		},
		{"valid-method-auth-error-responses", testdata.ValidMethodAuthErrorResponsesDSL, ""},
		{"valid-service-auth-error-responses", testdata.ValidServiceAuthErrorResponsesDSL, ""},
		{"valid-api-auth-error-responses", testdata.ValidAPIAuthErrorResponsesDSL, ""},
		{"invalid-auth-error-responses-placement", testdata.InvalidAuthErrorResponsesPlacementDSL,
			`invalid use of AuthErrorResponses`,
		},
		{"valid-error-remedy", testdata.ValidErrorRemedyDSL, ""},
		{"invalid-empty-error-remedy", testdata.InvalidEmptyErrorRemedyDSL,
			`error remedy: error remedy must define at least one of code, safe message, or retry hint`,
		},
		{"invalid-error-remedy-placement", testdata.InvalidErrorRemedyPlacementDSL,
			`invalid use of RemedyCode`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Error == "" {
				expr.RunDSL(t, tc.DSL)
			} else {
				err := expr.RunInvalidDSL(t, tc.DSL)
				assert.Contains(t, stripValidationLocations(err.Error()), tc.Error)
			}
		})
	}
}

func TestErrorRemedy(t *testing.T) {
	root := expr.RunDSL(t, testdata.ValidErrorRemedyDSL)
	method := root.Service("ValidErrorRemedyService").Method("SecureMethod")
	erro := method.Error("bad_request")
	if assert.NotNil(t, erro) && assert.NotNil(t, erro.Remedy) {
		assert.Equal(t, "bad_request.fix", erro.Remedy.Code)
		assert.Equal(t, "The request is invalid.", erro.Remedy.SafeMessage)
		assert.Equal(t, "Correct the payload and retry.", erro.Remedy.RetryHint)
	}
}

func TestAuthErrorResponses(t *testing.T) {
	t.Run("method scope injects method and endpoint auth errors", func(t *testing.T) {
		root := expr.RunDSL(t, testdata.ValidMethodAuthErrorResponsesDSL)
		method := root.Service("ValidMethodAuthErrorResponsesService").Method("SecureMethod")
		assert.NotNil(t, method.Error("unauthorized"))
		assert.NotNil(t, method.Error("forbidden"))

		endpoint := root.API.HTTP.Service("ValidMethodAuthErrorResponsesService").Endpoint("SecureMethod")
		if assert.Len(t, endpoint.HTTPErrors, 2) {
			assert.Equal(t, "unauthorized", endpoint.HTTPErrors[0].Name)
			assert.Equal(t, expr.StatusUnauthorized, endpoint.HTTPErrors[0].Response.StatusCode)
			assert.Equal(t, "forbidden", endpoint.HTTPErrors[1].Name)
			assert.Equal(t, expr.StatusForbidden, endpoint.HTTPErrors[1].Response.StatusCode)
		}
	})

	t.Run("service scope injects shared auth errors", func(t *testing.T) {
		root := expr.RunDSL(t, testdata.ValidServiceAuthErrorResponsesDSL)
		service := root.Service("ValidServiceAuthErrorResponsesService")
		assert.NotNil(t, service.Error("unauthorized"))
		assert.NotNil(t, service.Error("forbidden"))

		httpService := root.API.HTTP.Service("ValidServiceAuthErrorResponsesService")
		if assert.Len(t, httpService.HTTPErrors, 2) {
			assert.Equal(t, "unauthorized", httpService.HTTPErrors[0].Name)
			assert.Equal(t, expr.StatusUnauthorized, httpService.HTTPErrors[0].Response.StatusCode)
			assert.Equal(t, "forbidden", httpService.HTTPErrors[1].Name)
			assert.Equal(t, expr.StatusForbidden, httpService.HTTPErrors[1].Response.StatusCode)
		}
	})

	t.Run("api scope injects shared auth errors", func(t *testing.T) {
		root := expr.RunDSL(t, testdata.ValidAPIAuthErrorResponsesDSL)
		assert.NotNil(t, root.Error("unauthorized"))
		assert.NotNil(t, root.Error("forbidden"))
		if assert.Len(t, root.API.HTTP.Errors, 2) {
			assert.Equal(t, "unauthorized", root.API.HTTP.Errors[0].Name)
			assert.Equal(t, expr.StatusUnauthorized, root.API.HTTP.Errors[0].Response.StatusCode)
			assert.Equal(t, "forbidden", root.API.HTTP.Errors[1].Name)
			assert.Equal(t, expr.StatusForbidden, root.API.HTTP.Errors[1].Response.StatusCode)
		}
	})

	t.Run("existing error definitions are preserved", func(t *testing.T) {
		root := expr.RunDSL(t, testdata.ValidMethodAuthErrorResponsesReuseDSL)
		method := root.Service("ValidMethodAuthErrorResponsesReuseService").Method("SecureMethod")
		unauthorized := method.Error("unauthorized")
		if assert.NotNil(t, unauthorized) {
			assert.NotEqual(t, expr.ErrorResult, unauthorized.Type)
			assert.Equal(t, "MethodAuthErrorUnauthorized", unauthorized.Type.Name())
		}
	})

	t.Run("repeated calls are idempotent", func(t *testing.T) {
		root := expr.RunDSL(t, testdata.ValidMethodAuthErrorResponsesRepeatedDSL)
		method := root.Service("ValidMethodAuthErrorResponsesRepeatedService").Method("SecureMethod")
		httpEndpoint := root.API.HTTP.Service("ValidMethodAuthErrorResponsesRepeatedService").Endpoint("SecureMethod")

		assert.NotNil(t, method.Error("unauthorized"))
		assert.NotNil(t, method.Error("forbidden"))
		if assert.Len(t, httpEndpoint.HTTPErrors, 2) {
			assert.Equal(t, "unauthorized", httpEndpoint.HTTPErrors[0].Name)
			assert.Equal(t, "forbidden", httpEndpoint.HTTPErrors[1].Name)
		}
	})

	t.Run("existing mappings are preserved", func(t *testing.T) {
		root := expr.RunDSL(t, testdata.ValidMethodAuthErrorResponsesCustomMappingDSL)
		httpEndpoint := root.API.HTTP.Service("ValidMethodAuthErrorResponsesCustomMappingService").Endpoint("SecureMethod")
		if assert.Len(t, httpEndpoint.HTTPErrors, 2) {
			assert.Equal(t, "unauthorized", httpEndpoint.HTTPErrors[0].Name)
			assert.Equal(t, expr.StatusPaymentRequired, httpEndpoint.HTTPErrors[0].Response.StatusCode)
			assert.Equal(t, "Custom auth challenge", httpEndpoint.HTTPErrors[0].Response.Description)
			assert.Equal(t, "forbidden", httpEndpoint.HTTPErrors[1].Name)
			assert.Equal(t, expr.StatusForbidden, httpEndpoint.HTTPErrors[1].Response.StatusCode)
		}
	})

	t.Run("service mappings are reused by method helper", func(t *testing.T) {
		root := expr.RunDSL(t, testdata.ValidMethodAuthErrorResponsesReuseServiceMappingDSL)
		httpEndpoint := root.API.HTTP.Service("ValidMethodAuthErrorResponsesReuseServiceMappingService").Endpoint("SecureMethod")
		if assert.Len(t, httpEndpoint.HTTPErrors, 2) {
			assert.Equal(t, "unauthorized", httpEndpoint.HTTPErrors[0].Name)
			assert.Equal(t, expr.StatusUnauthorized, httpEndpoint.HTTPErrors[0].Response.StatusCode)
			assert.Equal(t, "Session expired", httpEndpoint.HTTPErrors[0].Response.Description)
			assert.Equal(t, "forbidden", httpEndpoint.HTTPErrors[1].Name)
			assert.Equal(t, expr.StatusForbidden, httpEndpoint.HTTPErrors[1].Response.StatusCode)
			assert.Equal(t, "Plan upgrade required", httpEndpoint.HTTPErrors[1].Response.Description)
		}
	})

	t.Run("api mappings are reused by method helper", func(t *testing.T) {
		root := expr.RunDSL(t, testdata.ValidMethodAuthErrorResponsesReuseAPIMappingDSL)
		httpEndpoint := root.API.HTTP.Service("ValidMethodAuthErrorResponsesReuseAPIMappingService").Endpoint("SecureMethod")
		if assert.Len(t, httpEndpoint.HTTPErrors, 2) {
			assert.Equal(t, "unauthorized", httpEndpoint.HTTPErrors[0].Name)
			assert.Equal(t, expr.StatusUnauthorized, httpEndpoint.HTTPErrors[0].Response.StatusCode)
			assert.Equal(t, "Session expired", httpEndpoint.HTTPErrors[0].Response.Description)
			assert.Equal(t, "forbidden", httpEndpoint.HTTPErrors[1].Name)
			assert.Equal(t, expr.StatusForbidden, httpEndpoint.HTTPErrors[1].Response.StatusCode)
			assert.Equal(t, "Team membership required", httpEndpoint.HTTPErrors[1].Response.Description)
		}
	})
}

func TestSessionSecurityLowersToMethodRequirements(t *testing.T) {
	root := expr.RunDSL(t, testdata.ValidSessionSecurityDSL)
	method := root.Service("ValidSessionSecurityService").Method("SecureMethod")
	assert.Len(t, method.Requirements, 2)
	assert.Len(t, method.Requirements[0].Schemes, 1)
	assert.Len(t, method.Requirements[1].Schemes, 1)
	assert.Equal(t, expr.JWTKind, method.Requirements[0].Schemes[0].Kind)
	assert.Equal(t, "jwt", method.Requirements[0].Schemes[0].SchemeName)
	assert.Equal(t, expr.APIKeyKind, method.Requirements[1].Schemes[0].Kind)
	assert.Equal(t, "api_key", method.Requirements[1].Schemes[0].SchemeName)
	assert.Len(t, root.SessionAuths, 1)
	assert.Equal(t, "app_session", root.SessionAuths[0].Name)
	assert.Len(t, root.SessionAuths[0].Transports, 2)
}

func TestAPISessionSecurityDerivesMethodRequirements(t *testing.T) {
	root := expr.RunDSL(t, testdata.ValidAPISessionSecurityDSL)
	method := root.Service("ValidAPISessionSecurityService").Method("SecureMethod")
	assert.Len(t, root.API.Requirements, 0)
	assert.Len(t, root.API.SessionAuths, 1)
	assert.Len(t, method.Requirements, 2)
	assert.Equal(t, expr.JWTKind, method.Requirements[0].Schemes[0].Kind)
	assert.Equal(t, expr.APIKeyKind, method.Requirements[1].Schemes[0].Kind)
}

func TestSessionSecurityNoSecurityOverrideClearsDerivedState(t *testing.T) {
	root := expr.RunDSL(t, testdata.ValidServiceSessionSecurityNoSecurityDSL)
	method := root.Service("ValidServiceSessionSecurityNoSecurityService").Method("SecureMethod")
	assert.Empty(t, method.Requirements)
	assert.Empty(t, method.SessionAuths)
	assert.Nil(t, method.Payload.Find("auth"))
	assert.Nil(t, method.Payload.Find("browser_session"))
}

func TestSessionSecurityInjectsPayloadFields(t *testing.T) {
	root := expr.RunDSL(t, testdata.ValidSessionSecurityAutoPayloadDSL)
	method := root.Service("ValidSessionSecurityAutoPayloadService").Method("SecureMethod")
	auth := method.Payload.Find("auth")
	assert.NotNil(t, auth)
	if assert.NotNil(t, auth) {
		_, ok := auth.Meta["security:token"]
		assert.True(t, ok)
		assert.Equal(t, expr.String, auth.Type)
	}
	browserSession := method.Payload.Find("browser_session")
	assert.NotNil(t, browserSession)
	if assert.NotNil(t, browserSession) {
		_, ok := browserSession.Meta["security:apikey:api_key"]
		assert.True(t, ok)
		assert.Equal(t, expr.String, browserSession.Type)
	}
}

func TestMethodSecurityHelpersAllowDetachedMethods(t *testing.T) {
	t.Run("effective requirements", func(t *testing.T) {
		method := &expr.MethodExpr{
			Name: "DetachedMethod",
		}

		require.NotPanics(t, func() {
			require.Nil(t, method.EffectiveRequirements())
			require.Nil(t, method.EffectiveSessionAuths())
		})
	})

	t.Run("finalize", func(t *testing.T) {
		expr.SetupTestDSL(t)

		method := &expr.MethodExpr{
			Name:    "DetachedMethod",
			Payload: &expr.AttributeExpr{Type: expr.Empty},
			Result:  &expr.AttributeExpr{Type: expr.Empty},
		}

		require.NotPanics(t, method.Finalize)
	})
}

func TestTransportOwnedCookieSessionSecuritySkipsPayloadInjection(t *testing.T) {
	root := expr.RunDSL(t, testdata.ValidCookieOnlyTransportOwnedSessionSecurityDSL)
	method := root.Service("ValidCookieOnlyTransportOwnedSessionSecurityService").Method("SecureMethod")
	assert.Nil(t, method.Payload.Find("browser_session_cookie"))
	assert.NotEmpty(t, method.Requirements)
	assert.Equal(t, expr.APIKeyKind, method.Requirements[0].Schemes[0].Kind)
	assert.Equal(t, "browser_session_cookie", method.Requirements[0].Schemes[0].SchemeName)
}

func TestMethodExprError(t *testing.T) {
	var (
		errorFoo = &expr.ErrorExpr{
			Name: "foo",
		}
		errorBar = &expr.ErrorExpr{
			Name: "bar",
		}
		errorBaz = &expr.ErrorExpr{
			Name: "baz",
		}
	)
	cases := map[string]struct {
		name     string
		expected *expr.ErrorExpr
	}{
		"exist in method": {
			name:     "foo",
			expected: errorFoo,
		},
		"exist in service": {
			name:     "bar",
			expected: errorBar,
		},
		"exist in root": {
			name:     "baz",
			expected: errorBaz,
		},
		"not exist": {
			name:     "qux",
			expected: nil,
		},
	}

	expr.Root.Errors = []*expr.ErrorExpr{
		errorBaz,
	}
	s := expr.ServiceExpr{
		Errors: []*expr.ErrorExpr{
			errorBar,
		},
	}
	m := expr.MethodExpr{
		Errors: []*expr.ErrorExpr{
			errorFoo,
		},
		Service: &s,
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			if actual := m.Error(tc.name); actual != tc.expected {
				t.Errorf("got %#v, expected %#v", actual, tc.expected)
			}
		})
	}
}

func TestMethodExprEvalName(t *testing.T) {
	cases := map[string]struct {
		name     string
		service  *expr.ServiceExpr
		expected string
	}{
		"unnamed": {name: "", service: nil, expected: "unnamed method"},
		"foo":     {name: "foo", service: nil, expected: fmt.Sprintf("method %#v", "foo")},
		"bar":     {name: "bar", service: &expr.ServiceExpr{Name: ""}, expected: fmt.Sprintf("unnamed service method %#v", "bar")},
		"baz":     {name: "baz", service: &expr.ServiceExpr{Name: "baz service"}, expected: fmt.Sprintf("service %#v method %#v", "baz service", "baz")},
	}
	for k, tc := range cases {
		m := expr.MethodExpr{Name: tc.name, Service: tc.service}
		if actual := m.EvalName(); actual != tc.expected {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestMethodExprIsPayloadStreaming(t *testing.T) {
	cases := map[string]struct {
		stream   expr.StreamKind
		expected bool
	}{
		"no stream": {
			stream:   expr.NoStreamKind,
			expected: false,
		},
		"client stream": {
			stream:   expr.ClientStreamKind,
			expected: true,
		},
		"server stream": {
			stream:   expr.ServerStreamKind,
			expected: false,
		},
		"BidirectionalStreamKind": {
			stream:   expr.BidirectionalStreamKind,
			expected: true,
		},
	}
	for k, tc := range cases {
		m := expr.MethodExpr{
			Stream: tc.stream,
		}
		if actual := m.IsPayloadStreaming(); actual != tc.expected {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestMethodExprValidateInterceptors(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Error string
	}{
		{"no-interceptors", testdata.NoInterceptorsDSL, ""},
		{"valid-interceptors", testdata.ValidInterceptorsDSL, ""},
		{"duplicate-interceptors", testdata.DuplicateInterceptorsDSL, ""}, // Duplicates are handled by merging
		{"mixed-interceptors", testdata.MixedInterceptorsDSL, ""},
		{"undefined-interceptor", testdata.UndefinedInterceptorDSL,
			`ServerInterceptor: interceptor "undefined" not found in service "Service" method "Method"`},
		{"empty-interceptor-name", testdata.EmptyInterceptorNameDSL,
			`ServerInterceptor: interceptor name cannot be empty`},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if c.Error == "" {
				expr.RunDSL(t, c.DSL)
				return
			}
			err := expr.RunInvalidDSL(t, c.DSL)
			assert.Contains(t, err.Error(), c.Error)
		})
	}
}
