package loom

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchUntaggedJSONClosedCountsBeforeClosure(t *testing.T) {
	integer := &JSONShape{Kind: "integer"}
	x := &JSONShape{Kind: "object", Fields: []JSONShapeField{{Name: "x", Required: true, Shape: integer}}}
	y := &JSONShape{Kind: "object", Fields: []JSONShapeField{{Name: "y", Shape: integer}}}
	kind := &JSONShape{Kind: "object", Fields: []JSONShapeField{{Name: "kind", Required: true, Shape: &JSONShape{Kind: "string", Rules: JSONShapeRules{Enums: [][]jsontext.Value{{jsontext.Value(`"a"`)}}}}}}}
	for _, tc := range []struct {
		name, wire string
		shapes     []*JSONShape
		want       int
		message    string
	}{
		{"open pair", `{"x":1}`, []*JSONShape{x, y}, -1, "matched 2 branches in schema"},
		{"discriminated overlap", `{"kind":"a","x":1}`, []*JSONShape{x, kind}, -1, "matched 2 branches in schema"},
		{"map", `{"z":1}`, []*JSONShape{{Kind: "map", Child: integer}, {Kind: "string"}}, 0, ""},
		{"any object", `{"z":1}`, []*JSONShape{{Kind: "any"}, {Kind: "string"}}, -1, "unevaluated"},
		{"any scalar", `7`, []*JSONShape{{Kind: "any"}, {Kind: "string"}}, 0, ""},
		{"nested selected only", `{"x":1,"z":2}`, []*JSONShape{{Kind: "union", Branches: []*JSONShape{x, {Kind: "string"}}}, {Kind: "boolean"}}, -1, "unevaluated"},
		{"typed additional", `{"x":1,"z":2}`, []*JSONShape{{Kind: "object", Fields: x.Fields, Additional: integer}, {Kind: "string"}}, 0, ""},
		{"explicit open", `{"z":2}`, []*JSONShape{{Kind: "object", Additional: &JSONShape{Kind: "any"}}, {Kind: "string"}}, 0, ""},
		{"closed", `{"x":1}`, []*JSONShape{{Kind: "object", Fields: x.Fields, Closed: true}, {Kind: "string"}}, 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			branches := make([]JSONUnionBranch, len(tc.shapes))
			for i, shape := range tc.shapes {
				branches[i] = JSONUnionBranch{Schema: shape, Decode: shape.Match}
			}
			got, err := MatchUntaggedJSONClosed("Value", []byte(tc.wire), branches)
			if tc.message != "" {
				require.ErrorContains(t, err, tc.message)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.want, got)
		})
	}
}
