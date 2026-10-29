package openapiimport

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDistinctMediaContractsRoundTrip(t *testing.T) {
	var source map[string]any
	require.NoError(t, json.Unmarshal(readFixture(t, "supported.json"), &source))
	source["components"].(map[string]any)["schemas"].(map[string]any)["MediaFormData"] = map[string]any{
		"type": "object", "properties": map[string]any{"tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}},
	}
	content := map[string]any{
		"application/x-www-form-urlencoded": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/MediaFormData"}, "encoding": map[string]any{"tags": map[string]any{"style": "form", "explode": true}}},
		"multipart/form-data":               map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/MediaFormData"}, "encoding": map[string]any{"tags": map[string]any{"contentType": "text/plain"}}},
		"application/*":                     map[string]any{"schema": map[string]any{"type": "string", "format": "binary"}},
		"application/json":                  map[string]any{"schema": map[string]any{"type": "integer"}, "example": 12},
		"application/vnd.loom+json":         map[string]any{"schema": map[string]any{"type": "integer"}, "example": 34},
		"text/plain":                        map[string]any{"schema": map[string]any{"type": "string"}, "example": "hello"},
	}
	source["paths"].(map[string]any)["/representations"] = map[string]any{"post": map[string]any{
		"operationId": "exchangeRepresentations",
		"requestBody": map[string]any{"required": true, "content": content},
		"responses":   map[string]any{"200": map[string]any{"description": "Selected representation.", "content": content}, "400": map[string]any{"description": "Rejected representation.", "content": map[string]any{"application/json": content["application/json"], "text/plain": content["text/plain"]}}},
	}}
	encoded, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(encoded)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, rendered)
	contract := readGeneratedOpenAPIContract(t, directory)
	operation := operationFromImportedSpec(t, contract, "/representations", "post")
	request := operation["requestBody"].(map[string]any)["content"].(map[string]any)
	response := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)
	for _, got := range []map[string]any{request, response} {
		require.Len(t, got, 6)
		form := got["application/x-www-form-urlencoded"].(map[string]any)
		require.Equal(t, true, form["encoding"].(map[string]any)["tags"].(map[string]any)["explode"])
		multipart := got["multipart/form-data"].(map[string]any)
		require.Equal(t, "text/plain", multipart["encoding"].(map[string]any)["tags"].(map[string]any)["contentType"])
		for media, expected := range map[string]any{"application/json": float64(12), "application/vnd.loom+json": float64(34), "text/plain": "hello"} {
			require.Equal(t, expected, got[media].(map[string]any)["example"])
		}
	}
	errorContent := operation["responses"].(map[string]any)["400"].(map[string]any)["content"].(map[string]any)
	require.Len(t, errorContent, 2)
	require.Equal(t, "hello", errorContent["text/plain"].(map[string]any)["example"])
	requireIndependentRegeneration(t, directory)
}

func TestDistinctMediaContractsRejectUnsupportedEncoding(t *testing.T) {
	for _, tc := range []struct {
		name, media string
		schema      map[string]any
		encoding    map[string]any
	}{
		{"multipart explode", "multipart/form-data", map[string]any{"type": "object", "properties": map[string]any{"tags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}}, map[string]any{"tags": map[string]any{"explode": false}}},
		{"unknown part", "multipart/form-data", map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}}, map[string]any{"missing": map[string]any{"contentType": "text/plain"}}},
		{"part range", "multipart/form-data", map[string]any{"type": "object", "properties": map[string]any{"data": map[string]any{"type": "string", "format": "binary"}}}, map[string]any{"data": map[string]any{"contentType": "image/*"}}},
		{"wrong part codec", "multipart/form-data", map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}}, map[string]any{"name": map[string]any{"contentType": "application/json"}}},
		{"non-text", "text/plain", map[string]any{"type": "integer"}, nil},
		{"non-binary", "application/octet-stream", map[string]any{"type": "integer"}, nil},
		{"nested nullable", "application/x-www-form-urlencoded", map[string]any{"type": "object", "properties": map[string]any{"nested": map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"type": []string{"string", "null"}}}}}}, nil},
		{"nested union", "multipart/form-data", map[string]any{"type": "object", "properties": map[string]any{"nested": map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"oneOf": []any{map[string]any{"type": "integer"}, map[string]any{"type": "string"}}}}}}}, nil},
		{"nested reference", "multipart/form-data", map[string]any{"type": "object", "properties": map[string]any{"nested": map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"$ref": "#/components/schemas/NullableMediaValue"}}}}}, nil},
		{"nullable form", "application/x-www-form-urlencoded", map[string]any{"type": []string{"object", "null"}, "properties": map[string]any{"name": map[string]any{"type": "string"}}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var source map[string]any
			require.NoError(t, json.Unmarshal(readFixture(t, "supported.json"), &source))
			source["components"].(map[string]any)["schemas"].(map[string]any)["MediaInvalid"] = tc.schema
			source["components"].(map[string]any)["schemas"].(map[string]any)["NullableMediaValue"] = map[string]any{"type": []string{"string", "null"}}
			source["paths"].(map[string]any)["/media-invalid"] = map[string]any{"post": map[string]any{
				"operationId": "invalidMedia",
				"requestBody": map[string]any{"content": map[string]any{
					"application/json": map[string]any{"schema": map[string]any{"type": "integer"}},
					tc.media:           map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/MediaInvalid"}, "encoding": tc.encoding},
				}},
				"responses": map[string]any{"204": map[string]any{"description": "No content."}},
			}}
			encoded, err := json.Marshal(source, json.Deterministic(true))
			require.NoError(t, err)
			_, diagnostics, err := Analyze(encoded)
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)
		})
	}
}
