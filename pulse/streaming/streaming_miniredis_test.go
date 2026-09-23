package streaming

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/rmap"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

// luaStructShim is a pure-Lua replacement for the Redis "struct" library used
// by the rmap mutation scripts that back sink bookkeeping. miniredis does not
// ship the struct library so the test client rewrites scripts to prepend this
// shim. Only the struct.pack "i" (4-byte little-endian length) and "c0" (raw
// string) format codes used by the production scripts are implemented.
const luaStructShim = `
local struct = { pack = function(fmt, ...)
   local args = {...}
   local out = {}
   local ai = 1
   local i = 1
   while i <= string.len(fmt) do
      local c = string.sub(fmt, i, i)
      if c == "i" then
         local n = args[ai]
         ai = ai + 1
         out[#out+1] = string.char(n % 256, math.floor(n / 256) % 256, math.floor(n / 65536) % 256, math.floor(n / 16777216) % 256)
      elseif c == "c" then
         while i < string.len(fmt) and string.match(string.sub(fmt, i + 1, i + 1), "%d") do
            i = i + 1
         end
         out[#out+1] = args[ai]
         ai = ai + 1
      end
      i = i + 1
   end
   return table.concat(out)
end }
`

// structShimHook rewrites EVAL and SCRIPT LOAD commands on their way to
// miniredis so scripts that use the Redis struct library keep working.
type structShimHook struct{}

func (structShimHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (structShimHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		shimStructScript(cmd)
		return next(ctx, cmd)
	}
}

func (structShimHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			shimStructScript(cmd)
		}
		return next(ctx, cmds)
	}
}

// shimStructScript prepends the struct shim to Lua sources sent via EVAL or
// SCRIPT LOAD. EVALSHA calls issued for the original source fail with NOSCRIPT
// and go-redis falls back to EVAL, which is then rewritten here.
func shimStructScript(cmd redis.Cmder) {
	args := cmd.Args()
	switch cmd.Name() {
	case "eval":
		if len(args) > 1 {
			if src, ok := args[1].(string); ok && strings.Contains(src, "struct.pack") {
				args[1] = luaStructShim + src
			}
		}
	case "script":
		if len(args) > 2 {
			if sub, ok := args[1].(string); ok && strings.EqualFold(sub, "load") {
				if src, ok := args[2].(string); ok && strings.Contains(src, "struct.pack") {
					args[2] = luaStructShim + src
				}
			}
		}
	}
}

// startTestRedis runs an in-process miniredis server and returns a client
// whose Lua scripts are rewritten to work around the missing struct library.
func startTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	_, rdb := startTestRedisServer(t)
	return rdb
}

// startTestRedisServer is startTestRedis but also returns the miniredis server
// so tests can control its clock.
func startTestRedisServer(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rdb.AddHook(structShimHook{})
	t.Cleanup(func() {
		require.NoError(t, rdb.Close())
	})
	return mr, rdb
}

// newTestStream creates a stream backed by the test Redis server.
func newTestStream(t *testing.T, rdb *redis.Client, name string) *Stream {
	t.Helper()
	s, err := NewStream(name, rdb)
	require.NoError(t, err)
	return s
}

// newTestSink creates a sink that polls quickly and is closed on test cleanup.
func newTestSink(t *testing.T, stream *Stream, name string, opts ...options.Sink) *Sink {
	t.Helper()
	opts = append([]options.Sink{
		options.WithSinkBlockDuration(50 * time.Millisecond),
		options.WithSinkAckGracePeriod(time.Second),
	}, opts...)
	sink, err := stream.NewSink(t.Context(), name, opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		sink.Close(context.Background())
	})
	return sink
}

// newTestSinkWithRuntime creates a sink with explicit background scheduling.
func newTestSinkWithRuntime(
	t *testing.T,
	stream *Stream,
	name string,
	runtime sinkRuntime,
	opts ...options.Sink,
) *Sink {
	t.Helper()
	opts = append([]options.Sink{
		options.WithSinkBlockDuration(50 * time.Millisecond),
		options.WithSinkAckGracePeriod(time.Second),
	}, opts...)
	sink, err := newSink(t.Context(), name, stream, runtime, opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		sink.Close(context.Background())
	})
	return sink
}

