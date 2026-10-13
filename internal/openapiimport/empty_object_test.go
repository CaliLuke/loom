package openapiimport

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/testprocess"
)

func TestImportedEmptyObjectOpenFormsAgree(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1"} {
		t.Run(version, func(t *testing.T) {
			var explicit []byte
			for _, schema := range []string{
				`{"type":"object","additionalProperties":true}`,
				`{"type":"object","additionalProperties":{}}`,
				`{"type":"object"}`,
				`{"type":"object","properties":{}}`,
			} {
				t.Run(schema, func(t *testing.T) {
					document, diagnostics, err := Analyze(schemaAdmissionSource(version, schema, "property"))
					require.NoError(t, err)
					require.Empty(t, diagnostics)
					design, err := Render(document, Options{PackageName: "design"})
					require.NoError(t, err)
					if explicit == nil {
						explicit = design
					}
					require.Equal(t, string(explicit), string(design), "equivalent open object domains need the same concrete DSL representation")
				})
			}
		})
	}
}

func TestImportedEmptyObjectRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name    string
		schema  string
		value   any
		invalid any
	}{
		{"open field", `{"type":"object"}`, map[string]any{"free": []any{1, nil, "a"}}, "wrong"},
		{"closed field", `{"type":"object","additionalProperties":false}`, map[string]any{}, map[string]any{"extra": true}},
		{"open map value", `{"type":"object","additionalProperties":{"type":"object"}}`, map[string]any{"entry": map[string]any{"free": true}}, map[string]any{"entry": "wrong"}},
		{"closed map value", `{"type":"object","additionalProperties":{"type":"object","additionalProperties":false}}`, map[string]any{"entry": map[string]any{}}, map[string]any{"entry": map[string]any{"extra": true}}},
		{"record map value", `{"type":"object","additionalProperties":{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}}`, map[string]any{"entry": map[string]any{"name": "a"}}, map[string]any{"entry": map[string]any{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := schemaAdmissionSource("3.1.1", `{"type":"object","properties":{"value":`+tc.schema+`}}`, "component")
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			moduleDir := requireRenderedDesignGenerates(t, design)
			requireTypeNullAcceptance(t, source, moduleDir, "Value", []struct {
				value any
				valid bool
			}{
				{map[string]any{}, true},
				{map[string]any{"value": tc.value}, true},
				{map[string]any{"value": tc.invalid}, false},
				{map[string]any{"value": "wrong"}, false},
				{map[string]any{"value": nil}, false},
			})
		})
	}
}

func TestImportedEmptyObjectNamedControl(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Object control, version: "1"}
paths:
  /record:
    get:
      operationId: read
      responses:
        "200":
          description: record
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Record'}
components:
  schemas:
    Open: {type: object, additionalProperties: true}
    Closed: {type: object, additionalProperties: false}
    Record:
      type: object
      properties:
        closed: {$ref: '#/components/schemas/Closed'}
        open: {$ref: '#/components/schemas/Open'}
        values: {type: object, additionalProperties: {$ref: '#/components/schemas/Closed'}}
        items: {type: array, items: {$ref: '#/components/schemas/Closed'}}
        normal: {type: object, properties: {name: {type: string}}}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	moduleDir := requireRenderedDesignGenerates(t, design)
	requireTypeNullAcceptance(t, source, moduleDir, "Record", []struct {
		value any
		valid bool
	}{
		{map[string]any{}, true},
		{map[string]any{"closed": map[string]any{}, "open": map[string]any{"extra": nil}, "values": map[string]any{"entry": map[string]any{}}, "items": []any{map[string]any{}}, "normal": map[string]any{"name": "a"}}, true},
		{map[string]any{"closed": map[string]any{"extra": true}}, false},
		{map[string]any{"values": map[string]any{"entry": map[string]any{"extra": true}}}, false},
		{map[string]any{"items": []any{map[string]any{"extra": true}}}, false},
		{map[string]any{"closed": "wrong"}, false},
	})
}

