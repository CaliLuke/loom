package openapiv3_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestRenderedSpecDeduplicatesGeneratedRequestBodiesAndUnionEnvelopes(t *testing.T) {
	artifacts := renderOpenAPIArtifacts(t, testdata.OpenAPISchemaDedupDSL)
	spec := decodeOpenAPIJSON(t, artifacts.JSON)

	requestExamples := make(map[string]struct{})
	for _, path := range []string{"/first", "/second"} {
		operation := requireOperation(t, spec, path, "post")
		requestBody := requireMap(t, operation["requestBody"], path+" request body")
		content := requireMap(t, requestBody["content"], path+" request content")
		media := requireMap(t, content["application/json"], path+" request media")
		schema := requireMap(t, media["schema"], path+" request schema")
		require.Equal(t, "#/components/schemas/FirstRequestBody", schema["$ref"])
		require.NotContains(t, schema, "oneOf")
		example := requireMap(t, media["example"], path+" request example")
		require.NotEmpty(t, requireString(t, example["name"], path+" request example name"))
		requestExamples[fmt.Sprint(media["example"])] = struct{}{}
	}
	require.Len(t, requestExamples, 2, "duplicate schemas retain their occurrence-specific request examples")

	components := requireMap(t, spec["components"], "components")
	schemas := requireMap(t, components["schemas"], "component schemas")
	require.Contains(t, schemas, "FirstRequestBody")
	require.NotContains(t, schemas, "SecondRequestBody")
	require.NotContains(t, components, "requestBodies", "union bodies with different examples must remain inline")

	var commonUnionSchema map[string]any
	unionExamples := make(map[string]struct{})
	for _, path := range []string{"/union/first", "/union/second"} {
		operation := requireOperation(t, spec, path, "post")
		requestBody := requireMap(t, operation["requestBody"], path+" request body")
		require.NotContains(t, requestBody, "$ref")
		content := requireMap(t, requestBody["content"], path+" request content")
		media := requireMap(t, content["application/json"], path+" request media")
		schema := requireMap(t, media["schema"], path+" request schema")
		require.Equal(t, media["example"], schema["example"])
		example := requireMap(t, media["example"], path+" request example")
		tag := requireString(t, example["type"], path+" request example tag")
		value := requireMap(t, example["value"], path+" request example value")
		switch tag {
		case "AlphaSchemaDedup":
			require.NotEmpty(t, requireString(t, value["alpha"], path+" alpha example"))
		case "BetaSchemaDedup":
			require.NotEmpty(t, requireString(t, value["beta"], path+" beta example"))
		default:
			t.Fatalf("unexpected union example tag %q", tag)
		}
		unionExamples[fmt.Sprint(media["example"])] = struct{}{}

		withoutExample := make(map[string]any, len(schema)-1)
		for key, value := range schema {
			if key != "example" {
				withoutExample[key] = value
			}
		}
		if commonUnionSchema == nil {
			commonUnionSchema = withoutExample
		} else {
			require.Equal(t, commonUnionSchema, withoutExample)
		}
	}
	require.Len(t, unionExamples, 2, "union request bodies retain their occurrence-specific examples")
	require.Equal(t, "object", commonUnionSchema["type"])
	require.Equal(t, []any{
		map[string]any{"$ref": "#/components/schemas/AlphaSchemaDedupOrBetaSchemaDedupAlphaSchemaDedupEnvelope"},
		map[string]any{"$ref": "#/components/schemas/AlphaSchemaDedupOrBetaSchemaDedupBetaSchemaDedupEnvelope"},
	}, commonUnionSchema["oneOf"])
	require.Equal(t, map[string]any{
		"propertyName": "type",
		"mapping": map[string]any{
			"AlphaSchemaDedup": "#/components/schemas/AlphaSchemaDedupOrBetaSchemaDedupAlphaSchemaDedupEnvelope",
			"BetaSchemaDedup":  "#/components/schemas/AlphaSchemaDedupOrBetaSchemaDedupBetaSchemaDedupEnvelope",
		},
	}, commonUnionSchema["discriminator"])

	envelopes := 0
	for name := range schemas {
		if strings.HasSuffix(name, "Envelope") {
			envelopes++
		}
	}
	require.Equal(t, 2, envelopes)
}