// receiveEvent waits for an event on c and fails the test on timeout.
func receiveEvent(t *testing.T, c <-chan *Event) *Event {
	t.Helper()
	select {
	case ev, ok := <-c:
		require.True(t, ok, "event channel closed")
		return ev
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for event")
		return nil
	}
}

func TestReaderReceivesEvents(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "reader-events")

	id1, err := stream.Add(ctx, "created", []byte("first"))
	require.NoError(t, err)
	id2, err := stream.Add(ctx, "updated", []byte("second"), options.WithTopic("orders"))
	require.NoError(t, err)

	reader, err := stream.NewReader(ctx,
		options.WithReaderBlockDuration(50*time.Millisecond),
		options.WithReaderStartAtOldest())
	require.NoError(t, err)
	t.Cleanup(reader.Close)
	c := reader.Subscribe()

	ev := receiveEvent(t, c)
	require.Equal(t, id1, ev.ID)
	require.Equal(t, "created", ev.EventName)
	require.Equal(t, []byte("first"), ev.Payload)
	require.Equal(t, "reader-events", ev.StreamName)
	require.Empty(t, ev.Topic)

	ev = receiveEvent(t, c)
	require.Equal(t, id2, ev.ID)
	require.Equal(t, "updated", ev.EventName)
	require.Equal(t, []byte("second"), ev.Payload)
	require.Equal(t, "orders", ev.Topic)

	reader.Close()
	require.Eventually(t, reader.IsClosed, 5*time.Second, 5*time.Millisecond)
}

func TestReaderReadsFromAddedStream(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	streamA := newTestStream(t, rdb, "reader-multi-a")
	streamB := newTestStream(t, rdb, "reader-multi-b")

	reader, err := streamA.NewReader(ctx,
		options.WithReaderBlockDuration(50*time.Millisecond),
		options.WithReaderStartAtOldest())
	require.NoError(t, err)
	t.Cleanup(reader.Close)
	require.NoError(t, reader.AddStream(ctx, streamB, options.WithAddStreamStartAtOldest()))
	c := reader.Subscribe()

	_, err = streamA.Add(ctx, "from-a", []byte("a"))
	require.NoError(t, err)
	_, err = streamB.Add(ctx, "from-b", []byte("b"))
	require.NoError(t, err)

	got := map[string]string{}
	for range 2 {
		ev := receiveEvent(t, c)
		got[ev.EventName] = ev.StreamName
	}
	require.Equal(t, map[string]string{"from-a": "reader-multi-a", "from-b": "reader-multi-b"}, got)
}

func TestSinkConsumesAndAcks(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "sink-ack")
	sink := newTestSink(t, stream, "acker", options.WithSinkStartAtOldest())
	c := sink.Subscribe()

	id1, err := stream.Add(ctx, "created", []byte("first"))
	require.NoError(t, err)
	id2, err := stream.Add(ctx, "updated", []byte("second"))
	require.NoError(t, err)

	ev1 := receiveEvent(t, c)
	require.Equal(t, id1, ev1.ID)
	require.Equal(t, "acker", ev1.SinkName)
	ev2 := receiveEvent(t, c)
	require.Equal(t, id2, ev2.ID)

	require.NoError(t, sink.Ack(ctx, ev1))
	require.NoError(t, sink.Ack(ctx, ev2))

	require.Eventually(t, func() bool {
		pending, err := rdb.XPending(ctx, streamKeyPrefix+"sink-ack", "acker").Result()
		return err == nil && pending.Count == 0
	}, 5*time.Second, 5*time.Millisecond)

	sink.Close(ctx)
	require.True(t, sink.IsClosed())
	sink.Close(ctx) // Close is idempotent.
}

