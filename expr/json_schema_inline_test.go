package expr

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInlineJSONSchema(t *testing.T) {
	t.Run("builds object schema with validations defaults and examples", func(t *testing.T) {
		attr := &AttributeExpr{
			Type: &Object{
				&NamedAttributeExpr{Name: "name", Attribute: &AttributeExpr{Type: String}},
				&NamedAttributeExpr{Name: "count", Attribute: &AttributeExpr{Type: Int}},
			},
			DefaultValue: map[string]any{
				"name":  "alpha",
				"count": 1,
			},
			Validation: &ValidationExpr{
				Required: []string{"name"},
			},
			UserExamples: []*ExampleExpr{{
				Summary: "example",
				Value: map[string]any{
					"name":  "beta",
					"count": 2,
				},
			}},
		}

		data := mustInlineJSONSchema(t, attr)

		require.Equal(t, "object", data["type"])
		require.Equal(t, false, data["additionalProperties"])
		require.ElementsMatch(t, []any{"name"}, data["required"].([]any))
		props := data["properties"].(map[string]any)
		require.Equal(t, "string", props["name"].(map[string]any)["type"])
		require.Equal(t, "integer", props["count"].(map[string]any)["type"])
		require.Equal(t, map[string]any{"name": "alpha", "count": float64(1)}, data["default"])
		require.Len(t, data["examples"].([]any), 1)
		require.Equal(t, map[string]any{"name": "beta", "count": float64(2)}, data["examples"].([]any)[0])
	})

	t.Run("builds union schema with canonicalized examples and explicit tags", func(t *testing.T) {
		typeA := &AttributeExpr{
			Type: &Object{
				&NamedAttributeExpr{Name: "kind", Attribute: &AttributeExpr{Type: String}},
			},
		}
		typeB := &AttributeExpr{
			Type: &Object{
				&NamedAttributeExpr{Name: "count", Attribute: &AttributeExpr{Type: Int}},
			},
		}
		union := &AttributeExpr{
			Type: &Union{
				Values: []*NamedAttributeExpr{
					{Name: "A", Attribute: &AttributeExpr{Type: typeA.Type, Meta: MetaExpr{"oneof:type:tag": []string{"kind_a"}}}},
					{Name: "B", Attribute: &AttributeExpr{Type: typeB.Type, Meta: MetaExpr{"oneof:type:tag": []string{"count_b"}}}},
				},
			},
			UserExamples: []*ExampleExpr{
				{Value: map[string]any{"kind": "demo"}},
				{Value: map[string]any{"count": 2}},
			},
		}

		data := mustInlineJSONSchema(t, union)

		require.Equal(t, "object", data["type"])
		oneOf := data["oneOf"].([]any)
		require.Len(t, oneOf, 2)

		first := oneOf[0].(map[string]any)
		firstProps := first["properties"].(map[string]any)
		require.Equal(t, []any{"kind_a"}, firstProps["type"].(map[string]any)["enum"])

		second := oneOf[1].(map[string]any)
		secondProps := second["properties"].(map[string]any)
		require.Equal(t, []any{"count_b"}, secondProps["type"].(map[string]any)["enum"])

		examples := data["examples"].([]any)
		require.Len(t, examples, 2)
		require.Equal(t, map[string]any{
			"type":  "kind_a",
			"value": map[string]any{"kind": "demo"},
		}, examples[0])
		require.Equal(t, map[string]any{
			"type":  "count_b",
			"value": map[string]any{"count": float64(2)},
		}, examples[1])
	})

	t.Run("leaves ambiguous union examples unchanged", func(t *testing.T) {
		union := &AttributeExpr{
			Type: &Union{
				Values: []*NamedAttributeExpr{
					{
						Name: "A",
						Attribute: &AttributeExpr{Type: &Object{
							&NamedAttributeExpr{Name: "shared", Attribute: &AttributeExpr{Type: String}},
						}},
					},
					{
						Name: "B",
						Attribute: &AttributeExpr{Type: &Object{
							&NamedAttributeExpr{Name: "shared", Attribute: &AttributeExpr{Type: String}},
						}},
					},
				},
			},
			UserExamples: []*ExampleExpr{{Value: map[string]any{"shared": "x"}}},
		}

		data := mustInlineJSONSchema(t, union)

		examples := data["examples"].([]any)
		require.Len(t, examples, 1)
		require.Equal(t, map[string]any{"shared": "x"}, examples[0])
	})

	t.Run("uses required fields to disambiguate optional object overlap", func(t *testing.T) {
		union := &AttributeExpr{
			Type: &Union{
				TypeKey:  "action",
				ValueKey: "payload",
				Values: []*NamedAttributeExpr{
					{
						Name: "reply",
						Attribute: &AttributeExpr{
							Type: &Object{
								&NamedAttributeExpr{Name: "thread_id", Attribute: &AttributeExpr{Type: String}},
								&NamedAttributeExpr{Name: "content", Attribute: &AttributeExpr{Type: String}},
							},
							Validation: &ValidationExpr{
								Required: []string{"thread_id"},
							},
						},
					},
					{
						Name: "resolve",
						Attribute: &AttributeExpr{
							Type: &Object{
								&NamedAttributeExpr{Name: "thread_id", Attribute: &AttributeExpr{Type: String}},
								&NamedAttributeExpr{Name: "resolved_by", Attribute: &AttributeExpr{Type: String}},
							},
							Validation: &ValidationExpr{
								Required: []string{"thread_id", "resolved_by"},
							},
						},
					},
				},
			},
			UserExamples: []*ExampleExpr{{Value: map[string]any{"thread_id": "T1"}}},
		}

		data := mustInlineJSONSchema(t, union)

		examples := data["examples"].([]any)
		require.Len(t, examples, 1)
		require.Equal(t, map[string]any{
			"action":  "reply",
			"payload": map[string]any{"thread_id": "T1"},
		}, examples[0])
	})

	t.Run("builds map and array schemas with length constraints", func(t *testing.T) {
		attr := &AttributeExpr{
			Type: &Object{
				&NamedAttributeExpr{
					Name: "tags",
					Attribute: &AttributeExpr{
						Type: &Array{ElemType: &AttributeExpr{Type: String}},
						Validation: &ValidationExpr{
							MinLength: intPtr(1),
							MaxLength: intPtr(3),
						},
					},
				},
				&NamedAttributeExpr{
					Name: "labels",
					Attribute: &AttributeExpr{
						Type: &Map{
							KeyType:  &AttributeExpr{Type: String},
							ElemType: &AttributeExpr{Type: Boolean},
						},
					},
				},
			},
		}

		data := mustInlineJSONSchema(t, attr)
		props := data["properties"].(map[string]any)

		tags := props["tags"].(map[string]any)
		require.Equal(t, "array", tags["type"])
		require.Equal(t, float64(1), tags["minItems"])
		require.Equal(t, float64(3), tags["maxItems"])
		require.Equal(t, "string", tags["items"].(map[string]any)["type"])

		labels := props["labels"].(map[string]any)
		require.Equal(t, "object", labels["type"])
		require.Equal(t, "boolean", labels["additionalProperties"].(map[string]any)["type"])
	})
}

func mustInlineJSONSchema(t *testing.T, attr *AttributeExpr) map[string]any {
	t.Helper()

	schema, err := InlineJSONSchema(attr)
	require.NoError(t, err)

	var data map[string]any
	require.NoError(t, json.Unmarshal(schema, &data))
	return data
}

func intPtr(v int) *int {
	return &v
}
