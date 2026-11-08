package openapiimport

import (
	"fmt"
	"testing"

	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"
)

type exampleMembershipCase struct {
	name string
	// media is the application/json media object of GET /value.
	media string
	// components is the optional components object.
	components string
	// source replaces the generated document when set.
	source string
	// rejected lists the exact examples diagnostics. Accepted cases require
	// no diagnostics at all.
	rejected []Diagnostic
	// codes lists further diagnostic codes the case must report.
	codes []string
	// kept lists the retained example values in component, then operation
	// order.
	kept []any
}

const membershipMediaPath = "#/paths/~1value/get/responses/200/content/application~1json"

// TestExampleMembershipFinalGraph checks that every retained example is a
// member of its final normalized owner: local assertions, the reference
// target and every retained base, resolved over the complete component graph
// regardless of declaration order or constructor promotion.
func TestExampleMembershipFinalGraph(t *testing.T) {
	incompatible := func(path string) Diagnostic {
		return Diagnostic{Code: "examples", Path: path, Message: "example value is not compatible with its schema"}
	}
	nonNullable := func(path string) Diagnostic {
		return Diagnostic{Code: "examples", Path: path, Message: "null example requires a nullable schema"}
	}
	base := `{"schemas":{"Base":{"type":"string","maxLength":6}}}`
	pet := `{"schemas":{"Pet":{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}}}}`
	wrapped := `{"schema":{"$ref":"#/components/schemas/Wrapped"}}`
	closable := `{"schemas":{"X":{"type":"object","properties":{"x":{"type":"integer"}},"required":["x"]},"S":{"type":"object","properties":{"y":{"type":"string"}},"required":["y"]},"Pet":{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}},"Child":{"type":"object","properties":{"a":{"type":"integer"}}},"Holder":{"type":"object","properties":{"child":{"$ref":"#/components/schemas/Child"}},"required":["child"]},"Closed":{"oneOf":[{"$ref":"#/components/schemas/X"}],"unevaluatedProperties":false}}}`
	closed := func(branches, examples string) string {
		return `{"schema":{"oneOf":` + branches + `,"unevaluatedProperties":false},"examples":` + examples + `}`
	}
	named := func(name string) string {
		return membershipMediaPath + "/examples/" + name + "/value"
	}
	for _, tc := range []exampleMembershipCase{
		{
			name:       "media reference local invalid",
			media:      `{"schema":{"$ref":"#/components/schemas/Base","minLength":5},"example":"x"}`,
			components: base,
			rejected:   []Diagnostic{incompatible(membershipMediaPath + "/example")},
		},
		{
			name:       "media reference target invalid",
			media:      `{"schema":{"$ref":"#/components/schemas/Base","minLength":5},"example":"toolong"}`,
			components: base,
			rejected:   []Diagnostic{incompatible(membershipMediaPath + "/example")},
		},
		{
			name:       "media reference valid",
			media:      `{"schema":{"$ref":"#/components/schemas/Base","minLength":5},"example":"valid"}`,
			components: base,
			kept:       []any{"valid"},
		},
		{
			name:       "referenced media example local invalid",
			media:      `{"schema":{"$ref":"#/components/schemas/Base","minLength":5},"examples":{"short":{"$ref":"#/components/examples/Short"}}}`,
			components: `{"schemas":{"Base":{"type":"string"}},"examples":{"Short":{"value":"x"}}}`,
			rejected: []Diagnostic{{
				Code: "examples", Path: membershipMediaPath + "/examples/short",
				Message: "referenced example value is not compatible with its schema",
			}},
		},
		{
			name:       "schema example beside reference target invalid",
			media:      `{"schema":{"$ref":"#/components/schemas/Name"}}`,
			components: `{"schemas":{"Base":{"type":"string","maxLength":3},"Name":{"$ref":"#/components/schemas/Base","examples":["toolong","ok"]}}}`,
			rejected:   []Diagnostic{incompatible("#/components/schemas/Name/examples/0")},
			kept:       []any{"ok"},
		},
		{
			name:       "schema example beside allOf reference local invalid",
			media:      `{"schema":{"allOf":[{"$ref":"#/components/schemas/Count"}],"minimum":5,"example":1}}`,
			components: `{"schemas":{"Count":{"type":"integer"}}}`,
			rejected:   []Diagnostic{incompatible(membershipMediaPath + "/schema/example")},
		},
		{
			name:       "reverse declaration order invalid",
			media:      `{"schema":{"$ref":"#/components/schemas/Holder"}}`,
			components: `{"schemas":{"Holder":{"type":"object","properties":{"code":{"$ref":"#/components/schemas/Code"}},"examples":[{"code":"x"},{"code":"valid"}]},"Code":{"type":"string","minLength":5}}}`,
			rejected:   []Diagnostic{incompatible("#/components/schemas/Holder/examples/0")},
			kept:       []any{map[string]any{"code": "valid"}},
		},
		{
			name:       "self recursive finite values",
			media:      `{"schema":{"$ref":"#/components/schemas/Node"}}`,
			components: `{"schemas":{"Node":{"type":"object","properties":{"value":{"type":"string"},"next":{"$ref":"#/components/schemas/Node"}},"examples":[{"next":{"next":{"value":"leaf"}}},{"next":{"next":{"value":1}}}]}}}`,
			rejected:   []Diagnostic{incompatible("#/components/schemas/Node/examples/1")},
			kept:       []any{map[string]any{"next": map[string]any{"next": map[string]any{"value": "leaf"}}}},
		},
		{
			name:       "base required property missing",
			media:      `{"schema":{"$ref":"#/components/schemas/Pet","properties":{"bark":{"type":"boolean"}}},"example":{"bark":true}}`,
			components: pet,
			rejected:   []Diagnostic{incompatible(membershipMediaPath + "/example")},
		},
		{
			name:       "base property type invalid",
			media:      `{"schema":{"$ref":"#/components/schemas/Pet","properties":{"bark":{"type":"boolean"}}},"example":{"name":1,"bark":true}}`,
			components: pet,
			rejected:   []Diagnostic{incompatible(membershipMediaPath + "/example")},
		},
		{
			name:       "base and local properties valid",
			media:      `{"schema":{"$ref":"#/components/schemas/Pet","properties":{"bark":{"type":"boolean"}}},"example":{"name":"rex","bark":true}}`,
			components: pet,
			kept:       []any{map[string]any{"name": "rex", "bark": true}},
		},
		{
			name:  "array item bounds",
			media: `{"schema":{"type":"array","items":{"type":"string"},"minItems":2,"maxItems":3},"examples":{"short":{"value":["a"]},"long":{"value":["a","b","c","d"]},"fit":{"value":["a","b"]}}}`,
			rejected: []Diagnostic{
				incompatible(membershipMediaPath + "/examples/long/value"),
				incompatible(membershipMediaPath + "/examples/short/value"),
			},
			kept: []any{[]any{"a", "b"}},
		},
		{
			name:     "typed additional properties",
			media:    `{"schema":{"type":"object","additionalProperties":{"type":"integer"}},"examples":{"text":{"value":{"a":"x"}},"number":{"value":{"a":1}}}}`,
			rejected: []Diagnostic{incompatible(membershipMediaPath + "/examples/text/value")},
			kept:     []any{map[string]any{"a": 1}},
		},
		{
			name:     "nullable enum",
			media:    `{"schema":{"type":["string","null"],"enum":["a","b",null]},"examples":{"empty":{"value":null},"other":{"value":"c"},"listed":{"value":"a"}}}`,
			rejected: []Diagnostic{incompatible(membershipMediaPath + "/examples/other/value")},
			kept:     []any{nil, "a"},
		},
		{
			name:       "nullable reference wrapper",
			media:      `{"schema":{"anyOf":[{"$ref":"#/components/schemas/Short"},{"type":"null"}]},"examples":{"empty":{"value":null},"tiny":{"value":"x"},"fit":{"value":"xy"}}}`,
			components: `{"schemas":{"Short":{"type":"string","minLength":2}}}`,
			rejected:   []Diagnostic{incompatible(membershipMediaPath + "/examples/tiny/value")},
			kept:       []any{nil, "xy"},
		},
		{
			name:     "oneOf overlap",
			media:    `{"schema":{"oneOf":[{"type":"string"},{"type":"string","maxLength":3}]},"examples":{"both":{"value":"ab"},"one":{"value":"abcdef"}}}`,
			rejected: []Diagnostic{incompatible(membershipMediaPath + "/examples/both/value")},
			kept:     []any{"abcdef"},
		},
		{
			name:     "oneOf null overlap",
			media:    `{"schema":{"oneOf":[{"type":["string","null"]},{"type":["integer","null"]}]},"examples":{"empty":{"value":null},"text":{"value":"a"}}}`,
			rejected: []Diagnostic{nonNullable(membershipMediaPath + "/examples/empty/value")},
			kept:     []any{"a"},
		},
		{
			name:       "nullable anyOf inner null",
			media:      wrapped,
			components: `{"schemas":{"Wrapped":{"anyOf":[{"type":"string","example":null},{"type":"null"}]}}}`,
			rejected:   []Diagnostic{nonNullable("#/components/schemas/Wrapped/anyOf/0/example")},
		},
		{
			name:       "nullable anyOf outer null",
			media:      wrapped,
			components: `{"schemas":{"Wrapped":{"anyOf":[{"type":"string"},{"type":"null"}],"example":null}}}`,
			kept:       []any{nil},
		},
		{
			name:       "singleton oneOf inner null",
			media:      wrapped,
			components: `{"schemas":{"Wrapped":{"oneOf":[{"type":"string","example":null},{"type":"null"}]}}}`,
			rejected:   []Diagnostic{nonNullable("#/components/schemas/Wrapped/oneOf/0/example")},
		},
		{
			name:       "singleton oneOf outer null",
			media:      wrapped,
			components: `{"schemas":{"Wrapped":{"oneOf":[{"type":"string"},{"type":"null"}],"example":null}}}`,
			kept:       []any{nil},
		},
		{
			name:       "nullable anyOf inner valid",
			media:      wrapped,
			components: `{"schemas":{"Wrapped":{"anyOf":[{"type":"string","example":"ok"},{"type":"null"}]}}}`,
			kept:       []any{"ok"},
		},
		{
			name:       "nullable anyOf inner null beside outer value",
			media:      wrapped,
			components: `{"schemas":{"Wrapped":{"anyOf":[{"type":"string","example":null},{"type":"null"}],"example":"ok"}}}`,
			rejected:   []Diagnostic{nonNullable("#/components/schemas/Wrapped/anyOf/0/example")},
			kept:       []any{"ok"},
		},
		{
			name:       "nullable anyOf inner reference overlay",
			media:      wrapped,
			components: `{"schemas":{"Code":{"type":"string","maxLength":4},"Wrapped":{"anyOf":[{"$ref":"#/components/schemas/Code","minLength":2,"examples":[null,"x","toolong","ab"]},{"type":"null"}]}}}`,
			rejected: []Diagnostic{
				nonNullable("#/components/schemas/Wrapped/anyOf/0/examples/0"),
				incompatible("#/components/schemas/Wrapped/anyOf/0/examples/1"),
				incompatible("#/components/schemas/Wrapped/anyOf/0/examples/2"),
			},
			kept: []any{"ab"},
		},
		{
			name:       "allOf inline branch with inherited base",
			media:      `{"schema":{"$ref":"#/components/schemas/Dog"}}`,
			components: `{"schemas":{"Pet":{"type":"object","required":["name"],"properties":{"name":{"type":"string"}}},"Dog":{"allOf":[{"$ref":"#/components/schemas/Pet"},{"type":"object","properties":{"bark":{"type":"boolean"}},"examples":[{"bark":true},{"name":"rex","bark":true}]}]}}}`,
			rejected:   []Diagnostic{incompatible("#/components/schemas/Dog/allOf/1/examples/0")},
			kept:       []any{map[string]any{"name": "rex", "bark": true}},
		},
		{
			name:       "closed union unevaluated member",
			media:      `{"schema":{"oneOf":[{"$ref":"#/components/schemas/X"}],"unevaluatedProperties":false},"example":{"x":1,"z":2}}`,
			components: closable,
			rejected:   []Diagnostic{incompatible(membershipMediaPath + "/example")},
		},
		{
			name:       "closed union evaluated member",
			media:      `{"schema":{"oneOf":[{"$ref":"#/components/schemas/X"}],"unevaluatedProperties":false},"example":{"x":1}}`,
			components: closable,
			kept:       []any{map[string]any{"x": 1}},
		},
		{
			name:       "open union unevaluated member",
			media:      `{"schema":{"oneOf":[{"$ref":"#/components/schemas/X"}]},"example":{"x":1,"z":2}}`,
			components: closable,
			kept:       []any{map[string]any{"x": 1, "z": 2}},
		},
		{
			name:       "closed union typed map",
			media:      `{"schema":{"oneOf":[{"type":"object","additionalProperties":{"type":"integer"}}],"unevaluatedProperties":false},"example":{"x":1,"z":2}}`,
			components: closable,
			kept:       []any{map[string]any{"x": 1, "z": 2}},
		},
		{
			name:       "closed union typed map invalid value",
			media:      closed(`[{"type":"object","additionalProperties":{"type":"integer"}}]`, `{"text":{"value":{"z":"a"}},"fit":{"value":{"z":2}}}`),
			components: closable,
			rejected:   []Diagnostic{incompatible(named("text"))},
			kept:       []any{map[string]any{"z": 2}},
		},
		{
			name:       "closed union explicit additional properties",
			media:      closed(`[{"type":"object","properties":{"x":{"type":"integer"}},"additionalProperties":true}]`, `{"open":{"value":{"x":1,"z":"a"}}}`),
			components: closable,
			kept:       []any{map[string]any{"x": 1, "z": "a"}},
		},
		{
			name:       "closed union unmatched branch member",
			media:      closed(`[{"$ref":"#/components/schemas/X"},{"$ref":"#/components/schemas/S"}]`, `{"borrowed":{"value":{"x":1,"y":2}},"own":{"value":{"x":1}}}`),
			components: closable,
			rejected:   []Diagnostic{incompatible(named("borrowed"))},
			kept:       []any{map[string]any{"x": 1}},
		},
		{
			name:       "closed union reference",
			media:      `{"schema":{"$ref":"#/components/schemas/Closed"},"examples":{"extra":{"value":{"x":1,"z":2}},"fit":{"value":{"x":1}}}}`,
			components: closable,
			rejected:   []Diagnostic{incompatible(named("extra"))},
			kept:       []any{map[string]any{"x": 1}},
		},
		{
			name:       "closed union component example",
			media:      `{"schema":{"$ref":"#/components/schemas/Closed"}}`,
			components: `{"schemas":{"X":{"type":"object","properties":{"x":{"type":"integer"}},"required":["x"]},"Closed":{"oneOf":[{"$ref":"#/components/schemas/X"}],"unevaluatedProperties":false,"examples":[{"x":1,"z":2},{"x":1}]}}}`,
			rejected:   []Diagnostic{incompatible("#/components/schemas/Closed/examples/0")},
			kept:       []any{map[string]any{"x": 1}},
		},
		{
			name:       "closed union nested branch",
			media:      closed(`[{"oneOf":[{"$ref":"#/components/schemas/X"},{"type":"string"}]},{"type":"boolean"}]`, `{"extra":{"value":{"x":1,"z":2}},"fit":{"value":{"x":1}}}`),
			components: closable,
			rejected:   []Diagnostic{incompatible(named("extra"))},
			kept:       []any{map[string]any{"x": 1}},
		},
		{
			name:       "closed union composed branch",
			media:      closed(`[{"$ref":"#/components/schemas/Pet","properties":{"bark":{"type":"boolean"}}}]`, `{"extra":{"value":{"name":"rex","bark":true,"z":1}},"fit":{"value":{"name":"rex","bark":true}}}`),
			components: closable,
			rejected:   []Diagnostic{incompatible(named("extra"))},
			kept:       []any{map[string]any{"name": "rex", "bark": true}},
		},
		{
			name:       "closed union unconstrained branch",
			media:      closed(`[{},{"type":"string"}]`, `{"member":{"value":{"z":1}},"empty":{"value":{}}}`),
			components: closable,
			rejected:   []Diagnostic{incompatible(named("member"))},
			kept:       []any{map[string]any{}},
		},
		{
			name:       "nullable closed union",
			media:      `{"schema":{"anyOf":[{"oneOf":[{"$ref":"#/components/schemas/X"}],"unevaluatedProperties":false},{"type":"null"}]},"examples":{"extra":{"value":{"x":1,"z":2}},"empty":{"value":null}}}`,
			components: closable,
			rejected:   []Diagnostic{incompatible(named("extra"))},
			kept:       []any{nil},
		},
		{
			name:       "closed union child position",
			media:      closed(`[{"$ref":"#/components/schemas/Holder"}]`, `{"child":{"value":{"child":{"a":1,"b":2}}},"extra":{"value":{"child":{"a":1},"z":1}}}`),
			components: closable,
			rejected:   []Diagnostic{incompatible(named("extra"))},
			kept:       []any{map[string]any{"child": map[string]any{"a": 1, "b": 2}}},
		},
		{
			name:  "promoted constructor",
			media: `{"schema":{"type":"object","required":["id"],"properties":{"id":{"type":"integer"}}},"examples":{"missing":{"value":{}},"fit":{"value":{"id":1}}}}`,
			rejected: []Diagnostic{
				incompatible(membershipMediaPath + "/examples/missing/value"),
			},
			kept: []any{map[string]any{"id": 1}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := tc.source
			if source == "" {
				source = membershipDocument(tc.media, tc.components)
			}
			document, diagnostics, err := Analyze([]byte(source))
			require.NoError(t, err)
			var rejected Diagnostics
			codes := make(map[string]bool)
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == "examples" {
					rejected = append(rejected, diagnostic)
					continue
				}
				codes[diagnostic.Code] = true
			}
			if len(tc.rejected) == 0 {
				require.Empty(t, rejected, diagnostics.Error())
			} else {
				require.Equal(t, tc.rejected, []Diagnostic(rejected), diagnostics.Error())
			}
			for _, code := range tc.codes {
				require.True(t, codes[code], "missing %s in %s", code, diagnostics.Error())
				delete(codes, code)
			}
			require.Empty(t, codes, diagnostics.Error())
			require.Equal(t, tc.kept, retainedExampleValues(document))
		})
	}
}

