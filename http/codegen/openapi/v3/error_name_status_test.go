package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestStatusErrorNameOpenAPI(t *testing.T) {
	for _, version := range []struct{ target, want string }{
		{"3.1", openapiv3.OpenAPICompatibilityVersion},
		{"3.2", openapiv3.OpenAPIVersion},
	} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.OpenAPIStatusErrorNameDSL, version.target, version.want)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			lintRenderedContract(t, spec)
			for _, mode := range []string{"body", "selected", "empty"} {
				responses := requireMap(t, requireOperation(t, spec, "/status/"+mode, "get")["responses"], "responses")
				for _, status := range []string{"400", "404"} {
					response := requireMap(t, responses[status], "error response")
					if ref, ok := response["$ref"].(string); ok {
						response = resolveRefMap(t, spec, ref)
					}
					require.Empty(t, response["headers"])
					if mode == "empty" {
						require.Empty(t, response["content"])
						continue
					}
					content := requireMap(t, response["content"], "content")
					media := requireMap(t, content["application/json"], "JSON media")
					schema := requireMap(t, media["schema"], "body schema")
					if ref, ok := schema["$ref"].(string); ok {
						schema = resolveRefMap(t, spec, ref)
					}
					if mode == "selected" {
						require.Equal(t, "string", schema["type"])
						continue
					}
					properties := requireMap(t, schema["properties"], "properties")
					require.NotContains(t, properties, "loomError")
					require.Contains(t, properties, "message")
					if required := schema["required"]; required != nil {
						require.NotContains(t, required, "loomError")
					}
				}
			}
		})
	}
}
