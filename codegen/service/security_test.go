package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/expr"
)

func TestSecureEndpointInit(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
		Code string
	}{
		{"endpoint-without-requirement", testdata.EndpointWithoutRequirementDSL, testdata.EndpointInitWithoutRequirementCode},
		{"endpoints-with-requirements", testdata.EndpointsWithRequirementsDSL, testdata.EndpointInitWithRequirementsCode},
		{"endpoints-with-service-requirements", testdata.EndpointsWithServiceRequirementsDSL, testdata.EndpointInitWithServiceRequirementsCode},
		{"endpoints-no-security", testdata.EndpointNoSecurityDSL, testdata.EndpointInitNoSecurityCode},
		{"cookie-only-transport-owned-session-security", testdata.EndpointWithCookieOnlyTransportOwnedSessionSecurityDSL, testdata.EndpointInitWithCookieOnlyTransportOwnedSessionSecurityCode},
		{"service-cookie-only-transport-owned-session-security", testdata.EndpointWithServiceCookieOnlyTransportOwnedSessionSecurityDSL, testdata.EndpointInitWithServiceCookieOnlyTransportOwnedSessionSecurityCode},
		{"api-cookie-only-transport-owned-session-security", testdata.EndpointWithAPITransportOwnedCookieSessionSecurityDSL, testdata.EndpointInitWithAPITransportOwnedCookieSessionSecurityCode},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			require.Len(t, root.Services, 1)
			fs := EndpointFile("", root.Services[0], services)
			require.NotNil(t, fs)
			sections := fs.AllSections()
			require.Greater(t, len(sections), 1)
			code := codegen.SectionCode(t, sections[2])
			assert.Equal(t, c.Code, code)
		})
	}
}

func TestExpandRequirementSchemesPreservesTransportLocation(t *testing.T) {
	base := RequirementsData{
		&RequirementData{
			Schemes: SchemesData{
				&SchemeData{SchemeName: "jwt", Type: "JWT", In: "header"},
				&SchemeData{SchemeName: "oauth2", Type: "OAuth2", In: "query"},
			},
		},
	}
	requirements := []*expr.SecurityExpr{
		{Schemes: []*expr.SchemeExpr{
			{SchemeName: "jwt", In: "metadata"},
			{SchemeName: "oauth2", In: "message"},
		}},
	}

	schemes := ExpandRequirementSchemes(requirements, base)
	require.Len(t, schemes, 2)
	assert.Equal(t, "metadata", schemes[0].In)
	assert.Equal(t, "message", schemes[1].In)
}

func TestPartitionSchemesByIn(t *testing.T) {
	schemes := SchemesData{
		&SchemeData{SchemeName: "basic", Type: "Basic"},
		&SchemeData{SchemeName: "jwt", Type: "JWT", In: "header"},
		&SchemeData{SchemeName: "oauth", Type: "OAuth2", In: "query"},
		&SchemeData{SchemeName: "session", Type: "APIKey"},
	}

	basic, grouped, fallback := PartitionSchemesByIn(schemes)
	require.NotNil(t, basic)
	assert.Equal(t, "basic", basic.SchemeName)
	require.Len(t, grouped["header"], 1)
	assert.Equal(t, "jwt", grouped["header"][0].SchemeName)
	require.Len(t, grouped["query"], 1)
	assert.Equal(t, "oauth", grouped["query"][0].SchemeName)
	require.Len(t, fallback, 1)
	assert.Equal(t, "session", fallback[0].SchemeName)
}

func TestSessionSecurityMatchesHandAuthoredSecurityCodegenData(t *testing.T) {
	manualRoot := codegen.RunDSL(t, testdata.EndpointWithBearerOrCookieSecurityDSL)
	sessionRoot := codegen.RunDSL(t, testdata.EndpointWithSessionSecurityDSL)

	manualServices := NewServicesData(manualRoot)
	sessionServices := NewServicesData(sessionRoot)

	manualMethod := manualServices.Get("EndpointWithBearerOrCookieSecurity").Method("Secure")
	sessionMethod := sessionServices.Get("EndpointWithSessionSecurity").Method("Secure")
	require.NotNil(t, manualMethod)
	require.NotNil(t, sessionMethod)

	require.Len(t, manualMethod.Requirements, 2)
	require.Len(t, sessionMethod.Requirements, 2)

	manualJWT := manualMethod.Requirements.Scheme("jwt")
	sessionJWT := sessionMethod.Requirements.Scheme("jwt")
	require.NotNil(t, manualJWT)
	require.NotNil(t, sessionJWT)
	assert.Equal(t, manualJWT.Type, sessionJWT.Type)
	assert.Equal(t, manualJWT.CredField, sessionJWT.CredField)
	assert.Equal(t, manualJWT.KeyAttr, sessionJWT.KeyAttr)

	manualAPIKey := manualMethod.Requirements.Scheme("api_key")
	sessionAPIKey := sessionMethod.Requirements.Scheme("api_key")
	require.NotNil(t, manualAPIKey)
	require.NotNil(t, sessionAPIKey)
	assert.Equal(t, manualAPIKey.Type, sessionAPIKey.Type)
	assert.Equal(t, manualAPIKey.CredField, sessionAPIKey.CredField)
	assert.Equal(t, manualAPIKey.KeyAttr, sessionAPIKey.KeyAttr)

	assert.Contains(t, sessionMethod.PayloadDef, "Auth *string")
	assert.Contains(t, sessionMethod.PayloadDef, "Message *string")
	testutil.AssertGo(t, "testdata/golden/security_session_payload_def.go.golden", sessionMethod.PayloadDef)
}

