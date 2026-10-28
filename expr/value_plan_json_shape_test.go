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
			{Name: "number", Attribute: &AttributeExpr{Type: UInt32, Validation: &ValidationExpr{ExclusiveMinimum: boundForTest(1.0)}}},
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

func TestJSONShapeMapLoweringMatchesPreparedSchema(t *testing.T) {
	item := &UserTypeExpr{TypeName: "Item", AttributeExpr: &AttributeExpr{
		Type:       &Object{{Name: "name", Attribute: &AttributeExpr{Type: String, Validation: &ValidationExpr{MinLength: new(2)}}}},
		Validation: &ValidationExpr{Required: []string{"name"}},
		Meta:       MetaExpr{"openapi:additionalProperties": {"false"}},
	}}
	dictionary := &UserTypeExpr{TypeName: "Dictionary", AttributeExpr: &AttributeExpr{
		Type:       &Map{KeyType: &AttributeExpr{Type: String}, ElemType: &AttributeExpr{Type: item, Nullable: true}},
		Validation: &ValidationExpr{MinLength: new(1)},
	}}
	source := &AttributeExpr{Type: dictionary, Nullable: true, Validation: &ValidationExpr{MaxLength: new(2)}}
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
		{`null`, true}, {`{}`, false}, {`[]`, false},
		{`{"a":{"name":"ok"}}`, true}, {`{"a":null}`, true},
		{`{"a":{"name":"ok"},"b":null}`, true},
		{`{"a":null,"b":null,"c":null}`, false},
		{`{"a":{"name":"x"}}`, false}, {`{"a":{}}`, false},
		{`{"a":{"name":"ok","extra":1}}`, false},
		{`{"a":{"name":"ok"},"b":{"name":"x"}}`, false},
	} {
		matched, err := shape.Match([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, matched, tc.wire)
		canonical, err := plan.MatchJSONSchema([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, canonical, matched, tc.wire)
	}
}

func TestJSONShapeNestedUnionMatchesPreparedSchema(t *testing.T) {
	first := &UserTypeExpr{TypeName: "Person", AttributeExpr: &AttributeExpr{
		Type:       &Object{{Name: "name", Attribute: &AttributeExpr{Type: String}}},
		Validation: &ValidationExpr{Required: []string{"name"}},
	}}
	second := &UserTypeExpr{TypeName: "Group", AttributeExpr: &AttributeExpr{
		Type:       &Object{{Name: "group", Attribute: &AttributeExpr{Type: String}}},
		Validation: &ValidationExpr{Required: []string{"group"}},
	}}
	choice := &Union{TypeName: "Entry", Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "person", Attribute: &AttributeExpr{Type: first}},
		{Name: "group", Attribute: &AttributeExpr{Type: second}},
	}}
	source := &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: choice}}}
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
		{`[]`, true}, {`[{"name":"person"},{"group":"team"}]`, true},
		{`[{}]`, false}, {`[{"name":"person","group":"team"}]`, false},
		{`[null]`, false},
	} {
		actual, err := shape.Match([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, actual, tc.wire)
		canonical, err := plan.MatchJSONSchema([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, canonical, actual)
	}
}