// TestExampleMembershipMatcher pins the shared matcher on final graphs that a
// parsed document cannot always reach: libopenapi rejects a dangling schema
// reference before analysis, so a missing target is checked here.
func TestExampleMembershipMatcher(t *testing.T) {
	ref := func(name string) *Schema {
		return &Schema{Ref: "#/components/schemas/" + name}
	}
	short := &Schema{Type: "string", MaxLength: new(int64(3))}
	nullableShort := &Schema{Type: "string", Nullable: true, MaxLength: new(int64(3))}
	node := &Schema{Type: "object", Properties: []NamedProperty{{Name: "next", Schema: ref("Node")}, {Name: "value", Schema: &Schema{Type: "string"}}}}
	graph := &exampleGraph{schemas: map[string]*Schema{
		"Short": short, "NullableShort": nullableShort, "Node": node,
		"Loop": ref("Alias"), "Alias": ref("Loop"),
		"Pet":  {Type: "object", Required: []string{"name"}, Properties: []NamedProperty{{Name: "name", Schema: &Schema{Type: "string"}}}},
		"Tags": {Type: "array", Items: &Schema{Type: "string", Nullable: true}},
	}}
	for _, tc := range []struct {
		name   string
		schema *Schema
		value  any
		want   bool
	}{
		{name: "missing target", schema: ref("Missing"), value: "x", want: false},
		{name: "missing target null", schema: ref("Missing"), value: nil, want: false},
		{name: "foreign component target", schema: &Schema{Ref: "#/components/parameters/Short"}, value: "x", want: false},
		{name: "alias cycle", schema: ref("Loop"), value: "x", want: false},
		{name: "recursive finite valid", schema: ref("Node"), value: map[string]any{"next": map[string]any{"value": "leaf"}}, want: true},
		{name: "recursive finite invalid leaf", schema: ref("Node"), value: map[string]any{"next": map[string]any{"value": 1}}, want: false},
		{name: "reference local and target", schema: &Schema{Ref: "#/components/schemas/Short", Type: "string", MinLength: new(int64(2))}, value: "ab", want: true},
		{name: "reference local violated", schema: &Schema{Ref: "#/components/schemas/Short", Type: "string", MinLength: new(int64(2))}, value: "a", want: false},
		{name: "reference target violated", schema: &Schema{Ref: "#/components/schemas/Short", Type: "string", MinLength: new(int64(2))}, value: "abcd", want: false},
		{name: "plain reference to nullable target null", schema: ref("NullableShort"), value: nil, want: true},
		{name: "plain reference to nonnullable target null", schema: ref("Short"), value: nil, want: false},
		{name: "nullable wrapper over nonnullable target null", schema: &Schema{Ref: "#/components/schemas/Short", Nullable: true}, value: nil, want: true},
		{name: "nullable wrapper still conjoins non-null", schema: &Schema{Ref: "#/components/schemas/Short", Nullable: true}, value: "abcd", want: false},
		{name: "nullable enum null", schema: &Schema{Type: "string", Nullable: true, Enum: []any{"a"}}, value: nil, want: true},
		{name: "enum excludes null", schema: &Schema{Type: "string", Enum: []any{"a"}}, value: nil, want: false},
		{name: "reference enum excludes null", schema: &Schema{Ref: "#/components/schemas/NullableShort", Enum: []any{"a"}}, value: nil, want: false},
		{name: "oneOf null exactly one", schema: &Schema{OneOf: []*Schema{nullableShort, {Type: "integer"}}}, value: nil, want: true},
		{name: "oneOf null overlap", schema: &Schema{OneOf: []*Schema{nullableShort, {Type: "integer", Nullable: true}}}, value: nil, want: false},
		{name: "base required", schema: &Schema{Type: "object", Bases: []*Schema{ref("Pet")}}, value: map[string]any{}, want: false},
		{name: "base satisfied", schema: &Schema{Type: "object", Bases: []*Schema{ref("Pet")}}, value: map[string]any{"name": "rex"}, want: true},
		{name: "base null", schema: &Schema{Type: "object", Bases: []*Schema{ref("Pet")}}, value: nil, want: false},
		{name: "typed additional property", schema: &Schema{Type: "object", AdditionalProperties: &AdditionalProperties{Schema: &Schema{Type: "integer"}}}, value: map[string]any{"a": "x"}, want: false},
		{name: "array minimum items", schema: &Schema{Type: "array", Items: &Schema{Type: "string"}, MinItems: new(int64(1))}, value: []any{}, want: false},
		{name: "array maximum items", schema: &Schema{Type: "array", Items: &Schema{Type: "string"}, MaxItems: new(int64(1))}, value: []any{"a", "b"}, want: false},
		{name: "reference array items owned by target", schema: &Schema{Ref: "#/components/schemas/Tags", Type: "array", MaxItems: new(int64(2))}, value: []any{nil}, want: true},
		{name: "reference array local bound", schema: &Schema{Ref: "#/components/schemas/Tags", Type: "array", MaxItems: new(int64(2))}, value: []any{"a", "b", "c"}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := graph.member(tc.schema, tc.value, make(map[string]struct{})); got != tc.want {
				t.Errorf("got %t, want %t", got, tc.want)
			}
		})
	}
}

