package expr

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type projectionCustom struct {
	calls *atomic.Int32
	wire  string
	err   error
}

func (v *projectionCustom) MarshalJSON() ([]byte, error) {
	v.calls.Add(1)
	return []byte(v.wire), v.err
}

func TestValueProjectionCustomAnySnapshotOncePerTarget(t *testing.T) {
	context := NewValueContext()
	attribute := &AttributeExpr{Type: Any}
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	var calls atomic.Int32
	raw := &projectionCustom{calls: &calls, wire: `{"z":9007199254740993,"a":{"y":2,"x":1}}`}
	resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
	require.Equal(t, ValueUnsupported, resolved.Outcome())
	require.Zero(t, calls.Load())
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanRuntime})
	require.NoError(t, err)
	var workers sync.WaitGroup
	results := make([]ProjectionResult, 8)
	for index := range results {
		workers.Go(func() {
			results[index] = context.ProjectJSON(resolved, plan)
		})
	}
	workers.Wait()
	require.EqualValues(t, 1, calls.Load())
	for _, result := range results {
		require.Equal(t, ProjectionEmitted, result.Outcome(), "%v", result.Diagnostics())
		wire, present := result.JSON()
		require.True(t, present)
		require.Equal(t, `{"a":{"x":1,"y":2},"z":9007199254740993}`, string(wire))
		wire[0] = '['
		copy, present := result.JSON()
		require.True(t, present)
		require.Equal(t, byte('{'), copy[0])
	}
	second, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	require.Equal(t, ProjectionEmitted, context.ProjectJSON(resolved, second).Outcome())
	require.EqualValues(t, 2, calls.Load())
}

func TestValueProjectionCustomFailureAndRuntimeBoundary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		typ     DataType
		use     ValuePlanUse
		failure error
		outcome ProjectionOutcome
		calls   int32
	}{
		{"authored codec error", Any, ValuePlanRuntime, errors.New("codec failed"), ProjectionUnrepresentable, 1},
		{"declared runtime boundary", String, ValuePlanRuntime, nil, ProjectionUnsupported, 0},
		{"declared docs snapshot", String, ValuePlanDocumentation, nil, ProjectionEmitted, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := NewValueContext()
			attribute := &AttributeExpr{Type: tc.typ}
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			var calls atomic.Int32
			raw := &projectionCustom{calls: &calls, wire: `"owned"`, err: tc.failure}
			resolved := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: raw}), ValueRoleExample)
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: tc.use})
			require.NoError(t, err)
			result := context.ProjectJSON(resolved, plan)
			require.Equal(t, tc.outcome, result.Outcome(), "%v", result.Diagnostics())
			require.Equal(t, tc.calls, calls.Load())
			if tc.failure != nil {
				require.ErrorIs(t, result.Err(), tc.failure)
			}
		})
	}
}

func TestValueProjectionUsesActualCustomProtocols(t *testing.T) {
	for _, tc := range []struct {
		name    string
		raw     any
		outcome ValueOutcome
		wire    string
	}{
		{"pointer receiver value", snapshotPointerCodec{"ignored"}, ValueUnsupported, `"pointer codec"`},
		{"text appender", snapshotTextAppender("text"), ValueUnsupported, `"text"`},
		{"wrong method signature", snapshotWrongCodec{"builtin"}, ValueResolved, `["builtin"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := NewValueContext()
			attribute := &AttributeExpr{Type: Any}
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			result := context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: tc.raw}), ValueRoleExample)
			require.Equal(t, tc.outcome, result.Outcome())
			plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanRuntime})
			require.NoError(t, err)
			projected := context.ProjectJSON(result, plan)
			require.Equal(t, ProjectionEmitted, projected.Outcome(), "%v", projected.Diagnostics())
			wire, present := projected.JSON()
			require.True(t, present)
			require.Equal(t, tc.wire, string(wire))
		})
	}
}
