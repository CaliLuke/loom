package pool

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestGuardObservationAfterCleanup(t *testing.T) {
	// This test owns cleanup timing; no background sweep should choose it.
	node := sinklessNode(t, startTestRedis(t), "guard-observation", WithAckGracePeriod(time.Hour))
	const key = "key"
	const guard = "1:1-0"
	var captured atomic.Bool
	node.rdb.AddHook(constructionFailureHook{
		after:   true,
		succeed: true,
		match: func(cmd redis.Cmder) bool {
			args := cmd.Args()
			return (cmd.Name() == "evalsha" || cmd.Name() == "eval") && len(args) > 6 &&
				fmt.Sprint(args[3]) == rmapContentKey(jobPendingMapName(node.PoolName)) &&
				fmt.Sprint(args[5]) == key && fmt.Sprint(args[6]) == guard &&
				captured.CompareAndSwap(false, true)
		},
		committed: func(redis.Cmder) {
			require.Eventually(t, func() bool {
				value, exists := node.jobPendingMap.Get(key)
				return exists && value == guard
			}, time.Second, time.Millisecond)
			node.cleanupStalePendingJobs(t.Context())
			require.Eventually(t, func() bool {
				_, exists := node.jobPendingMap.Get(key)
				return !exists
			}, time.Second, time.Millisecond)
		},
	})
	setGuard(t, node, key, guard)
	require.True(t, captured.Load())
	_, exists := node.jobPendingMap.Get(key)
	require.False(t, exists, "the observed write has already been superseded by legitimate cleanup")
}
