package openapiv3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRequiredOccurrenceSchemaIsolation(t *testing.T) {
	for _, version := range []struct {
		target, want string
	}{
		{"3.1", openapiv3.OpenAPICompatibilityVersion},
		{"3.2", openapiv3.OpenAPIVersion},
	} {
		t.Run(version.target, func(t *testing.T) {
			spec := renderTypedSpec(t, testdata.RequiredOccurrenceDSL, version.target, version.want)
			schemas := typedComponents(spec)
			require.Equal(t, []any{"id"}, schemas["Base"].(map[string]any)["required"])
			requireTypedWire(t, spec, "Base", []typedWireCase{
				{`{"id":""}`, true}, {`{}`, false},
			})
			requireTypedWire(t, spec, occurrenceSchemaName(spec, "/left"), []typedWireCase{
				{`{"id":"","left":""}`, true}, {`{"id":"","right":""}`, false},
			})
			requireTypedWire(t, spec, occurrenceSchemaName(spec, "/right"), []typedWireCase{
				{`{"id":"","right":""}`, true}, {`{"id":"","left":""}`, false},
			})
			fields := schemas["Occurrences"].(map[string]any)["properties"].(map[string]any)
			require.Equal(t, []any{"left"}, fields["direct"].(map[string]any)["required"])
			require.Equal(t, []any{"right"}, fields["list"].(map[string]any)["items"].(map[string]any)["required"])
			require.Equal(t, []any{"left"}, fields["index"].(map[string]any)["additionalProperties"].(map[string]any)["required"])
			deferred := schemas["Deferred"].(map[string]any)
			require.NotContains(t, deferred, "required")
			child := deferred["properties"].(map[string]any)["child"].(map[string]any)
			require.Equal(t, []any{"name"}, child["required"])
			require.Equal(t, "#/components/schemas/Deferred", child["$ref"])
			requireTypedExamplesValid(t, spec)
		})
	}
}

func occurrenceSchemaName(spec map[string]any, path string) string {
	operation := spec["paths"].(map[string]any)[path].(map[string]any)["post"].(map[string]any)
	body := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)
	return strings.TrimPrefix(body["schema"].(map[string]any)["$ref"].(string), "#/components/schemas/")
}
