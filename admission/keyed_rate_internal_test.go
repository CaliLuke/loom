package admission

import (
	"context"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

// TestKeyedRateChurnWorkIsBounded rotates new keys through a full KeyedRate at
// maxKeys * PerSecond keys per second, so buckets refill at staggered times and
// each new key evicts about one. Each Allow must examine only the buckets it
// evicts, not every stored bucket.
func TestKeyedRateChurnWorkIsBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const maxKeys = 2000
		const interval = time.Second / maxKeys
		keyed, err := NewKeyedRate(RatePolicy{PerSecond: 1, Burst: 1}, maxKeys)
		require.NoError(t, err)
		ctx := context.Background()
		for i := range maxKeys {
			decision, err := keyed.Allow(ctx, fmt.Sprintf("fill-%d", i))
			require.NoError(t, err)
			require.True(t, decision.Allowed)
			time.Sleep(interval)
		}
		keyed.inspected = 0
		const churn = 4 * maxKeys
		admitted := 0
		for i := range churn {
			decision, err := keyed.Allow(ctx, fmt.Sprintf("churn-%d", i))
			require.NoError(t, err)
			if decision.Allowed {
				admitted++
			}
			time.Sleep(interval)
		}
		require.Equal(t, churn, admitted, "each new key finds a full bucket to evict")
		require.LessOrEqual(t, keyed.inspected, uint64(2*churn), "eviction must not scan every stored bucket")
		require.LessOrEqual(t, keyed.Len(), maxKeys)
	})
}