// TestExampleMembershipAuthoredOwner keeps each example bound to the domain it
// was authored on when normalization copies it into another schema, and also
// requires membership in the schema that renders it.
func TestExampleMembershipAuthoredOwner(t *testing.T) {
	branch := &Schema{Type: "string"}
	wrapper := &Schema{Type: "string", Nullable: true}
	sample := func(value any, owner *Schema, path string) Example {
		return Example{Value: value, source: exampleSource{owner: owner, path: path}}
	}
	graph := &exampleGraph{}
	kept := graph.retain([]Example{
		sample(nil, branch, "inner-null"),
		sample("ok", branch, "inner-text"),
		sample(nil, wrapper, "outer-null"),
		sample(nil, &Schema{Ref: "#/components/schemas/Missing", Nullable: false}, "missing-owner"),
	}, wrapper)
	require.Equal(t, []any{"ok", nil}, []any{kept[0].Value, kept[1].Value})
	require.Len(t, kept, 2)
	require.Equal(t, "outer-null", kept[1].source.path)
	narrowed := graph.retain([]Example{sample(nil, wrapper, "narrowed")}, branch)
	require.Empty(t, narrowed)
	paths := make([]string, 0, len(graph.diagnostics))
	for _, diagnostic := range graph.diagnostics {
		paths = append(paths, diagnostic.Path)
	}
	require.Equal(t, []string{"inner-null", "missing-owner", "narrowed"}, paths)
}

