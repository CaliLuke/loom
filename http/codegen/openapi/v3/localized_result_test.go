package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestLocalizedResultFieldsRetainSchema(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.LocalizedResultFieldsDSL)
	spec := decodeOpenAPIJSON(t, artifacts.JSON)
	holder := requireComponentSchema(t, spec, "LocalizedHolder")
	properties := requireMap(t, holder["properties"], "holder fields")
	for _, name := range []string{"custom", "plain", "items", "by_key"} {
		t.Run(name, func(t *testing.T) {
			field := requireMap(t, properties[name], name)
			if name == "items" {
				field = requireMap(t, field["items"], "array element")
			}
			if name == "by_key" {
				field = requireMap(t, field["additionalProperties"], "map element")
			}
			component := resolveRefMap(t, spec, requireString(t, field["$ref"], "field reference"))
			fields := requireMap(t, component["properties"], "record fields")
			require.Contains(t, fields, "id")
			require.Contains(t, fields, "name")
			require.Contains(t, component["required"], "id")
		})
	}
}
