package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaAnnotationsExactBoundary(t *testing.T) {
	for _, role := range []string{"default", "enum"} {
		for _, exactDecimal := range []bool{false, true} {
			t.Run(role+map[bool]string{false: "/binary", true: "/decimal"}[exactDecimal], func(t *testing.T) {
				bound := any(0.1)
				if exactDecimal {
					bound = "0.1"
				}
				attribute := &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Minimum: boundForTest(bound)}}
				if role == "default" {
					attribute.DefaultValue = 0.1
				} else {
					attribute.Validation.Values = []any{0.1}
				}
				context := NewValueContext()
				occurrence, err := context.NewOccurrence(attribute)
				require.NoError(t, err)
				plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
				require.NoError(t, err)
				values, err := context.PrepareJSONSchemaAnnotations(occurrence, plan)
				if !exactDecimal {
					require.ErrorContains(t, err, role)
					require.ErrorContains(t, err, "unrepresentable")
					return
				}
				require.NoError(t, err)
				if role == "default" {
					require.True(t, values.HasDefault)
					require.Equal(t, 0.1, values.Default.DeclaredValue())
				} else {
					require.Equal(t, 0.1, values.Enums[0][0].DeclaredValue())
				}
			})
		}
	}
}

func TestSchemaAnnotationsRetainInheritedEnumPredicates(t *testing.T) {
	base := namedScalar("EnumSource", Float32, &ValidationExpr{Values: []any{1.23456789, 2.0}})
	derived := namedScalar("Refined", base, &ValidationExpr{Minimum: boundForTest(1.5)})
	attribute := &AttributeExpr{Type: derived}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	constraints := occurrence.EffectiveConstraints()
	target := DupAtt(attribute)
	target.Validation = constraints.Validation().Lowered()
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	base.Attribute().Validation.Values[0] = float64(9)
	values, err := context.PrepareJSONSchemaAnnotations(occurrence, plan)
	require.NoError(t, err)
	require.Equal(t, float32(1.23456789), values.Enums[0][0].DeclaredValue())
	require.Equal(t, float32(2), values.Enums[0][1].DeclaredValue())
	require.Equal(t, "EnumSource", values.Enums[0][0].Declaration().Declaration)
}

func TestInlineSchemaRejectsUnrepresentableRequiredAnnotations(t *testing.T) {
	for _, role := range []string{"default", "enum"} {
		attribute := &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Minimum: boundForTest(0.1)}}
		if role == "default" {
			attribute.DefaultValue = 0.1
		} else {
			attribute.Validation.Values = []any{0.1}
		}
		_, err := InlineJSONSchema(&AttributeExpr{Type: &Object{{Name: "amount", Attribute: attribute}}})
		require.ErrorContains(t, err, `object field "amount"`)
		require.ErrorContains(t, err, role)
		require.ErrorContains(t, err, "unrepresentable")
		attribute.Validation.Minimum = boundForTest("0.1")
		_, err = InlineJSONSchema(attribute)
		require.NoError(t, err)
	}
}

func TestSchemaAnnotationsOpaqueBoundaryPreservesKnownFailures(t *testing.T) {
	for _, valid := range []bool{false, true} {
		bound := any(0.1)
		if valid {
			bound = "0.1"
		}
		attribute := &AttributeExpr{
			Type: &Object{
				{Name: "amount", Attribute: &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Minimum: boundForTest(bound)}}},
				{Name: "opaque", Attribute: &AttributeExpr{Type: Any}},
			},
			DefaultValue: map[string]any{"amount": 0.1, "opaque": snapshotTextAppender("text")},
		}
		context := NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
		require.NoError(t, err)
		annotations, err := context.PrepareJSONSchemaAnnotations(occurrence, plan)
		if !valid {
			require.ErrorContains(t, err, "amount")
			require.ErrorContains(t, err, "unrepresentable")
		} else {
			require.NoError(t, err)
			require.Equal(t, ProjectionUnsupported, annotations.Default.Projection().Outcome())
		}
	}
}
