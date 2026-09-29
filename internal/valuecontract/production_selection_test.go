package valuecontract

import (
	"encoding/json/jsontext"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func checkProductionSelectionConformance(t *testing.T, executable string) {
	t.Helper()
	for _, tier := range []string{"local", "reference", "base", "type", "empty", "null", "invalid"} {
		for _, reachable := range []bool{false, true} {
			for _, suppress := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/reachable=%t/suppressed=%t", tier, reachable, suppress), func(t *testing.T) {
					checkProductionSelection(t, executable, tier, reachable, suppress)
				})
			}
		}
	}
}

func checkProductionSelection(t *testing.T, executable, tier string, reachable, suppressed bool) {
	t.Helper()
	context := expr.NewValueContext()
	examples := map[string]*expr.ExampleExpr{}
	for _, name := range []string{"old", "local", "reference", "later reference", "base", "type", "null", "invalid"} {
		examples[name] = &expr.ExampleExpr{Summary: name, Value: name}
	}
	examples["null"].Value, examples["null"].ExplicitNull = nil, true
	examples["invalid"].Value = 42
	reference := &expr.UserTypeExpr{TypeName: "Reference", AttributeExpr: &expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{examples["reference"]}}}
	later := &expr.UserTypeExpr{TypeName: "Later", AttributeExpr: &expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{examples["later reference"]}}}
	base := &expr.UserTypeExpr{TypeName: "Base", AttributeExpr: &expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{examples["base"]}}}
	named := &expr.UserTypeExpr{TypeName: "Named", AttributeExpr: &expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{examples["type"]}}}
	attribute := &expr.AttributeExpr{Type: named}
	groups := map[string][]string{"local": {}, "reference": {}, "later reference": {}, "base": {}, "type": {"type"}}
	switch tier {
	case "local", "null", "invalid":
		attribute.UserExamples = []*expr.ExampleExpr{examples["old"], examples[tier]}
		groups["local"] = []string{"old", tier}
		fallthrough
	case "reference":
		attribute.References = []expr.DataType{reference, later}
		groups["reference"], groups["later reference"] = []string{"reference"}, []string{"later reference"}
		fallthrough
	case "base":
		attribute.Bases = []expr.DataType{base}
		groups["base"] = []string{"base"}
	case "empty":
		attribute.Type = expr.String
		groups["type"] = nil
	}
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	identity := referenceIdentity{Occurrence: 1, Declaration: 1}
	input := newProductionInput(t)
	bySource := make(map[expr.ValueIdentity]any)
	byName := make(map[string]any)
	for index, name := range []string{"old", "local", "reference", "later reference", "base", "type", "null", "invalid"} {
		single, err := context.NewOccurrence(&expr.AttributeExpr{Type: expr.String, UserExamples: []*expr.ExampleExpr{examples[name]}})
		require.NoError(t, err)
		source, present := context.SelectExample(single, expr.ExamplePolicy{Reachable: true}).Source()
		require.True(t, present)
		supplied := map[string]any{"source": map[string]any{"occurrence": identity, "origin": index + 1, "role": "authoredExample"}, "value": input.encode(examples[name].Value)}
		byName[name], bySource[source.ID()] = supplied, supplied
	}
	group := func(name string) []any {
		values := make([]any, 0, len(groups[name]))
		for _, member := range groups[name] {
			values = append(values, byName[member])
		}
		return values
	}
	command := referenceConstructor("selectExample", map[string]any{
		"reachable": reachable, "suppressGenerated": suppressed,
		"sources": map[string]any{"localExamples": group("local"), "references": []any{group("reference"), group("later reference")}, "bases": []any{group("base")}, "typeExamples": group("type")},
	})
	actual := runReference(t, executable, []any{command})[0]
	selection := context.SelectExample(occurrence, expr.ExamplePolicy{Reachable: reachable, SuppressGenerated: suppressed})
	var expected any
	switch selection.State() {
	case expr.ExampleExcluded:
		expected = "excluded"
	case expr.ExampleSuppressed:
		expected = "suppressed"
	case expr.ExampleAbsent:
		expected = "absent"
	case expr.ExampleSelected:
		source, present := selection.Source()
		require.True(t, present)
		supplied, known := bySource[source.ID()]
		require.True(t, known, "selection manufactured an unrecognized source")
		expected = referenceConstructor("selected", map[string]any{"supplied": supplied})
		resolved := context.Resolve(occurrence, source, expr.ValueRoleExample)
		legacy, present := resolved.LegacyValue()
		require.True(t, present)
		require.Equal(t, examples[source.Origin()].Value, legacy)
		require.Equal(t, examples[source.Origin()].ExplicitNull, source.ExplicitNull())
		require.False(t, resolved.Synthesized())
	}
	require.Equal(t, productionCanonical(t, jsontext.Value(productionJSON(t, expected))), productionCanonical(t, actual))
}
