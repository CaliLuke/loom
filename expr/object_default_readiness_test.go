package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestObjectDefaultProjectionBoundary documents why the JSON projection owner
// cannot establish native-Go default representability: its explicit observation
// policy may omit undeclared members even for an admitted default.
func TestObjectDefaultProjectionBoundary(t *testing.T) {
	attribute := &AttributeExpr{Type: &Object{{Name: "x", Attribute: &AttributeExpr{Type: Int}}}}
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"x": 1, "extra": 2}}), ValueRoleDefault)
	require.Equal(t, ValueResolved, resolved.Outcome())
	value, ok := resolved.Value()
	require.True(t, ok)
	require.Len(t, value.Fields(), 2)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{
		Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanRuntime,
		Fields:     []ValueFieldPolicy{{Parent: attribute, Target: attribute.Find("x"), Name: "x", WireName: "x", Visible: true, Presence: ValueFieldRetain, NumericKind: IntKind}},
		Containers: []ValueContainerPolicy{{Target: attribute, RejectUnknownMembers: false, PreserveAdditional: false}},
	})
	require.NoError(t, err)
	projected := context.ProjectJSON(resolved, plan)
	require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
	wire, ok := projected.JSON()
	require.True(t, ok)
	require.JSONEq(t, `{"x":1}`, string(wire))
}

func TestResolvedRepresentationBoundary(t *testing.T) {
	context := NewValueContext()
	minimum := boundForTest(0)
	attribute := &AttributeExpr{Type: &Object{
		{Name: "x", Attribute: &AttributeExpr{Type: Int, Validation: &ValidationExpr{Minimum: minimum}}},
		{Name: "data", Attribute: &AttributeExpr{Type: Any}},
	}}
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	calls := 0
	opaque := &snapshotCustom{calls: &calls}
	result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"x": 1, "data": opaque}}), ValueRoleDefault)
	require.Equal(t, ValueUnsupported, result.Outcome())
	_, admitted := result.Value()
	require.False(t, admitted)
	value, retained := result.RepresentationValue()
	require.True(t, retained)
	fields := value.Fields()
	source, present := fields[1].Value.SourceValue()
	require.True(t, present)
	require.Same(t, opaque, source)
	require.Zero(t, calls)
	invalid := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: map[string]any{"x": -1, "data": opaque}}), ValueRoleDefault)
	_, retained = invalid.RepresentationValue()
	require.False(t, retained, "opaque sibling must not hide an independently invalid member")
	_, present = (ResolvedValue{}).SourceValue()
	require.False(t, present)

	raw := map[string]any{"x": 1, "data": []string{"original"}}
	result = context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), ValueRoleDefault)
	value, retained = result.RepresentationValue()
	require.True(t, retained)
	source, present = value.SourceValue()
	require.True(t, present)
	source.(map[string]any)["data"].([]string)[0] = "mutated"
	again, present := value.SourceValue()
	require.True(t, present)
	require.Equal(t, "original", again.(map[string]any)["data"].([]string)[0])
	require.Equal(t, "original", raw["data"].([]string)[0])
}
