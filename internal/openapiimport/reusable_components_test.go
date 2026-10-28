package openapiimport

import (
	"encoding/json/v2"
	"os"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestReusableTransportComponentRoundTrip(t *testing.T) {
	source, err := os.ReadFile("testdata/supported.yaml")
	require.NoError(t, err)
	var input map[string]any
	require.NoError(t, yaml.Unmarshal(source, &input))
	input["openapi"] = "3.2.0"
	components := input["components"].(map[string]any)
	paths := input["paths"].(map[string]any)
	create := paths["/pets"].(map[string]any)["post"].(map[string]any)
	get := paths["/pets/{pet_id}"].(map[string]any)["get"].(map[string]any)
	response := get["responses"].(map[string]any)["200"].(map[string]any)
	header := response["headers"].(map[string]any)["X-Trace-ID"]
	components["headers"] = map[string]any{"Trace": map[string]any{"$ref": "#/components/headers/ZTrace"}, "ZTrace": header}
	response["headers"] = map[string]any{"X-Trace-ID": map[string]any{"$ref": "#/components/headers/Trace", "description": ""}}
	response["summary"] = "Base success"
	components["responses"] = map[string]any{"PetResponse": map[string]any{"$ref": "#/components/responses/ZPetResponse", "summary": "Alias success"}, "ZPetResponse": response}
	get["responses"].(map[string]any)["200"] = map[string]any{"$ref": "#/components/responses/PetResponse", "description": "Local success", "summary": ""}
	components["requestBodies"] = map[string]any{"PetInput": map[string]any{"$ref": "#/components/requestBodies/ZPetInput"}, "ZPetInput": create["requestBody"]}
	create["requestBody"] = map[string]any{"$ref": "#/components/requestBodies/PetInput", "description": ""}

	// A second occurrence inherits both components without local overrides.
	otherWire, err := json.Marshal(paths["/pets/{pet_id}"], json.Deterministic(true))
	require.NoError(t, err)
	var other map[string]any
	require.NoError(t, json.Unmarshal(otherWire, &other))
	otherGet := other["get"].(map[string]any)
	otherGet["operationId"] = "pets.getOther"
	otherGet["responses"].(map[string]any)["200"] = map[string]any{"$ref": "#/components/responses/PetResponse"}
	paths["/other/{pet_id}"] = other
	wire, err := json.Marshal(input, json.Deterministic(true))
	require.NoError(t, err)
	document, diagnostics, err := Analyze(wire)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	before, err := json.Marshal(document, json.Deterministic(true))
	require.NoError(t, err)
	design, err := Render(document, Options{})
	require.NoError(t, err)
	after, err := json.Marshal(document, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, before, after)
	module := requireRenderedDesignGenerates(t, design)
	validateReferenceSpec(t, module)
	contract := readGeneratedOpenAPIContract(t, module)
	generated := contract["components"].(map[string]any)
	for _, tc := range []struct{ kind, alias, target string }{
		{"requestBodies", "PetInput", "ZPetInput"},
		{"responses", "PetResponse", "ZPetResponse"},
		{"headers", "Trace", "ZTrace"},
	} {
		entries := generated[tc.kind].(map[string]any)
		require.Contains(t, entries, tc.target)
		require.Equal(t, "#/components/"+tc.kind+"/"+tc.target, entries[tc.alias].(map[string]any)["$ref"])
	}
	bodies := generated["requestBodies"].(map[string]any)
	require.Equal(t, "Pet to create.", bodies["ZPetInput"].(map[string]any)["description"])
	responses := generated["responses"].(map[string]any)
	require.Equal(t, "Found pet.", responses["ZPetResponse"].(map[string]any)["description"])
	require.Equal(t, "Base success", responses["ZPetResponse"].(map[string]any)["summary"])
	require.Equal(t, "Alias success", responses["PetResponse"].(map[string]any)["summary"])
	generatedPaths := contract["paths"].(map[string]any)
	actualCreate := generatedPaths["/pets"].(map[string]any)["post"].(map[string]any)
	require.Equal(t, map[string]any{"$ref": "#/components/requestBodies/PetInput", "description": ""}, actualCreate["requestBody"])
	inheritedGet := generatedPaths["/other/{pet_id}"].(map[string]any)["get"].(map[string]any)
	require.Equal(t, map[string]any{"$ref": "#/components/responses/PetResponse"}, inheritedGet["responses"].(map[string]any)["200"])
	require.Equal(t, "Request trace identifier.", generated["headers"].(map[string]any)["ZTrace"].(map[string]any)["description"])
	actualGet := generatedPaths["/pets/{pet_id}"].(map[string]any)["get"].(map[string]any)
	require.Equal(t, map[string]any{"$ref": "#/components/responses/PetResponse", "description": "Local success", "summary": ""}, actualGet["responses"].(map[string]any)["200"])
}
