package loom

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONShapeSingletonClosure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		branch *JSONShape
		wire   string
		want   bool
	}{
		{"string", &JSONShape{Kind: "string"}, `"ok"`, true},
		{"string rejects object", &JSONShape{Kind: "string"}, `{}`, false},
		{"Any empty object", &JSONShape{Kind: "any"}, `{}`, true},
		{"Any evaluated closure", &JSONShape{Kind: "any"}, `{"x":1}`, false},
		{"Any null", &JSONShape{Kind: "any"}, `null`, true},
		{"map evaluated members", &JSONShape{Kind: "map", Child: &JSONShape{Kind: "integer"}}, `{"x":1}`, true},
		{"map invalid value", &JSONShape{Kind: "map", Child: &JSONShape{Kind: "integer"}}, `{"x":"bad"}`, false},
		{"array", &JSONShape{Kind: "array", Child: &JSONShape{Kind: "string"}}, `["ok"]`, true},
		{"array invalid value", &JSONShape{Kind: "array", Child: &JSONShape{Kind: "string"}}, `[1]`, false},
		{"nested singleton", &JSONShape{Kind: "union", Branches: []*JSONShape{{Kind: "string"}}}, `"ok"`, true},
		{"nested multiple", &JSONShape{Kind: "union", Branches: []*JSONShape{{Kind: "string"}, {Kind: "integer"}}}, `1`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			shape := &JSONShape{Kind: "union", UnevaluatedClosed: true, Branches: []*JSONShape{tc.branch}}
			match, err := shape.Match([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, match)
		})
	}
}

func TestJSONShapeSingletonRejectsUnconsumedCycle(t *testing.T) {
	cycle := &JSONShape{Kind: "union", Nullable: true}
	cycle.Branches = []*JSONShape{cycle}
	for _, wire := range []string{`null`, `"ok"`, `{}`} {
		_, err := cycle.Match([]byte(wire))
		require.Error(t, err)
	}
}
