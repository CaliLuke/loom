package ir

import (
	jsonv2 "encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/expr"
)

func TestOpenAPIExampleValueUsesJSONMemberNamesForScalarMapKeys(t *testing.T) {
	t.Parallel()

	boolKey := &expr.AttributeExpr{Type: expr.Boolean}
	cases := []struct {
		name     string
		attr     *expr.AttributeExpr
		raw      any
		expected any
		json     string
		yaml     string
	}{
		{
			name:     "bool keys",
			attr:     &expr.AttributeExpr{Type: &expr.Map{KeyType: boolKey, ElemType: &expr.AttributeExpr{Type: expr.Boolean}}},
			raw:      map[bool]bool{true: false, false: true},
			expected: map[string]any{"true": false, "false": true},
			json:     `{"false":true,"true":false}`,
			yaml:     "\"false\": true\n\"true\": false\n",
		},
		{
			name: "nested bool keys",
			attr: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: &expr.Map{
				KeyType:  boolKey,
				ElemType: &expr.AttributeExpr{Type: &expr.Map{KeyType: boolKey, ElemType: &expr.AttributeExpr{Type: expr.String}}},
			}}}},
			raw:      []any{map[bool]map[bool]string{true: {false: "x"}}},
			expected: []any{map[string]any{"true": map[string]any{"false": "x"}}},
			json:     `[{"true":{"false":"x"}}]`,
			yaml:     "- \"true\":\n    \"false\": x\n",
		},
		{
			name:     "integer keys keep their JSON names",
			attr:     &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.Int}, ElemType: &expr.AttributeExpr{Type: expr.String}}},
			raw:      map[int]string{1: "one", 20: "twenty"},
			expected: map[string]any{"1": "one", "20": "twenty"},
			json:     `{"1":"one","20":"twenty"}`,
			yaml:     "\"1\": one\n\"20\": twenty\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			value, ok := OpenAPIExampleValue(tc.attr, tc.raw)
			require.True(t, ok)
			require.Equal(t, tc.expected, value)
			jsonValue, err := jsonv2.Marshal(value, jsonv2.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, tc.json, string(jsonValue))
			yamlValue, err := yaml.Marshal(value)
			require.NoError(t, err)
			require.Equal(t, tc.yaml, string(yamlValue))
		})
	}
}

func TestAnalyzerWithoutExampleValueUsesJSONMemberNamesForMapKeys(t *testing.T) {
	t.Parallel()

	attr := &expr.AttributeExpr{Type: &expr.Map{
		KeyType: &expr.AttributeExpr{Type: expr.String},
		ElemType: &expr.AttributeExpr{Type: &expr.Map{
			KeyType:  &expr.AttributeExpr{Type: expr.Boolean},
			ElemType: &expr.AttributeExpr{Type: expr.Int},
		}},
	}}
	schema := NewAnalyzer(expr.NewRandom("map-keys"), false).AnalyzeSchema(attr)
	require.NotNil(t, schema.AdditionalProperties)
	require.NotNil(t, schema.AdditionalProperties.Schema)
	for _, example := range []any{schema.Example, schema.AdditionalProperties.Schema.Example} {
		require.NotNil(t, example)
		_, err := jsonv2.Marshal(example, jsonv2.Deterministic(true))
		require.NoError(t, err)
	}
	inner, ok := schema.AdditionalProperties.Schema.Example.(map[string]any)
	require.Truef(t, ok, "inner example is %T", schema.AdditionalProperties.Schema.Example)
	require.NotEmpty(t, inner)
	for key := range inner {
		require.Contains(t, []string{"true", "false"}, key)
	}
}