func TestSinkNoAckLeavesNothingPending(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "sink-noack")
	sink := newTestSink(t, stream, "noacker", options.WithSinkStartAtOldest(), options.WithSinkNoAck())
	c := sink.Subscribe()

	_, err := stream.Add(ctx, "created", []byte("payload"))
	require.NoError(t, err)

	ev := receiveEvent(t, c)
	require.Equal(t, "created", ev.EventName)

	pending, err := rdb.XPending(ctx, streamKeyPrefix+"sink-noack", "noacker").Result()
	require.NoError(t, err)
	require.Zero(t, pending.Count)
}

func TestSinkTopicFilter(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "sink-topic")
	sink := newTestSink(t, stream, "filtered", options.WithSinkStartAtOldest(), options.WithSinkTopic("orders"))
	c := sink.Subscribe()

	_, err := stream.Add(ctx, "skipped", []byte("no match"), options.WithTopic("payments"))
	require.NoError(t, err)
	matchID, err := stream.Add(ctx, "kept", []byte("match"), options.WithTopic("orders"))
	require.NoError(t, err)

	ev := receiveEvent(t, c)
	require.Equal(t, matchID, ev.ID, "only the matching topic event must be delivered")
	require.Equal(t, "kept", ev.EventName)
}

func TestAddOnlyIfStreamExists(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "maybe-stream")

	id, err := stream.Add(ctx, "created", []byte("payload"), options.WithOnlyIfStreamExists())
	require.NoError(t, err)
	require.Empty(t, id, "no event must be added when the stream does not exist")

	exists, err := rdb.Exists(ctx, streamKeyPrefix+"maybe-stream").Result()
	require.NoError(t, err)
	require.Zero(t, exists)

	// Once the stream exists the option is a no-op.
	_, err = stream.Add(ctx, "created", []byte("payload"))
	require.NoError(t, err)
	id, err = stream.Add(ctx, "again", []byte("payload"), options.WithOnlyIfStreamExists())
	require.NoError(t, err)
	require.NotEmpty(t, id)
}

func TestStreamRemoveAndDestroy(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "cleanup")
	sink := newTestSink(t, stream, "cleaner", options.WithSinkStartAtOldest())

	id, err := stream.Add(ctx, "created", []byte("payload"))
	require.NoError(t, err)
	require.NoError(t, stream.Remove(ctx, id))
	length, err := rdb.XLen(ctx, streamKeyPrefix+"cleanup").Result()
	require.NoError(t, err)
	require.Zero(t, length)

	sink.Close(ctx)
	require.NoError(t, stream.Destroy(ctx))

	exists, err := rdb.Exists(ctx, streamKeyPrefix+"cleanup").Result()
	require.NoError(t, err)
	require.Zero(t, exists, "stream key must be deleted")
	// The rmap destroy protocol keeps the internal revision bookkeeping so
	// live replicas converge; only user content must be gone.
	fields, err := rdb.HKeys(ctx, "map:stream:cleanup:sinks:content").Result()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"=rev", "=kind"}, fields, "sink consumers map must only retain internal bookkeeping")
}

func TestSinkAddAndRemoveStream(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	streamA := newTestStream(t, rdb, "sink-multi-a")
	streamB := newTestStream(t, rdb, "sink-multi-b")
	sink := newTestSink(t, streamA, "multi", options.WithSinkStartAtOldest())
	c := sink.Subscribe()

	require.NoError(t, sink.AddStream(ctx, streamB))
	require.NoError(t, sink.AddStream(ctx, streamB), "adding the same stream twice is a no-op")

	_, err := streamB.Add(ctx, "from-b", []byte("b"))
	require.NoError(t, err)
	ev := receiveEvent(t, c)
	require.Equal(t, "from-b", ev.EventName)
	require.Equal(t, "sink-multi-b", ev.StreamName)
	require.NoError(t, sink.Ack(ctx, ev))

	require.NoError(t, sink.RemoveStream(ctx, streamB))
	require.NoError(t, sink.RemoveStream(ctx, streamB), "removing a removed stream is a no-op")

	// The consumer group of the removed stream is destroyed with its last consumer.
	groups, err := rdb.XInfoGroups(ctx, streamKeyPrefix+"sink-multi-b").Result()
	require.NoError(t, err)
	require.Empty(t, groups)
}

