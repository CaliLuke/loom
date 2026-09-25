package streaming

import (
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/internal/redistest"
)

// claimTestMinIdle is the minimum idle time of the claimIdle tests that age
// entries. On a real server the tests wait that long.
const claimTestMinIdle = 300 * time.Millisecond

// TestClaimIdleBatches checks claimIdle against Redis: it claims the idle
// entries of every consumer in batches, acks the pending entries of deleted
// events, counts each claim as a delivery, and leaves entries that are not
// idle.
func TestClaimIdleBatches(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	const key, group = "claim-idle-batches", "g"
	require.NoError(t, rdb.XGroupCreateMkStream(ctx, key, group, "0").Err())
	ids := make([]string, 5)
	for i := range ids {
		id, err := rdb.XAdd(ctx, &redis.XAddArgs{Stream: key, Values: []any{"n", i}}).Result()
		require.NoError(t, err)
		ids[i] = id
	}
	require.NoError(t, rdb.XReadGroup(ctx, &redis.XReadGroupArgs{Group: group, Consumer: "c1", Streams: []string{key, ">"}, Count: 3}).Err())
	require.NoError(t, rdb.XReadGroup(ctx, &redis.XReadGroupArgs{Group: group, Consumer: "c2", Streams: []string{key, ">"}}).Err())
	require.NoError(t, rdb.XDel(ctx, key, ids[1], ids[3]).Err())

	// No entry is idle for an hour.
	messages, deleted, next, err := claimIdle(ctx, rdb, claimArgs{stream: key, group: group, consumer: "c3", minIdle: time.Hour, start: "0-0"})
	require.NoError(t, err)
	assert.Empty(t, messages)
	assert.Zero(t, deleted)
	assert.Equal(t, claimDone, next)

	args := claimArgs{stream: key, group: group, consumer: "c3", start: "0-0", count: 2}
	var claimed []string
	total := 0
	for batch := 0; ; batch++ {
		require.Less(t, batch, 5, "claim loop does not end")
		messages, deleted, next, err := claimIdle(ctx, rdb, args)
		require.NoError(t, err)
		for _, msg := range messages {
			claimed = append(claimed, msg.ID)
		}
		total += deleted
		if next == claimDone {
			break
		}
		args.start = next
	}
	assert.Equal(t, []string{ids[0], ids[2], ids[4]}, claimed)
	assert.Equal(t, 2, total)

	pending, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{Stream: key, Group: group, Start: "-", End: "+", Count: 10}).Result()
	require.NoError(t, err)
	require.Len(t, pending, 3)
	for i, p := range pending {
		assert.Equal(t, claimed[i], p.ID)
		assert.Equal(t, "c3", p.Consumer)
		assert.Equal(t, int64(2), p.RetryCount, "delivery count of %s", p.ID)
	}
}

// TestClaimIdleSkipsEntriesThatAreNotIdle checks that a batch scans at most
// count pending entries, idle or not, and that the scan reaches idle entries
// after a batch of entries that are not idle, across cursor steps. It acks
// the idle entry of a deleted event and leaves the others with their owner.
func TestClaimIdleSkipsEntriesThatAreNotIdle(t *testing.T) {
	srv := startTestServer(t)
	rdb := srv.Client
	ctx := t.Context()
	const key, group = "claim-idle-not-idle", "g"
	ids := deliverClaimTestEntries(t, rdb, key, group, 6)
	require.NoError(t, rdb.XDel(ctx, key, ids[4]).Err())
	ageClaimTestEntries(t, srv)
	// Claiming the first three again makes them not idle.
	require.NoError(t, rdb.XClaim(ctx, &redis.XClaimArgs{Stream: key, Group: group, Consumer: "c1", Messages: ids[:3]}).Err())

	args := claimArgs{stream: key, group: group, consumer: "c3", minIdle: claimTestMinIdle, start: "0-0", count: 2}
	type batch struct {
		claimed []string
		deleted int
		done    bool
	}
	var got []batch
	for len(got) < 6 {
		messages, deleted, next, err := claimIdle(ctx, rdb, args)
		require.NoError(t, err)
		b := batch{deleted: deleted, done: next == claimDone}
		for _, msg := range messages {
			b.claimed = append(b.claimed, msg.ID)
		}
		got = append(got, b)
		if b.done {
			break
		}
		args.start = next
	}
	assert.Equal(t, []batch{
		{},
		{claimed: []string{ids[3]}},
		{claimed: []string{ids[5]}, deleted: 1},
		{done: true},
	}, got)

	pending, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{Stream: key, Group: group, Start: "-", End: "+", Count: 10}).Result()
	require.NoError(t, err)
	owners := make(map[string]string, len(pending))
	for _, p := range pending {
		owners[p.ID] = p.Consumer
	}
	assert.Equal(t, map[string]string{ids[0]: "c1", ids[1]: "c1", ids[2]: "c1", ids[3]: "c3", ids[5]: "c3"}, owners)
}