func TestImportedEmptyObjectAdmissionControls(t *testing.T) {
	for _, version := range []string{"3.0.3", "3.1.1"} {
		t.Run(version, func(t *testing.T) {
			for _, schema := range []string{
				`{"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":true}`,
				`{"type":"object","required":["name"],"additionalProperties":true}`,
				`{"type":"object","properties":{"name":{"type":"string"}},"additionalProperties":{"type":"string"}}`,
			} {
				document, diagnostics, err := Analyze(schemaAdmissionSource(version, schema, "property"))
				require.NoError(t, err)
				fatal, _ := diagnostics.Classify(false)
				require.NotEmpty(t, fatal, "mixed object must retain admission: %s", schema)
				_, err = Render(document, Options{PackageName: "design"})
				require.Error(t, err)
			}
			document, diagnostics, err := Analyze(schemaAdmissionSource(version, `{"type":"object","additionalProperties":false}`, "items"))
			require.NoError(t, err)
			require.Len(t, diagnostics, 1)
			require.Equal(t, "schema-inline-array-item-promoted", diagnostics[0].Code)
			fatal, warnings := diagnostics.Classify(true)
			require.Empty(t, fatal)
			require.Len(t, warnings, 1)
			_, err = Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
		})
	}
}

func TestImportedEmptyObjectDirectRenderAdmission(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mapValue bool
		record   bool
		valid    bool
	}{
		{"empty field", false, false, false},
		{"empty map value", true, false, false},
		{"record map value", true, true, false},
		{"record field control", false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := `{"type":"object","properties":{"value":{"type":"string"}}}`
			if tc.mapValue {
				schema = `{"type":"object","additionalProperties":{"type":"string"}}`
			}
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", schema, "component"))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			closed := false
			object := &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Allowed: &closed}}
			if tc.record {
				object.Properties = []NamedProperty{{Name: "name", Schema: &Schema{Type: "string"}}}
			}
			root := document.Components.Schemas[0].Schema
			path := "#/components/schemas/Value/properties/value"
			if tc.mapValue {
				root.AdditionalProperties.Schema = object
				path = "#/components/schemas/Value/additionalProperties"
			} else {
				root.Properties[0].Schema = object
			}
			_, err = Render(document, Options{PackageName: "design"})
			if tc.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), "cannot preserve the input contract")
			require.Contains(t, err.Error(), path)
		})
	}
}

func TestImportedEmptyObjectNullableCollision(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Nullable objects, version: "1"}
paths:
  /record:
    get:
      operationId: read
      responses:
        "200":
          description: record
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Record'}
components:
  schemas:
    RecordClosed: {type: string}
    Record:
      type: object
      properties:
        closed: {type: [object, 'null'], additionalProperties: false}
        open: {type: [object, 'null']}
        values: {type: object, additionalProperties: {type: [object, 'null'], additionalProperties: false}}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	var closedRef string
	for _, named := range document.Components.Schemas {
		if named.Name == "Record" {
			closedRef = named.Schema.Properties[0].Schema.Ref
		}
	}
	require.Equal(t, "#/components/schemas/RecordClosed2", closedRef)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	moduleDir := requireRenderedDesignGenerates(t, design)
	requireTypeNullAcceptance(t, source, moduleDir, "Record", []struct {
		value any
		valid bool
	}{
		{map[string]any{}, true},
		{map[string]any{"closed": nil, "open": nil, "values": map[string]any{"entry": nil}}, true},
		{map[string]any{"closed": map[string]any{}, "open": map[string]any{"extra": true}, "values": map[string]any{"entry": map[string]any{}}}, true},
		{map[string]any{"closed": map[string]any{"extra": true}}, false},
		{map[string]any{"closed": "wrong"}, false},
		{map[string]any{"values": map[string]any{"entry": "wrong"}}, false},
	})
}

func TestImportedEmptyObjectRootBindings(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Object roots, version: "1"}
paths:
  /value:
    post:
      operationId: value
      requestBody:
        required: true
        content:
          application/json:
            schema: {type: object, additionalProperties: false}
      responses:
        "200":
          description: value
          headers:
            X-Label: {schema: {type: string}}
          content:
            application/json:
              schema: {type: object, additionalProperties: false}
  /plain:
    get:
      operationId: plain
      responses:
        "200":
          description: plain
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Closed'}
  /inherited:
    get:
      operationId: inherited
      responses:
        "200":
          description: inherited
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Inherited'}
  /inline:
    get:
      operationId: inline
      responses:
        "200":
          description: inline
          content:
            application/json:
              schema: {type: object, additionalProperties: false}
