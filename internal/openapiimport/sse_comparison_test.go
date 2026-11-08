package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const sseComparisonMapExpansion = "{type: object, required: [message], properties: {message: {type: string}, attrs: {type: object, additionalProperties: {type: object, required: [k], properties: {k: {type: string}}}}}}"

// Consuming an admitted x-loom-async contract must not change rendered DSL.
// The expansion and reference spellings of the same derived message describe
// the same event, so both must render exactly the no-extension output.
func TestImportSSEComparisonDoesNotPublishDeclarations(t *testing.T) {
	render := func(extension string) string {
		document, diagnostics, err := Analyze(sseComparisonMapSource(extension))
		require.NoError(t, err)
		require.Empty(t, diagnostics)
		out, err := Render(document, Options{})
		require.NoError(t, err)
		return string(out)
	}
	without := render("")
	reference := render(sseAsyncContract("['201']", "{$ref: '#/components/schemas/Event'}"))
	expansion := render(sseAsyncContract("['201']", sseComparisonMapExpansion))
	require.Equal(t, without, reference, "reference spelling changed rendered DSL")
	require.Equal(t, without, expansion, "expansion spelling of consumed reserved metadata changed rendered DSL")
	requireRenderedDesignEvaluates(t, []byte(expansion), 1, "")
}

// A derived message naming a different declaration conflicts with the
// event's declaration identity and must not be consumed as equal.
func TestImportSSEComparisonPreservesDeclarationIdentity(t *testing.T) {
	other := "    Other:\n      type: object\n      required: [message]\n      properties:\n        message: {type: string}\n"
	source := string(sseInverseSource([]string{"201"}, "", sseAsyncContract("['201']", "{$ref: '#/components/schemas/Other'}"))) + other
	_, diagnostics, err := Analyze([]byte(source))
	require.NoError(t, err)
	require.Contains(t, diagnostics.Error(), "sse-async")
}

func TestImportSSEComparisonChildDeclarationIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, child string
		valid       bool
	}{
		{"same declaration", "{$ref: '#/components/schemas/Child'}", true},
		{"distinct declaration", "{$ref: '#/components/schemas/OtherChild'}", false},
		{"alias declaration", "{$ref: '#/components/schemas/Alias'}", false},
		{"one sided expansion", "{type: string}", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			extension := sseAsyncContract("['201']", "{type: object, required: [message], properties: {message: "+tc.child+"}}")
			source := strings.Replace(string(sseInverseSource([]string{"201"}, "", extension)), "message: {type: string}\n", "message: {$ref: '#/components/schemas/Child'}\n", 1)
			source += "    Child: {type: string}\n    OtherChild: {type: string}\n    Alias: {$ref: '#/components/schemas/Child'}\n"
			_, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if tc.valid {
				require.Empty(t, diagnostics)
			} else {
				require.Contains(t, diagnostics.Error(), "sse-async")
			}
		})
	}
}

func TestImportSSEComparisonRejectsUnsupportedFacts(t *testing.T) {
	for _, tc := range []struct{ name, extra string }{
		{"content encoding", "contentEncoding: base64"},
		{"raw binary codec", "format: binary"},
		{"content media type", "contentMediaType: application/json"},
		{"content schema", "contentSchema: {type: integer}"},
		{"unknown keyword", "unevaluatedProperties: false"},
		{"opaque extension", "x-opaque: {codec: custom}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema := "{type: object, required: [message], properties: {message: {type: string, " + tc.extra + "}}}"
			_, diagnostics, err := Analyze(sseInverseSource([]string{"201"}, "", sseAsyncContract("['201']", schema)))
			require.NoError(t, err)
			require.NotEmpty(t, diagnostics)
			require.Contains(t, diagnostics.Error(), "x-loom-async")
		})
	}
}

func TestImportOrdinaryEmptyExamplesRequireRepresentation(t *testing.T) {
	for name, schema := range map[string]string{
		"inline":            "{type: string, examples: []}",
		"reference overlay": "{$ref: '#/components/schemas/Name', examples: []}",
	} {
		t.Run(name, func(t *testing.T) {
			_, diagnostics, err := Analyze([]byte(`openapi: 3.1.0
info: {title: EmptyExamples, version: '1'}
paths:
  /names:
    get:
      operationId: names
      responses:
        '200':
          description: names
          content:
            application/json:
              schema: ` + schema + `
components:
  schemas:
    Name: {type: string, examples: [declared]}
`))
			require.NoError(t, err)
			require.Contains(t, diagnostics.Error(), "schema-examples-presence")
		})
	}
}

func TestSSEExampleComparisonIgnoresAcquisitionProvenance(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  any
		change func(*Example)
		equal  bool
	}{
		{"same contract", "sample", func(*Example) {
		}, true},
		{"different value", "sample", func(e *Example) {
			e.Value = "other"
		}, false},
		{"different description", "sample", func(e *Example) {
			e.Description = "other"
		}, false},
		{"different name", "sample", func(e *Example) {
			e.Name = "other"
		}, false},
		{"different summary", "sample", func(e *Example) {
			e.Summary = "other"
		}, false},
		{"different reference", "sample", func(e *Example) {
			e.ComponentName = "Other"
		}, false},
		{"unresolved versus null", nil, func(e *Example) {
			e.unresolved = true
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			left := &Schema{Type: "string", Nullable: true, sourcePath: "#/responses/200"}
			right := &Schema{Type: "string", Nullable: true, sourcePath: "#/responses/201"}
			left.Examples = []Example{{Value: tc.value, source: exampleSource{owner: left, path: "#/responses/200/example"}}}
			right.Examples = []Example{{Value: tc.value, source: exampleSource{owner: right, path: "#/responses/201/example", referenced: true}}}
			tc.change(&right.Examples[0])
			planner := documentPlanner{}
			for _, comparison := range []struct {
				name  string
				equal func(*Schema, *Schema) bool
			}{
				{"occurrence", sameSSEOccurrence},
				{"graph", func(a, b *Schema) bool {
					return planner.sameSSEGraph(a, b, make(map[sseSchemaPair]bool))
				}},
			} {
				t.Run(comparison.name, func(t *testing.T) {
					require.Equal(t, tc.equal, comparison.equal(left, right))
					require.Equal(t, tc.equal, comparison.equal(right, left))
				})
			}
			require.Same(t, left, left.Examples[0].source.owner)
			require.Same(t, right, right.Examples[0].source.owner)
			require.Equal(t, "#/responses/200/example", left.Examples[0].source.path)
			require.Equal(t, "#/responses/201/example", right.Examples[0].source.path)
			require.True(t, right.Examples[0].source.referenced)
		})
	}
}

func sseComparisonMapSource(extension string) []byte {
	source := string(sseInverseSource([]string{"201"}, "", extension))
	return []byte(strings.Replace(source, "        message: {type: string}\n", "        message: {type: string}\n        attrs: {type: object, additionalProperties: {type: object, required: [k], properties: {k: {type: string}}}}\n", 1))
}