func TestSinkClaimsPendingEventsOfClosedSink(t *testing.T) {
	mr, rdb := startTestRedisServer(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "sink-claim")
	id, _ := closeSinkWithPendingEvent(t, stream, "claimer")

	idleChecks := make(chan time.Time)
	second := newTestSinkWithRuntime(t, stream, "claimer", sinkRuntime{
		idleCheckPeriod: 25 * time.Millisecond,
		idleChecks:      idleChecks,
	},
		options.WithSinkAckGracePeriod(claimTestAckGracePeriod))
	secondCh := second.Subscribe()

	// Age the pending entry past the ack grace period on the Redis clock
	// instead of waiting on the wall clock.
	mr.SetTime(time.Now().Add(time.Hour))
	claimed := awaitIdleClaim(t, mr, "claimer", idleChecks, secondCh)
	require.Equal(t, id, claimed.ID)
	require.Equal(t, "created", claimed.EventName)
	require.NoError(t, second.Ack(ctx, claimed))

	pending, err := rdb.XPending(ctx, stream.key, "claimer").Result()
	require.NoError(t, err)
	require.Zero(t, pending.Count)
	second.Close(ctx)
}

func TestSinkKeepsStaleConsumerWithPendingEventsUntilClaimed(t *testing.T) {
	mr, rdb := startTestRedisServer(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "sink-stale-pending")
	id, firstConsumer := closeSinkWithPendingEvent(t, stream, "claimer")

	// Expire the closed consumer keep-alive so every cleanup pass sees it as
	// stale while its pending entry is still younger than the ack grace period.
	keepAlives, err := rmap.Join(ctx, sinkKeepAliveMapName("claimer"), rdb)
	require.NoError(t, err)
	_, err = keepAlives.Set(ctx, firstConsumer, "1")
	require.NoError(t, err)
	keepAlives.Close()

	// Creating a sink runs the stale consumer cleanup before any claim.
	idleChecks := make(chan time.Time)
	second := newTestSinkWithRuntime(t, stream, "claimer", sinkRuntime{
		idleCheckPeriod: 25 * time.Millisecond,
		idleChecks:      idleChecks,
	},
		options.WithSinkAckGracePeriod(claimTestAckGracePeriod))
	secondCh := second.Subscribe()
	requireConsumerPending(t, rdb, stream.key, "claimer", firstConsumer, 1)

	// A periodic cleanup pass must also keep the consumer.
	second.lock.Lock()
	second.deleteStaleConsumers(ctx)
	second.lock.Unlock()
	requireConsumerPending(t, rdb, stream.key, "claimer", firstConsumer, 1)

	mr.SetTime(time.Now().Add(time.Hour))
	claimed := awaitIdleClaim(t, mr, "claimer", idleChecks, secondCh)
	require.Equal(t, id, claimed.ID)
	require.NoError(t, second.Ack(ctx, claimed))

	// Once its pending entry moved, the stale consumer is deleted.
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		mr.Del(staleLockName("claimer"))
		select {
		case idleChecks <- time.Now():
		default:
		}
		pending, err := consumerPending(ctx, rdb, stream.key, "claimer", firstConsumer)
		assert.NoError(c, err)
		assert.Equal(c, int64(-1), pending)
	}, 10*time.Second, 5*time.Millisecond)
	second.Close(ctx)
}

