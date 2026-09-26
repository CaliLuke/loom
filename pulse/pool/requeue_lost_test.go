package pool

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLostRebalanceStartIsRecovered replays the requeue_live_asis_r7 TLC
// trace (pulse/pool/tla/cfg/requeue_live_asis_r7.cfg, 13 states, violates
// JobRecovered, issue #416). The model has a fixed set of workers, so a
// worker that comes back after a false death stands for w2 joining:
//
//  1. w1 runs the key, and w2 joins the pool, so the key now hashes to w2,
//  2. w1.rebalance stops the key and adds a start event for it to the pool
//     stream,
//  3. MAXLEN trims that event before any node delivers it, as it does when
//     maxQueuedJobs events are added while the pool sink lags,
//  4. a node routes pool events again.
//
// Rebalance used to leave the key in w1's job map entry. The orphan sweep
// skips a key that the job map lists, and cleanup skips a live worker, so the
// key ran nowhere for as long as w1 stayed in the pool. Rebalance must remove
// the key from w1's entry, so the orphan sweep requeues it and w2 starts it.
//
// Removing the entry alone lets the orphan sweep add a second start while the
// first is still queued (requeue_runner_release, AtMostOneRunner). The
// rebalance start therefore carries a guard, and the sweep must not requeue
// the key before that start is lost.
func TestLostRebalanceStartIsRecovered(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	const pool = "requeue-lost-rebalance"
	// The orphan sweep grace is max(2*workerTTL, ackGracePeriod) = 4s.
	opts := []NodeOption{WithWorkerTTL(2 * time.Second), WithAckGracePeriod(2 * time.Second)}
	nodeA := addTestNode(t, rdb, pool, opts...)
	handler1 := newRecordingHandler()
	w1, err := nodeA.AddWorker(ctx, handler1)
	require.NoError(t, err)

	// Step 1: the key hashes to the second worker once w2 joins.
	key := keysHashedTo(nodeA, 1, 2, 1)[0]
	require.NoError(t, nodeA.DispatchJob(ctx, key, []byte("payload")))
	require.Len(t, w1.Jobs(), 1)

	// The pool sink stops reading, so the rebalance start stays undelivered.
	nodeA.poolSink.Close(ctx)
	poolStreamKey := nodeA.poolStream.Key()
	before := countStartEvents(t, rdb.XRange(ctx, poolStreamKey, "-", "+").Val())[key]

	// Step 2: w2 joins and w1 rebalances the key.
	handler2 := newRecordingHandler()
	_, err = nodeA.AddWorker(ctx, handler2)
	require.NoError(t, err)
	// The worker map update that w2 causes can reach the watcher before
	// w2's keep-alive does, so rebalance explicitly once w2 is active.
	require.Eventually(t, func() bool {
		return len(nodeA.activeWorkers()) == 2
	}, 10*time.Second, 5*time.Millisecond)
	nodeA.handleWorkerMapUpdate(ctx)
	require.Eventually(t, func() bool {
		return handler1.wasStopped(key) &&
			countStartEvents(t, rdb.XRange(ctx, poolStreamKey, "-", "+").Val())[key] > before
	}, 10*time.Second, 5*time.Millisecond, "w1 did not rebalance %s", key)
	require.Empty(t, w1.Jobs())
	assert.Eventually(t, func() bool {
		keys, _ := nodeA.jobMap.GetValues(w1.ID)
		return !slices.Contains(keys, key)
	}, 5*time.Second, 5*time.Millisecond, "rebalance left %s in the job map entry of w1", key)

	// While the start is queued, the orphan sweep adds no second start, even
	// once its grace has passed.
	queued := countStartEvents(t, rdb.XRange(ctx, poolStreamKey, "-", "+").Val())[key]
	assert.Never(t, func() bool {
		return countStartEvents(t, rdb.XRange(ctx, poolStreamKey, "-", "+").Val())[key] > queued
	}, 8*time.Second, 50*time.Millisecond, "the orphan sweep requeued %s while its start was queued", key)

	// Step 3: the start event is trimmed before delivery.
	require.NoError(t, rdb.XTrimMaxLen(ctx, poolStreamKey, 0).Err())
	require.Zero(t, rdb.XLen(ctx, poolStreamKey).Val())

	// Step 4: a node without workers routes pool events.
	addTestNode(t, rdb, pool, opts...)
	require.Eventually(t, func() bool {
		_, ok := handler2.startedPayload(key)
		return ok
	}, 30*time.Second, 10*time.Millisecond, "w2 did not start %s", key)
}

