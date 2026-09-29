package ir

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaNotRenderAndHash(t *testing.T) {
	schema := &Schema{Type: "string", ContentEncoding: "base64", Not: &Schema{}}
	rendered := RenderSchema(schema)
	require.NotNil(t, rendered.Not)
	data, err := json.Marshal(rendered)
	require.NoError(t, err)
	require.JSONEq(t, `{"type":"string","contentEncoding":"base64","not":{}}`, string(data))
	rendered.Not.Pattern = "changed"
	require.Empty(t, schema.Not.Pattern, "rendering must own the child")
	plain, err := json.Marshal(&Schema{})
	require.NoError(t, err)
	require.NotContains(t, string(plain), `"Not"`, "absent keyword preserves existing hash bytes")
	negated, err := json.Marshal(schema)
	require.NoError(t, err)
	require.Contains(t, string(negated), `"Not":`)

	components := map[string]*Schema{
		"Left":      {Not: &Schema{Ref: "#/components/schemas/A"}},
		"Right":     {Not: &Schema{Ref: "#/components/schemas/A_2"}},
		"Different": {Not: &Schema{Ref: "#/components/schemas/C"}},
		"A":         {Pattern: "a"}, "A_2": {Pattern: "a"}, "C": {Pattern: "b"},
	}
	cache := make(map[string]string)
	withSibling := &Schema{Ref: "#/components/schemas/A_2", Not: &Schema{Ref: "#/components/schemas/A_2"}}
	normalized := normalizeSchemaForHash(withSibling, components, cache, make(map[string]struct{}), responseSemanticHash)
	require.Equal(t, "#/components/schemas/A", normalized.Ref)
	require.NotNil(t, normalized.Not)
	require.Equal(t, "#/components/schemas/A", normalized.Not.Ref)
	require.Equal(t, "#/components/schemas/A_2", withSibling.Not.Ref)
	left := schemaHashByName("Left", components, cache, make(map[string]struct{}), responseSemanticHash)
	require.Equal(t, left, schemaHashByName("Right", components, cache, make(map[string]struct{}), responseSemanticHash))
	require.NotEqual(t, left, schemaHashByName("Different", components, cache, make(map[string]struct{}), responseSemanticHash))
}
