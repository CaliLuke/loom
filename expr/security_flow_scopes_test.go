package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOAuthFlowScopeOwnership(t *testing.T) {
	scheme := &SchemeExpr{Kind: OAuth2Kind, Scopes: []*ScopeExpr{{Name: "shared", Description: "default"}}, Flows: []*FlowExpr{
		{Kind: PasswordFlowKind},
		{Kind: ImplicitFlowKind, Scopes: []*ScopeExpr{}},
		{Kind: ClientCredentialsFlowKind, Scopes: []*ScopeExpr{{Name: "write", Description: "machine"}, {Name: "shared", Description: "client"}}},
	}}
	require.Equal(t, scheme.Scopes, scheme.FlowScopes(scheme.Flows[0]))
	require.Empty(t, scheme.FlowScopes(scheme.Flows[1]))
	require.NotNil(t, scheme.FlowScopes(scheme.Flows[1]))
	require.Equal(t, "client", scheme.FlowScopes(scheme.Flows[2])[1].Description)
	require.Equal(t, []string{"shared", "write"}, scheme.ScopeNames())
	require.Equal(t, []string{"write"}, (&SecurityExpr{Scopes: []string{"write"}}).RequiredScopes(scheme))
	copy := DupScheme(scheme)
	copy.Scopes[0].Description = "changed"
	copy.Flows[2].Scopes[0].Name = "changed"
	require.Equal(t, "default", scheme.Scopes[0].Description)
	require.Equal(t, "write", scheme.Flows[2].Scopes[0].Name)
	require.NotNil(t, copy.Flows[1].Scopes)
	projected := scheme.FlowScopes(scheme.Flows[0])
	projected[0].Name = "changed"
	require.Equal(t, "shared", scheme.Scopes[0].Name)
}

func TestOAuthFlowScopeDuplicateDescriptions(t *testing.T) {
	for _, local := range []bool{false, true} {
		t.Run(map[bool]string{false: "defaults", true: "local"}[local], func(t *testing.T) {
			inventory := []*ScopeExpr{{Name: "read", Description: "first"}, {Name: "write", Description: "other"}, {Name: "read", Description: "last"}}
			scheme := &SchemeExpr{Scopes: inventory}
			flow := &FlowExpr{}
			if local {
				flow.Scopes = inventory
			}
			got := scheme.FlowScopes(flow)
			require.Equal(t, []*ScopeExpr{{Name: "read", Description: "last"}, {Name: "write", Description: "other"}}, got)
			require.Equal(t, "first", inventory[0].Description)
			got[0].Description = "changed"
			require.Equal(t, "last", inventory[2].Description)
		})
	}
}
