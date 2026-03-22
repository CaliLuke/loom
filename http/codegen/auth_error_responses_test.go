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

	t.Run("openapi reuses canonical auth error components across scopes", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name                 string
			dsl                  func()
			path1                string
			path2                string
			wantUnauthorizedDesc string
			wantForbiddenDesc    string
		}{
			{
				name:                 "method",
				dsl:                  methodScopedAuthErrorReuseDSL,
				path1:                "/method/first",
				path2:                "/method/second",
				wantUnauthorizedDesc: "unauthorized: Session expired",
				wantForbiddenDesc:    "forbidden: Plan upgrade required",
			},
			{
				name:                 "service",
				dsl:                  serviceScopedAuthErrorReuseDSL,
				path1:                "/service/first",
				path2:                "/service/second",
				wantUnauthorizedDesc: "unauthorized: Session expired",
				wantForbiddenDesc:    "forbidden: Plan upgrade required",
			},
			{
				name:                 "api",
				dsl:                  apiScopedAuthErrorReuseDSL,
				path1:                "/api/first",
				path2:                "/api/second",
				wantUnauthorizedDesc: "unauthorized: Session expired",
				wantForbiddenDesc:    "forbidden: Team membership required",
			},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				root := RunHTTPDSL(t, tc.dsl)
				openapi.Definitions = make(map[string]*openapi.Schema)

				spec := openapiv3.New(root)
				require.NotNil(t, spec)
				require.NotNil(t, spec.Components)
				require.Contains(t, spec.Components.Responses, "UnauthorizedError")
				require.Contains(t, spec.Components.Responses, "ForbiddenError")
				require.NotNil(t, spec.Components.Responses["UnauthorizedError"].Value.Description)
				require.NotNil(t, spec.Components.Responses["ForbiddenError"].Value.Description)
				require.Equal(t, tc.wantUnauthorizedDesc, *spec.Components.Responses["UnauthorizedError"].Value.Description)
				require.Equal(t, tc.wantForbiddenDesc, *spec.Components.Responses["ForbiddenError"].Value.Description)
				require.Equal(t, "#/components/responses/UnauthorizedError", spec.Paths[tc.path1].Get.Responses["401"].Ref)
				require.Equal(t, "#/components/responses/UnauthorizedError", spec.Paths[tc.path2].Get.Responses["401"].Ref)
				require.Equal(t, "#/components/responses/ForbiddenError", spec.Paths[tc.path1].Get.Responses["403"].Ref)
				require.Equal(t, "#/components/responses/ForbiddenError", spec.Paths[tc.path2].Get.Responses["403"].Ref)

				v3JSON := renderOpenAPIJSON(t, openapiv3.Files, root)
				doc := parseOpenAPIV3Document(t, v3JSON)
				for _, path := range []string{tc.path1, tc.path2} {
					pathItem, ok := doc.Paths.PathItems.Get(path)
					require.True(t, ok)
					require.NotNil(t, pathItem.Get)
					unauthorized, ok := pathItem.Get.Responses.Codes.Get("401")
					require.True(t, ok)
					require.Equal(t, tc.wantUnauthorizedDesc, unauthorized.Description)
					forbidden, ok := pathItem.Get.Responses.Codes.Get("403")
					require.True(t, ok)
					require.Equal(t, tc.wantForbiddenDesc, forbidden.Description)
				}
			})
		}
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

