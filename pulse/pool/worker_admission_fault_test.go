package pool

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming"
)

// TestWorkerAdmissionPrivateFailures checks rejected and uncertain operations
// without changing a healthy peer's membership, lease or private stream.
func TestWorkerAdmissionPrivateFailures(t *testing.T) {
	cases := []struct {
		name    string
		stage   string
		after   bool
		cancel  bool
		expired bool
	}{
		{name: "stream committed", stage: "stream", after: true},
		{name: "membership rejected", stage: "member"},
		{name: "membership committed", stage: "member", after: true},
		{name: "membership wait canceled", stage: "member", after: true, cancel: true},
		{name: "heartbeat wait canceled", stage: "heartbeat", after: true, cancel: true},
		{name: "lease expires before transfer", stage: "heartbeat", after: true, expired: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			node := addTestNode(t, rdb, "admission-fault", WithWorkerTTL(time.Second))
			peer, err := node.AddWorker(t.Context(), newRecordingHandler())
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			id := ""
			rdb.AddHook(constructionFailureHook{
				after: tc.after, succeed: tc.cancel || tc.expired,
				committed: func(redis.Cmder) {
					if tc.cancel {
						cancel()
					}
					if tc.expired {
						time.Sleep(800 * time.Millisecond)
					}
				},
				match: func(cmd redis.Cmder) bool {
					args := cmd.Args()
					switch tc.stage {
					case "stream":
						if cmd.Name() == "xadd" && len(args) > 1 && strings.Contains(fmt.Sprint(args[1]), ":stream:worker:") {
							key := fmt.Sprint(args[1])
							id = key[strings.LastIndexByte(key, ':')+1:]
							return true
						}
					case "member":
						if (cmd.Name() == "eval" || cmd.Name() == "evalsha") && len(args) == 9 && args[3] == rmapContentKey(workerMapName(node.PoolName)) {
							id = fmt.Sprint(args[5])
							return true
						}
					case "heartbeat":
						if len(args) == 7 && args[6] != peer.ID && ((cmd.Name() == "evalsha" && args[1] == luaOwnerHeartbeat.Hash()) ||
							(cmd.Name() == "eval" && strings.Contains(fmt.Sprint(args[1]), "worker is no longer registered"))) {
							id = fmt.Sprint(args[6])
							return true
						}
					}
					return false
				},
			})
			worker, err := node.AddWorker(ctx, newRecordingHandler())
			require.Error(t, err)
			require.Nil(t, worker)
			require.NotEmpty(t, id)
			switch {
			case tc.cancel:
				require.ErrorIs(t, err, context.Canceled)
			case tc.expired:
				require.ErrorContains(t, err, "initial worker lease expired")
			default:
				require.ErrorIs(t, err, errConstructionInjected)
			}
			for _, name := range []string{workerMapName(node.PoolName), workerKeepAliveMapName(node.PoolName)} {
				values, readErr := rdb.HGetAll(t.Context(), rmapContentKey(name)).Result()
				require.NoError(t, readErr)
				require.NotContains(t, values, id)
				require.Contains(t, values, peer.ID)
			}
			exists, err := rdb.Exists(t.Context(), "pulse:stream:"+workerStreamName(id)).Result()
			require.NoError(t, err)
			require.Zero(t, exists)
			exists, err = rdb.Exists(t.Context(), peer.stream.Key()).Result()
			require.NoError(t, err)
			require.EqualValues(t, 1, exists)
			require.Len(t, node.Workers(), 1)
		})
	}
}

// TestWorkerAdmissionCleanupErrors checks that every private withdrawal is
// attempted and reported even after an earlier cleanup operation fails.
func TestWorkerAdmissionCleanupErrors(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "admission-cleanup-errors")
	attempts := make(map[string]int)
	rdb.AddHook(constructionFailureHook{match: func(cmd redis.Cmder) bool {
		args := cmd.Args()
		if len(args) > 1 && (cmd.Name() == "xadd" || cmd.Name() == "del") && strings.Contains(fmt.Sprint(args[1]), ":stream:worker:") {
			attempts[cmd.Name()]++
			return true
		}
		if (cmd.Name() == "eval" || cmd.Name() == "evalsha") && len(args) == 8 {
			for _, name := range []string{workerMapName(node.PoolName), workerKeepAliveMapName(node.PoolName)} {
				if args[3] == rmapContentKey(name) {
					attempts[name]++
					return true
				}
			}
		}
		return false
	}})
	worker, err := node.AddWorker(t.Context(), newRecordingHandler())
	require.Nil(t, worker)
	require.ErrorIs(t, err, errConstructionInjected)
	for _, operation := range []string{"initialize worker stream", "withdraw worker stream", "withdraw worker membership", "withdraw worker heartbeat"} {
		require.ErrorContains(t, err, operation)
	}
	for _, operation := range []string{"xadd", "del", workerMapName(node.PoolName), workerKeepAliveMapName(node.PoolName)} {
		require.Positive(t, attempts[operation], operation)
	}
	require.Empty(t, node.Workers())
}

