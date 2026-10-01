package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestOpenAPISchemaUsesRetainedSemanticUnionSelection(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Union{Values: []*expr.NamedAttributeExpr{
		{Name: "left", Attribute: &expr.AttributeExpr{Type: expr.String}},
		{Name: "right", Attribute: &expr.AttributeExpr{Type: expr.String}},
	}}}
	generator := &expr.ExampleGenerator{Randomizer: &selectionRandomizer{
		Randomizer: expr.NewFakerRandomizer("retained-selection"), choices: []int{0, 1},
	}}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	selection := context.SelectExample(occurrence, expr.ExamplePolicy{Reachable: true})
	retained := context.Synthesize(selection, generator)
	value, present := retained.DeclaredJSONValue()
	require.True(t, present)
	schemaPlan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{
		Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema,
	})
	require.NoError(t, err)
	documentationPlan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{
		Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation,
	})
	require.NoError(t, err)
	source := &service.ValueData{Context: context, Occurrence: occurrence, Example: retained}
	target := &transportir.ValueTarget{
		Source: source, Anchor: source, Codec: expr.ValueCodecJSON, Plan: schemaPlan,
		AnchorPlan: documentationPlan, ExampleOccurrence: occurrence, ExamplePlan: documentationPlan,
	}
	require.NoError(t, representation.PrepareTargetExamples(target, attribute, generator))

	schema := NewAnalyzer(generator, false).analyzePreparedOccurrence(attribute, "retained", target)
	require.Equal(t, value, schema.Example,
		"OpenAPI must consume the retained semantic branch instead of sampling again")
}

func TestPreparedExamplesKeepNeutralSchemaAuthority(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "data", Attribute: &expr.AttributeExpr{
			Type: expr.Bytes, UserExamples: []*expr.ExampleExpr{{Value: "hi"}},
		}},
		{Name: "label", Attribute: &expr.AttributeExpr{
			Type: expr.String, UserExamples: []*expr.ExampleExpr{{Value: "sibling"}},
		}},
	}}
	analyzer := NewAnalyzer(expr.NewRandom("neutral-prepared-examples"), false)
	schema := analyzer.AnalyzeSchema(attribute)
	require.Equal(t, "binary", schema.Properties["data"].Format)
	require.Empty(t, schema.Properties["data"].ContentEncoding)
	require.Equal(t, "aGk=", schema.Properties["data"].Example)
	require.Equal(t, "sibling", schema.Properties["label"].Example)
}

func TestSchemaPlanScopeRestoresStructuralAndExamplePositions(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: &expr.Object{
		{Name: "data", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
	}}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	structural, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{
		Target: attribute, Codec: expr.ValueCodecText, Use: expr.ValuePlanSchema,
	})
	require.NoError(t, err)
	examples, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{
		Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation,
	})
	require.NoError(t, err)
	analyzer := NewAnalyzer(nil, false)
	analyzer.plan = structural.Root()
	analyzer.examplePlan = examples.Root()
	require.Panics(t, func() {
		restore := analyzer.schemaPlanScope(expr.ValuePlanNode{}, examples.Root().Members()[0].Node)
		defer restore()
		require.False(t, analyzer.plan.Valid())
		require.Equal(t, examples.Root().Members()[0].Node, analyzer.examplePlan)
		panic("scope restoration")
	})
	require.Equal(t, structural.Root(), analyzer.plan)
	require.Equal(t, examples.Root(), analyzer.examplePlan)
}
