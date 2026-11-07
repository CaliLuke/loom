package openapiimport

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type unconstrainedExampleCase struct {
	name string
	// media is the application/json media object of GET /value.
	media string
	// components is the optional components object.
	components string
	// source replaces the generated document when set.
	source string
	// want lists retained examples in media, response schema and component
	// schema order.
	want []unconstrainedExampleValue
	// diagnostics lists diagnostics that must be reported. Accepted cases
	// require no diagnostics at all.
	diagnostics []Diagnostic
	// codes lists further diagnostic codes that must be reported.
	codes []string
}

type unconstrainedExampleValue struct {
	scope string
	value any
}

const unconstrainedExamplePath = "#/paths/~1value/get/responses/200/content/application~1json"

func TestUnconstrainedExampleMembership(t *testing.T) {
	preserved := map[string]any{"preserved": true}
	incompatible := func(path string) Diagnostic {
		return Diagnostic{Code: "examples", Path: unconstrainedExamplePath + path, Message: "example value is not compatible with its schema"}
	}
	nonNullable := func(path string) Diagnostic {
		return Diagnostic{Code: "examples", Path: unconstrainedExamplePath + path, Message: "null example requires a nullable schema"}
	}
	media := func(value any) unconstrainedExampleValue {
		return unconstrainedExampleValue{scope: "media", value: value}
	}
	schema := func(value any) unconstrainedExampleValue {
		return unconstrainedExampleValue{scope: "schema", value: value}
	}
	for _, tc := range []unconstrainedExampleCase{
		{
			// Retained actual-main reproduction: inputs/main/valid_any_object_rejected.json.
			name:   "schema any object",
			source: `{"openapi":"3.2.0","info":{"title":"Any annotation","version":"1"},"paths":{"/value":{"get":{"operationId":"value","responses":{"200":{"description":"value","content":{"application/json":{"schema":{"description":"Anything","examples":[{"preserved":true}]}}}}}}}}}`,
			want:   []unconstrainedExampleValue{schema(preserved)},
		},
		{name: "schema any scalar", media: `{"schema":{"example":"value"}}`, want: []unconstrainedExampleValue{schema("value")}},
		{name: "schema any array", media: `{"schema":{"examples":[[1,"x",null]]}}`, want: []unconstrainedExampleValue{schema([]any{1, "x", nil})}},
		{name: "schema any null", media: `{"schema":{"examples":[null]}}`, want: []unconstrainedExampleValue{schema(nil)}},
		{name: "media any object", media: `{"schema":{},"example":{"preserved":true}}`, want: []unconstrainedExampleValue{media(preserved)}},
		{name: "media any scalar", media: `{"schema":{},"example":1.5}`, want: []unconstrainedExampleValue{media(1.5)}},
		{name: "media any array", media: `{"schema":{},"example":[true,{"x":1}]}`, want: []unconstrainedExampleValue{media([]any{true, map[string]any{"x": 1}})}},
		{name: "media any null", media: `{"schema":{},"example":null}`, want: []unconstrainedExampleValue{media(nil)}},
		{name: "media true object", media: `{"schema":true,"example":{"preserved":true}}`, want: []unconstrainedExampleValue{media(preserved)}},
		{name: "media true null", media: `{"schema":true,"example":null}`, want: []unconstrainedExampleValue{media(nil)}},
		{
			name:  "media any named examples",
			media: `{"schema":{},"examples":{"object":{"value":{"preserved":true}},"empty":{"value":null}}}`,
			want:  []unconstrainedExampleValue{media(preserved), media(nil)},
		},
		{
			name:       "media any referenced example",
			media:      `{"schema":{},"examples":{"object":{"$ref":"#/components/examples/Thing"}}}`,
			components: `{"examples":{"Thing":{"value":{"preserved":true}}}}`,
			want:       []unconstrainedExampleValue{media(preserved)},
		},
		{
			name:       "named any media object",
			media:      `{"schema":{"$ref":"#/components/schemas/Anything"},"example":{"preserved":true}}`,
			components: `{"schemas":{"Anything":{}}}`,
			want:       []unconstrainedExampleValue{media(preserved)},
		},
		{
			name:       "named any media null",
			media:      `{"schema":{"$ref":"#/components/schemas/Anything"},"example":null}`,
			components: `{"schemas":{"Anything":{}}}`,
			want:       []unconstrainedExampleValue{media(nil)},
		},
		{
			name:       "named true media object",
			media:      `{"schema":{"$ref":"#/components/schemas/Anything"},"example":{"preserved":true}}`,
			components: `{"schemas":{"Anything":true}}`,
			want:       []unconstrainedExampleValue{media(preserved)},
		},
		{
			name:       "named true media null",
			media:      `{"schema":{"$ref":"#/components/schemas/Anything"},"example":null}`,
			components: `{"schemas":{"Anything":true}}`,
			want:       []unconstrainedExampleValue{media(nil)},
		},
		{
			name:       "named any schema examples",
			media:      `{"schema":{"$ref":"#/components/schemas/Anything"}}`,
			components: `{"schemas":{"Anything":{"description":"Anything","examples":[{"preserved":true},null]}}}`,
			want:       []unconstrainedExampleValue{schema(preserved), schema(nil)},
		},
		{name: "nullable any composition null", media: `{"schema":{"anyOf":[{},{"type":"null"}],"examples":[null,"value"]}}`, want: []unconstrainedExampleValue{schema(nil), schema("value")}},
		{
			name:  "nested object any property",
			media: `{"schema":{"type":"object","properties":{"anything":{}},"examples":[{"anything":{"x":1}},{"anything":null}]}}`,
			want:  []unconstrainedExampleValue{schema(map[string]any{"anything": map[string]any{"x": 1}}), schema(map[string]any{"anything": nil})},
		},
		{
			name:  "nested array any items",
			media: `{"schema":{"type":"array","items":{},"examples":[["x",{"a":1},null]]}}`,
			want:  []unconstrainedExampleValue{schema([]any{"x", map[string]any{"a": 1}, nil})},
		},
		{name: "oneOf any unique object", media: `{"schema":{"oneOf":[{},{"type":"string"}],"examples":[{"preserved":true}]}}`, want: []unconstrainedExampleValue{schema(preserved)}},
		{name: "oneOf any unique null", media: `{"schema":{"oneOf":[{},{"type":"string"}],"examples":[null]}}`, want: []unconstrainedExampleValue{schema(nil)}},
		{
			// Retained actual-main reproduction: inputs/main/ambiguous_oneof_string_accepted.json.
			name:        "oneOf any ambiguous string",
			source:      `{"openapi":"3.2.0","info":{"title":"Any annotation","version":"1"},"paths":{"/value":{"get":{"operationId":"value","responses":{"200":{"description":"value","content":{"application/json":{"schema":{"oneOf":[{}, {"type":"string"}],"examples":["value"]}}}}}}}}}`,
			diagnostics: []Diagnostic{incompatible("/schema/examples/0")},
		},
		{name: "oneOf any ambiguous media string", media: `{"schema":{"oneOf":[{},{"type":"string"}]},"example":"value"}`, diagnostics: []Diagnostic{incompatible("/example")}},
		{
			name:        "nested oneOf any",
			media:       `{"schema":{"type":"object","properties":{"v":{"oneOf":[{},{"type":"string"}]}},"examples":[{"v":{"a":1}},{"v":"x"}]}}`,
			want:        []unconstrainedExampleValue{schema(map[string]any{"v": map[string]any{"a": 1}})},
			diagnostics: []Diagnostic{incompatible("/schema/examples/1")},
		},
		{name: "oneOf typed unique", media: `{"schema":{"oneOf":[{"type":"string"},{"type":"integer"}],"examples":["x",1]}}`, want: []unconstrainedExampleValue{schema("x"), schema(1)}},
		{name: "oneOf typed no match", media: `{"schema":{"oneOf":[{"type":"string"},{"type":"integer"}],"examples":[true]}}`, diagnostics: []Diagnostic{incompatible("/schema/examples/0")}},
		{name: "oneOf typed ambiguous", media: `{"schema":{"oneOf":[{"type":"string"},{"type":"string","maxLength":5}],"examples":["x"]}}`, diagnostics: []Diagnostic{incompatible("/schema/examples/0")}},
		{name: "oneOf typed unique null", media: `{"schema":{"oneOf":[{"type":["string","null"]},{"type":"integer"}],"examples":[null]}}`, want: []unconstrainedExampleValue{schema(nil)}},
		{name: "oneOf typed ambiguous null", media: `{"schema":{"oneOf":[{"type":["string","null"]},{"type":["integer","null"]}],"examples":[null]}}`, diagnostics: []Diagnostic{nonNullable("/schema/examples/0")}},
		{name: "oneOf bare null", media: `{"schema":{"oneOf":[{"type":"string"},{"type":"null"}],"examples":[null]}}`, want: []unconstrainedExampleValue{schema(nil)}},
		{name: "typed string", media: `{"schema":{"type":"string","examples":["value"]}}`, want: []unconstrainedExampleValue{schema("value")}},
		{name: "nullable typed null", media: `{"schema":{"type":["string","null"],"examples":[null]}}`, want: []unconstrainedExampleValue{schema(nil)}},
		{name: "typed mismatch", media: `{"schema":{"type":"string","examples":[{"preserved":true}]}}`, diagnostics: []Diagnostic{incompatible("/schema/examples/0")}},
		{name: "typed null", media: `{"schema":{"type":"string","examples":[null]}}`, diagnostics: []Diagnostic{nonNullable("/schema/examples/0")}},
		{
			name:        "typeless constrained",
			media:       `{"schema":{"minLength":1,"examples":["value"]}}`,
			diagnostics: []Diagnostic{incompatible("/schema/examples/0")},
			codes:       []string{"schema"},
		},
		{
			name:        "false schema",
			media:       `{"schema":false,"example":"value"}`,
			diagnostics: []Diagnostic{incompatible("/example")},
			codes:       []string{"schema-boolean"},
		},
		{
			name: "non-JSON value",
			source: `openapi: 3.2.0
info: {title: Any annotation, version: "1"}
paths:
  /value:
    get:
      operationId: value
      responses:
        "200":
          description: value
          content:
            application/json:
              schema: {}
              example: {1: one}
`,
			diagnostics: []Diagnostic{{Code: "examples", Path: unconstrainedExamplePath + "/example", Message: "value of type map[interface {}]interface {} is not a supported Loom example"}},
		},
		{name: "any scalar default", media: `{"schema":{"default":"value"}}`, codes: []string{"schema-keyword"}},
		{name: "any object default", media: `{"schema":{"default":{"preserved":true}}}`, codes: []string{"schema-keyword"}},
		{name: "any null default", media: `{"schema":{"default":null}}`, codes: []string{"schema-keyword"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.source
			if source == "" {
				source = unconstrainedExampleDocument(tc.media, tc.components)
			}
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			if len(tc.diagnostics) == 0 && len(tc.codes) == 0 {
				require.Empty(t, diagnostics)
			}
			for _, diagnostic := range tc.diagnostics {
				require.Contains(t, diagnostics, diagnostic)
			}
			for _, code := range tc.codes {
				requireDiagnosticCode(t, diagnostics, code)
			}
			if len(tc.codes) > 0 {
				return
			}
			got := retainedUnconstrainedExamples(t, document)
			if len(tc.want) == 0 {
				require.Empty(t, got)
				return
			}
			require.Equal(t, tc.want, got)
		})
	}
}

