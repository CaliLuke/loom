package codegen

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/dsl"
	"goa.design/goa/v3/http/codegen/openapi"
	openapiv3 "goa.design/goa/v3/http/codegen/openapi/v3"
	"goa.design/goa/v3/http/codegen/testdata"
)

func TestOpenAPIOperationSecurityRequirements(t *testing.T) {
	t.Run("service security is emitted on operations", func(t *testing.T) {
		root := RunHTTPDSL(t, inheritedOperationSecurityDSL)
		openapi.Definitions = make(map[string]*openapi.Schema)

		spec := renderOpenAPIJSON(t, openapiv3.Files, root)
		parseOpenAPIV3Document(t, spec)

		op := operationFromSpec(t, spec, "/secured", "get")
		security, ok := op["security"].([]any)
		require.True(t, ok)
		require.Len(t, security, 1)
	})

	t.Run("session security is emitted on inherited operations", func(t *testing.T) {
		root := RunHTTPDSL(t, apiLevelSessionCookieSecurityDSL)
		openapi.Definitions = make(map[string]*openapi.Schema)

		spec := renderOpenAPIJSON(t, openapiv3.Files, root)
		parseOpenAPIV3Document(t, spec)

		op := operationFromSpec(t, spec, "/auth/profile", "get")
		security, ok := op["security"].([]any)
		require.True(t, ok)
		require.Len(t, security, 2)
	})

	t.Run("NoSecurity emits an explicit empty operation security array", func(t *testing.T) {
		root := RunHTTPDSL(t, noSecurityOperationDSL)
		openapi.Definitions = make(map[string]*openapi.Schema)

		spec := renderOpenAPIJSON(t, openapiv3.Files, root)
		parseOpenAPIV3Document(t, spec)

		op := operationFromSpec(t, spec, "/public", "get")
		security, ok := op["security"]
		require.True(t, ok)
		require.Empty(t, security.([]any))
	})

	t.Run("service session security applies to secured meal planner operations only", func(t *testing.T) {
		root := RunHTTPDSL(t, testdata.MealPlannerDSL)
		openapi.Definitions = make(map[string]*openapi.Schema)

		spec := renderOpenAPIJSON(t, openapiv3.Files, root)
		parseOpenAPIV3Document(t, spec)

		secured := operationFromSpec(t, spec, "/recipes", "get")
		security, ok := secured["security"].([]any)
		require.True(t, ok)
		require.Len(t, security, 2)

		public := operationFromSpec(t, spec, "/healthz", "get")
		noSecurity, ok := public["security"]
		require.True(t, ok)
		require.Empty(t, noSecurity.([]any))
	})
}

func operationFromSpec(t *testing.T, spec []byte, path, method string) map[string]any {
	t.Helper()

	var doc map[string]any
	require.NoError(t, json.Unmarshal(spec, &doc))

	paths, ok := doc["paths"].(map[string]any)
	require.True(t, ok)

	pathItem, ok := paths[path].(map[string]any)
	require.True(t, ok)

	op, ok := pathItem[method].(map[string]any)
	require.True(t, ok)
	return op
}

var inheritedOperationSecurityDSL = func() {
	var jwt = dsl.JWTSecurity("jwt")

	dsl.Service("securedService", func() {
		dsl.Security(jwt)

		dsl.Method("show", func() {
			dsl.Payload(func() {
				dsl.Token("token", dsl.String)
			})
			dsl.Result(dsl.Empty)
			dsl.HTTP(func() {
				dsl.GET("/secured")
				dsl.Response(dsl.StatusOK)
			})
		})
	})
}

var noSecurityOperationDSL = func() {
	var jwt = dsl.JWTSecurity("jwt")

	dsl.Service("publicOverride", func() {
		dsl.Security(jwt)

		dsl.Method("show", func() {
			dsl.NoSecurity()
			dsl.Result(dsl.Empty)
			dsl.HTTP(func() {
				dsl.GET("/public")
				dsl.Response(dsl.StatusOK)
			})
		})
	})
}
