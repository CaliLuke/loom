package enumvalue

import (
	"fmt"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/examplevalue"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAllMatchesIndependentSources(t *testing.T) {
	minimum := 2
	union := &expr.Union{Values: []*expr.NamedAttributeExpr{
		{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "Number", Attribute: &expr.AttributeExpr{Type: expr.Float32}},
	}}
	object := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "blob:b", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
	}}
	calls := 0
	opaque := &opaqueEnumCodec{Text: "opaque", Calls: &calls}
	cases := []struct {
		name      string
		attribute *expr.AttributeExpr
		values    []any
	}{
		{"enum admission and fallback", &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"a", "b"}}}, []any{"a", "b", "outside", nil, 17, "a"}},
		{"bytes", &expr.AttributeExpr{Type: expr.Bytes}, []any{"first", []byte("second"), []byte(nil), nil}},
		{"precision", &expr.AttributeExpr{Type: expr.Float32}, []any{0.1, 0.2, 0.1}},
		{"declared shape fallback", &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Float32}}, Validation: &expr.ValidationExpr{MinLength: &minimum}}, []any{[]float64{0.1}, []float64{0.2}, []float64{0.1, 0.2}}},
		{"branch identity", &expr.AttributeExpr{Type: union}, []any{examplevalue.Union{Branch: 0, Value: "a"}, examplevalue.Union{Branch: 1, Value: 0.1}, examplevalue.Union{Branch: 0, Value: "b"}}},
		{"missing attribute", nil, []any{nil, "a", 12}},
		{"missing type", &expr.AttributeExpr{}, []any{nil, "a", 12}},
		{"objects and invalid shapes", object, []any{map[string]any{"blob:b": "a"}, map[string]any{"blob:b": 42}, map[string]any{"blob:b": "b"}, opaque}},
		{"malformed declaration", &expr.AttributeExpr{Type: &expr.Object{
			{Name: "first:wire", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "second:wire", Attribute: &expr.AttributeExpr{Type: expr.String}},
		}}, []any{map[string]any{"wire": "a"}, map[string]any{"wire": "b"}}},
		{"empty values", object, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expected := make([]any, len(tc.values))
			for i, v := range tc.values {
				expected[i] = Normalize(tc.attribute, v)
			}
			require.Equal(t, expected, NormalizeAll(tc.attribute, tc.values))
		})
	}
	require.Zero(t, calls, "normalization must not invoke opaque codecs")
}

func TestNormalizeAllLargeEnumPreservesMembers(t *testing.T) {
	values := make([]any, 433)
	for i := range values {
		values[i] = fmt.Sprintf("Zone/Region%d", i)
	}
	attribute := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: values}}
	require.Equal(t, values, NormalizeAll(attribute, values))
	require.Equal(t, values, attribute.Validation.Values)
}
