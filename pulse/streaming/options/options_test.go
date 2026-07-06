package options

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseStreamOptions(t *testing.T) {
	defaults := ParseStreamOptions()
	require.Equal(t, 1000, defaults.MaxLen)
	require.Zero(t, defaults.TTL)
	require.False(t, defaults.TTLSliding)
	require.NotNil(t, defaults.Logger)

	absolute := ParseStreamOptions(WithStreamMaxLen(25), WithStreamTTL(time.Minute))
	require.Equal(t, 25, absolute.MaxLen)
	require.Equal(t, time.Minute, absolute.TTL)
	require.False(t, absolute.TTLSliding)

	sliding := ParseStreamOptions(WithStreamSlidingTTL(2 * time.Minute))
	require.Equal(t, 2*time.Minute, sliding.TTL)
	require.True(t, sliding.TTLSliding)
}

func TestParseReaderOptions(t *testing.T) {
	startAt := time.UnixMilli(1234)
	opts := ParseReaderOptions(
		WithReaderBlockDuration(time.Second),
		WithReaderMaxPolled(7),
		WithReaderTopic("orders"),
		WithReaderTopicPattern("^orders:"),
		WithReaderBufferSize(8),
		WithReaderStartAt(startAt),
	)

	require.Equal(t, time.Second, opts.BlockDuration)
	require.Equal(t, int64(7), opts.MaxPolled)
	require.Equal(t, "orders", opts.Topic)
	require.Equal(t, "^orders:", opts.TopicPattern)
	require.Equal(t, 8, opts.BufferSize)
	require.Equal(t, "1234-0", opts.LastEventID)

	require.Equal(t, "0", ParseReaderOptions(WithReaderStartAtOldest()).LastEventID)
	require.Equal(t, "$", ParseReaderOptions(WithReaderStartAtNewest()).LastEventID)
	require.Equal(t, "10-0", ParseReaderOptions(WithReaderStartAfter("10-0")).LastEventID)
}

func TestParseSinkOptions(t *testing.T) {
	startAt := time.UnixMilli(5678)
	opts := ParseSinkOptions(
		WithSinkBlockDuration(time.Second),
		WithSinkMaxPolled(11),
		WithSinkTopic("orders"),
		WithSinkTopicPattern("^orders:"),
		WithSinkBufferSize(12),
		WithSinkStartAt(startAt),
		WithSinkNoAck(),
		WithSinkAckGracePeriod(3*time.Second),
	)

	require.Equal(t, time.Second, opts.BlockDuration)
	require.Equal(t, int64(11), opts.MaxPolled)
	require.Equal(t, "orders", opts.Topic)
	require.Equal(t, "^orders:", opts.TopicPattern)
	require.Equal(t, 12, opts.BufferSize)
	require.Equal(t, "5678-0", opts.LastEventID)
	require.True(t, opts.NoAck)
	require.Equal(t, 3*time.Second, opts.AckGracePeriod)

	require.Equal(t, "0", ParseSinkOptions(WithSinkStartAtOldest()).LastEventID)
	require.Equal(t, "$", ParseSinkOptions(WithSinkStartAtNewest()).LastEventID)
	require.Equal(t, "10-0", ParseSinkOptions(WithSinkStartAfter("10-0")).LastEventID)
}

func TestParseAddEventOptions(t *testing.T) {
	opts := ParseAddEventOptions(WithTopic("orders"), WithOnlyIfStreamExists())

	require.Equal(t, "orders", opts.Topic)
	require.True(t, opts.OnlyIfStreamExists)
}