func TestSecureEndpointIsolatesAlternativeRequirementContexts(t *testing.T) {
	cases := []struct {
		name        string
		dsl         func()
		serviceName string
	}{
		{
			name:        "hand-authored security",
			dsl:         testdata.EndpointWithBearerOrCookieSecurityDSL,
			serviceName: "EndpointWithBearerOrCookieSecurity",
		},
		{
			name:        "session security",
			dsl:         testdata.EndpointWithSessionSecurityDSL,
			serviceName: "EndpointWithSessionSecurity",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, tc.dsl)
			service := NewServicesData(root).Get(tc.serviceName)
			require.NotNil(t, service)
			method := endpointData(service).Methods[0]
			require.NotNil(t, method)
			assert.Equal(t, "Secure", method.Name)
			require.Len(t, method.Requirements, 2)

			code := codegen.SectionCode(t, endpointMethodSection(method))
			assert.Contains(t, code, "authCtx := ctx")
			assert.Contains(t, code, "authCtx, err = authJWTFn(authCtx,")
			assert.Contains(t, code, "if err != nil {\n\t\t\tauthCtx = ctx")
			assert.Contains(t, code, "authCtx, err = authAPIKeyFn(authCtx,")
			assert.Contains(t, code, "ctx = authCtx")
			assert.NotContains(t, code, "ctx, err = authJWTFn(ctx,")
			assert.NotContains(t, code, "ctx, err = authAPIKeyFn(ctx,")
		})
	}
}

func TestSecureEndpointPreservesContextWithinCompoundRequirement(t *testing.T) {
	root := codegen.RunDSL(t, testdata.EndpointWithCompoundOrSecurityDSL)
	service := NewServicesData(root).Get("EndpointWithCompoundOrSecurity")
	require.NotNil(t, service)
	method := endpointData(service).Methods[0]
	require.NotNil(t, method)
	require.Len(t, method.Requirements, 2)
	require.Len(t, method.Requirements[0].Schemes, 2)

	code := codegen.SectionCode(t, endpointMethodSection(method))
	andGuard := strings.Index(code, "if err == nil {")
	basicAuth := strings.Index(code, "authCtx, err = authBasicFn(authCtx,")
	jwtAuth := strings.Index(code, "authCtx, err = authJWTFn(authCtx,")
	reset := strings.Index(code, "if err != nil {\n\t\t\tauthCtx = ctx")
	apiKeyAuth := strings.Index(code, "authCtx, err = authAPIKeyFn(authCtx,")
	positions := []struct {
		name  string
		index int
	}{
		{name: "basic authorizer", index: basicAuth},
		{name: "AND guard", index: andGuard},
		{name: "JWT authorizer", index: jwtAuth},
		{name: "alternative reset", index: reset},
		{name: "API key authorizer", index: apiKeyAuth},
	}
	for _, position := range positions {
		assert.NotEqual(t, -1, position.index, position.name)
	}
	assert.Greater(t, andGuard, basicAuth)
	assert.Greater(t, jwtAuth, andGuard)
	assert.Greater(t, reset, jwtAuth)
	assert.Greater(t, apiKeyAuth, reset)
}

func TestAPISessionSecurityMatchesHandAuthoredSecurityCodegenData(t *testing.T) {
	manualRoot := codegen.RunDSL(t, testdata.EndpointWithBearerOrCookieAPISecurityDSL)
	sessionRoot := codegen.RunDSL(t, testdata.EndpointWithAPISessionSecurityDSL)

	manualServices := NewServicesData(manualRoot)
	sessionServices := NewServicesData(sessionRoot)

	manualMethod := manualServices.Get("EndpointWithBearerOrCookieAPISecurity").Method("Secure")
	sessionMethod := sessionServices.Get("EndpointWithAPISessionSecurity").Method("Secure")
	require.NotNil(t, manualMethod)
	require.NotNil(t, sessionMethod)

	require.Len(t, manualMethod.Requirements, 2)
	require.Len(t, sessionMethod.Requirements, 2)

	manualJWT := manualMethod.Requirements.Scheme("jwt")
	sessionJWT := sessionMethod.Requirements.Scheme("jwt")
	require.NotNil(t, manualJWT)
	require.NotNil(t, sessionJWT)
	assert.Equal(t, manualJWT.Type, sessionJWT.Type)
	assert.Equal(t, manualJWT.CredField, sessionJWT.CredField)

	manualAPIKey := manualMethod.Requirements.Scheme("api_key")
	sessionAPIKey := sessionMethod.Requirements.Scheme("api_key")
	require.NotNil(t, manualAPIKey)
	require.NotNil(t, sessionAPIKey)
	assert.Equal(t, manualAPIKey.Type, sessionAPIKey.Type)
	assert.Equal(t, manualAPIKey.CredField, sessionAPIKey.CredField)

	assert.Contains(t, sessionMethod.PayloadDef, "Auth *string")
	assert.Contains(t, sessionMethod.PayloadDef, "Message *string")
	testutil.AssertGo(t, "testdata/golden/security_api_session_payload_def.go.golden", sessionMethod.PayloadDef)
}

