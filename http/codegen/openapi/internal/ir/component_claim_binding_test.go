package ir

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestComponentClaimPreparedBinding(t *testing.T) {
	plans := make([]expr.ValuePlanNode, 2)
	for index := range plans {
		attribute := &expr.AttributeExpr{Type: expr.String}
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation})
		require.NoError(t, err)
		plans[index] = plan.Root()
	}
	first, second := &transportir.ValueTarget{}, &transportir.ValueTarget{}
	a := NewAnalyzer(expr.NewRandom("claim binding"), false)
	a.preparedExamples, a.examplePlan = first, plans[0]
	a.registerComponentReference("F", toRef("Public_variant"), "Public", true)
	for _, test := range []struct {
		name   string
		target *transportir.ValueTarget
		plan   expr.ValuePlanNode
		want   string
	}{
		{"exact pair", first, plans[0], toRef("Public_variant")},
		{"same target different plan", first, plans[1], ""},
		{"different target same plan", second, plans[0], ""},
		{"nil target", nil, plans[0], ""},
		{"invalid plan", first, expr.ValuePlanNode{}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			a.preparedExamples, a.examplePlan = test.target, test.plan
			require.Equal(t, test.want, findMatchingSchemaRef(a.schemasByFingerprint["F"], "Public", true, preparedComponentBinding{a.preparedExamples, a.examplePlan}))
		})
	}
	require.False(t, compatibleSchemaName(a.schemasByFingerprint["F"][0], "Public", true), "family preference cannot treat a suffix claim as the literal canonical name")
	require.Equal(t, toRef("Public_variant"), findMatchingSchemaRef(a.schemasByFingerprint["F"], "Public", false, preparedComponentBinding{}), "preferred requests retain the hint rule")
	a.preparedExamples, a.examplePlan = nil, expr.ValuePlanNode{}
	a.registerComponentReference("public", toRef("Public"), "Public", true)
	require.Equal(t, toRef("Public"), findMatchingSchemaRef(a.schemasByFingerprint["public"], "Public", true, preparedComponentBinding{}), "public canonical registration needs no invented prepared binding")
}

func TestComponentClaimRejectsUnboundSuffix(t *testing.T) {
	attribute := &expr.AttributeExpr{Type: expr.String}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanDocumentation})
	require.NoError(t, err)
	for _, test := range []struct {
		name   string
		target *transportir.ValueTarget
		plan   expr.ValuePlanNode
	}{
		{"nil target", nil, plan.Root()},
		{"invalid plan", &transportir.ValueTarget{}, expr.ValuePlanNode{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			a := NewAnalyzer(expr.NewRandom("invalid claim"), false)
			a.preparedExamples, a.examplePlan = test.target, test.plan
			require.PanicsWithValue(t, "OpenAPI canonical suffix requires a prepared example binding", func() {
				a.registerComponentReference("F", toRef("Public_variant"), "Public", true)
			})
			require.Empty(t, a.schemasByFingerprint, "invalid binding must fail before reservation publication")
			require.Empty(t, a.componentAcquisitions)
			require.Empty(t, a.schemas)
			require.Empty(t, a.schemaFingerprints)
		})
	}
}
