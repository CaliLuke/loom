package admission_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/admission"
)

func TestKeyedRateValidation(t *testing.T) {
	valid := admission.RatePolicy{PerSecond: 1, Burst: 1}
	for _, tc := range []struct {
		name    string
		policy  admission.RatePolicy
		maxKeys int
	}{
		{"zero rate", admission.RatePolicy{Burst: 1}, 1},
		{"zero burst", admission.RatePolicy{PerSecond: 1}, 1},
		{"zero keys", valid, 0},
		{"negative keys", valid, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keyed, err := admission.NewKeyedRate(tc.policy, tc.maxKeys)
			require.Error(t, err)
			require.Nil(t, keyed)
		})
	}
	var zero admission.KeyedRate
	_, err := zero.Allow(context.Background(), "a")
	require.Error(t, err)
	require.Error(t, zero.SetPolicy(valid))
}

func TestKeyedRateKeysHaveIndependentBudgets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keyed := newKeyedRate(t, admission.RatePolicy{PerSecond: 1, Burst: 1}, 4)
		requireKeyedAllowed(t, keyed, "a", true)
		denied := requireKeyedAllowed(t, keyed, "a", false)
		require.Equal(t, time.Second, denied.RetryAfter)
		requireKeyedAllowed(t, keyed, "b", true)
		require.Equal(t, 2, keyed.Len())
	})
}

func TestKeyedRateCapacityDeniesNewKeysUntilABucketIsFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keyed := newKeyedRate(t, admission.RatePolicy{PerSecond: 1, Burst: 2}, 2)
		requireKeyedAllowed(t, keyed, "a", true)
		time.Sleep(500 * time.Millisecond)
		requireKeyedAllowed(t, keyed, "b", true)

		denied := requireKeyedAllowed(t, keyed, "c", false)
		require.Equal(t, 500*time.Millisecond, denied.RetryAfter, "a refills first")
		require.Equal(t, uint64(1), denied.Version)
		time.Sleep(200 * time.Millisecond)
		denied = requireKeyedAllowed(t, keyed, "d", false)
		require.Equal(t, 300*time.Millisecond, denied.RetryAfter)
		require.Equal(t, 2, keyed.Len())

		time.Sleep(300 * time.Millisecond)
		requireKeyedAllowed(t, keyed, "c", true)
		require.Equal(t, 2, keyed.Len(), "only the full bucket a was evicted")
		denied = requireKeyedAllowed(t, keyed, "d", false)
		require.Equal(t, 500*time.Millisecond, denied.RetryAfter, "b refills next")
	})
}

func TestKeyedRateEvictionNeverResetsSpentQuota(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keyed := newKeyedRate(t, admission.RatePolicy{PerSecond: 1, Burst: 2}, 2)
		requireKeyedAllowed(t, keyed, "spent", true)
		requireKeyedAllowed(t, keyed, "spent", true)
		requireKeyedAllowed(t, keyed, "spent", false)
		for i := range 20 {
			time.Sleep(50 * time.Millisecond)
			_, err := keyed.Allow(t.Context(), fmt.Sprintf("rotating-%d", i))
			require.NoError(t, err)
		}
		// One second has passed: spent has refilled one token, not the burst.
		requireKeyedAllowed(t, keyed, "spent", true)
		requireKeyedAllowed(t, keyed, "spent", false)
	})
}

func TestKeyedRatePolicyUpdateReevaluatesEviction(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keyed := newKeyedRate(t, admission.RatePolicy{PerSecond: 1, Burst: 3}, 1)
		requireKeyedAllowed(t, keyed, "a", true)
		denied := requireKeyedAllowed(t, keyed, "b", false)
		require.Equal(t, time.Second, denied.RetryAfter)

		// Two tokens remain; lowering the burst to two makes a full at once.
		require.NoError(t, keyed.SetPolicy(admission.RatePolicy{PerSecond: 1, Burst: 2}))
		allowed := requireKeyedAllowed(t, keyed, "b", true)
		require.Equal(t, uint64(2), allowed.Version)
		require.Equal(t, 1, keyed.Len())
	})
}

