package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"

	"testing"

	"github.com/pb33f/libopenapi"

	"github.com/stretchr/testify/require"
)

func TestMediaRootMapAdmission(t *testing.T) {
	for _, media := range []string{"application/x-www-form-urlencoded", "multipart/form-data"} {
		for _, multiple := range []bool{false, true} {
			for _, wrapper := range []struct {
				name  string
				depth int
				union bool
			}{{"direct", 0, false}, {"singleton", 1, false}, {"nested singleton", 2, false}, {"union", 1, true}, {"nested union", 2, true}} {
				t.Run(fmt.Sprintf("%s/multiple=%t/%s", media, multiple, wrapper.name), func(t *testing.T) {
					source := mediaRootMapSource(t, media, multiple, wrapper.depth, wrapper.union, map[string]any{"type": "string"})
					_, diagnostics, err := Analyze(source)
					require.NoError(t, err)
					if media == "application/x-www-form-urlencoded" || !multiple && !wrapper.union {
						require.Empty(t, diagnostics)
					} else {
						require.NotEmpty(t, diagnostics)
						for _, diagnostic := range diagnostics {
							require.Equal(t, "media-schema", diagnostic.Code)
						}
					}
				})
			}
		}
	}
}

func mediaRootMapSource(t *testing.T, media string, multiple bool, depth int, union bool, element map[string]any) []byte {
	t.Helper()
	schema := map[string]any{"type": "object", "additionalProperties": element}
	for range depth {
		alternatives := []any{schema}
		if union {
			alternatives = append(alternatives, map[string]any{"type": "object", "properties": map[string]any{"kind": map[string]any{"type": "integer"}}, "required": []string{"kind"}})
		}
		schema = map[string]any{"oneOf": alternatives}
	}
	content := map[string]any{media: map[string]any{"schema": schema}}
	if multiple {
		content["application/json"] = map[string]any{"schema": map[string]any{"type": "integer"}}
	}
	source := map[string]any{
		"openapi": "3.1.1", "info": map[string]any{"title": "Map media", "version": "1"},
		"paths": map[string]any{"/maps": map[string]any{"post": map[string]any{
			"operationId": "writeMaps", "requestBody": map[string]any{"required": true, "content": content},
			"responses": map[string]any{"204": map[string]any{"description": "Written."}},
		}}},
	}
	wire, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	return wire
}

func TestFormRootMapElementAdmission(t *testing.T) {
	for _, tc := range []struct {
		name     string
		element  map[string]any
		accepted bool
	}{
		{"scalar", map[string]any{"type": "string"}, true},
		{"list", map[string]any{"type": "array", "items": map[string]any{"type": "string"}}, true},
		{"object", map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "integer"}}}, true},
		{"record items", map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "string"}}}}, true},
		{"source absent", map[string]any{"oneOf": []any{map[string]any{"type": "array", "items": map[string]any{"type": "string"}}}}, false},
		{"nullable scalar", map[string]any{"type": []string{"string", "null"}}, false},
		{"typeless union", map[string]any{"oneOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "integer"}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diagnostics, err := Analyze(mediaRootMapSource(t, "application/x-www-form-urlencoded", true, 0, false, tc.element))
			require.NoError(t, err)
			if tc.accepted {
				for _, diagnostic := range diagnostics {
					require.Equal(t, "schema-inline-array-item-promoted", diagnostic.Code)
				}
			} else {
				require.NotEmpty(t, diagnostics)
			}
		})
	}
}

func TestFormRootMapMediaRoundTrip(t *testing.T) {
	var source map[string]any
	require.NoError(t, json.Unmarshal(mediaRootMapSource(t, "application/x-www-form-urlencoded", true, 0, false, map[string]any{"type": "string"}), &source))
	paths := source["paths"].(map[string]any)
	for _, tc := range []struct {
		name    string
		depth   int
		element map[string]any
	}{
		{"nested", 2, map[string]any{"type": "string"}},
		{"list", 0, map[string]any{"type": "array", "items": map[string]any{"type": "string"}}},
		{"object", 0, map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "integer"}}}},
	} {
		var extra map[string]any
		require.NoError(t, json.Unmarshal(mediaRootMapSource(t, "application/x-www-form-urlencoded", true, tc.depth, tc.depth > 0, tc.element), &extra))
		path := extra["paths"].(map[string]any)["/maps"].(map[string]any)
		path["post"].(map[string]any)["operationId"] = "write" + tc.name
		paths["/"+tc.name] = path
	}
	wire, err := json.Marshal(source, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(wire)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, rendered)
	contract := readGeneratedOpenAPIContract(t, directory)
	for path := range paths {
		operation := operationFromImportedSpec(t, contract, path, "post")
		content := operation["requestBody"].(map[string]any)["content"].(map[string]any)
		require.Contains(t, content, "application/x-www-form-urlencoded")
		require.Contains(t, content, "application/json")
	}
	for _, name := range []string{"openapi.json", "openapi.yaml"} {
		wire, err := os.ReadFile(filepath.Join(directory, "gen", "http", name))
		require.NoError(t, err)
		parsed, err := libopenapi.NewDocument(wire)
		require.NoError(t, err)
		_, err = parsed.BuildV3Model()
		require.NoError(t, err)
	}
	requireIndependentRegeneration(t, directory)
}