// TestExampleMembershipClosureProfiles holds the shared matcher to the existing
// closure profiles, whose expected membership the source oracle establishes,
// on the analyzed normalized graph.
func TestExampleMembershipClosureProfiles(t *testing.T) {
	profiles := append(singletonClosureProfiles(), oneOfClosureProfiles()...)
	document, _, err := Analyze(oneOfClosureDocument(t, profiles...))
	require.NoError(t, err)
	graph := &exampleGraph{schemas: make(map[string]*Schema, len(document.Components.Schemas))}
	for _, named := range document.Components.Schemas {
		graph.schemas[named.Name] = named.Schema
	}
	responses := make(map[string]*Schema, len(document.Operations))
	for _, operation := range document.Operations {
		responses[operation.Path] = operation.Responses[0].Response.Schema
	}
	for _, profile := range profiles {
		for _, value := range profile.values {
			t.Run(profile.name+" "+value.wire, func(t *testing.T) {
				var instance any
				require.NoError(t, yaml.Unmarshal([]byte(value.wire), &instance))
				matched, _ := graph.member(responses["/"+profile.name], instance, make(map[string]struct{}))
				require.Equal(t, value.valid, matched)
			})
		}
	}
}

// TestExampleMembershipSelectionProvenance reports a rejected component
// example at its authored path and only while that component is selected.
func TestExampleMembershipSelectionProvenance(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Selection, version: '1'}
paths:
  /selected:
    get:
      operationId: selected
      responses:
        '200':
          description: selected
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Selected'}
  /other:
    get:
      operationId: other
      responses:
        '200':
          description: other
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Other'}
components:
  schemas:
    Selected: {$ref: '#/components/schemas/Code', examples: [x]}
    Other: {$ref: '#/components/schemas/Code', examples: [y]}
    Code: {type: string, minLength: 3}
