package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOrphanSweepUsesOwnership(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(map[bool]string{false: "stale index hides orphan", true: "missing index hides owner"}[owned], func(t *testing.T) {
			rdb := startTestRedis(t)
			n := sinklessNode(t, rdb, "orphan-owner", WithWorkerTTL(time.Hour))
			w := ownershipWorker(t, n, "w1")
			job := &Job{Key: "k", Payload: []byte("p"), CreatedAt: time.Now(), NodeID: n.ID}
			require.NoError(t, w.startJob(t.Context(), job))
			if owned {
				// B5: payload arrives before the job index, even past the grace.
				_, err := n.jobMap.Delete(t.Context(), w.ID)
				require.NoError(t, err)
				require.Eventually(t, func() bool {
					return n.jobMap.Len() == 0
				}, time.Second, time.Millisecond)
			} else {
				// The index is stale in the opposite direction after release.
				require.NoError(t, rdb.HDel(t.Context(), n.ownersKey(), job.Key).Err())
				require.Eventually(t, func() bool {
					return n.jobMap.Len() == 1
				}, time.Second, time.Millisecond)
			}
			require.Eventually(t, func() bool {
				_, ok := n.JobPayload(job.Key)
				return ok
			}, time.Second, time.Millisecond)
			n.orphanedPayloads.Store(job.Key, time.Now().Add(-3*time.Hour).UnixNano())
			n.requeueOrphanedPayloads(t.Context())
			want := int64(1)
			if owned {
				want = 0
			}
			length, err := rdb.XLen(t.Context(), n.poolStream.Key()).Result()
			require.NoError(t, err)
			require.Equal(t, want, length)
		})
	}
}

func TestRequeueChecksCurrentOwnershipAndPayload(t *testing.T) {
	for _, state := range []string{"owned", "stopped", "replaced"} {
		t.Run(state, func(t *testing.T) {
			rdb := startTestRedis(t)
			n := sinklessNode(t, rdb, "requeue-current", WithWorkerTTL(time.Hour))
			w := ownershipWorker(t, n, "w1")
			job := &Job{Key: "k", Payload: []byte("old"), CreatedAt: time.Now(), NodeID: n.ID}
			require.NoError(t, w.startJob(t.Context(), job))
			switch state {
			case "stopped":
				require.NoError(t, w.stopJob(t.Context(), job.Key))
			case "replaced":
				released, err := w.releaseJob(t.Context(), job, false)
				require.NoError(t, err)
				require.True(t, released)
				require.NoError(t, rdb.HSet(t.Context(), rmapContentKey(jobPayloadMapName(n.PoolName)), job.Key, "new").Err())
			}
			queued, err := n.claimRequeue(t.Context(), job)
			require.NoError(t, err)
			require.False(t, queued, "stale snapshot must not enqueue old work")
			length, err := rdb.XLen(t.Context(), n.poolStream.Key()).Result()
			require.NoError(t, err)
			require.Zero(t, length)
		})
	}
}
