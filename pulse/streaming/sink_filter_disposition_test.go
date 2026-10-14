package streaming

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

func TestSinkTerminalFilteredEventsAcknowledged(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "r4-filter")
	sink := newTestSink(t, stream, "r4f", options.WithSinkStartAtOldest(), options.WithSinkTopic("orders"))
	events := sink.Subscribe()
	for range 5 {
		_, err := stream.Add(ctx, "skipped", []byte("x"), options.WithTopic("payments"))
		require.NoError(t, err)
	}
	marker, err := stream.Add(ctx, "barrier", nil, options.WithTopic("orders"))
	require.NoError(t, err)
	event := receiveEvent(t, events)
	require.Equal(t, marker, event.ID)
	require.NoError(t, sink.Ack(ctx, event))
	pending, err := rdb.XPending(ctx, streamKeyPrefix+"r4-filter", "r4f").Result()
	require.NoError(t, err)
	require.Zero(t, pending.Count, "events the sink filtered out stay pending")
}

func TestSinkTerminalFilteredClaimAcknowledged(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "claim-filter")
	idle := make(chan time.Time)
	sink := newTestSinkWithRuntime(t, stream, "shared", sinkRuntime{idleChecks: idle}, options.WithSinkStartAtOldest(), options.WithSinkTopic("orders"))
	id, err := stream.Add(t.Context(), "payment", nil, options.WithTopic("payments"))
	require.NoError(t, err)
	_, err = rdb.XReadGroup(t.Context(), &redis.XReadGroupArgs{Group: sink.Name, Consumer: "previous", Streams: []string{stream.key, ">"}, Block: -1}).Result()
	require.NoError(t, err)
	sink.Subscribe()
	_, err = sink.claim(t.Context(), stream.Name, claimArgs{stream: stream.key, group: sink.Name, consumer: sinkConsumer(sink), minIdle: 0, start: "0-0"})
	require.NoError(t, err)
	pending, err := rdb.XPending(t.Context(), stream.key, sink.Name).Result()
	require.NoError(t, err)
	require.Zero(t, pending.Count, "claimed filtered entry %s remains pending without any application acknowledgment owner", id)
}
