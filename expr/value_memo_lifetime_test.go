package expr

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"weak"

	"github.com/stretchr/testify/require"
)

type memoPanicJSON struct {
	calls   atomic.Int32
	entered chan struct{}
	release chan struct{}
}

func TestValueMemoDoesNotRetainDiscardedOwners(t *testing.T) {
	for _, kind := range []string{"resolution", "synthesis", "projection"} {
		t.Run(kind, func(t *testing.T) {
			context := NewValueContext()
			alive := discardedValueMemo(t, context, kind)
			for range 10 {
				runtime.GC()
			}
			require.False(t, alive(), "a live context must not retain an unused %s owner", kind)
			runtime.KeepAlive(context)
		})
	}
}

func TestValueMemoLiveSourceAndCaptureIdentity(t *testing.T) {
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: String})
	require.NoError(t, err)
	source := context.SupplyValue(ValueInput{Raw: "value"})
	first := context.Resolve(occurrence, source, ValueRoleExample)
	alias := source
	runtime.GC()
	require.Same(t, first.value.node, context.Resolve(occurrence, alias, ValueRoleExample).value.node)
	require.NotSame(t, first.value.node, context.Resolve(occurrence, alias, ValueRoleDefault).value.node)
	require.NotSame(t, first.value.node, context.Resolve(occurrence, context.SupplyValue(ValueInput{Raw: "value"}), ValueRoleExample).value.node)
	selection := context.SelectExample(occurrence, ExamplePolicy{Reachable: true, SynthesisScope: "backend"})
	generator := NewRandom("memo-identity")
	synthesized := context.Synthesize(selection, generator)
	runtime.GC()
	require.Same(t, synthesized.value.node, context.Synthesize(selection, generator).value.node)
	other := context.SelectExample(occurrence, ExamplePolicy{Reachable: true, SynthesisScope: "other"})
	require.NotSame(t, synthesized.value.node, context.Synthesize(other, generator).value.node)
}

func TestValueMemoConcurrentResolutionAndSynthesis(t *testing.T) {
	context := NewValueContext()
	occurrence, err := context.NewOccurrence(&AttributeExpr{Type: String})
	require.NoError(t, err)
	source := context.SupplyValue(ValueInput{Raw: "value"})
	selection := context.SelectExample(occurrence, ExamplePolicy{Reachable: true})
	generator := NewRandom("memo-concurrent")
	var workers sync.WaitGroup
	resolved, synthesized := make([]ValueResult, 8), make([]ValueResult, 8)
	for index := range resolved {
		workers.Go(func() {
			resolved[index] = context.Resolve(occurrence, source, ValueRoleExample)
			synthesized[index] = context.Synthesize(selection, generator)
		})
	}
	workers.Wait()
	for index := range resolved {
		require.Same(t, resolved[0].value.node, resolved[index].value.node)
		require.Same(t, synthesized[0].value.node, synthesized[index].value.node)
	}
}

func TestValueProjectionMemoSurvivesPlanCopiesAndSubplans(t *testing.T) {
	context := NewValueContext()
	attribute := &AttributeExpr{Type: Any}
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	var calls atomic.Int32
	source := context.SupplyValue(ValueInput{Raw: &projectionCustom{calls: &calls, wire: `"value"`}})
	result := context.Resolve(occurrence, source, ValueRoleExample)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	copy := plan
	subplan, err := plan.ForOccurrence(occurrence, plan.Root())
	require.NoError(t, err)
	for _, retained := range []ValuePlan{plan, copy, subplan} {
		runtime.GC()
		require.Equal(t, ProjectionEmitted, context.ProjectJSON(result, retained).Outcome())
	}
	require.EqualValues(t, 1, calls.Load())
	independent, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	require.Equal(t, ProjectionEmitted, context.ProjectJSON(result, independent).Outcome())
	require.EqualValues(t, 2, calls.Load())
}

func TestValueProjectionMemoReleasesReadersAfterPanic(t *testing.T) {
	context := NewValueContext()
	attribute := &AttributeExpr{Type: Any}
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	codec := &memoPanicJSON{entered: make(chan struct{}), release: make(chan struct{})}
	source := context.SupplyValue(ValueInput{Raw: codec})
	result := context.Resolve(occurrence, source, ValueRoleExample)
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	panicked := make(chan any, 1)
	go func() {
		defer func() {
			panicked <- recover()
		}()
		context.ProjectJSON(result, plan)
	}()
	<-codec.entered
	reader := make(chan ProjectionResult, 1)
	go func() {
		reader <- context.ProjectJSON(result, plan)
	}()
	close(codec.release)
	require.Equal(t, "memo codec panic", <-panicked)
	require.Equal(t, ProjectionUnsupported, (<-reader).Outcome())
	require.Equal(t, ProjectionUnsupported, context.ProjectJSON(result, plan).Outcome())
	require.EqualValues(t, 1, codec.calls.Load())
}

func (v *memoPanicJSON) MarshalJSON() ([]byte, error) {
	v.calls.Add(1)
	close(v.entered)
	<-v.release
	panic("memo codec panic")
}

// Keep the discarded handles outside the caller's stack during collection.
//
//go:noinline
func discardedValueMemo(t *testing.T, context *ValueContext, kind string) func() bool {
	t.Helper()
	attribute := &AttributeExpr{Type: String}
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	if kind == "synthesis" {
		result := context.Synthesize(context.SelectExample(occurrence, ExamplePolicy{Reachable: true}), NewRandom("memo-lifetime"))
		require.Equal(t, ValueResolved, result.Outcome())
		pointer := weak.Make(occurrence.graph)
		return func() bool {
			return pointer.Value() != nil
		}
	}
	source := context.SupplyValue(ValueInput{Raw: "value"})
	result := context.Resolve(occurrence, source, ValueRoleExample)
	require.Equal(t, ValueResolved, result.Outcome())
	if kind == "resolution" {
		pointer := weak.Make(source.data)
		return func() bool {
			return pointer.Value() != nil
		}
	}
	plan, err := context.NewValuePlan(occurrence, ValuePlanRequest{Target: attribute, Codec: ValueCodecJSON, Use: ValuePlanDocumentation})
	require.NoError(t, err)
	require.Equal(t, ProjectionEmitted, context.ProjectJSON(result, plan).Outcome())
	pointer := weak.Make(plan.root)
	return func() bool {
		return pointer.Value() != nil
	}
}
