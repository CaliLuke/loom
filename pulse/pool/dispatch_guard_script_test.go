package pool

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming"
)

// guardScriptNode returns a node whose pool sink is closed and whose sink
// group is destroyed, so the test controls the delivery state of the pool
// stream entries.
func guardScriptNode(t *testing.T, pool string) (*redis.Client, *Node) {
	t.Helper()
	rdb := startTestRedis(t)
	return rdb, sinklessNode(t, rdb, pool)
}

// sinklessNode is guardScriptNode on the server of rdb, with node options
// opts.
func sinklessNode(t *testing.T, rdb *redis.Client, pool string, opts ...NodeOption) *Node {
	t.Helper()
	node := addTestNode(t, rdb, pool, opts...)
	node.poolSink.Close(t.Context())
	require.NoError(t, rdb.XGroupDestroy(t.Context(), node.poolStream.Key(), poolSinkName).Err())
	return node
}

// setGuard writes guard for key through the pending rmap and waits until the
// node's replica shows it.
func setGuard(t *testing.T, node *Node, key, guard string) {
	t.Helper()
	_, err := node.jobPendingMap.Set(t.Context(), key, guard)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		v, ok := node.jobPendingMap.Get(key)
		return ok && v == guard
	}, 5*time.Second, time.Millisecond)
}

// staleUntil is a guard TTL that has passed.
const staleUntil = "1"

