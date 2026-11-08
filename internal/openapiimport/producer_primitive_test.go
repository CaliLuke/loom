package openapiimport

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportProducerPrimitiveLayers(t *testing.T) {
	for _, tc := range []struct{ name, base, local string }{
		{"integer", "{type: integer, format: int64, minimum: 0, example: 1}", "minimum: 5, example: 8"},
		{"string", "{type: string, minLength: 1, example: base}", "minLength: 2, example: local"},
		{"number", "{type: number, minimum: 0, example: 1.5}", "maximum: 10, example: 2.5"},
		{"boolean", "{type: boolean, example: true}", "example: false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("openapi: 3.2.0\ninfo: {title: Primitive, version: '1'}\npaths: {}\ncomponents:\n  schemas:\n    Value: {allOf: [" + tc.base + "], " + tc.local + "}\n")
			document, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			require.Len(t, document.Components.Schemas, 2)
			value := document.Components.Schemas[0].Schema
			require.Empty(t, value.Ref)
			require.Len(t, value.Bases, 1)
			require.Empty(t, value.Bases[0].Ref)
			require.Same(t, value.Bases[0], document.Components.Schemas[1].Schema)
			require.Len(t, value.Examples, 1)
			require.Len(t, value.Bases[0].Examples, 1)
			require.NotEqual(t, value.Examples[0].Value, value.Bases[0].Examples[0].Value)
			require.Equal(t, "#/components/schemas/Value/allOf/0", value.Bases[0].sourcePath)
			out, err := Render(document, Options{})
			require.NoError(t, err)
			require.Contains(t, string(out), `Type("Value", ImportedValueBase`)
		})
	}
}

func TestImportProducerPrimitiveWitness(t *testing.T) {
	source, err := os.ReadFile("testdata/producer_primitive.json")
	require.NoError(t, err)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	require.Len(t, document.Components.Schemas, 1)
	operation := document.Operations[0]
	first, second := operation.Responses[0].Response, operation.Responses[1].Response
	require.Same(t, first.Schema, second.Schema)
	require.NotEqual(t, first.stream.path, second.stream.path)
	require.Len(t, first.Schema.Bases, 1)
	require.Same(t, first.Schema.Bases[0], document.Components.Schemas[0].Schema)
	require.Empty(t, first.Schema.Ref)
	out, err := Render(document, Options{})
	require.NoError(t, err)
	require.NotContains(t, string(out), `Meta("openapi:typename:canonical", "true")`)
	require.Contains(t, string(out), `Minimum("5")`)
	require.Contains(t, string(out), `Minimum("0")`)
	pruner := walkSelectedRoots(document, nil)
	pruner.retain(document)
	require.Len(t, document.Components.Schemas, 1, "selection must retain the recorded native constructor")
	_, err = Render(document, Options{})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, out, 1, `
 event := expr.Root.Services[0].Methods[0].StreamingResult
 if event.Validation == nil || event.Validation.Minimum == nil || event.Validation.Minimum.String() != "5" { t.Fatal("local minimum lost") }
 base, ok := event.Type.(expr.UserType)
 if !ok || base.Attribute().Validation.Minimum.String() != "0" { t.Fatal("contributor minimum lost") }
 if len(event.UserExamples) != 1 || len(base.Attribute().UserExamples) != 1 { t.Fatalf("example ownership lost: local=%#v base=%#v", event.UserExamples, base.Attribute().UserExamples) }
 if event.UserExamples[0].Value == base.Attribute().UserExamples[0].Value { t.Fatal("examples collapsed") }
 if _, canonical := base.Attribute().Meta["openapi:typename:canonical"]; canonical { t.Fatal("constructor claimed canonical identity") }
 if len(expr.Root.API.HTTP.Services[0].HTTPEndpoints[0].Responses) != 2 { t.Fatal("status occurrences lost") }
 `)
}

func TestImportProducerPrimitiveRejectsUnsupported(t *testing.T) {
	for _, schema := range []string{
		"{allOf: [{type: integer}, {minimum: 0}]}",
		"{allOf: [{type: [integer, 'null']}]}",
		"{allOf: [{type: integer}], type: number}",
		"{allOf: [{type: integer, format: int64}], format: int32}",
		"{allOf: [{type: object, properties: {value: {type: string}}}]}",
		"{allOf: [{type: array, items: {type: string}}]}",
	} {
		t.Run(schema, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte("openapi: 3.2.0\ninfo: {title: Rejected, version: '1'}\npaths: {}\ncomponents: {schemas: {Value: " + schema + "}}\n"))
			require.NoError(t, err)
			require.Contains(t, diagnostics.Error(), "schema-composition")
			_, err = Render(document, Options{})
			require.Error(t, err)
		})
	}
}