func TestSessionSecurityNoSecurityOverrideRemovesGeneratedRequirements(t *testing.T) {
	root := codegen.RunDSL(t, testdata.EndpointWithServiceSessionSecurityNoSecurityDSL)
	services := NewServicesData(root)
	method := services.Get("EndpointWithServiceSessionSecurityNoSecurity").Method("Secure")
	require.NotNil(t, method)
	assert.Empty(t, method.Requirements)
	assert.NotContains(t, method.PayloadDef, "Auth *string")
	assert.NotContains(t, method.PayloadDef, "BrowserSession *string")
	assert.Contains(t, method.PayloadDef, "Message *string")
}

func TestCookieOnlyTransportOwnedSessionSecurityOmitsGeneratedCredentialField(t *testing.T) {
	cases := []struct {
		name        string
		dsl         func()
		serviceName string
		schemeName  string
	}{
		{
			name:        "method session security",
			dsl:         testdata.EndpointWithCookieOnlyTransportOwnedSessionSecurityDSL,
			serviceName: "EndpointWithCookieOnlyTransportOwnedSessionSecurity",
			schemeName:  "browser_session_cookie",
		},
		{
			name:        "service session security",
			dsl:         testdata.EndpointWithServiceCookieOnlyTransportOwnedSessionSecurityDSL,
			serviceName: "EndpointWithServiceCookieOnlyTransportOwnedSessionSecurity",
			schemeName:  "service_browser_session_cookie",
		},
		{
			name:        "api session security",
			dsl:         testdata.EndpointWithAPITransportOwnedCookieSessionSecurityDSL,
			serviceName: "EndpointWithAPITransportOwnedCookieSessionSecurity",
			schemeName:  "api_browser_session_cookie",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, tc.dsl)
			services := NewServicesData(root)
			method := services.Get(tc.serviceName).Method("Secure")
			require.NotNil(t, method)
			require.Len(t, method.Requirements, 1)
			require.Len(t, method.Requirements[0].Schemes, 1)
			scheme := method.Requirements.Scheme(tc.schemeName)
			require.NotNil(t, scheme)
			assert.Equal(t, "APIKey", scheme.Type)
			assert.True(t, scheme.TransportOwned)
			assert.Empty(t, scheme.CredField)
			assert.Empty(t, scheme.KeyAttr)
			assert.NotContains(t, method.PayloadDef, "BrowserSession")
			assert.Contains(t, method.PayloadDef, "Message *string")
		})
	}
}

func TestSecureEndpoint(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
		Code string
	}{
		{"with-required-scopes", testdata.EndpointWithRequiredScopesDSL, testdata.EndpointWithRequiredScopesCode},
		{"with-optional-required-scopes", testdata.EndpointWithOptionalRequiredScopesDSL, testdata.EndpointWithOptionalRequiredScopesCode},
		{"with-api-key-override", testdata.EndpointWithAPIKeyOverrideDSL, testdata.EndpointWithAPIKeyOverrideCode},
		{"with-oauth2", testdata.EndpointWithOAuth2DSL, testdata.EndpointWithOAuth2Code},
		{"with-cookie-only-transport-owned-session-security", testdata.EndpointWithCookieOnlyTransportOwnedSessionSecurityDSL, testdata.EndpointWithCookieOnlyTransportOwnedSessionSecurityCode},
		{"with-service-cookie-only-transport-owned-session-security", testdata.EndpointWithServiceCookieOnlyTransportOwnedSessionSecurityDSL, testdata.EndpointWithServiceCookieOnlyTransportOwnedSessionSecurityCode},
		{"with-api-cookie-only-transport-owned-session-security", testdata.EndpointWithAPITransportOwnedCookieSessionSecurityDSL, testdata.EndpointWithAPITransportOwnedCookieSessionSecurityCode},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			require.Len(t, root.Services, 1)
			fs := EndpointFile("", root.Services[0], services)
			require.NotNil(t, fs)
			sections := fs.AllSections()
			code := codegen.SectionCode(t, sections[4])
			assert.Equal(t, c.Code, code)
		})
	}
}

func TestSecureWithSkipRequestBodyEncodeDecode(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
		Code string
	}{
		{"with-basicauth", testdata.EndpointWithBasicAuthAndSkipRequestBodyEncodeDecodeDSL, testdata.EndpointWithBasicAuthAndSkipRequestBodyEncodeDecodeCode},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)
			services := NewServicesData(root)
			require.Len(t, root.Services, 1)
			fs := EndpointFile("", root.Services[0], services)
			require.NotNil(t, fs)
			sections := fs.AllSections()
			code := codegen.SectionCode(t, sections[5])
			assert.Equal(t, c.Code, code)
		})
	}
}
