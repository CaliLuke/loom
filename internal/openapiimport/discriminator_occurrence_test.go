package openapiimport

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiscriminatorOccurrenceSelection(t *testing.T) {
	for _, tc := range []struct {
		name, media, encoding, schema string
		tagged, rejected              bool
	}{
		{name: "JSON body", media: "application/json", schema: `{$ref: '#/components/schemas/Event'}`, tagged: true},
		{name: "JSON form field", media: "application/x-www-form-urlencoded", tagged: true},
		{name: "JSON multipart field", media: "multipart/form-data", tagged: true},
		{name: "explicit JSON field", media: "application/x-www-form-urlencoded", encoding: "            encoding: {event: {contentType: application/json}}\n", tagged: true},
		{name: "deepObject envelope field stays full object", media: "application/x-www-form-urlencoded", encoding: "            encoding: {event: {style: deepObject, explode: true}}\n"},
		{name: "form root", media: "application/x-www-form-urlencoded", schema: `{$ref: '#/components/schemas/Event'}`},
		{name: "multipart root", media: "multipart/form-data", schema: `{$ref: '#/components/schemas/Event'}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.schema == "" {
				tc.schema = `{type: object, properties: {event: {$ref: '#/components/schemas/Event'}}, required: [event]}`
			}
			operation := fmt.Sprintf(`
  /events:
    post:
      requestBody:
        content:
          %s:
            schema: %s
%s      responses: {'204': {description: done}}`, tc.media, tc.schema, tc.encoding)
			doc, diagnostics, err := Analyze(discriminatorEnvelopeDocument("", "", `{propertyName: kind, mapping: {created: Created, deleted: Deleted}}`, operation))
			require.NoError(t, err)
			if tc.rejected {
				require.NotEmpty(t, diagnostics)
				return
			}
			require.Empty(t, diagnostics)
			design, err := Render(doc, Options{PackageName: "design"})
			require.NoError(t, err)
			if tc.tagged {
				require.Contains(t, string(design), `Meta("oneof:value:field", "data")`)
			} else {
				require.NotContains(t, string(design), `Meta("oneof:value:field", "data")`)
			}
		})
	}
}

func TestDiscriminatorSharedOccurrenceAndNestedBranch(t *testing.T) {
	for _, extra := range []string{
		`    Container:
      oneOf: [{$ref: '#/components/schemas/Event'}, {type: string}]
`,
		`    Container:
      oneOf:
        - {type: object, properties: {event: {$ref: '#/components/schemas/Event'}}}
        - {type: string}
`,
		`    SharedEnvelope: {$ref: '#/components/schemas/Created'}
`,
	} {
		t.Run(extra, func(t *testing.T) {
			source := append(discriminatorEnvelopeDocument("", "", `{propertyName: kind, mapping: {created: Created, deleted: Deleted}}`, "{}"), []byte(extra)...)
			doc, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(doc, Options{PackageName: "design"})
			require.NoError(t, err)
			require.NotContains(t, string(design), `Meta("oneof:value:field", "data")`)
		})
	}
}

func TestDiscriminatorExactEnvelopeFacts(t *testing.T) {
	base := string(discriminatorEnvelopeDocument("", "", `{propertyName: kind, mapping: {created: Created, deleted: Deleted}}`, "{}"))
	for _, tc := range []struct {
		name, old, value string
		rejected         bool
	}{
		{name: "explicit open", old: "      description: Created wire envelope.", value: "      additionalProperties: true"},
		{name: "title", old: "      description: Created wire envelope.", value: "      title: Wire"},
		{name: "example", old: "      description: Created wire envelope.", value: "      example: {kind: created, data: sample}"},
		{name: "read only", old: "      description: Created wire envelope.", value: "      readOnly: true"},
		{name: "write only", old: "      description: Created wire envelope.", value: "      writeOnly: true"},
		{name: "deprecated false", old: "      description: Created wire envelope.", value: "      deprecated: false"},
		{name: "extension", old: "      description: Created wire envelope.", value: "      x-audit: true"},
		{name: "two enum values", old: "enum: [created]", value: "enum: [created, made]"},
		{name: "tag reference", old: "{type: string, enum: [created]}", value: "{$ref: '#/components/schemas/Tag'}"},
		{name: "different payload key", old: "data: {type: integer}", value: "payload: {type: integer}"},
		{name: "payload object", old: "data: {type: string}", value: "data: {type: object, properties: {name: {type: string}}}"},
		{name: "extra member", old: "      description: Created wire envelope.", value: "      description: Created wire envelope."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(base, tc.old, tc.value, 1)
			if tc.name == "tag reference" {
				source += "    Tag: {type: string, enum: [created]}\n"
			}
			if tc.name == "extra member" {
				source = strings.Replace(source, "        data: {type: string}", "        data: {type: string}\n        extra: {type: string}", 1)
			}
			doc, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if tc.rejected {
				require.NotEmpty(t, diagnostics)
				return
			}
			require.Empty(t, diagnostics)
			design, err := Render(doc, Options{PackageName: "design"})
			require.NoError(t, err)
			require.NotContains(t, string(design), `Meta("oneof:value:field", "data")`)
		})
	}
}

func TestDiscriminatorNestedEnvelopeReferenceStaysFullObject(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Nested envelope use, version: '1'}
paths: {}
components:
  schemas:
    Event:
      oneOf:
        - type: object
          properties:
            kind: {type: string, enum: [created]}
            data: {$ref: '#/components/schemas/Event/oneOf/1'}
          required: [kind, data]
        - type: object
          properties:
            kind: {type: string, enum: [deleted]}
            data: {type: integer}
          required: [kind, data]
      discriminator:
        propertyName: kind
        mapping: {created: '#/components/schemas/Event/oneOf/0', deleted: '#/components/schemas/Event/oneOf/1'}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.NotContains(t, string(design), `Meta("oneof:value:field", "data")`, "a payload reference is an independent use of the envelope")
}

func TestDiscriminatorJSONQueryOccurrence(t *testing.T) {
	operation := `
  /events:
    get:
      parameters:
        - name: event
          in: query
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Event'}
      responses: {'204': {description: done}}`
	document, diagnostics, err := Analyze(discriminatorEnvelopeDocument("", "", `{propertyName: kind, mapping: {created: Created, deleted: Deleted}}`, operation))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.NotContains(t, string(design), `Meta("oneof:value:field", "data")`, "native tagged parameter unions remain outside the codec grammar")
	require.Contains(t, string(design), `Meta("openapi:contentType", "application/json")`)
}

func TestJSONQueryUnionPreservesCapturedContent(t *testing.T) {
	for _, hinted := range []bool{false, true} {
		t.Run(fmt.Sprintf("hint=%t", hinted), func(t *testing.T) {
			source := string(discriminatorDocument("3.2.0", `{propertyName: kind}`, `[{$ref: '#/components/schemas/Cat'}, {$ref: '#/components/schemas/Dog'}]`, ""))
			if !hinted {
				source = strings.Replace(source, "      discriminator: {propertyName: kind}\n", "", 1)
			}
			source = strings.Replace(source, "paths: {}", `paths:
  /pets:
    get:
      parameters:
        - name: pet
          in: query
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
      responses: {'204': {description: done}}`, 1)
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			design, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			require.Contains(t, string(design), `Meta("openapi:contentType", "application/json")`, "native FullObject union defaults to a style codec, so imported JSON content must remain explicit")
		})
	}
}
