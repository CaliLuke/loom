package expr

import (
	"encoding/json/jsontext"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type valuePlanOpaqueEnumCodec struct {
	calls *atomic.Int32
}

func (v valuePlanOpaqueEnumCodec) MarshalJSON() ([]byte, error) {
	v.calls.Add(1)
	return []byte(`"opaque"`), nil
}

func TestValuePlanDocumentationPreservesOpaqueEnumClause(t *testing.T) {
	attribute := &AttributeExpr{Type: Any, Validation: &ValidationExpr{
		EnumClauses: [][]any{{jsontext.Value(`{"opaque":true}`)}},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)

	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.NoError(t, err)
	require.True(t, plan.root.hasEnum)
	require.Equal(t, [][]ResolvedValue{{}}, plan.root.enumClauses)
	require.False(t, schemaJSON(plan.root, jsontext.Value(`"ordinary"`)))
	_, accepted := decodeJSON(plan.root, jsontext.Value(`"ordinary"`))
	require.False(t, accepted)
	ordinary := context.ResolveDeclaredShape(occurrence,
		context.SupplyValue(ValueInput{Raw: "ordinary"}))
	require.Equal(t, ValueResolved, ordinary.Outcome())
	require.Equal(t, ProjectionUnrepresentable, context.ProjectJSON(ordinary, plan).Outcome())
}

func TestValuePlanDocumentationConjoinsKnownAndOpaqueEnumCandidates(t *testing.T) {
	tests := []struct {
		name    string
		clauses [][]any
		known   bool
	}{
		{
			name:    "mixed clause retains known member",
			clauses: [][]any{{"known", jsontext.Value(`{"opaque":true}`)}},
			known:   true,
		},
		{
			name: "opaque-only intersecting clause rejects known member",
			clauses: [][]any{
				{"known", jsontext.Value(`{"opaque":true}`)},
				{jsontext.Value(`{"other":true}`)},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			attribute := &AttributeExpr{Type: Any, Validation: &ValidationExpr{EnumClauses: test.clauses}}
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
				Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
			})
			require.NoError(t, err)
			require.Equal(t, test.known, schemaJSON(plan.root, jsontext.Value(`"known"`)))
			_, accepted := decodeJSON(plan.root, jsontext.Value(`"known"`))
			require.Equal(t, test.known, accepted)
			require.False(t, schemaJSON(plan.root, jsontext.Value(`"other"`)))
			_, accepted = decodeJSON(plan.root, jsontext.Value(`"other"`))
			require.False(t, accepted)

			known := context.ResolveDeclaredShape(occurrence,
				context.SupplyValue(ValueInput{Raw: "known"}))
			require.Equal(t, ValueResolved, known.Outcome())
			wantOutcome := ProjectionUnrepresentable
			if test.known {
				wantOutcome = ProjectionEmitted
			}
			require.Equal(t, wantOutcome, context.ProjectJSON(known, plan).Outcome())
			other := context.ResolveDeclaredShape(occurrence,
				context.SupplyValue(ValueInput{Raw: "other"}))
			require.Equal(t, ValueResolved, other.Outcome())
			require.Equal(t, ProjectionUnrepresentable, context.ProjectJSON(other, plan).Outcome())
		})
	}
}

func TestValuePlanDocumentationDoesNotMaterializeOpaqueEnumCodec(t *testing.T) {
	var calls atomic.Int32
	attribute := &AttributeExpr{Type: Any, Validation: &ValidationExpr{
		Values: []any{valuePlanOpaqueEnumCodec{calls: &calls}},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.NoError(t, err)
	require.Equal(t, [][]ResolvedValue{{}}, plan.root.enumClauses)
	require.Zero(t, calls.Load())
}

func TestValuePlanDocumentationRejectsMalformedEnumCandidate(t *testing.T) {
	attribute := &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	target := DupAtt(attribute)
	target.Validation = &ValidationExpr{EnumClauses: [][]any{{[]any{1}}}}
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.ErrorContains(t, err, "declared type")
}

func TestValuePlanDocumentationRejectsOpaqueEnumWithCheckableFailure(t *testing.T) {
	var calls atomic.Int32
	attribute := &AttributeExpr{Type: &Object{
		{Name: "opaque", Attribute: &AttributeExpr{Type: Any}},
		{Name: "invalid", Attribute: &AttributeExpr{Type: Int}},
	}}
	raw := map[string]any{
		"opaque":  valuePlanOpaqueEnumCodec{calls: &calls},
		"invalid": "not an integer",
	}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)

	invalid := context.ResolveDeclaredShape(occurrence.Members()[1].Occurrence,
		context.SupplyValue(ValueInput{Raw: raw["invalid"]}))
	require.Equal(t, ValueInvalid, invalid.Outcome())
	require.True(t, invalid.checkableFailure)
	result := context.ResolveDeclaredShape(occurrence,
		context.SupplyValue(ValueInput{Raw: raw}))
	require.Equal(t, ValueUnsupported, result.Outcome())
	require.True(t, result.checkableFailure)
	require.NotNil(t, result.value.node)
	require.Zero(t, calls.Load())

	target, err := context.NewOccurrence(DupAtt(attribute))
	require.NoError(t, err)
	require.NotNil(t, target.node.constraints)
	target.node.constraints.validation.rules.EnumClauses = [][]any{{raw}}
	node := &valuePlanNode{}
	builder := valuePlanBuilder{context: context, source: occurrence}
	err = builder.enums(node, occurrence.node, target.node)
	require.ErrorContains(t, err, "enum does not resolve against its declared shape")
	require.Zero(t, calls.Load())
}
