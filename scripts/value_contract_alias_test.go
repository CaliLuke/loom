package scripts_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

// TestValueContractLegacyAliasOverlap keeps the current public adapter weakness
// explicit until the typed resolver replaces it. Legal source/wire name overlap
// must validate every retained assignment, including the string field here.
func TestValueContractLegacyAliasOverlap(t *testing.T) {
	branch := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "a", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:json": {"b"}}}},
		{Name: "b", Attribute: &expr.AttributeExpr{Type: expr.Int, Meta: expr.MetaExpr{"struct:tag:json": {"c"}}}},
	}}
	union := &expr.AttributeExpr{Type: &expr.Union{
		TypeKey: "kind", ValueKey: "data", Values: []*expr.NamedAttributeExpr{
			{Name: "Overlapping", Attribute: branch},
			{Name: "Boolean", Attribute: &expr.AttributeExpr{Type: expr.Boolean}},
		},
	}}
	for _, tc := range []struct {
		name  string
		input map[string]any
	}{
		{name: "only overlapping alias", input: map[string]any{"b": 1}},
		{name: "wire precedence hides source", input: map[string]any{"a": "ok", "b": 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			actual := expr.CanonicalizeExample(union, tc.input)
			require.Equal(t, map[string]any{
				"kind": "Overlapping", "data": map[string]any{"b": 1, "c": 1},
			}, actual)
			wire, err := json.Marshal(actual)
			require.NoError(t, err)
			var decoded struct {
				Kind string `json:"kind"`
				Data struct {
					B string `json:"b"`
					C int    `json:"c"`
				} `json:"data"`
			}
			err = json.Unmarshal(wire, &decoded)
			require.Error(t, err)
			var semantic *json.SemanticError
			require.ErrorAs(t, err, &semantic)
			require.Equal(t, "/data/b", string(semantic.JSONPointer))
		})
	}
}
