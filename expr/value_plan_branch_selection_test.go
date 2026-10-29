package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValuePlanSelectsUnionBranch(t *testing.T) {
	left := &AttributeExpr{Type: String}
	right := &AttributeExpr{Type: Int}
	source := &AttributeExpr{Type: &Object{{Name: "body", Attribute: &AttributeExpr{Type: &Union{TypeName: "Media", Values: []*NamedAttributeExpr{
		{Name: "text", Attribute: left}, {Name: "number", Attribute: right},
	}}}}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(source)
	require.NoError(t, err)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(left), Selection: []string{"body", "text"}, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	for _, tc := range []struct {
		name    string
		body    any
		emitted bool
	}{
		{"selected", "hello", true},
		{"other", 42, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"body": tc.body}}), ValueRoleExample)
			require.Equal(t, ValueResolved, result.Outcome())
			wire, ok := context.ProjectJSON(result, plan).JSON()
			require.Equal(t, tc.emitted, ok)
			if ok {
				require.JSONEq(t, `"hello"`, string(wire))
			}
		})
	}
	branch := occurrence.Members()[0].Occurrence.Branches()[0].Occurrence
	local, err := plan.ForOccurrence(branch, plan.Root())
	require.NoError(t, err)
	require.Empty(t, local.selection)
	_, err = context.NewValuePlan(occurrence, ValuePlanRequest{Target: DupAtt(left), Selection: []string{"body", "missing"}, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.Error(t, err)
}
