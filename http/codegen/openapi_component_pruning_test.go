package codegen

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
)

func TestOpenAPIPrunesUnusedComponentSchemas(t *testing.T) {
	root := RunHTTPDSL(t, openAPIUnusedComponentSchemasDSL)

	spec := renderOpenAPIJSON(t, openapiv3.Files, root)
	parseOpenAPIV3Document(t, spec)

	schemas := componentSchemasFromSpec(t, spec)
	media := postRequestMediaTypeFromSpec(t, spec, "/components", "application/json")
	body, ok := media["schema"].(map[string]any)
	require.True(t, ok)
	ref, ok := body["$ref"].(string)
	require.True(t, ok)
	name, local := strings.CutPrefix(ref, "#/components/schemas/")
	require.True(t, local)
	payload, ok := schemas[name].(map[string]any)
	require.True(t, ok, "the operation's allocated payload must remain reachable")
	require.Equal(t, []any{"nested"}, payload["required"])
	properties, ok := payload["properties"].(map[string]any)
	require.True(t, ok)
	require.Len(t, properties, 1)
	nested, ok := properties["nested"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "#/components/schemas/ReachableNested", nested["$ref"])
	require.Contains(t, schemas, "ReachableNested")
	require.Len(t, schemas, 2, "only the operation payload and its nested declaration are reachable")
	require.NotContains(t, schemas, "UnusedPayload")
	require.NotContains(t, schemas, "UnusedResult")
}

func componentSchemasFromSpec(t *testing.T, spec []byte) map[string]any {
	t.Helper()

	var doc map[string]any
	require.NoError(t, json.Unmarshal(spec, &doc))

	components, ok := doc["components"].(map[string]any)
	require.True(t, ok)

	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok)
	return schemas
}

var openAPIUnusedComponentSchemasDSL = func() {
	var reachableNested = dsl.Type("ReachableNested", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Required("id")
	})

	var reachablePayload = dsl.Type("ReachablePayload", func() {
		dsl.Attribute("nested", reachableNested)
		dsl.Required("nested")
	})

	_ = dsl.Type("UnusedPayload", func() {
		dsl.Attribute("shadow", dsl.String)
	})

	_ = dsl.ResultType("UnusedResult", func() {
		dsl.Attribute("unused", dsl.String)
		dsl.View("default", func() {
			dsl.Attribute("unused")
		})
	})

	dsl.Service("componentPruning", func() {
		dsl.Method("create", func() {
			dsl.Payload(reachablePayload)
			dsl.Result(dsl.Empty)
			dsl.HTTP(func() {
				dsl.POST("/components")
				dsl.Response(dsl.StatusNoContent)
			})
		})
	})
}
