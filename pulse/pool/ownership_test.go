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
	require.NoError(t, startIdentifiedJob(t.Context(), w1, &job))
	require.NoError(t, startIdentifiedJob(t.Context(), w2, &copy))
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
	require.NoError(t, startIdentifiedJob(t.Context(), w, job))
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
		workerTTL: n.workerTTL, leaseWake: make(chan struct{}, 1),
	}
	w.recordLease(time.Now())
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
	require.NoError(t, startIdentifiedJob(t.Context(), w1, j1))
	require.Equal(t, "9007199254740993", j1.incarnation)
	require.Equal(t, uint64(9007199254740994), j1.Epoch)
	released, err := w1.releaseJob(t.Context(), j1, false)
	require.NoError(t, err)
	require.True(t, released)
	j2 := &Job{Key: "k", Payload: []byte("second"), CreatedAt: time.Now(), NodeID: n.ID}
	require.NoError(t, startIdentifiedJob(t.Context(), w2, j2))
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
	require.NoError(t, startIdentifiedJob(t.Context(), w, job))
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
	op, err := unmarshalOperation(raw)
	require.NoError(t, err)
	require.Equal(t, job.incarnation, op.token)
	decoded, err := unmarshalJob(op.data)
	require.NoError(t, err)
	require.Equal(t, job.Key, decoded.Key)
	require.Equal(t, job.Payload, decoded.Payload)
	require.Equal(t, n.ID, decoded.NodeID)
	decoded.incarnation = op.token
	require.Equal(t, marshalTestOperation(t, n, decoded), raw, "Lua encoding must match Go byte for byte")
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
				require.ErrorIs(t, startIdentifiedJob(ctx, w, &Job{Key: "k"}), ErrRequeue)
				require.Empty(t, w.Jobs())
			}
			require.NoError(t, rdb.HSet(ctx, keepalive, "old", "1").Err())
			require.NoError(t, n.Health(ctx))
		})
	}
}

// TestOwnershipProtocolShutdown rejects persisted older operations and retains
// monotonic identity after an explicitly completed shutdown.
func TestOwnershipProtocolShutdown(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	const pool = "protocol-shutdown"
	require.NoError(t, rdb.Set(ctx, pool+":protocol-version", "2", 0).Err())
	_, err := AddNode(ctx, pool, rdb)
	require.ErrorContains(t, err, "unsupported pool protocol")
	require.NoError(t, rdb.Del(ctx, pool+":protocol-version").Err())
	n := addTestNode(t, rdb, pool, WithWorkerTTL(time.Hour))
	require.NoError(t, rdb.HSet(ctx, pool+":owner-epochs", "k", "42").Err())
	require.NoError(t, n.Close(context.Background()))
	n.cleanupPool(ctx)
	require.EqualValues(t, 0, rdb.Exists(ctx, n.ownersKey(), pool+":protocol", pool+":protocol-version").Val())
	require.Equal(t, "42", rdb.HGet(ctx, pool+":owner-epochs", "k").Val())
}