func TestImportProducerPrimitiveUnequalEvents(t *testing.T) {
	source, err := os.ReadFile("testdata/producer_primitive.json")
	require.NoError(t, err)
	for _, tc := range []struct{ name, old, replacement string }{
		{"local minimum", `"minimum": 5`, `"minimum": 6`},
		{"contributor minimum", `"minimum": 0`, `"minimum": 1`},
		{"local sample", "4893012258144077634", "4893012258144077635"},
		{"contributor sample", "3210637814790232894", "3210637814790232895"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := strings.Replace(string(source), tc.old, tc.replacement, 1)
			require.NotEqual(t, string(source), changed)
			document, diagnostics, err := Analyze([]byte(changed))
			require.NoError(t, err)
			require.Contains(t, diagnostics.Error(), "sse-event")
			require.NotSame(t, document.Operations[0].Responses[0].Response.Schema, document.Operations[0].Responses[1].Response.Schema)
			_, err = Render(document, Options{})
			require.Error(t, err)
		})
	}
}

func TestImportProducerPrimitiveDeclarationIdentity(t *testing.T) {
	inline := &Schema{Type: "string"}
	components := Components{Schemas: []NamedSchema{
		{Name: "A", Schema: inline}, {Name: "B", Schema: &Schema{Type: "string"}},
	}}
	for _, tc := range []struct {
		name         string
		declarations []string
		valid        bool
	}{
		{"inline equality", []string{"", ""}, true},
		{"reference and inline", []string{"A", ""}, true},
		{"inline and reference", []string{"", "A"}, true},
		{"shared declaration", []string{"A", "A"}, true},
		{"distinct declarations", []string{"A", "B"}, false},
		{"inline then distinct declarations", []string{"", "A", "B"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operation := Operation{}
			for i, declaration := range tc.declarations {
				schema := inline
				identity := ""
				if declaration != "" {
					identity = componentDeclaration(declaration)
					schema = &Schema{Ref: identity}
				}
				operation.Responses = append(operation.Responses, StatusResponse{
					Status: []string{"200", "201", "202"}[i], Response: Response{Schema: schema, stream: &sseEvent{declaration: identity}},
				})
			}
			_, diagnostics := classifyOperationResponses(&operation, "#/operation", components)
			if tc.valid {
				require.Empty(t, diagnostics)
			} else {
				require.Contains(t, diagnostics.Error(), "sse-event")
			}
		})
	}
}

func TestImportProducerPrimitiveContributorExampleOwner(t *testing.T) {
	source, err := os.ReadFile("testdata/producer_primitive.json")
	require.NoError(t, err)
	changed := strings.ReplaceAll(string(source), "3210637814790232894", "1")
	changed = strings.ReplaceAll(changed, "4893012258144077634", "8")
	document, diagnostics, err := Analyze([]byte(changed))
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	local := document.Operations[0].Responses[0].Response.Schema
	require.Len(t, local.Examples, 1)
	require.Len(t, local.Bases[0].Examples, 1)
	require.EqualValues(t, 8, local.Examples[0].Value)
	require.EqualValues(t, 1, local.Bases[0].Examples[0].Value)
	out, err := Render(document, Options{})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, out, 1, `
 event := expr.Root.Services[0].Methods[0].StreamingResult
 base, ok := event.Type.(expr.UserType)
 if !ok { t.Fatal("lost contributor constructor") }
 if len(base.Attribute().UserExamples) != 1 || base.Attribute().UserExamples[0].Value != int64(1) && base.Attribute().UserExamples[0].Value != 1 { t.Fatalf("contributor sample: %#v", base.Attribute().UserExamples) }
 if len(event.UserExamples) != 1 { t.Fatalf("local sample count: %d; values=%#v", len(event.UserExamples), event.UserExamples) }
 if event.UserExamples[0].Value != int64(8) && event.UserExamples[0].Value != 8 { t.Fatal("local sample changed") }
 `)
}

func TestImportProducerPrimitiveMixedDeclaration(t *testing.T) {
	for _, target := range []string{"Event", "Other"} {
		t.Run(target, func(t *testing.T) {
			extension := sseAsyncContract("['200', '201']", "{$ref: '#/components/schemas/"+target+"'}")
			source := string(sseInverseSource([]string{"200", "201"}, "", extension))
			source = strings.Replace(source, "                      $ref: '#/components/schemas/Event'", "                      type: string", 1)
			source = strings.Replace(source, "      type: object\n      required: [message]\n      properties:\n        message: {type: string}", "      type: string", 1)
			source += "    Other: {type: string}\n"
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if target == "Other" {
				require.Contains(t, diagnostics.Error(), "sse-async")
				return
			}
			require.Empty(t, diagnostics)
			require.Equal(t, "#/components/schemas/Event", document.Operations[0].Responses[0].Response.Schema.Ref)
			out, err := Render(document, Options{})
			require.NoError(t, err)
			require.Contains(t, string(out), "StreamingResult(ImportedEvent)")
		})
	}
}
