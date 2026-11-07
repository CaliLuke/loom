package loom

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONShapeUnionClosure(t *testing.T) {
	integer := &JSONShape{Kind: "integer"}
	x := &JSONShape{Kind: "object", Fields: []JSONShapeField{{Name: "x", Required: true, Shape: integer}}}
	open := &JSONShape{Kind: "union", Branches: []*JSONShape{x, {Kind: "string"}}}
	for _, tc := range []struct {
		name, wire string
		branches   []*JSONShape
		want       bool
	}{
		{"declared", `{"x":1}`, open.Branches, true},
		{"undeclared", `{"x":1,"z":2}`, open.Branches, false},
		{"scalar", `"text"`, open.Branches, true},
		{"ambiguous before closure", `{"x":1}`, []*JSONShape{x, {Kind: "object"}}, false},
		{"any empty", `{}`, []*JSONShape{{Kind: "any"}, {Kind: "string"}}, true},
		{"any member", `{"z":1}`, []*JSONShape{{Kind: "any"}, {Kind: "string"}}, false},
		{"map member", `{"z":1}`, []*JSONShape{{Kind: "map", Child: integer}, {Kind: "string"}}, true},
		{"explicit open", `{"z":1}`, []*JSONShape{{Kind: "object", Additional: &JSONShape{Kind: "any"}}, {Kind: "string"}}, true},
		{"typed extras", `{"z":1}`, []*JSONShape{{Kind: "object", Additional: integer}, {Kind: "string"}}, true},
		{"nested undeclared", `{"x":1,"z":2}`, []*JSONShape{open, {Kind: "boolean"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, alias := range []bool{false, true} {
				shape := &JSONShape{Kind: "union", Branches: tc.branches}
				if alias {
					shape = &JSONShape{Kind: "alias", Child: shape}
				}
				shape.UnevaluatedClosed = true
				matched, err := shape.Match([]byte(tc.wire))
				require.NoError(t, err)
				require.Equal(t, tc.want, matched, "alias=%v", alias)
			}
		})
	}
	matched, err := open.Match([]byte(`{"x":1,"z":2}`))
	require.NoError(t, err)
	require.True(t, matched, "local closure must not close a shared branch")
}

func TestJSONShapeClosureRejectsNonUnionDescription(t *testing.T) {
	for _, shape := range []*JSONShape{
		{Kind: "object", UnevaluatedClosed: true},
		{Kind: "alias", Child: &JSONShape{Kind: "string"}, UnevaluatedClosed: true},
	} {
		_, err := shape.Match([]byte(`null`))
		require.ErrorContains(t, err, "union")
	}
}

func TestJSONShapeRecursiveClosure(t *testing.T) {
	choice := &JSONShape{Kind: "union", UnevaluatedClosed: true}
	choice.Branches = []*JSONShape{
		{Kind: "object", Fields: []JSONShapeField{
			{Name: "x", Required: true, Shape: &JSONShape{Kind: "integer"}},
			{Name: "next", Shape: choice},
		}},
		{Kind: "string"},
	}
	for _, depth := range []int{8, 16, 32, 64} {
		prefix := strings.Repeat(`{"x":1,"next":`, depth)
		suffix := strings.Repeat(`}`, depth)
		valid := []byte(prefix + `"end"` + suffix)
		matched, err := choice.Match(valid)
		require.NoError(t, err)
		require.True(t, matched)
		names, err := choice.Evaluated(valid)
		require.NoError(t, err)
		require.Equal(t, []string{"next", "x"}, names)
		invalid := []byte(prefix + `{"x":1,"extra":2}` + suffix)
		matched, err = choice.Match(invalid)
		require.NoError(t, err)
		require.False(t, matched)
		names, err = choice.Evaluated(invalid)
		require.NoError(t, err)
		require.Empty(t, names, "failed nested branches contribute no annotations")
	}
}