// TestRebalanceRelease preserves the payload for recovery while removing
// the stopped job from the worker's entry. Releasing it again is a no-op.
func TestRebalanceRelease(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, "requeue-release-restart")
	handler := newRecordingHandler()
	w, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)
	require.NoError(t, node.DispatchJob(ctx, "k1", []byte("payload")))
	jobs := w.Jobs()
	require.Len(t, jobs, 1)
	listed := func(want bool) func() bool {
		return func() bool {
			keys, _ := node.jobMap.GetValues(w.ID)
			return slices.Contains(keys, "k1") == want
		}
	}
	require.Eventually(t, listed(true), 5*time.Second, 5*time.Millisecond)

	released, err := w.releaseTrackedJob(ctx, "k1")
	require.NoError(t, err)
	require.True(t, released)
	require.True(t, handler.wasStopped("k1"))
	require.Empty(t, w.Jobs())
	require.Eventually(t, listed(false), 5*time.Second, 5*time.Millisecond, "the released key stays in the job map entry")
	payload, ok := node.JobPayload("k1")
	require.True(t, ok, "the payload must stay for the orphan sweep")
	require.Equal(t, "payload", string(payload))

	released, err = w.releaseTrackedJob(ctx, "k1")
	require.NoError(t, err)
	require.False(t, released, "an untracked key is not released")
}

// TestClaimRequeue checks luaClaimRequeue beyond the delivery states of
// TestDispatchGuardInFlight: it adds the entry Stream.Add writes for the job
// with a guard "nowNanos:eventID" whose TTL has passed, refuses a second
// requeue while that event is in flight, replaces a guard whose event is not
// in flight whatever its TTL, and rejects a malformed guard.
func TestClaimRequeue(t *testing.T) {
	future := strconv.FormatInt(time.Now().Add(time.Hour).UnixNano(), 10)
	job := &Job{Key: "k1", Payload: []byte("p"), CreatedAt: time.Unix(0, 1)}
	t.Run("adds a guarded start", func(t *testing.T) {
		rdb, node := guardScriptNode(t, "requeue-claim")
		ctx := t.Context()
		before := time.Now().UnixNano()
		queued, err := node.claimRequeue(ctx, job)
		require.NoError(t, err)
		require.True(t, queued)

		guard, ok := pendingGuard(t, rdb, "requeue-claim")
		require.True(t, ok)
		until, eventID, err := parsePendingGuard(guard)
		require.NoError(t, err)
		require.GreaterOrEqual(t, until, before)
		require.LessOrEqual(t, until, time.Now().UnixNano(), "the guard TTL must have passed")
		addID, err := node.poolStream.Add(ctx, evStartJob, marshalJob(job))
		require.NoError(t, err)
		stream := node.poolStream.Key()
		scriptEntry := rdb.XRange(ctx, stream, eventID, eventID).Val()
		addEntry := rdb.XRange(ctx, stream, addID, addID).Val()
		require.Len(t, scriptEntry, 1)
		require.Len(t, addEntry, 1)
		require.Equal(t, addEntry[0].Values, scriptEntry[0].Values)

		queued, err = node.claimRequeue(ctx, job)
		require.NoError(t, err)
		require.False(t, queued, "requeued while the first start is in flight")
		again, ok := pendingGuard(t, rdb, "requeue-claim")
		require.True(t, ok)
		require.Equal(t, guard, again)
	})
	cases := []struct {
		name    string
		guard   func(t *testing.T, rdb *redis.Client, stream string) string
		wantErr string
	}{
		{
			name: "active TTL, acked event",
			guard: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				id := deliverEvent(t, rdb, stream)
				require.NoError(t, rdb.XAck(t.Context(), stream, poolSinkName, id).Err())
				return future + ":" + id
			},
		},
		{
			name: "legacy guard without event id",
			guard: func(*testing.T, *redis.Client, string) string {
				return future
			},
		},
		{
			name: "malformed",
			guard: func(*testing.T, *redis.Client, string) string {
				return "not-a-guard"
			},
			wantErr: "malformed pending guard",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb, node := guardScriptNode(t, "requeue-claim-replace")
			old := tc.guard(t, rdb, node.poolStream.Key())
			setGuard(t, node, "k1", old)

			inFlight, checkErr := node.requeueInFlight(t.Context(), "k1")
			queued, err := node.claimRequeue(t.Context(), job)
			if tc.wantErr != "" {
				require.ErrorContains(t, checkErr, tc.wantErr)
				require.ErrorContains(t, err, tc.wantErr)
				require.False(t, queued)
				return
			}
			require.NoError(t, checkErr)
			require.False(t, inFlight)
			require.NoError(t, err)
			require.True(t, queued)
			guard, ok := pendingGuard(t, rdb, "requeue-claim-replace")
			require.True(t, ok)
			require.NotEqual(t, old, guard)
		})
	}
}

