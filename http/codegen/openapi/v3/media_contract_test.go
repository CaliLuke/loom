package openapiv3_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedDistinctMediaContracts(t *testing.T) {
	var vacuum contractVacuum
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "" {
		vacuum = installContractVacuum(t)
	}
	for _, version := range []struct{ target, version string }{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.MultipleMediaContractsDSL, version.target, version.version)
			for _, spec := range []map[string]any{decodeOpenAPIJSON(t, artifacts.JSON), decodeOpenAPIExampleYAML(t, artifacts.YAML)} {
				op := requireOperation(t, spec, "/media-contracts", "post")
				request := requireMap(t, op["requestBody"], "request body")
				requestContent := requireMap(t, request["content"], "request content")
				require.Len(t, requestContent, 7)
				responses := requireMap(t, op["responses"], "responses")
				response := requireMap(t, responses["200"], "response")
				responseContent := requireMap(t, response["content"], "response content")
				require.Len(t, responseContent, 7)
				for _, content := range []map[string]any{requestContent, responseContent} {
					plain := requireMap(t, content["text/plain"], "text media")
					require.Equal(t, "hello", plain["example"])
					schema := requireMap(t, plain["schema"], "text schema")
					require.Equal(t, "string", schema["type"])
					vendor := requireMap(t, content["application/vnd.loom+json"], "vendor media")
					require.EqualValues(t, 7, vendor["example"])
					schema = requireMap(t, vendor["schema"], "vendor schema")
					require.Equal(t, "integer", schema["type"])
					regular := requireMap(t, content["application/json"], "json media")
					examples := requireMap(t, regular["examples"], "json examples")
					require.Len(t, examples, 2)
				}
			}
			if vacuum.binary != "" {
				for ext, data := range map[string][]byte{"json": artifacts.JSON, "yaml": artifacts.YAML} {
					path := filepath.Join(t.TempDir(), "openapi."+ext)
					require.NoError(t, os.WriteFile(path, data, 0600))
					require.NoError(t, vacuum.lint(path))
				}
			}
		})
	}
}

func TestRenderedMediaContractsProjectParentExamples(t *testing.T) {
	artifacts := renderOpenAPIArtifactsForVersion(t, testdata.MultipleMediaContractsDSL, "3.2", "3.2.0")
	op := requireOperation(t, decodeOpenAPIJSON(t, artifacts.JSON), "/media-root-examples", "post")
	request := requireMap(t, op["requestBody"], "request")
	content := requireMap(t, request["content"], "content")
	media := requireMap(t, content["application/json"], "json")
	examples := requireMap(t, media["examples"], "examples")
	require.Len(t, examples, 2)
}
