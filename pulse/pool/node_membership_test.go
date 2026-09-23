package pool

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestClaimCleanupLockSingleWinnerAcrossNodes verifies that when two nodes
// observe the same cleanup lock value in their local replicas, at most one of
// them acquires the lock for the worker.
func TestClaimCleanupLockSingleWinnerAcrossNodes(t *testing.T) {
	cases := []struct {
		name        string
		seed        string
		wantWinners int
	}{
		{
			name:        "stale lock",
			seed:        strconv.FormatInt(time.Now().Add(-time.Hour).UnixNano(), 10),
			wantWinners: 1,
		},
		{
			name:        "malformed lock",
			seed:        "not-a-timestamp",
			wantWinners: 1,
		},
		{
			name:        "valid lock",
			seed:        strconv.FormatInt(time.Now().Add(time.Hour).UnixNano(), 10),
			wantWinners: 0,
		},
		{
			name:        "no lock",
			wantWinners: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			nodeA := addTestNode(t, rdb, "cleanup-lock")
			nodeB := addTestNode(t, rdb, "cleanup-lock")
			const workerID = "stale-worker"
			ctx := t.Context()

			if tc.seed != "" {
				_, err := nodeA.workerCleanupMap.Set(ctx, workerID, tc.seed)
				require.NoError(t, err)
				for _, node := range []*Node{nodeA, nodeB} {
					require.Eventually(t, func() bool {
						v, ok := node.workerCleanupMap.Get(workerID)
						return ok && v == tc.seed
					}, 5*time.Second, 10*time.Millisecond)
				}
			}

			// Both nodes observe the lock before either acts.
			tsA, okA := nodeA.workerCleanupMap.Get(workerID)
			tsB, okB := nodeB.workerCleanupMap.Get(workerID)

			gotA := nodeA.claimCleanupLock(ctx, workerID, tsA, okA)
			gotB := nodeB.claimCleanupLock(ctx, workerID, tsB, okB)

			winners := 0
			for _, got := range []bool{gotA, gotB} {
				if got {
					winners++
				}
			}
			require.Equal(t, tc.wantWinners, winners, "nodeA=%v nodeB=%v", gotA, gotB)
		})
	}
}
