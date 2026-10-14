package openapiimport

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testprocess"
)

func TestImportedObjectFieldNames(t *testing.T) {
	for _, tc := range []struct {
		name, properties, override string
	}{
		{"jsonld", `"@type": {type: string}, type: {type: string}`, "Type2"},
		{"hyphen", `foo-bar: {type: string}, fooBar: {type: string}`, "FooBar2"},
		{"underscore", `user_id: {type: string}, userId: {type: string}`, "UserID2"},
		{"initialism", `UserID: {type: string}, user_id: {type: string}`, "UserID2"},
		{"occupied suffix", `"@type": {type: string}, type: {type: string}, type2: {type: string}`, "Type3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`openapi: 3.1.1
info: {title: Fields, version: '1'}
paths:
  /value:
    get:
      operationId: value
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Value'}
components:
  schemas:
    Value: {type: object, properties: {%s}}
`, tc.properties))
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{})
			require.NoError(t, err)
			require.Contains(t, string(rendered), fmt.Sprintf(`Meta("struct:field:name", %q)`, tc.override))
		})
	}
}

func TestImportedInheritedFieldNamesGenerate(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Inherited fields, version: '1'}
paths:
  /value:
    get:
      operationId: value
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Derived'}
components:
  schemas:
    Base:
      type: object
      properties:
        type: {type: string}
    Derived:
      allOf:
        - {$ref: '#/components/schemas/Base'}
        - type: object
          properties:
            '@type': {type: string}
            type2: {type: string}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-allof-flattened")
	rendered, err := Render(document, Options{})
	require.NoError(t, err)
	requireRenderedDesignGenerates(t, rendered)
}

func TestImportedObjectFieldFamilyGenerates(t *testing.T) {
	source, err := os.ReadFile("testdata/object_field_names.yaml")
	require.NoError(t, err)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	fatal, _ := diagnostics.Classify(true)
	require.Empty(t, fatal)
	before, err := json.Marshal(document, json.Deterministic(true))
	require.NoError(t, err)
	rendered, err := Render(document, Options{})
	require.NoError(t, err)
	again, err := Render(document, Options{})
	require.NoError(t, err)
	require.Equal(t, rendered, again)
	after, err := json.Marshal(document, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, before, after)
	protobuf, err := os.ReadFile("testdata/object_field_names_grpc.go.txt")
	require.NoError(t, err)
	rendered = append(rendered, protobuf...)
	module := requireRenderedDesignGenerates(t, rendered)
	harness, err := os.ReadFile("testdata/object_field_names_wire.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(module, "wire_test.go"), harness, 0o600))
	for _, args := range [][]string{{"test", "-race", "-mod=mod", "-v", "./..."}, {"vet", "./..."}} {
		cmd := testprocess.CommandContext(t.Context(), "go", args...)
		cmd.Dir = module
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		t.Log(string(output))
	}
	generated, err := os.ReadFile(filepath.Join(module, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(generated)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)
	contract := readGeneratedOpenAPIContract(t, module)
	schemas := contract["components"].(map[string]any)["schemas"].(map[string]any)
	for _, tc := range []struct {
		name string
		keys []string
	}{
		{"Fields", []string{"@type", "type", "type2", "foo-bar", "fooBar", "user_id", "userId", "nested", "base", "derived"}},
		{"Base", []string{"type", "error"}},
		{"Derived", []string{"type", "error", "@type", "type2"}},
		{"Failure", []string{"type", "error", "@type", "errorField", "errorField2", "loomErrorName", "@loomErrorName"}},
	} {
		properties := schemas[tc.name].(map[string]any)["properties"].(map[string]any)
		require.ElementsMatch(t, tc.keys, objectPropertyKeys(properties))
	}
}

func objectPropertyKeys(properties map[string]any) []string {
	keys := make([]string, 0, len(properties))
	for key := range properties {
		keys = append(keys, key)
	}
	return keys
}
