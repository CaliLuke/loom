package expr

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInlineJSONSchema(t *testing.T) {
	t.Run("builds object schema with validations defaults and examples", func(t *testing.T) {
		attr := &AttributeExpr{
			Title: "Example Object",
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
		require.Equal(t, "Example Object", data["title"])
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
		require.Equal(t, map[string]any{"propertyName": "type"}, data["discriminator"])
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

	t.Run("preserves wrapper metadata for user types", func(t *testing.T) {
		userType := &UserTypeExpr{
			TypeName: "Wrapped",
			AttributeExpr: &AttributeExpr{
				Type: &Object{
					&NamedAttributeExpr{Name: "id", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json:name": []string{"wire-id"}}}},
				},
			},
		}
		attr := &AttributeExpr{
			Type:         userType,
			Description:  "wrapper description",
			UserExamples: []*ExampleExpr{{Value: map[string]any{"id": "example"}}},
			Validation: &ValidationExpr{
				Required: []string{"id"},
			},
		}

		data := mustInlineJSONSchema(t, attr)

		require.Equal(t, "wrapper description", data["description"])
		require.ElementsMatch(t, []any{"wire-id"}, data["required"].([]any))
		require.Equal(t, map[string]any{"wire-id": "example"}, data["examples"].([]any)[0])
	})

	t.Run("rejects recursive user types", func(t *testing.T) {
		recursive := &UserTypeExpr{TypeName: "Node"}
		recursive.AttributeExpr = &AttributeExpr{
			Type: &Object{
				&NamedAttributeExpr{
					Name: "next",
					Attribute: &AttributeExpr{
						Type: recursive,
					},
				},
			},
		}

		_, err := InlineJSONSchema(&AttributeExpr{Type: recursive})

		require.Error(t, err)
		require.ErrorContains(t, err, "recursive")
	})
	type namedJSONMap map[string]any

	t.Run("uses effective JSON names for properties and required fields", func(t *testing.T) {
		renamed := &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json:name": []string{"wire-name"}}}
		ignored := &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": []string{"-"}}}
		attr := &AttributeExpr{
			Type: &Object{
				&NamedAttributeExpr{Name: "design_name", Attribute: renamed},
				&NamedAttributeExpr{Name: "ignored", Attribute: ignored},
			},
			DefaultValue: namedJSONMap{"design_name": "authored", "wire-name": "default", "ignored": "secret"},
			UserExamples: []*ExampleExpr{{Value: namedJSONMap{"design_name": "authored", "wire-name": "example", "ignored": "secret"}}},
			Validation: &ValidationExpr{
				Required: []string{"design_name", "ignored"},
				Values:   []any{namedJSONMap{"design_name": "enum", "ignored": "secret"}},
			},
		}
		data := mustInlineJSONSchema(t, attr)

		properties := data["properties"].(map[string]any)
		require.Contains(t, properties, "wire-name")
		require.NotContains(t, properties, "design_name")
		require.NotContains(t, properties, "ignored")
		require.Equal(t, []any{"wire-name"}, data["required"])
		require.Equal(t, map[string]any{"wire-name": "default"}, data["default"])
		require.Equal(t, map[string]any{"wire-name": "example"}, data["examples"].([]any)[0])
		require.Equal(t, []any{map[string]any{"wire-name": "enum"}}, data["enum"])
	})

	t.Run("canonicalizes constrained tagged union enums", func(t *testing.T) {
		empty := &Union{
			TypeKey:  "kind",
			ValueKey: "value",
			Values: []*NamedAttributeExpr{
				{Name: "Empty", Attribute: &AttributeExpr{Type: &Object{}}},
				{Name: "Text", Attribute: &AttributeExpr{Type: String}},
			},
		}
		emptySchema := mustInlineJSONSchema(t, &AttributeExpr{
			Type:       empty,
			Validation: &ValidationExpr{Values: []any{map[string]any{}}},
		})
		require.Equal(t, []any{map[string]any{
			"kind":  "Empty",
			"value": map[string]any{},
		}}, emptySchema["enum"])

		numeric := &Union{
			TypeKey:  "kind",
			ValueKey: "value",
			Values: []*NamedAttributeExpr{
				{Name: "One", Attribute: &AttributeExpr{Type: Int64, Validation: &ValidationExpr{Values: []any{int64(1)}}}},
				{Name: "Text", Attribute: &AttributeExpr{Type: String}},
			},
		}
		numericSchema := mustInlineJSONSchema(t, &AttributeExpr{
			Type:       numeric,
			Validation: &ValidationExpr{Values: []any{int(1)}},
		})
		require.Equal(t, []any{map[string]any{
			"kind":  "One",
			"value": float64(1),
		}}, numericSchema["enum"])
	})

	t.Run("emits exact primitive integer domains and exclusive bounds", func(t *testing.T) {
		minimum := float64(5)
		exclusiveMaximum := float64(10)
		schema, err := InlineJSONSchema(&AttributeExpr{Type: &Object{
			&NamedAttributeExpr{Name: "signed", Attribute: &AttributeExpr{Type: Int32, Validation: &ValidationExpr{
				Minimum: &minimum, ExclusiveMaximum: &exclusiveMaximum,
			}}},
			&NamedAttributeExpr{Name: "unsigned", Attribute: &AttributeExpr{Type: UInt64}},
		}})

		require.NoError(t, err)
		require.Contains(t, string(schema), `"signed":{"type":"integer","minimum":5,"maximum":2147483647,"exclusiveMaximum":10}`)
		require.Contains(t, string(schema), `"unsigned":{"type":"integer","minimum":0,"maximum":18446744073709551615}`)
	})

	t.Run("retains primitive domains through user type validation overlays", func(t *testing.T) {
		minimum := float64(5)
		signed := &UserTypeExpr{TypeName: "Signed", AttributeExpr: &AttributeExpr{Type: Int32}}
		unsigned := &UserTypeExpr{TypeName: "Unsigned", AttributeExpr: &AttributeExpr{Type: UInt64}}
		schema, err := InlineJSONSchema(&AttributeExpr{Type: &Object{
			&NamedAttributeExpr{Name: "signed", Attribute: &AttributeExpr{Type: signed, Validation: &ValidationExpr{Minimum: &minimum}}},
			&NamedAttributeExpr{Name: "unsigned", Attribute: &AttributeExpr{Type: unsigned, Validation: &ValidationExpr{Values: []any{uint64(1)}}}},
		}})

		require.NoError(t, err)
		var data map[string]any
		require.NoError(t, json.Unmarshal(schema, &data))
		properties := data["properties"].(map[string]any)
		signedSchema := properties["signed"].(map[string]any)
		require.Equal(t, float64(5), signedSchema["minimum"])
		signedBase := signedSchema["allOf"].([]any)[0].(map[string]any)
		require.Equal(t, float64(2147483647), signedBase["maximum"])
		unsignedSchema := properties["unsigned"].(map[string]any)
		require.Equal(t, []any{float64(1)}, unsignedSchema["enum"])
		unsignedBase := unsignedSchema["allOf"].([]any)[0].(map[string]any)
		require.Equal(t, float64(0), unsignedBase["minimum"])
		require.Equal(t, float64(18446744073709551615), unsignedBase["maximum"])
	})

	t.Run("clamps weaker validation bounds to primitive domains", func(t *testing.T) {
		negative := float64(-5)
		oversized := float64(1 << 40)
		schema, err := InlineJSONSchema(&AttributeExpr{Type: &Object{
			&NamedAttributeExpr{Name: "unsigned", Attribute: &AttributeExpr{Type: UInt64, Validation: &ValidationExpr{Minimum: &negative}}},
			&NamedAttributeExpr{Name: "signed", Attribute: &AttributeExpr{Type: Int32, Validation: &ValidationExpr{Maximum: &oversized}}},
		}})

		require.NoError(t, err)
		require.Contains(t, string(schema), `"unsigned":{"type":"integer","minimum":0,"maximum":18446744073709551615}`)
		require.Contains(t, string(schema), `"signed":{"type":"integer","minimum":-2147483648,"maximum":2147483647}`)
	})

	t.Run("retains named array length bounds through unrelated overlays", func(t *testing.T) {
		minimum := 3
		list := &UserTypeExpr{TypeName: "List", AttributeExpr: &AttributeExpr{
			Type:       &Array{ElemType: &AttributeExpr{Type: String}},
			Validation: &ValidationExpr{MinLength: &minimum},
		}}
		data := mustInlineJSONSchema(t, &AttributeExpr{
			Type:       list,
			Validation: &ValidationExpr{Values: []any{[]any{"value"}}},
		})

		base := data["allOf"].([]any)[0].(map[string]any)
		require.Equal(t, float64(3), base["minItems"])
		require.NotContains(t, data, "minLength")
	})

	t.Run("composes inherited and occurrence alias constraints", func(t *testing.T) {
		baseMinimum := float64(10)
		occurrenceMinimum := float64(5)
		alias := &UserTypeExpr{TypeName: "Bounded", AttributeExpr: &AttributeExpr{
			Type:       Int,
			Validation: &ValidationExpr{Minimum: &baseMinimum},
		}}
		data := mustInlineJSONSchema(t, &AttributeExpr{
			Type:       alias,
			Validation: &ValidationExpr{Minimum: &occurrenceMinimum},
		})

		require.Equal(t, float64(5), data["minimum"])
		base := data["allOf"].([]any)[0].(map[string]any)
		require.Equal(t, float64(10), base["minimum"])
	})
	t.Run("does not wrap equivalent alias validation", func(t *testing.T) {
		baseValidation := &ValidationExpr{Required: []string{"value"}}
		alias := &UserTypeExpr{TypeName: "Record", AttributeExpr: &AttributeExpr{
			Type: &Object{
				&NamedAttributeExpr{Name: "value", Attribute: &AttributeExpr{Type: String}},
			},
			Validation: baseValidation,
		}}
		data := mustInlineJSONSchema(t, &AttributeExpr{
			Type:       alias,
			Validation: &ValidationExpr{Required: []string{"value"}},
		})

		require.NotContains(t, data, "allOf")
		require.Contains(t, data, "properties")
	})
	t.Run("leaves arbitrary JSON unconstrained", func(t *testing.T) {
		schema, err := InlineJSONSchema(&AttributeExpr{Type: Any})

		require.NoError(t, err)
		require.JSONEq(t, `{}`, string(schema))
	})

	t.Run("rejects invalid effective JSON names", func(t *testing.T) {
		tests := map[string]*Object{
			"empty": {
				&NamedAttributeExpr{Name: "value", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": []string{",omitempty"}}}},
			},
			"duplicate": {
				&NamedAttributeExpr{Name: "first", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": []string{"same"}}}},
				&NamedAttributeExpr{Name: "second", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json:name": []string{"same"}}}},
			},
			"design-wire-cross-collision": {
				&NamedAttributeExpr{Name: "foo", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": []string{"bar"}}}},
				&NamedAttributeExpr{Name: "bar", Attribute: &AttributeExpr{Type: String, Meta: MetaExpr{"struct:tag:json": []string{"baz"}}}},
			},
		}
		for name, object := range tests {
			t.Run(name, func(t *testing.T) {
				_, err := InlineJSONSchema(&AttributeExpr{Type: object})
				require.Error(t, err)
			})
		}
	})
}

func TestInlineJSONSchemaRepresentsNullableValues(t *testing.T) {
	t.Parallel()

	data := mustInlineJSONSchema(t, &AttributeExpr{Type: String, Nullable: true})
	anyOf := data["anyOf"].([]any)
	require.Equal(t, "string", anyOf[0].(map[string]any)["type"])
	require.Equal(t, "null", anyOf[1].(map[string]any)["type"])

	named := &UserTypeExpr{
		TypeName:      "MaybeString",
		UID:           "maybe-string",
		AttributeExpr: &AttributeExpr{Type: String, Nullable: true},
	}
	inherited := mustInlineJSONSchema(t, &AttributeExpr{Type: named})
	require.Len(t, inherited["anyOf"], 2)
}
func TestInlineJSONSchemaUsesDeterministicObjectOrdering(t *testing.T) {
	properties := &Object{
		&NamedAttributeExpr{Name: "zulu", Attribute: &AttributeExpr{Type: String}},
		&NamedAttributeExpr{Name: "yankee", Attribute: &AttributeExpr{Type: String}},
		&NamedAttributeExpr{Name: "xray", Attribute: &AttributeExpr{Type: String}},
		&NamedAttributeExpr{Name: "whiskey", Attribute: &AttributeExpr{Type: String}},
		&NamedAttributeExpr{Name: "bravo", Attribute: &AttributeExpr{Type: String}},
		&NamedAttributeExpr{Name: "alpha", Attribute: &AttributeExpr{Type: String}},
	}
	attr := &AttributeExpr{
		Type: properties,
		UserExamples: []*ExampleExpr{{Value: map[string]any{
			"zulu":    "z",
			"yankee":  "y",
			"xray":    "x",
			"whiskey": "w",
			"bravo":   "b",
			"alpha":   "a",
		}}},
	}
	const wantExample = `"examples":[{"alpha":"a","bravo":"b","whiskey":"w","xray":"x","yankee":"y","zulu":"z"}]`
	const wantProperties = `"properties":{"alpha":{"type":"string"},"bravo":{"type":"string"},"whiskey":{"type":"string"},"xray":{"type":"string"},"yankee":{"type":"string"},"zulu":{"type":"string"}}`

	for range 20 {
		schema, err := InlineJSONSchema(attr)
		require.NoError(t, err)
		require.Contains(t, string(schema), wantExample)
		require.Contains(t, string(schema), wantProperties)
	}
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
