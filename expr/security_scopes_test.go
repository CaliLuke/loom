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

func TestPerSchemeRequirementOwnership(t *testing.T) {
	left := &SchemeExpr{SchemeName: "left", Kind: OAuth2Kind, Name: "Authorization", Scopes: []*ScopeExpr{{Name: "read"}}}
	right := &SchemeExpr{SchemeName: "right", Kind: OAuth2Kind, Scopes: left.Scopes}
	key := &SchemeExpr{SchemeName: "key", Kind: APIKeyKind}
	req := &SecurityExpr{Schemes: []*SchemeExpr{left, right, key}, SchemeRequirements: map[string][]string{"left": {"read"}, "right": {}, "key": {"admin"}}}
	require.Equal(t, []string{"read"}, req.RequiredScopes(left))
	require.Empty(t, req.RequiredScopes(right), "a shared inventory must not share requirements")
	require.Equal(t, []string{"admin"}, req.RequiredScopes(key), "roles need no inventory")
	copy := DupRequirement(req)
	require.True(t, requirementEqual(req, copy))
	require.Equal(t, "Authorization", copy.Schemes[0].Name)
	copy.SchemeRequirements["left"][0] = "write"
	require.False(t, requirementEqual(req, copy))
	require.Equal(t, []string{"read"}, req.RequiredScopes(left))
	req.Scopes = []string{"read"}
	require.Equal(t, []string{"read"}, req.RequiredScopes(left), "shared and explicit requirements are additive without duplicates")
	require.Equal(t, []string{"read"}, req.RequiredScopes(right))
}
