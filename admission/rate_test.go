package admission_test

import (
	"context"
	"math"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/admission"
)

func TestRatePolicyValidation(t *testing.T) {
	valid := admission.RatePolicy{PerSecond: 1, Burst: 2}
	r, err := admission.NewRate(valid)
	require.NoError(t, err)
	for _, policy := range []admission.RatePolicy{
		{PerSecond: 0, Burst: 1}, {PerSecond: -1, Burst: 1},
		{PerSecond: math.NaN(), Burst: 1}, {PerSecond: math.Inf(1), Burst: 1},
		{PerSecond: math.MaxFloat64, Burst: 1},
		{PerSecond: 1, Burst: 0}, {PerSecond: 1, Burst: -1},
	} {
		invalid, err := admission.NewRate(policy)
		require.Error(t, err)
		require.Nil(t, invalid)
		require.Error(t, r.SetPolicy(policy))
		got, version := r.Policy()
		require.Equal(t, valid, got)
		require.Equal(t, uint64(1), version)
	}
	var zero admission.Rate
	require.Error(t, zero.SetPolicy(valid))
	_, err = zero.Allow(context.Background())
	require.Error(t, err)
}

func TestRateUpdatesPreserveAllowance(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, err := admission.NewRate(admission.RatePolicy{PerSecond: 2, Burst: 2})
		require.NoError(t, err)
		allowRate(t, r, true)
		allowRate(t, r, true)
		require.Equal(t, 500*time.Millisecond, allowRate(t, r, false).RetryAfter)
		time.Sleep(250 * time.Millisecond)
		larger := admission.RatePolicy{PerSecond: 4, Burst: 4}
		require.NoError(t, r.SetPolicy(larger))
		decision := allowRate(t, r, false)
		require.Equal(t, uint64(2), decision.Version)
		require.Equal(t, 125*time.Millisecond, decision.RetryAfter, "old rate refills half a token before the update")
		time.Sleep(125 * time.Millisecond)
		allowRate(t, r, true)
		require.NoError(t, r.SetPolicy(larger))
		require.Equal(t, uint64(2), allowRate(t, r, false).Version, "unchanged updates neither refill nor change version")
		time.Sleep(time.Second)
		require.NoError(t, r.SetPolicy(admission.RatePolicy{PerSecond: 4, Burst: 1}))
		allowRate(t, r, true)
		allowRate(t, r, false)
		require.NoError(t, r.SetPolicy(admission.RatePolicy{PerSecond: 4, Burst: 3}))
		allowRate(t, r, false)
		time.Sleep(250 * time.Millisecond)
		allowRate(t, r, true)
		allowRate(t, r, false)
	})
}

func TestRateDecreaseUsesOldRefillThenNewRate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, err := admission.NewRate(admission.RatePolicy{PerSecond: 2, Burst: 1})
		require.NoError(t, err)
		allowRate(t, r, true)
		time.Sleep(250 * time.Millisecond)
		require.NoError(t, r.SetPolicy(admission.RatePolicy{PerSecond: 1, Burst: 1}))
		require.Equal(t, 500*time.Millisecond, allowRate(t, r, false).RetryAfter)
		time.Sleep(500 * time.Millisecond)
		allowRate(t, r, true)
		allowRate(t, r, false)
	})
}

func TestRateCancellationConsumesNothing(t *testing.T) {
	r, err := admission.NewRate(admission.RatePolicy{PerSecond: 1, Burst: 1})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = r.Allow(ctx)
	require.ErrorIs(t, err, context.Canceled)
	allowRate(t, r, true)
}

func TestRateInvalidUpdatePreservesSpentQuota(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r, err := admission.NewRate(admission.RatePolicy{PerSecond: 1, Burst: 1})
		require.NoError(t, err)
		allowRate(t, r, true)
		for _, invalid := range []admission.RatePolicy{{PerSecond: 2, Burst: 0}, {PerSecond: math.NaN(), Burst: 2}} {
			require.Error(t, r.SetPolicy(invalid))
			decision := allowRate(t, r, false)
			require.Equal(t, uint64(1), decision.Version)
			require.Equal(t, time.Second, decision.RetryAfter)
		}
	})
}

func TestRateConcurrentPolicyUpdatesDoNotRefill(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		first := admission.RatePolicy{PerSecond: 1, Burst: 2}
		second := admission.RatePolicy{PerSecond: 4, Burst: 1}
		r, err := admission.NewRate(first)
		require.NoError(t, err)
		start := make(chan struct{})
		var admitted atomic.Int64
		for i := range 100 {
			go func() {
				<-start
				policy := first
				if i%2 == 0 {
					policy = second
				}
				require.NoError(t, r.SetPolicy(policy))
				decision, err := r.Allow(context.Background())
				require.NoError(t, err)
				require.Positive(t, decision.Version)
				if decision.Allowed {
					admitted.Add(1)
				}
				observed, version := r.Policy()
				require.Contains(t, []admission.RatePolicy{first, second}, observed)
				require.Positive(t, version)
			}()
		}
		close(start)
		synctest.Wait()
		require.GreaterOrEqual(t, admitted.Load(), int64(1))
		require.LessOrEqual(t, admitted.Load(), int64(2), "updates cannot recreate spent allowance")
	})
}

func TestRateRetryDelayBoundaries(t *testing.T) {
	for _, tc := range []struct {
		delay   time.Duration
		seconds int64
	}{
		{0, 0}, {time.Nanosecond, 1}, {time.Second, 1},
		{time.Second + time.Nanosecond, 2}, {time.Duration(math.MaxInt64), 9223372037},
	} {
		require.Equal(t, tc.seconds, (admission.RateDecision{RetryAfter: tc.delay}).RetryAfterSeconds())
	}
	synctest.Test(t, func(t *testing.T) {
		r, err := admission.NewRate(admission.RatePolicy{PerSecond: math.SmallestNonzeroFloat64, Burst: 1})
		require.NoError(t, err)
		allowRate(t, r, true)
		require.Equal(t, time.Duration(math.MaxInt64), allowRate(t, r, false).RetryAfter)
	})
}

func allowRate(t *testing.T, r *admission.Rate, allowed bool) admission.RateDecision {
	t.Helper()
	decision, err := r.Allow(context.Background())
	require.NoError(t, err)
	require.Equal(t, allowed, decision.Allowed)
	return decision
}
