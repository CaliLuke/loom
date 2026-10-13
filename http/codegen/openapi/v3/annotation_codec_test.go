package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestTargetCodecSchemaAnnotations(t *testing.T) {
	for _, test := range []struct {
		name string
		dsl  func()
		want string
		path []string
	}{
		{"cookie", testdata.PayloadCookieCustomTextUnmarshalerDefaultDSL, "550e8400-e29b-41d4-a716-446655440000", nil},
		{"mapped-sse", testdata.SSEMappedAnnotationDSL, "fallback", []string{"components", "schemas", "WatchResponseBody", "properties", "data"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifacts(t, test.dsl)
			validateOpenAPI(t, artifacts.JSON)
			validateOpenAPI(t, artifacts.YAML)
			schema := decodeOpenAPIJSON(t, artifacts.JSON)
			if test.name == "cookie" {
				operation := requireOperation(t, schema, "/", "get")
				parameters := requireSlice(t, operation["parameters"], "parameters")
				parameter := requireMap(t, parameters[0], "parameter")
				require.Equal(t, "cookie", parameter["in"])
				schema = requireMap(t, parameter["schema"], "schema")
			} else {
				for _, field := range test.path {
					schema = requireMap(t, schema[field], field)
				}
				require.Equal(t, []any{"fallback", "explicit"}, schema["enum"])
			}
			require.Equal(t, test.want, schema["default"])
			lintEffectiveCompositionArtifacts(t, artifacts)
		})
	}
}
