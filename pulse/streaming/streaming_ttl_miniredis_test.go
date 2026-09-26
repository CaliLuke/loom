package streaming

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/internal/redistest"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

const (
	// testTTL is the stream TTL of the TTL tests.
	testTTL = time.Hour
	// testShortTTL is the remaining TTL the TTL tests give the stream key
	// before a write, as if most of testTTL had elapsed.
	testShortTTL = 10 * time.Minute
)

// ttlModes are the stream TTL modes. extended reports whether a write
// extends the remaining TTL.
var ttlModes = []struct {
	name     string
	opt      options.Stream
	extended bool
}{
	{name: "fixed", opt: options.WithStreamTTL(testTTL)},
	{name: "sliding", opt: options.WithStreamSlidingTTL(testTTL), extended: true},
}

// TestStreamAddAppliesTTL checks that Add sets the stream TTL in the same
// script as the XADD, on Redis 6.2 and later. A fixed TTL is set by the first
// Add and never extended by later ones. A sliding TTL is reset by every Add.
func TestStreamAddAppliesTTL(t *testing.T) {
	for _, mode := range ttlModes {
		t.Run(mode.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			ctx := t.Context()
			rec := redistest.Record(rdb)
			stream, err := NewStream("ttl-add-"+mode.name, rdb, mode.opt, options.WithStreamMaxLen(10))
			require.NoError(t, err)

			first, err := stream.Add(ctx, "created", []byte("payload"), options.WithTopic("topic"))
			require.NoError(t, err)
			require.NotEmpty(t, first)
			assertTTLBetween(t, rdb, stream.key, testShortTTL, testTTL)

			// Pretend most of the TTL elapsed, then add again.
			require.NoError(t, rdb.PExpire(ctx, stream.key, testShortTTL).Err())
			rec.Reset()
			second, err := stream.Add(ctx, "updated", []byte("payload"))
			require.NoError(t, err)
			require.NotEmpty(t, second)
			assertNoSeparateExpiry(t, rec.Names())
			if mode.extended {
				assertTTLBetween(t, rdb, stream.key, testShortTTL, testTTL)
			} else {
				assertTTLBetween(t, rdb, stream.key, 0, testShortTTL)
			}

			entries, err := rdb.XRange(ctx, stream.key, "-", "+").Result()
			require.NoError(t, err)
			require.Len(t, entries, 2)
			assert.Equal(t, first, entries[0].ID)
			assert.Equal(t, map[string]any{nameKey: "created", payloadKey: "payload", topicKey: "topic"}, entries[0].Values)
			assert.Equal(t, map[string]any{nameKey: "updated", payloadKey: "payload"}, entries[1].Values)
		})
	}
}

// TestNewSinkAppliesTTL checks that NewSink sets the stream TTL in the same
// script that creates the consumer group, and that a fixed TTL is not
// extended by a later sink.
func TestNewSinkAppliesTTL(t *testing.T) {
	for _, mode := range ttlModes {
		t.Run(mode.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			ctx := t.Context()
			rec := redistest.Record(rdb)
			stream, err := NewStream("ttl-sink-"+mode.name, rdb, mode.opt)
			require.NoError(t, err)

			newTestSink(t, stream, "first")
			assertTTLBetween(t, rdb, stream.key, testShortTTL, testTTL)

			require.NoError(t, rdb.PExpire(ctx, stream.key, testShortTTL).Err())
			rec.Reset()
			newTestSink(t, stream, "second")
			// The same group again takes the BUSYGROUP path.
			newTestSink(t, stream, "second")
			assertNoSeparateExpiry(t, rec.Names())
			if mode.extended {
				assertTTLBetween(t, rdb, stream.key, testShortTTL, testTTL)
			} else {
				assertTTLBetween(t, rdb, stream.key, 0, testShortTTL)
			}
		})
	}
}

// TestNewSinkReportsGroupCreateError checks that NewSink returns the error of
// a failed consumer group creation and applies no TTL then.
func TestNewSinkReportsGroupCreateError(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	stream, err := NewStream("ttl-sink-error", rdb, options.WithStreamTTL(testTTL))
	require.NoError(t, err)
	require.NoError(t, rdb.Set(ctx, stream.key, "not a stream", 0).Err())

	_, err = stream.NewSink(ctx, "sink")
	require.ErrorContains(t, err, "failed to create Redis consumer group sink")
	require.ErrorContains(t, err, "WRONGTYPE")
	ttl, err := rdb.PTTL(ctx, stream.key).Result()
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl)
}

