package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type schemaCallbackValue struct {
	calls *int
}

func (v schemaCallbackValue) MarshalJSON() ([]byte, error) {
	*v.calls++
	return []byte(`"custom"`), nil
}

func TestValuePlanSchemaDoesNotResolveEnumValues(t *testing.T) {
	calls := 0
	custom := &AttributeExpr{Type: Any, Validation: &ValidationExpr{Values: []any{schemaCallbackValue{&calls}}}}
	attribute := &AttributeExpr{Type: &Object{
		{Name: "builtin", Attribute: &AttributeExpr{Type: Bytes}},
		{Name: "custom", Attribute: custom},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanSchema,
		Codecs: []ValueCodecPolicy{{Target: custom, Codec: ValueCodecCustom}},
	})
	require.NoError(t, err, "schema graph capture does not require resolving excluded contract enum values")
	require.Zero(t, calls)
	require.Len(t, plan.Root().Members()[1].Node.Attribute().Validation.Values, 1)
	require.Equal(t, ValueCodecJSON, plan.Root().Members()[0].Node.Codec())
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"builtin": []byte("hi")}}), ValueRoleExample)
	require.Equal(t, ProjectionInvalidPlan, context.ProjectJSON(result, plan).Outcome())
	require.Zero(t, calls)
}

func TestValuePlanSchemaCodecOwnership(t *testing.T) {
	custom := &AttributeExpr{Type: Bytes}
	attribute := &AttributeExpr{Type: &Object{
		{Name: "builtin", Attribute: &AttributeExpr{Type: Bytes}},
		{Name: "custom", Attribute: custom},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
		Codecs: []ValueCodecPolicy{{Target: custom, Codec: ValueCodecCustom}},
	})
	require.NoError(t, err)
	fields := plan.Root().Members()
	require.Len(t, fields, 2)
	require.Equal(t, ValueCodecJSON, fields[0].Node.Codec())
	require.Equal(t, ValueCodecCustom, fields[1].Node.Codec())
	copy := fields[0].Node.Attribute()
	copy.Nullable = true
	require.False(t, fields[0].Node.Attribute().Nullable, "schema queries must not mutate the captured plan")
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"builtin": []byte("hi"), "custom": []byte("hi")}}), ValueRoleExample)
	require.Equal(t, ProjectionUnsupported, context.ProjectJSON(result, plan).Outcome(), "a nested external codec cannot acquire a builtin runtime claim")
}

func TestValuePlanSchemaAttributeDoesNotRegisterTypes(t *testing.T) {
	original := *GeneratedResultTypes
	t.Cleanup(func() {
		*GeneratedResultTypes = original
	})
	result := &ResultTypeExpr{UserTypeExpr: &UserTypeExpr{TypeName: "SchemaQueryItems", AttributeExpr: &AttributeExpr{
		Type: &Array{ElemType: &AttributeExpr{Type: String}},
	}}, Identifier: "application/vnd.schema-query-items"}
	GeneratedResultTypes.Append(result)
	attribute := &AttributeExpr{Type: result}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanSchema})
	require.NoError(t, err)
	before := len(*GeneratedResultTypes)
	for range 3 {
		copy := plan.Root().Attribute()
		require.NotNil(t, copy)
		require.Len(t, *GeneratedResultTypes, before)
		AsArray(copy.Type).ElemType.Type = Bytes
		require.Equal(t, String, AsArray(plan.Root().Attribute().Type).ElemType.Type)
	}
}

func TestValuePlanSchemaAttributeOwnsRecursiveSnapshots(t *testing.T) {
	raw := map[string]any{"bytes": []byte("hi")}
	named := &UserTypeExpr{TypeName: "SchemaQueryNode", AttributeExpr: &AttributeExpr{}}
	named.AttributeExpr.Type = &Object{
		{Name: "next", Attribute: &AttributeExpr{Type: named}},
		{Name: "blob", Attribute: &AttributeExpr{Type: Bytes, DefaultValue: []byte("hi"),
			Meta: MetaExpr{"description": {"original"}}, UserExamples: []*ExampleExpr{{Value: raw}}}},
	}
	attribute := &AttributeExpr{Type: named}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanSchema})
	require.NoError(t, err)
	raw["bytes"].([]byte)[0] = 'x'
	authoredExample := (*AsObject(named.AttributeExpr.Type))[1].Attribute.UserExamples[0]
	authoredExample.Value = map[string]any{"bytes": []byte("changed")}
	authoredExample.ExplicitNull = true
	first := plan.Root().Attribute()
	object := AsObject(first.Type)
	require.Same(t, first.Type, (*object)[0].Attribute.Type, "recursive identity must survive one query")
	blob := (*object)[1].Attribute
	require.False(t, blob.UserExamples[0].ExplicitNull)
	require.Equal(t, []byte("hi"), blob.UserExamples[0].Value.(map[string]any)["bytes"])
	blob.UserExamples[0].Value.(map[string]any)["bytes"].([]byte)[0] = 'z'
	blob.DefaultValue.([]byte)[0] = 'z'
	blob.Meta["description"][0] = "changed"
	second := AsObject(plan.Root().Attribute().Type)
	require.Equal(t, []byte("hi"), (*second)[1].Attribute.UserExamples[0].Value.(map[string]any)["bytes"])
	require.Equal(t, []byte("hi"), (*second)[1].Attribute.DefaultValue)
	require.Equal(t, "original", (*second)[1].Attribute.Meta["description"][0])
}
