package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscriminatorRejectedSourceMatrix(t *testing.T) {
	for _, tc := range []struct{ name, hint, extra string }{
		{"missing property", "{mapping: {cat: Cat}}", ""},
		{"empty property", "{propertyName: ''}", ""},
		{"empty fallback", "{propertyName: kind, defaultMapping: ''}", ""},
		{"missing fallback", "{propertyName: kind, defaultMapping: Missing}", ""},
		{"relative mapping without base", "{propertyName: kind, mapping: {cat: './cats.yaml#/Cat'}}", ""},
		{"relative fallback without base", "{propertyName: kind, defaultMapping: './cats.yaml#/Cat'}", ""},
		{"assertion sibling", "{propertyName: kind}", "      minProperties: 1"},
		{"unknown sibling", "{propertyName: kind}", "      unrecognized: true"},
		{"closure true", "{propertyName: kind}", "      unevaluatedProperties: true"},
		{"nonredundant required", "{propertyName: kind}", "      required: [missing]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, diagnostics, err := Analyze(discriminatorDocument("3.2.0", tc.hint, `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`, tc.extra))
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)
			fatal, _ := diagnostics.Classify(true)
			require.NotEmpty(t, fatal, "discriminator contracts cannot become lossy hints")
		})
	}
}

func TestDiscriminatorStyledNestedFieldsRemainRejected(t *testing.T) {
	for _, content := range []string{"", ", contentType: application/json"} {
		source := string(readFixture(t, "discriminator/media.yaml"))
		source = strings.Replace(source, "event: {contentType: application/json}", "event: {style: deepObject, explode: true"+content+"}", 1)
		_, diagnostics, err := Analyze([]byte(source))
		require.NoError(t, err)
		requireDiagnosticCode(t, diagnostics, "media-encoding")
	}
}

func TestDiscriminatorArbitraryMappingKeys(t *testing.T) {
	hint := `{propertyName: kind, mapping: {'': Cat, 'cat=old': Cat, ' cat ': Cat, dog: Dog}, x-audit: {z: 2, a: 1}}`
	document, diagnostics, err := Analyze(discriminatorDocument("3.2.0", hint, `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`, ""))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	for _, fragment := range []string{`Meta("openapi:discriminator:mapping", "")`, `Meta("openapi:discriminator:mapping", "cat=old")`, `Meta("openapi:discriminator:mapping", " cat ")`, `Meta("openapi:discriminator:extension:x-audit", "{\"a\":1,\"z\":2}")`} {
		require.Contains(t, string(design), fragment)
	}
}

func TestDiscriminatorParentChildrenAreDirect(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Direct parents, version: '1'}
paths: {}
components:
  schemas:
    Parent:
      type: object
      properties: {kind: {type: string}}
      required: [kind]
      discriminator: {propertyName: kind, mapping: {grandchild: Grandchild}}
    Child:
      allOf: [{$ref: '#/components/schemas/Parent'}, {type: object, properties: {x: {type: string}}}]
    Grandchild:
      allOf: [{$ref: '#/components/schemas/Child'}, {type: object, properties: {y: {type: string}}}]
`)
	_, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-discriminator")
	source = []byte(strings.Replace(string(source), "grandchild: Grandchild", "child: Child", 1))
	_, diagnostics, err = Analyze(source)
	require.NoError(t, err)
	for _, diagnostic := range diagnostics {
		require.NotEqual(t, "schema-discriminator", diagnostic.Code, "unsupported child composition must not reject the parent hint")
	}
}
