package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestRenderedMapCardinalityUsesPropertyKeywordsForAllTargets(t *testing.T) {
	for _, tc := range []struct {
		name        string
		target      string
		wantVersion string
	}{
		{name: "OpenAPI 3.1", target: "3.1", wantVersion: openapiv3.OpenAPICompatibilityVersion},
		{name: "OpenAPI 3.2", wantVersion: openapiv3.OpenAPIVersion},
	} {
		t.Run(tc.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifactsForVersion(t, mapCardinalityDSL, tc.target, tc.wantVersion)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			operation := requireOperation(t, spec, "/", "post")
			request := requireMap(t, operation["requestBody"], "request body")
			content := requireMap(t, request["content"], "request content")
			media := requireMap(t, content["application/json"], "request media")
			root := resolveCardinalitySchema(t, spec, requireMap(t, media["schema"], "request schema"))
			properties := requireMap(t, root["properties"], "request properties")

			text := requireMap(t, properties["text"], "text schema")
			require.Equal(t, float64(1), text["minLength"])
			require.Equal(t, float64(8), text["maxLength"])
			require.NotContains(t, text, "minProperties")

			items := requireMap(t, properties["items"], "array schema")
			require.Equal(t, float64(2), items["minItems"])
			require.Equal(t, float64(6), items["maxItems"])
			require.NotContains(t, items, "minProperties")

			direct := requireMap(t, properties["direct"], "inline map schema")
			require.Equal(t, float64(3), direct["minProperties"])
			require.Equal(t, float64(5), direct["maxProperties"])
			require.NotContains(t, direct, "minLength")
			require.NotContains(t, direct, "maxLength")

			named := resolveCardinalitySchema(t, spec, requireMap(t, properties["named"], "named map schema"))
			require.Equal(t, float64(4), named["minProperties"])
			require.Equal(t, float64(7), named["maxProperties"])
			require.NotContains(t, named, "minLength")
			require.NotContains(t, named, "maxLength")
			require.Contains(t, string(artifacts.YAML), "minProperties: 3")
		})
	}
}

func mapCardinalityDSL() {
	var NamedMap = dsl.Type("NamedMap", dsl.MapOf(dsl.String, dsl.String), func() {
		dsl.MinLength(4)
		dsl.MaxLength(7)
	})
	var _ = dsl.Service("cardinality", func() {
		dsl.Method("create", func() {
			dsl.Payload(func() {
				dsl.Attribute("text", dsl.String, func() {
					dsl.MinLength(1)
					dsl.MaxLength(8)
				})
				dsl.Attribute("items", dsl.ArrayOf(dsl.String), func() {
					dsl.MinLength(2)
					dsl.MaxLength(6)
				})
				dsl.Attribute("direct", dsl.MapOf(dsl.String, dsl.String), func() {
					dsl.MinLength(3)
					dsl.MaxLength(5)
				})
				dsl.Attribute("named", NamedMap)
			})
			dsl.HTTP(func() {
				dsl.POST("/")
			})
		})
	})
}

func resolveCardinalitySchema(t *testing.T, spec, schema map[string]any) map[string]any {
	t.Helper()
	ref, ok := schema["$ref"].(string)
	if !ok {
		return schema
	}
	const prefix = "#/components/schemas/"
	require.Contains(t, ref, prefix)
	return requireComponentSchema(t, spec, ref[len(prefix):])
}
