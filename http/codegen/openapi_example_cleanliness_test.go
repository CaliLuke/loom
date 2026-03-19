package codegen

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"goa.design/goa/v3/http/codegen/openapi"
	openapiv3 "goa.design/goa/v3/http/codegen/openapi/v3"
	"goa.design/goa/v3/http/codegen/testdata"
)

func TestOpenAPIClosedObjectWrapperUnionExamplesAreSuppressed(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MealPlannerDSL)
	openapi.Definitions = make(map[string]*openapi.Schema)

	spec := renderOpenAPIJSON(t, openapiv3.Files, root)
	parseOpenAPIV3Document(t, spec)

	recipeSelection := componentSchemaFromSpec(t, spec, "RecipeSelection")
	require.NotContains(t, recipeSelection, "example")

	properties, ok := recipeSelection["properties"].(map[string]any)
	require.True(t, ok)
	modeSchema, ok := properties["selection"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, modeSchema, "example")

	previewBody := componentSchemaFromSpec(t, spec, "PreviewSelectionRequestBody")
	require.NotContains(t, previewBody, "example")

	previewMediaType := operationMediaTypeFromSpec(t, spec, "/plans/preview", "post", "application/json")
	require.NotContains(t, previewMediaType, "example")
}

func TestOpenAPIExampleFalseSuppressesWrapperRequestExamples(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MealPlannerDSL)
	openapi.Definitions = make(map[string]*openapi.Schema)

	spec := renderOpenAPIJSON(t, openapiv3.Files, root)
	parseOpenAPIV3Document(t, spec)

	suppressedMediaType := operationMediaTypeFromSpec(t, spec, "/plans/preview-suppressed", "post", "application/json")
	require.NotContains(t, suppressedMediaType, "example")
}

func TestOpenAPIMultipartBinaryExamplesUseStrings(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MealPlannerDSL)
	openapi.Definitions = make(map[string]*openapi.Schema)

	spec := renderOpenAPIJSON(t, openapiv3.Files, root)
	parseOpenAPIV3Document(t, spec)

	importBody := componentSchemaFromSpec(t, spec, "ImportPantryRequestBody")
	properties, ok := importBody["properties"].(map[string]any)
	require.True(t, ok)
	fileSchema, ok := properties["file"].(map[string]any)
	require.True(t, ok)

	if example, ok := fileSchema["example"]; ok {
		_, isString := example.(string)
		require.True(t, isString, "binary property example must be a string when present")
	}

	if example, ok := importBody["example"].(map[string]any); ok {
		fileExample, hasFile := example["file"]
		if hasFile {
			_, isString := fileExample.(string)
			require.True(t, isString, "binary object example must use a string for file when present")
		}
	}

	importMediaType := operationMediaTypeFromSpec(t, spec, "/pantries/{pantry_id}/imports", "post", "multipart/form-data")
	if example, ok := importMediaType["example"].(map[string]any); ok {
		fileExample, hasFile := example["file"]
		if hasFile {
			_, isString := fileExample.(string)
			require.True(t, isString, "binary media type example must use a string for file when present")
		}
	}
}

func TestOpenAPISSEStreamingResponsesRemainHTTP200(t *testing.T) {
	root := RunHTTPDSL(t, testdata.MealPlannerDSL)
	openapi.Definitions = make(map[string]*openapi.Schema)

	spec := renderOpenAPIJSON(t, openapiv3.Files, root)
	parseOpenAPIV3Document(t, spec)

	op := operationFromSpec(t, spec, "/events", "get")
	responses, ok := op["responses"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, responses, "200")
	require.NotContains(t, responses, "101")
}

func TestOpenAPIClosedObjectUnionCollectionExamplesAreSuppressed(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ActivityFeedDSL)
	openapi.Definitions = make(map[string]*openapi.Schema)

	spec := renderOpenAPIJSON(t, openapiv3.Files, root)
	parseOpenAPIV3Document(t, spec)

	activityEnvelope := componentSchemaFromSpec(t, spec, "ActivityEnvelope")
	require.NotContains(t, activityEnvelope, "example")

	activitiesMediaType := operationResponseMediaTypeFromSpec(t, spec, "/projects/{project_id}/activities", "get", "200", "application/json")
	require.NotContains(t, activitiesMediaType, "example")

	schema, ok := activitiesMediaType["schema"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, schema, "example")
}

func TestOpenAPIStreamingSyntheticExamplesAreSuppressedWhenIncomplete(t *testing.T) {
	root := RunHTTPDSL(t, testdata.StreamingPartialExamplesDSL)
	openapi.Definitions = make(map[string]*openapi.Schema)

	spec := renderOpenAPIJSON(t, openapiv3.Files, root)
	parseOpenAPIV3Document(t, spec)

	sseMediaType := operationResponseMediaTypeFromSpec(t, spec, "/events", "get", "200", "application/json")
	require.NotContains(t, sseMediaType, "example")

	wsMediaType := operationResponseMediaTypeFromSpec(t, spec, "/ws/projects/{projectID}", "get", "101", "application/json")
	require.NotContains(t, wsMediaType, "example")
}

func componentSchemaFromSpec(t *testing.T, spec []byte, name string) map[string]any {
	t.Helper()

	var doc map[string]any
	require.NoError(t, json.Unmarshal(spec, &doc))

	components, ok := doc["components"].(map[string]any)
	require.True(t, ok)
	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok)
	schema, ok := schemas[name].(map[string]any)
	require.True(t, ok, name)
	return schema
}

func operationMediaTypeFromSpec(t *testing.T, spec []byte, path, method, contentType string) map[string]any {
	t.Helper()

	op := operationFromSpec(t, spec, path, method)
	requestBody, ok := op["requestBody"].(map[string]any)
	require.True(t, ok)
	content, ok := requestBody["content"].(map[string]any)
	require.True(t, ok)
	mediaType, ok := content[contentType].(map[string]any)
	require.True(t, ok, contentType)
	return mediaType
}

func operationResponseMediaTypeFromSpec(t *testing.T, spec []byte, path, method, status, contentType string) map[string]any {
	t.Helper()

	op := operationFromSpec(t, spec, path, method)
	responses, ok := op["responses"].(map[string]any)
	require.True(t, ok)
	response, ok := responses[status].(map[string]any)
	require.True(t, ok, status)
	content, ok := response["content"].(map[string]any)
	require.True(t, ok)
	mediaType, ok := content[contentType].(map[string]any)
	require.True(t, ok, contentType)
	return mediaType
}
