package loom

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONShapeCompactExactNumericBounds(t *testing.T) {
	for _, test := range []struct {
		name  string
		shape *JSONShape
		wire  string
		want  bool
	}{
		{"tiny minimum accepts one", &JSONShape{Kind: "number", Rules: JSONShapeRules{Minimum: "1e-1000001"}}, "1", true},
		{"tiny minimum rejects zero", &JSONShape{Kind: "number", Rules: JSONShapeRules{Minimum: "1e-1000001"}}, "0", false},
		{"tiny minimum includes equality", &JSONShape{Kind: "number", Rules: JSONShapeRules{Minimum: "1e-1000001"}}, "1e-1000001", true},
		{"tiny minimum rejects smaller", &JSONShape{Kind: "number", Rules: JSONShapeRules{Minimum: "1e-1000001"}}, "1e-1000002", false},
		{"huge integer is integral", &JSONShape{Kind: "integer"}, "1e1000001", true},
		{"tiny decimal is fractional", &JSONShape{Kind: "integer"}, "1e-1000001", false},
		{"tiny decimal is a number", &JSONShape{Kind: "number"}, "1e-1000001", true},
		{"ordinary number", &JSONShape{Kind: "number"}, "1", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			matched, err := test.shape.Match([]byte(test.wire))
			require.NoError(t, err)
			require.Equal(t, test.want, matched)
		})
	}
}

func TestJSONShapeNumericDescriptorUsesJSONGrammar(t *testing.T) {
	for _, bound := range []string{"+1", "0x1p0", "1/2"} {
		t.Run(bound, func(t *testing.T) {
			shape := &JSONShape{Kind: "number", Rules: JSONShapeRules{Minimum: bound}}
			_, err := shape.Match([]byte("2"))
			require.Error(t, err, "a runtime JSON descriptor must not inherit DSL or rational-parser syntax")
		})
	}
}

func TestUntaggedJSONCompactNumericBound(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "number first"
		if reverse {
			name = "number last"
		}
		t.Run(name, func(t *testing.T) {
			branches := []JSONUnionBranch{
				{Schema: &JSONShape{Kind: "number", Rules: JSONShapeRules{Minimum: "1e-1000001"}}, Decode: func(wire jsontext.Value) (bool, error) {
					var value float64
					return DecodeJSONCandidate(wire, &value)
				}},
				{Schema: &JSONShape{Kind: "string"}, Decode: func(wire jsontext.Value) (bool, error) {
					var value string
					return DecodeJSONCandidate(wire, &value)
				}},
			}
			want := 0
			if reverse {
				branches[0], branches[1] = branches[1], branches[0]
				want = 1
			}
			selected, err := MatchUntaggedJSON("ExactNumber", []byte("1"), branches)
			require.NoError(t, err)
			require.Equal(t, want, selected)
		})
	}
}
