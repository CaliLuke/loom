package loom

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONShapeNestedUnions(t *testing.T) {
	first := &JSONShape{Kind: "object", Fields: []JSONShapeField{{Name: "name", Required: true, Shape: &JSONShape{Kind: "string"}}}}
	second := &JSONShape{Kind: "object", Fields: []JSONShapeField{{Name: "group", Required: true, Shape: &JSONShape{Kind: "string"}}}}
	choice := &JSONShape{Kind: "union", Branches: []*JSONShape{first, second}}
	for _, tc := range []struct {
		name  string
		shape *JSONShape
		wire  string
		want  bool
	}{
		{"first", choice, `{"name":"person"}`, true},
		{"second", choice, `{"group":"team"}`, true},
		{"zero", choice, `{}`, false},
		{"multiple", choice, `{"name":"person","group":"team"}`, false},
		{"null", choice, `null`, false},
		{"empty array", &JSONShape{Kind: "array", Child: choice}, `[]`, true},
		{"mixed rows", &JSONShape{Kind: "array", Child: choice}, `[{"name":"person"},{"group":"team"}]`, true},
		{"invalid row", &JSONShape{Kind: "array", Child: choice}, `[{"name":"person"},{}]`, false},
		{"ambiguous row", &JSONShape{Kind: "array", Child: choice}, `[{"name":"person","group":"team"}]`, false},
		{"map values", &JSONShape{Kind: "map", Child: choice}, `{"a":{"name":"person"},"b":{"group":"team"}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			matched, err := tc.shape.Match([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, matched)
		})
	}
	choice.Nullable = true
	matched, err := choice.Match([]byte(`null`))
	require.NoError(t, err)
	require.True(t, matched)
}

func TestJSONShapeUnionRejectsMalformedDescriptions(t *testing.T) {
	cycle := &JSONShape{Kind: "union", Nullable: true}
	cycle.Branches = []*JSONShape{{Kind: "alias", Child: cycle}, {Kind: "string"}}
	for _, shape := range []*JSONShape{
		{Kind: "union"},
		{Kind: "union", Branches: []*JSONShape{{Kind: "string"}}},
		{Kind: "union", Branches: []*JSONShape{{Kind: "string"}, nil}},
		{Kind: "union", Branches: []*JSONShape{{Kind: "string"}, {Kind: "unknown"}}},
		cycle,
	} {
		shape.Nullable = true
		for _, wire := range []string{`null`, `"text"`, `{}`} {
			_, err := shape.Match([]byte(wire))
			require.Error(t, err)
		}
	}
}

func TestJSONShapeUnionChecksEveryDescription(t *testing.T) {
	failing := &JSONShape{Kind: "number", Rules: JSONShapeRules{Minimum: "invalid"}}
	for _, branches := range [][]*JSONShape{
		{{Kind: "number"}, failing},
		{failing, {Kind: "number"}},
	} {
		shape := &JSONShape{Kind: "union", Branches: branches}
		_, err := shape.Match(jsontext.Value(`1`))
		require.Error(t, err)
	}
}

func TestJSONShapeRecursiveNestedUnion(t *testing.T) {
	tree := &JSONShape{Kind: "union"}
	tree.Branches = []*JSONShape{
		{Kind: "object", Closed: true, Fields: []JSONShapeField{{Name: "children", Required: true, Shape: &JSONShape{Kind: "array", Child: tree}}}},
		{Kind: "object", Closed: true, Fields: []JSONShapeField{{Name: "name", Required: true, Shape: &JSONShape{Kind: "string"}}}},
	}
	for _, tc := range []struct {
		wire string
		want bool
	}{
		{`{"children":[]}`, true},
		{`{"children":[{"name":"leaf"},{"children":[]}]}`, true},
		{`{"children":[{}]}`, false},
	} {
		matched, err := tree.Match([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, matched)
	}
}