// TestDispatchGuardInFlight checks every path that clears a stale guard
// against every delivery state of its start event. A guard whose event is in
// flight stays: not yet delivered, or delivered and unacked, including after
// the stream trimmed the entry while it was pending, and after Redis 7
// XAUTOCLAIM then purged its pending entry (issue #385). A guard whose event
// is acked and still in the stream, was trimmed before delivery, or is older
// than pendingEventTTL, is cleared.
func TestDispatchGuardInFlight(t *testing.T) {
	states := []struct {
		name     string
		prepare  func(t *testing.T, rdb *redis.Client, stream string) string
		inFlight bool
	}{
		{
			name: "no sink group",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				return rdb.XAdd(t.Context(), &redis.XAddArgs{Stream: stream, Values: []any{"n", "j", "p", "x"}}).Val()
			},
			inFlight: true,
		},
		{
			name: "not yet delivered",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				require.NoError(t, rdb.XGroupCreateMkStream(t.Context(), stream, poolSinkName, "$").Err())
				return rdb.XAdd(t.Context(), &redis.XAddArgs{Stream: stream, Values: []any{"n", "j", "p", "x"}}).Val()
			},
			inFlight: true,
		},
		{
			name: "delivered and unacked",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				return deliverEvent(t, rdb, stream)
			},
			inFlight: true,
		},
		{
			name: "acked",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				id := deliverEvent(t, rdb, stream)
				require.NoError(t, rdb.XAck(t.Context(), stream, poolSinkName, id).Err())
				return id
			},
			inFlight: false,
		},
		{
			// The pending entry survives the trim on every version, until
			// XAUTOCLAIM reaches it. Redis 7 and later XAUTOCLAIM then purges
			// it (TestRedisDispatchGuardAfterAutoClaimOfTrimmedStart).
			name: "trimmed while pending",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				id := deliverEvent(t, rdb, stream)
				trimPending(t, rdb, stream, id)
				return id
			},
			inFlight: true,
		},
		{
			name: "trimmed while pending, older than pendingEventTTL",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				// Stream id 1-0 dates from 1970, far beyond pendingEventTTL.
				id := deliverEventWithID(t, rdb, stream, "1-0")
				trimPending(t, rdb, stream, id)
				return id
			},
			inFlight: false,
		},
		{
			name: "undelivered, older than pendingEventTTL",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				return rdb.XAdd(t.Context(), &redis.XAddArgs{Stream: stream, ID: "1-0", Values: []any{"n", "j", "p", "x"}}).Val()
			},
			inFlight: false,
		},
		{
			// The state Redis 7 XAUTOCLAIM leaves after it purges a trimmed
			// pending entry: delivered, in neither the stream nor the pending
			// list. A router may have queued the event on a worker stream
			// (issue #385, TLC config guard_asis_r7). XACK stands in for the
			// purge so that every server reaches the state.
			name: "purged after trim",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				id := deliverEvent(t, rdb, stream)
				require.NoError(t, rdb.XDel(t.Context(), stream, id).Err())
				require.NoError(t, rdb.XAck(t.Context(), stream, poolSinkName, id).Err())
				return id
			},
			inFlight: true,
		},
		{
			// Redis cannot tell this state from a purge. A worker ack
			// deletes the guard (TestAckStartDeletesGuard), so a guard left
			// on an acked event comes from an ack that did not start the job.
			name: "trimmed after an ack that kept the guard",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				id := deliverEvent(t, rdb, stream)
				require.NoError(t, rdb.XAck(t.Context(), stream, poolSinkName, id).Err())
				require.NoError(t, rdb.XDel(t.Context(), stream, id).Err())
				return id
			},
			inFlight: true,
		},
		{
			name: "purged after trim, older than pendingEventTTL",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				id := deliverEventWithID(t, rdb, stream, "1-0")
				require.NoError(t, rdb.XDel(t.Context(), stream, id).Err())
				require.NoError(t, rdb.XAck(t.Context(), stream, poolSinkName, id).Err())
				return id
			},
			inFlight: false,
		},
		{
			// A trim before delivery loses the event: no router ever saw it.
			name: "trimmed before delivery",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				require.NoError(t, rdb.XGroupCreateMkStream(t.Context(), stream, poolSinkName, "$").Err())
				id := rdb.XAdd(t.Context(), &redis.XAddArgs{Stream: stream, Values: []any{"n", "j", "p", "x"}}).Val()
				require.NoError(t, rdb.XDel(t.Context(), stream, id).Err())
				return id
			},
			inFlight: false,
		},
		{
			// Once the group delivers a later entry, the lost event's id is
			// below last-delivered-id and it looks purged, so the guard is
			// held until pendingEventTTL: the safe direction.
			name: "trimmed before delivery, later entries delivered",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				ctx := t.Context()
				require.NoError(t, rdb.XGroupCreateMkStream(ctx, stream, poolSinkName, "$").Err())
				id := rdb.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: []any{"n", "j", "p", "x"}}).Val()
				require.NoError(t, rdb.XDel(ctx, stream, id).Err())
				require.NoError(t, rdb.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: []any{"n", "j", "p", "y"}}).Err())
				res, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group: poolSinkName, Consumer: "c1", Streams: []string{stream, ">"}, Count: 1, Block: -1,
				}).Result()
				require.NoError(t, err)
				require.Len(t, res, 1)
				require.Len(t, res[0].Messages, 1)
				return id
			},
			inFlight: true,
		},
		{
			name: "trimmed, no sink group",
			prepare: func(t *testing.T, rdb *redis.Client, stream string) string {
				t.Helper()
				id := rdb.XAdd(t.Context(), &redis.XAddArgs{Stream: stream, Values: []any{"n", "j", "p", "x"}}).Val()
				require.NoError(t, rdb.XDel(t.Context(), stream, id).Err())
				return id
			},
			inFlight: false,
		},
	}
	paths := []struct {
		name string
		// clear runs the path and reports whether it cleared or replaced the
		// guard.
		clear func(t *testing.T, node *Node, guard string) bool
	}{
		{
			name: "claimDispatch stale replacement",
			clear: func(t *testing.T, node *Node, _ string) bool {
				t.Helper()
				_, _, err := node.claimDispatch(t.Context(), "k1", marshalJobKey("k1"))
				if err != nil {
					require.ErrorIs(t, err, ErrJobExists)
					return false
				}
				return true
			},
		},
		{
			name: "dispatcher release",
			clear: func(t *testing.T, node *Node, guard string) bool {
				t.Helper()
				node.releaseDispatchPending("k1", guard)
				return guardGone(t, node, guard)
			},
		},
		{
			name: "stale sweep",
			clear: func(t *testing.T, node *Node, guard string) bool {
				t.Helper()
				node.cleanupStalePendingJobs(t.Context())
				return guardGone(t, node, guard)
			},
		},
	}
	for _, state := range states {
		for _, path := range paths {
			t.Run(state.name+"/"+path.name, func(t *testing.T) {
				rdb, node := guardScriptNode(t, "guard-inflight")
				id := state.prepare(t, rdb, node.poolStream.Key())
				guard := staleUntil + ":" + id
				setGuard(t, node, "k1", guard)

				cleared := path.clear(t, node, guard)
				require.Equal(t, !state.inFlight, cleared, "guard cleared")
			})
		}
	}
}

