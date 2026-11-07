package ir

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

func TestAdditionalBaselineProjectionOwnsChild(t *testing.T) {
	for _, nested := range []bool{false, true} {
		bytes := &expr.AttributeExpr{Type: expr.Bytes, Validation: &expr.ValidationExpr{MinLength: new(2), MaxLength: new(2)}}
		extra := bytes
		if nested {
			extra = &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: bytes}}}}
		}
		attribute := &expr.AttributeExpr{Type: &expr.Object{}, AdditionalAttributes: extra}
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		jsonPlan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
		require.NoError(t, err)
		rawPlan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecRaw, Use: expr.ValuePlanSchema})
		require.NoError(t, err)
		analyzer := NewAnalyzer(nil, false)
		baseline := analyzer.AnalyzeSchema(attribute)
		before, err := json.Marshal(baseline, json.Deterministic(true))
		require.NoError(t, err)
		require.Len(t, analyzer.constructions[baseline].slots, 1)
		require.Equal(t, jsonPlan.Root().Additional(), analyzer.constructions[baseline].slots[0].plan(jsonPlan.Root()))
		require.NotEmpty(t, byteProjectionIdentity(attribute, jsonPlan.Root()))
		require.Empty(t, byteProjectionIdentity(attribute, rawPlan.Root()))
		for _, plan := range []expr.ValuePlan{jsonPlan, rawPlan, jsonPlan} {
			actual := analyzer.projectBaseline(baseline, attribute, plan.Root()).AdditionalProperties.Schema
			if nested {
				actual = actual.Items.AdditionalProperties.Schema
			}
			if plan.Root().Codec() == expr.ValueCodecJSON {
				require.Equal(t, "base64", actual.ContentEncoding)
				require.Empty(t, actual.Format)
				require.Nil(t, actual.MaxLength)
			} else {
				require.Equal(t, "binary", actual.Format)
				require.Equal(t, 2, *actual.MaxLength)
			}
		}
		after, err := json.Marshal(baseline, json.Deterministic(true))
		require.NoError(t, err)
		require.Equal(t, string(before), string(after))
	}
}
