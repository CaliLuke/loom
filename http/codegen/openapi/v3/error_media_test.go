package openapiv3_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedErrorMedia(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.ErrorMediaDSL)
	jsonSpec := decodeOpenAPIJSON(t, artifacts.JSON)
	var yamlSpec map[string]any
	require.NoError(t, yaml.Unmarshal(artifacts.YAML, &yamlSpec))
	for _, spec := range []map[string]any{jsonSpec, yamlSpec} {
		for _, path := range []string{"/mixed", "/reversed", "/same_media"} {
			operation := requireOperation(t, spec, path, "get")
			responses := requireMap(t, operation["responses"], "responses")
			response := requireMap(t, responses["500"], "500 response")
			if ref, ok := response["$ref"].(string); ok {
				response = requireComponentResponse(t, spec, strings.TrimPrefix(ref, "#/components/responses/"))
			}
			content := requireMap(t, response["content"], "response content")
			if path == "/same_media" {
				require.Len(t, content, 1)
				media := requireMap(t, content["application/json"], "JSON alternatives")
				schema := requireMap(t, media["schema"], "alternative schema")
				require.Len(t, schema["anyOf"], 2)
				continue
			}
			require.Len(t, content, 2)
			require.Contains(t, content, "application/json")
			html := requireMap(t, content["text/html; charset=utf-8"], "HTML representation")
			require.Equal(t, "string", requireMap(t, html["schema"], "HTML schema")["type"])
		}
	}
	if output := os.Getenv("LOOM_ERROR_MEDIA_OUTPUT"); output != "" {
		require.NoError(t, os.WriteFile(filepath.Join(output, "openapi.json"), artifacts.JSON, 0o600))
		require.NoError(t, os.WriteFile(filepath.Join(output, "openapi.yaml"), artifacts.YAML, 0o600))
		return
	}
	for range 2 {
		output := t.TempDir()
		command := exec.Command(os.Args[0], "-test.run=^TestRenderedErrorMedia$", "-test.count=1")
		command.Env = append(os.Environ(), "LOOM_ERROR_MEDIA_OUTPUT="+output)
		logs, err := command.CombinedOutput()
		require.NoError(t, err, string(logs))
		for name, want := range map[string][]byte{"openapi.json": artifacts.JSON, "openapi.yaml": artifacts.YAML} {
			got, err := os.ReadFile(filepath.Join(output, name))
			require.NoError(t, err)
			require.Equal(t, want, got, "independent generation of %s", name)
		}
	}
}
