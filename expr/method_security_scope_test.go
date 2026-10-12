package expr_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestAuthenticationScope(t *testing.T) {
	for _, tc := range []struct {
		name, api, service, method string
		want                       []string
		wantSession                bool
	}{
		{"method ordinary replaces service session", "", "session", "basic", []string{"basic"}, false},
		{"method ordinary replaces API session", "session", "", "basic", []string{"basic"}, false},
		{"service ordinary replaces API session", "session", "basic", "", []string{"basic"}, false},
		{"method session replaces service ordinary", "", "basic", "session", []string{"jwt", "key"}, true},
		{"method session replaces API ordinary", "basic", "", "session", []string{"jwt", "key"}, true},
		{"service session replaces API ordinary", "basic", "session", "", []string{"jwt", "key"}, true},
		{"same scope alternatives", "", "", "both", []string{"basic", "jwt", "key"}, true},
		{"anonymous replaces inherited session", "", "session", "anonymous", []string{"anonymous"}, false},
		{"anonymous alongside session", "", "", "optional-session", []string{"anonymous", "jwt", "key"}, true},
		{"NoSecurity overrides sessions", "session", "", "none", nil, false},
		{"session inheritance", "session", "", "", []string{"jwt", "key"}, true},
		{"ordinary inheritance", "basic", "", "", []string{"basic"}, false},
		{"empty", "", "", "", nil, false},
		{"redundant anonymous", "session", "", "none-anonymous", nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				basic := BasicAuthSecurity("basic")
				jwt := JWTSecurity("jwt")
				key := APIKeySecurity("key")
				session := SessionAuth("session", func() {
					BearerTransport(jwt, "auth")
					CookieTransport(key, "cookie", func() {
						CookieName("session")
					})
				})
				declare := func(kind string) {
					switch kind {
					case "basic":
						Security(basic)
					case "session":
						SessionSecurity(session)
					case "both":
						Security(basic)
						SessionSecurity(session)
					case "anonymous":
						Security()
					case "optional-session":
						Security()
						SessionSecurity(session)
					case "none":
						NoSecurity()
					case "none-anonymous":
						NoSecurity()
						Security()
					}
				}
				API("scope", func() {
					declare(tc.api)
				})
				Service("scope", func() {
					declare(tc.service)
					Method("read", func() {
						declare(tc.method)
						Payload(func() {
							Attribute("message", String)
							if len(tc.want) > 0 && tc.want[0] == "basic" {
								Username("user")
								Password("password")
							}
						})
						HTTP(func() {
							GET("/")
						})
					})
				})
			})
			method := root.Service("scope").Method("read")
			require.Equal(t, tc.want, requirementNames(method.Requirements))
			require.Equal(t, tc.want, requirementNames(method.EffectiveRequirements()))
			require.Equal(t, tc.wantSession, len(method.EffectiveSessionAuths()) > 0)
			require.Equal(t, tc.wantSession, method.Payload.Find("auth") != nil)
			endpoint := root.API.HTTP.Service("scope").Endpoint("read")
			require.Equal(t, tc.want, requirementNames(endpoint.Requirements))
		})
	}
}

func TestNoSecurityConflict(t *testing.T) {
	for _, session := range []bool{false, true} {
		for _, noSecurityFirst := range []bool{false, true} {
			t.Run(map[bool]string{false: "ordinary", true: "session"}[session]+map[bool]string{false: "-last", true: "-first"}[noSecurityFirst], func(t *testing.T) {
				err := expr.RunInvalidDSL(t, func() {
					basic := BasicAuthSecurity("basic")
					jwt := JWTSecurity("jwt")
					appSession := SessionAuth("session", func() {
						BearerTransport(jwt, "auth")
					})
					Service("scope", func() {
						Method("read", func() {
							if noSecurityFirst {
								NoSecurity()
							}
							if session {
								SessionSecurity(appSession)
							} else {
								Security(basic)
							}
							if !noSecurityFirst {
								NoSecurity()
							}
						})
					})
				})
				require.ErrorContains(t, err, "NoSecurity cannot be combined with Security or SessionSecurity")
			})
		}
	}
}

func requirementNames(requirements []*expr.SecurityExpr) []string {
	var names []string
	for _, requirement := range requirements {
		if len(requirement.Schemes) == 0 {
			names = append(names, "anonymous")
		}
		for _, scheme := range requirement.Schemes {
			names = append(names, scheme.SchemeName)
		}
	}
	return names
}

func TestAuthenticationScopeQueries(t *testing.T) {
	expr.SetupTestDSL(t)
	basic := &expr.SchemeExpr{Kind: expr.BasicAuthKind, SchemeName: "basic"}
	jwt := &expr.SchemeExpr{Kind: expr.JWTKind, SchemeName: "jwt"}
	session := &expr.SessionAuthExpr{
		Name: "session",
		Transports: []*expr.SessionTransportExpr{{
			Kind: expr.SessionBearerTransportKind, Scheme: jwt, FieldName: "auth",
		}},
	}
	outer := &expr.ServiceExpr{SessionAuths: []*expr.SessionAuthExpr{session}}
	method := &expr.MethodExpr{
		Name: "query", Service: outer,
		Requirements: []*expr.SecurityExpr{{Schemes: []*expr.SchemeExpr{basic, jwt}, Scopes: []string{"read"}}, {}},
	}
	for range 2 {
		requirements := method.EffectiveRequirements()
		require.Len(t, requirements, 2)
		require.Len(t, requirements[0].Schemes, 2)
		require.Equal(t, []string{"read"}, requirements[0].Scopes)
		require.Empty(t, requirements[1].Schemes)
		require.Empty(t, method.EffectiveSessionAuths())
		requirements[0].Schemes[0].Name = "transport-name"
		require.Empty(t, basic.Name)
		require.Len(t, outer.SessionAuths, 1)
	}
	method.Finalize()
	require.Equal(t, []string{"basic", "jwt", "anonymous"}, requirementNames(method.Requirements))
	require.Empty(t, method.SessionAuths)
	method.Finalize()
	require.Equal(t, []string{"basic", "jwt", "anonymous"}, requirementNames(method.EffectiveRequirements()))
}
