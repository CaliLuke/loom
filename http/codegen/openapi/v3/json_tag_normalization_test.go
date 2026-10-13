package openapiv3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestAuthoredJSONTagExamples(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.OpenAPIAuthoredJSONTagsDSL)
	if output := os.Getenv("LOOM_JSON_TAG_SPEC_OUTPUT"); output != "" {
		require.NoError(t, os.MkdirAll(output, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(output, "openapi.json"), artifacts.JSON, 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(output, "openapi.yaml"), artifacts.YAML, 0o600))
	}
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"json", artifacts.JSON},
		{"yaml", artifacts.YAML},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var spec map[string]any
			require.NoError(t, yaml.Unmarshal(tc.data, &spec))
			schema := requireComponentSchema(t, spec, "TagRecord")
			require.Equal(t, map[string]any{"plain": "", "items": []any{}, "omitted": "", "empty": "", "names": []any{}, "labels": map[string]any{}, "flag": false, "count": 0, "text": ""}, schema["example"])
			operation := requireOperation(t, spec, "/tagged", "post")
			responses := requireMap(t, operation["responses"], "responses")
			response := requireMap(t, responses["200"], "OK response")
			content := requireMap(t, response["content"], "response content")
			media := requireMap(t, content["application/json"], "JSON response")
			require.Equal(t, map[string]any{"plain": "", "items": []any{}, "empty": "", "names": []any{}, "labels": map[string]any{}, "flag": false, "count": 0, "text": ""}, media["example"])
			request := requireMap(t, operation["requestBody"], "request body")
			requestContent := requireMap(t, request["content"], "request content")
			requestMedia := requireMap(t, requestContent["application/json"], "JSON request")
			require.Equal(t, schema["example"], requestMedia["example"])
		})
	}

	lintEffectiveCompositionArtifacts(t, artifacts)
}
