package ir

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestSecurityBindingNameReservations(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		schemes := []*expr.SchemeExpr{
			{SchemeName: "auth", Kind: expr.APIKeyKind, In: "query", Name: "a/b"},
			{SchemeName: "auth", Kind: expr.APIKeyKind, In: "query", Name: "a?b"},
			{SchemeName: "auth_query_a_b", Kind: expr.APIKeyKind, In: "cookie", Name: "session"},
			{SchemeName: "auth_query_a_b_2", Kind: expr.APIKeyKind, In: "header", Name: "X-Token"},
		}
		if reverse {
			slices.Reverse(schemes)
		}
		bindings, err := newSecurityBindings([]*expr.SecurityExpr{{Schemes: schemes}})
		require.NoError(t, err)
		require.Len(t, bindings.schemes, 4)
		require.Equal(t, "a/b", bindings.schemes["auth_query_a_b_3"].Name)
		require.Equal(t, "a?b", bindings.schemes["auth_query_a_b_4"].Name)
		require.Equal(t, "session", bindings.schemes["auth_query_a_b"].Name)
		require.Equal(t, "X-Token", bindings.schemes["auth_query_a_b_2"].Name)
	}
}

func TestSecurityBindingRequirementComposition(t *testing.T) {
	apiKey := &expr.SchemeExpr{SchemeName: "key", Kind: expr.APIKeyKind, In: "query", Name: "key"}
	oauth := &expr.SchemeExpr{SchemeName: "oauth", Kind: expr.OAuth2Kind, In: "header", Name: "Authorization"}
	jwt := &expr.SchemeExpr{SchemeName: "jwt", Kind: expr.JWTKind, In: "header", Name: "authorization"}
	requirements := []*expr.SecurityExpr{
		{Schemes: []*expr.SchemeExpr{apiKey, oauth}, Scopes: []string{"read", "write"}},
		{Schemes: []*expr.SchemeExpr{jwt}, Scopes: []string{"private"}},
		{Schemes: []*expr.SchemeExpr{{Kind: expr.NoKind}}},
	}
	bindings, err := newSecurityBindings(requirements)
	require.NoError(t, err)
	got := bindings.requirements(requirements)
	require.Equal(t, []map[string][]string{
		{"key": {}, "oauth": {"read", "write"}},
		{"jwt": {}},
		{},
	}, got)
	require.Equal(t, "Authorization", bindings.schemes["jwt"].Name)
	got[0]["oauth"][0] = "changed"
	require.Equal(t, "read", requirements[0].Scopes[0], "projection must not alias authored scopes")
}

func TestSecurityBindingsShareHeaderCaseVariants(t *testing.T) {
	requirements := []*expr.SecurityExpr{{Schemes: []*expr.SchemeExpr{
		{SchemeName: "auth", Kind: expr.JWTKind, In: "header", Name: "Authorization"},
		{SchemeName: "auth", Kind: expr.JWTKind, In: "header", Name: "AUTHORIZATION"},
	}}}
	bindings, err := newSecurityBindings(requirements)
	require.NoError(t, err)
	require.Len(t, bindings.schemes, 1)
	require.Equal(t, "Authorization", bindings.schemes["auth"].Name)
}
