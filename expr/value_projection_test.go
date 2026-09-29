package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValueProjectionPreservesBranchAndSameWireContract(t *testing.T) {
	for _, tagged := range []bool{true, false} {
		attribute := &AttributeExpr{Type: &Union{Untagged: !tagged, Values: []*NamedAttributeExpr{
			{Name: "Bytes", Attribute: &AttributeExpr{Type: Bytes}},
			{Name: "Text", Attribute: &AttributeExpr{Type: String}},
		}}}
		context := NewValueContext()
		occurrence, err := context.NewOccurrence(attribute)
		require.NoError(t, err)
		resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: []byte("hi")}), ValueRoleExample)
		require.Equal(t, ValueResolved, resolved.Outcome())
		plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
		require.NoError(t, err)
		projected := context.ProjectJSON(resolved, plan)
		if tagged {
			require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
			wire, present := projected.JSON()
			require.True(t, present)
			require.JSONEq(t, `{"type":"Bytes","value":"aGk="}`, string(wire))
		} else {
			require.Equal(t, ProjectionUnrepresentable, projected.Outcome())
		}
	}
}

func TestValueProjectionObservesBeforeConstructingOmittedChild(t *testing.T) {
	child := &AttributeExpr{Type: &Object{{Name: "required", Attribute: &AttributeExpr{Type: String}}}, Validation: &ValidationExpr{Required: []string{"required"}}}
	attribute := &AttributeExpr{Type: &Object{{Name: "child", Attribute: child}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"child": map[string]any{}}}), ValueRoleExample)
	require.Equal(t, ValueIncomplete, resolved.Outcome())
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
		Fields: []ValueFieldPolicy{{Parent: attribute, Target: child, Name: "child", WireName: "child", Visible: true, Presence: ValueFieldOmitEmpty}},
	})
	require.NoError(t, err)
	projected := context.ProjectJSON(resolved, plan)
	require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
	wire, present := projected.JSON()
	require.True(t, present)
	require.Equal(t, `{}`, string(wire))
}

func TestValueProjectionSelectedBodyIgnoresMissingHeader(t *testing.T) {
	body := &AttributeExpr{Type: String}
	attribute := &AttributeExpr{Type: &Object{{Name: "header", Attribute: &AttributeExpr{Type: String}}, {Name: "body", Attribute: body}}, Validation: &ValidationExpr{Required: []string{"header", "body"}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"body": "hi"}}), ValueRoleExample)
	require.Equal(t, ValueIncomplete, resolved.Outcome())
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: body, Selection: []string{"body"}, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	projected := context.ProjectJSON(resolved, plan)
	require.Equal(t, ProjectionEmitted, projected.Outcome())
	wire, present := projected.JSON()
	require.True(t, present)
	require.Equal(t, `"hi"`, string(wire))
}

func TestValueProjectionSchemaDoesNotBorrowRuntimeKeyParsing(t *testing.T) {
	attribute := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Integers", Attribute: &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: Int}, ElemType: &AttributeExpr{Type: String}}}},
		{Name: "Booleans", Attribute: &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: Boolean}, ElemType: &AttributeExpr{Type: String}}}},
	}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[int]string{1: "one"}}), ValueRoleExample)
	require.Equal(t, ValueResolved, resolved.Outcome())
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	// Both emitted schemas admit an object with String values. The fact that
	// one runtime key parser rejects "1" cannot make schema oneOf unique.
	require.Equal(t, ProjectionUnrepresentable, context.ProjectJSON(resolved, plan).Outcome())
}

