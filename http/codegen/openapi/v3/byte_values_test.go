package openapiv3_test

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedByteEnumValues(t *testing.T) {
	allBytes := make([]byte, 256)
	for index := range allBytes {
		allBytes[index] = byte(index)
	}
	expected := []any{"AH8=", base64.StdEncoding.EncodeToString(allBytes)}
	for _, version := range []struct{ target, version string }{{"3.1", "3.1.1"}, {"3.2", "3.2.0"}} {
		t.Run(version.target, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, testdata.CollectionEnumDSL, version.target, version.version)
			document := decodeOpenAPIJSON(t, artifacts.JSON)
			operation := requireOperation(t, document, "/bytes", "post")
			request := requireMap(t, operation["requestBody"], "byte request")
			requestContent := requireMap(t, request["content"], "byte request content")
			requestMedia := requireMap(t, requestContent["application/json"], "byte request media")
			responseMedia := requireResponseMediaType(t, operation, "application/json")
			for _, media := range []map[string]any{requestMedia, responseMedia} {
				schema := requireMap(t, media["schema"], "byte schema")
				if ref, ok := schema["$ref"].(string); ok {
					schema = requireComponentSchema(t, document, strings.TrimPrefix(ref, "#/components/schemas/"))
				}
				require.Equal(t, expected, schema["enum"])
				require.Contains(t, expected, media["example"])
				decoded, err := base64.StdEncoding.DecodeString(media["example"].(string))
				require.NoError(t, err)
				require.True(t, len(decoded) == 2 || len(decoded) == 256)
			}
		})
	}
}
