package codegen

import (
	"encoding/json/v2"
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
	require.Contains(t, schemas, "ReachablePayload")
	require.Contains(t, schemas, "ReachableNested")
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