func TestKeyedRatePolicyUpdatesApplyToStoredBuckets(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		policy := admission.RatePolicy{PerSecond: 1, Burst: 1}
		keyed := newKeyedRate(t, policy, 2)
		requireKeyedAllowed(t, keyed, "a", true)

		require.NoError(t, keyed.SetPolicy(policy))
		got, version := keyed.Policy()
		require.Equal(t, policy, got)
		require.Equal(t, uint64(1), version, "identical settings keep the version")
		require.Error(t, keyed.SetPolicy(admission.RatePolicy{PerSecond: -1, Burst: 1}))
		_, version = keyed.Policy()
		require.Equal(t, uint64(1), version, "invalid settings change nothing")

		faster := admission.RatePolicy{PerSecond: 10, Burst: 1}
		require.NoError(t, keyed.SetPolicy(faster))
		denied := requireKeyedAllowed(t, keyed, "a", false)
		require.Equal(t, 100*time.Millisecond, denied.RetryAfter, "stored bucket refills at the new rate")
		require.Equal(t, uint64(2), denied.Version)
	})
}

func TestKeyedRateCanceledContextStoresNoKey(t *testing.T) {
	keyed := newKeyedRate(t, admission.RatePolicy{PerSecond: 1, Burst: 1}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := keyed.Allow(ctx, "a")
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, keyed.Len())
}

func TestKeyedRateConcurrentKeysStayBounded(t *testing.T) {
	const maxKeys = 8
	keyed := newKeyedRate(t, admission.RatePolicy{PerSecond: 1000, Burst: 1}, maxKeys)
	var wg sync.WaitGroup
	for worker := range 16 {
		wg.Go(func() {
			for i := range 200 {
				_, err := keyed.Allow(context.Background(), fmt.Sprintf("%d-%d", worker, i%32))
				if err != nil {
					t.Errorf("allow: %v", err)
					return
				}
				if n := keyed.Len(); n > maxKeys {
					t.Errorf("stored %d keys, limit %d", n, maxKeys)
					return
				}
			}
		})
	}
	wg.Wait()
}

func TestKeyedRateAllowIsARateCheck(t *testing.T) {
	keyed := newKeyedRate(t, admission.RatePolicy{PerSecond: 1, Burst: 1}, 2)
	control, err := admission.NewRateAdmission(keyed.Allow, time.Second)
	require.NoError(t, err)
	decision, err := control.Check(context.Background(), "a")
	require.NoError(t, err)
	require.True(t, decision.Allowed)
	decision, err = control.Check(context.Background(), "a")
	require.NoError(t, err)
	require.False(t, decision.Allowed)
	require.Positive(t, decision.RetryAfter)
}

func newKeyedRate(t *testing.T, policy admission.RatePolicy, maxKeys int) *admission.KeyedRate {
	t.Helper()
	keyed, err := admission.NewKeyedRate(policy, maxKeys)
	require.NoError(t, err)
	return keyed
}

func requireKeyedAllowed(t *testing.T, keyed *admission.KeyedRate, key string, allowed bool) admission.RateDecision {
	t.Helper()
	decision, err := keyed.Allow(t.Context(), key)
	require.NoError(t, err)
	require.Equal(t, allowed, decision.Allowed, "key %q", key)
	if allowed {
		require.Zero(t, decision.RetryAfter)
	} else {
		require.Positive(t, decision.RetryAfter)
	}
	return decision
}

func TestKeyedRateEvictsABucketThatFillsBeforeEarlierOnes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keyed := newKeyedRate(t, admission.RatePolicy{PerSecond: 1, Burst: 3}, 2)
		for range 3 {
			requireKeyedAllowed(t, keyed, "a", true)
		}
		requireKeyedAllowed(t, keyed, "b", true)
		denied := requireKeyedAllowed(t, keyed, "c", false)
		require.Equal(t, time.Second, denied.RetryAfter, "b is full after one second")

		time.Sleep(time.Second)
		requireKeyedAllowed(t, keyed, "c", true)
		// a is full at 3s, but c, inserted later, is full at 2s.
		denied = requireKeyedAllowed(t, keyed, "d", false)
		require.Equal(t, time.Second, denied.RetryAfter)
		time.Sleep(time.Second)
		requireKeyedAllowed(t, keyed, "d", true)
		require.Equal(t, 2, keyed.Len())
	})
}

func TestKeyedRateReordersAStoredBucketAfterItSpends(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		keyed := newKeyedRate(t, admission.RatePolicy{PerSecond: 1, Burst: 2}, 2)
		requireKeyedAllowed(t, keyed, "a", true) // a is full at 1s
		time.Sleep(500 * time.Millisecond)
		requireKeyedAllowed(t, keyed, "b", true) // b is full at 1.5s
		time.Sleep(400 * time.Millisecond)
		requireKeyedAllowed(t, keyed, "a", true) // a is now full at 2.1s
		time.Sleep(600 * time.Millisecond)
		// At 1.5s b is full; a spending again must not hide b from eviction.
		requireKeyedAllowed(t, keyed, "c", true)
		require.Equal(t, 2, keyed.Len())
	})
}
