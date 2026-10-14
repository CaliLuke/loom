package rmap

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type reconnectSnapshotGate struct {
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func TestCloseDuringReconnectSnapshot(t *testing.T) {
	rdb := startTestRedis(t)
	sm := joinMap(t, rdb, "close-reconnect")
	ch := sm.Subscribe()
	gate := &reconnectSnapshotGate{entered: make(chan struct{}), release: make(chan struct{})}
	rdb.AddHook(gate)
	// Closing the old PubSub makes the replica own a replacement subscription.
	require.NoError(t, sm.sub.Close())
	select {
	case <-gate.entered:
	case <-time.After(time.Second):
		t.Fatal("replacement snapshot did not enter")
	}
	done := make(chan struct{})
	var release sync.Once
	t.Cleanup(func() {
		release.Do(func() {
			close(gate.release)
		})
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("Close did not join reconnect cleanup")
		}
	})
	go func() {
		sm.Close()
		close(done)
	}()
	<-sm.closectx.Done()
	select {
	case <-done:
		t.Error("Close abandoned replacement subscription")
	default:
	}
	release.Do(func() {
		close(gate.release)
	})
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close did not finish replacement cleanup")
	}
	select {
	case _, open := <-ch:
		require.False(t, open)
	case <-time.After(time.Second):
		t.Fatal("subscriber remained open")
	}
	require.Eventually(t, func() bool {
		counts, err := rdb.PubSubNumSub(t.Context(), sm.chankey).Result()
		require.NoError(t, err)
		return counts[sm.chankey] == 0
	}, time.Second, time.Millisecond)
	require.NoError(t, rdb.Ping(t.Context()).Err())
}

func (h *reconnectSnapshotGate) DialHook(next redis.DialHook) redis.DialHook {
	return next
}
func (h *reconnectSnapshotGate) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *reconnectSnapshotGate) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "hgetall" {
			h.once.Do(func() {
				close(h.entered)
				<-h.release
			})
		}
		return next(ctx, cmd)
	}
}
