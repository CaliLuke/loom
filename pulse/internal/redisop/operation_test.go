package redisop

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type readBudgetObserver struct {
	remaining time.Duration
	err       error
}

func TestOwnedBudgetsPreserveCallerOptions(t *testing.T) {
	for _, tc := range []struct {
		name              string
		read, write, want time.Duration
	}{
		{"disabled", -1, -1, Budget},
		{"short-read", 20 * time.Millisecond, time.Hour, 20 * time.Millisecond},
		{"short-write", time.Hour, 40 * time.Millisecond, 40 * time.Millisecond},
		{"long", time.Hour, time.Hour, Budget},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caller := redis.NewClient(&redis.Options{ReadTimeout: tc.read, WriteTimeout: tc.write, PoolTimeout: time.Hour, ContextTimeoutEnabled: false})
			t.Cleanup(func() {
				require.NoError(t, caller.Close())
			})
			before := *caller.Options()
			borrowed := Client(caller)
			require.Equal(t, tc.want, borrowed.Options().ReadTimeout)
			require.Equal(t, tc.want, borrowed.Options().WriteTimeout)
			require.Equal(t, before.PoolTimeout, borrowed.Options().PoolTimeout)
			require.False(t, borrowed.Options().ContextTimeoutEnabled)
			require.Equal(t, before.ReadTimeout, caller.Options().ReadTimeout)
			require.Equal(t, before.WriteTimeout, caller.Options().WriteTimeout)
			ctx, cancel := Context(t.Context(), caller)
			defer cancel()
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			require.WithinDuration(t, time.Now().Add(tc.want), deadline, 10*time.Millisecond)
			earlier, stop := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
			defer stop()
			bounded, release := Context(earlier, caller)
			defer release()
			require.ErrorIs(t, bounded.Err(), context.DeadlineExceeded)
		})
	}
}

func TestNativeBlockContract(t *testing.T) {
	for _, tc := range []struct{ configured, want time.Duration }{
		{-1, -1}, {0, Budget}, {time.Millisecond, time.Millisecond}, {Budget, Budget}, {time.Hour, Budget},
	} {
		require.Equal(t, tc.want, blockDuration(tc.configured))
	}
}

func TestReadReservesResponseBudget(t *testing.T) {
	for _, tc := range []struct {
		name              string
		block, read, want time.Duration
	}{
		{"zero", 0, -1, 2 * Budget},
		{"long", time.Hour, -1, 2 * Budget},
		{"short", time.Millisecond, -1, Budget + time.Millisecond},
		{"nonblocking", -1, -1, Budget},
		{"stricter-caller", 0, 200 * time.Millisecond, 200 * time.Millisecond},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sentinel := errors.New("observed dial deadline")
			observer := &readBudgetObserver{err: sentinel}
			caller := redis.NewClient(&redis.Options{ReadTimeout: tc.read, WriteTimeout: -1, MaxRetries: -1})
			t.Cleanup(func() {
				require.NoError(t, caller.Close())
			})
			borrowed, block := Read(caller, tc.block)
			require.Equal(t, blockDuration(tc.block), block)
			borrowed.AddHook(observer)
			require.ErrorIs(t, borrowed.Ping(t.Context()).Err(), sentinel)
			require.InDelta(t, float64(tc.want), float64(observer.remaining), float64(50*time.Millisecond))
			require.LessOrEqual(t, borrowed.Options().ReadTimeout, Budget)
		})
	}
}

func (h *readBudgetObserver) DialHook(next redis.DialHook) redis.DialHook {
	return next
}
func (h *readBudgetObserver) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *readBudgetObserver) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			return errors.New("missing operation deadline")
		}
		h.remaining = time.Until(deadline)
		return h.err
	}
}
