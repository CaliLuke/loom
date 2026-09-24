package pool

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// The tests in this file pin the Redis stream behaviours that the pool
// scripts rely on. They run against miniredis by default and against real
// Redis in the opt-in tier (`make test-pulse-redis`, issue #383). Where Redis
// versions differ, the expected behaviour depends on the server version.
// miniredis follows Redis 6.2 where the pool depends on it, such as keeping
// pending entries of deleted ids, though some replies use the Redis 7 shape.

// luaSingleIDPending runs the single-id XPENDING form that in_flight uses and
// returns the number of entries and the id of the first one, or the error
// reply.
var luaSingleIDPending = redis.NewScript(`
local pending = redis.pcall("XPENDING", KEYS[1], ARGV[1], ARGV[2], ARGV[2], 1)
if type(pending) == "table" and pending.err ~= nil then
   return {-1, pending.err}
end
if #pending == 0 then
   return {0, ""}
end
return {#pending, pending[1][1]}
`)

// luaGroupInfo returns the flat field list that XINFO GROUPS reports for the
// group named ARGV[1], as in_flight reads it.
var luaGroupInfo = redis.NewScript(`
for _, info in ipairs(redis.call("XINFO", "GROUPS", KEYS[1])) do
   for i = 1, #info, 2 do
      if info[i] == "name" and info[i + 1] == ARGV[1] then
         return info
      end
   end
end
return {}
`)

// TestRedisSingleIDPending checks the extended XPENDING form for a single id,
// called from Lua: it lists only that id while it is pending, including after
// XDEL and XTRIM removed the entry, and fails with NOGROUP without a group.
func TestRedisSingleIDPending(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	const stream, group = "semantics:pending", "g"
	ids := deliverEntries(t, rdb, stream, group, "c1", 4)
	require.NoError(t, rdb.XAck(ctx, stream, group, ids[1]).Err())
	require.NoError(t, rdb.XDel(ctx, stream, ids[2]).Err())
	require.NoError(t, rdb.XTrimMaxLen(ctx, stream, 0).Err())
	require.Zero(t, rdb.XLen(ctx, stream).Val(), "the stream is empty")

	cases := []struct {
		name      string
		stream    string
		id        string
		wantCount int64
	}{
		{name: "pending, trimmed", stream: stream, id: ids[0], wantCount: 1},
		{name: "acked", stream: stream, id: ids[1], wantCount: 0},
		{name: "pending, deleted", stream: stream, id: ids[2], wantCount: 1},
		{name: "pending, last", stream: stream, id: ids[3], wantCount: 1},
		{name: "never added", stream: stream, id: "1-1", wantCount: 0},
		{name: "no group", stream: "semantics:nogroup", id: ids[0], wantCount: -1},
	}
	require.NoError(t, rdb.XAdd(ctx, &redis.XAddArgs{Stream: "semantics:nogroup", Values: []any{"n", "x"}}).Err())
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := luaSingleIDPending.Run(ctx, rdb, []string{tc.stream}, group, tc.id).Slice()
			require.NoError(t, err)
			require.Len(t, res, 2)
			require.Equal(t, tc.wantCount, res[0])
			switch tc.wantCount {
			case 1:
				require.Equal(t, tc.id, res[1])
			case -1:
				require.Contains(t, res[1], "NOGROUP")
			}
		})
	}
}

// TestRedisXInfoGroups checks the XINFO GROUPS reply that in_flight parses in
// Lua and that go-redis parses: name, pending and last-delivered-id on every
// version, plus entries-read and lag on Redis 7 and later.
func TestRedisXInfoGroups(t *testing.T) {
	srv := startTestServer(t)
	rdb := srv.Client
	ctx := t.Context()
	const stream, group = "semantics:groups", "g"
	delivered := deliverEntries(t, rdb, stream, group, "c1", 2)
	for range 3 {
		require.NoError(t, rdb.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: []any{"n", "x"}}).Err())
	}
	lastDelivered := delivered[len(delivered)-1]

	flat, err := luaGroupInfo.Run(ctx, rdb, []string{stream}, group).Slice()
	require.NoError(t, err)
	require.Zero(t, len(flat)%2, "XINFO GROUPS fields come in pairs: %v", flat)
	fields := make(map[string]any, len(flat)/2)
	for i := 0; i < len(flat); i += 2 {
		name, ok := flat[i].(string)
		require.True(t, ok, "field name %v", flat[i])
		fields[name] = flat[i+1]
	}
	require.Equal(t, group, fields["name"])
	require.Equal(t, int64(2), fields["pending"])
	require.Equal(t, lastDelivered, fields["last-delivered-id"])
	_, hasEntriesRead := fields["entries-read"]
	_, hasLag := fields["lag"]
	if srv.MajorVersion() >= 7 {
		require.True(t, hasEntriesRead, "entries-read on Redis %d", srv.MajorVersion())
		require.True(t, hasLag, "lag on Redis %d", srv.MajorVersion())
		require.Equal(t, int64(2), fields["entries-read"])
		require.Equal(t, int64(3), fields["lag"])
	}
	if srv.Real() && srv.MajorVersion() < 7 {
		require.False(t, hasEntriesRead, "entries-read on Redis %d", srv.MajorVersion())
		require.False(t, hasLag, "lag on Redis %d", srv.MajorVersion())
	}

	groups, err := rdb.XInfoGroups(ctx, stream).Result()
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, group, groups[0].Name)
	require.Equal(t, int64(2), groups[0].Pending)
	require.Equal(t, lastDelivered, groups[0].LastDeliveredID)
}