// TestWorkerAdmissionQueuedBeforeFailure retains the original pool event when
// routing reaches the prepared stream before a publication reply fails.
func TestWorkerAdmissionQueuedBeforeFailure(t *testing.T) {
	rdb := startTestRedis(t)
	node := sinklessNode(t, rdb, "admission-queued", WithWorkerTTL(5*time.Second))
	ctx := t.Context()
	require.NoError(t, rdb.XGroupCreate(ctx, node.poolStream.Key(), poolSinkName, "$").Err())
	job := &Job{Key: "queued", Payload: []byte("preserved"), NodeID: node.ID, CreatedAt: time.Now()}
	id, err := node.poolStream.Add(ctx, evStartJob, marshalTestOperation(t, node, job))
	require.NoError(t, err)
	batch, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{Group: poolSinkName, Consumer: "first", Streams: []string{node.poolStream.Key(), ">"}, Count: 1, Block: -1}).Result()
	require.NoError(t, err)
	require.Len(t, batch, 1)
	require.Equal(t, id, batch[0].Messages[0].ID)
	ev := &streaming.Event{ID: id, EventName: evStartJob, Payload: marshalTestOperation(t, node, job)}
	var injected atomic.Bool
	failedID := ""
	rdb.AddHook(constructionFailureHook{after: true, committed: func(cmd redis.Cmder) {
		injected.Store(true)
		failedID = fmt.Sprint(cmd.Args()[6])
		require.Eventually(t, func() bool {
			ids := node.activeWorkers()
			return len(ids) == 1 && ids[0] == failedID
		}, time.Second, time.Millisecond)
		require.NoError(t, node.routeWorkerEvent(ctx, ev))
	}, match: func(cmd redis.Cmder) bool {
		args := cmd.Args()
		return !injected.Load() && len(args) == 7 && ((cmd.Name() == "evalsha" && args[1] == luaOwnerHeartbeat.Hash()) ||
			(cmd.Name() == "eval" && strings.Contains(fmt.Sprint(args[1]), "worker is no longer registered")))
	}})
	failedHandler := newRecordingHandler()
	worker, err := node.AddWorker(ctx, failedHandler)
	require.ErrorIs(t, err, errConstructionInjected)
	require.Nil(t, worker)
	require.True(t, injected.Load())
	_, failedStarted := failedHandler.startedPayload(job.Key)
	require.False(t, failedStarted)
	pending, err := rdb.XPending(ctx, node.poolStream.Key(), poolSinkName).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, pending.Count)
	recovered, _, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: node.poolStream.Key(), Group: poolSinkName, Consumer: "recovery", Start: "0-0", MinIdle: 0, Count: 1}).Result()
	require.NoError(t, err)
	require.Len(t, recovered, 1)
	require.Equal(t, id, recovered[0].ID)
	handler := newRecordingHandler()
	healthy, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		ids := node.activeWorkers()
		return len(ids) == 1 && ids[0] == healthy.ID
	}, time.Second, time.Millisecond)
	require.NoError(t, node.routeWorkerEvent(ctx, ev))
	require.Eventually(t, func() bool {
		payload, ok := handler.startedPayload(job.Key)
		return ok && string(payload) == "preserved"
	}, time.Second, time.Millisecond)
	_, failedStarted = failedHandler.startedPayload(job.Key)
	require.False(t, failedStarted, "failed attempt must not handle its discarded copy")
}

// TestWorkerAdmissionFailedCleanupDoesNotRenew checks the uncertain-cleanup
// branch: stale publication never gains a runtime that renews or handles jobs.
func TestWorkerAdmissionFailedCleanupDoesNotRenew(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "admission-uncertain-cleanup", WithWorkerTTL(500*time.Millisecond))
	var heartbeats atomic.Int32
	id := ""
	rdb.AddHook(constructionFailureHook{after: true, committed: func(cmd redis.Cmder) {
		id = fmt.Sprint(cmd.Args()[6])
		heartbeats.Add(1)
	}, match: func(cmd redis.Cmder) bool {
		args := cmd.Args()
		return len(args) == 7 && ((cmd.Name() == "evalsha" && args[1] == luaOwnerHeartbeat.Hash()) ||
			(cmd.Name() == "eval" && strings.Contains(fmt.Sprint(args[1]), "worker is no longer registered")))
	}})
	rdb.AddHook(constructionFailureHook{match: func(cmd redis.Cmder) bool {
		args := cmd.Args()
		if cmd.Name() == "del" && len(args) > 1 && strings.Contains(fmt.Sprint(args[1]), ":stream:worker:") {
			return true
		}
		return (cmd.Name() == "eval" || cmd.Name() == "evalsha") && len(args) == 8 &&
			(args[3] == rmapContentKey(workerMapName(node.PoolName)) || args[3] == rmapContentKey(workerKeepAliveMapName(node.PoolName)))
	}})
	handler := newRecordingHandler()
	worker, err := node.AddWorker(t.Context(), handler)
	require.Nil(t, worker)
	require.ErrorIs(t, err, errConstructionInjected)
	require.ErrorContains(t, err, "withdraw worker heartbeat")
	require.NotEmpty(t, id)
	lease, err := rdb.HGet(t.Context(), rmapContentKey(workerKeepAliveMapName(node.PoolName)), id).Result()
	require.NoError(t, err)
	stream, err := node.getWorkerStream(id)
	require.NoError(t, err)
	job := &Job{Key: "must-not-start", Payload: []byte("discarded"), NodeID: node.ID, CreatedAt: time.Now()}
	_, err = stream.Add(t.Context(), evStartJob, marshalEnvelope(node.ID, marshalTestOperation(t, node, job)))
	require.NoError(t, err)
	// Cross the first renewal interval, before the initial lease expires.
	time.Sleep(300 * time.Millisecond)
	require.EqualValues(t, 1, heartbeats.Load())
	current, err := rdb.HGet(t.Context(), rmapContentKey(workerKeepAliveMapName(node.PoolName)), id).Result()
	require.NoError(t, err)
	require.Equal(t, lease, current)
	_, started := handler.startedPayload(job.Key)
	require.False(t, started)
	require.Empty(t, node.Workers())
}