// TestAckStartDeletesGuard checks luaAckStart, which acks a start event after
// a worker acked it. It XACKs the event in every case, and deletes the
// pending guard of the job only when the guard names that event, including
// after the stream trimmed the event (issue #385: in_flight then holds the
// guard until pendingEventTTL, so the ack must delete it). A guard that
// names another event, or no event, stays. The deletion reaches the node's
// replica, so the rmap notification is published.
func TestAckStartDeletesGuard(t *testing.T) {
	future := strconv.FormatInt(time.Now().Add(time.Hour).UnixNano(), 10)
	cases := []struct {
		name string
		// guard returns the guard to write for the acked event id, or ""
		// for no guard.
		guard   func(id string) string
		trim    bool
		deleted bool
	}{
		{name: "guard names the event", guard: func(id string) string { return staleUntil + ":" + id }, deleted: true},
		{name: "active guard names the event", guard: func(id string) string { return future + ":" + id }, deleted: true},
		{name: "guard names the trimmed event", guard: func(id string) string { return staleUntil + ":" + id }, trim: true, deleted: true},
		{name: "guard names another event", guard: func(string) string { return future + ":1-1" }},
		{name: "guard without event id", guard: func(string) string { return future }},
		{name: "no guard", guard: func(string) string { return "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb, node := guardScriptNode(t, "guard-ack")
			ctx := t.Context()
			stream := node.poolStream.Key()
			id := deliverEvent(t, rdb, stream)
			if tc.trim {
				trimPending(t, rdb, stream, id)
			}
			guard := tc.guard(id)
			if guard != "" {
				setGuard(t, node, "k1", guard)
			}

			ev := &streaming.Event{ID: id, EventName: evStartJob, Payload: marshalJob(&Job{Key: "k1", NodeID: node.ID})}
			require.NoError(t, node.ackRoutedEvent(ctx, ev))

			pending, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
				Stream: stream, Group: poolSinkName, Start: id, End: id, Count: 1,
			}).Result()
			require.NoError(t, err)
			require.Empty(t, pending, "the start event is acked")
			if guard == "" {
				_, ok := pendingGuard(t, rdb, node.PoolName)
				require.False(t, ok, "a guard was written")
				return
			}
			require.Equal(t, tc.deleted, guardGone(t, node, guard), "guard deleted")
			if tc.deleted {
				require.Eventually(t, func() bool {
					_, ok := node.jobPendingMap.Get("k1")
					return !ok
				}, 5*time.Second, time.Millisecond, "the replica still has the guard")
			}
		})
	}
}

