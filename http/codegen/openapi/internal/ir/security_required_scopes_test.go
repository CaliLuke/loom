package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestSecurityBindingRequiredScopeOwnership(t *testing.T) {
	read := &expr.SchemeExpr{SchemeName: "read", Kind: expr.OAuth2Kind, In: "header", Name: "Authorization", Scopes: []*expr.ScopeExpr{{Name: "read"}, {Name: "shared"}}}
	write := &expr.SchemeExpr{SchemeName: "write", Kind: expr.OAuth2Kind, In: "header", Name: "Authorization", Scopes: []*expr.ScopeExpr{{Name: "write"}, {Name: "shared"}}}
	key := &expr.SchemeExpr{SchemeName: "key", Kind: expr.APIKeyKind, In: "header", Name: "X-Key"}
	requirements := []*expr.SecurityExpr{
		{Schemes: []*expr.SchemeExpr{read, write, key}, Scopes: []string{"shared", "write", "read"}},
		{Schemes: []*expr.SchemeExpr{read, write}, Scopes: []string{"read"}},
	}
	bindings, err := newSecurityBindings(requirements)
	require.NoError(t, err)
	got := bindings.requirements(requirements)
	require.Equal(t, []map[string][]string{
		{"read": {"shared", "read"}, "write": {"shared", "write"}, "key": {}},
		{"read": {"read"}, "write": {}},
	}, got)
	got[0]["read"][0] = "changed"
	require.Equal(t, "shared", requirements[0].Scopes[0])
	require.Equal(t, []string{"read"}, got[1]["read"])
}
