package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONShapeTypedAdditionalAgreement(t *testing.T) {
	attribute := &AttributeExpr{
		Type:                 &Object{{Name: "name", Attribute: &AttributeExpr{Type: String}}},
		AdditionalAttributes: &AttributeExpr{Type: Int, Validation: &ValidationExpr{Minimum: boundForTest(1)}},
		Validation:           &ValidationExpr{Required: []string{"name", "count"}},
	}
	for _, alias := range []bool{false, true} {
		source := attribute
		if alias {
			source = &AttributeExpr{Type: &UserTypeExpr{TypeName: "Record", AttributeExpr: attribute}}
		}
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
			{`{"name":"ok","count":1}`, true},
			{`{"name":"ok","count":1,"extra":2}`, true},
			{`{"name":"ok","count":1,"extra":"bad"}`, false},
			{`{"name":"ok","count":1,"extra":0}`, false},
			{`{"name":"ok"}`, false}, {`{"count":1}`, false},
			{`{"name":"ok","count":0}`, false},
		} {
			prepared, err := plan.MatchJSONSchema([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, prepared, "prepared alias=%v %s", alias, tc.wire)
			lowered, err := shape.Match([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, lowered, "lowered alias=%v %s", alias, tc.wire)
		}
		evaluated, err := shape.Evaluated([]byte(`{"name":"ok","count":1,"extra":2}`))
		require.NoError(t, err)
		require.Equal(t, []string{"count", "extra", "name"}, evaluated)
	}
}

func TestJSONShapeTypedAdditionalOmittedMembers(t *testing.T) {
	source := &AttributeExpr{Type: &Object{
		{Name: "name", Attribute: &AttributeExpr{Type: String}},
		{Name: "secret", Attribute: &AttributeExpr{Type: Int}},
	}, AdditionalAttributes: &AttributeExpr{Type: Int}}
	target := DupAtt(source)
	target.omitMember("secret")
	target.Delete("secret")
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	shape, err := plan.JSONShape()
	require.NoError(t, err)
	for _, tc := range []struct {
		wire string
		want bool
	}{
		{`{"name":"ok","extra":1}`, true},
		{`{"name":"ok","secret":1}`, false},
	} {
		got, err := plan.MatchJSONSchema([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, got, tc.wire)
		got, err = shape.Match([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, got, tc.wire)
	}
}

func TestTypedAdditionalDecoderDomain(t *testing.T) {
	source := &AttributeExpr{Type: &Object{{Name: "secret", Attribute: &AttributeExpr{Type: Int}}}, AdditionalAttributes: &AttributeExpr{Type: Int32}, Validation: &ValidationExpr{Required: []string{"required"}}}
	target := DupAtt(source)
	target.omitMember("secret")
	target.Delete("secret")
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: target, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	for _, tc := range []struct {
		wire            string
		schema, decoder bool
	}{
		{`{"required":1,"extra":2}`, true, true},
		{`{"required":1,"extra":2147483648}`, true, false},
		{`{"required":1,"secret":2}`, false, false},
		{`{"extra":2}`, false, false},
	} {
		schema, err := plan.MatchJSONSchema([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.schema, schema, tc.wire)
		_, decoded := decodeJSON(plan.root, []byte(tc.wire))
		require.Equal(t, tc.decoder, decoded, tc.wire)
	}
}
