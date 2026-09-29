package ir

import (
	"os"
	"testing"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/schematest"
	"github.com/stretchr/testify/require"
)

// TestBaselineByteProjectionRenderedConstraints checks complete rendered graphs,
// independently of the construction records used to locate their byte gates.
func TestBaselineByteProjectionRenderedConstraints(t *testing.T) {
	if os.Getenv("LOOM_OPENAPI_CONTRACT") != "1" {
		t.Skip("independent schema engine runs in make openapi-contract")
	}
	pointer := func(value int) *int {
		return &value
	}
	maximum := int(^uint(0) >> 1)
	instances := []any{"", "aA==", "aGk=", "aGV5", "bG9uZw==", "aGl=", "aGk=\n", "aGk=\r", "aGk=\u2028", "aGk=\u2029", "!", "aGk", nil}
	type testCase struct {
		name                                    string
		inner, outer                            *expr.ValidationExpr
		innerNullable, outerNullable, reference bool
		lengths                                 []int
	}
	cases := []testCase{
		{name: "effective interval", inner: &expr.ValidationExpr{MinLength: pointer(2)}, outer: &expr.ValidationExpr{MaxLength: pointer(3)}, lengths: []int{2, 3}},
		{name: "referenced interval", inner: &expr.ValidationExpr{MinLength: pointer(2)}, outer: &expr.ValidationExpr{MaxLength: pointer(3)}, reference: true, lengths: []int{2, 3}},
		{name: "intersection before overflow", inner: &expr.ValidationExpr{MaxLength: &maximum}, outer: &expr.ValidationExpr{MaxLength: pointer(2)}, lengths: []int{0, 1, 2}},
		{name: "contradictory interval", inner: &expr.ValidationExpr{MinLength: &maximum}, outer: &expr.ValidationExpr{MaxLength: pointer(2)}},
		{name: "nullable declaration", inner: &expr.ValidationExpr{MinLength: pointer(2)}, outer: &expr.ValidationExpr{MaxLength: pointer(2)}, innerNullable: true, reference: true, lengths: []int{2}},
		{name: "nullable occurrence", inner: &expr.ValidationExpr{MinLength: pointer(2)}, outer: &expr.ValidationExpr{MaxLength: pointer(2)}, outerNullable: true, reference: true, lengths: []int{2}},
	}
	batches := make([]schematest.Batch, 0, 2*len(cases))
	for _, test := range cases {
		declared := &expr.AttributeExpr{Type: expr.Bytes, Validation: test.inner, Nullable: test.innerNullable}
		if test.reference {
			declared.Meta = expr.MetaExpr{"openapi:typename": {"ConstraintBytes"}, "openapi:typename:canonical": {"true"}}
		}
		attribute := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "ConstraintBytes", AttributeExpr: declared}, Validation: test.outer, Nullable: test.outerNullable}
		context := expr.NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err, test.name)
		plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanSchema})
		require.NoError(t, err, test.name)
		analyzer := NewAnalyzer(nil, false)
		schema := analyzer.analyzeOccurrence(attribute, "constraint/"+test.name, plan.Root())
		analyzer.finalizeRepresentations()
		batches = append(batches, schematest.Batch{Schema: map[string]any{"allOf": []any{RenderSchema(schema)}, "components": map[string]any{"schemas": RenderSchemaMap(analyzer.schemas)}}, Instances: instances})
		// Async construction retains its existing inline shape and null policy.
		// The original plan independently supplies the complete alias interval
		// for byte projection; flattening must not discard those bounds.
		asyncAnalyzer := NewAnalyzer(nil, false)
		prepared := analyzeAsyncSchema(asyncAnalyzer, attribute, inlineAsyncTestEndpoint(t, attribute), false, "constraint/"+test.name)
		asyncAnalyzer.finalizeRepresentations()
		materialized := materializeAsyncSchema(prepared, asyncAnalyzer.schemas)
		batches = append(batches, schematest.Batch{Schema: map[string]any{"allOf": []any{RenderSchema(materialized.schema)}, "components": map[string]any{"schemas": RenderSchemaMap(asyncAnalyzer.schemas)}}, Instances: instances})
	}
	results := schematest.New(t).Check(t, batches)
	for index, batch := range results {
		test := cases[index/2]
		accepted := make(map[int]bool)
		for _, length := range test.lengths {
			accepted[length] = true
		}
		for sample, result := range batch {
			expected := false
			if sample < 5 {
				expected = accepted[sample]
			}
			if sample == 5 {
				expected = accepted[2]
			}
			if sample == len(instances)-1 {
				expected = test.innerNullable
				if index%2 == 0 {
					expected = expected || test.outerNullable
				}
			}
			require.Equal(t, expected, result.Valid, "%s sample %v: %s", test.name, instances[sample], result.Errors)
		}
	}
}