// TestDispatchGuardComparesStreamIDsAsNumbers checks that a guard whose event
// id has a multi-digit sequence is compared with the group's
// last-delivered-id as numbers: ms-10 comes after ms-9, although it sorts
// before it as a string.
func TestDispatchGuardComparesStreamIDsAsNumbers(t *testing.T) {
	cases := []struct {
		name          string
		guardSeq      string
		lastDelivered string
		inFlight      bool
	}{
		{name: "ms-10 after last delivered ms-9", guardSeq: "10", lastDelivered: "9", inFlight: true},
		{name: "ms-9 before last delivered ms-10", guardSeq: "9", lastDelivered: "10", inFlight: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb, node := guardScriptNode(t, "guard-ids")
			ctx := t.Context()
			stream := node.poolStream.Key()
			ms := strconv.FormatInt(time.Now().UnixMilli(), 10)
			for _, seq := range []string{"9", "10"} {
				require.NoError(t, rdb.XAdd(ctx, &redis.XAddArgs{Stream: stream, ID: ms + "-" + seq, Values: []any{"n", "j", "p", "x"}}).Err())
			}
			require.NoError(t, rdb.XGroupCreate(ctx, stream, poolSinkName, ms+"-"+tc.lastDelivered).Err())
			guard := staleUntil + ":" + ms + "-" + tc.guardSeq
			setGuard(t, node, "k1", guard)

			node.releaseDispatchPending("k1", guard)
			require.Equal(t, !tc.inFlight, guardGone(t, node, guard))
		})
	}
}

// TestDispatchGuardAfterClientError covers a claimDispatch whose reply the
// client never sees: the script ran on the server, so the guard and its
// event exist. A retry returns ErrJobExists, before and after the guard TTL,
// while the event is unacked.
func TestDispatchGuardAfterClientError(t *testing.T) {
	_, node := guardScriptNode(t, "guard-client-error")
	ctx := t.Context()
	now := time.Now()
	_, err := node.runClaimDispatch(ctx, "k1", marshalJobKey("k1"), now, now.Add(100*time.Millisecond))
	require.NoError(t, err) // The reply is discarded, as after a client error.

	require.ErrorIs(t, node.DispatchJob(ctx, "k1", nil), ErrJobExists)
	time.Sleep(150 * time.Millisecond)
	require.ErrorIs(t, node.DispatchJob(ctx, "k1", nil), ErrJobExists)
}

// TestDispatchGuardFormats checks the guard formats claimDispatch reads. A
// guard written by an earlier release ("untilNanos") has no event id and is
// honored by its TTL alone. A malformed guard is rejected.
func TestDispatchGuardFormats(t *testing.T) {
	future := strconv.FormatInt(time.Now().Add(time.Hour).UnixNano(), 10)
	cases := []struct {
		name    string
		guard   string
		wantErr string
		exists  bool
	}{
		{name: "legacy active", guard: future, exists: true},
		{name: "legacy stale", guard: staleUntil},
		{name: "active with event id", guard: future + ":1-0", exists: true},
		{name: "malformed", guard: "not-a-guard", wantErr: "malformed pending guard"},
		{name: "malformed event id", guard: staleUntil + ":1-x", wantErr: "malformed pending guard"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, node := guardScriptNode(t, "guard-formats")
			setGuard(t, node, "k1", tc.guard)

			guard, eventID, err := node.claimDispatch(t.Context(), "k1", marshalJobKey("k1"))
			switch {
			case tc.wantErr != "":
				require.ErrorContains(t, err, tc.wantErr)
			case tc.exists:
				require.ErrorIs(t, err, ErrJobExists)
			default:
				require.NoError(t, err)
				require.Equal(t, eventID, guard[len(guard)-len(eventID):])
			}
		})
	}
}