components:
  schemas:
    Closed: {type: object, additionalProperties: false}
    Inherited:
      allOf:
        - {$ref: '#/components/schemas/Closed'}
        - {type: object}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Len(t, diagnostics, 1)
	require.Equal(t, "schema-allof-flattened", diagnostics[0].Code)
	var operation *Operation
	for i := range document.Operations {
		if document.Operations[i].Path == "/value" {
			operation = &document.Operations[i]
		}
	}
	require.NotNil(t, operation)
	for _, schema := range []*Schema{operation.RequestBody.Schema, operation.Responses[0].Response.Schema} {
		require.NotEmpty(t, schema.Ref, "root rendered as a field needs a concrete object")
	}
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	moduleDir := requireRenderedDesignGenerates(t, design)
	contract := readGeneratedOpenAPIContract(t, moduleDir)
	wireOperation := operationFromImportedSpec(t, contract, "/value", "post")
	request := wireOperation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	responseEnvelope := wireOperation["responses"].(map[string]any)["200"].(map[string]any)
	require.Contains(t, responseEnvelope, "content", "closed empty object response body must remain present")
	response := responseEnvelope["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	for _, schema := range []map[string]any{request, response} {
		if ref, ok := schema["$ref"].(string); ok {
			schema = referencedSchema(t, contract, ref)
		}
		require.Equal(t, "object", schema["type"])
		require.Equal(t, false, schema["additionalProperties"])
	}
	for _, path := range []string{"/plain", "/inline", "/inherited"} {
		response := operationFromImportedSpec(t, contract, path, "get")["responses"].(map[string]any)["200"].(map[string]any)
		require.Contains(t, response, "content", "source-declared empty content must remain present: %s", path)
		schema := response["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
		if ref, ok := schema["$ref"].(string); ok {
			schema = referencedSchema(t, contract, ref)
		}
		require.Equal(t, "object", schema["type"])
		if path != "/inherited" {
			require.Equal(t, false, schema["additionalProperties"])
		}
	}
	harness, err := os.ReadFile("testdata/empty_object_wire.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "wire_test.go"), harness, 0o600))
	command := testprocess.CommandContext(t.Context(), "go", "test", "-mod=mod", "-race", "-count=1", ".")
	command.Dir = moduleDir
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	operation.Parameters = []Parameter{{Name: "filter", In: "query", Schema: &Schema{Type: "string"}}}
	for _, tc := range []struct {
		name   string
		schema **Schema
		path   string
	}{
		{"parameter", &operation.Parameters[0].Schema, "/parameters/0/schema"},
		{"request", &operation.RequestBody.Schema, "/requestBody/content/application~1json/schema"},
		{"response", &operation.Responses[0].Response.Schema, "/responses/200/content/schema"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := *tc.schema
			closed := false
			*tc.schema = &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Allowed: &closed}}
			_, err := Render(document, Options{PackageName: "design"})
			*tc.schema = original
			require.Error(t, err)
			require.Contains(t, err.Error(), "cannot preserve the input contract")
			require.Contains(t, err.Error(), tc.path)
		})
	}
}

func TestImportedEmptyObjectUnionAdmission(t *testing.T) {
	for _, open := range []string{
		`{"type":"object"}`,
		`{"type":"object","additionalProperties":true}`,
		`{"type":"object","additionalProperties":{}}`,
	} {
		t.Run(open, func(t *testing.T) {
			schema := `{"oneOf":[` + open + `,{"type":"object","additionalProperties":false}]}`
			document, diagnostics, err := Analyze(schemaAdmissionSource("3.1.1", schema, "component"))
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "schema-oneof-branch")
			_, err = Render(document, Options{PackageName: "design"})
			require.Error(t, err)
		})
	}
}

func TestImportedEmptyObjectFormSpellings(t *testing.T) {
	var explicit []byte
	for _, schema := range []string{
		`{"type":"object","additionalProperties":true}`,
		`{"type":"object","additionalProperties":{}}`,
		`{"type":"object"}`,
	} {
		t.Run(schema, func(t *testing.T) {
			source := strings.ReplaceAll(string(schemaAdmissionSource("3.1.1", schema, "request")), "application/json", "application/x-www-form-urlencoded")
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Contains(t, string(design), "SkipRequestBodyEncodeDecode()")
			if explicit == nil {
				explicit = design
			}
			require.Equal(t, string(explicit), string(design))
		})
	}
}

func TestImportedObjectResponseReferenceIdentity(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: References, version: "1"}
paths:
  /record:
    get:
      responses:
        "200":
          description: record
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Alias'}
components:
  schemas:
    Alias: {$ref: '#/components/schemas/Empty~1Record'}
    Empty/Record: {type: object, additionalProperties: false}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), "Body(ImportedAlias)")
}
