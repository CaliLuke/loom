package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseNodeOptions(t *testing.T) {
	defaults := parseOptions()
	require.Equal(t, 30*time.Second, defaults.workerTTL)
	require.Equal(t, 2*time.Minute, defaults.workerShutdownTTL)
	require.Equal(t, 5*time.Second, defaults.jobSinkBlockDuration)
	require.Equal(t, 1000, defaults.maxQueuedJobs)
	require.Equal(t, 20*time.Second, defaults.ackGracePeriod)
	require.False(t, defaults.clientOnly)
	require.NotNil(t, defaults.logger)

	opts := parseOptions(
		WithWorkerTTL(time.Second),
		WithWorkerShutdownTTL(2*time.Second),
		WithJobSinkBlockDuration(3*time.Second),
		WithMaxQueuedJobs(4),
		WithClientOnly(),
		WithAckGracePeriod(5*time.Second),
	)
	require.Equal(t, time.Second, opts.workerTTL)
	require.Equal(t, 2*time.Second, opts.workerShutdownTTL)
	require.Equal(t, 3*time.Second, opts.jobSinkBlockDuration)
	require.Equal(t, 4, opts.maxQueuedJobs)
	require.True(t, opts.clientOnly)
	require.Equal(t, 5*time.Second, opts.ackGracePeriod)
}

func TestParseTickerOptions(t *testing.T) {
	opts := parseTickerOptions()
	require.NotNil(t, opts.logger)
}