// TestSinkAddStreamAppliesTTL checks that Sink.AddStream sets the TTL of a
// stream it creates in the same script that creates the consumer group, and
// that a fixed TTL is not extended when AddStream joins an existing stream.
func TestSinkAddStreamAppliesTTL(t *testing.T) {
	for _, mode := range ttlModes {
		t.Run(mode.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			ctx := t.Context()
			rec := redistest.Record(rdb)
			main, err := NewStream("ttl-addstream-main-"+mode.name, rdb)
			require.NoError(t, err)
			first, err := NewStream("ttl-addstream-first-"+mode.name, rdb, mode.opt)
			require.NoError(t, err)
			sink := newTestSink(t, main, "sink")

			rec.Reset()
			require.NoError(t, sink.AddStream(ctx, first))
			assertNoSeparateExpiry(t, rec.Names())
			assertTTLBetween(t, rdb, first.key, testShortTTL, testTTL)

			// A second sink joins the existing stream after most of the
			// TTL elapsed.
			require.NoError(t, rdb.PExpire(ctx, first.key, testShortTTL).Err())
			other := newTestSink(t, main, "other")
			rec.Reset()
			require.NoError(t, other.AddStream(ctx, first))
			assertNoSeparateExpiry(t, rec.Names())
			if mode.extended {
				assertTTLBetween(t, rdb, first.key, testShortTTL, testTTL)
			} else {
				assertTTLBetween(t, rdb, first.key, 0, testShortTTL)
			}
		})
	}
}

// TestSinkAddStreamReportsGroupCreateError checks that Sink.AddStream returns
// the error of a failed consumer group creation of a TTL stream, applies no
// TTL then, and does not add the stream to the sink or its consumer to the
// consumers map of the stream.
func TestSinkAddStreamReportsGroupCreateError(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	main, err := NewStream("ttl-addstream-error-main", rdb)
	require.NoError(t, err)
	stream, err := NewStream("ttl-addstream-error", rdb, options.WithStreamTTL(testTTL))
	require.NoError(t, err)
	require.NoError(t, rdb.Set(ctx, stream.key, "not a stream", 0).Err())
	sink := newTestSink(t, main, "sink")

	err = sink.AddStream(ctx, stream)
	require.ErrorContains(t, err, "failed to create Redis consumer group sink")
	require.ErrorContains(t, err, "WRONGTYPE")
	ttl, err := rdb.PTTL(ctx, stream.key).Result()
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl)
	assert.Len(t, sink.streams, 1)
	assert.Empty(t, streamConsumers(t, rdb, stream))
}

// TestAddOnlyIfStreamExistsWithTTL checks that Add with
// WithOnlyIfStreamExists neither creates a missing stream nor fails when the
// stream has a TTL.
func TestAddOnlyIfStreamExistsWithTTL(t *testing.T) {
	for _, mode := range ttlModes {
		t.Run(mode.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			ctx := t.Context()
			stream, err := NewStream("ttl-only-exists-"+mode.name, rdb, mode.opt)
			require.NoError(t, err)

			id, err := stream.Add(ctx, "created", []byte("payload"), options.WithOnlyIfStreamExists())
			require.NoError(t, err)
			assert.Empty(t, id)
			exists, err := rdb.Exists(ctx, stream.key).Result()
			require.NoError(t, err)
			assert.Zero(t, exists)
		})
	}
}

// assertTTLBetween asserts that key has a remaining TTL in (low, high].
func assertTTLBetween(t *testing.T, rdb *redis.Client, key string, low, high time.Duration) {
	t.Helper()
	ttl, err := rdb.PTTL(t.Context(), key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, low, "TTL of %s", key)
	assert.LessOrEqual(t, ttl, high, "TTL of %s", key)
}

// assertNoSeparateExpiry asserts that names, the commands of a stream write,
// hold no expiry command: the expiry must be part of the write script.
func assertNoSeparateExpiry(t *testing.T, names []string) {
	t.Helper()
	require.NotEmpty(t, names)
	for _, name := range names {
		assert.NotContains(t, []string{"expire", "pexpire", "expireat", "pexpireat"}, name, "commands %v", names)
	}
}
