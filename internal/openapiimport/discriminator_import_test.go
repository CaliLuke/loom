package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func discriminatorDocument(version, discriminator, branches, extra string) []byte {
	return []byte(fmt.Sprintf(`openapi: %s
info: {title: Discriminators, version: '1'}
paths: {}
components:
  schemas:
    Cat:
      type: object
      properties: {kind: {type: string, enum: [cat]}, name: {type: string}}
      required: [kind, name]
    Dog:
      type: object
      properties: {kind: {type: string, enum: [dog]}, age: {type: integer}}
      required: [kind, age]
    Pet:
      oneOf: %s
      discriminator: %s
%s
`, version, branches, discriminator, extra))
}

func TestDiscriminatorImportAdmission(t *testing.T) {
	for _, tc := range []struct{ name, version, hint, branches, extra, code, message string }{
		{name: "aliases", hint: `{propertyName: kind, mapping: {cat: Cat, kitty: Cat, dog: Dog}}`},
		{name: "shadow implicit", hint: `{propertyName: kind, mapping: {Cat: Dog}}`},
		{name: "duplicates", hint: `{propertyName: kind, mapping: {cat: Cat}}`, branches: `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`},
		{name: "external fallback", hint: `{propertyName: kind, defaultMapping: 'https://E.TEST:443/model#/Other', x-audit: {z: 1, a: true}}`},
		{name: "wrong target", hint: `{propertyName: kind, mapping: {pet: Pet}}`, code: "schema-discriminator", message: "alternate"},
		{name: "missing target", hint: `{propertyName: kind, mapping: {cat: Missing}}`, code: "schema-discriminator", message: "unresolved"},
		{name: "external mapping", hint: `{propertyName: kind, mapping: {cat: 'https://e.test/Cat'}}`, code: "schema-discriminator", message: "alternate"},
		{name: "old fallback", version: "3.1.1", hint: `{propertyName: kind, defaultMapping: Cat}`, code: "versioned-field", message: "defaultMapping"},
		{name: "old extension", version: "3.0.3", hint: `{propertyName: kind, x-audit: true}`, code: "versioned-field", message: "extension"},
		{name: "unknown field", hint: `{propertyName: kind, unexpected: true}`, code: "schema-discriminator", message: "unexpected"},
		{name: "unknown sibling", hint: `{propertyName: kind}`, extra: "      minProperties: 1", code: "schema-composition", message: "minProperties"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.version == "" {
				tc.version = "3.2.0"
			}
			if tc.branches == "" {
				tc.branches = `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`
			}
			doc, diagnostics, err := Analyze(discriminatorDocument(tc.version, tc.hint, tc.branches, tc.extra))
			require.NoError(t, err)
			if tc.code != "" {
				for _, d := range diagnostics {
					if d.Code == tc.code && strings.Contains(d.Message, tc.message) {
						return
					}
				}
				t.Errorf("missing %s/%s: %#v", tc.code, tc.message, diagnostics)
				return
			}
			require.Empty(t, diagnostics)
			rendered, err := Render(doc, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Contains(t, string(rendered), `Meta("openapi:discriminator:propertyName", "kind")`)
			if tc.name == "aliases" {
				require.Contains(t, string(rendered), `Meta("openapi:discriminator:mapping", "kitty")`)
			}
			if tc.name == "duplicates" {
				require.Equal(t, 1, strings.Count(string(rendered), `Meta("openapi:discriminator:mapping", "cat")`))
			}
		})
	}
}

func discriminatorEnvelopeDocument(extra, tagExtra, hint, operation string) []byte {
	return []byte(fmt.Sprintf(`openapi: 3.2.0
info: {title: Envelopes, version: '1'}
paths: %s
components:
  schemas:
    Created:
      type: object
      description: Created wire envelope.
      properties:
        kind: {type: string, enum: [created]%s}
        data: {type: string}
      required: [kind, data]
%s
    Deleted:
      type: object
      properties:
        kind: {type: string, const: deleted}
        data: {type: integer}
      required: [kind, data]
    Event:
      type: object
      oneOf: [{$ref: '#/components/schemas/Created'}, {$ref: '#/components/schemas/Deleted'}]
      discriminator: %s
`, operation, tagExtra, extra, hint))
}

