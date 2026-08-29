package openapi

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSchemaSerializationContract(t *testing.T) {
	schema := NewSchema()
	schema.Type = Object
	schema.Properties["name"] = &Schema{Type: String}
	schema.Extensions = map[string]any{"x-origin": "contract"}

	encodedJSON, err := json.Marshal(schema)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"type": "object",
		"properties": {"name": {"type": "string"}},
		"x-origin": "contract"
	}`, string(encodedJSON))

	encodedYAML, err := yaml.Marshal(schema)
	require.NoError(t, err)
	var decodedYAML map[string]any
	require.NoError(t, yaml.Unmarshal(encodedYAML, &decodedYAML))
	require.Equal(t, map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"x-origin": "contract",
	}, decodedYAML)
}

func TestNewSchemaOwnsMutableMaps(t *testing.T) {
	first := NewSchema()
	second := NewSchema()
	first.Properties["value"] = &Schema{Type: String}
	first.Defs["Value"] = &Schema{Type: String}

	require.Empty(t, second.Properties)
	require.Empty(t, second.Defs)
}
