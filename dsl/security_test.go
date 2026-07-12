package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestSecuritySchemesDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		basic := BasicAuthSecurity("basic", func() {
			Description("Use your username and password")
		})
		oauth2 := OAuth2Security("oauth2", func() {
			AuthorizationCodeFlow("/authorize", "/token", "/refresh")
			ImplicitFlow("/authorize", "/refresh")
			PasswordFlow("/token", "/refresh")
			ClientCredentialsFlow("/token", "/refresh")
			Scope("api:read", "Read access")
			Scope("api:write", "Write access")
		})
		jwt := JWTSecurity("jwt", func() {
			Scope("api:read", "Read access")
		})
		APIKeySecurity("api_key", func() {
			Description("Shared secret")
		})

		Service("secured", func() {
			Method("login", func() {
				Security(basic)
				Payload(func() {
					Username("user", String)
					Password("pass", String)
				})
			})
			Method("read", func() {
				Security(jwt, func() {
					Scope("api:read")
				})
				Payload(func() {
					Token("token", String)
				})
			})
			Method("lookup", func() {
				Security("api_key")
				Payload(func() {
					APIKey("api_key", "key", String)
				})
			})
			Method("write", func() {
				Security(oauth2, func() {
					Scope("api:write")
				})
				Payload(func() {
					AccessToken("access", String)
				})
			})
			Method("health", func() {
				NoSecurity()
			})
		})
	})

	require.Len(t, root.Schemes, 4)

	basic := root.Schemes[0]
	require.Equal(t, expr.BasicAuthKind, basic.Kind)
	require.Equal(t, "basic", basic.SchemeName)
	require.Equal(t, "Use your username and password", basic.Description)

	oauth2 := root.Schemes[1]
	require.Equal(t, expr.OAuth2Kind, oauth2.Kind)
	require.Equal(t, "oauth2", oauth2.SchemeName)
	require.Len(t, oauth2.Flows, 4)
	require.Equal(t, expr.AuthorizationCodeFlowKind, oauth2.Flows[0].Kind)
	require.Equal(t, "/authorize", oauth2.Flows[0].AuthorizationURL)
	require.Equal(t, "/token", oauth2.Flows[0].TokenURL)
	require.Equal(t, "/refresh", oauth2.Flows[0].RefreshURL)
	require.Equal(t, expr.ImplicitFlowKind, oauth2.Flows[1].Kind)
	require.Equal(t, "/authorize", oauth2.Flows[1].AuthorizationURL)
	require.Equal(t, "/refresh", oauth2.Flows[1].RefreshURL)
	require.Equal(t, expr.PasswordFlowKind, oauth2.Flows[2].Kind)
	require.Equal(t, "/token", oauth2.Flows[2].TokenURL)
	require.Equal(t, expr.ClientCredentialsFlowKind, oauth2.Flows[3].Kind)
	require.Equal(t, "/token", oauth2.Flows[3].TokenURL)
	require.Len(t, oauth2.Scopes, 2)
	require.Equal(t, "api:read", oauth2.Scopes[0].Name)
	require.Equal(t, "Read access", oauth2.Scopes[0].Description)
	require.Equal(t, "api:write", oauth2.Scopes[1].Name)

	jwt := root.Schemes[2]
	require.Equal(t, expr.JWTKind, jwt.Kind)
	require.Equal(t, "jwt", jwt.SchemeName)
	require.Equal(t, "header", jwt.In)
	require.Equal(t, "Authorization", jwt.Name)
	require.Len(t, jwt.Scopes, 1)
	require.Equal(t, "api:read", jwt.Scopes[0].Name)

	apiKey := root.Schemes[3]
	require.Equal(t, expr.APIKeyKind, apiKey.Kind)
	require.Equal(t, "api_key", apiKey.SchemeName)
	require.Equal(t, "Shared secret", apiKey.Description)

	svc := root.Service("secured")
	require.NotNil(t, svc)

	login := svc.Method("login")
	require.NotNil(t, login)
	require.Len(t, login.Requirements, 1)
	require.Len(t, login.Requirements[0].Schemes, 1)
	require.Equal(t, expr.BasicAuthKind, login.Requirements[0].Schemes[0].Kind)
	require.Equal(t, "basic", login.Requirements[0].Schemes[0].SchemeName)

	read := svc.Method("read")
	require.NotNil(t, read)
	require.Len(t, read.Requirements, 1)
	require.Equal(t, expr.JWTKind, read.Requirements[0].Schemes[0].Kind)
	require.Equal(t, []string{"api:read"}, read.Requirements[0].Scopes)

	lookup := svc.Method("lookup")
	require.NotNil(t, lookup)
	require.Len(t, lookup.Requirements, 1)
	require.Equal(t, expr.APIKeyKind, lookup.Requirements[0].Schemes[0].Kind)
	require.Equal(t, "api_key", lookup.Requirements[0].Schemes[0].SchemeName)

	write := svc.Method("write")
	require.NotNil(t, write)
	require.Len(t, write.Requirements, 1)
	require.Equal(t, expr.OAuth2Kind, write.Requirements[0].Schemes[0].Kind)
	require.Equal(t, []string{"api:write"}, write.Requirements[0].Scopes)

	health := svc.Method("health")
	require.NotNil(t, health)
	// Finalize clears the NoKind requirement and keeps the meta marker.
	require.Empty(t, health.Requirements)
	require.Contains(t, health.Meta, "security:no")
}

func TestSecurityAPIAndServiceLevelDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		jwt := JWTSecurity("jwt", func() {
			Scope("api:read", "Read access")
		})

		API("secured_api", func() {
			Security(jwt, func() {
				Scope("api:read")
			})
		})

		Service("svc", func() {
			Security(jwt)
			Method("show", func() {
				Payload(func() {
					Token("token", String)
				})
			})
		})
	})

	require.Len(t, root.API.Requirements, 1)
	require.Len(t, root.API.Requirements[0].Schemes, 1)
	require.Equal(t, expr.JWTKind, root.API.Requirements[0].Schemes[0].Kind)
	require.Equal(t, []string{"api:read"}, root.API.Requirements[0].Scopes)

	svc := root.Service("svc")
	require.NotNil(t, svc)
	require.Len(t, svc.Requirements, 1)
	require.Equal(t, expr.JWTKind, svc.Requirements[0].Schemes[0].Kind)
	require.Equal(t, "jwt", svc.Requirements[0].Schemes[0].SchemeName)
}

func TestSecurityDSLErrors(t *testing.T) {
	cases := []struct {
		name    string
		dsl     func()
		wantErr string
	}{
		{
			name: "unknown scheme name",
			dsl: func() {
				Service("svc", func() {
					Method("show", func() {
						Security("unknown")
					})
				})
			},
			wantErr: `security scheme "unknown" not found`,
		},
		{
			name: "redefined scheme",
			dsl: func() {
				BasicAuthSecurity("dup")
				BasicAuthSecurity("dup")
			},
			wantErr: `cannot redefine security scheme with name "dup"`,
		},
		{
			name: "scheme defined outside top level",
			dsl: func() {
				Service("svc", func() {
					BasicAuthSecurity("basic")
				})
			},
			wantErr: "invalid use of BasicAuthSecurity",
		},
		{
			name: "flow in non-oauth2 scheme",
			dsl: func() {
				JWTSecurity("jwt", func() {
					ImplicitFlow("/authorize", "/refresh")
				})
			},
			wantErr: "cannot specify flow for non-oauth2 security scheme.",
		},
		{
			name: "jwt requirement without token attribute",
			dsl: func() {
				jwt := JWTSecurity("jwt")
				Service("svc", func() {
					Method("show", func() {
						Security(jwt)
						Payload(func() {
							Attribute("id", String)
						})
					})
				})
			},
			wantErr: "does not define a JWT attribute, use Token to define one",
		},
		{
			name: "basic auth requirement without username attribute",
			dsl: func() {
				basic := BasicAuthSecurity("basic")
				Service("svc", func() {
					Method("login", func() {
						Security(basic)
						Payload(func() {
							Password("pass", String)
						})
					})
				})
			},
			wantErr: "does not define a username attribute, use Username to define one",
		},
		{
			name: "required scope not defined by any scheme",
			dsl: func() {
				jwt := JWTSecurity("jwt", func() {
					Scope("api:read", "Read access")
				})
				Service("svc", func() {
					Method("show", func() {
						Security(jwt, func() {
							Scope("api:missing")
						})
						Payload(func() {
							Token("token", String)
						})
					})
				})
			},
			wantErr: `security scope "api:missing" not found in any of the security schemes.`,
		},
		{
			name: "token attribute without jwt scheme",
			dsl: func() {
				Service("svc", func() {
					Method("show", func() {
						Payload(func() {
							Token("token", String)
						})
					})
				})
			},
			wantErr: "defines a JWT token attribute, but no JWT auth security scheme exist",
		},
		{
			name: "no security outside method",
			dsl: func() {
				Service("svc", func() {
					NoSecurity()
				})
			},
			wantErr: "invalid use of NoSecurity",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.dsl)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
