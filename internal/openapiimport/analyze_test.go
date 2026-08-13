package openapiimport

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnalyzeJSONYAMLParity(t *testing.T) {
	yamlSource := readFixture(t, "supported.yaml")
	jsonSource := readFixture(t, "supported.json")

	yamlDocument, yamlDiagnostics, err := Analyze(yamlSource)
	require.NoError(t, err)
	require.Empty(t, yamlDiagnostics)
	jsonDocument, jsonDiagnostics, err := Analyze(jsonSource)
	require.NoError(t, err)
	require.Empty(t, jsonDiagnostics)

	require.Equal(t, yamlDocument, jsonDocument)
	require.Equal(t, "3.1.1", yamlDocument.OpenAPIVersion)
	require.Equal(t, "Pet API", yamlDocument.Title)
	require.Len(t, yamlDocument.Components.Schemas, 3)
	require.Len(t, yamlDocument.Operations, 2)
	require.Equal(t, "PetsCreate", yamlDocument.Operations[0].GoName)
	require.Equal(t, "#/components/schemas/NewPet", yamlDocument.Operations[0].RequestBody.Schema.Ref)
	require.Equal(t, "PetsGet", yamlDocument.Operations[1].GoName)
	require.Equal(t, "#/components/schemas/Pet", yamlDocument.Operations[1].Responses[0].Response.Schema.Ref)
}

func TestAnalyzeAssignsDeterministicCollisionNames(t *testing.T) {
	source := []byte(`openapi: 3.0.3
info: {title: Collisions, version: "1"}
paths:
  /b:
    get:
      operationId: get_pet
      responses: {"204": {description: done}}
  /a:
    get:
      operationId: get-pet
      responses: {"204": {description: done}}
components:
  schemas:
    pet_id: {type: string}
    pet-id: {type: integer}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Equal(t, []string{"GetPet", "GetPet2"}, []string{
		document.Operations[0].GoName,
		document.Operations[1].GoName,
	})
	require.Equal(t, []string{"PetID", "PetID2"}, []string{
		document.Components.Schemas[0].GoName,
		document.Components.Schemas[1].GoName,
	})
}

func TestAnalyzeAggregatesAndSortsUnsupportedFeatures(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Unsupported, version: "1"}
servers: [{url: https://example.com}]
security: [{token: []}]
paths:
  /pets:
    get:
      callbacks:
        update:
          '{$request.body#/callback}':
            post:
              responses: {"204": {description: done}}
      responses:
        default: {description: fallback}
components:
  securitySchemes:
    token: {type: http, scheme: bearer}
  schemas:
    Pet:
      oneOf:
        - {type: string}
        - {type: integer}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.NotNil(t, document)
	require.GreaterOrEqual(t, len(diagnostics), 5)
	for i := 1; i < len(diagnostics); i++ {
		require.LessOrEqual(t, diagnostics[i-1].Path, diagnostics[i].Path)
	}
	requireDiagnosticCode(t, diagnostics, "callbacks")
	requireDiagnosticCode(t, diagnostics, "default-response")
	requireDiagnosticCode(t, diagnostics, "schema-composition")
	requireDiagnosticCode(t, diagnostics, "security")
	requireDiagnosticCode(t, diagnostics, "security-schemes")
	requireDiagnosticCode(t, diagnostics, "servers")
}

func TestAnalyzeRejectsUnsupportedVersions(t *testing.T) {
	tests := map[string]string{
		"swagger": `swagger: "2.0"
info: {title: Old, version: "1"}
paths: {}`,
		"future": `openapi: 3.3.0
info: {title: Future, version: "1"}
paths: {}`,
	}
	for name, source := range tests {
		t.Run(name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(source))
			require.Error(t, err)
			require.ErrorIs(t, err, ErrUnsupportedVersion)
			require.Nil(t, document)
			require.Empty(t, diagnostics)
		})
	}
}

func TestAnalyzeRejectsExternalReferencesWithoutWriting(t *testing.T) {
	temp := t.TempDir()
	t.Chdir(temp)
	source := []byte(`openapi: 3.1.1
info: {title: External, version: "1"}
paths:
  /pets:
    get:
      responses:
        "200":
          description: pets
          content:
            application/json:
              schema: {$ref: ./models.yaml#/Pet}
components:
  schemas:
    Owner: {$ref: https://example.com/models.yaml#/Owner}
`)

	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Nil(t, document)
	require.Len(t, diagnostics, 2)
	requireDiagnosticCode(t, diagnostics, "external-reference")
	entries, err := os.ReadDir(temp)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestAnalyzeReportsMalformedInput(t *testing.T) {
	document, diagnostics, err := Analyze([]byte("openapi: ["))
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrUnsupportedVersion))
	require.Nil(t, document)
	require.Empty(t, diagnostics)
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return source
}

func requireDiagnosticCode(t *testing.T, diagnostics Diagnostics, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Errorf("diagnostic code %q not found in %#v", code, diagnostics)
}

func TestAnalyzeDoesNotRetainParserSpecificTypes(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.NotContains(t, reflect.TypeOf(*document).PkgPath(), "libopenapi")
}
