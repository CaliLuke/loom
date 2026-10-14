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
			writer := joinMap(t, rdb, "observation-cancel")
			committed, err := writer.runSetScript(t.Context(), "committed", "value")
			require.NoError(t, err)
			m := newPendingObservationMap(t, writer)
			gate := &observationReadGate{entered: make(chan struct{}, 1)}
			rdb.AddHook(gate)
			ctx, cancel := context.WithCancel(context.WithValue(t.Context(), observationContextKey{}, true))
			defer cancel()
			result := make(chan error, 1)
			m.wait.Add(1)
			go func() {
				defer m.wait.Done()
				result <- m.observe(observationRequest{ctx: ctx, generation: committed.generation, rev: committed.rev})
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
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(time.Second):
				t.Fatal("observation did not stop")
			}
			// A canceled observer cannot undo a committed remote write or close the client.
			value, err := rdb.HGet(t.Context(), m.hashkey, "committed").Result()
			require.NoError(t, err)
			require.Equal(t, "value", value)
			require.Empty(t, m.Map(), "failed observation must not publish partial state")
			if mode == "caller-cancel" {
				require.NoError(t, m.observe(observationRequest{ctx: t.Context(), generation: committed.generation, rev: committed.rev}))
				require.Equal(t, map[string]string{"committed": "value"}, m.Map())
			}
			_, err = writer.SetAndWait(t.Context(), "later", "works")
			require.NoError(t, err, "observer cancellation must not close the borrowed client")
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

// newPendingObservationMap isolates the observation owner from background
// publication. A real writer has committed a revision that this observer has
// deliberately not seen, so its next observation must enter the snapshot read.
func newPendingObservationMap(t *testing.T, writer *Map) *Map {
	t.Helper()
	closectx, closer := context.WithCancel(context.WithoutCancel(t.Context()))
	observer := &Map{
		Name:     writer.Name,
		hashkey:  writer.hashkey,
		rdb:      writer.rdb,
		content:  make(map[string]string),
		closectx: closectx,
		closer:   closer,
		done:     make(chan struct{}),
	}
	t.Cleanup(observer.Close)
	return observer
}
