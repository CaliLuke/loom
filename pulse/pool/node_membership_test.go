package pool

import (
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
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

			_, gotA := nodeA.claimCleanupLock(ctx, workerID, tsA, okA)
			_, gotB := nodeB.claimCleanupLock(ctx, workerID, tsB, okB)

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

// TestEvictionKeepsCleanupLockOfAnotherHolder replays the lock_asis TLC trace
// (pulse/pool/tla/cfg/lock_asis.cfg, 11 states, violates
// CleanupLockMutualExclusion on main at 967f4fbe). Eviction deleted the
// cleanup lock without checking its holder, so a node that had observed no
// lock could acquire it while another node still held it. Only the holder's
// token now releases the lock.
func TestEvictionKeepsCleanupLockOfAnotherHolder(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	nodeA := addTestNode(t, rdb, "cleanup-lock-evict")
	nodeB := addTestNode(t, rdb, "cleanup-lock-evict")
	const workerID = "w1"

	// States 3-4: both nodes begin cleanup of w1 and observe no lock.
	_, seenA := nodeA.workerCleanupMap.Get(workerID)
	tsB, seenB := nodeB.workerCleanupMap.Get(workerID)
	require.False(t, seenA)
	require.False(t, seenB)

	// States 5-6: nodeA acquires the lock, finishes cleanup and deletes w1,
	// which releases its own lock.
	token, ok := nodeA.claimCleanupLock(ctx, workerID, "", false)
	require.True(t, ok)
	require.NoError(t, nodeA.deleteWorker(ctx, workerID, token))

	// States 7-9: a stale replica shows w1 again; nodeA observes no lock and
	// acquires it a second time.
	held, ok := nodeA.claimCleanupLock(ctx, workerID, "", false)
	require.True(t, ok)

	// State 10: nodeA's eviction path deletes w1 without holding the lock.
	require.NoError(t, nodeA.deleteWorker(ctx, workerID, ""))

	// State 11: nodeB acts on the lock value it observed in state 4. The lock
	// nodeA still holds must refuse it.
	_, ok = nodeB.claimCleanupLock(ctx, workerID, tsB, seenB)
	require.False(t, ok, "two nodes hold the cleanup lock")
	got, err := rdb.HGet(ctx, rmapContentKey(workerCleanupMapName("cleanup-lock-evict")), workerID).Result()
	require.NoError(t, err)
	require.Equal(t, held, got)
}

// TestCleanupLockReleaseRequiresHolderToken checks that deleteWorker releases
// the cleanup lock only for the token that currently holds it.
func TestCleanupLockReleaseRequiresHolderToken(t *testing.T) {
	cases := []struct {
		name     string
		token    func(held string) string
		released bool
	}{
		{name: "holder token", token: func(held string) string { return held }, released: true},
		{name: "other token", token: func(string) string { return "1" }, released: false},
		{name: "no token", token: func(string) string { return "" }, released: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			ctx := t.Context()
			node := addTestNode(t, rdb, "cleanup-lock-release")
			const workerID = "w1"
			held, ok := node.claimCleanupLock(ctx, workerID, "", false)
			require.True(t, ok)

			require.NoError(t, node.deleteWorker(ctx, workerID, tc.token(held)))

			got, err := rdb.HGet(ctx, rmapContentKey(workerCleanupMapName("cleanup-lock-release")), workerID).Result()
			if tc.released {
				require.ErrorIs(t, err, redis.Nil)
				return
			}
			require.NoError(t, err)
			require.Equal(t, held, got)
		})
	}
}