`)
	_, diagnostics, _, err := AnalyzeSelected(Source{Bytes: source}, Selection{Paths: []string{"/selected"}})
	require.NoError(t, err)
	require.Equal(t, Diagnostics{{
		Code: "examples", Path: "#/components/schemas/Selected/examples/0", Message: "example value is not compatible with its schema",
	}}, diagnostics)
}

// TestExampleMembershipRenderedDesignEvaluates renders the final-graph
// members of every changed contract into one design and evaluates it. The
// nullable reference wrapper keeps only non-null samples here: its null
// membership is covered above, and its body rendering is a separate owner.
func TestExampleMembershipRenderedDesignEvaluates(t *testing.T) {
	source := []byte(`openapi: 3.1.1
info: {title: Membership, version: '1'}
paths:
  /reference:
    get:
      operationId: reference
      responses:
        '200':
          description: reference
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Code', minLength: 2}
              examples: {fit: {value: ab}, short: {value: a}}
  /pet:
    get:
      operationId: pet
      responses:
        '200':
          description: pet
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet', properties: {bark: {type: boolean}}}
              examples: {fit: {value: {name: rex, bark: true}}, missing: {value: {bark: true}}}
  /holder:
    get:
      operationId: holder
      responses:
        '200':
          description: holder
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Holder'}
  /map:
    get:
      operationId: map
      responses:
        '200':
          description: map
          content:
            application/json:
              schema: {type: object, additionalProperties: {type: integer}}
              examples: {fit: {value: {a: 1}}, text: {value: {a: x}}}
  /list:
    get:
      operationId: list
      responses:
        '200':
          description: list
          content:
            application/json:
              schema: {type: array, items: {type: string}, minItems: 1, maxItems: 2}
              examples: {fit: {value: [a]}, long: {value: [a, b, c]}}
  /maybe:
    get:
      operationId: maybe
      responses:
        '200':
          description: maybe
          content:
            application/json:
              schema: {anyOf: [{$ref: '#/components/schemas/Code'}, {type: 'null'}]}
              examples: {fit: {value: ab}, long: {value: toolong}}
