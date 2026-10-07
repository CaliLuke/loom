package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONShapeLoweringMatchesPreparedSchema(t *testing.T) {
	item := &UserTypeExpr{TypeName: "Item", AttributeExpr: &AttributeExpr{
		Type: &Object{
			{Name: "name", Attribute: &AttributeExpr{Type: String, Validation: &ValidationExpr{MinLength: new(2), Pattern: "^a"}}},
			{Name: "blob", Attribute: &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{Values: []any{"hi"}}}},
			{Name: "number", Attribute: &AttributeExpr{Type: UInt32, Validation: &ValidationExpr{ExclusiveMinimum: new(1.0)}}},
		},
		Validation: &ValidationExpr{Required: []string{"name"}},
		Meta:       MetaExpr{"openapi:additionalProperties": {"false"}},
	}}
	base := &UserTypeExpr{TypeName: "Items", AttributeExpr: &AttributeExpr{
		Type:       &Array{ElemType: &AttributeExpr{Type: item}},
		Validation: &ValidationExpr{MinLength: new(1)},
	}}
	source := &AttributeExpr{Type: base, Validation: &ValidationExpr{MaxLength: new(2)}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	shape, err := plan.JSONShape()
	require.NoError(t, err)
	for _, tc := range []struct {
		wire string
		want bool
	}{
		{`[{"name":"ab","blob":"aGk=","number":2}]`, true},
		{`[{"name":"ab"}]`, true}, {`[]`, false}, {`[{"name":"a"}]`, false},
		{`[{"name":"bb"}]`, false}, {`[{"name":"ab","extra":1}]`, false},
		{`[{"name":"ab","number":1}]`, false}, {`[{"name":"ab","blob":"hi"}]`, false},
		{`[{"name":"ab"},{"name":"ab"},{"name":"ab"}]`, false},
		{`[null]`, false}, {`null`, false},
	} {
		matched, err := shape.Match([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, matched, tc.wire)
		canonical, err := plan.MatchJSONSchema([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, canonical, matched, tc.wire)
	}
	*shape.Rules.MaxLength = 0
	canonical, err := plan.MatchJSONSchema([]byte(`[{"name":"ab"}]`))
	require.NoError(t, err)
	require.True(t, canonical, "lowered description must not mutate its source plan")
}
