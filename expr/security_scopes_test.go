package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequiredScopeProjection(t *testing.T) {
	for _, kind := range []SchemeKind{BasicAuthKind, APIKeyKind, JWTKind, OAuth2Kind} {
		t.Run(kind.String(), func(t *testing.T) {
			scheme := &SchemeExpr{Kind: kind, Scopes: []*ScopeExpr{{Name: "read"}, {Name: "shared"}}}
			first := &SecurityExpr{Scopes: []string{"shared", "other", "read"}}
			second := &SecurityExpr{Scopes: []string{"read"}}
			got := first.RequiredScopes(scheme)
			require.Equal(t, []string{"shared", "read"}, got)
			require.Equal(t, []string{"read"}, second.RequiredScopes(scheme))
			got[0] = "changed"
			require.Equal(t, []string{"shared", "other", "read"}, first.Scopes)
			require.Equal(t, "read", scheme.Scopes[0].Name)
			require.Equal(t, []string{"read"}, second.RequiredScopes(scheme))
			require.Empty(t, first.RequiredScopes(&SchemeExpr{Kind: kind}))
			require.Empty(t, (&SecurityExpr{}).RequiredScopes(scheme))
		})
	}
}
