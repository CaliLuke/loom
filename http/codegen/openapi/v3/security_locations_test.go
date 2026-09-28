package openapiv3_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

var credentialLocations = []struct {
	in, name string
}{
	{"header", "Authorization"},
	{"header", "X-Token"},
	{"query", "token"},
	{"cookie", "session"},
}

func TestSecurityCredentialLocations(t *testing.T) {
	for _, jwt := range []bool{false, true} {
		for _, inherited := range []bool{false, true} {
			var previous map[string]*openapiv3.SecuritySchemeRef
			for _, reverse := range []bool{false, true} {
				t.Run(fmt.Sprintf("jwt=%t/inherited=%t/reverse=%t", jwt, inherited, reverse), func(t *testing.T) {
					root := httpgen.RunHTTPDSL(t, securityLocationsDSL(jwt, inherited, reverse, ""))
					spec := openapiv3.New(root)
					require.NotNil(t, spec)
					for i, location := range credentialLocations {
						operation := spec.Paths[fmt.Sprintf("/m%d", i)].Get
						require.Len(t, operation.Security, 1)
						require.Len(t, operation.Security[0], 1)
						for name, scopes := range operation.Security[0] {
							require.Empty(t, scopes)
							ref := spec.Components.SecuritySchemes[name]
							require.NotNil(t, ref)
							if jwt && i == 0 {
								require.Equal(t, "http", ref.Value.Type)
								require.Equal(t, "bearer", ref.Value.Scheme)
							} else {
								require.Equal(t, "apiKey", ref.Value.Type)
								require.Equal(t, location.in, ref.Value.In)
								require.Equal(t, location.name, ref.Value.Name)
							}
						}
					}
					require.Len(t, spec.Components.SecuritySchemes, len(credentialLocations))
					if inherited {
						require.Len(t, spec.Security, 1)
						for name := range spec.Security[0] {
							scheme := spec.Components.SecuritySchemes[name].Value
							if jwt {
								require.Equal(t, "bearer", scheme.Scheme)
							} else {
								require.Equal(t, "Authorization", scheme.Name)
							}
						}
						require.NotNil(t, spec.Paths["/public"].Get.Security)
						require.Empty(t, spec.Paths["/public"].Get.Security)
					}
					if reverse {
						require.Equal(t, previous, spec.Components.SecuritySchemes, "endpoint order must not select credential locations or component names")
					} else {
						previous = spec.Components.SecuritySchemes
					}
				})
			}
		}
	}
}

func TestSecurityLocationProfiles(t *testing.T) {
	for _, target := range []string{"3.1", "3.2"} {
		t.Run(target, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, testdata.SecurityDSL)
			root.API.Meta = expr.MetaExpr{"openapi:version": {target}}
			files, err := openapiv3.Files(root)
			require.NoError(t, err)
			version := openapiv3.OpenAPIVersion
			if target == "3.1" {
				version = openapiv3.OpenAPICompatibilityVersion
			}
			for _, file := range files {
				buf := renderSection(t, file.AllSections()[0])
				validateOpenAPIVersion(t, buf.Bytes(), version)
			}
			spec := openapiv3.New(root)
			require.Equal(t, []map[string][]string{
				{"api_key_header_X-Key": {}},
				{"oauth2": {"api:read", "api:write"}},
			}, spec.Paths["/"].Post.Security)
			require.NotNil(t, spec.Components.SecuritySchemes["oauth2"].Value.Flows.AuthorizationCode)
			require.Len(t, spec.Paths["/"].Get.Security[0], 3, "AND requirements remain combined")
		})
	}
}

func TestSecurityLocationExternalURI(t *testing.T) {
	for _, tc := range []struct {
		name string
		uris []string
		want string
	}{
		{"single", []string{"https://example.com/auth"}, "https://example.com/auth"},
		{"override", []string{"https://example.com/old", "https://example.com/new"}, "https://example.com/new"},
		{"clear", []string{"https://example.com/old", ""}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, securityLocationsDSL(true, true, false, tc.uris...))
			spec := openapiv3.New(root)
			require.NotNil(t, spec)
			requirements := make([][]map[string][]string, 0, 1+len(credentialLocations))
			requirements = append(requirements, spec.Security)
			for i := range credentialLocations {
				requirements = append(requirements, spec.Paths[fmt.Sprintf("/m%d", i)].Get.Security)
			}
			if tc.want != "" {
				require.Empty(t, spec.Components.SecuritySchemes)
				for _, requirement := range requirements {
					require.Equal(t, []map[string][]string{{tc.want: {}}}, requirement)
				}
			} else {
				require.Len(t, spec.Components.SecuritySchemes, len(credentialLocations))
				for _, requirement := range requirements {
					require.Len(t, requirement, 1)
					require.Len(t, requirement[0], 1)
					for name := range requirement[0] {
						require.Contains(t, spec.Components.SecuritySchemes, name)
					}
				}
			}
		})
	}
}

