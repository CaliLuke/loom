package unionnames

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestResolve(t *testing.T) {
	for _, test := range []struct {
		name  string
		names []string
		want  map[string]Member
	}{
		{"ordinary", []string{"left", "right"}, map[string]Member{"left": {"Left", "Left"}, "right": {"Right", "Right"}}},
		{"reserved and authored suffix", []string{"Kind", "Kind2"}, map[string]Member{"Kind": {"Kind", "Kind3"}, "Kind2": {"Kind2", "Kind2"}}},
		{"accessors", []string{"text", "AsText", "SetText"}, map[string]Member{"text": {"Text", "Text"}, "AsText": {"AsText", "AsText2"}, "SetText": {"SetText", "SetText2"}}},
		{"sanitized and authored suffix", []string{"a_b", "a-b", "AB2"}, map[string]Member{"a-b": {"AB", "AB"}, "a_b": {"AB3", "AB3"}, "AB2": {"AB2", "AB2"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			union := &expr.Union{}
			for _, name := range test.names {
				union.Values = append(union.Values, &expr.NamedAttributeExpr{Name: name})
			}
			require.Equal(t, test.want, Resolve(union))
			slices.Reverse(union.Values)
			require.Equal(t, test.want, Resolve(union), "declaration order must not select names")
		})
	}
}
