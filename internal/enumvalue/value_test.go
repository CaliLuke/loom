package enumvalue

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestNormalizePreservesAnyValuesAndWireNamePrecedence(t *testing.T) {
	object := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "data:wire", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		{Name: "raw", Attribute: &expr.AttributeExpr{Type: expr.Any}},
	}}
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		value     any
		expected  string
	}{
		{"object", object, map[string]any{"data:wire": "discarded", "wire": []byte(nil), "raw": jsontext.Value(`9007199254740993`), "extra": true}, `{"extra":true,"raw":9007199254740993,"wire":""}`},
		{"map", &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.UInt64}, ElemType: &expr.AttributeExpr{Type: expr.Any}}}, map[uint64]any{9007199254740993: []byte(nil), 2: jsontext.Value(`{"exact":18446744073709551615}`)}, `{"2":{"exact":18446744073709551615},"9007199254740993":""}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before, err := json.Marshal(tc.value, json.Deterministic(true))
			require.NoError(t, err)
			for range 32 {
				projected := Normalize(tc.attribute, tc.value)
				encoded, err := json.Marshal(projected, json.Deterministic(true))
				require.NoError(t, err)
				require.Equal(t, tc.expected, string(encoded))
			}
			after, err := json.Marshal(tc.value, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, before, after, "projection mutated its authored value")
		})
	}
}
