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
			name:     "ambiguous union example is preserved",
			attr:     &expr.AttributeExpr{Type: union},
			example:  map[string]any{},
			expected: map[string]any{},
		},
		{
			name: "union object uses required fields to disambiguate optional overlap",
			attr: &expr.AttributeExpr{Type: &expr.Union{
				TypeKey:  "action",
				ValueKey: "payload",
				Values: []*expr.NamedAttributeExpr{
					{
						Name: "Reply",
						Attribute: &expr.AttributeExpr{
							Type: &expr.Object{
								{Name: "thread_id", Attribute: &expr.AttributeExpr{Type: expr.String}},
								{Name: "content", Attribute: &expr.AttributeExpr{Type: expr.String}},
							},
							Validation: &expr.ValidationExpr{
								Required: []string{"thread_id"},
							},
						},
					},
					{
						Name: "Resolve",
						Attribute: &expr.AttributeExpr{
							Type: &expr.Object{
								{Name: "thread_id", Attribute: &expr.AttributeExpr{Type: expr.String}},
								{Name: "resolved_by", Attribute: &expr.AttributeExpr{Type: expr.String}},
							},
							Validation: &expr.ValidationExpr{
								Required: []string{"thread_id", "resolved_by"},
							},
						},
					},
				},
			}},
			example: map[string]any{"thread_id": "T1"},
			expected: map[string]any{
				"action":  "Reply",
				"payload": map[string]any{"thread_id": "T1"},
			},
		},
		{
			name:    "array canonicalizes union elements",
			attr:    &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: union}}},
			example: []any{map[string]any{"name": "alice"}, map[string]any{"items": []any{"a"}}},
			expected: []any{
				map[string]any{
					"kind": "single",
					"data": map[string]any{"name": "alice"},
				},
				map[string]any{
					"kind": "batch",
					"data": map[string]any{"items": []any{"a"}},
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