components:
  schemas:
    Holder:
      type: object
      properties:
        code: {$ref: '#/components/schemas/Code'}
        next: {$ref: '#/components/schemas/Holder'}
      examples: [{next: {code: ab}}, {next: {code: toolong}}]
    Code: {type: string, maxLength: 4}
    Pet:
      type: object
      required: [name]
      properties:
        name: {type: string}
`)
	document, diagnostics, err := Analyze(source)
	require.NoError(t, err)
	paths := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		require.Equal(t, "examples", diagnostic.Code, diagnostics.Error())
		paths = append(paths, diagnostic.Path)
	}
	media := func(path, name string) string {
		return "#/paths/~1" + path + "/get/responses/200/content/application~1json/examples/" + name + "/value"
	}
	require.Equal(t, []string{
		"#/components/schemas/Holder/examples/1",
		media("list", "long"), media("map", "text"), media("maybe", "long"),
		media("pet", "missing"), media("reference", "short"),
	}, paths)
	require.ElementsMatch(t, []any{
		map[string]any{"next": map[string]any{"code": "ab"}},
		"ab", map[string]any{"bark": true, "name": "rex"}, map[string]any{"a": 1}, []any{"a"}, "ab",
	}, retainedExampleValues(document))
	rendered, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	requireRenderedDesignEvaluates(t, rendered, 6)
}

func membershipDocument(media, components string) string {
	document := fmt.Sprintf(`{"openapi":"3.1.1","info":{"title":"Membership","version":"1"},"paths":{"/value":{"get":{"operationId":"value","responses":{"200":{"description":"value","content":{"application/json":%s}}}}}}`, media)
	if components != "" {
		document += `,"components":` + components
	}
	return document + "}"
}

// retainedExampleValues lists retained example values from every final owner:
// component schema graphs, then operation media and schema graphs.
func retainedExampleValues(document *Document) []any {
	var values []any
	seen := make(map[*Schema]bool)
	var visit func(*Schema)
	visit = func(schema *Schema) {
		if schema == nil || seen[schema] {
			return
		}
		seen[schema] = true
		for _, example := range schema.Examples {
			values = append(values, example.Value)
		}
		for _, property := range schema.Properties {
			visit(property.Schema)
		}
		visit(schema.Items)
		if schema.AdditionalProperties != nil {
			visit(schema.AdditionalProperties.Schema)
		}
		for _, branch := range schema.OneOf {
			visit(branch)
		}
		for _, base := range schema.Bases {
			visit(base)
		}
	}
	for _, named := range document.Components.Schemas {
		visit(named.Schema)
	}
	for _, operation := range document.Operations {
		for _, response := range operation.Responses {
			for _, example := range response.Response.Examples {
				values = append(values, example.Value)
			}
			visit(response.Response.Schema)
		}
	}
	return values
}
