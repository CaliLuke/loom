//go:build unix

package testdatacompile

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCommandTimeoutEndsDescendants proves that cancellation closes child pipes.
func TestCommandTimeoutEndsDescendants(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := runCommand(ctx, t.TempDir(), nil, "sh", "-c", "sleep 10 & wait")
	require.Error(t, err)
	require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	require.Less(t, time.Since(start), 2*time.Second, "a surviving child retains the output pipe")
}
