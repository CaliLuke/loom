package codegen

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/dsl"
)

func TestJSONRPCSessionSecurityHarness(t *testing.T) {
	t.Run("api-level session security derives json-rpc transport auth handling", func(t *testing.T) {
		root := RunJSONRPCDSL(t, jsonrpcAPISessionSecurityDSL)
		require.Len(t, root.API.Requirements, 0)
		require.Len(t, root.API.SessionAuths, 1)

		services := CreateJSONRPCServices(root)
		svc := services.Get("jsonrpcSessionSecurity")
		require.NotNil(t, svc)
		endpoint := svc.Endpoint("secure")
		require.NotNil(t, endpoint)

		require.Len(t, endpoint.Method.Requirements, 2)
		assert.NotNil(t, endpoint.Method.Requirements.Scheme("jwt"))
		assert.NotNil(t, endpoint.Method.Requirements.Scheme("api_key"))
		require.NotNil(t, endpoint.Payload)
		require.NotNil(t, endpoint.Payload.Request)
		assert.Len(t, endpoint.Payload.Request.Headers, 1)
		assert.Len(t, endpoint.Payload.Request.Cookies, 1)
		assert.Equal(t, "Authorization", endpoint.Payload.Request.Headers[0].HTTPName)
		assert.Equal(t, "__Host-ak_session", endpoint.Payload.Request.Cookies[0].HTTPName)

		clientCode := jsonrpcGeneratedCode(t, ClientFiles("", services))
		assert.Contains(t, clientCode, `req.AddCookie(&http.Cookie{`)
		assert.Contains(t, clientCode, `"__Host-ak_session"`)
		assert.Contains(t, clientCode, `"Authorization"`)

		serverCode := jsonrpcGeneratedCode(t, ServerFiles("", services))
		assert.Contains(t, serverCode, `r.Cookie("__Host-ak_session")`)
		assert.Contains(t, serverCode, `"Authorization"`)
	})

	t.Run("no security override clears derived json-rpc auth state", func(t *testing.T) {
		root := RunJSONRPCDSL(t, jsonrpcSessionSecurityNoSecurityDSL)
		services := CreateJSONRPCServices(root)
		svc := services.Get("jsonrpcSessionSecurityNoSecurity")
		require.NotNil(t, svc)
		endpoint := svc.Endpoint("secure")
		require.NotNil(t, endpoint)

		assert.Empty(t, endpoint.Method.Requirements)
		require.NotNil(t, endpoint.Payload)
		require.NotNil(t, endpoint.Payload.Request)
		assert.Empty(t, endpoint.Payload.Request.Headers)
		assert.Empty(t, endpoint.Payload.Request.Cookies)
		assert.NotContains(t, endpoint.Method.PayloadDef, "Auth *string")
		assert.NotContains(t, endpoint.Method.PayloadDef, "BrowserSession *string")

		clientCode := jsonrpcGeneratedCode(t, ClientFiles("", services))
		assert.NotContains(t, clientCode, `req.AddCookie(&http.Cookie{`)
		assert.NotContains(t, clientCode, `"Authorization"`)

		serverCode := jsonrpcGeneratedCode(t, ServerFiles("", services))
		assert.NotContains(t, serverCode, `r.Cookie("__Host-ak_session")`)
		assert.NotContains(t, serverCode, `"Authorization"`)
	})
}

func jsonrpcGeneratedCode(t *testing.T, files []*codegen.File) string {
	t.Helper()

	parts := make([]string, 0, len(files))
	for _, file := range files {
		for _, section := range file.AllSections() {
			var buf bytes.Buffer
			require.NoError(t, section.Write(&buf))
			parts = append(parts, buf.String())
		}
	}
	return strings.Join(parts, "\n")
}

var jsonrpcAPISessionSecurityDSL = func() {
	var jwt = dsl.JWTSecurity("jwt")
	var apiKey = dsl.APIKeySecurity("api_key")
	var appSession = dsl.SessionAuth("jsonrpc_api_session", func() {
		dsl.BearerTransport(jwt, "auth")
		dsl.CookieTransport(apiKey, "browser_session", func() {
			dsl.CookieName("__Host-ak_session")
		})
	})

	dsl.API("jsonrpcSessionSecurityAPI", func() {
		dsl.JSONRPC(func() {})
		dsl.SessionSecurity(appSession)
	})

	dsl.Service("jsonrpcSessionSecurity", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("secure", func() {
			dsl.Payload(func() {
				dsl.ID("id")
				dsl.Attribute("message", dsl.String)
			})
			dsl.Result(func() {
				dsl.ID("id")
				dsl.Attribute("response", dsl.String)
			})
			dsl.JSONRPC(func() {})
		})
	})
}

var jsonrpcSessionSecurityNoSecurityDSL = func() {
	var jwt = dsl.JWTSecurity("jwt")
	var apiKey = dsl.APIKeySecurity("api_key")
	var appSession = dsl.SessionAuth("jsonrpc_no_security_session", func() {
		dsl.BearerTransport(jwt, "auth")
		dsl.CookieTransport(apiKey, "browser_session", func() {
			dsl.CookieName("__Host-ak_session")
		})
	})

	dsl.API("jsonrpcSessionSecurityNoSecurityAPI", func() {
		dsl.JSONRPC(func() {})
	})

	dsl.Service("jsonrpcSessionSecurityNoSecurity", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.SessionSecurity(appSession)
		dsl.Method("secure", func() {
			dsl.NoSecurity()
			dsl.Payload(func() {
				dsl.ID("id")
				dsl.Attribute("message", dsl.String)
			})
			dsl.Result(func() {
				dsl.ID("id")
				dsl.Attribute("response", dsl.String)
			})
			dsl.JSONRPC(func() {})
		})
	})
}
