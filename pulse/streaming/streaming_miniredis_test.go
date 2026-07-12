package streaming

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

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
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rdb.AddHook(structShimHook{})
	t.Cleanup(func() {
		require.NoError(t, rdb.Close())
	})
	return rdb
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
	rdb := startTestRedis(t)
	ctx := t.Context()

	// Speed up the idle message check for this test only. The variable is
	// restored after both sinks are fully closed so no goroutine reads it
	// concurrently.
	oldCheckIdlePeriod := checkIdlePeriod
	checkIdlePeriod = 25 * time.Millisecond
	t.Cleanup(func() {
		checkIdlePeriod = oldCheckIdlePeriod
	})

	stream := newTestStream(t, rdb, "sink-claim")
	first := newTestSink(t, stream, "claimer",
		options.WithSinkStartAtOldest(),
		options.WithSinkAckGracePeriod(150*time.Millisecond))
	firstCh := first.Subscribe()

	id, err := stream.Add(ctx, "created", []byte("payload"))
	require.NoError(t, err)

	// Receive the event but never ack it, then close the sink instance so its
	// pending entry can be claimed by the next instance.
	ev := receiveEvent(t, firstCh)
	require.Equal(t, id, ev.ID)
	first.Close(ctx)

	second := newTestSink(t, stream, "claimer",
		options.WithSinkAckGracePeriod(150*time.Millisecond))
	secondCh := second.Subscribe()

	claimed := receiveEvent(t, secondCh)
	require.Equal(t, id, claimed.ID)
	require.Equal(t, "created", claimed.EventName)
	require.NoError(t, second.Ack(ctx, claimed))

	require.Eventually(t, func() bool {
		pending, err := rdb.XPending(ctx, streamKeyPrefix+"sink-claim", "claimer").Result()
		return err == nil && pending.Count == 0
	}, 10*time.Second, 5*time.Millisecond)
	second.Close(ctx)
}
