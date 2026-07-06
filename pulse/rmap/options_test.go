package rmap

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseOptions(t *testing.T) {
	defaults := parseOptions()
	require.NotNil(t, defaults.Logger)
	require.Zero(t, defaults.TTL)
	require.False(t, defaults.TTLSliding)

	absolute := parseOptions(WithTTL(time.Minute))
	require.Equal(t, time.Minute, absolute.TTL)
	require.False(t, absolute.TTLSliding)

	sliding := parseOptions(WithSlidingTTL(2 * time.Minute))
	require.Equal(t, 2*time.Minute, sliding.TTL)
	require.True(t, sliding.TTLSliding)
}
