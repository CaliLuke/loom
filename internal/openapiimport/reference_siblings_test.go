package openapiimport

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/libopenapi"

	"github.com/stretchr/testify/require"
)

func TestReferenceSiblingAssertionsAndAnnotations(t *testing.T) {
	for _, tc := range []struct {
		name, declaration, siblings, keyword string
		want                                 any
	}{
		{"string bounds", `{type: string, minLength: 3, pattern: '^[a-z]+$'}`, `maxLength: 8`, "maxLength", float64(8)},
		{"numeric bounds", `{type: integer, minimum: 2}`, `maximum: 10`, "maximum", float64(10)},
		{"membership", `{type: string, enum: [draft, ready, shipped]}`, `enum: [ready, shipped]`, "enum", []any{"ready", "shipped"}},
		{"array bounds", `{type: array, items: {type: string}, minItems: 1}`, `maxItems: 4`, "maxItems", float64(4)},
		{"required properties", `{type: object, properties: {id: {type: string}, name: {type: string}}, required: [id]}`, `required: [name]`, "required", []any{"name"}},
		{"format", `{type: string}`, `format: uuid`, "format", "uuid"},
		{"default", `{type: integer, minimum: 2}`, `default: 3`, "default", float64(3)},
		{"nullable enum excludes null", `{type: [string, 'null'], enum: [a]}`, `minLength: 1`, "minLength", float64(1)},
		{"annotations", `{type: string, description: Original}`, `description: Occurrence`, "description", "Occurrence"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(`openapi: 3.2.0
info: {title: Reference contracts, version: "1"}
paths:
 /products:
  post:
   operationId: createProduct
   requestBody:
    required: true
    content:
     application/json:
      schema:
       type: object
       properties:
        original: {$ref: '#/components/schemas/Value'}
        refined:
         $ref: '#/components/schemas/Value'
         ` + tc.siblings + `
       required: [original, refined]
   responses: {"204": {description: Accepted}}
components:
 schemas:
  Value: ` + tc.declaration + "\n")
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			module := requireRenderedDesignGenerates(t, rendered)
			contract := readGeneratedOpenAPIContract(t, module)
			validateReferenceSpec(t, module)
			validateReferenceRuntime(t, module, tc.name)
			op := operationFromImportedSpec(t, contract, "/products", "post")
			body := op["requestBody"].(map[string]any)
			if ref, ok := body["$ref"].(string); ok {
				body = contract["components"].(map[string]any)["requestBodies"].(map[string]any)[ref[len("#/components/requestBodies/"):]].(map[string]any)
			}
			schema := body["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
			if ref, ok := schema["$ref"].(string); ok {
				schema = referencedSchema(t, contract, ref)
			}
			properties := schema["properties"].(map[string]any)
			original := properties["original"].(map[string]any)
			refined := properties["refined"].(map[string]any)
			require.Equal(t, "#/components/schemas/Value", original["$ref"])
			require.Equal(t, original["$ref"], refined["$ref"])
			require.Equal(t, tc.want, refined[tc.keyword])
			base := referencedSchema(t, contract, original["$ref"].(string))
			require.NotEqual(t, tc.want, base[tc.keyword], "occurrence must not mutate its base")
		})
	}
}