func TestUnconstrainedExampleMatcher(t *testing.T) {
	accepted, rejected := true, false
	values := []any{map[string]any{"preserved": true}, []any{"x", nil}, "value", 1, 1.5, true, nil}
	graph := &exampleGraph{}
	for _, tc := range []struct {
		name   string
		schema *Schema
		want   bool
	}{
		{name: "unconstrained", schema: &Schema{Unconstrained: true}, want: true},
		{name: "boolean true", schema: &Schema{Boolean: &accepted, Unconstrained: true}, want: true},
		{name: "boolean false", schema: &Schema{Boolean: &rejected}, want: false},
		// A missing type alone is not the unconstrained domain.
		{name: "unclassified typeless", schema: &Schema{}, want: false},
		{name: "typeless constrained", schema: &Schema{MinLength: new(int64(1))}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, value := range values {
				if got, _ := graph.member(tc.schema, value, make(map[string]struct{})); got != tc.want {
					t.Errorf("value %#v: got %t, want %t", value, got, tc.want)
				}
			}
		})
	}
	union := &Schema{OneOf: []*Schema{{Unconstrained: true}, {Type: "string"}}}
	for _, tc := range []struct {
		value any
		want  bool
	}{
		{value: map[string]any{"preserved": true}, want: true},
		{value: nil, want: true},
		{value: 1, want: true},
		{value: "value", want: false},
	} {
		if got, _ := graph.member(union, tc.value, make(map[string]struct{})); got != tc.want {
			t.Errorf("oneOf value %#v: got %t, want %t", tc.value, got, tc.want)
		}
	}
}

