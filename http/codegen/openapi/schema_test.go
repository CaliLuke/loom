package openapi

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/expr"
)

type exactNumber string

func (number exactNumber) MarshalJSON() ([]byte, error) {
	return []byte(number), nil
}

func (number exactNumber) MarshalYAML() (any, error) {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: string(number)}, nil
}
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

func TestSchemaExtensionsPreserveExactJSONNumbers(t *testing.T) {
	schema := NewSchema()
	schema.Example = exactNumber("9007199254740993")
	schema.Extensions = ExtensionsFromExpr(expr.MetaExpr{
		"openapi:extension:x-origin": {"true"},
		"openapi:extension:x-limit":  {"123456789012345678901234567890"},
	})

	encodedJSON, err := json.Marshal(schema, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, `{"example":9007199254740993,"x-limit":123456789012345678901234567890,"x-origin":true}`, string(encodedJSON))

	encodedYAML, err := yaml.Marshal(schema)
	require.NoError(t, err)
	require.Contains(t, string(encodedYAML), "example: 9007199254740993")
	require.Contains(t, string(encodedYAML), "x-limit: 123456789012345678901234567890")
}

func TestNewSchemaOwnsMutableMaps(t *testing.T) {
	first := NewSchema()
	second := NewSchema()
	first.Properties["value"] = &Schema{Type: String}
	first.Defs["Value"] = &Schema{Type: String}

	require.Empty(t, second.Properties)
	require.Empty(t, second.Defs)
}
