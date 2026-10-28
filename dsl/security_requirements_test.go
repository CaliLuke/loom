package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestPerSchemeRequirementsDSL(t *testing.T) {
	design := func(requirements func()) func() {
		return func() {
			OAuth2Security("oauth", func() {
				ClientCredentialsFlow("https://example.com/token", "")
				Scope("read")
			})
			APIKeySecurity("key")
			Service("secure", func() {
				Security("oauth", "key", requirements)
				Method("read", func() {
					Payload(func() {
						AccessToken("token")
						APIKey("key", "key")
					})
				})
			})
		}
	}
	root := expr.RunDSL(t, design(func() {
		ScopeFor("oauth", "read")
		RoleFor("key", "admin")
	}))
	req := root.Services[0].Methods[0].EffectiveRequirements()[0]
	require.Equal(t, map[string][]string{"oauth": {"read"}, "key": {"admin"}}, req.SchemeRequirements)
	for _, tc := range []struct {
		name string
		dsl  func()
		want string
	}{
		{"unknown scope", func() {
			ScopeFor("oauth", "unknown")
		}, "not found in scheme"},
		{"wrong scheme", func() {
			ScopeFor("missing", "read")
		}, "not in this requirement"},
		{"roles on oauth", func() {
			RoleFor("oauth", "admin")
		}, "non-OAuth scheme"},
		{"scopes on key", func() {
			ScopeFor("key", "read")
		}, "OAuth2 scheme"},
		{"nil scheme", func() {
			RoleFor((*expr.SchemeExpr)(nil), "admin")
		}, "not in this requirement"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, design(tc.dsl))
			require.ErrorContains(t, err, tc.want)
		})
	}
}
