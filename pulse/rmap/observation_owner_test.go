package rmap

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type observationContextKey struct{}

type observationReadGate struct {
	entered chan struct{}
}

func TestReplicaObservationCancellationAndClose(t *testing.T) {
	for _, mode := range []string{"caller-cancel", "map-close"} {
		t.Run(mode, func(t *testing.T) {
			rdb := startTestRedis(t)
			m := joinMap(t, rdb, "observation-cancel")
			require.NoError(t, m.sub.Unsubscribe(t.Context(), m.chankey))
			gate := &observationReadGate{entered: make(chan struct{}, 1)}
			rdb.AddHook(gate)
			ctx, cancel := context.WithCancel(context.WithValue(t.Context(), observationContextKey{}, true))
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := m.SetAndWait(ctx, "committed", "value")
				result <- err
			}()
			select {
			case <-gate.entered:
			case <-time.After(time.Second):
				t.Fatal("post-write observation did not enter")
			}
			if mode == "caller-cancel" {
				cancel()
			} else {
				m.Close()
			}
			select {
			case err := <-result:
				require.Error(t, err)
				if mode == "caller-cancel" {
					require.ErrorIs(t, err, context.Canceled)
				}
			case <-time.After(time.Second):
				t.Fatal("observation did not stop")
			}
			// A canceled observer cannot undo a committed remote write or close the client.
			value, err := rdb.HGet(t.Context(), m.hashkey, "committed").Result()
			require.NoError(t, err)
			require.Equal(t, "value", value)
			if mode == "caller-cancel" {
				_, err = m.SetAndWait(t.Context(), "later", "works")
				require.NoError(t, err, "canceled observation must release the sole publisher")
			}
		})
	}
}

func TestReplicaConcurrentWriteObservations(t *testing.T) {
	rdb := startTestRedis(t)
	m := joinMap(t, rdb, "concurrent-observe")
	require.NoError(t, m.sub.Unsubscribe(t.Context(), m.chankey))
	results := make(chan error, 24)
	for i := range 24 {
		go func() {
			key := fmt.Sprintf("key-%d", i/2)
			_, err := m.SetAndWait(t.Context(), key, "same")
			results <- err
		}()
	}
	for range 24 {
		select {
		case err := <-results:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent observation did not finish")
		}
	}
	require.Len(t, m.Map(), 12)
}

func (h *observationReadGate) DialHook(next redis.DialHook) redis.DialHook {
	return next
}
func (h *observationReadGate) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *observationReadGate) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "hgetall" && ctx.Value(observationContextKey{}) == true {
			h.entered <- struct{}{}
			<-ctx.Done()
			return ctx.Err()
		}
		return next(ctx, cmd)
	}
}
