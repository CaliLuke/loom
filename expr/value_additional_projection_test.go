package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypedAdditionalValueProjection(t *testing.T) {
	length := 2
	bytes := &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{MinLength: &length, MaxLength: &length}}
	for _, tc := range []struct {
		name           string
		value          *AttributeExpr
		raw            any
		wire, rejected string
	}{
		{"bytes", bytes, []byte("hi"), `{"extra":"aGk="}`, `{"extra":"hi"}`},
		{"array", &AttributeExpr{Type: &Array{ElemType: bytes}}, [][]byte{[]byte("hi")}, `{"extra":["aGk="]}`, `{"extra":["hi"]}`},
		{"map", &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: bytes}}, map[string][]byte{"child": []byte("hi")}, `{"extra":{"child":"aGk="}}`, `{"extra":{"child":"hi"}}`},
		{"object", &AttributeExpr{Type: &Object{{Name: "payload", Attribute: bytes}}}, map[string]any{"payload": []byte("hi")}, `{"extra":{"payload":"aGk="}}`, `{"extra":{"payload":"hi"}}`},
		{"tagged union", &AttributeExpr{Type: &Union{Values: []*NamedAttributeExpr{{Name: "Text", Attribute: &AttributeExpr{Type: String}}, {Name: "Number", Attribute: &AttributeExpr{Type: Int}}}}}, "hi", `{"extra":{"type":"Text","value":"hi"}}`, `{"extra":"hi"}`},
		{"nullable", &AttributeExpr{Type: Bytes, Nullable: true}, nil, `{"extra":null}`, `{"extra":false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &AttributeExpr{Type: &Object{}, AdditionalAttributes: tc.value, Validation: &ValidationExpr{Required: []string{"extra"}}}
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(source)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
			require.NoError(t, err)
			shape, shapeErr := plan.JSONShape()
			if tc.name != "tagged union" {
				require.NoError(t, shapeErr)
			}
			for _, check := range []struct {
				wire string
				want bool
			}{{tc.wire, true}, {tc.rejected, false}, {`{}`, false}} {
				got, err := plan.MatchJSONSchema([]byte(check.wire))
				require.NoError(t, err)
				require.Equal(t, check.want, got, check.wire)
				if tc.name != "tagged union" {
					got, err = shape.Match([]byte(check.wire))
					require.NoError(t, err)
					require.Equal(t, check.want, got, check.wire)
				}
			}
			resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"extra": tc.raw}}), ValueRoleExample)
			require.Equal(t, ValueResolved, resolved.Outcome(), "%v", resolved.Diagnostics())
			projected := context.ProjectJSON(resolved, plan)
			require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
			wire, present := projected.JSON()
			require.True(t, present)
			require.JSONEq(t, tc.wire, string(wire))
			decoded, valid := decodeJSON(plan.root, wire)
			require.True(t, valid)
			value, ok := resolved.Value()
			require.True(t, ok)
			observed, failure := observeJSON(plan.root, ValueRoleExample, value)
			require.Nil(t, failure)
			require.True(t, projectionEqual(observed, decoded), "typed additional values must retain their carrier")
		})
	}
}

func TestTypedAdditionalRecursiveProjection(t *testing.T) {
	record := &UserTypeExpr{TypeName: "Recursive"}
	record.AttributeExpr = &AttributeExpr{Type: &Object{{Name: "payload", Attribute: &AttributeExpr{Type: Bytes}}}, AdditionalAttributes: &AttributeExpr{Type: record}}
	source := &AttributeExpr{Type: record}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	value := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"child": map[string]any{"payload": []byte("hi")}}}), ValueRoleExample)
	require.Equal(t, ValueResolved, value.Outcome())
	projected := context.ProjectJSON(value, plan)
	require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
	wire, present := projected.JSON()
	require.True(t, present)
	require.JSONEq(t, `{"child":{"payload":"aGk="}}`, string(wire))
	extra := plan.Root().Underlying().Additional()
	require.True(t, extra.Valid())
	owner, err := plan.OccurrenceForTarget(occurrence, extra)
	require.NoError(t, err)
	_, err = plan.ForOccurrence(owner, extra)
	require.NoError(t, err)
	shape, err := plan.JSONShape()
	require.NoError(t, err)
	match, err := shape.Match(wire)
	require.NoError(t, err)
	require.True(t, match)
}

func TestTypedAdditionalOmittedMembers(t *testing.T) {
	source := &AttributeExpr{Type: &Object{{Name: "secret", Attribute: &AttributeExpr{Type: Int}}}, AdditionalAttributes: &AttributeExpr{Type: Bytes}, Validation: &ValidationExpr{Required: []string{"a:b"}}}
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
		{`{"a:b":"aGk="}`, true}, {`{"a:b":"aGk=","secret":"aGk="}`, false}, {`{"a":"aGk="}`, false},
	} {
		matched, err := plan.MatchJSONSchema([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, matched, tc.wire)
		matched, err = shape.Match([]byte(tc.wire))
		require.NoError(t, err)
		require.Equal(t, tc.want, matched, tc.wire)
		_, matched = decodeJSON(plan.root, []byte(tc.wire))
		require.Equal(t, tc.want, matched, tc.wire)
	}
}

func TestAdditionalEmptyKeyRetainsPresence(t *testing.T) {
	source := &AttributeExpr{Type: &Object{}, AdditionalAttributes: &AttributeExpr{Type: Int}, Validation: &ValidationExpr{Required: []string{""}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"": 0}}), ValueRoleExample)
	require.Equal(t, ValueResolved, resolved.Outcome())
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(source), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	projected := context.ProjectJSON(resolved, plan)
	require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
	wire, present := projected.JSON()
	require.True(t, present)
	require.Equal(t, `{"":0}`, string(wire))
}

func TestTypedAdditionalOccurrenceRequirements(t *testing.T) {
	base := &UserTypeExpr{TypeName: "Record", AttributeExpr: &AttributeExpr{Type: &Object{}, AdditionalAttributes: &AttributeExpr{Type: Bytes}}}
	x := &AttributeExpr{Type: base, Validation: &ValidationExpr{Required: []string{"x"}}}
	y := &AttributeExpr{Type: base, Validation: &ValidationExpr{Required: []string{"y"}}}
	alias := &UserTypeExpr{TypeName: "XRecord", AttributeExpr: x}
	context := NewValueContext()
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		wire      string
		raw       any
		want      bool
	}{
		{"base", &AttributeExpr{Type: base}, `{}`, map[string]any{}, true},
		{"x absent", x, `{}`, map[string]any{}, false},
		{"x present", x, `{"x":"aGk="}`, map[string]any{"x": []byte("hi")}, true},
		{"sibling absent", y, `{"x":"aGk="}`, map[string]any{"x": []byte("hi")}, false},
		{"sibling present", y, `{"y":"aGk="}`, map[string]any{"y": []byte("hi")}, true},
		{"alias absent", &AttributeExpr{Type: alias}, `{}`, map[string]any{}, false},
		{"nested absent", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: x}}}}, `[{"child":{}}]`, []any{map[string]any{"child": map[string]any{}}}, false},
		{"nested present", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: String}, ElemType: x}}}}, `[{"child":{"x":"aGk="}}]`, []any{map[string]any{"child": map[string]any{"x": []byte("hi")}}}, true},
		{"base after siblings", &AttributeExpr{Type: base}, `{}`, map[string]any{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			occurrence, err := context.NewOccurrence(tc.attribute)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(tc.attribute), Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
			require.NoError(t, err)
			matched, err := plan.MatchJSONSchema([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, matched, "schema")
			shape, err := plan.JSONShape()
			require.NoError(t, err)
			matched, err = shape.Match([]byte(tc.wire))
			require.NoError(t, err)
			require.Equal(t, tc.want, matched, "shape")
			_, matched = decodeJSON(plan.root, []byte(tc.wire))
			require.Equal(t, tc.want, matched, "decoder")
			result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.raw}), ValueRoleExample)
			require.Equal(t, tc.want, result.Outcome() == ValueResolved, "source: %v", result.Diagnostics())
		})
	}
	require.Nil(t, base.Validation, "occurrence requirements must not mutate the shared declaration")
}