func TestSecurityLocationAPISessionDefaults(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, func() {
		cookie := APIKeySecurity("session_cookie")
		session := SessionAuth("session", func() {
			CookieTransport(cookie, "", func() {
				CookieName("__Host-session")
			})
		})
		API("session-defaults", func() {
			SessionSecurity(session)
		})
		Service("svc", func() {
			Files("/assets", "public")
			Method("m", func() {
				HTTP(func() {
					GET("/m")
				})
			})
		})
	})
	spec := openapiv3.New(root)
	require.NotNil(t, spec)
	require.Len(t, spec.Components.SecuritySchemes, 1)
	scheme := spec.Components.SecuritySchemes["session_cookie"].Value
	require.Equal(t, "cookie", scheme.In)
	require.Equal(t, "__Host-session", scheme.Name)
	for _, requirements := range [][]map[string][]string{
		spec.Security, spec.Paths["/m"].Get.Security, spec.Paths["/assets"].Get.Security,
	} {
		require.Equal(t, []map[string][]string{{"session_cookie": {}}}, requirements)
	}
}

func TestUnsupportedSecurityCredentialLocations(t *testing.T) {
	for _, tc := range []struct {
		name, kind, in, want string
	}{
		{"jwt path", "jwt", "path", "path"},
		{"oauth query", "oauth", "query", "Authorization"},
		{"oauth custom header", "oauth", "header", "Authorization"},
		{"oauth cookie", "oauth", "cookie", "Authorization"},
		{"oauth path", "oauth", "path", "path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := httpgen.RunHTTPDSL(t, func() {
				var scheme *expr.SchemeExpr
				if tc.kind == "jwt" {
					scheme = JWTSecurity("auth")
				} else {
					scheme = OAuth2Security("auth", func() {
						ClientCredentialsFlow("https://example.com/token", "")
					})
				}
				Service("svc", func() {
					Method("m", func() {
						Security(scheme)
						Payload(func() {
							if tc.kind == "jwt" {
								Token("credential", String)
							} else {
								AccessToken("credential", String)
							}
							Required("credential")
						})
						HTTP(func() {
							if tc.in == "path" {
								GET("/m/{credential}")
							} else {
								GET("/m")
							}
							mapCredential(tc.in, "credential")
						})
					})
				})
			})
			_, err := openapiv3.Files(root)
			require.ErrorContains(t, err, tc.want)
			endpoint := root.API.HTTP.Services[0].HTTPEndpoints[0]
			if endpoint.Meta == nil {
				endpoint.Meta = expr.MetaExpr{}
			}
			endpoint.Meta["openapi:generate"] = []string{"false"}
			_, err = openapiv3.Files(root)
			require.NoError(t, err, "excluding the endpoint must retain transport support")
		})
	}
}

func securityLocationsDSL(jwt, inherited, reverse bool, uris ...string) func() {
	return func() {
		var scheme *expr.SchemeExpr
		metadata := func() {
			for _, uri := range uris {
				Meta("openapi:security:uri", uri)
			}
		}
		if jwt {
			scheme = JWTSecurity("auth", metadata)
		} else {
			scheme = APIKeySecurity("auth", metadata)
		}
		API("security-locations", func() {
			if inherited {
				Security(scheme)
			}
		})
		order := []int{0, 1, 2, 3}
		if reverse {
			slices.Reverse(order)
		}
		Service("svc", func() {
			Method("public", func() {
				NoSecurity()
				HTTP(func() {
					GET("/public")
				})
			})
			for _, i := range order {
				location := credentialLocations[i]
				Method(fmt.Sprintf("m%d", i), func() {
					if !inherited {
						Security(scheme)
					}
					Payload(func() {
						if jwt {
							Token("credential", String)
						} else {
							APIKey("auth", "credential", String)
						}
						Required("credential")
					})
					HTTP(func() {
						GET(fmt.Sprintf("/m%d", i))
						mapCredential(location.in, location.name)
					})
				})
			}
		})
	}
}

func mapCredential(in, name string) {
	mapping := "credential:" + name
	switch in {
	case "header":
		Header(mapping)
	case "cookie":
		Cookie(mapping)
	default:
		Param(mapping)
	}
}