// TestRedisClaimDispatchApproxMaxLen checks the XADD ... MAXLEN ~ that
// luaClaimDispatch runs: the pool stream stays near maxQueuedJobs, and the
// newest start event is kept. Approximate trimming removes whole stream
// nodes only, so the stream can exceed the bound by one node
// (stream-node-max-entries, 100 by default).
func TestRedisClaimDispatchApproxMaxLen(t *testing.T) {
	const maxQueued, dispatches, nodeEntries = 10, 500, 100
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := sinklessNode(t, rdb, "semantics-maxlen", WithMaxQueuedJobs(maxQueued))

	var lastID string
	for i := range dispatches {
		key := "k" + strconv.Itoa(i)
		_, id, err := node.claimDispatch(ctx, key, marshalJobKey(key))
		require.NoError(t, err)
		lastID = id
	}
	length, err := rdb.XLen(ctx, node.poolStream.Key()).Result()
	require.NoError(t, err)
	require.GreaterOrEqual(t, length, int64(maxQueued))
	require.LessOrEqual(t, length, int64(maxQueued+nodeEntries))
	require.Len(t, rdb.XRange(ctx, node.poolStream.Key(), lastID, lastID).Val(), 1, "the newest start event")
}

// TestRedisAutoClaimDeletedEntries checks how XAUTOCLAIM treats pending
// entries whose ids were deleted from the stream, by XDEL or by a trim. Redis
// 6.2 keeps them in the pending list, claims them and replies with a null
// entry for each; Redis 7 and later purge them and report their ids in a
// third reply element. miniredis keeps them as Redis 6.2 does. Issue #385
// hinges on this difference.
func TestRedisAutoClaimDeletedEntries(t *testing.T) {
	srv := startTestServer(t)
	rdb := srv.Client
	ctx := t.Context()
	const stream, group = "semantics:autoclaim", "g"
	ids := deliverEntries(t, rdb, stream, group, "c1", 3)
	require.NoError(t, rdb.XDel(ctx, stream, ids[0]).Err())
	require.NoError(t, rdb.XTrimMinID(ctx, stream, ids[2]).Err())
	purges := srv.MajorVersion() >= 7

	raw, err := rdb.Do(ctx, "XAUTOCLAIM", stream, group, "c2", 0, "0-0").Slice()
	require.NoError(t, err)
	switch {
	case purges:
		require.Len(t, raw, 3)
		require.ElementsMatch(t, []any{ids[0], ids[1]}, raw[2], "deleted ids")
	case srv.Real():
		require.Len(t, raw, 2, "Redis 6.2 reports no deleted ids")
	default:
		// miniredis replies in the Redis 7 shape but reports no deleted
		// ids, as it keeps their pending entries like Redis 6.2.
		require.Len(t, raw, 3)
		require.Empty(t, raw[2], "deleted ids")
	}

	pending, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{Stream: stream, Group: group, Start: "-", End: "+", Count: 10}).Result()
	require.NoError(t, err)
	got := make(map[string]string, len(pending))
	for _, p := range pending {
		got[p.ID] = p.Consumer
	}
	require.Equal(t, "c2", got[ids[2]], "the live entry is claimed")
	for _, id := range ids[:2] {
		owner, kept := got[id]
		require.Equal(t, !purges, kept, "pending entry of deleted id %s kept on Redis %d (0 is miniredis)", id, srv.MajorVersion())
		if kept && srv.Real() {
			// Redis 6.2 claims the entry. miniredis leaves it with its
			// owner, which the pool does not rely on.
			require.Equal(t, "c2", owner, "owner of deleted id %s", id)
		}
	}
}

