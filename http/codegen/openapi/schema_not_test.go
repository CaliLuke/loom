package openapi

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSchemaNotSerialization(t *testing.T) {
	for _, wire := range []string{`{"not":{}}`, `{"not":{"pattern":"[^A-Za-z0-9+/=]"}}`, `{"type":"string"}`} {
		t.Run(wire, func(t *testing.T) {
			var schema Schema
			require.NoError(t, json.Unmarshal([]byte(wire), &schema))
			got, err := json.Marshal(&schema)
			require.NoError(t, err)
			require.JSONEq(t, wire, string(got))
			encodedYAML, err := yaml.Marshal(&schema)
			require.NoError(t, err)
			var expected, actual map[string]any
			require.NoError(t, json.Unmarshal([]byte(wire), &expected))
			require.NoError(t, yaml.Unmarshal(encodedYAML, &actual))
			require.Equal(t, expected, actual)
		})
	}
}
