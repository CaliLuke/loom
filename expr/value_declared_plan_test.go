package expr

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeclaredJSONValueSelectsRetainedPlanMember(t *testing.T) {
	text := &AttributeExpr{Type: String}
	bytes := &AttributeExpr{Type: Bytes}
	nullable := &AttributeExpr{Type: String, Nullable: true}
	absent := &AttributeExpr{Type: String}
	opaque := &AttributeExpr{Type: Any}
	root := &AttributeExpr{Type: &Object{
		{Name: "text", Attribute: text},
		{Name: "bytes", Attribute: bytes},
		{Name: "nullable", Attribute: nullable},
		{Name: "absent", Attribute: absent},
		{Name: "opaque", Attribute: opaque},
	}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(root)
	require.NoError(t, err)
	rawOpaque := jsontext.Value(`{"kept":true}`)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{
		"text": "selected", "bytes": "precise", "nullable": nil, "opaque": rawOpaque,
	}}), ValueRoleExample)
	require.Equal(t, ValueUnsupported, result.Outcome())

	for _, test := range []struct {
		name      string
		selection string
		attr      *AttributeExpr
		want      any
		ok        bool
	}{
		{name: "text", selection: "text", attr: text, want: "selected", ok: true},
		{name: "bytes", selection: "bytes", attr: bytes, want: []byte("precise"), ok: true},
		{name: "explicit null", selection: "nullable", attr: nullable, want: nil, ok: true},
		{name: "absent", selection: "absent", attr: absent},
		{name: "borrowed opaque", selection: "opaque", attr: opaque, want: rawOpaque, ok: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			plan, planErr := context.NewValuePlan(occurrence, ValuePlanRequest{
				Target: test.attr, Selection: []string{test.selection},
				Codec: ValueCodecText, Use: ValuePlanDocumentation,
			})
			require.NoError(t, planErr)
			got, ok := context.DeclaredJSONValue(result, plan)
			require.Equal(t, test.ok, ok)
			require.Equal(t, test.want, got)
		})
	}
}

func TestDeclaredJSONValueRequiresExactPlanOwnership(t *testing.T) {
	attribute := &AttributeExpr{Type: String}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: "value"}), ValueRoleExample)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.NoError(t, err)
	value, ok := context.DeclaredJSONValue(result, plan)
	require.True(t, ok)
	require.Equal(t, "value", value)

	schema, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanSchema,
	})
	require.NoError(t, err)
	_, ok = context.DeclaredJSONValue(result, schema)
	require.False(t, ok)
	_, ok = context.DeclaredJSONValue(result, ValuePlan{})
	require.False(t, ok)

	foreignContext := NewValueContext()
	foreignOccurrence, err := foreignContext.NewOccurrence(attribute)
	require.NoError(t, err)
	foreignPlan, err := foreignContext.NewValuePlan(foreignOccurrence, ValuePlanRequest{
		Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.NoError(t, err)
	_, ok = context.DeclaredJSONValue(result, foreignPlan)
	require.False(t, ok)

	sibling, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	siblingPlan, err := context.NewValuePlan(sibling, ValuePlanRequest{
		Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation,
	})
	require.NoError(t, err)
	_, ok = context.DeclaredJSONValue(result, siblingPlan)
	require.False(t, ok)

	invalid := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: 42}), ValueRoleExample)
	require.Equal(t, ValueInvalid, invalid.Outcome())
	_, ok = context.DeclaredJSONValue(invalid, plan)
	require.False(t, ok)
}
