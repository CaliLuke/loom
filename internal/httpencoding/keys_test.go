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
			err := CheckNamespaces(tc.loc, map[string]KeySet{"left": tc.left, "right": tc.right})
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
