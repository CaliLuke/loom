package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaAnnotationsRetainDeclaredValuesAtTargetCodecBoundary(t *testing.T) {
	for _, codec := range []struct {
		name  string
		value ValueCodec
	}{
		{"json", ValueCodecJSON}, {"text", ValueCodecText}, {"raw", ValueCodecRaw}, {"custom", ValueCodecCustom},
	} {
		for _, role := range []string{"default", "enum"} {
			t.Run(codec.name+"/"+role, func(t *testing.T) {
				attribute := &AttributeExpr{Type: String, Validation: &ValidationExpr{}}
				if role == "default" {
					attribute.DefaultValue = "portable"
				} else {
					attribute.Validation.Values = []any{"portable"}
				}
				context := NewValueContext()
				occurrence, err := context.NewOccurrence(attribute)
				require.NoError(t, err)
				plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: codec.value, Use: ValuePlanDocumentation})
				require.NoError(t, err)
				annotations, err := context.PrepareJSONSchemaAnnotations(occurrence, plan)
				require.NoError(t, err)
				value := annotations.Default
				if role == "enum" {
					value = annotations.Enums[0][0]
				} else {
					require.True(t, annotations.HasDefault)
				}
				require.Equal(t, "portable", value.DeclaredValue())
				expected := ProjectionUnsupported
				if codec.value == ValueCodecJSON {
					expected = ProjectionEmitted
				}
				require.Equal(t, expected, value.Projection().Outcome())
			})
		}
	}
}

func TestSchemaAnnotationsTargetCodecPreservesKnownSiblingChecks(t *testing.T) {
	for _, valid := range []bool{false, true} {
		t.Run(map[bool]string{false: "invalid", true: "valid"}[valid], func(t *testing.T) {
			bound := any(0.1)
			if valid {
				bound = "0.1"
			}
			custom := &AttributeExpr{Type: String}
			attribute := &AttributeExpr{
				Type: &Object{
					{Name: "amount", Attribute: &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Minimum: boundForTest(bound)}}},
					{Name: "custom", Attribute: custom},
				},
				DefaultValue: map[string]any{"amount": 0.1, "custom": "portable"},
			}
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
				Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
				Codecs: []ValueCodecPolicy{{Target: custom, Codec: ValueCodecCustom}},
			})
			require.NoError(t, err)
			annotations, err := context.PrepareJSONSchemaAnnotations(occurrence, plan)
			if !valid {
				require.ErrorContains(t, err, "unrepresentable")
				return
			}
			require.NoError(t, err)
			require.Equal(t, ProjectionUnsupported, annotations.Default.Projection().Outcome())
			require.Equal(t, attribute.DefaultValue, annotations.Default.DeclaredValue())
		})
	}
}

func TestSchemaAnnotationsTargetCodecRejectsInvalidSource(t *testing.T) {
	attribute := &AttributeExpr{Type: String, DefaultValue: 42}
	_, err := NewValueContext().NewOccurrence(attribute)
	require.ErrorContains(t, err, "violates the effective contract")
}
