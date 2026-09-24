package pool

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// failingStartHandler is a JobHandler whose Start always fails.
type failingStartHandler struct{}

// Start implements JobHandler.
func (failingStartHandler) Start(*Job) error {
	return errors.New("start failed")
}

// Stop implements JobHandler.
func (failingStartHandler) Stop(string) error {
	return nil
}

// pendingGuard returns the pending guard stored in Redis for job k1.
func pendingGuard(t *testing.T, rdb *redis.Client, pool string) (string, bool) {
	t.Helper()
	guard, err := rdb.HGet(t.Context(), rmapContentKey(jobPendingMapName(pool)), "k1").Result()
	if errors.Is(err, redis.Nil) {
		return "", false
	}
	require.NoError(t, err)
	return guard, true
}

// TestDispatchGuardClearedAfterAckedStart covers dispatches whose start event
// is acked before the dispatch return arrives: the guard is cleared, so a
// retry after a start error is admitted at once, and a successful dispatch
// leaves no guard behind.
func TestDispatchGuardClearedAfterAckedStart(t *testing.T) {
	t.Run("start error then retry", func(t *testing.T) {
		rdb := startTestRedis(t)
		ctx := t.Context()
		node := addTestNode(t, rdb, "guard-start-error")
		_, err := node.AddWorker(ctx, failingStartHandler{})
		require.NoError(t, err)

		err = node.DispatchJob(ctx, "k1", nil)
		require.ErrorContains(t, err, "start failed")
		_, ok := pendingGuard(t, rdb, "guard-start-error")
		require.False(t, ok, "guard kept after an acked start error")

		err = node.DispatchJob(ctx, "k1", nil)
		require.ErrorContains(t, err, "start failed")
		require.NotErrorIs(t, err, ErrJobExists)
	})
	t.Run("successful start", func(t *testing.T) {
		rdb := startTestRedis(t)
		ctx := t.Context()
		node := addTestNode(t, rdb, "guard-start-ok")
		_, err := node.AddWorker(ctx, newRecordingHandler())
		require.NoError(t, err)

		require.NoError(t, node.DispatchJob(ctx, "k1", nil))
		_, ok := pendingGuard(t, rdb, "guard-start-ok")
		require.False(t, ok, "guard kept after an acked start")
	})
}

// TestDispatchGuardKeptWhileStartUnacked replays the double_noredeliver TLC
// trace (pulse/pool/tla/cfg/double_noredeliver.cfg, 8 states, violates
// NoDoubleStartSameWorker on main at 967f4fbe): DispatchJob adds s1, then
// times out or is cancelled and releases the guard while s1 is still queued,
// so a retry is admitted and adds s2. The node has no worker, so s1 stays
// unacked. Every retry must return ErrJobExists, including after the guard
// TTL, and the stale-guard sweep must keep the guard.
func TestDispatchGuardKeptWhileStartUnacked(t *testing.T) {
	cases := []struct {
		name  string
		first func(t *testing.T, node *Node)
	}{
		{
			name: "dispatcher timeout",
			first: func(t *testing.T, node *Node) {
				t.Helper()
				require.ErrorContains(t, node.DispatchJob(t.Context(), "k1", nil), "timed out")
			},
		},
		{
			name: "caller cancellation",
			first: func(t *testing.T, node *Node) {
				t.Helper()
				// Cancel only once the guard exists, so the start event was
				// queued before the caller gave up.
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				errc := make(chan error, 1)
				go func() {
					errc <- node.DispatchJob(ctx, "k1", nil)
				}()
				require.Eventually(t, func() bool {
					_, ok := pendingGuard(t, node.rdb, node.PoolName)
					return ok
				}, 5*time.Second, time.Millisecond)
				cancel()
				require.ErrorIs(t, <-errc, context.Canceled)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			const pool = "guard-unacked"
			node := addTestNode(t, rdb, pool, WithAckGracePeriod(200*time.Millisecond))

			tc.first(t, node)
			require.ErrorIs(t, node.DispatchJob(t.Context(), "k1", nil), ErrJobExists, "immediate retry")

			// After the guard TTL (2*ackGracePeriod) the event is still unacked.
			guard, ok := pendingGuard(t, rdb, pool)
			require.True(t, ok)
			time.Sleep(500 * time.Millisecond)
			require.ErrorIs(t, node.DispatchJob(t.Context(), "k1", nil), ErrJobExists, "retry after the guard TTL")

			require.Eventually(t, func() bool {
				v, ok := node.jobPendingMap.Get("k1")
				return ok && v == guard
			}, 5*time.Second, 5*time.Millisecond)
			node.cleanupStalePendingJobs(t.Context())
			got, ok := pendingGuard(t, rdb, pool)
			require.True(t, ok, "stale sweep cleared the guard of an unacked event")
			require.Equal(t, guard, got)
		})
	}
}
