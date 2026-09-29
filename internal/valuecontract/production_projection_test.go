package valuecontract

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type productionProjected struct {
	outcome string
	wire    string
}

func checkProductionProjectionConformance(t *testing.T, executable string) {
	t.Helper()
	t.Run("external boundary", checkProductionProjectionExternal)
	cases := productionProjectionCases()
	require.NotEmpty(t, cases)
	for _, tc := range cases {
		for _, use := range []expr.ValuePlanUse{expr.ValuePlanDocumentation, expr.ValuePlanRuntime} {
			name := "documentation"
			if use == expr.ValuePlanRuntime {
				name = "runtime"
			}
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				actual, expected := productionProjectPair(t, executable, tc, use)
				require.Equal(t, expected, actual)
			})
		}
	}
}

func productionProjectionCases() []productionProjectionCase {
	scalar := func(typ expr.DataType) *expr.AttributeExpr { return &expr.AttributeExpr{Type: typ} }
	object := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "empty", Attribute: scalar(expr.String)},
		{Name: "explicit", Attribute: &expr.AttributeExpr{Type: expr.String, Nullable: true}},
		{Name: "hidden", Attribute: scalar(expr.String)},
	}}
	child := &expr.AttributeExpr{Type: &expr.Object{{Name: "required", Attribute: scalar(expr.String)}}, Validation: &expr.ValidationExpr{Required: []string{"required"}}}
	nested := &expr.AttributeExpr{Type: &expr.Object{{Name: "child", Attribute: child}}}
	union := func(untagged bool) *expr.AttributeExpr {
		return scalar(&expr.Union{Untagged: untagged, Values: []*expr.NamedAttributeExpr{{Name: "Bytes", Attribute: scalar(expr.Bytes)}, {Name: "Text", Attribute: scalar(expr.String)}}})
	}
	selected := &expr.AttributeExpr{Type: &expr.Object{{Name: "header", Attribute: scalar(expr.String)}, {Name: "body", Attribute: scalar(expr.String)}}, Validation: &expr.ValidationExpr{Required: []string{"header", "body"}}}
	arrayEnum := &expr.AttributeExpr{Type: &expr.Array{ElemType: scalar(expr.String)}, Validation: &expr.ValidationExpr{Values: []any{[]string{"one"}, []string{"two"}}}}
	mapEnum := &expr.AttributeExpr{Type: &expr.Map{KeyType: scalar(expr.String), ElemType: scalar(expr.String)}, Validation: &expr.ValidationExpr{Values: []any{map[string]string{"key": "one"}, map[string]string{"key": "two"}}}}
	alias := &expr.UserTypeExpr{TypeName: "Blob", AttributeExpr: scalar(expr.Bytes)}
	return []productionProjectionCase{
		{name: "bytes exactly once", attribute: scalar(expr.Bytes), raw: []byte("hi")},
		{name: "named bytes", attribute: scalar(alias), raw: []byte("hi")},
		{name: "string", attribute: scalar(expr.String), raw: "aGk="},
		{name: "boolean", attribute: scalar(expr.Boolean), raw: true},
		{name: "nullable null", attribute: &expr.AttributeExpr{Type: expr.String, Nullable: true}},
		{name: "tagged retained branch", attribute: union(false), raw: []byte("hi")},
		{name: "untagged ambiguity", attribute: union(true), raw: []byte("hi"), untagged: true},
		{name: "array", attribute: scalar(&expr.Array{ElemType: scalar(expr.String)}), raw: []string{"one", "two"}},
		{name: "empty array", attribute: scalar(&expr.Array{ElemType: scalar(expr.String)}), raw: []string{}},
		{name: "nil array", attribute: scalar(&expr.Array{ElemType: scalar(expr.String)}), raw: []string(nil)},
		{name: "map", attribute: scalar(&expr.Map{KeyType: scalar(expr.String), ElemType: scalar(expr.String)}), raw: map[string]string{"b": "two", "a": "one"}},
		{name: "Any bytes", attribute: scalar(expr.Any), raw: []byte("hi")},
		{name: "Any nil collection", attribute: scalar(expr.Any), raw: []string(nil)},
		{name: "presence and visibility", attribute: object, raw: map[string]any{"empty": "", "explicit": nil, "hidden": "secret"}, fields: map[string]expr.ValueFieldPresence{"empty": expr.ValueFieldOmitEmpty}, hidden: map[string]bool{"hidden": true}},
		{name: "presence retained empty", attribute: object, raw: map[string]any{"empty": "", "explicit": nil}, hidden: map[string]bool{"hidden": true}},
		{name: "nested partial omitted", attribute: nested, raw: map[string]any{"child": map[string]any{}}, fields: map[string]expr.ValueFieldPresence{"child": expr.ValueFieldOmitEmpty}},
		{name: "nested partial retained", attribute: nested, raw: map[string]any{"child": map[string]any{}}},
		{name: "selected body omits missing header", attribute: selected, raw: map[string]any{"body": "retained"}, selected: "body"},
		{name: "selected header remains incomplete", attribute: selected, raw: map[string]any{"body": "retained"}, selected: "header"},
		{name: "whole array enum", attribute: arrayEnum, raw: []string{"one"}},
		{name: "target array enum rejects", attribute: arrayEnum, raw: []string{"one"}, targetEnums: []any{[]string{"two"}}},
		{name: "whole map enum", attribute: mapEnum, raw: map[string]string{"key": "one"}},
		{name: "target map enum rejects", attribute: mapEnum, raw: map[string]string{"key": "one"}, targetEnums: []any{map[string]string{"key": "two"}}},
		{name: "bytes schema enum", attribute: &expr.AttributeExpr{Type: expr.Bytes, Validation: &expr.ValidationExpr{Values: []any{[]byte("hi")}}}, raw: []byte("hi")},
	}
}

