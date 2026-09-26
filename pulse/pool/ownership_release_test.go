package pool

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type releaseReplyHook struct {
	after   bool
	fired   atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (h *releaseReplyHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *releaseReplyHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *releaseReplyHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		args := cmd.Args()
		if len(args) < 2 || h.fired.Load() {
			return next(ctx, cmd)
		}
		script, ok := args[1].(string)
		if !ok || !((cmd.Name() == "evalsha" && script == luaReleaseJob.Hash()) ||
			(cmd.Name() == "eval" && strings.Contains(script, `local remaining = {}`))) {
			return next(ctx, cmd)
		}
		if h.entered != nil {
			close(h.entered)
			<-h.release
		}
		if h.after {
			if err := next(ctx, cmd); err != nil {
				return err
			}
		}
		h.fired.Store(true)
		return errors.New("injected release connection loss")
	}
}

func TestOwnershipReleaseReply(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "after"}[after], func(t *testing.T) {
			rdb := startTestRedis(t)
			n := addTestNode(t, rdb, "release-reply", WithWorkerTTL(time.Hour))
			w1, w2 := ownershipWorker(t, n, "w1"), ownershipWorker(t, n, "w2")
			first := &Job{Key: "k", Payload: []byte("first"), CreatedAt: time.Now(), NodeID: n.ID}
			require.NoError(t, w1.startJob(t.Context(), first))
			hook := &releaseReplyHook{after: after}
			rdb.AddHook(hook)
			require.ErrorContains(t, w1.stopJob(t.Context(), "k"), "connection loss")
			require.True(t, hook.fired.Load())
			require.Empty(t, w1.Jobs())
			require.Len(t, w1.pendingReleases, 1)
			second := &Job{Key: "k", Payload: []byte("second"), CreatedAt: time.Now(), NodeID: n.ID}
			if after {
				require.NoError(t, w2.startJob(t.Context(), second))
			}
			w1.retryReleases(t.Context())
			require.Empty(t, w1.pendingReleases)
			if !after {
				require.NoError(t, w2.startJob(t.Context(), second))
			}
			require.Len(t, w2.Jobs(), 1)
			require.Equal(t, "second", rdb.HGet(t.Context(), rmapContentKey(jobPayloadMapName(n.PoolName)), "k").Val())
		})
	}
}

// TestOwnerCleanupRebalance covers both orders of B4: cleanup before a stale
// rebalance, and rebalance before cleanup reads the old worker's keys.
func TestOwnerCleanupRebalance(t *testing.T) {
	for _, cleanupFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "rebalance first", true: "cleanup first"}[cleanupFirst], func(t *testing.T) {
			rdb := startTestRedis(t)
			n := sinklessNode(t, rdb, "cleanup-rebalance", WithWorkerTTL(time.Hour))
			w1, w2 := ownershipWorker(t, n, "w1"), ownershipWorker(t, n, "w2")
			job := &Job{Key: "k", Payload: []byte("p"), CreatedAt: time.Now(), NodeID: n.ID}
			require.NoError(t, w1.startJob(t.Context(), job))
			if cleanupFirst {
				require.NoError(t, rdb.HSet(t.Context(), rmapContentKey(workerKeepAliveMapName(n.PoolName)), w1.ID, "1").Err())
				n.cleanupWorker(t.Context(), w1.ID)
			}
			w1.rebalance(t.Context(), []string{w2.ID})
			if !cleanupFirst {
				require.NoError(t, rdb.HSet(t.Context(), rmapContentKey(workerKeepAliveMapName(n.PoolName)), w1.ID, "1").Err())
				n.cleanupWorker(t.Context(), w1.ID)
			}
			messages, err := rdb.XRange(t.Context(), n.poolStream.Key(), "-", "+").Result()
			require.NoError(t, err)
			for _, message := range messages {
				start, err := unmarshalJob([]byte(message.Values["p"].(string)))
				require.NoError(t, err)
				require.NoError(t, w2.startJob(t.Context(), start))
			}
			require.Len(t, w2.Jobs(), 1)
			require.Equal(t, "w2:2", rdb.HGet(t.Context(), n.ownersKey(), "k").Val())
		})
	}
}

