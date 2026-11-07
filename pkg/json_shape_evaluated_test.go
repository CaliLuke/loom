package loom

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONShapeEvaluatedMembers(t *testing.T) {
	integer := &JSONShape{Kind: "integer"}
	object := &JSONShape{Kind: "object", Fields: []JSONShapeField{{Name: "x", Shape: integer}}}
	for _, tc := range []struct {
		name    string
		shape   *JSONShape
		wire    string
		members []string
	}{
		{"absent additional", object, `{"x":1,"y":2}`, []string{"x"}},
		{"explicit additional", &JSONShape{Kind: "object", Additional: &JSONShape{Kind: "any"}}, `{"z":1,"a":false}`, []string{"a", "z"}},
		{"typed additional", &JSONShape{Kind: "object", Additional: integer}, `{"z":1,"a":2}`, []string{"a", "z"}},
		{"closed", &JSONShape{Kind: "object", Closed: true, Fields: object.Fields}, `{"x":1}`, []string{"x"}},
		{"map", &JSONShape{Kind: "map", Child: integer}, `{"z":1}`, []string{"z"}},
		{"any", &JSONShape{Kind: "any"}, `{"z":1}`, nil},
		{"nested", &JSONShape{Kind: "union", Branches: []*JSONShape{object, {Kind: "string"}}}, `{"x":1,"y":2}`, []string{"x"}},
		{"alias", &JSONShape{Kind: "alias", Child: object}, `{"x":1}`, []string{"x"}},
		{"non object", &JSONShape{Kind: "string"}, `"x"`, nil},
		{"mismatch", object, `{"x":false}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.shape.Evaluated(jsontext.Value(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.members, got)
		})
	}
}

func TestJSONShapeAdditionalMembership(t *testing.T) {
	shape := &JSONShape{Kind: "object", Additional: &JSONShape{Kind: "integer"}, ForbiddenMembers: []string{"secret"}, Fields: []JSONShapeField{{Name: "x", Required: true, Shape: &JSONShape{Kind: "string"}}}}
	for _, tc := range []struct {
		wire string
		want bool
	}{
		{`{"x":"ok"}`, true}, {`{"x":"ok","extra":2}`, true},
		{`{"x":"ok","extra":"bad"}`, false}, {`{"extra":2}`, false},
		{`{"x":"ok","secret":2}`, false},
	} {
		got, err := shape.Match(jsontext.Value(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, got, tc.wire)
	}
	broken := &JSONShape{Kind: "object", Nullable: true, Additional: &JSONShape{Kind: "array"}}
	_, err := broken.Match(jsontext.Value(`null`))
	require.Error(t, err, "invalid additional description must fail before matching")
}
