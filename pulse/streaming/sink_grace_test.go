package streaming

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

func TestSinkPositiveGraceRetainsLiveConsumer(t *testing.T) {
	for _, path := range []string{"rotation", "cleanup"} {
		t.Run(path, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "positive-grace")
			sink := newTestSink(t, stream, "grace", options.WithSinkAckGracePeriod(time.Duration(math.MaxInt64)))
			consumer := sink.consumer
			record, _, err := readSinkMembership(t.Context(), stream, sink.Name)
			require.NoError(t, err)
			var member sinkMember
			for _, candidate := range record.Members {
				if candidate.Name == consumer {
					member = candidate
				}
			}
			require.Positive(t, member.LastKeepAlive, "constructor heartbeat must be in the membership authority")
			if path == "rotation" {
				require.NoError(t, sink.ensureConsumer(t.Context()))
				require.Equal(t, consumer, sink.consumer, "fresh consumer must not rotate within a positive grace")
			} else {
				sink.lock.Lock()
				err := sink.deleteStreamStaleConsumers(t.Context(), stream)
				sink.lock.Unlock()
				require.NoError(t, err)
				consumers, err := rdb.XInfoConsumers(t.Context(), stream.key, sink.Name).Result()
				require.NoError(t, err)
				require.Len(t, consumers, 1, "fresh consumer must not be deleted within a positive grace")
				require.Equal(t, consumer, consumers[0].Name)
			}
		})
	}
}

func TestConsumerKeepAliveExpiryThreshold(t *testing.T) {
	for _, tc := range []struct {
		name           string
		elapsed, grace time.Duration
		stale          bool
	}{
		{name: "future timestamp", elapsed: -1, grace: 1},
		{name: "fresh", elapsed: 0, grace: 1},
		{name: "first grace", elapsed: 1, grace: 1},
		{name: "equal threshold", elapsed: 2, grace: 1},
		{name: "just expired", elapsed: 3, grace: 1, stale: true},
		{name: "normal just live", elapsed: 2*time.Second - 1, grace: time.Second},
		{name: "normal expired", elapsed: 2*time.Second + 1, grace: time.Second, stale: true},
		{name: "largest representable equality", elapsed: math.MaxInt64 - 1, grace: math.MaxInt64 / 2},
		{name: "largest representable expired", elapsed: math.MaxInt64, grace: math.MaxInt64 / 2, stale: true},
		{name: "threshold beyond duration", elapsed: math.MaxInt64, grace: math.MaxInt64/2 + 1},
		{name: "maximum grace", elapsed: math.MaxInt64, grace: math.MaxInt64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.stale, consumerKeepAliveExpired(tc.elapsed, tc.grace))
		})
	}
}