// TestParsePendingGuard checks the Go guard parser.
func TestParsePendingGuard(t *testing.T) {
	cases := []struct {
		value   string
		until   int64
		eventID string
		wantErr bool
	}{
		{value: "123", until: 123},
		{value: "123:5-10", until: 123, eventID: "5-10"},
		{value: "", wantErr: true},
		{value: "-1", wantErr: true},
		{value: "abc", wantErr: true},
		{value: "123:", wantErr: true},
		{value: "123:5", wantErr: true},
		{value: "123:5-", wantErr: true},
		{value: "123:a-1", wantErr: true},
	}
	for _, tc := range cases {
		until, eventID, err := parsePendingGuard(tc.value)
		if tc.wantErr {
			require.Error(t, err, tc.value)
			continue
		}
		require.NoError(t, err, tc.value)
		require.Equal(t, tc.until, until, tc.value)
		require.Equal(t, tc.eventID, eventID, tc.value)
	}
}

// TestClaimDispatchEntryMatchesStreamAdd pins the start event that
// claimDispatch adds to the entry Stream.Add writes for the same job.
func TestClaimDispatchEntryMatchesStreamAdd(t *testing.T) {
	rdb, node := guardScriptNode(t, "guard-golden")
	ctx := t.Context()
	job := marshalJob(&Job{Key: "k1", Payload: []byte("p"), CreatedAt: time.Unix(0, 1), NodeID: node.ID})

	_, scriptID, err := node.claimDispatch(ctx, "k1", job)
	require.NoError(t, err)
	addID, err := node.poolStream.Add(ctx, evStartJob, job)
	require.NoError(t, err)

	stream := node.poolStream.Key()
	scriptEntry := rdb.XRange(ctx, stream, scriptID, scriptID).Val()
	addEntry := rdb.XRange(ctx, stream, addID, addID).Val()
	require.Len(t, scriptEntry, 1)
	require.Len(t, addEntry, 1)
	require.Equal(t, addEntry[0].Values, scriptEntry[0].Values)
	require.Equal(t, map[string]any{"n": evStartJob, "p": string(job)}, scriptEntry[0].Values)
	require.Equal(t, time.Duration(-1), rdb.PTTL(ctx, stream).Val(), "pool stream TTL")
}

// deliverEvent adds an event to stream and delivers it to a consumer of the
// sink group without acking it.
func deliverEvent(t *testing.T, rdb *redis.Client, stream string) string {
	t.Helper()
	return deliverEventWithID(t, rdb, stream, "*")
}

// trimPending removes the entry id from stream and checks that it is still
// in the sink group's pending list, as after a MAXLEN trim of an unacked
// entry.
func trimPending(t *testing.T, rdb *redis.Client, stream, id string) {
	t.Helper()
	ctx := t.Context()
	require.NoError(t, rdb.XDel(ctx, stream, id).Err())
	pending, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream, Group: poolSinkName, Start: id, End: id, Count: 1,
	}).Result()
	require.NoError(t, err)
	require.Len(t, pending, 1, "the pending entry must survive the trim")
}

// deliverEventWithID adds an event with the stream id id ("*" for a new id)
// to stream and delivers it to a consumer of the sink group without acking
// it.
func deliverEventWithID(t *testing.T, rdb *redis.Client, stream, id string) string {
	t.Helper()
	ctx := t.Context()
	require.NoError(t, rdb.XGroupCreateMkStream(ctx, stream, poolSinkName, "$").Err())
	id = rdb.XAdd(ctx, &redis.XAddArgs{Stream: stream, ID: id, Values: []any{"n", "j", "p", "x"}}).Val()
	res, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    poolSinkName,
		Consumer: "c1",
		Streams:  []string{stream, ">"},
		Count:    1,
		Block:    -1,
	}).Result()
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Len(t, res[0].Messages, 1)
	require.Equal(t, id, res[0].Messages[0].ID)
	return id
}

// guardGone reports whether the pending guard for k1 no longer has the value
// guard in Redis.
func guardGone(t *testing.T, node *Node, guard string) bool {
	t.Helper()
	v, err := node.rdb.HGet(t.Context(), rmapContentKey(jobPendingMapName(node.PoolName)), "k1").Result()
	if errors.Is(err, redis.Nil) {
		return true
	}
	require.NoError(t, err)
	return v != guard
}
