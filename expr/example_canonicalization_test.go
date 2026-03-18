package expr_test

import (
	"reflect"
	"testing"

	"goa.design/goa/v3/expr"
)

func TestCanonicalizeExample(t *testing.T) {
	union := &expr.Union{
		TypeKey:  "kind",
		ValueKey: "data",
		Values: []*expr.NamedAttributeExpr{
			{
				Name: "Single",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
					},
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"single"}},
				},
			},
			{
				Name: "Batch",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Object{
						{Name: "items", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}}},
					},
					Meta: expr.MetaExpr{"oneof:type:tag": []string{"batch"}},
				},
			},
		},
	}

	cases := []struct {
		name     string
		attr     *expr.AttributeExpr
		example  any
		expected any
	}{
		{
			name:    "union object uses tagged discriminator",
			attr:    &expr.AttributeExpr{Type: union},
			example: map[string]any{"name": "alice"},
			expected: map[string]any{
				"kind": "single",
				"data": map[string]any{"name": "alice"},
			},
		},
		{
			name:    "nested object canonicalizes union field",
			attr:    &expr.AttributeExpr{Type: &expr.Object{{Name: "payload", Attribute: &expr.AttributeExpr{Type: union}}}},
			example: map[string]any{"payload": map[string]any{"items": []any{"a", "b"}}},
			expected: map[string]any{
				"payload": map[string]any{
					"kind": "batch",
					"data": map[string]any{"items": []any{"a", "b"}},
				},
			},
		},
		{
			name:     "non-union examples are unchanged",
			attr:     &expr.AttributeExpr{Type: expr.String},
			example:  "plain",
			expected: "plain",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expr.CanonicalizeExample(tc.attr, tc.example)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("got %#v, expected %#v", actual, tc.expected)
			}
		})
	}
}
