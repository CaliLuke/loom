package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
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
	assert.Contains(t, sessionMethod.PayloadDef, "BrowserSession *string")
	assert.Contains(t, sessionMethod.PayloadDef, "Message *string")
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
	assert.Contains(t, sessionMethod.PayloadDef, "BrowserSession *string")
	assert.Contains(t, sessionMethod.PayloadDef, "Message *string")
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
