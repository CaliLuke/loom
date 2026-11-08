package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportProducerDefaultDialect(t *testing.T) {
	for _, tc := range []struct {
		name, version, dialect string
		valid                  bool
	}{
		{"3.1 default", "3.1.1", "https://spec.openapis.org/oas/3.1/dialect/base", true},
		{"3.2 default", "3.2.0", "https://spec.openapis.org/oas/3.1/dialect/base", true},
		{"custom", "3.2.0", "https://example.test/custom", false},
		{"different JSON Schema dialect", "3.2.0", "https://json-schema.org/draft/2020-12/schema", false},
		{"3.0 has no dialect field", "3.0.3", "https://spec.openapis.org/oas/3.1/dialect/base", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := "openapi: " + tc.version + "\njsonSchemaDialect: " + tc.dialect + "\ninfo: {title: Dialect, version: '1'}\npaths:\n  /value:\n    get:\n      operationId: value\n      responses:\n        '200': {description: value, content: {application/json: {schema: {type: string}}}}\n"
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if !tc.valid {
				require.Contains(t, diagnostics.Error(), "json-schema-dialect")
				return
			}
			require.Empty(t, diagnostics)
			_, err = Render(document, Options{})
			require.NoError(t, err)
		})
	}
}

func TestImportProducerHeaderExamples(t *testing.T) {
	for _, tc := range []struct {
		name, fields, schema string
		media                any
		hasMedia             bool
	}{
		{"distinct scopes", "example: wire", "{type: string, example: schema}", "wire", true},
		{"schema only", "", "{type: string, example: schema}", nil, false},
		{"explicit null", "example: null", "{type: [string, 'null'], example: schema}", nil, true},
		{"merged header", "<<: {example: wire}", "{type: string, example: schema}", "wire", true},
		{"named example", "examples: {wire: {summary: wire summary, description: wire description, value: wire}}", "{type: string, example: schema}", "wire", true},
		{"reusable example", "examples: {wire: {$ref: '#/components/examples/Wire'}}", "{type: string, example: schema}", "wire", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(producerHeaderSource(tc.fields, tc.schema))
			require.NoError(t, err)
			if tc.name == "explicit null" {
				require.Contains(t, diagnostics.Error(), "nullable values require a JSON codec")
			} else {
				require.Empty(t, diagnostics)
			}
			header := document.Operations[0].Responses[0].Response.Headers[0].Header
			examples := header.Schema.Examples
			count := 1
			if tc.hasMedia {
				count++
			}
			require.Len(t, examples, count)
			require.Equal(t, "schema", examples[0].Scope)
			require.Equal(t, "schema", examples[0].Value)
			if tc.hasMedia {
				require.Equal(t, "media", examples[1].Scope)
				require.Equal(t, tc.media, examples[1].Value)
			}
			if tc.name == "named example" {
				require.Equal(t, "wire summary", examples[1].Summary)
				require.Equal(t, "wire description", examples[1].Description)
			}
			if tc.name == "reusable example" {
				require.Equal(t, "Wire", examples[1].ComponentName)
			}
			out, err := Render(document, Options{})
			if tc.name == "explicit null" {
				require.ErrorContains(t, err, "nullable values require a JSON codec")
				return
			}
			require.NoError(t, err)
			require.Contains(t, string(out), `Meta("openapi:example:scope", "schema")`)
			if tc.hasMedia {
				require.Contains(t, string(out), `Meta("openapi:example:scope", "media")`)
			} else {
				require.NotContains(t, string(out), `Meta("openapi:example:scope", "media")`)
			}
		})
	}
}

func TestImportProducerHeaderExamplesRejectUnsupported(t *testing.T) {
	for _, fields := range []string{
		"examples: {wire: {externalValue: 'https://example.test/value'}}",
		"examples: {wire: {value: wire, x-private: true}}",
		"example: 42",
	} {
		t.Run(fields, func(t *testing.T) {
			_, diagnostics, err := Analyze(producerHeaderSource(fields, "{type: string}"))
			require.NoError(t, err)
			require.Contains(t, diagnostics.Error(), "examples")
		})
	}
}

func TestImportProducerSSEHeaderExamplesCompile(t *testing.T) {
	source := producerSSEMetadataSource()
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	out, err := Render(document, Options{})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, out, 1, `
 response := expr.Root.API.HTTP.Services[0].HTTPEndpoints[0].Responses[0]
 if response.Handshake == nil { t.Fatal("missing typed handshake") }
 fields := expr.AsObject(response.Handshake.Type)
 schemaCount, mediaCount := 0, 0
 for _, field := range *fields {
  if len(field.Attribute.UserExamples) == 0 { continue }
  values := map[string]any{}
  for _, example := range field.Attribute.UserExamples {
   scope, ok := example.Meta.Last("openapi:example:scope")
   if ok { values[scope] = example.Value }
  }
  if value, ok := values["schema"]; ok {
   if value != "schema" { t.Fatalf("wrong schema sample: %#v", value) }
   schemaCount++
  }
  if values["media"] != "wire" { t.Fatalf("wrong media sample: %#v", values) }
  mediaCount++
 }
 if schemaCount != 1 || mediaCount != 2 { t.Fatalf("lost scope owners: schema=%d media=%d", schemaCount, mediaCount) }
 declared := expr.Root.UserType("HeaderValue")
 if declared == nil || len(declared.Attribute().UserExamples) != 1 { t.Fatal("lost header declaration sample") }
 sample := declared.Attribute().UserExamples[0]
 scope, _ := sample.Meta.Last("openapi:example:scope")
 if scope != "schema" || sample.Value != "declared" { t.Fatalf("changed declaration sample: %#v", sample) }
`)
}

func TestImportProducerExternalDefaultDialect(t *testing.T) {
	for _, dialect := range []string{"https://spec.openapis.org/oas/3.1/dialect/base", "https://example.test/custom"} {
		t.Run(dialect, func(t *testing.T) {
			source := resolverFiles(t, resolverEntry("", "model.yaml#/components/schemas/Value"), map[string]string{
				"model.yaml": "openapi: 3.2.0\njsonSchemaDialect: " + dialect + "\ncomponents:\n  schemas:\n    Value: {type: string}\n",
			})
			document, diagnostics, _, err := AnalyzeSelected(source, Selection{})
			require.NoError(t, err)
			if strings.Contains(dialect, "example.test") {
				require.Contains(t, diagnostics.Error(), "json-schema-dialect")
				return
			}
			require.Empty(t, diagnostics)
			_, err = Render(document, Options{})
			require.NoError(t, err)
		})
	}
}

func producerSSEMetadataSource() []byte {
	source := string(sseInverseSource([]string{"201"}, "", ""))
	source = strings.Replace(source, "openapi: 3.2.0\n", "openapi: 3.2.0\njsonSchemaDialect: https://spec.openapis.org/oas/3.1/dialect/base\n", 1)
	source = strings.Replace(source, "X-Trace: {required: true, schema: {type: string}}", "X-Trace: {required: true, example: wire, schema: {type: string, example: schema}}\n            X-Reference: {example: wire, schema: {$ref: '#/components/schemas/HeaderValue'}}", 1)
	return []byte(source + "    HeaderValue: {type: string, example: declared}\n")
}

func producerHeaderSource(fields, schema string) []byte {
	if fields != "" {
		fields += ", "
	}
	return []byte(`openapi: 3.2.0
info: {title: Headers, version: '1'}
paths:
  /value:
    get:
      operationId: value
      responses:
        '200':
          description: value
          headers:
            X-Value: {` + fields + `schema: ` + schema + `}
components:
  examples:
    Wire: {value: wire}
`)
}