func TestSlowHandlerKeepsLease(t *testing.T) {
	rdb := startTestRedis(t)
	n := addTestNode(t, rdb, "slow-handler-lease", WithWorkerTTL(400*time.Millisecond))
	h := &gatedHandler{entered: make(chan struct{}), release: make(chan struct{})}
	w, err := n.AddWorker(t.Context(), h)
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() {
		done <- w.startJob(t.Context(), &Job{Key: "k", CreatedAt: time.Now(), NodeID: n.ID})
	}()
	<-h.entered
	defer func() {
		close(h.release)
		require.NoError(t, <-done)
	}()
	key := rmapContentKey(workerKeepAliveMapName(n.PoolName))
	initial, err := rdb.HGet(t.Context(), key, w.ID).Int64()
	require.NoError(t, err)
	// A timestamp advancing by more than the whole lease proves that the
	// callback remained blocked across multiple independent renewals.
	require.Eventually(t, func() bool {
		latest, err := rdb.HGet(t.Context(), key, w.ID).Int64()
		return err == nil && latest-initial > int64(n.workerTTL)
	}, 3*time.Second, 10*time.Millisecond)
	n.cleanupWorker(t.Context(), w.ID)
	require.True(t, rdb.HExists(t.Context(), rmapContentKey(workerMapName(n.PoolName)), w.ID).Val())
	require.Equal(t, w.ID+":1", rdb.HGet(t.Context(), n.ownersKey(), "k").Val())
}

func TestStaleRebalanceDoesNotReviveCompletedJob(t *testing.T) {
	rdb := startTestRedis(t)
	n := sinklessNode(t, rdb, "stale-rebalance", WithWorkerTTL(time.Hour))
	w1, w2 := ownershipWorker(t, n, "w1"), ownershipWorker(t, n, "w2")
	first := &Job{Key: "k", Payload: []byte("old"), CreatedAt: time.Now(), NodeID: n.ID}
	require.NoError(t, w1.startJob(t.Context(), first))
	// Cleanup has transferred ownership while the old worker is paused.
	released, err := w1.releaseJob(t.Context(), first, false)
	require.NoError(t, err)
	require.True(t, released)
	require.NoError(t, w2.startJob(t.Context(), &Job{Key: "k", Payload: []byte("new"), NodeID: n.ID}))
	require.NoError(t, w2.stopJob(t.Context(), "k"))
	w1.rebalance(t.Context(), []string{w2.ID})
	require.Zero(t, rdb.XLen(t.Context(), n.poolStream.Key()).Val(), "stale worker resurrected a completed job")
	require.False(t, rdb.HExists(t.Context(), n.ownersKey(), "k").Val())
}

func TestSlowReleaseKeepsLease(t *testing.T) {
	rdb := startTestRedis(t)
	n := addTestNode(t, rdb, "slow-release-lease", WithWorkerTTL(400*time.Millisecond))
	w, err := n.AddWorker(t.Context(), newRecordingHandler())
	require.NoError(t, err)
	require.NoError(t, w.startJob(t.Context(), &Job{Key: "k", NodeID: n.ID, CreatedAt: time.Now()}))
	hook := &releaseReplyHook{entered: make(chan struct{}), release: make(chan struct{})}
	rdb.AddHook(hook)
	done := make(chan error, 1)
	go func() { done <- w.stopJob(t.Context(), "k") }()
	<-hook.entered
	defer func() {
		close(hook.release)
		require.ErrorContains(t, <-done, "connection loss")
	}()
	key := rmapContentKey(workerKeepAliveMapName(n.PoolName))
	initial, err := rdb.HGet(t.Context(), key, w.ID).Int64()
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		latest, err := rdb.HGet(t.Context(), key, w.ID).Int64()
		return err == nil && latest-initial > int64(n.workerTTL)
	}, 3*time.Second, 10*time.Millisecond)
	n.cleanupWorker(t.Context(), w.ID)
	require.True(t, rdb.HExists(t.Context(), rmapContentKey(workerMapName(n.PoolName)), w.ID).Val())
}
