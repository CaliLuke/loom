package httpencoding

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckNamespaces(t *testing.T) {
	for _, tc := range []struct {
		name        string
		loc         Location
		left, right KeySet
		accepted    bool
	}{
		{"separate literals", NamedQuery, KeySet{Literals: []string{"a"}}, KeySet{Literals: []string{"b"}}, true},
		{"literal collision", NamedQuery, KeySet{Literals: []string{"a"}}, KeySet{Literals: []string{"a"}}, false},
		{"literal prefix", NamedQuery, KeySet{Literals: []string{"m[x]"}}, KeySet{Prefixes: []string{"m"}}, false},
		{"prefix collision", NamedQuery, KeySet{Prefixes: []string{"m"}}, KeySet{Prefixes: []string{"m"}}, false},
		{"two residuals", Cookie, KeySet{Residual: true}, KeySet{Residual: true}, false},
		{"residual and credential", Cookie, KeySet{Residual: true}, KeySet{Literals: []string{"sid"}}, true},
		{"residual and prefix", FormField, KeySet{Residual: true}, KeySet{Prefixes: []string{"m"}}, true},
		{"whole query exclusive", WholeQuery, KeySet{Residual: true}, KeySet{Literals: []string{"a"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckNamespaces(tc.loc, []Binding{{Name: "left", Keys: tc.left}, {Name: "right", Keys: tc.right}})
			if tc.accepted {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "left")
				require.ErrorContains(t, err, "right")
			}
		})
	}
	t.Run("deep object brackets", func(t *testing.T) {
		for _, value := range []struct {
			name  string
			shape BranchShape
		}{
			{"n[x]", BranchShape{Shape: ScalarMap}},
			{"n", BranchShape{Shape: Object, Properties: []PropertyFact{{WireName: "x[y]"}}}},
		} {
			_, _, err := Resolve(NamedQuery, value.name, value.shape, Serialization{Style: "deepObject"})
			require.Error(t, err)
		}
	})
}

func TestCheckNamespacesPreservesBindingIdentity(t *testing.T) {
	for _, tc := range []struct {
		name        string
		location    Location
		left, right KeySet
		accepted    bool
	}{
		{"whole query", WholeQuery, KeySet{Literals: []string{"n"}}, KeySet{Residual: true}, false},
		{"overlapping literals", NamedQuery, KeySet{Literals: []string{"n"}}, KeySet{Literals: []string{"n"}}, false},
		{"disjoint names", NamedQuery, KeySet{Literals: []string{"n"}}, KeySet{Literals: []string{"m"}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bindings := []Binding{{Name: "same label", Keys: tc.left}, {Name: "same label", Keys: tc.right}}
			err := CheckNamespaces(tc.location, bindings)
			if tc.accepted {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, `bindings "same label" and "same label"`)
			}
		})
	}
	t.Run("stable diagnostics without mutating claims", func(t *testing.T) {
		keys := KeySet{Literals: []string{"n"}}
		bindings := []Binding{{Name: "z", Keys: keys}, {Name: "a", Keys: keys}}
		require.EqualError(t, CheckNamespaces(NamedQuery, bindings), `bindings "a" and "z" claim overlapping parameter namespaces`)
		require.Equal(t, "z", bindings[0].Name)
	})
}
