package openapiimport

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
		`Method("PetsCreate", func() {`,
		`Meta("openapi:operationId", "pets.create")`,
		`POST("/pets")`,
		`Response(201, func() {`,
		`Method("PetsGet", func() {`,
		`Error("Status404", ImportedProblem)`,
		`GET("/pets/{id}")`,
		`Header("xTraceID:X-Trace-ID")`,
		`Response("Status404", 404, func() {`,
	} {
		require.Contains(t, string(yamlSource), want)
	}
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
		{name: "OpenAPI 3.0 target", edit: func(d *Document) { d.OpenAPIVersion = "3.0.3" }, want: "cannot target OpenAPI 3.0"},
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
		{name: "unsupported format", edit: func(d *Document) {
			d.Components.Schemas[0].Schema = &Schema{Type: "string", Format: "duration"}
		}, want: `string format "duration" is not renderable`},
		{name: "response reference", edit: func(d *Document) {
			d.Operations[0].Responses[0].Response = Response{Ref: "#/components/responses/OK"}
		}, want: "response references are not renderable"},
		{name: "parameter deprecation", edit: func(d *Document) {
			d.Operations[0].Parameters = []Parameter{{Name: "q", In: "query", Deprecated: true, Schema: &Schema{Type: "string"}}}
		}, want: "deprecated parameters are not renderable"},
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