// TestClaimIdleBatchOfDeletedEntries checks a batch made only of pending
// entries of deleted events: claimIdle acks them all, claims nothing, and the
// next batch finds nothing.
func TestClaimIdleBatchOfDeletedEntries(t *testing.T) {
	srv := startTestServer(t)
	rdb := srv.Client
	ctx := t.Context()
	const key, group = "claim-idle-deleted", "g"
	ids := deliverClaimTestEntries(t, rdb, key, group, 3)
	require.NoError(t, rdb.XDel(ctx, key, ids...).Err())
	ageClaimTestEntries(t, srv)

	args := claimArgs{stream: key, group: group, consumer: "c3", minIdle: claimTestMinIdle, start: "0-0", count: 3}
	messages, deleted, next, err := claimIdle(ctx, rdb, args)
	require.NoError(t, err)
	assert.Empty(t, messages)
	assert.Equal(t, 3, deleted)
	require.NotEqual(t, claimDone, next, "the batch is full")

	args.start = next
	messages, deleted, next, err = claimIdle(ctx, rdb, args)
	require.NoError(t, err)
	assert.Empty(t, messages)
	assert.Zero(t, deleted)
	assert.Equal(t, claimDone, next)

	pending, err := rdb.XPending(ctx, key, group).Result()
	require.NoError(t, err)
	assert.Zero(t, pending.Count)
}

func TestParseClaimIdle(t *testing.T) {
	entry := func(id string, fields ...any) any {
		return []any{id, fields}
	}
	cases := []struct {
		name     string
		reply    []any
		messages []redis.XMessage
		deleted  int
		last     string
		err      string
	}{
		{
			name:  "claimed entries",
			reply: []any{"", []any{entry("1-0", "n", "a", "p", "x"), entry("2-0", "n", "b")}, int64(0)},
			messages: []redis.XMessage{
				{ID: "1-0", Values: map[string]any{"n": "a", "p": "x"}},
				{ID: "2-0", Values: map[string]any{"n": "b"}},
			},
		},
		{
			name:     "full batch with acked entries",
			reply:    []any{"3-0", []any{entry("2-0")}, int64(2)},
			messages: []redis.XMessage{{ID: "2-0", Values: map[string]any{}}},
			deleted:  2,
			last:     "3-0",
		},
		{
			name:     "nothing claimed",
			reply:    []any{"", []any{}, int64(0)},
			messages: []redis.XMessage{},
		},
		{name: "too short", reply: []any{"", []any{}}, err: "got 2 reply elements"},
		{name: "last not a string", reply: []any{int64(0), []any{}, int64(0)}, err: "last id is int64"},
		{name: "entries not an array", reply: []any{"", "x", int64(0)}, err: "entries are string"},
		{name: "deleted not an integer", reply: []any{"", []any{}, "x"}, err: "deleted count is string"},
		{name: "null entry", reply: []any{"", []any{nil}, int64(0)}, err: "entry 0: entry is <nil>"},
		{name: "entry not an array", reply: []any{"", []any{"1-0"}, int64(0)}, err: "entry 0: entry is string"},
		{name: "entry id not a string", reply: []any{"", []any{[]any{int64(1), []any{}}}, int64(0)}, err: "entry 0: id is int64"},
		{name: "null fields", reply: []any{"", []any{[]any{"1-0", nil}}, int64(0)}, err: "entry 0: entry 1-0 fields are <nil>"},
		{name: "odd fields", reply: []any{"", []any{entry("1-0", "n")}, int64(0)}, err: "entry 0: entry 1-0 fields"},
		{name: "field name not a string", reply: []any{"", []any{entry("1-0", int64(1), "v")}, int64(0)}, err: "field 0 name is int64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			messages, deleted, last, err := parseClaimIdle(tc.reply)
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.messages, messages)
			assert.Equal(t, tc.deleted, deleted)
			assert.Equal(t, tc.last, last)
		})
	}
}

func TestNextStreamID(t *testing.T) {
	cases := []struct {
		id, next, err string
	}{
		{id: "0-0", next: "0-1"},
		{id: "1700000000000-41", next: "1700000000000-42"},
		{id: "5-18446744073709551615", next: "6-0"},
		{id: "18446744073709551615-18446744073709551615", next: claimDone},
		{id: "5", err: "has no sequence number"},
		{id: "x-0", err: `stream id "x-0"`},
		{id: "5-x", err: `stream id "5-x"`},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			next, err := nextStreamID(tc.id)
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.next, next)
		})
	}
}

func TestMinIdleMillis(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want int64
	}{
		{d: 0, want: 0},
		{d: time.Nanosecond, want: 1},
		{d: 999 * time.Microsecond, want: 1},
		{d: time.Millisecond, want: 1},
		{d: 1500 * time.Microsecond, want: 1},
		{d: 150 * time.Millisecond, want: 150},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, minIdleMillis(tc.d), "minIdleMillis(%v)", tc.d)
	}
}

// deliverClaimTestEntries adds n entries to the stream key and delivers them
// to consumer c1 of a new group. It returns their ids in order.
func deliverClaimTestEntries(t *testing.T, rdb *redis.Client, key, group string, n int) []string {
	t.Helper()
	ctx := t.Context()
	require.NoError(t, rdb.XGroupCreateMkStream(ctx, key, group, "0").Err())
	ids := make([]string, n)
	for i := range ids {
		id, err := rdb.XAdd(ctx, &redis.XAddArgs{Stream: key, Values: []any{"n", i}}).Result()
		require.NoError(t, err)
		ids[i] = id
	}
	require.NoError(t, rdb.XReadGroup(ctx, &redis.XReadGroupArgs{Group: group, Consumer: "c1", Streams: []string{key, ">"}}).Err())
	return ids
}

// ageClaimTestEntries ages the pending entries of srv past claimTestMinIdle:
// it moves the miniredis clock, or waits on a real server.
func ageClaimTestEntries(t *testing.T, srv *redistest.Server) {
	t.Helper()
	if mr := srv.Miniredis(); mr != nil {
		mr.SetTime(time.Now().Add(time.Hour))
		return
	}
	time.Sleep(claimTestMinIdle + 50*time.Millisecond)
}
