package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnionClosureMetadata(t *testing.T) {
	union := &Union{TypeName: "Choice", Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "text", Attribute: &AttributeExpr{Type: String}},
		{Name: "number", Attribute: &AttributeExpr{Type: Int}},
	}}
	for _, tc := range []struct {
		name, message string
		typ           DataType
		values        []string
	}{
		{"false", "", union, []string{"false"}},
		{"repeated false", "", union, []string{"false", "false"}},
		{"true", "only false", union, []string{"true"}},
		{"schema", "only false", union, []string{`{"type":"string"}`}},
		{"empty", "only false", union, nil},
		{"mixed", "only false", union, []string{"true", "false"}},
		{"object", "requires a union", &Object{}, []string{"false"}},
		{"primitive", "requires a union", String, []string{"false"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			att := &AttributeExpr{Type: tc.typ, Meta: MetaExpr{"openapi:unevaluatedProperties": tc.values}}
			err := att.Validate("closure", att)
			if tc.message != "" {
				require.ErrorContains(t, err, tc.message)
			} else {
				require.Empty(t, err.Errors)
			}
		})
	}
}

func TestUnionClosurePreparedAndLowered(t *testing.T) {
	record := &UserTypeExpr{TypeName: "Record", AttributeExpr: &AttributeExpr{
		Type:       &Object{{Name: "x", Attribute: &AttributeExpr{Type: Int}}},
		Validation: &ValidationExpr{Required: []string{"x"}},
	}}
	choice := &UserTypeExpr{TypeName: "Choice", AttributeExpr: &AttributeExpr{Type: &Union{
		TypeName: "Choice", Untagged: true, Values: []*NamedAttributeExpr{
			{Name: "record", Attribute: &AttributeExpr{Type: record}},
			{Name: "text", Attribute: &AttributeExpr{Type: String}},
		},
	}}}
	for _, inherited := range []bool{false, true} {
		for _, closed := range []bool{false, true} {
			source := &AttributeExpr{Type: Dup(choice)}
			if closed {
				owner := source
				if inherited {
					owner = source.Type.(UserType).Attribute()
				}
				owner.Meta = MetaExpr{"openapi:unevaluatedProperties": {"false"}}
			}
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(source)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
			require.NoError(t, err)
			shape, err := plan.JSONShape()
			require.NoError(t, err)
			for _, wire := range []string{`{"x":1}`, `{"x":1,"extra":2}`, `"text"`} {
				want := !closed || wire != `{"x":1,"extra":2}`
				prepared, err := plan.MatchJSONSchema([]byte(wire))
				require.NoError(t, err)
				require.Equal(t, want, prepared, "prepared closed=%v inherited=%v %s", closed, inherited, wire)
				lowered, err := shape.Match([]byte(wire))
				require.NoError(t, err)
				require.Equal(t, want, lowered, "lowered closed=%v inherited=%v %s", closed, inherited, wire)
				_, decoded := decodeJSON(plan.root, []byte(wire))
				require.Equal(t, want, decoded, "decoder closed=%v inherited=%v %s", closed, inherited, wire)
			}
		}
	}
	require.Empty(t, choice.Meta, "occurrence policy must leave the shared declaration open")
}

func TestUnionClosurePreservesIndependentDecoderSelection(t *testing.T) {
	branch := func(name string, typ DataType) *NamedAttributeExpr {
		return &NamedAttributeExpr{Name: name, Attribute: &AttributeExpr{Type: &UserTypeExpr{
			TypeName: name, AttributeExpr: &AttributeExpr{
				Type:       &Object{{Name: "x", Attribute: &AttributeExpr{Type: typ}}},
				Validation: &ValidationExpr{Required: []string{"x"}},
			},
		}}}
	}
	source := &AttributeExpr{
		Type: &Union{TypeName: "WidthChoice", Untagged: true, Values: []*NamedAttributeExpr{branch("Narrow", Int32), branch("Wide", Int64)}},
		Meta: MetaExpr{"openapi:unevaluatedProperties": {"false"}},
	}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	delete(source.Meta, "openapi:unevaluatedProperties")
	for _, tc := range []struct {
		wire    string
		decoded bool
	}{
		{`{"x":2147483648}`, true},
		{`{"x":2147483648,"extra":1}`, false},
		{`{"x":1}`, false},
	} {
		schema, err := plan.MatchJSONSchema([]byte(tc.wire))
		require.NoError(t, err)
		require.False(t, schema, "both mathematical integer schemas match %s", tc.wire)
		_, decoded := decodeJSON(plan.root, []byte(tc.wire))
		require.Equal(t, tc.decoded, decoded, "captured decoder closure %s", tc.wire)
	}
}

func TestUnionClosureTaggedEnvelope(t *testing.T) {
	source := &AttributeExpr{
		Type: &Union{TypeName: "Tagged", TypeKey: "kind", ValueKey: "content", Values: []*NamedAttributeExpr{
			{Name: "Text", Attribute: &AttributeExpr{Type: String}},
			{Name: "Number", Attribute: &AttributeExpr{Type: Int}},
		}},
		Meta: MetaExpr{"openapi:unevaluatedProperties": {"false"}},
	}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	for _, tc := range []struct {
		wire string
		want bool
	}{
		{`{"kind":"Text","content":"ok"}`, true},
		{`{"kind":"Text","content":"ok","extra":1}`, false},
		{`{"kind":"Text","extra":1}`, false},
	} {
		schema, err := plan.MatchJSONSchema([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, schema, "schema %s", tc.wire)
		_, decoded := decodeJSON(plan.root, []byte(tc.wire))
		require.Equal(t, tc.want, decoded, "decoder %s", tc.wire)
	}
}
