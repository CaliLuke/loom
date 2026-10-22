package openapiv3_test

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedDocumentationUnionShapes(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.RawRequestBodyOpenAPIDSL)
	var document map[string]any
	require.NoError(t, json.Unmarshal(artifacts.JSON, &document))
	resolve := func(raw any) map[string]any {
		schema := requireMap(t, raw, "schema")
		if ref, ok := schema["$ref"].(string); ok {
			return componentSchema(t, document, strings.TrimPrefix(ref, "#/components/schemas/"))
		}
		return schema
	}
	exchange := requireOperation(t, document, "/exchange", "post")
	request := requireMap(t, exchange["requestBody"], "request body")
	if ref, ok := request["$ref"].(string); ok {
		components := requireMap(t, document["components"], "components")
		bodies := requireMap(t, components["requestBodies"], "request bodies")
		request = requireMap(t, bodies[strings.TrimPrefix(ref, "#/components/requestBodies/")], "request body")
	}
	content := requireMap(t, request["content"], "request content")
	media := requireMap(t, content["application/json"], "request media")
	scalarChoice := resolve(media["schema"])
	require.NotContains(t, scalarChoice, "discriminator")
	branches := requireSlice(t, scalarChoice["oneOf"], "scalar/object branches")
	require.Len(t, branches, 2)
	kinds := make([]any, 0, len(branches))
	for _, branch := range branches {
		kinds = append(kinds, resolve(branch)["type"])
	}
	require.ElementsMatch(t, []any{"string", "object"}, kinds)

	response := resolve(requireResponseMediaType(t, exchange, "application/json")["schema"])
	require.NotContains(t, response, "discriminator")
	branches = requireSlice(t, response["oneOf"], "array/object branches")
	require.Len(t, branches, 2)
	kinds = kinds[:0]
	for _, branch := range branches {
		schema := resolve(branch)
		kinds = append(kinds, schema["type"])
		if schema["type"] == "object" {
			require.Equal(t, []any{"results"}, schema["required"])
			schema = resolve(requireMap(t, schema["properties"], "page fields")["results"])
		}
		item := resolve(schema["items"])
		require.Equal(t, []any{"choice"}, item["required"])
		choice := resolve(requireMap(t, item["properties"], "item fields")["choice"])
		require.Contains(t, choice, "discriminator", "nested tagged unions retain authored semantics")
		require.Len(t, requireSlice(t, choice["oneOf"], "nested branches"), 2)
	}
	require.ElementsMatch(t, []any{"array", "object"}, kinds)
	typed := requireOperation(t, document, "/choice", "post")
	typedChoice := resolve(requireResponseMediaType(t, typed, "application/json")["schema"])
	require.Contains(t, typedChoice, "discriminator", "documentation must not alter the shared typed response")
}

func TestDocumentationBodyExplicitComponentNames(t *testing.T) {
	for _, named := range []bool{false, true} {
		name := "inline"
		if named {
			name = "named"
		}
		t.Run(name, func(t *testing.T) {
			artifacts := renderOpenAPIArtifacts(t, func() {
				body := Type("Source", func() {
					Attribute("label", String)
				})
				Service("documentation", func() {
					Method("raw", func() {
						HTTP(func() {
							POST("/raw")
							SkipRequestBodyEncodeDecode()
							OpenAPIRequestBody(body, "application/json", true, func() {
								Meta("openapi:typename", "Wire_Request")
							})
							SkipResponseBodyEncodeDecode()
							Response(StatusOK, func() {
								if named {
									OpenAPIBody(body, func() {
										Meta("openapi:typename", "Wire_Response")
									})
								} else {
									OpenAPIBody(func() {
										Attribute("label", String)
										Meta("openapi:typename", "Wire_Response")
									})
								}
							})
						})
					})
				})
			})
			var document map[string]any
			require.NoError(t, json.Unmarshal(artifacts.JSON, &document))
			for _, name := range []string{"Wire_Request", "Wire_Response"} {
				schema := componentSchema(t, document, name)
				require.Equal(t, "object", schema["type"])
				require.Contains(t, requireMap(t, schema["properties"], "fields"), "label")
			}
		})
	}
}
