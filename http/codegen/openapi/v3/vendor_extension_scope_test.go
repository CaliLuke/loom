package openapiv3

import (
	"encoding/json/v2"
	"strconv"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

func TestVendorExtensionScopesSurviveRendering(t *testing.T) {
	document := New(expr.RunDSL(t, testdata.OpenAPIVendorExtensionScopeDSL))
	require.NotNil(t, document)

	tests := []struct {
		name string
		path []string
		want any
	}{
		{
			name: "document",
			path: []string{"x-migration-review"},
			want: map[string]any{"status": "reviewed"},
		},
		{
			name: "operation",
			path: []string{"paths", "/items/{id}", "get", "x-operation-note"},
			want: "gated",
		},
		{
			name: "parameter",
			path: []string{"paths", "/items/{id}", "get", "parameters", "0", "x-param-note"},
			want: "identifier",
		},
		{
			name: "parameter schema",
			path: []string{"paths", "/items/{id}", "get", "parameters", "0", "schema", "x-param-schema-note"},
			want: "uuid-ish",
		},
		{
			name: "request body",
			path: []string{"paths", "/items", "post", "requestBody", "x-request-note"},
			want: "body-gated",
		},
		{
			name: "media type",
			path: []string{"paths", "/items", "post", "requestBody", "content", "application/json", "x-media-note"},
			want: "json-only",
		},
		{
			name: "response",
			path: []string{"paths", "/items/{id}", "get", "responses", "200", "x-response-note"},
			want: map[string]any{"state": "primary"},
		},
		{
			name: "response header",
			path: []string{"paths", "/items/{id}", "get", "responses", "200", "headers", "X-Cursor", "x-header-note"},
			want: "pagination",
		},
		{
			name: "component schema",
			path: []string{"components", "schemas", "Item", "x-schema-note"},
			want: map[string]any{"audience": "public"},
		},
	}

	jsonDocument, err := json.Marshal(document)
	require.NoError(t, err)
	var decodedJSON map[string]any
	require.NoError(t, json.Unmarshal(jsonDocument, &decodedJSON))

	yamlDocument, err := yaml.Marshal(document)
	require.NoError(t, err)
	var decodedYAML map[string]any
	require.NoError(t, yaml.Unmarshal(yamlDocument, &decodedYAML))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, lookupDocumentValue(t, decodedJSON, test.path), "JSON rendering")
			require.Equal(t, test.want, lookupDocumentValue(t, decodedYAML, test.path), "YAML rendering")
		})
	}
}

func TestVendorExtensionTagScopeSurvivesRendering(t *testing.T) {
	document := New(expr.RunDSL(t, testdata.OpenAPIVendorExtensionScopeDSL))
	require.NotNil(t, document)

	jsonDocument, err := json.Marshal(document)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(jsonDocument, &decoded))

	tags, ok := decoded["tags"].([]any)
	require.True(t, ok, "document has no rendered tags")
	for _, tag := range tags {
		rendered, ok := tag.(map[string]any)
		if !ok || rendered["name"] != "catalog" {
			continue
		}
		require.Equal(t, "blue", rendered["x-color"])
		return
	}
	t.Errorf("tag %q is not present in the rendered document", "catalog")
}

// lookupDocumentValue walks a decoded OpenAPI document along path, where a
// numeric segment indexes a sequence and any other segment keys a mapping.
func lookupDocumentValue(t *testing.T, document any, path []string) any {
	t.Helper()
	current := document
	for index, segment := range path {
		switch container := current.(type) {
		case map[string]any:
			value, ok := container[segment]
			if !ok {
				t.Errorf("path %v: key %q is absent", path[:index+1], segment)
				return nil
			}
			current = value
		case []any:
			position, err := strconv.Atoi(segment)
			if err != nil {
				t.Errorf("path %v: segment %q does not index a sequence", path[:index+1], segment)
				return nil
			}
			if position < 0 || position >= len(container) {
				t.Errorf("path %v: index %d is out of range", path[:index+1], position)
				return nil
			}
			current = container[position]
		default:
			t.Errorf("path %v: value is not traversable", path[:index+1])
			return nil
		}
	}
	return normalizeDecodedValue(current)
}

// normalizeDecodedValue converts YAML mappings to the string-keyed shape the
// JSON decoder produces so both renderings compare against one expectation.
func normalizeDecodedValue(value any) any {
	switch actual := value.(type) {
	case map[any]any:
		normalized := make(map[string]any, len(actual))
		for key, item := range actual {
			normalized[key.(string)] = normalizeDecodedValue(item)
		}
		return normalized
	case map[string]any:
		normalized := make(map[string]any, len(actual))
		for key, item := range actual {
			normalized[key] = normalizeDecodedValue(item)
		}
		return normalized
	case []any:
		normalized := make([]any, len(actual))
		for index, item := range actual {
			normalized[index] = normalizeDecodedValue(item)
		}
		return normalized
	default:
		return value
	}
}
