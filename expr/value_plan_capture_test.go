package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValuePlanCapturedTargetRetainsShapeAndAncestry(t *testing.T) {
	context := NewValueContext()
	source := &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}}
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	target := DupAtt(source)
	captured, err := context.NewOccurrence(target)
	require.NoError(t, err)
	request := ValuePlanRequest{TargetOccurrence: captured, Codec: ValueCodecJSON, Use: ValuePlanDocumentation}
	before, err := context.NewValuePlan(occurrence, request)
	require.NoError(t, err)
	require.Same(t, captured.node.attribute, before.root.attribute, "plans must reuse the captured target storage")

	// Both root and descendant provenance can change independently of shape.
	source.valueSourceOrigin = &AttributeExpr{Type: String}
	field := AsObject(target.Type).Attribute("value")
	field.valueSourceOrigin = &AttributeExpr{Type: String}
	field.Meta = MetaExpr{"struct:tag:json": {"changed"}}
	target.valueSourceOrigin = &AttributeExpr{Type: String}
	after, err := context.NewValuePlan(occurrence, request)
	require.NoError(t, err)
	require.Equal(t, before.Root().Members()[0].WireName, after.Root().Members()[0].WireName)
	require.Equal(t, "value", after.Root().Members()[0].WireName)
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.ErrorContains(t, err, "ancestry")
}

func TestValuePlanCapturedTargetSeparatesRevisions(t *testing.T) {
	context := NewValueContext()
	source := &AttributeExpr{Type: &Object{{Name: "value", Attribute: &AttributeExpr{Type: String}}}}
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	target := DupAtt(source)
	captured, err := context.NewOccurrence(target)
	require.NoError(t, err)
	AsObject(target.Type).Attribute("value").Meta = MetaExpr{"struct:tag:json": {"changed"}}
	old, err := context.NewValuePlan(occurrence, ValuePlanRequest{TargetOccurrence: captured, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	fresh, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	require.Equal(t, "value", old.Root().Members()[0].WireName)
	require.Equal(t, "changed", fresh.Root().Members()[0].WireName)
}

func TestValuePlanCapturedTargetPreservesRecursiveSelection(t *testing.T) {
	list := &UserTypeExpr{TypeName: "List", AttributeExpr: &AttributeExpr{}}
	list.Type = &Object{
		{Name: "value", Attribute: &AttributeExpr{Type: String}},
		{Name: "next", Attribute: &AttributeExpr{Type: list, Nullable: true}},
	}
	attribute := &AttributeExpr{Type: list}
	context := NewValueContext()
	source, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	for _, test := range []struct {
		name      string
		target    *AttributeExpr
		selection []string
	}{
		{"root", DupAtt(attribute), nil},
		{"member", DupAtt(AsObject(list.Type).Attribute("value")), []string{"value"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			captured, err := context.NewOccurrence(test.target)
			require.NoError(t, err)
			rawPlan, err := context.NewValuePlan(source, ValuePlanRequest{Target: test.target, Selection: test.selection, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
			require.NoError(t, err)
			capturedPlan, err := context.NewValuePlan(source, ValuePlanRequest{TargetOccurrence: captured, Selection: test.selection, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
			require.NoError(t, err)
			rawShape, err := rawPlan.JSONShape()
			require.NoError(t, err)
			capturedShape, err := capturedPlan.JSONShape()
			require.NoError(t, err)
			require.Equal(t, rawShape, capturedShape)
		})
	}
}

func TestValuePlanCapturedTargetInputAdmission(t *testing.T) {
	context := NewValueContext()
	attribute := &AttributeExpr{Type: String}
	source, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	foreign, err := NewValueContext().NewOccurrence(attribute)
	require.NoError(t, err)
	unrelated, err := context.NewOccurrence(&AttributeExpr{Type: String})
	require.NoError(t, err)
	for _, test := range []struct {
		name     string
		raw      *AttributeExpr
		captured ValueOccurrence
		message  string
	}{
		{"both", attribute, source, "exactly one"},
		{"neither", nil, ValueOccurrence{}, "exactly one"},
		{"foreign", nil, foreign, "context"},
		{"unrelated", nil, unrelated, "ancestry"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := context.NewValuePlan(source, ValuePlanRequest{
				Target: test.raw, TargetOccurrence: test.captured, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
			})
			require.ErrorContains(t, err, test.message)
		})
	}
}

func TestValuePlanCapturedTargetKeepsPoliciesIndependent(t *testing.T) {
	context := NewValueContext()
	attribute := &AttributeExpr{Type: String}
	source, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	jsonPlan, err := context.NewValuePlan(source, ValuePlanRequest{TargetOccurrence: source, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	textPlan, err := context.NewValuePlan(source, ValuePlanRequest{TargetOccurrence: source, Codec: ValueCodecText, Use: ValuePlanRuntime})
	require.NoError(t, err)
	require.Equal(t, ValueCodecJSON, jsonPlan.Root().Codec())
	require.Equal(t, ValueCodecText, textPlan.Root().Codec())
}

// BenchmarkValuePlanTargetCapture separates immutable target reuse from the
// raw-attribute entrypoint. Width changes declaration size, not source identity.
func BenchmarkValuePlanTargetCapture(b *testing.B) {
	for _, width := range []int{8, 64, 256} {
		for _, captured := range []bool{false, true} {
			b.Run(fmt.Sprintf("fields=%d/captured=%t", width, captured), func(b *testing.B) {
				object := &Object{}
				for index := range width {
					*object = append(*object, &NamedAttributeExpr{Name: fmt.Sprintf("field%d", index), Attribute: &AttributeExpr{Type: String}})
				}
				attribute := &AttributeExpr{Type: object}
				context := NewValueContext()
				occurrence, err := context.NewOccurrence(attribute)
				require.NoError(b, err)
				request := ValuePlanRequest{Codec: ValueCodecJSON, Use: ValuePlanSchema}
				if captured {
					request.TargetOccurrence = occurrence
				} else {
					request.Target = attribute
				}
				b.ReportAllocs()
				for b.Loop() {
					_, err := context.NewValuePlan(occurrence, request)
					require.NoError(b, err)
				}
			})
		}
	}
}
