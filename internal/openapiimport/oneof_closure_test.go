package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"
)

type oneOfClosureProfile struct {
	name, branches string
	values         []oneOfSemanticCase
}

// The first six profiles are the design's closure table. The last two distinguish
// additionalProperties presence from absence. Values align with the mounted
// UntaggedRoots harness; this importer oracle reads schema structure independently.
func oneOfClosureProfiles() []oneOfClosureProfile {
	return []oneOfClosureProfile{
		{"open-pair", `[{"$ref":"#/components/schemas/X"},{"$ref":"#/components/schemas/Y"}]`, []oneOfSemanticCase{{`{"x":1}`, false}, {`{"y":1}`, true}, {`{}`, true}, {`{"z":1}`, false}, {`{"x":1,"y":2}`, false}}},
		{"discriminated-pair", `[{"$ref":"#/components/schemas/X"},{"$ref":"#/components/schemas/Kind"}]`, []oneOfSemanticCase{{`{"kind":"a","x":1}`, false}, {`{"kind":"a"}`, true}, {`{"x":1}`, true}, {`{"kind":"b"}`, false}, {`{"x":1,"z":2}`, false}}},
		{"map", `[{"type":"object","additionalProperties":{"type":"integer"}},{"type":"string"}]`, []oneOfSemanticCase{{`{"z":1}`, true}, {`{"z":"bad"}`, false}, {`{}`, true}, {`"a"`, true}, {`null`, false}}},
		{"any", `[{}, {"type":"string"}]`, []oneOfSemanticCase{{`{"z":1}`, false}, {`{}`, true}, {`7`, true}, {`null`, true}, {`"a"`, false}}},
		{"nested", `[{"oneOf":[{"$ref":"#/components/schemas/X"},{"type":"string"}]},{"type":"boolean"}]`, []oneOfSemanticCase{{`{"x":1,"z":2}`, false}, {`{"x":1}`, true}, {`true`, true}, {`"a"`, true}, {`{}`, false}}},
		{"additional", `[{"$ref":"#/components/schemas/Additional"},{"type":"string"}]`, []oneOfSemanticCase{{`{"x":1,"z":2}`, true}, {`{"x":1,"z":"bad"}`, false}, {`{"z":2}`, false}, {`"a"`, true}}},
		{"explicit-open", `[{"type":"object","additionalProperties":true},{"type":"string"}]`, []oneOfSemanticCase{{`{"z":2}`, true}, {`{"z":"a"}`, true}, {`{}`, true}, {`null`, false}}},
		{"closed-object", `[{"$ref":"#/components/schemas/Closed"},{"type":"string"}]`, []oneOfSemanticCase{{`{"x":1}`, true}, {`{"x":1,"z":2}`, false}, {`{}`, false}, {`"a"`, true}}},
	}
}

func oneOfClosureDocument(t *testing.T, profiles ...oneOfClosureProfile) []byte {
	if len(profiles) == 0 {
		profiles = oneOfClosureProfiles()
	}
	t.Helper()
	var components map[string]any
	require.NoError(t, json.Unmarshal([]byte(`{"schemas":{
 "X":{"type":"object","properties":{"x":{"type":"integer"}},"required":["x"]},
 "Y":{"type":"object","properties":{"y":{"type":"integer"}}},
 "Kind":{"type":"object","properties":{"kind":{"type":"string","enum":["a"]}},"required":["kind"]},
 "Additional":{"type":"object","properties":{"x":{"type":"integer"}},"required":["x"],"additionalProperties":{"type":"integer"}},
 "Closed":{"type":"object","properties":{"x":{"type":"integer"}},"required":["x"],"additionalProperties":false}
 }}`), &components))
	paths := make(map[string]any)
	for _, profile := range profiles {
		var branches []any
		require.NoError(t, json.Unmarshal([]byte(profile.branches), &branches))
		schema := map[string]any{"oneOf": branches, "unevaluatedProperties": false}
		paths["/"+profile.name] = map[string]any{"post": map[string]any{"operationId": profile.name, "responses": map[string]any{"200": map[string]any{"description": "Closure profile", "content": map[string]any{"application/json": map[string]any{"schema": schema}}}}}}
	}
	source, err := json.Marshal(map[string]any{"openapi": "3.2.0", "info": map[string]any{"title": "Closure profiles", "version": "1"}, "paths": paths, "components": components}, json.Deterministic(true))
	require.NoError(t, err)
	return source
}

func TestOneOfClosureSourceProfiles(t *testing.T) {
	source := oneOfClosureDocument(t)
	var document map[string]any
	require.NoError(t, json.Unmarshal(source, &document))
	requireOneOfClosureProfiles(t, document)
	imported, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(imported, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("openapi:unevaluatedProperties", "false")`)
}

// Closure is checked by a small explicit-value oracle, never gojsonschema's
// Draft 7 engine (which ignores unevaluatedProperties). Generated wire matching
// of these same profiles lives in UntaggedRootsHarness, not in this oracle.
func TestOneOfClosure(t *testing.T) {
	source := oneOfClosureDocument(t)
	imported, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(imported, Options{PackageName: "design"})
	require.NoError(t, err)
	directory := requireRenderedDesignGenerates(t, design)
	for _, filename := range []string{"openapi.json", "openapi.yaml"} {
		wire, err := os.ReadFile(filepath.Join(directory, "gen", "http", filename))
		require.NoError(t, err)
		parsed, err := libopenapi.NewDocument(wire)
		require.NoError(t, err)
		_, err = parsed.BuildV3Model()
		require.NoError(t, err)
	}
	var original map[string]any
	require.NoError(t, json.Unmarshal(source, &original))
	requireOneOfClosureProfiles(t, original)
	requireOneOfClosureProfiles(t, readGeneratedOpenAPIContract(t, directory))
}

func requireOneOfClosureProfiles(t *testing.T, document map[string]any) {
	t.Helper()
	for _, profile := range oneOfClosureProfiles() {
		t.Run(profile.name, func(t *testing.T) {
			operation := document["paths"].(map[string]any)["/"+profile.name].(map[string]any)["post"].(map[string]any)
			schema := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
			occurrence := resolveOneOfSpecReference(t, document, schema)
			require.Equal(t, false, occurrence["unevaluatedProperties"])
			require.Len(t, occurrence["oneOf"], 2)
			require.Equal(t, 1, countOneOfClosureKeywords(t, document, occurrence, make(map[string]bool)), "closure must occur only at the outer union, never on its branches")
			for _, tc := range profile.values {
				var value any
				require.NoError(t, json.Unmarshal([]byte(tc.wire), &value))
				valid, _ := evaluateOneOfClosureSchema(t, document, schema, value)
				require.Equal(t, tc.valid, valid, "%s", tc.wire)
			}
		})
	}
}

func countOneOfClosureKeywords(t *testing.T, document map[string]any, value any, seen map[string]bool) int {
	t.Helper()
	count := 0
	switch actual := value.(type) {
	case map[string]any:
		if reference, ok := actual["$ref"].(string); ok && !seen[reference] {
			seen[reference] = true
			count += countOneOfClosureKeywords(t, document, oneOfSpecReferenceTarget(t, document, reference), seen)
		}
		for key, child := range actual {
			if key == "unevaluatedProperties" {
				count++
			}
			count += countOneOfClosureKeywords(t, document, child, seen)
		}
	case []any:
		for _, child := range actual {
			count += countOneOfClosureKeywords(t, document, child, seen)
		}
	}
	return count
}
