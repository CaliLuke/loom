package openapiv3_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestEffectiveWireBindingsOpenAPI(t *testing.T) {
	for _, test := range []struct {
		name string
		dsl  func()
	}{
		{"mappings", testdata.ExplicitAuthorizationWireBindingsDSL("http")},
		{"scope", testdata.EffectiveWireBindingScopeDSL("http")},
		{"oas32", testdata.OpenAPI32FeaturesDSL},
	} {
		t.Run(test.name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifacts(t, test.dsl)
			spec := decodeOpenAPIJSON(t, artifacts.JSON)
			paths := requireMap(t, spec["paths"], "paths")
			switch test.name {
			case "scope":
				operation := requireMap(t, requireMap(t, paths["/"], "path")["post"], "operation")
				require.Equal(t, []any{map[string]any{"local": []any{}}}, operation["security"])
				require.NotNil(t, operation["requestBody"])
				schema := requireComponentSchema(t, spec, "ReadRequestBody")
				require.Contains(t, requireMap(t, schema["properties"], "properties"), "other")
			case "mappings":
				for _, path := range []string{"/either", "/explicit"} {
					operation := requireMap(t, requireMap(t, paths[path], "path")["post"], "operation")
					require.Equal(t, []any{map[string]any{"basic": []any{}}, map[string]any{"jwt": []any{}}}, operation["security"])
				}
				operation := requireMap(t, requireMap(t, paths["/and-key"], "path")["post"], "operation")
				require.Equal(t, []any{map[string]any{"jwt": []any{}, "key": []any{}}}, operation["security"])
				components := requireMap(t, spec["components"], "components")
				schemes := requireMap(t, components["securitySchemes"], "schemes")
				key := requireMap(t, schemes["key"], "key scheme")
				require.Equal(t, "header", key["in"])
				require.Equal(t, "X-Api-Key", key["name"])
			default:
				operation := requireMap(t, requireMap(t, paths["/books"], "path")["query"], "operation")
				require.Equal(t, []any{map[string]any{"https://auth.example.com/security/external": []any{}}, map[string]any{"device": []any{"catalog:read"}}}, operation["security"])
			}
			lintEffectiveCompositionArtifacts(t, artifacts)
		})
	}
}
