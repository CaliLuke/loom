package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedSchemaExampleSurfaces(t *testing.T) {
	for _, version := range []struct {
		target, version string
	}{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.StreamingPartialExamplesDSL, version.target, version.version)
			for _, spec := range []map[string]any{
				decodeOpenAPIJSON(t, artifacts.JSON), decodeOpenAPIExampleYAML(t, artifacts.YAML),
			} {
				metadata := requireOperation(t, spec, "/metadata", "get")
				parameters := metadata["parameters"].([]any)
				require.Len(t, parameters, 1)
				parameter := requireMap(t, parameters[0], "chunks parameter")
				parameterSchema := requireMap(t, parameter["schema"], "chunks schema")
				items := requireMap(t, parameterSchema["items"], "chunk item schema")
				require.Equal(t, "AP8=", items["example"])
				require.Equal(t, []any{"AP8="}, parameter["example"])

				responses := requireMap(t, metadata["responses"], "metadata responses")
				response := requireMap(t, responses["200"], "metadata response")
				headers := requireMap(t, response["headers"], "metadata headers")
				header := requireMap(t, headers["X-Checksum"], "checksum header")
				headerSchema := requireMap(t, header["schema"], "checksum schema")
				require.Equal(t, "AP8=", headerSchema["example"])
				require.Equal(t, "AP8=", header["example"])

				stream := requireOperation(t, spec, "/ws/projects/{projectID}", "get")
				async := requireMap(t, stream[openAPIAsyncExtension], "async extension")
				messages := requireMap(t, async["messages"], "async messages")
				outbound := requireMap(t, messages["outbound"], "outbound message")
				schema := requireMap(t, outbound["schema"], "outbound schema")
				require.NotContains(t, schema, "example", "incomplete authored object must be omitted")
				properties := requireMap(t, schema["properties"], "outbound properties")
				attachment := requireMap(t, properties["attachment"], "attachment schema")
				require.Equal(t, "AP8=", attachment["example"])
				note := requireMap(t, properties["note"], "nullable note schema")
				require.Contains(t, note, "example")
				require.Nil(t, note["example"])
			}
		})
	}
}