var methodScopedAuthErrorReuseDSL = func() {
	var jwt = dsl.JWTSecurity("jwt", func() {
		dsl.Description("Application bearer")
	})
	var unauthorized = dsl.Type("MethodScopedUnauthorized", func() {
		dsl.Attribute("reason", dsl.String, func() {
			dsl.Example("session_expired")
		})
		dsl.Required("reason")
	})
	var forbidden = dsl.Type("MethodScopedForbidden", func() {
		dsl.Attribute("reason", dsl.String, func() {
			dsl.Example("plan_upgrade_required")
		})
		dsl.Required("reason")
	})

	dsl.Service("methodScopedAuthErrors", func() {
		for _, method := range []struct {
			name string
			path string
		}{
			{name: "first", path: "/method/first"},
			{name: "second", path: "/method/second"},
		} {
			method := method
			dsl.Method(method.name, func() {
				dsl.Security(jwt)
				dsl.Error("unauthorized", unauthorized)
				dsl.Error("forbidden", forbidden)
				dsl.Payload(func() {
					dsl.Token("auth", dsl.String)
				})
				dsl.Result(dsl.Empty)
				dsl.HTTP(func() {
					dsl.GET(method.path)
					dsl.Response("unauthorized", dsl.StatusUnauthorized, func() {
						dsl.Description("Session expired")
					})
					dsl.Response("forbidden", dsl.StatusForbidden, func() {
						dsl.Description("Plan upgrade required")
					})
					dsl.AuthErrorResponses()
					dsl.Response(dsl.StatusOK)
				})
			})
		}
	})
}

var serviceScopedAuthErrorReuseDSL = func() {
	var jwt = dsl.JWTSecurity("jwt", func() {
		dsl.Description("Application bearer")
	})
	var unauthorized = dsl.Type("ServiceScopedUnauthorized", func() {
		dsl.Attribute("reason", dsl.String, func() {
			dsl.Example("session_expired")
		})
		dsl.Required("reason")
	})
	var forbidden = dsl.Type("ServiceScopedForbidden", func() {
		dsl.Attribute("reason", dsl.String, func() {
			dsl.Example("plan_upgrade_required")
		})
		dsl.Required("reason")
	})

	dsl.Service("serviceScopedAuthErrors", func() {
		dsl.Security(jwt)
		dsl.Error("unauthorized", unauthorized)
		dsl.Error("forbidden", forbidden)
		dsl.HTTP(func() {
			dsl.Response("unauthorized", dsl.StatusUnauthorized, func() {
				dsl.Description("Session expired")
			})
			dsl.Response("forbidden", dsl.StatusForbidden, func() {
				dsl.Description("Plan upgrade required")
			})
		})
		for _, method := range []struct {
			name string
			path string
		}{
			{name: "first", path: "/service/first"},
			{name: "second", path: "/service/second"},
		} {
			method := method
			dsl.Method(method.name, func() {
				dsl.Payload(func() {
					dsl.Token("auth", dsl.String)
				})
				dsl.Result(dsl.Empty)
				dsl.HTTP(func() {
					dsl.GET(method.path)
					dsl.AuthErrorResponses()
					dsl.Response(dsl.StatusOK)
				})
			})
		}
	})
}

var apiScopedAuthErrorReuseDSL = func() {
	var jwt = dsl.JWTSecurity("jwt", func() {
		dsl.Description("Application bearer")
	})
	var unauthorized = dsl.Type("APIScopedUnauthorized", func() {
		dsl.Attribute("reason", dsl.String, func() {
			dsl.Example("session_expired")
		})
		dsl.Required("reason")
	})
	var forbidden = dsl.Type("APIScopedForbidden", func() {
		dsl.Attribute("reason", dsl.String, func() {
			dsl.Example("team_membership_required")
		})
		dsl.Required("reason")
	})

	dsl.API("api-scoped-auth-errors", func() {
		dsl.Security(jwt)
		dsl.Error("unauthorized", unauthorized)
		dsl.Error("forbidden", forbidden)
		dsl.HTTP(func() {
			dsl.Response("unauthorized", dsl.StatusUnauthorized, func() {
				dsl.Description("Session expired")
			})
			dsl.Response("forbidden", dsl.StatusForbidden, func() {
				dsl.Description("Team membership required")
			})
		})
	})
	dsl.Service("apiScopedAuthErrors", func() {
		for _, method := range []struct {
			name string
			path string
		}{
			{name: "first", path: "/api/first"},
			{name: "second", path: "/api/second"},
		} {
			method := method
			dsl.Method(method.name, func() {
				dsl.Payload(func() {
					dsl.Token("auth", dsl.String)
				})
				dsl.Result(dsl.Empty)
				dsl.HTTP(func() {
					dsl.GET(method.path)
					dsl.AuthErrorResponses()
					dsl.Response(dsl.StatusOK)
				})
			})
		}
	})
}
