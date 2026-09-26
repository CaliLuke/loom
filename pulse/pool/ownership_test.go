package pool

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestOwnershipAcrossWorkers replays runner_asis: two routers with different
// membership snapshots deliver the same start to different workers.
func TestOwnershipAcrossWorkers(t *testing.T) {
	rdb := startTestRedis(t)
	n1 := addTestNode(t, rdb, "owners", WithWorkerTTL(time.Hour))
	n2 := addTestNode(t, rdb, "owners", WithWorkerTTL(time.Hour))
	w1 := ownershipWorker(t, n1, "w1")
	w2 := ownershipWorker(t, n2, "w2")
	job := Job{Key: "key", Payload: []byte("original"), CreatedAt: time.Now(), NodeID: n1.ID}
	copy := job
	require.NoError(t, w1.startJob(t.Context(), &job))
	require.NoError(t, w2.startJob(t.Context(), &copy))
	require.Len(t, w1.Jobs(), 1)
	require.Empty(t, w2.Jobs(), "a second worker must not start an owned key")
}

// TestCleanupRechecksLiveness replays redesign_fence_norecheck: cleanup was
// selected from a stale replica, but Redis already holds a fresh keep-alive.
func TestCleanupRechecksLiveness(t *testing.T) {
	rdb := startTestRedis(t)
	n := addTestNode(t, rdb, "cleanup-recheck", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	job := &Job{Key: "key", Payload: []byte("payload"), CreatedAt: time.Now(), NodeID: n.ID}
	require.NoError(t, w.startJob(t.Context(), job))
	n.cleanupWorker(t.Context(), w.ID)
	require.True(t, rdb.HExists(t.Context(), rmapContentKey(workerMapName(n.PoolName)), w.ID).Val(), "fresh worker must survive cleanup")
	require.EqualValues(t, 0, rdb.XLen(t.Context(), n.poolStream.Key()).Val(), "cleanup must not requeue a live worker's job")
}

func ownershipWorker(t *testing.T, n *Node, id string) *Worker {
	t.Helper()
	registerFakeWorker(t, n, id)
	w := &Worker{
		ID: id, node: n, handler: newRecordingHandler(), logger: n.logger,
		jobsMap: n.jobMap, jobPayloadsMap: n.jobPayloadMap,
		done: make(chan struct{}), runtimeCtx: t.Context(),
	}
	t.Cleanup(func() {
		close(w.done)
		w.wg.Wait()
	})
	return w
}

// TestOwnershipEpochRelease ensures an old worker cannot erase a new owner's
// payload, including epochs beyond Lua's exact integer range.
func TestOwnershipEpochRelease(t *testing.T) {
	rdb := startTestRedis(t)
	n := addTestNode(t, rdb, "epochs", WithWorkerTTL(time.Hour))
	w1, w2 := ownershipWorker(t, n, "w1"), ownershipWorker(t, n, "w2")
	require.NoError(t, rdb.HSet(t.Context(), n.PoolName+":owner-epochs", "k", "9007199254740992").Err())
	j1 := &Job{Key: "k", Payload: []byte("first"), CreatedAt: time.Now(), NodeID: n.ID}
	require.NoError(t, w1.startJob(t.Context(), j1))
	require.Equal(t, uint64(9007199254740993), j1.Epoch)
	released, err := w1.releaseJob(t.Context(), j1, false)
	require.NoError(t, err)
	require.True(t, released)
	j2 := &Job{Key: "k", Payload: []byte("second"), CreatedAt: time.Now(), NodeID: n.ID}
	require.NoError(t, w2.startJob(t.Context(), j2))
	require.Equal(t, j1.Epoch+1, j2.Epoch)
	released, err = w1.releaseJob(t.Context(), j1, true)
	require.NoError(t, err)
	require.False(t, released)
	require.Equal(t, "second", rdb.HGet(t.Context(), rmapContentKey(jobPayloadMapName(n.PoolName)), "k").Val())
}

// TestCleanupUsesOwners covers the lost-job replica trace: the jobs replica
// and even its Redis index lack the key, but authoritative ownership survives.
func TestCleanupUsesOwners(t *testing.T) {
	rdb := startTestRedis(t)
	n := addTestNode(t, rdb, "cleanup-owners", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	job := &Job{Key: "key,with:punctuation", Payload: []byte{0, 1, 255}, CreatedAt: time.Now(), NodeID: n.ID}
	require.NoError(t, w.startJob(t.Context(), job))
	require.NoError(t, rdb.HDel(t.Context(), rmapContentKey(jobMapName(n.PoolName)), w.ID).Err())
	require.NoError(t, rdb.HSet(t.Context(), rmapContentKey(workerKeepAliveMapName(n.PoolName)), w.ID, "1").Err())
	token, ok := n.acquireCleanupLock(t.Context(), w.ID)
	require.True(t, ok)
	status, err := n.cleanupOwnedJobs(t.Context(), w.ID, "wrong-token")
	require.NoError(t, err)
	require.EqualValues(t, 0, status)
	require.True(t, rdb.HExists(t.Context(), n.ownersKey(), job.Key).Val())
	status, err = n.cleanupOwnedJobs(t.Context(), w.ID, token)
	require.NoError(t, err)
	require.EqualValues(t, 2, status)
	require.False(t, rdb.HExists(t.Context(), n.ownersKey(), job.Key).Val())
	messages, err := rdb.XRange(t.Context(), n.poolStream.Key(), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)
	raw := []byte(messages[0].Values["p"].(string))
	decoded, err := unmarshalJob(raw)
	require.NoError(t, err)
	require.Equal(t, job.Key, decoded.Key)
	require.Equal(t, job.Payload, decoded.Payload)
	require.Equal(t, n.ID, decoded.NodeID)
	require.Equal(t, marshalJob(decoded), raw, "Lua encoding must match Go byte for byte")
	require.NotEmpty(t, rdb.HGet(t.Context(), rmapContentKey(jobPendingMapName(n.PoolName)), job.Key).Val())
	status, err = n.cleanupOwnedJobs(t.Context(), w.ID, token)
	require.NoError(t, err)
	require.EqualValues(t, 0, status)
	require.EqualValues(t, 1, rdb.XLen(t.Context(), n.poolStream.Key()).Val())
}

func TestOwnershipProtocol(t *testing.T) {
	for _, clientOnly := range []bool{false, true} {
		t.Run(strconv.FormatBool(clientOnly), func(t *testing.T) {
			rdb := startTestRedis(t)
			ctx := t.Context()
			const pool = "protocol"
			keepalive := rmapContentKey(nodeKeepAliveMapName(pool))
			require.NoError(t, rdb.HSet(ctx, keepalive, "old", strconv.FormatInt(time.Now().UnixNano(), 10)).Err())
			opts := []NodeOption{WithWorkerTTL(time.Hour)}
			if clientOnly {
				opts = append(opts, WithClientOnly())
			}
			_, err := AddNode(ctx, pool, rdb, opts...)
			require.ErrorContains(t, err, "incompatible pool node old")
			require.NoError(t, rdb.HDel(ctx, keepalive, "old").Err())
			n := addTestNode(t, rdb, pool, opts...)
			require.NoError(t, n.Health(ctx))
			require.NoError(t, rdb.HSet(ctx, keepalive, "old", strconv.FormatInt(time.Now().UnixNano(), 10)).Err())
			require.ErrorContains(t, n.Health(ctx), "incompatible pool node")
			if !clientOnly {
				w := ownershipWorker(t, n, "w1")
				require.ErrorIs(t, w.startJob(ctx, &Job{Key: "k"}), ErrRequeue)
				require.Empty(t, w.Jobs())
			}
			require.NoError(t, rdb.HSet(ctx, keepalive, "old", "1").Err())
			require.NoError(t, n.Health(ctx))
		})
	}
}

func TestOwnerBackfillAndShutdown(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	const pool = "backfill"
	require.NoError(t, rdb.HSet(ctx, rmapContentKey(jobMapName(pool)), "alive", `["k"]`, "dead", `["lost"]`).Err())
	require.NoError(t, rdb.HSet(ctx, rmapContentKey(workerKeepAliveMapName(pool)), "alive", strconv.FormatInt(time.Now().UnixNano(), 10), "dead", "1").Err())
	n := addTestNode(t, rdb, pool, WithWorkerTTL(time.Hour))
	require.Equal(t, "alive:1", rdb.HGet(ctx, n.ownersKey(), "k").Val())
	require.False(t, rdb.HExists(ctx, n.ownersKey(), "lost").Val())
	require.NoError(t, rdb.HDel(ctx, n.ownersKey(), "k").Err())
	require.NoError(t, joinOwnershipProtocol(ctx, n))
	require.False(t, rdb.HExists(ctx, n.ownersKey(), "k").Val(), "marker prevents repeated backfill")
	require.NoError(t, n.Close(context.Background()))
	n.cleanupPool(ctx)
	require.EqualValues(t, 0, rdb.Exists(ctx, n.ownersKey(), pool+":protocol", pool+":protocol-version").Val())
	require.Equal(t, "1", rdb.HGet(ctx, pool+":owner-epochs", "k").Val())
}