func TestDiscriminatorEnvelopeSelection(t *testing.T) {
	for _, tc := range []struct {
		name, extra, tagExtra, hint, operation string
		tagged                                 bool
	}{
		{name: "exact", tagged: true},
		{name: "closed branch", extra: "      additionalProperties: false"},
		{name: "tag assertion", tagExtra: ", maxLength: 20"},
		{name: "tag annotation", tagExtra: ", description: discriminator"},
		{name: "mismatched mapping", hint: `{propertyName: kind, mapping: {made: Created, deleted: Deleted}}`},
		{name: "alias", hint: `{propertyName: kind, mapping: {created: Created, made: Created, deleted: Deleted}}`, tagged: true},
		{name: "form root", operation: `
  /events:
    post:
      requestBody:
        content:
          application/x-www-form-urlencoded:
            schema: {$ref: '#/components/schemas/Event'}
      responses: {'204': {description: done}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hint == "" {
				tc.hint = `{propertyName: kind, mapping: {created: Created, deleted: Deleted}}`
			}
			if tc.operation == "" {
				tc.operation = "{}"
			}
			doc, diagnostics, err := Analyze(discriminatorEnvelopeDocument(tc.extra, tc.tagExtra, tc.hint, tc.operation))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(doc, Options{PackageName: "design"})
			require.NoError(t, err)
			if tc.tagged {
				require.Contains(t, string(design), `Meta("oneof:value:field", "data")`)
				require.Contains(t, string(design), `Meta("openapi:component:unionEnvelope", "Created")`)
			} else {
				require.Contains(t, string(design), `Meta("openapi:discriminator:propertyName", "kind")`)
			}
		})
	}
}

func TestDiscriminatorReferenceClosure(t *testing.T) {
	operation := `
  /events:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Event', unevaluatedProperties: false}
      responses: {'204': {description: done}}`
	document, diagnostics, err := Analyze(discriminatorEnvelopeDocument("", "", `{propertyName: kind, mapping: {created: Created, deleted: Deleted}}`, operation))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("openapi:unevaluatedProperties", "false")`)
}

func TestDiscriminatorFullObjectReferenceClosure(t *testing.T) {
	source := string(discriminatorDocument("3.2.0", `{propertyName: kind}`, `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`, ""))
	source = strings.Replace(source, "paths: {}", `paths:
  /pets:
    post:
      requestBody:
        content:
          application/json:
            schema: {$ref: '#/components/schemas/Pet', unevaluatedProperties: false}
      responses: {'204': {description: done}}`, 1)
	document, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("openapi:unevaluatedProperties", "false")`)
}

func TestDiscriminatorSurvivesNullableNormalization(t *testing.T) {
	source := string(discriminatorDocument("3.2.0", `{propertyName: kind, mapping: {cat: Cat, dog: Dog}}`, `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`, ""))
	source = strings.Replace(source, "    Pet:\n      oneOf:", "    Pet:\n      anyOf:\n        - type: 'null'\n        - oneOf:", 1)
	source = strings.Replace(source, "      discriminator:", "          discriminator:", 1)
	document, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Contains(t, string(design), `Meta("openapi:discriminator:propertyName", "kind")`)
	require.Contains(t, string(design), `Meta("openapi:discriminator:mapping", "cat")`)
	require.Contains(t, string(design), `Nullable()`)
}

func TestDiscriminatorNullableNormalizationKeepsSiblingAdmission(t *testing.T) {
	source := string(discriminatorDocument("3.2.0", `{propertyName: kind, mapping: {cat: Cat, dog: Dog}}`, `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`, "      type: string"))
	source = strings.Replace(source, "    Pet:\n      oneOf:", "    Pet:\n      anyOf:\n        - type: 'null'\n        - oneOf:", 1)
	source = strings.Replace(source, "      discriminator:", "          discriminator:", 1)
	source = strings.Replace(source, "      type: string", "          type: string", 1)
	_, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	requireDiagnosticCode(t, diagnostics, "schema-composition")
}