// TestRedisDispatchGuardAfterAutoClaimOfTrimmedStart checks the dispatch
// guard of a start event that was delivered, trimmed while unacked, and then
// reached by XAUTOCLAIM, as after ackGracePeriod on a busy pool stream.
//
// This pins the known gap of issue #385. On Redis 6.2 (and miniredis) the
// pending entry survives XAUTOCLAIM, so the guard stays. On Redis 7 and later
// XAUTOCLAIM purges it, so the guard is released before pendingEventTTL
// although the start may still be queued on a worker stream. The fix for
// #385 must make the guard stay on every version and change this test.
func TestRedisDispatchGuardAfterAutoClaimOfTrimmedStart(t *testing.T) {
	srv := startTestServer(t)
	rdb := srv.Client
	ctx := t.Context()
	node := sinklessNode(t, rdb, "semantics-guard-autoclaim")
	stream := node.poolStream.Key()
	id := deliverEvent(t, rdb, stream)
	trimPending(t, rdb, stream, id)
	guard := staleUntil + ":" + id
	setGuard(t, node, "k2", guard)

	// The raw command, because go-redis XAutoClaim cannot parse the Redis
	// 6.2 reply for a deleted id. The sink parses it too (issue #408).
	require.NoError(t, rdb.Do(ctx, "XAUTOCLAIM", stream, poolSinkName, "c2", 0, "0-0").Err())
	status, err := node.runReleaseDispatch(ctx, "k2", guard)
	require.NoError(t, err)

	want := dispatchReleaseInFlight
	if srv.MajorVersion() >= 7 {
		// Known gap, issue #385: the fix must flip this to
		// dispatchReleaseInFlight on every version.
		want = dispatchReleaseDeleted
	}
	require.Equal(t, want, status, "release status on Redis %d (0 is miniredis); see issue #385", srv.MajorVersion())
}

// TestRedisClaimDispatchAtomicUnderConcurrency races concurrent claims of the
// same keys over separate connections. luaClaimDispatch is one atomic script,
// so each key is claimed exactly once, gets exactly one start event, and
// keeps the guard of the winning claim.
func TestRedisClaimDispatchAtomicUnderConcurrency(t *testing.T) {
	const keys, claimers = 20, 32
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := sinklessNode(t, rdb, "semantics-atomic")

	type result struct {
		guard, eventID string
		err            error
	}
	results := make([][]result, keys)
	var wg sync.WaitGroup
	for k := range keys {
		results[k] = make([]result, claimers)
		key := "k" + strconv.Itoa(k)
		for c := range claimers {
			wg.Go(func() {
				guard, eventID, err := node.claimDispatch(ctx, key, marshalJobKey(key))
				results[k][c] = result{guard: guard, eventID: eventID, err: err}
			})
		}
	}
	wg.Wait()

	events := make(map[string]bool)
	for k := range keys {
		key := "k" + strconv.Itoa(k)
		var winners []result
		for _, r := range results[k] {
			if r.err != nil {
				require.ErrorIs(t, r.err, ErrJobExists, key)
				continue
			}
			winners = append(winners, r)
		}
		require.Len(t, winners, 1, "claims of %s", key)
		stored, err := rdb.HGet(ctx, rmapContentKey(jobPendingMapName(node.PoolName)), key).Result()
		require.NoError(t, err)
		require.Equal(t, winners[0].guard, stored, "guard of %s", key)
		events[winners[0].eventID] = true
	}
	entries, err := rdb.XRange(ctx, node.poolStream.Key(), "-", "+").Result()
	require.NoError(t, err)
	require.Len(t, entries, keys, "one start event per key")
	for _, e := range entries {
		require.True(t, events[e.ID], "start event %s has a winning guard", e.ID)
	}
}

// deliverEntries adds n entries to stream, creates group at the start of the
// stream, and delivers them all to consumer without acking them. It returns
// their ids in order.
func deliverEntries(t *testing.T, rdb *redis.Client, stream, group, consumer string, n int) []string {
	t.Helper()
	ctx := t.Context()
	ids := make([]string, n)
	for i := range n {
		id, err := rdb.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: []any{"n", fmt.Sprint(i)}}).Result()
		require.NoError(t, err)
		ids[i] = id
	}
	require.NoError(t, rdb.XGroupCreate(ctx, stream, group, "0").Err())
	res, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    int64(n),
		Block:    -1,
	}).Result()
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Len(t, res[0].Messages, n)
	return ids
}
