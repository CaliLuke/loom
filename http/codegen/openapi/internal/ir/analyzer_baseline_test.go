package ir

import (
	"encoding/json/v2"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/stretchr/testify/require"
)

func TestBaselineProjectionUsesOwnedSlotsAndPreservesInput(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		limit := 2
		bytes := &expr.AttributeExpr{Type: expr.Bytes, Validation: &expr.ValidationExpr{MaxLength: &limit}, UserExamples: []*expr.ExampleExpr{{Value: "hi"}}}
		fields := &expr.Object{{Name: "data", Attribute: bytes}, {Name: "other", Attribute: &expr.AttributeExpr{Type: expr.String}}}
		attribute := &expr.AttributeExpr{Type: fields, Nullable: nullable}
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		jsonPlan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
		require.NoError(t, err)
		rawPlan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecText, Use: expr.ValuePlanSchema})
		require.NoError(t, err)
		calls := 0
		analyzer := NewAnalyzer(expr.NewRandom("baseline-slots"), false, WithExampleValue(func(attr *expr.AttributeExpr, raw any) (any, bool) {
			calls++
			return OpenAPIExampleValue(attr, raw)
		}))
		baseline := analyzer.AnalyzeSchema(attribute)
		before, err := json.Marshal(baseline, json.Deterministic(true))
		require.NoError(t, err)
		initialCalls := calls
		for _, plan := range []expr.ValuePlan{jsonPlan, rawPlan, rawPlan, jsonPlan} {
			projected := analyzer.projectBaseline(baseline, attribute, plan.Root())
			actual := projected
			if nullable {
				require.True(t, asyncNullableWrapper(projected))
				actual = projected.AnyOf[0]
			}
			if plan.Root().Codec() == expr.ValueCodecJSON {
				require.Equal(t, "base64", actual.Properties["data"].ContentEncoding)
				require.NotNil(t, actual.Properties["data"].Not)
				require.Nil(t, actual.Properties["data"].MaxLength)
			} else {
				require.Equal(t, "binary", actual.Properties["data"].Format)
				require.Equal(t, 2, *actual.Properties["data"].MaxLength)
			}
			after, err := json.Marshal(baseline, json.Deterministic(true))
			require.NoError(t, err)
			require.Equal(t, string(before), string(after), "projection cannot change a shared baseline")
			require.Equal(t, initialCalls, calls, "projection never samples an annotation")
		}
		require.Equal(t, 2, limit)
	}
}

func TestBaselineConstructionRecordsDoNotGuessAssertionEdges(t *testing.T) {
	a := NewAnalyzer(nil, false)
	shape := &Schema{Properties: map[string]*Schema{"value": {Type: "string"}}}
	a.recordStructuralSchema(shape, &expr.AttributeExpr{Type: &expr.Object{{Name: "value", Attribute: &expr.AttributeExpr{Type: expr.String}}}})
	require.Len(t, a.constructions[shape].slots, 1)
	slot := a.constructions[shape].slots[0]
	require.Equal(t, "value", slot.member)
	require.Same(t, shape.Properties["value"], slot.get(shape))
	arbitrary := &Schema{AllOf: []*Schema{shape}}
	_, found := a.constructions[arbitrary]
	require.False(t, found, "arbitrary assertions are not construction correspondence")
	a.applyNullableSchema(shape)
	require.Len(t, a.constructions[shape].slots, 1)
	require.Equal(t, schemaAnyOf, a.constructions[shape].slots[0].location)
	require.Equal(t, "value", a.constructions[shape.AnyOf[0]].slots[0].member)
	attribute := &expr.AttributeExpr{Type: expr.String}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
	require.NoError(t, err)
	a.plan = plan.Root()
	require.Panics(t, func() {
		restore := a.schemaPlanScope(expr.ValuePlanNode{})
		defer restore()
		panic("invalid child")
	})
	require.Equal(t, plan.Root(), a.plan)
}

func TestBaselineConstructionPairsActualTargetSlots(t *testing.T) {
	for _, untagged := range []bool{false, true} {
		raw := &expr.AttributeExpr{Type: expr.Bytes}
		union := &expr.Union{TypeName: "Slots", TypeKey: "tag", ValueKey: "payload", Untagged: untagged, Values: []*expr.NamedAttributeExpr{
			{Name: "json", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
			{Name: "raw", Attribute: raw},
		}}
		attribute := &expr.AttributeExpr{Type: &expr.Object{
			{Name: "renamed", Attribute: &expr.AttributeExpr{Type: expr.Bytes, Meta: expr.MetaExpr{"struct:tag:json:name": {"wire"}}}},
			{Name: "hidden", Attribute: &expr.AttributeExpr{Type: expr.Bytes, Meta: expr.MetaExpr{"struct:tag:json:name": {"-"}}}},
			{Name: "array", Attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Bytes}}}},
			{Name: "map", Attribute: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.Bytes}}}},
			{Name: "choice", Attribute: &expr.AttributeExpr{Type: union}},
		}}
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema, Codecs: []expr.ValueCodecPolicy{{Target: raw, Codec: expr.ValueCodecRaw}}})
		require.NoError(t, err)
		analyzer := NewAnalyzer(nil, false)
		baseline := analyzer.AnalyzeSchema(attribute)
		require.Len(t, analyzer.constructions[baseline].slots, 4)
		for _, slot := range analyzer.constructions[baseline].slots {
			require.True(t, slot.plan(plan.Root()).Valid(), slot.member)
			require.Same(t, baseline.Properties[slot.name], slot.get(baseline))
		}
		projected := analyzer.projectBaseline(baseline, attribute, plan.Root())
		require.NotContains(t, projected.Properties, "hidden")
		require.NotContains(t, projected.Properties, "-")
		for _, bytes := range []*Schema{projected.Properties["wire"], projected.Properties["array"].Items, projected.Properties["map"].AdditionalProperties.Schema} {
			require.Equal(t, "base64", bytes.ContentEncoding)
		}
		choice := projected.Properties["choice"]
		require.Len(t, choice.OneOf, 2)
		for index, tag := range []string{"json", "raw"} {
			payload := choice.OneOf[index]
			if !untagged {
				require.Equal(t, payload.Ref, choice.Discriminator.Mapping[tag])
				name, ok := schemaComponentName(payload.Ref)
				require.True(t, ok)
				payload = analyzer.schemas[name].Properties["payload"]
			}
			if tag == "json" {
				require.Equal(t, "base64", payload.ContentEncoding)
			} else {
				require.Equal(t, "binary", payload.Format)
				require.Empty(t, payload.ContentEncoding)
			}
		}
	}
}
