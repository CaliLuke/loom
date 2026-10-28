package admission_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/admission"
)

// pendingDeadlineContext models the interval after a fixed deadline has elapsed
// but before its timer callback closes Done and publishes Err.
type pendingDeadlineContext struct {
	context.Context
	deadline time.Time
	done     <-chan struct{}
}

// Deadline returns the already elapsed deadline independently of timer delivery.
func (c pendingDeadlineContext) Deadline() (time.Time, bool) {
	return c.deadline, true
}

// Done remains open while the modeled timer callback is waiting to run.
func (c pendingDeadlineContext) Done() <-chan struct{} {
	return c.done
}

func TestRateExpiredDeadlineBeforeTimerSignal(t *testing.T) {
	ctx := pendingDeadlineContext{Context: context.Background(), deadline: time.Now().Add(-time.Second), done: make(chan struct{})}
	t.Run("local bucket", func(t *testing.T) {
		budget, err := admission.NewRate(admission.RatePolicy{PerSecond: 1, Burst: 1})
		require.NoError(t, err)
		decision, err := budget.Allow(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.False(t, decision.Allowed)
		allowRate(t, budget, true)
	})
	t.Run("backend", func(t *testing.T) {
		calls := 0
		control, err := admission.NewRateAdmission(func(context.Context, string) (admission.RateDecision, error) {
			calls++
			return admission.RateDecision{Allowed: true}, nil
		}, time.Second)
		require.NoError(t, err)
		decision, err := control.Check(ctx, "group")
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.False(t, decision.Allowed)
		require.Zero(t, calls)
	})
}

func TestRateAdmissionLateBackendAnswer(t *testing.T) {
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)
	control, err := admission.NewRateAdmission(func(ctx context.Context, _ string) (admission.RateDecision, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		// Avoid waiting on Done: the timer may not have been scheduled when the
		// backend returns. The result must be denied with either timer ordering.
		for time.Now().Before(deadline.Add(time.Millisecond)) {
		}
		return admission.RateDecision{Allowed: true}, nil
	}, time.Millisecond)
	require.NoError(t, err)
	decision, err := control.Check(context.Background(), "group")
	require.False(t, decision.Allowed)
	requireRateUnavailable(t, err)
}
