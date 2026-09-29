package valuecontract

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type productionChoiceRandom struct {
	expr.Randomizer
	calls []string
}

func (r *productionChoiceRandom) Int() int {
	r.calls = append(r.calls, "branch")
	return 1
}

func (r *productionChoiceRandom) String() string {
	r.calls = append(r.calls, "payload")
	return "retained"
}

func checkProductionSynthesisConformance(t *testing.T, executable string) {
	t.Helper()
	for _, named := range []bool{false, true} {
		name := "direct"
		if named {
			name = "named"
		}
		t.Run(name, func(t *testing.T) {
			union := &expr.Union{Values: []*expr.NamedAttributeExpr{
				{Name: "Left", Attribute: &expr.AttributeExpr{Type: expr.String}},
				{Name: "Right", Attribute: &expr.AttributeExpr{Type: expr.String}},
			}}
			attribute := &expr.AttributeExpr{Type: union}
			if named {
				attribute = &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Named", AttributeExpr: attribute}}
			}
			context := expr.NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			input := newProductionInput(t)
			graph := newProductionGraph(t, input)
			root := graph.capture(attribute, occurrence)
			graph.ranks()
			effective := occurrence
			if named {
				effective = occurrence.Underlying()
			}
			// The random fixture specifies the second branch before either
			// implementation runs; the model input never borrows Resolve output.
			selected := referenceConstructor("selected", map[string]any{"occurrence": graph.identity(effective.ID()), "branch": graph.identity(effective.Branches()[1].ID), "payload": input.encode("retained")})
			request := map[string]any{
				"declarations": graph.declarations, "root": root,
				"supplied": map[string]any{"source": map[string]any{"occurrence": root, "origin": 1, "role": "authoredExample"}, "value": selected},
				"codecs":   input.codecs(), "checks": []any{}, "projection": nil,
			}
			actual := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{referenceConstructor("evaluate", map[string]any{"request": request})})[0])
			random := &productionChoiceRandom{Randomizer: expr.NewRandom("unused").Randomizer}
			generator := &expr.ExampleGenerator{Randomizer: random}
			selection := context.SelectExample(occurrence, expr.ExamplePolicy{Reachable: true})
			resolved := context.Synthesize(selection, generator)
			require.Equal(t, expr.ValueResolved, resolved.Outcome(), "%v", resolved.Diagnostics())
			require.True(t, resolved.Synthesized())
			legacy, present := resolved.LegacyValue()
			require.True(t, present)
			require.Equal(t, "retained", legacy)
			expected := graph.output(input, resolved, legacy)
			require.Equal(t, productionCanonical(t, jsontext.Value(productionJSON(t, expected))), productionCanonical(t, actual["resolved"]))
			again := context.Synthesize(selection, generator)
			require.True(t, resolved.SourceID() == again.SourceID())
			require.Equal(t, []string{"branch", "payload"}, random.calls)
			bare := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: legacy}), expr.ValueRoleExample)
			require.Equal(t, expr.ValueAmbiguous, bare.Outcome(), "discarding choice evidence must not accidentally remain resolved")
		})
	}
}
