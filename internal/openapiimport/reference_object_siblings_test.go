package openapiimport

import (
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferenceObjectSiblingProperties(t *testing.T) {
	for _, name := range []string{"inline field", "named field"} {
		t.Run(name, func(t *testing.T) {
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
         properties:
          name: {minLength: 3}
          enabled: {type: boolean}
         required: [name]
       required: [original, refined]
   responses: {"204": {description: Accepted}}
components:
 schemas:
  Value:
   type: object
   properties:
    id: {type: string, minLength: 1}
    name: {type: string, maxLength: 8}
   required: [id]
`)
			if name == "named field" {
				source = []byte(strings.Replace(string(source), "name: {type: string, maxLength: 8}", "name: {$ref: '#/components/schemas/Name'}", 1) + "  Name: {type: string, maxLength: 8}\n")
			}
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			module := requireRenderedDesignGenerates(t, rendered)
			validateReferenceSpec(t, module)
			validateReferenceRuntime(t, module, "object properties")
			contract := readGeneratedOpenAPIContract(t, module)
			operation := operationFromImportedSpec(t, contract, "/products", "post")
			body := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
			if ref, ok := body["$ref"].(string); ok {
				body = referencedSchema(t, contract, ref)
			}
			refined := body["properties"].(map[string]any)["refined"].(map[string]any)
			if ref, ok := refined["$ref"].(string); ok {
				refined = referencedSchema(t, contract, ref)
			}
			branches := refined["allOf"].([]any)
			require.Equal(t, "#/components/schemas/Value", branches[0].(map[string]any)["$ref"])
			base := referencedSchema(t, contract, "#/components/schemas/Value")
			require.NotContains(t, base["properties"].(map[string]any)["name"], "minLength")
			// These generated document-level fields have separate import tickets;
			// the schema reference graph is retained intact for this round trip.
			delete(contract, "$self")
			delete(contract, "jsonSchemaDialect")
			delete(contract, "servers")
			wire, err := json.Marshal(contract, json.Deterministic(true))
			require.NoError(t, err)
			reimported, diagnostics, err := Analyze(wire)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(reimported, Options{PackageName: "design"})
			require.NoError(t, err)
			validateReferenceRuntime(t, requireRenderedDesignGenerates(t, design), "object properties")
		})
	}
}

func TestReferenceObjectAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, base, siblings string
		accepted             bool
	}{
		{"closed existing property", `{type: object, additionalProperties: false, properties: {name: {type: string}}}`, `properties: {name: {minLength: 2}}`, true},
		{"closed sibling complete keys", `{type: object, properties: {name: {type: string}}}`, `properties: {name: {minLength: 2}}
   additionalProperties: false`, true},
		{"closed sibling loses key", `{type: object, properties: {id: {type: string}, name: {type: string}}}`, `properties: {name: {minLength: 2}}
   additionalProperties: false`, false},
		{"free map", `{type: object}`, `properties: {name: {type: string}}`, false},
		{"changed field type", `{type: object, properties: {name: {type: string}}}`, `properties: {name: {type: integer}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(`openapi: 3.2.0
info: {title: References, version: "1"}
paths: {}
components:
 schemas:
  Base: ` + tc.base + `
  Refined:
   $ref: '#/components/schemas/Base'
   ` + tc.siblings + "\n")
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			if tc.accepted {
				require.Empty(t, diagnostics)
			} else {
				requireDiagnosticCode(t, diagnostics, "schema-reference-siblings")
			}
			_, err = Render(document, Options{})
			require.Equal(t, tc.accepted, err == nil, "%v", err)
		})
	}
}
