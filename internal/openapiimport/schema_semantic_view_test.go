package openapiimport

import (
	"testing"

	"github.com/CaliLuke/libopenapi"
	"github.com/pb33f/go-yaml"

	"github.com/stretchr/testify/require"
)

func TestSchemaSemanticViewRepresentations(t *testing.T) {
	const prefix = `openapi: 3.1.1
info: {title: Semantic views, version: '1'}
paths: {}
x-base: &base {type: integer, minimum: 5, maximum: 10}
x-second: &second {type: integer, minimum: 99, maximum: 100}
components:
  schemas:
    Bound: `
	var explicit []byte
	for _, tc := range []struct{ name, schema string }{
		{"explicit", "{type: integer, minimum: 5, maximum: 10}"},
		{"alias", "*base"},
		{"merge", "{<<: *base}"},
		{"first source", "{<<: [*base, *second]}"},
		{"local override", "{<<: *second, minimum: 5, maximum: 10}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(prefix + tc.schema + "\n"))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Len(t, document.Components.Schemas, 1)
			schema := document.Components.Schemas[0].Schema
			require.Equal(t, "5", schema.Minimum.String())
			require.Equal(t, "10", schema.Maximum.String())
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			if explicit == nil {
				explicit = design
			}
			require.Equal(t, string(explicit), string(design))
		})
	}
}

func TestSchemaSemanticViewNullableAlternatives(t *testing.T) {
	const prefix = `openapi: 3.1.1
info: {title: Semantic views, version: '1'}
paths: {}
x-null: &null {type: 'null'}
components:
  schemas:
    Nullable:
      anyOf:
        - {type: string}
        - `
	var explicit []byte
	for _, tc := range []struct{ name, schema string }{
		{"explicit", "{type: 'null'}"},
		{"alias", "*null"},
		{"merge", "{<<: *null}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(prefix + tc.schema + "\n"))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			if explicit == nil {
				explicit = design
			}
			require.Equal(t, string(explicit), string(design))
		})
	}
}

func TestSchemaSemanticViewInheritedRejections(t *testing.T) {
	for _, tc := range []struct{ name, base, body, code string }{
		{"unknown keyword", "{type: string, mystery: true}", "{<<: *base}", "schema-keyword-unknown"},
		{"aliased unknown keyword", "{type: string, mystery: true}", "*base", "schema-keyword-unknown"},
		{"annotated null", "{type: 'null', description: annotated}", "{anyOf: [{type: string}, {<<: *base}]}", "schema-composition"},
		{"constrained null", "{type: 'null', enum: [null]}", "{anyOf: [{type: string}, {<<: *base}]}", "schema-composition"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "openapi: 3.1.1\ninfo: {title: Rejected, version: '1'}\npaths: {}\nx-base: &base " + tc.base + "\ncomponents: {schemas: {Value: " + tc.body + "}}\n"
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)
			requireDiagnosticCode(t, diagnostics, tc.code)
			fatal, _ := diagnostics.Classify(true)
			requireDiagnosticCode(t, fatal, tc.code)
			if tc.code == "schema-keyword-unknown" {
				require.Contains(t, diagnostics.Error(), "/mystery")
			} else {
				_, err = Render(document, Options{PackageName: "design"})
				require.Error(t, err)
			}
		})
	}
}

func TestSchemaSemanticViewBooleanAliases(t *testing.T) {
	for _, boolean := range []string{"true", "false"} {
		t.Run(boolean, func(t *testing.T) {
			prefix := "openapi: 3.1.1\ninfo: {title: Boolean, version: '1'}\npaths: {}\nx-boolean: &boolean " + boolean + "\ncomponents:\n  schemas:\n    Record:\n      type: object\n      properties:\n        value: "
			explicit, explicitDiagnostics, err := Analyze([]byte(prefix + boolean + "\n"))
			require.NoError(t, err)
			aliased, aliasedDiagnostics, err := Analyze([]byte(prefix + "*boolean\n"))
			require.NoError(t, err)
			require.Equal(t, explicitDiagnostics, aliasedDiagnostics)
			require.Equal(t, explicit.Components.Schemas[0].Schema, aliased.Components.Schemas[0].Schema)
		})
	}
}

func TestSchemaSemanticViewCycleRejection(t *testing.T) {
	for _, base := range []string{"&base {<<: *base}", "&base [*base]"} {
		source := "openapi: 3.1.1\ninfo: {title: Cyclic, version: '1'}\npaths: {}\nx-base: " + base + "\ncomponents: {schemas: {Value: {<<: *base}}}\n"
		_, _, err := Analyze([]byte(source))
		require.Error(t, err)
	}
}

func TestSchemaSemanticViewPreservesAuthoredSource(t *testing.T) {
	const source = `openapi: 3.1.1
info: {title: Source, version: '1'}
paths: {}
x-base: &base {type: 'null'} # source comment
components:
  schemas:
    Value: {<<: *base} # occurrence comment
`
	document, err := libopenapi.NewDocument([]byte(source))
	require.NoError(t, err)
	root := document.GetSpecInfo().RootNode
	before, err := yaml.Marshal(root)
	require.NoError(t, err)
	model, err := document.BuildV3Model()
	require.NoError(t, err)
	proxy, found := model.Model.Components.Schemas.Get("Value")
	require.True(t, found)
	authored := proxy.GetValueNode()
	require.NotNil(t, authored)
	line, column, comment := authored.Line, authored.Column, authored.LineComment
	a := analyzer{version: "3.1.1"}
	a.schemaUnknownKeywordDiagnostics(proxy, "#/components/schemas/Value")
	isNullOnlySchema(proxy)
	a.schema(proxy, "#/components/schemas/Value")
	require.Same(t, authored, proxy.GetValueNode())
	require.Equal(t, line, authored.Line)
	require.Equal(t, column, authored.Column)
	require.Equal(t, comment, authored.LineComment)
	after, err := yaml.Marshal(root)
	require.NoError(t, err)
	require.Equal(t, before, after)
}
