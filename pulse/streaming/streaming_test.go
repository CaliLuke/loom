package streaming

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

func TestNewStreamBuildsLocalMetadataWithoutRedis(t *testing.T) {
	stream, err := NewStream("events", nil,
		options.WithStreamMaxLen(42),
		options.WithStreamSlidingTTL(30*time.Second),
	)

	require.NoError(t, err)
	require.Equal(t, "events", stream.Name)
	require.Equal(t, 42, stream.MaxLen)
	require.Equal(t, 30*time.Second, stream.ttl)
	require.True(t, stream.ttlSliding)
	require.Equal(t, streamKeyPrefix+"events", stream.key)
}

func TestNewStreamValidatesNameAndTTL(t *testing.T) {
	_, err := NewStream("bad name", nil)
	require.ErrorContains(t, err, "not a valid name")

	_, err = NewStream("events", nil, options.WithStreamTTL(-time.Second))
	require.ErrorContains(t, err, "ttl must be >= 0")
}

func TestStreamKeyValidation(t *testing.T) {
	require.True(t, isValidRedisKeyName("events:tenant-1"))
	require.False(t, isValidRedisKeyName(""))
	require.False(t, isValidRedisKeyName("events tenant"))
	require.False(t, isValidRedisKeyName(strings.Repeat("a", 513)))
}

func TestSinkNameHelpers(t *testing.T) {
	stream := &Stream{Name: "events"}

	require.Equal(t, "stream:events:sinks", consumersMapName(stream))
	require.Equal(t, "sink:orders:keepalive", sinkKeepAliveMapName("orders"))
	require.Equal(t, "sink:orders:stalelease", staleLockName("orders"))
}

func TestIsBusyGroupErr(t *testing.T) {
	require.False(t, isBusyGroupErr(nil))
	require.False(t, isBusyGroupErr(errors.New("some other redis error")))
	require.True(t, isBusyGroupErr(errors.New("BUSYGROUP Consumer Group name already exists")))
}
