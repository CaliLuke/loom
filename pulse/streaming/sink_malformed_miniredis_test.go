package streaming

import (
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

func TestSinkAcksMalformedEntries(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	stream := newTestStream(t, rdb, "sink-malformed")
	sink := newTestSink(t, stream, "malformed", options.WithSinkStartAtOldest())
	events := sink.Subscribe()

	malformedID, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream.key,
		Values: map[string]any{"unexpected": "field"},
	}).Result()
	require.NoError(t, err)
	validID, err := stream.Add(ctx, "created", []byte("payload"))
	require.NoError(t, err)

	event := receiveEvent(t, events)
	require.Equal(t, validID, event.ID)
	require.Equal(t, "created", event.EventName)
	require.NoError(t, sink.Ack(ctx, event))

	require.Eventually(t, func() bool {
		pending, err := rdb.XPending(ctx, stream.key, sink.Name).Result()
		return err == nil && pending.Count == 0
	}, 5*time.Second, 20*time.Millisecond, "malformed entry %s stayed pending", malformedID)
}
