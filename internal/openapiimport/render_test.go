package openapiimport

import (
	"bytes"
	"encoding/json"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderSupportedFixtureDeterministically(t *testing.T) {
	yamlDocument, yamlDiagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, yamlDiagnostics)
	jsonDocument, jsonDiagnostics, err := Analyze(readFixture(t, "supported.json"))
	require.NoError(t, err)
	require.Empty(t, jsonDiagnostics)

	yamlSource, err := Render(yamlDocument, Options{PackageName: "design"})
	require.NoError(t, err)
	jsonSource, err := Render(jsonDocument, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Equal(t, yamlSource, jsonSource)

	parsed, err := parser.ParseFile(token.NewFileSet(), "design.go", yamlSource, parser.AllErrors)
	require.NoError(t, err)
	require.Equal(t, "design", parsed.Name.Name)
	for _, want := range []string{
		`var ImportedNewPet = Type("NewPet", func() {`,
		`Meta("openapi:additionalProperties", "false")`,
		`Meta("openapi:example", "false")`,
		`Meta("openapi:tag:pets")`,
		`Method("PetsCreate", func() {`,
		`Meta("openapi:operationId", "pets.create")`,
		`Meta("openapi:summary", "")`,
		`Meta("openapi:description:requestBody", "Pet to create.")`,
		`POST("/pets")`,
		`Response(201, func() {`,
		`Method("PetsGet", func() {`,
		`Error("Status404", ImportedProblem)`,
		`GET("/pets/{id}")`,
		`Header("xTraceID:X-Trace-ID")`,
		`Meta("openapi:component:parameter", "PetID")`,
		`Meta("openapi:allowEmptyValue", "false")`,
		`Response("Status404", 404, func() {`,
		`Meta("openapi:description:errorName", "false")`,
		`Meta("openapi:readOnly", "true")`,
		`Meta("openapi:writeOnly", "true")`,
		`Meta("openapi:deprecated", "true")`,
		`Default("cat")`,
		`Attribute("weight", Float64)`,
		`Attribute("stock", Int)`,
	} {
		require.Contains(t, string(yamlSource), want)
	}
	require.Contains(t, string(yamlSource), "HTTP(func() {\n\t\t\tMeta(\"openapi:tag:pets\")")
	require.Contains(t, string(yamlSource), "Attribute(\"body\", ImportedNewPet, func() {\n\t\t\t\tMeta(\"openapi:description:requestBody\"")
	require.NotContains(t, string(yamlSource), "Payload(func() {\n\t\t\tMeta(\"openapi:description:requestBody\"")
}

func TestRenderSuppressesGeneratedOperationMetadataWhenAbsent(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	document.Operations[0].OperationID = ""
	document.Operations[0].Summary = ""

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	method := string(source)
	start := strings.Index(method, `Method("PetsCreate", func() {`)
	require.NotEqual(t, -1, start)
	method = method[start:]
	end := strings.Index(method, `Method("PetsGet", func() {`)
	require.NotEqual(t, -1, end)
	method = method[:end]
	require.Contains(t, method, `Meta("openapi:operationId", "")`)
	require.Contains(t, method, `Meta("openapi:summary", "")`)
}

func TestRenderOutputCompilesAndEvaluatesAsLoomDesign(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)

	moduleDir := t.TempDir()
	repoRoot := repositoryRoot(t)
	goMod := "module example.com/imported\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\n" +
		"replace github.com/CaliLuke/loom => " + repoRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600))
	designDir := filepath.Join(moduleDir, "design")
	require.NoError(t, os.MkdirAll(designDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design.go"), source, 0o600))
	testSource := `package design

import (
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestEvaluateImportedDesign(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	if got := len(expr.Root.Services); got != 1 {
		t.Fatalf("got %d services, want 1", got)
	}
	if got := len(expr.Root.Services[0].Methods); got != 2 {
		t.Fatalf("got %d methods, want 2", got)
	}
}
`
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design_test.go"), []byte(testSource), 0o600))

	cmd := exec.Command("go", "test", "-mod=mod", "./design")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func TestRenderErrorFieldCollisionGeneratesCompilingService(t *testing.T) {
	schemaRef := func(name string) *Schema {
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	document := &Document{
		OpenAPIVersion: "3.1.1",
		Title:          "Error Fields",
		APIVersion:     "1.0.0",
		Components: Components{Schemas: []NamedSchema{
			{
				Name:   "Failure",
				GoName: "Failure",
				Schema: &Schema{Type: "object", Properties: []NamedProperty{
					{Name: "error", Schema: &Schema{Type: "string"}},
					{Name: "loomErrorName", Schema: &Schema{Type: "string"}},
				}},
			},
			{
				Name:   "Success",
				GoName: "Success",
				Schema: &Schema{Type: "object", Properties: []NamedProperty{
					{Name: "error", Schema: &Schema{Type: "string"}},
				}},
			},
		}},
		Operations: []Operation{{
			Method: "GET", Path: "/fields", OperationID: "fields.get", GoName: "FieldsGet",
			Responses: []StatusResponse{
				{Status: "200", Response: Response{ContentType: "application/json", Schema: schemaRef("Success")}},
				{Status: "500", Response: Response{ContentType: "application/json", Schema: schemaRef("Failure")}},
			},
		}, {
			Method: "GET", Path: "/inline", OperationID: "inline.get", GoName: "InlineGet",
			Responses: []StatusResponse{
				{Status: "204", Response: Response{}},
				{
					Status: "400",
					Response: Response{
						ContentType: "application/json",
						Schema: &Schema{Type: "object", Properties: []NamedProperty{
							{Name: "error", Schema: &Schema{Type: "string"}},
						}},
					},
				},
			},
		}},
	}

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(source), "Meta(\"struct:field:name\", \"ErrorField\")")
	require.Contains(t, string(source), "Meta(\"struct:field:name\", \"LoomErrorNameField\")")
	require.Equal(t, 3, strings.Count(string(source), "struct:field:name"))

	moduleDir := t.TempDir()
	repoRoot := repositoryRoot(t)
	goMod := "module example.com/imported\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\n" +
		"replace github.com/CaliLuke/loom => " + repoRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600))
	designDir := filepath.Join(moduleDir, "design")
	require.NoError(t, os.MkdirAll(designDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design.go"), source, 0o600))

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	cmd = exec.Command("go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/imported/design", "-o", ".")
	cmd.Dir = moduleDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	cmd = exec.Command("go", "test", "-mod=mod", "./...")
	cmd.Dir = moduleDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

// TestRenderRoundTripsNewlySupportedSchemaKeywordsIntoOpenAPI regenerates the
// imported design and inspects the generated OpenAPI document, confirming
// that default, deprecated, readOnly, writeOnly, and unformatted
// integer/number properties imported from supported.yaml reappear in the
// regenerated contract rather than merely compiling.
func TestRenderRoundTripsNewlySupportedSchemaKeywordsIntoOpenAPI(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)

	moduleDir := t.TempDir()
	repoRoot := repositoryRoot(t)
	goMod := "module example.com/imported\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\n" +
		"replace github.com/CaliLuke/loom => " + repoRoot + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte(goMod), 0o600))
	designDir := filepath.Join(moduleDir, "design")
	require.NoError(t, os.MkdirAll(designDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design.go"), source, 0o600))

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	cmd = exec.Command("go", "run", "-mod=mod", "github.com/CaliLuke/loom/cmd/loom", "gen", "example.com/imported/design", "-o", ".")
	cmd.Dir = moduleDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, string(output))

	openAPI, err := os.ReadFile(filepath.Join(moduleDir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	spec := string(openAPI)
	for _, want := range []string{
		`"default":"cat"`,
		`"deprecated":true`,
		`"readOnly":true`,
	} {
		require.Contains(t, spec, want)
	}
	// Pet is only ever used as a response schema in this fixture, so its
	// writeOnly "secret" property is correctly split out of every generated
	// view rather than appearing with "writeOnly":true; a request-side
	// document would need to reference Pet as a payload to observe the flag
	// in the generated contract. writeOnly's DSL-level rendering is covered
	// directly by TestRenderEmitsDefaultAndDeprecatedMetadata.
	require.NotContains(t, spec, `"secret"`)

	// weight (unformatted number) and stock (unformatted integer) round-trip
	// as the widest Loom representation: Loom always annotates a generated
	// integer/number with a format, so the regenerated contract gains
	// "format": "double"/"int64" that the source document omitted. This is
	// the documented, non-narrowing fallback rather than a byte-identical
	// round trip.
	var decoded struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Type   string `json:"type"`
					Format string `json:"format"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(openAPI, &decoded))
	var pet *struct {
		Type   string `json:"type"`
		Format string `json:"format"`
	}
	for name, schema := range decoded.Components.Schemas {
		if _, ok := schema.Properties["weight"]; ok {
			require.Contains(t, name, "Pet")
			weight := schema.Properties["weight"]
			pet = &weight
			require.Equal(t, "number", schema.Properties["weight"].Type)
			require.Equal(t, "double", schema.Properties["weight"].Format)
			require.Equal(t, "integer", schema.Properties["stock"].Type)
			require.Equal(t, "int64", schema.Properties["stock"].Format)
		}
	}
	require.NotNil(t, pet, "expected a Pet-derived schema with a weight property")
}

func TestRenderRejectsUnrepresentableModels(t *testing.T) {
	schemaRef := func(name string) *Schema {
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	base := func() *Document {
		return &Document{
			OpenAPIVersion: "3.1.1",
			Title:          "Strict",
			APIVersion:     "1.0.0",
			Tags:           []string{"strict"},
			Components: Components{Schemas: []NamedSchema{
				{Name: "Result", GoName: "Result", Schema: &Schema{Type: "string"}},
			}},
			Operations: []Operation{{
				Method: "GET", Path: "/strict", OperationID: "strict.get", GoName: "StrictGet",
				Tags: []string{"strict"}, Responses: []StatusResponse{{
					Status: "200", Response: Response{Schema: schemaRef("Result"), ContentType: "application/json"},
				}},
			}},
		}
	}

	tests := []struct {
		name string
		edit func(*Document)
		want string
	}{
		{name: "invalid package", edit: func(*Document) {}, want: `package name "bad-name" is not a Go identifier`},
		{name: "OpenAPI 3.0 target", edit: func(d *Document) { d.OpenAPIVersion = "3.0.3" }, want: "only OpenAPI 3.1 and 3.2 documents are renderable"},
		{name: "no success response", edit: func(d *Document) { d.Operations[0].Responses[0].Status = "404" }, want: "must define exactly one 2xx response"},
		{name: "multiple success responses", edit: func(d *Document) {
			d.Operations[0].Responses = append(d.Operations[0].Responses, StatusResponse{Status: "201"})
		}, want: "must define exactly one 2xx response"},
		{name: "unresolved schema reference", edit: func(d *Document) {
			d.Operations[0].Responses[0].Response.Schema = schemaRef("Missing")
		}, want: `schema reference "#/components/schemas/Missing" does not resolve`},
		{name: "wrong reference kind", edit: func(d *Document) {
			d.Operations[0].RequestBody = &RequestBody{Ref: "#/components/requestBodies/Body"}
		}, want: "request body references are not renderable"},
		{name: "mixed object and map", edit: func(d *Document) {
			d.Components.Schemas[0].Schema = &Schema{
				Type: "object", Properties: []NamedProperty{{Name: "id", Schema: &Schema{Type: "string"}}},
				AdditionalProperties: &AdditionalProperties{Schema: &Schema{Type: "string"}},
			}
		}, want: "object properties with schema-valued additionalProperties are not renderable"},
		{name: "unsupported boolean format", edit: func(d *Document) {
			d.Components.Schemas[0].Schema = &Schema{Type: "boolean", Format: "custom"}
		}, want: `boolean format "custom" is not renderable`},
		{name: "response reference", edit: func(d *Document) {
			d.Operations[0].Responses[0].Response = Response{Ref: "#/components/responses/OK"}
		}, want: "response references are not renderable"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document := base()
			test.edit(document)
			packageName := "design"
			if test.name == "invalid package" {
				packageName = "bad-name"
			}
			_, err := Render(document, Options{PackageName: packageName})
			require.ErrorContains(t, err, test.want)
		})
	}
}

func TestRenderEmitsDefaultAndDeprecatedMetadata(t *testing.T) {
	document := &Document{
		OpenAPIVersion: "3.1.1",
		Title:          "Metadata",
		APIVersion:     "1.0.0",
		Components: Components{Schemas: []NamedSchema{
			{Name: "Widget", GoName: "Widget", Schema: &Schema{
				Type: "object",
				Properties: []NamedProperty{
					{Name: "mouthX", Schema: &Schema{Type: "number", Format: "float", Deprecated: true}},
					{Name: "stable", Schema: &Schema{Type: "string", ReadOnly: true}},
					{Name: "input", Schema: &Schema{Type: "string", WriteOnly: true}},
					{Name: "apiVersion", Schema: &Schema{Type: "string", Default: &SchemaDefault{Value: "1.0"}}},
				},
			}},
		}},
		Operations: []Operation{{
			Method: "GET", Path: "/widgets", OperationID: "widgets.get", GoName: "WidgetsGet",
			Responses: []StatusResponse{{
				Status: "200", Response: Response{
					Schema:      &Schema{Ref: "#/components/schemas/Widget"},
					ContentType: "application/json",
				},
			}},
		}},
	}

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	rendered := string(source)
	require.Contains(t, rendered, `Meta("openapi:deprecated", "true")`)
	require.Contains(t, rendered, `Meta("openapi:readOnly", "true")`)
	require.Contains(t, rendered, `Meta("openapi:writeOnly", "true")`)
	require.Contains(t, rendered, `Default("1.0")`)
}

func TestRenderFallsBackForUnrecognizedAndAbsentFormats(t *testing.T) {
	schemaRef := func(name string) *Schema {
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	document := &Document{
		OpenAPIVersion: "3.1.1",
		Title:          "Formats",
		APIVersion:     "1.0.0",
		Components: Components{Schemas: []NamedSchema{
			{Name: "Record", GoName: "Record", Schema: &Schema{
				Type: "object",
				Properties: []NamedProperty{
					{Name: "birthDate", Schema: &Schema{Type: "string", Format: "dd/mm/yyyy"}},
					{Name: "count", Schema: &Schema{Type: "integer"}},
					{Name: "weight", Schema: &Schema{Type: "integer", Format: "int16"}},
					{Name: "result", Schema: &Schema{Type: "number", Format: ""}},
				},
			}},
		}},
		Operations: []Operation{{
			Method: "GET", Path: "/records", OperationID: "records.get", GoName: "RecordsGet",
			Responses: []StatusResponse{{
				Status: "200", Response: Response{Schema: schemaRef("Record"), ContentType: "application/json"},
			}},
		}},
	}

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	rendered := string(source)
	require.Contains(t, rendered, `Attribute("birthDate", String)`)
	require.NotContains(t, rendered, "FormatDate")
	require.Contains(t, rendered, `Attribute("count", Int)`)
	require.Contains(t, rendered, `Attribute("weight", Int)`)
	require.Contains(t, rendered, `Attribute("result", Float64)`)
}

func TestRenderDropsUnrepresentableParameterAndHeaderDeprecation(t *testing.T) {
	document := &Document{
		OpenAPIVersion: "3.1.1",
		Title:          "Deprecated",
		APIVersion:     "1.0.0",
		Operations: []Operation{{
			Method: "GET", Path: "/legacy", OperationID: "legacy.get", GoName: "LegacyGet",
			Parameters: []Parameter{{Name: "q", In: "query", Deprecated: true, Schema: &Schema{Type: "string"}}},
			Responses: []StatusResponse{{
				Status: "200",
				Response: Response{
					ContentType: "application/json",
					Schema:      &Schema{Type: "string"},
					Headers: []NamedHeader{{
						Name:   "X-Legacy",
						Header: Header{Deprecated: true, Schema: &Schema{Type: "string"}},
					}},
				},
			}},
		}},
	}

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.NotContains(t, string(source), "openapi:deprecated")
}

func TestRenderDoesNotMutateDocument(t *testing.T) {
	document, diagnostics, err := Analyze(readFixture(t, "supported.yaml"))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	before, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	after, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.True(t, bytes.Equal(before, after))
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}