func productionProjectPair(t *testing.T, executable string, tc productionProjectionCase, use expr.ValuePlanUse) (productionProjected, productionProjected) {
	t.Helper()
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(tc.attribute)
	require.NoError(t, err)
	input := newProductionInput(t)
	graph := newProductionGraph(t, input)
	root := graph.capture(tc.attribute, occurrence)
	graph.ranks()
	supplied := input.encode(tc.raw)
	targets := productionProjectionTargets(t, graph, tc)
	targets, targetRoot := productionProjectionRoot(t, graph, occurrence, targets, tc.selected)
	mode := "documentation"
	if use == expr.ValuePlanRuntime {
		mode = "runtime"
	}
	codecs := input.codecs()
	request := map[string]any{"declarations": graph.declarations, "root": root, "supplied": map[string]any{"source": map[string]any{"occurrence": root, "origin": 1, "role": "authoredExample"}, "value": supplied}, "codecs": codecs, "checks": []any{}, "projection": map[string]any{"root": targetRoot, "use": mode, "targets": targets}}
	command := referenceConstructor("evaluate", map[string]any{"request": request})
	actual := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
	if readings, missing := actual["numericRequests"]; missing {
		texts := referenceDecode[[]string](t, readings)
		rows := make([]any, 0, len(texts))
		for _, text := range texts {
			rows = append(rows, map[string]any{"text": text, "schema": productionSchemaNumber(text)})
		}
		codecs["numberReadings"] = rows
		actual = referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
	}
	require.Equal(t, "true", string(actual["declarationsValid"]))
	require.Equal(t, "true", string(actual["targetsValid"]))
	source := context.SupplyValue(expr.ValueInput{Raw: tc.raw})
	resolved := context.Resolve(occurrence, source, expr.ValueRoleExample)
	require.Equal(t, productionCanonical(t, jsontext.Value(productionJSON(t, graph.output(input, resolved, tc.raw)))), productionCanonical(t, actual["resolved"]))
	require.Contains(t, actual, "projected", "codec obligations must all be discharged: %s", productionJSON(t, actual))
	plan, err := context.NewValuePlan(occurrence, productionProjectionPlan(tc, use))
	require.NoError(t, err)
	projected := context.ProjectJSON(resolved, plan)
	output := productionProjected{outcome: map[expr.ProjectionOutcome]string{expr.ProjectionEmitted: "emitted", expr.ProjectionIncomplete: "incomplete", expr.ProjectionUnrepresentable: "unrepresentable", expr.ProjectionUnsupported: "unsupported", expr.ProjectionInvalidPlan: "invalidPlan"}[projected.Outcome()]}
	require.NotEmpty(t, output.outcome)
	if wire, present := projected.JSON(); present {
		output.wire = productionCanonical(t, wire)
	}
	expected := productionProjected{}
	if actual["projected"].Kind() == '"' {
		expected.outcome = referenceDecode[string](t, actual["projected"])
	} else {
		emitted := referenceDecode[map[string]map[string]jsontext.Value](t, actual["projected"])
		require.Contains(t, emitted, "emitted")
		expected = productionProjected{outcome: "emitted", wire: productionCanonical(t, productionReferenceWire(t, emitted["emitted"]["value"]))}
	}
	return output, expected
}