// TestRebalanceWaitsForUnackedStart replays the requeue_owner_guard TLC trace
// (pulse/pool/tla/cfg/requeue_owner_guard.cfg, violates OwnerReached): w1
// starts the key, and w2 joins before the ack of the start deletes its
// guard. The guard is in flight, so a requeue would be refused. Rebalance
// must keep the job running on w1 without stopping it, and move it to w2
// once the guard is gone, although the worker map does not change again.
func TestRebalanceWaitsForUnackedStart(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	const pool = "requeue-unacked-start"
	opts := []NodeOption{WithWorkerTTL(2 * time.Second), WithAckGracePeriod(2 * time.Second)}
	node := addTestNode(t, rdb, pool, opts...)
	handler1 := newRecordingHandler()
	w1, err := node.AddWorker(ctx, handler1)
	require.NoError(t, err)
	key := keysHashedTo(node, 1, 2, 1)[0]
	require.NoError(t, node.DispatchJob(ctx, key, []byte("payload")))
	require.Len(t, w1.Jobs(), 1)

	// The ack of the start has not deleted the guard yet. A guard that
	// names a delivered event in neither the stream nor the pending list is
	// in flight (the Redis 7 purge state), and no router ever acks it.
	poolStreamKey := node.poolStream.Key()
	entries := rdb.XRange(ctx, poolStreamKey, "-", "+").Val()
	require.NotEmpty(t, entries)
	ms, _, ok := strings.Cut(entries[len(entries)-1].ID, "-")
	require.True(t, ok)
	msInt, err := strconv.ParseInt(ms, 10, 64)
	require.NoError(t, err)
	setGuard(t, node, key, staleUntil+":"+strconv.FormatInt(msInt-1, 10)+"-0")
	inFlight, err := node.requeueInFlight(ctx, key)
	require.NoError(t, err)
	require.True(t, inFlight)
	before := countStartEvents(t, rdb.XRange(ctx, poolStreamKey, "-", "+").Val())[key]

	handler2 := newRecordingHandler()
	_, err = node.AddWorker(ctx, handler2)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		return len(node.activeWorkers()) == 2
	}, 10*time.Second, 5*time.Millisecond)
	node.handleWorkerMapUpdate(ctx)

	// While the guard is in flight, the job keeps running on w1: no stop,
	// no restart, no second start event.
	assert.Never(t, func() bool {
		return handler1.wasStopped(key) ||
			countStartEvents(t, rdb.XRange(ctx, poolStreamKey, "-", "+").Val())[key] > before
	}, 3*time.Second, 20*time.Millisecond, "rebalance stopped or requeued %s while a start was in flight", key)
	require.Len(t, w1.Jobs(), 1)

	// The ack deletes the guard. Only the retry can move the job now.
	_, err = node.jobPendingMap.Delete(ctx, key)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		_, ok := handler2.startedPayload(key)
		return ok && handler1.wasStopped(key)
	}, 20*time.Second, 10*time.Millisecond, "w2 did not take over %s", key)
}