func TestSinkDeleteIdleConsumerKeepsPendingEntries(t *testing.T) {
	_, rdb := startTestRedisServer(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "sink-delete-idle")
	id, owner := closeSinkWithPendingEvent(t, stream, "claimer")
	sink := newTestSinkWithRuntime(t, stream, "claimer", sinkRuntime{
		idleCheckPeriod: 25 * time.Millisecond,
		idleChecks:      make(chan time.Time),
	},
		options.WithSinkAckGracePeriod(claimTestAckGracePeriod))
	require.NoError(t, rdb.XGroupCreateConsumer(ctx, stream.key, "claimer", "idle").Err())

	cases := []struct {
		name        string
		consumer    string
		prepare     func(t *testing.T)
		wantDeleted bool
		wantPending int64
	}{
		{name: "owner of pending entry", consumer: owner, prepare: func(*testing.T) {}, wantDeleted: false, wantPending: 1},
		{name: "consumer without pending entries", consumer: "idle", prepare: func(*testing.T) {}, wantDeleted: true, wantPending: -1},
		{
			name:     "owner after ack",
			consumer: owner,
			prepare: func(t *testing.T) {
				require.NoError(t, rdb.XAck(ctx, stream.key, "claimer", id).Err())
			},
			wantDeleted: true,
			wantPending: -1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.prepare(t)
			deleted, err := sink.deleteIdleConsumer(ctx, stream.key, tc.consumer)
			require.NoError(t, err)
			require.Equal(t, tc.wantDeleted, deleted)
			requireConsumerPending(t, rdb, stream.key, "claimer", tc.consumer, tc.wantPending)
		})
	}
}

// claimTestAckGracePeriod is the ack grace period of sinks in claim tests.
const claimTestAckGracePeriod = 150 * time.Millisecond

// closeSinkWithPendingEvent adds an event to stream, receives it with a new
// sink instance named name without acking it, and closes that instance. It
// returns the event ID and the consumer that still owns the pending entry.
func closeSinkWithPendingEvent(t *testing.T, stream *Stream, name string) (string, string) {
	t.Helper()
	ctx := t.Context()
	first := newTestSinkWithRuntime(t, stream, name, sinkRuntime{
		idleCheckPeriod: 25 * time.Millisecond,
		idleChecks:      make(chan time.Time),
	},
		options.WithSinkStartAtOldest(),
		options.WithSinkAckGracePeriod(claimTestAckGracePeriod))
	firstCh := first.Subscribe()
	id, err := stream.Add(ctx, "created", []byte("payload"))
	require.NoError(t, err)
	ev := receiveEvent(t, firstCh)
	require.Equal(t, id, ev.ID)
	first.Close(ctx)
	requireConsumerPending(t, stream.rdb, stream.key, name, first.consumer, 1)
	return id, first.consumer
}

// awaitIdleClaim triggers idle message checks of the sink named name until it
// claims a pending event and delivers it on events. It clears the check lease
// before each trigger so every check claims regardless of wall-clock time.
func awaitIdleClaim(t *testing.T, mr *miniredis.Miniredis, name string, checks chan<- time.Time, events <-chan *Event) *Event {
	t.Helper()
	claimed := make(chan *Event, 1)
	require.Eventually(t, func() bool {
		mr.Del(staleLockName(name))
		select {
		case ev := <-events:
			claimed <- ev
			return true
		case checks <- time.Now():
		default:
		}
		return false
	}, 10*time.Second, 5*time.Millisecond)
	return <-claimed
}

// consumerPending returns the pending entry count of consumer in group, or -1
// if the consumer does not exist.
func consumerPending(ctx context.Context, rdb *redis.Client, key, group, consumer string) (int64, error) {
	consumers, err := rdb.XInfoConsumers(ctx, key, group).Result()
	if err != nil {
		return 0, err
	}
	for _, c := range consumers {
		if c.Name == consumer {
			return c.Pending, nil
		}
	}
	return -1, nil
}

// requireConsumerPending asserts the pending entry count of consumer in group;
// want -1 asserts that the consumer does not exist.
func requireConsumerPending(t *testing.T, rdb *redis.Client, key, group, consumer string, want int64) {
	t.Helper()
	got, err := consumerPending(t.Context(), rdb, key, group, consumer)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
