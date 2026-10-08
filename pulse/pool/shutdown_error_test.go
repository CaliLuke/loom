package pool

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestShutdownReturnsRequestError(t *testing.T) {
	for _, tc := range []struct {
		name     string
		deadline bool
		want     error
	}{
		{name: "cancelled", want: context.Canceled},
		{name: "expired", deadline: true, want: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			node := addTestNode(t, rdb, "shutdown-error-"+tc.name)
			ctx, cancel := context.WithCancel(t.Context())
			if tc.deadline {
				cancel()
				ctx, cancel = context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
			} else {
				cancel()
			}
			defer cancel()
			result := make(chan error, 1)
			go func() {
				result <- node.Shutdown(ctx)
			}()
			select {
			case err := <-result:
				require.ErrorIs(t, err, tc.want)
				require.False(t, node.IsClosed())
				require.Zero(t, node.nodeShutdownMap.Len())
			case <-time.After(2 * time.Second):
				// Release the original unbounded wait so a failing regression
				// cannot leak the Shutdown goroutine into the next test.
				require.NoError(t, node.Close(context.Background()))
				<-result
				t.Error("Shutdown waited for closure after its request failed")
			}
		})
	}
}