func TestRenderRetainsUnconstrainedExamples(t *testing.T) {
	source := []byte(`openapi: 3.2.0
info: {title: Any examples, version: "1"}
paths:
  /schema:
    get:
      operationId: schemaExample
      responses:
        "200":
          description: value
          content:
            application/json:
              schema: {description: Anything, examples: [{preserved: true}, null]}
  /media:
    get:
      operationId: mediaExample
      responses:
        "200":
          description: value
          content:
            application/json:
              schema: true
              example: null
  /named:
    get:
      operationId: namedExample
      responses:
        "200":
          description: value
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Anything'}
              example: [1, {x: y}]
  /nested:
    get:
      operationId: nestedExample
      responses:
        "200":
          description: value
          content:
            application/json:
              schema:
                type: object
                properties:
                  anything: {}
                  items: {type: array, items: {}}
                examples: [{anything: {x: 1}, items: [x, null]}]
  /union:
    get:
      operationId: unionExample
      responses:
        "200":
          description: value
          content:
            application/json:
              schema:
                oneOf: [{}, {type: string}]
                examples: [{preserved: true}, null]
  /choice:
    get:
      operationId: choiceExample
      responses:
        "200":
          description: value
          content:
            application/json:
              schema:
                oneOf: [{type: [string, "null"]}, {type: integer}]
                examples: [null, 1]
components:
  schemas:
    Anything:
      examples: [{preserved: true}]
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	require.Empty(t, diagnostics)
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(rendered)
	for _, expected := range []string{
		`Value(Val{"preserved": true})`,
		`Value(Null())`,
		`Value([]any{1, Val{"x": "y"}})`,
		`Value(Val{"anything": Val{"x": 1}, "items": []any{"x", nil}})`,
	} {
		require.Contains(t, design, expected)
	}
	requireRenderedDesignEvaluates(t, rendered, 6)
	moduleDir := requireRenderedDesignGenerates(t, rendered)
	contract := readGeneratedOpenAPIContract(t, moduleDir)
	for _, tc := range []struct {
		path string
		want []any
	}{
		{path: "/schema", want: []any{map[string]any{"preserved": true}, nil}},
		{path: "/union", want: []any{map[string]any{"preserved": true}, nil}},
		{path: "/choice", want: []any{nil, float64(1)}},
	} {
		operation := operationFromImportedSpec(t, contract, tc.path, "get")
		content := operation["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)
		require.Equal(t, tc.want, content["schema"].(map[string]any)["examples"], tc.path)
	}
	requireIndependentRegeneration(t, moduleDir)
}

func unconstrainedExampleDocument(media, components string) string {
	document := fmt.Sprintf(`{"openapi":"3.2.0","info":{"title":"Any annotation","version":"1"},"paths":{"/value":{"get":{"operationId":"value","responses":{"200":{"description":"value","content":{"application/json":%s}}}}}}`, media)
	if components != "" {
		document += `,"components":` + components
	}
	return document + "}"
}

func retainedUnconstrainedExamples(t *testing.T, document *Document) []unconstrainedExampleValue {
	t.Helper()
	require.Len(t, document.Operations, 1)
	require.Len(t, document.Operations[0].Responses, 1)
	response := document.Operations[0].Responses[0].Response
	var examples []Example
	examples = append(examples, response.Examples...)
	if response.Schema != nil {
		examples = append(examples, response.Schema.Examples...)
	}
	for _, named := range document.Components.Schemas {
		examples = append(examples, named.Schema.Examples...)
	}
	values := make([]unconstrainedExampleValue, 0, len(examples))
	for _, example := range examples {
		values = append(values, unconstrainedExampleValue{scope: example.Scope, value: example.Value})
	}
	return values
}
