package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
)

func TestGRPCSessionSecurityHarness(t *testing.T) {
	t.Run("api-level session security derives grpc metadata schemes", func(t *testing.T) {
		root := RunGRPCDSL(t, grpcAPISessionSecurityDSL)
		services := CreateGRPCServices(root)
		service := services.Get("grpcSessionSecurity")
		require.NotNil(t, service)
		require.Len(t, service.Endpoints, 1)

		endpoint := service.Endpoints[0]
		require.Len(t, endpoint.Method.Requirements, 2)
		assert.NotNil(t, endpoint.Method.Requirements.Scheme("jwt"))
		assert.NotNil(t, endpoint.Method.Requirements.Scheme("api_key"))
		assert.NotNil(t, grpcScheme(endpoint.MetadataSchemes, "jwt"))
		assert.NotNil(t, grpcScheme(endpoint.MetadataSchemes, "api_key"))
		assert.Empty(t, endpoint.MessageSchemes)
		assert.Contains(t, endpoint.Method.PayloadDef, "Auth *string")
		assert.Contains(t, endpoint.Method.PayloadDef, "BrowserSession *string")
	})

	t.Run("no security override clears grpc derived schemes", func(t *testing.T) {
		root := RunGRPCDSL(t, grpcSessionSecurityNoSecurityDSL)
		services := CreateGRPCServices(root)
		service := services.Get("grpcSessionSecurityNoSecurity")
		require.NotNil(t, service)
		require.Len(t, service.Endpoints, 1)

		endpoint := service.Endpoints[0]
		assert.Empty(t, endpoint.Method.Requirements)
		assert.Empty(t, endpoint.MetadataSchemes)
		assert.Empty(t, endpoint.MessageSchemes)
		assert.NotContains(t, endpoint.Method.PayloadDef, "Auth *string")
		assert.NotContains(t, endpoint.Method.PayloadDef, "BrowserSession *string")
		assert.Contains(t, endpoint.Method.PayloadDef, "Message *string")
	})
}

func grpcScheme(schemes []*service.SchemeData, name string) *service.SchemeData {
	for _, scheme := range schemes {
		if scheme.SchemeName == name {
			return scheme
		}
	}
	return nil
}

var grpcAPISessionSecurityDSL = func() {
	var jwt = dsl.JWTSecurity("jwt")
	var apiKey = dsl.APIKeySecurity("api_key")
	var appSession = dsl.SessionAuth("grpc_api_session", func() {
		dsl.BearerTransport(jwt, "auth")
		dsl.CookieTransport(apiKey, "browser_session")
	})

	dsl.API("grpcSessionSecurityAPI", func() {
		dsl.SessionSecurity(appSession)
	})

	dsl.Service("grpcSessionSecurity", func() {
		dsl.Method("Secure", func() {
			dsl.Payload(func() {
				dsl.Field(1, "message", dsl.String)
			})
			dsl.Result(dsl.String)
			dsl.GRPC(func() {})
		})
	})
}

var grpcSessionSecurityNoSecurityDSL = func() {
	var jwt = dsl.JWTSecurity("jwt")
	var apiKey = dsl.APIKeySecurity("api_key")
	var appSession = dsl.SessionAuth("grpc_no_security_session", func() {
		dsl.BearerTransport(jwt, "auth")
		dsl.CookieTransport(apiKey, "browser_session")
	})

	dsl.Service("grpcSessionSecurityNoSecurity", func() {
		dsl.SessionSecurity(appSession)
		dsl.Method("Secure", func() {
			dsl.NoSecurity()
			dsl.Payload(func() {
				dsl.Field(1, "message", dsl.String)
			})
			dsl.Result(dsl.String)
			dsl.GRPC(func() {})
		})
	})
}
