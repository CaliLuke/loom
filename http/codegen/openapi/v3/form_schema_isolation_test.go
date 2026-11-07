package openapiv3_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestFormRootEncodingSchemaIsolation(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.UntaggedFormDSL)
	validateOpenAPI(t, artifacts.JSON)
	validateOpenAPI(t, artifacts.YAML)
	spec := decodeOpenAPIJSON(t, artifacts.JSON)
	schemas := requireMap(t, requireMap(t, spec["components"], "components")["schemas"], "schemas")
	resolve := func(value map[string]any) map[string]any {
		if ref, ok := value["$ref"].(string); ok {
			return requireMap(t, schemas[strings.TrimPrefix(ref, "#/components/schemas/")], ref)
		}
		return value
	}
	var fields map[string]any
	for name, schema := range schemas {
		if name == "FormRootFields" || strings.HasPrefix(name, "FormRootFields_") {
			fields = requireMap(t, schema, name)
			break
		}
	}
	require.NotNil(t, fields)
	properties := requireMap(t, fields["properties"], "properties")
	for _, tc := range []struct {
		field, key string
		array      bool
	}{{"value", "x", false}, {"jsonValue", "jsonX", false}, {"records", "jsonX", true}} {
		t.Run(tc.field, func(t *testing.T) {
			field := resolve(requireMap(t, properties[tc.field], tc.field))
			var record map[string]any
			if tc.array {
				record = resolve(requireMap(t, field["items"], "items"))
			} else {
				record = resolve(requireMap(t, requireSlice(t, field["oneOf"], "oneOf")[1], "record"))
			}
			require.Contains(t, requireMap(t, record["properties"], "record.properties"), tc.key)
		})
	}
}
