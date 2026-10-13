package openapiv3_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestProblemPropertiesOpenAPI(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.ProblemPropertiesDSL)
	var yamlSpec map[string]any
	require.NoError(t, yaml.Unmarshal(artifacts.YAML, &yamlSpec))
	for _, spec := range []map[string]any{decodeOpenAPIJSON(t, artifacts.JSON), yamlSpec} {
		for _, name := range []string{"full", "pruned", "timeout", "header", "custom"} {
			path := "/properties/" + name
			if name == "full" {
				path = "/one/two"
			}
			operation := requireOperation(t, spec, path, "get")
			response := requireMap(t, requireMap(t, operation["responses"], "responses")["400"], "response")
			if ref, ok := response["$ref"].(string); ok {
				response = resolveRefMap(t, spec, ref)
			}
			media := "application/json"
			if name == "full" || name == "header" {
				media = "application/problem+json"
			}
			content := requireMap(t, response["content"], "content")
			schema := requireMap(t, requireMap(t, content[media], media)["schema"], "schema")
			if ref, ok := schema["$ref"].(string); ok {
				schema = resolveRefMap(t, spec, ref)
			}
			if name == "timeout" {
				require.Equal(t, "boolean", schema["type"])
				continue
			}
			properties := requireMap(t, schema["properties"], "properties")
			if name == "pruned" || name == "custom" {
				require.Len(t, properties, 1)
				for _, property := range []string{"timeout", "temporary", "fault"} {
					require.NotContains(t, properties, property)
				}
				continue
			}
			for _, property := range []string{"timeout", "temporary", "fault"} {
				if name == "header" && property == "temporary" {
					require.NotContains(t, properties, property)
					header := requireMap(t, requireMap(t, response["headers"], "headers")["X-Temporary"], "temporary header")
					if ref, ok := header["$ref"].(string); ok {
						header = resolveRefMap(t, spec, ref)
					}
					require.Equal(t, "boolean", requireMap(t, header["schema"], "header schema")["type"])
					continue
				}
				require.Equal(t, "boolean", requireMap(t, properties[property], property)["type"])
				require.Contains(t, schema["required"], property)
			}
		}
	}
	if output := os.Getenv("LOOM_PROBLEM_PROPERTIES_OUTPUT"); output != "" {
		for name, data := range map[string][]byte{"openapi.json": artifacts.JSON, "openapi.yaml": artifacts.YAML} {
			require.NoError(t, os.WriteFile(filepath.Join(output, name), data, 0o600))
		}
		return
	}
	lintEffectiveCompositionArtifacts(t, artifacts)
	for range 2 {
		output := t.TempDir()
		command := exec.Command(os.Args[0], "-test.run=^TestProblemPropertiesOpenAPI$", "-test.count=1")
		command.Env = append(os.Environ(), "LOOM_PROBLEM_PROPERTIES_OUTPUT="+output)
		logs, err := command.CombinedOutput()
		require.NoError(t, err, string(logs))
		for name, want := range map[string][]byte{"openapi.json": artifacts.JSON, "openapi.yaml": artifacts.YAML} {
			got, err := os.ReadFile(filepath.Join(output, name))
			require.NoError(t, err)
			require.Equal(t, want, got, "independent generation of %s", name)
		}
	}
}