func TestValueProjectionRuntimeScalarAndAnyMatrix(t *testing.T) {
	minimum := .1
	length := 2
	alias := &UserTypeExpr{TypeName: "Blob", AttributeExpr: &AttributeExpr{Type: Bytes}}
	for _, tc := range []struct {
		name      string
		attribute *AttributeExpr
		raw       any
		outcome   ProjectionOutcome
		wire      string
	}{
		{"bytes", &AttributeExpr{Type: Bytes}, []byte("hi"), ProjectionEmitted, `"aGk="`},
		{"named bytes occurrence length", &AttributeExpr{Type: alias, Validation: &ValidationExpr{MinLength: &length, MaxLength: &length}}, []byte("hi"), ProjectionEmitted, `"aGk="`},
		{"decimal bound", &AttributeExpr{Type: Float64, Validation: &ValidationExpr{Minimum: &minimum}}, .1, ProjectionEmitted, `0.1`},
		{"integer overflow", &AttributeExpr{Type: Int32}, uint32(1 << 31), ProjectionUnrepresentable, ""},
		{"Any exact integer", &AttributeExpr{Type: Any}, uint64(9007199254740993), ProjectionEmitted, `9007199254740993`},
		{"Any bytes", &AttributeExpr{Type: Any}, []byte("hi"), ProjectionEmitted, `"aGk="`},
		{"Any byte array", &AttributeExpr{Type: Any}, [2]byte{104, 105}, ProjectionEmitted, `"aGk="`},
		{"byte slice declared array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: UInt}}}, []byte("hi"), ProjectionEmitted, `[104,105]`},
		{"byte array declared array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: UInt}}}, [2]byte{104, 105}, ProjectionEmitted, `[104,105]`},
		{"Any nil array", &AttributeExpr{Type: Any}, []string(nil), ProjectionEmitted, `[]`},
		{"Any nil bytes", &AttributeExpr{Type: Any}, []byte(nil), ProjectionEmitted, `""`},
		{"declared nil array", &AttributeExpr{Type: &Array{ElemType: &AttributeExpr{Type: String}}}, []string(nil), ProjectionIncomplete, ""},
		{"nullable null", &AttributeExpr{Type: String, Nullable: true}, nil, ProjectionEmitted, `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := NewValueContext()
			occurrence, err := context.NewOccurrence(tc.attribute)
			require.NoError(t, err)
			resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.raw}), ValueRoleExample)
			require.Equal(t, ValueResolved, resolved.Outcome(), "%v", resolved.Diagnostics())
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: tc.attribute, Codec: ValueCodecJSON, Use: ValuePlanRuntime})
			require.NoError(t, err)
			projected := context.ProjectJSON(resolved, plan)
			require.Equal(t, tc.outcome, projected.Outcome(), "%v", projected.Diagnostics())
			if tc.outcome == ProjectionEmitted {
				wire, present := projected.JSON()
				require.True(t, present)
				require.Equal(t, tc.wire, string(wire))
			}
		})
	}
}

func TestValueProjectionSchemaUniqueButRuntimeAmbiguous(t *testing.T) {
	attribute := &AttributeExpr{Type: &Union{Untagged: true, Values: []*NamedAttributeExpr{
		{Name: "Bytes", Attribute: &AttributeExpr{Type: Bytes, Validation: &ValidationExpr{Values: []any{[]byte("hi")}}}},
		{Name: "Text", Attribute: &AttributeExpr{Type: String, Validation: &ValidationExpr{Values: []any{"aGl="}}}},
	}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: "aGl="}), ValueRoleExample)
	require.Equal(t, ValueResolved, resolved.Outcome())
	for _, use := range []ValuePlanUse{ValuePlanDocumentation, ValuePlanRuntime} {
		request := ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: use}
		if use == ValuePlanRuntime {
			request.Containers = []ValueContainerPolicy{{Target: attribute}}
		}
		plan, err := context.NewValuePlan(occurrence, request)
		require.NoError(t, err)
		projected := context.ProjectJSON(resolved, plan)
		if use == ValuePlanDocumentation {
			require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
		} else {
			require.Equal(t, ProjectionUnrepresentable, projected.Outcome())
		}
	}
}

func TestValueProjectionRuntimeMixedWidthKeys(t *testing.T) {
	attribute := &AttributeExpr{Type: &Map{KeyType: &AttributeExpr{Type: Any}, ElemType: &AttributeExpr{Type: String}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[any]string{float32(.1): "narrow", float64(float32(.1)): "wide"}}), ValueRoleExample)
	require.Equal(t, ValueResolved, resolved.Outcome())
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanRuntime})
	require.NoError(t, err)
	projected := context.ProjectJSON(resolved, plan)
	require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
	wire, present := projected.JSON()
	require.True(t, present)
	require.Equal(t, `{"0.1":"narrow","0.10000000149011612":"wide"}`, string(wire))
}

func TestValueProjectionRequiresExactOccurrenceOwner(t *testing.T) {
	childAttribute := &AttributeExpr{Type: Any}
	attribute := &AttributeExpr{Type: &Array{ElemType: childAttribute}}
	context := NewValueContext()
	root, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	child := root
	child.node = root.node.declaration.element
	source := context.SupplyValue(ValueInput{Raw: []int{1}})
	rootResult := context.Resolve(root, source, ValueRoleExample)
	childResult := context.Resolve(child, source, ValueRoleExample)
	require.Equal(t, ValueResolved, rootResult.Outcome())
	require.Equal(t, ValueResolved, childResult.Outcome())
	plan, err := context.NewValuePlan(child, ValuePlanRequest{Target: childAttribute, Codec: ValueCodecJSON, Use: ValuePlanRuntime})
	require.NoError(t, err)
	if outcome := context.ProjectJSON(rootResult, plan).Outcome(); outcome != ProjectionInvalidPlan {
		t.Errorf("wrong occurrence outcome = %v, want InvalidPlan", outcome)
	}
	projected := context.ProjectJSON(childResult, plan)
	require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
	wire, present := projected.JSON()
	require.True(t, present)
	require.Equal(t, `[1]`, string(wire))
}
