package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImportedBinaryRepresentationLocations checks byte-carrier admission at
// actual wire occurrences, including shared references reused in two contexts.
func TestImportedBinaryRepresentationLocations(t *testing.T) {
	for _, tc := range []struct {
		name      string
		operation string
		rejected  bool
	}{
		{"JSON root", `get:
      responses: {"200": {description: data, content: {application/json: {schema: {$ref: '#/components/schemas/Raw'}}}}}`, true},
		{"JSON field", `get:
      responses: {"200": {description: data, content: {application/json: {schema: {type: object, properties: {data: {$ref: '#/components/schemas/Raw'}}}}}}}`, true},
		{"JSON array", `get:
      responses: {"200": {description: data, content: {application/json: {schema: {type: array, items: {$ref: '#/components/schemas/Raw'}}}}}}`, true},
		{"JSON map", `get:
      responses: {"200": {description: data, content: {application/json: {schema: {type: object, additionalProperties: {$ref: '#/components/schemas/Raw'}}}}}}`, true},
		{"query", `get:
      parameters: [{name: data, in: query, schema: {$ref: '#/components/schemas/Raw'}}]
      responses: {"204": {description: ok}}`, true},
		{"response header", `get:
      responses: {"204": {description: ok, headers: {Data: {schema: {$ref: '#/components/schemas/Raw'}}}}}`, true},
		{"multipart part", `post:
      requestBody:
        content: {multipart/form-data: {schema: {type: object, properties: {data: {$ref: '#/components/schemas/Raw'}}}}}
      responses: {"204": {description: ok}}`, false},
		{"multipart nested array", `post:
      requestBody:
        content: {multipart/form-data: {schema: {type: object, properties: {data: {type: array, items: {$ref: '#/components/schemas/Raw'}}}}}}
      responses: {"204": {description: ok}}`, true},
		{"multipart nested object", `post:
      requestBody:
        content: {multipart/form-data: {schema: {type: object, properties: {data: {type: object, properties: {value: {$ref: '#/components/schemas/Raw'}}}}}}}
      responses: {"204": {description: ok}}`, true},
		{"JSON alias", `get:
      responses: {"200": {description: data, content: {application/json: {schema: {$ref: '#/components/schemas/Alias'}}}}}`, true},
		{"raw response", `get:
      responses: {"200": {description: data, content: {application/octet-stream: {schema: {$ref: '#/components/schemas/Raw'}}}}}`, false},
		{"mixed raw and JSON", `get:
      responses: {"200": {description: data, content: {application/octet-stream: {schema: {$ref: '#/components/schemas/Raw'}}}}}
    post:
      requestBody:
        content: {application/json: {schema: {$ref: '#/components/schemas/Raw'}}}
      responses: {"204": {description: ok}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := fmt.Sprintf(`openapi: 3.1.1
info: {title: Binary, version: "1"}
paths:
  /data:
    %s
components:
  schemas:
    Raw: {type: string, format: binary}
    Alias: {$ref: "#/components/schemas/Raw"}
`, tc.operation)
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			require.NotNil(t, document)
			design, renderErr := Render(document, Options{PackageName: "design"})
			if tc.rejected {
				found := false
				for _, diagnostic := range diagnostics {
					if diagnostic.Code == "schema-binary-location" {
						found = true
						assert.Contains(t, diagnostic.Path, "#/operations/")
					}
				}
				assert.True(t, found, "missing binary occurrence diagnostic: %v", diagnostics)
				assert.Error(t, renderErr, "direct Render must share location admission")
				assert.Empty(t, design)
				return
			}
			require.Empty(t, diagnostics)
			require.NoError(t, renderErr)
			require.Contains(t, string(design), `Meta("openapi:format", "binary")`)
		})
	}
}
