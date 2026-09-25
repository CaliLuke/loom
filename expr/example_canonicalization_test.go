package expr_test

import (
	"reflect"
	"testing"

	"github.com/CaliLuke/loom/expr"
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
	zero := 0.0

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
			name: "untagged union uses enum constraints before wire projection",
			attr: &expr.AttributeExpr{Type: &expr.Union{
				Untagged: true,
				Values: []*expr.NamedAttributeExpr{
					{Name: "A", Attribute: constrainedWirePayload("a")},
					{Name: "B", Attribute: constrainedWirePayload("b")},
				},
			}},
			example:  map[string]any{"kind": "b", "payload": "value"},
			expected: map[string]any{"kind": "b", "data": "value"},
		},
		{
			name: "primitive union uses enum constraints",
			attr: &expr.AttributeExpr{Type: &expr.Union{
				TypeKey:  "kind",
				ValueKey: "value",
				Values: []*expr.NamedAttributeExpr{
					{Name: "A", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"a"}}}},
					{Name: "B", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"b"}}}},
				},
			}},
			example:  "b",
			expected: map[string]any{"kind": "B", "value": "b"},
		},
		{
			name: "tagged union matches an empty object branch",
			attr: &expr.AttributeExpr{Type: &expr.Union{
				TypeKey:  "kind",
				ValueKey: "value",
				Values: []*expr.NamedAttributeExpr{
					{Name: "Empty", Attribute: &expr.AttributeExpr{Type: &expr.Object{}}},
					{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
				},
			}},
			example:  map[string]any{},
			expected: map[string]any{"kind": "Empty", "value": map[string]any{}},
		},
		{
			name: "tagged union matches extra fields in its sole open object branch",
			attr: &expr.AttributeExpr{Type: &expr.Union{
				Values: []*expr.NamedAttributeExpr{
					{Name: "Object", Attribute: &expr.AttributeExpr{Type: &expr.Object{
						{Name: "known", Attribute: &expr.AttributeExpr{Type: expr.String}},
					}}},
					{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
				},
			}},
			example:  map[string]any{"extra": "accepted"},
			expected: map[string]any{"type": "Object", "value": map[string]any{"extra": "accepted"}},
		},
		{
			name: "tagged union matches equivalent numeric enum types",
			attr: &expr.AttributeExpr{Type: &expr.Union{
				TypeKey:  "kind",
				ValueKey: "value",
				Values: []*expr.NamedAttributeExpr{
					{Name: "One", Attribute: &expr.AttributeExpr{Type: expr.Int64, Validation: &expr.ValidationExpr{Values: []any{int64(1)}}}},
					{Name: "Text", Attribute: &expr.AttributeExpr{Type: expr.String}},
				},
			}},
			example:  int(1),
			expected: map[string]any{"kind": "One", "value": int(1)},
		},
		{
			name: "tagged union honors exclusive bounds for typed integers",
			attr: &expr.AttributeExpr{Type: &expr.Union{
				TypeKey:  "kind",
				ValueKey: "value",
				Values: []*expr.NamedAttributeExpr{
					{Name: "NonPositive", Attribute: &expr.AttributeExpr{Type: expr.Int64, Validation: &expr.ValidationExpr{Maximum: &zero}}},
					{Name: "Positive", Attribute: &expr.AttributeExpr{Type: expr.Int64, Validation: &expr.ValidationExpr{ExclusiveMinimum: &zero}}},
				},
			}},
			example:  int64(2),
			expected: map[string]any{"kind": "Positive", "value": int64(2)},
		},
		{
			name:     "nil slices preserve JSON null semantics",
			attr:     &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}},
			example:  []string(nil),
			expected: nil,
		},
		{
			name:     "nil maps preserve JSON null semantics",
			attr:     &expr.AttributeExpr{Type: &expr.Map{ElemType: &expr.AttributeExpr{Type: expr.String}}},
			example:  map[string]string(nil),
			expected: nil,
		},
		{
			name:     "bool map keys become JSON member names",
			attr:     &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Boolean}, ElemType: &expr.AttributeExpr{Type: expr.Boolean}}},
			example:  map[bool]bool{true: false, false: true},
			expected: map[string]any{"true": false, "false": true},
		},
		{
			name:     "bool map value keys become JSON member names",
			attr:     &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Boolean}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
			example:  expr.MapVal{true: "yes", false: "no"},
			expected: map[string]any{"true": "yes", "false": "no"},
		},
		{
			name:     "integer map keys become JSON member names",
			attr:     &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Int}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
			example:  map[int]string{-3: "a", 12: "b"},
			expected: map[string]any{"-3": "a", "12": "b"},
		},
		{
			name:     "unsigned map keys become JSON member names",
			attr:     &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.UInt64}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
			example:  map[uint64]string{18446744073709551615: "max"},
			expected: map[string]any{"18446744073709551615": "max"},
		},
		{
			name:     "float map keys use JSON number text",
			attr:     &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Float64}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
			example:  map[float64]string{1.5: "a", 1e21: "b"},
			expected: map[string]any{"1.5": "a", "1e+21": "b"},
		},
		{
			name:     "float32 map keys use their own precision",
			attr:     &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Float32}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
			example:  map[float32]string{0.1: "a"},
			expected: map[string]any{"0.1": "a"},
		},
		{
			name: "non-string map keys canonicalize their values",
			attr: &expr.AttributeExpr{Type: &expr.Map{
				KeyType:  &expr.AttributeExpr{Type: expr.Boolean},
				ElemType: constrainedWirePayload("a"),
			}},
			example:  map[bool]any{true: map[string]any{"kind": "a", "payload": "x"}},
			expected: map[string]any{"true": map[string]any{"kind": "a", "data": "x"}},
		},
		{
			name: "union selects a bool-keyed map branch",
			attr: &expr.AttributeExpr{Type: &expr.Union{
				TypeKey:  "kind",
				ValueKey: "data",
				Values: []*expr.NamedAttributeExpr{
					{Name: "flags", Attribute: &expr.AttributeExpr{Type: &expr.Map{
						KeyType:  &expr.AttributeExpr{Type: expr.Boolean},
						ElemType: &expr.AttributeExpr{Type: expr.String},
					}}},
					{Name: "label", Attribute: &expr.AttributeExpr{Type: expr.String}},
				},
			}},
			example:  map[bool]string{true: "on"},
			expected: map[string]any{"kind": "flags", "data": map[string]any{"true": "on"}},
		},
		{
			name:     "colliding map member names are preserved",
			attr:     &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Any}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
			example:  map[any]any{1: "a", "1": "b"},
			expected: map[any]any{1: "a", "1": "b"},
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

func constrainedWirePayload(kind string) *expr.AttributeExpr {
	return &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: "kind", Attribute: &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{kind}}}},
			{Name: "payload", Attribute: &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:tag:json:name": []string{"data"}}}},
		},
		Validation: &expr.ValidationExpr{Required: []string{"kind", "payload"}},
	}
}
