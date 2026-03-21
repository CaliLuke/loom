package codegen

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/dsl"
	"goa.design/goa/v3/http/codegen/openapi"
	openapiv3 "goa.design/goa/v3/http/codegen/openapi/v3"
)

func TestAuthErrorResponses(t *testing.T) {
	t.Run("openapi includes standard auth error responses", func(t *testing.T) {
		root := RunHTTPDSL(t, authErrorResponsesDSL)
		openapi.Definitions = make(map[string]*openapi.Schema)

		v3JSON := renderOpenAPIJSON(t, openapiv3.Files, root)
		doc := parseOpenAPIV3Document(t, v3JSON)

		pathItem, ok := doc.Paths.PathItems.Get("/auth/profile")
		require.True(t, ok)
		require.NotNil(t, pathItem.Get)
		require.NotNil(t, pathItem.Get.Responses)
		unauthorized, ok := pathItem.Get.Responses.Codes.Get("401")
		require.True(t, ok)
		require.Equal(t, "unauthorized: Authentication is required.", unauthorized.Description)
		forbidden, ok := pathItem.Get.Responses.Codes.Get("403")
		require.True(t, ok)
		require.Equal(t, "forbidden: Access is forbidden.", forbidden.Description)
	})

	t.Run("server encoder includes standard auth error cases", func(t *testing.T) {
		root := RunHTTPDSL(t, authErrorResponsesDSL)
		services := CreateHTTPServices(root)

		serverFiles := ServerFiles("", services)
		require.Len(t, serverFiles, 2)
		var serverEncode bytes.Buffer
		for _, section := range serverFiles[1].AllSections() {
			require.NoError(t, section.Write(&serverEncode))
		}
		require.Contains(t, serverEncode.String(), `case "unauthorized":`)
		require.Contains(t, serverEncode.String(), "http.StatusUnauthorized")
		require.Contains(t, serverEncode.String(), `case "forbidden":`)
		require.Contains(t, serverEncode.String(), "http.StatusForbidden")
	})
}

var authErrorResponsesDSL = func() {
	var jwt = dsl.JWTSecurity("jwt", func() {
		dsl.Description("Application bearer")
	})

	dsl.Service("authErrorResponses", func() {
		dsl.Method("profile", func() {
			dsl.Security(jwt)
			dsl.Payload(func() {
				dsl.Token("auth", dsl.String)
			})
			dsl.Result(dsl.Empty)
			dsl.HTTP(func() {
				dsl.GET("/auth/profile")
				dsl.AuthErrorResponses()
				dsl.Response(dsl.StatusOK)
			})
		})
	})
}
