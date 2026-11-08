package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolverSourceDialect(t *testing.T) {
	for _, version := range []string{"3.1.1", "3.2.0"} {
		for _, tc := range []struct {
			name, kind, body string
			blocked          bool
		}{
			{"schema", "schemas", "{type: string}", true},
			{"boolean schema", "schemas", "true", true},
			{"body", "requestBodies", "{content: {application/json: {schema: {type: string}}}}", true},
			{"response", "responses", "{description: OK, content: {application/json: {schema: {type: string}}}}", true},
			{"header", "headers", "{schema: {type: string}}", true},
			{"parameter", "parameters", "{name: value, in: query, schema: {type: string}}", true},
			{"path item", "pathItems", "{post: {requestBody: {content: {application/json: {schema: {type: string}}}}, responses: {'204': {description: OK}}}}", true},
			{"response without schema", "responses", "{description: OK}", false},
			{"example data", "examples", "{value: {schema: {type: string}}}", false},
			{"security scheme", "securitySchemes", "{type: apiKey, in: header, name: Key}", false},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				entry := fmt.Sprintf("openapi: %s\ninfo: {title: Entry, version: '1'}\npaths: {}\ncomponents:\n  %s:\n    X: {$ref: 'model.yaml#/components/%s/X'}\n", version, tc.kind, tc.kind)
				body := fmt.Sprintf("openapi: %s\njsonSchemaDialect: https://example.test/custom-dialect\ncomponents:\n  %s:\n    X: %s\n", version, tc.kind, tc.body)
				source := resolverFiles(t, []byte(entry), map[string]string{"model.yaml": body})
				resolved, diagnostics, err := resolveDescription(source)
				require.NoError(t, err)
				ref := resolverEdge(t, resolved, "model.yaml#/components/"+tc.kind+"/X")
				if tc.blocked {
					requireDiagnosticCode(t, diagnostics, "json-schema-dialect")
					require.Nil(t, ref.target)
				} else {
					require.Empty(t, diagnostics)
					require.NotNil(t, ref.target)
				}
			})
		}
	}
}

func TestLooseDocumentDialectName(t *testing.T) {
	source := resolverFiles(t, resolverEntry("", "model.yaml#/X"), map[string]string{
		"model.yaml": "jsonSchemaDialect: ordinary data\nX: {type: string}\n",
	})
	_, diagnostics, err := resolveDescription(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
}

func TestExternalDialectSelection(t *testing.T) {
	for _, tc := range []struct{ name, reference, declaration string }{
		{"schema", "content: {application/json: {schema: {$ref: 'model.yaml#/components/schemas/X'}}}", "schemas:\n    X: {type: string}"},
		{"body", "$ref: 'model.yaml#/components/requestBodies/X'", "requestBodies:\n    X: {content: {application/json: {schema: {type: string}}}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := fmt.Sprintf(`openapi: 3.2.0
info: {title: Entry, version: '1'}
paths:
  /bad:
    post:
      requestBody: {%s}
      responses: {'204': {description: Done}}
  /good:
    get:
      responses: {'204': {description: Done}}
`, tc.reference)
			external := "openapi: 3.2.0\njsonSchemaDialect: https://example.test/custom-dialect\ncomponents:\n  " + tc.declaration + "\n"
			source := resolverFiles(t, []byte(entry), map[string]string{"model.yaml": external})
			for _, selected := range []string{"", "/bad", "/good"} {
				t.Run(selected, func(t *testing.T) {
					selection := Selection{}
					if selected != "" {
						selection.PathPrefixes = []string{selected}
					}
					document, diagnostics, _, err := AnalyzeSelected(source, selection)
					require.NoError(t, err)
					design, err := Render(document, Options{})
					if selected == "/good" {
						require.Empty(t, diagnostics)
						require.NoError(t, err)
						require.NotEmpty(t, design)
					} else {
						requireDiagnosticCode(t, diagnostics, "json-schema-dialect")
						require.ErrorContains(t, err, "json-schema-dialect")
						require.Empty(t, design)
						require.NotNil(t, diagnostics[0].Origin)
						require.Contains(t, diagnostics[0].Origin.Authored, "model.yaml#/")
					}
				})
			}
			partial, _, err := AnalyzePartial(source, Selection{}, false)
			require.NoError(t, err)
			require.Len(t, partial.Document.Operations, 1)
			require.Equal(t, "/good", partial.Document.Operations[0].Path)
			require.Len(t, partial.Skipped, 1)
			require.Equal(t, "/bad", partial.Skipped[0].Path)
			requireDiagnosticCode(t, partial.Skipped[0].Diagnostics, "json-schema-dialect")
		})
	}
}

func TestDescriptionDuplicateKeys(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"schema constraints", "type: string\nmaxLength: 2\nmaxLength: 200\n"},
		{"nested schema", "type: object\nproperties:\n  child: {type: string, maxLength: 2, maxLength: 200}\n"},
		{"JSON", `{"type":"string","maxLength":2,"maxLength":200}`},
		{"anchor", "base: &base {type: string, maxLength: 2, maxLength: 200}\nX: {<<: *base}\n"},
		{"repeated merge", "a: &a {type: string}\nb: &b {maxLength: 2}\nX: {<<: *a, <<: *b}\n"},
		{"explicit after merge", "base: &base {type: string}\nX: {<<: *base, maxLength: 2, maxLength: 200}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := parseDescriptionTree([]byte(tc.source))
			require.ErrorContains(t, err, "already defined")
			source := resolverFiles(t, resolverEntry("", "model.yaml"), map[string]string{"model.yaml": tc.source})
			document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
			require.NoError(t, err)
			requireDiagnosticCode(t, diagnostics, "external-reference")
			require.Contains(t, diagnostics.Error(), "already defined")
			_, err = Render(document, Options{})
			require.ErrorContains(t, err, "external-reference")
		})
	}
	for _, tc := range []struct{ name, source, maximum string }{
		{"explicit override", "base: &base {type: string, maxLength: 2}\nX: {<<: *base, maxLength: 200}\n", "200"},
		{"first merge wins", "a: &a {type: string, maxLength: 2}\nb: &b {maxLength: 200}\nX: {<<: [*a, *b]}\n", "2"},
		{"aliased sequence", "a: &a {type: string, maxLength: 2}\nb: &b {maxLength: 200}\nbases: &bases [*a, *b]\nX: {<<: *bases}\n", "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _, err := parseDescriptionTree([]byte(tc.source))
			require.NoError(t, err)
			maximum, err := descriptionNode(root, "/X/maxLength")
			require.NoError(t, err)
			require.Equal(t, tc.maximum, maximum.Value)
			source := resolverFiles(t, resolverEntry("", "model.yaml#/X"), map[string]string{"model.yaml": tc.source})
			document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			_, err = Render(document, Options{})
			require.NoError(t, err)
		})
	}
}
