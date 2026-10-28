package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestOAuthFlowScopeDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		auth := OAuth2Security("auth", func() {
			AuthorizationCodeFlow("https://example.com/auth", "https://example.com/token", "", func() {
				Scope("read", "User consent")
			})
			ImplicitFlow("https://example.com/auth", "", func() {})
			PasswordFlow("https://example.com/token", "")
			ClientCredentialsFlow("https://example.com/token", "", func() {
				Scope("write", "Machine grant")
			})
			DeviceAuthorizationFlow("https://example.com/device", "https://example.com/token", "", func() {
				Scope("device", "Device grant")
			})
			Scope("default", "Shared default declared after flows")
		})
		Service("flows", func() {
			Method("read", func() {
				Security(auth, func() {
					ScopeFor(auth, "read", "write", "device", "default")
				})
				Payload(func() {
					AccessToken("token")
				})
			})
		})
	})
	scheme := root.Schemes[0]
	require.Len(t, scheme.Flows, 5)
	for i, want := range [][]string{{"read"}, {}, {"default"}, {"write"}, {"device"}} {
		scopes := scheme.FlowScopes(scheme.Flows[i])
		names := make([]string, 0, len(scopes))
		for _, scope := range scopes {
			names = append(names, scope.Name)
		}
		require.ElementsMatch(t, want, names)
	}
}

func TestOAuthFlowScopeBlockErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		body func()
		want string
	}{
		{"multiple blocks", func() {
			ClientCredentialsFlow("https://example.com/token", "", func() {}, func() {})
		}, "too many"},
		{"nil block", func() {
			ClientCredentialsFlow("https://example.com/token", "", nil)
		}, "must not be nil"},
		{"invalid nested DSL", func() {
			ClientCredentialsFlow("https://example.com/token", "", func() {
				RoleFor("auth", "admin")
			})
		}, "invalid use"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				OAuth2Security("auth", tc.body)
			})
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestOAuthFlowScopeDefaultsCannotGrantUnavailableNames(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "shared requirement", true: "scheme requirement"}[explicit], func(t *testing.T) {
			err := expr.RunInvalidDSL(t, func() {
				auth := OAuth2Security("auth", func() {
					Scope("default")
					ClientCredentialsFlow("https://example.com/token", "", func() {})
				})
				Service("flows", func() {
					Method("read", func() {
						Security(auth, func() {
							if explicit {
								ScopeFor(auth, "default")
							} else {
								Scope("default")
							}
						})
						Payload(func() {
							AccessToken("token")
						})
					})
				})
			})
			require.ErrorContains(t, err, "default")
		})
	}
}