func TestReferenceSiblingsUnsupportedIntersectionsFailClosed(t *testing.T) {
	for _, tc := range []struct{ name, declaration, siblings string }{
		{"incompatible type", `{type: string}`, `type: integer`},
		{"nullable narrowing", `{type: [string, 'null']}`, `type: string`},
		{"nullable membership narrowing", `{type: [string, 'null']}`, `enum: [yes]`},
		{"closed object addition", `{type: object, additionalProperties: false, properties: {name: {type: string}}}`, `properties: {extra: {type: string}}`},
		{"array item composition", `{type: array, items: {type: string}}`, `items: {maxLength: 8}`},
		{"object closure composition", `{type: object, properties: {name: {type: string}}}`, `additionalProperties: false`},
		{"composition sibling", `{type: string}`, `allOf: [{maxLength: 8}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(`openapi: 3.2.0
info: {title: Reference contracts, version: "1"}
paths: {}
components:
 schemas:
  Base: ` + tc.declaration + `
  Refined:
   $ref: '#/components/schemas/Base'
   ` + tc.siblings + "\n")
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-reference-siblings")
			fatal, _ := diagnostics.Classify(true)
			require.NotEmpty(t, fatal)
			_, err = Render(document, Options{PackageName: "design"})
			require.Error(t, err)
		})
	}
}

func validateReferenceSpec(t *testing.T, module string) {
	t.Helper()
	path := filepath.Join(module, "gen", "http", "openapi.json")
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(before)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)
	generate := exec.Command(filepath.Join(module, "loom"), "gen", "example.com/imported/design", "-o", ".")
	generate.Dir = module
	output, err := generate.CombinedOutput()
	require.NoError(t, err, string(output))
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, before, after, "independent generation preserves contract bytes")
}

func validateReferenceRuntime(t *testing.T, module, specimen string) {
	t.Helper()
	cases := map[string][]string{
		"inherited integer items":     {`{"original":{"ns":[1]},"refined":{"ns":[1,2]}}`, `{"original":{"ns":[3]},"refined":{"ns":[1]}}`, `{"original":{"ns":[1]},"refined":{"ns":[1,2,1]}}`, `{"original":{"ns":[1]},"refined":{"ns":[3]}}`},
		"object properties":           {`{"original":{"id":"a"},"refined":{"id":"b","name":"abc","enabled":true}}`, `{"original":{"id":""},"refined":{"id":"b","name":"abc"}}`, `{"original":{"id":"a"},"refined":{"id":"b","name":"a"}}`, `{"original":{"id":"a"},"refined":{"id":"b","name":"abcdefghi"}}`},
		"nullable enum excludes null": {`{"original":"a","refined":"a"}`, `{"original":"a","refined":null}`, `{"original":null,"refined":"a"}`},
		"string bounds":               {`{"original":"abc","refined":"abcd"}`, `{"original":"a","refined":"abcd"}`, `{"original":"abc","refined":"abcdefghi"}`},
		"numeric bounds":              {`{"original":2,"refined":5}`, `{"original":1,"refined":5}`, `{"original":2,"refined":11}`},
		"membership":                  {`{"original":"draft","refined":"ready"}`, `{"original":"other","refined":"ready"}`, `{"original":"draft","refined":"draft"}`},
		"array bounds":                {`{"original":["a"],"refined":["b"]}`, `{"original":[],"refined":["b"]}`, `{"original":["a"],"refined":["a","b","c","d","e"]}`},
		"required properties":         {`{"original":{"id":"a"},"refined":{"id":"b","name":""}}`, `{"original":{},"refined":{"id":"b","name":""}}`, `{"original":{"id":"a"},"refined":{"id":"b"}}`},
		"format":                      {`{"original":"anything","refined":"123e4567-e89b-12d3-a456-426614174000"}`, `{"original":"anything","refined":"bad"}`},
		"default":                     {`{"original":2,"refined":3}`, `{"original":1,"refined":3}`, `{"original":2,"refined":1}`},
		"annotations":                 {`{"original":"base","refined":"local"}`},
	}
	wires, ok := cases[specimen]
	require.True(t, ok)
	source := `package imported_test
import (
 "testing"
 "encoding/json/v2"
 server "example.com/imported/gen/http/reference_contracts/server"
)
func TestReferenceContract(t *testing.T) {
`
	for index, wire := range wires {
		source += fmt.Sprintf(`{
 var body server.CreateProductRequestBody
 if err := json.Unmarshal([]byte(%q), &body); err != nil {
  t.Fatal(err)
 }
 err := server.ValidateCreateProductRequestBody(&body)
 if (err == nil) != %t {
  t.Errorf("case %%d: unexpected validation %%v", %d, err)
 }
}
`, wire, index == 0, index)
	}
	source += "}\n"
	require.NoError(t, os.WriteFile(filepath.Join(module, "reference_test.go"), []byte(source), 0o600))
	command := exec.Command("go", "test", "-mod=mod", ".")
	command.Dir = module
	output, err := command.CombinedOutput()
	require.NoError(t, err, string(output))
}
