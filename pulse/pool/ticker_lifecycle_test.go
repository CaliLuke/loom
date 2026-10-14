package pool

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
)

type (
	tickerParentContext struct {
		context.Context
		done   chan struct{}
		active atomic.Int32
	}
	tickerWriteFailure struct {
		after bool
		used  atomic.Bool
	}
)

var errTickerWrite = errors.New("injected ticker write failure")

func TestTickerFailedConstructionReleasesLocalOwnership(t *testing.T) {
	node, parent := tickerLifecycleNode(t)
	baseline := tickerSubscriberCount(node.tickerMap)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for range 3 {
		ticker, err := node.NewTicker(ctx, "failed", time.Hour)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, ticker)
		assert.Equal(t, baseline, tickerSubscriberCount(node.tickerMap))
		assert.Zero(t, parent.active.Load(), "failed construction must release its parent cancellation callback")
	}
}

func TestTickerSelectedTickAfterCloseDoesNotRearm(t *testing.T) {
	node, _ := tickerLifecycleNode(t)
	ticker := tickerLifecycleParticipant(t, node, "closed")
	ticker.Close()
	ticker.handleTick()
	assert.False(t, ticker.timer.Stop(), "a timer event already selected before Close must not rearm")
}

func TestTickerTerminalOwnerRejectsRearm(t *testing.T) {
	node, _ := tickerLifecycleNode(t)
	ticker := tickerLifecycleParticipant(t, node, "notice")
	ticker.Close()
	// The map notification path can have read its next value before Close
	// acquires the ticker lock. Every rearm goes through this same owner.
	ticker.lock.Lock()
	ticker.initTimer()
	ticker.lock.Unlock()
	assert.False(t, ticker.timer.Stop(), "terminal ownership must reject rearm from a preselected notification")
}

func TestTickerLocalClosePreservesSharedEntry(t *testing.T) {
	node, parent := tickerLifecycleNode(t)
	ticker, err := node.NewTicker(t.Context(), "healthy", time.Hour)
	require.NoError(t, err)
	t.Cleanup(ticker.Close)
	require.Equal(t, 1, tickerSubscriberCount(node.tickerMap))
	require.Equal(t, int32(1), parent.active.Load())
	ticker.Close()
	ticker.Close()
	assert.Zero(t, tickerSubscriberCount(node.tickerMap))
	assert.Zero(t, parent.active.Load())
	_, exists, _, err := node.tickerMap.TestAndSetEx(t.Context(), ticker.name, "does-not-match", "unused")
	require.NoError(t, err)
	assert.True(t, exists, "local Close must preserve the shared entry")
}

func TestTickerFailedCommittedConstructionPreservesPeer(t *testing.T) {
	node, parent := tickerLifecycleNode(t)
	peer, err := node.NewTicker(t.Context(), "shared", time.Hour)
	require.NoError(t, err)
	t.Cleanup(peer.Close)
	node.rdb.AddHook(&tickerWriteFailure{after: true})
	failed, err := node.NewTicker(t.Context(), "shared", 2*time.Hour)
	assert.ErrorIs(t, err, errTickerWrite)
	assert.Nil(t, failed)
	assert.Equal(t, 1, tickerSubscriberCount(node.tickerMap))
	assert.Equal(t, int32(1), parent.active.Load())
	value, exists, _, err := node.tickerMap.TestAndSetEx(t.Context(), peer.name, "does-not-match", "unused")
	require.NoError(t, err)
	require.True(t, exists, "uncertain constructor abort must not delete shared state")
	_, interval := deserialize(value)
	assert.Equal(t, 2*time.Hour, interval, "injected failure follows the committed mutation")
	assert.NoError(t, peer.runtimeCtx.Err())
}

func TestTickerTransientAdvanceFailureRemainsRetryable(t *testing.T) {
	node, _ := tickerLifecycleNode(t)
	ticker := tickerLifecycleParticipant(t, node, "retry")
	ticker.timer.Stop()
	node.rdb.AddHook(&tickerWriteFailure{})
	ticker.handleTick()
	assert.True(t, ticker.timer.Stop(), "a live participant must rearm after transient failure")
	assert.NoError(t, ticker.runtimeCtx.Err())
	assert.Equal(t, 1, tickerSubscriberCount(node.tickerMap))
}

func TestTickerGlobalStopReleasesLocalOwnership(t *testing.T) {
	node, parent := tickerLifecycleNode(t)
	ticker := tickerLifecycleParticipant(t, node, "global")
	ticker.Stop()
	_, exists, _, err := node.tickerMap.TestAndSetEx(t.Context(), ticker.name, "does-not-match", "unused")
	require.NoError(t, err)
	assert.False(t, exists)
	assert.ErrorIs(t, ticker.runtimeCtx.Err(), context.Canceled)
	assert.Zero(t, tickerSubscriberCount(node.tickerMap))
	assert.Zero(t, parent.active.Load())
}

func TestTickerFailedGlobalDeleteStillReleasesLocalOwnership(t *testing.T) {
	node, parent := tickerLifecycleNode(t)
	ticker := tickerLifecycleParticipant(t, node, "failed-stop")
	node.rdb.AddHook(&tickerWriteFailure{})
	ticker.Stop()
	assert.ErrorIs(t, ticker.runtimeCtx.Err(), context.Canceled)
	assert.Zero(t, tickerSubscriberCount(node.tickerMap))
	assert.Zero(t, parent.active.Load())
	assert.False(t, ticker.timer.Stop())
	_, exists, _, err := node.tickerMap.TestAndSetEx(t.Context(), ticker.name, "does-not-match", "unused")
	require.NoError(t, err)
	assert.True(t, exists, "a failed global delete does not imply shared state was removed")
}

func TestTickerRemoteAbsenceEndsLocalOwnership(t *testing.T) {
	node, parent := tickerLifecycleNode(t)
	ticker := tickerLifecycleParticipant(t, node, "remote")
	_, err := node.tickerMap.Delete(t.Context(), ticker.name)
	require.NoError(t, err)
	// The timer event is selected before the map deletion notification is
	// processed. This participant has no event goroutine, fixing that ordering.
	assert.NotPanics(t, ticker.handleTick)
	assert.ErrorIs(t, ticker.runtimeCtx.Err(), context.Canceled)
	assert.Zero(t, tickerSubscriberCount(node.tickerMap))
	assert.Zero(t, parent.active.Load())
}

func TestTickerEventLoopExitReleasesLocalOwnership(t *testing.T) {
	for _, reason := range []string{"runtime cancellation", "map closure", "remote deletion"} {
		t.Run(reason, func(t *testing.T) {
			node, parent := tickerLifecycleNode(t)
			ticker := tickerLifecycleParticipant(t, node, "event-exit")
			switch reason {
			case "runtime cancellation":
				ticker.runtimeCancel()
			case "map closure":
				node.tickerMap.Close()
			case "remote deletion":
				_, err := node.tickerMap.Delete(t.Context(), ticker.name)
				require.NoError(t, err)
				require.Eventually(t, func() bool {
					_, exists := node.tickerMap.Get(ticker.name)
					return !exists
				}, time.Second, time.Millisecond)
			}
			ticker.wg.Add(1)
			done := make(chan struct{})
			go func() {
				ticker.handleEvents()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("event loop must release ownership without waiting on itself")
			}
			assert.Nil(t, ticker.mapch)
			assert.False(t, ticker.timer.Stop())
			assert.ErrorIs(t, ticker.runtimeCtx.Err(), context.Canceled)
			assert.Zero(t, parent.active.Load())
		})
	}
}

func TestTickerAdvanceMismatchKeepsLiveParticipant(t *testing.T) {
	node, parent := tickerLifecycleNode(t)
	ticker := tickerLifecycleParticipant(t, node, "mismatch")
	next := serialize(time.Now().Add(2*time.Hour), time.Hour)
	_, err := node.tickerMap.SetAndWait(t.Context(), ticker.name, next)
	require.NoError(t, err)
	ticker.handleTick()
	assert.Equal(t, next, ticker.next)
	assert.NoError(t, ticker.runtimeCtx.Err())
	assert.Equal(t, int32(1), parent.active.Load())
	assert.True(t, ticker.timer.Stop())
}

func tickerLifecycleNode(t *testing.T) (*Node, *tickerParentContext) {
	t.Helper()
	rdb := startTestRedis(t)
	mapped, err := rmap.Join(t.Context(), "ticker-lifecycle", rdb)
	require.NoError(t, err)
	t.Cleanup(mapped.Close)
	parent := &tickerParentContext{Context: context.Background(), done: make(chan struct{})}
	return &Node{PoolName: "ticker-lifecycle", tickerMap: mapped, runtimeCtx: parent, rdb: rdb}, parent
}

func tickerLifecycleParticipant(t *testing.T, node *Node, name string) *Ticker {
	t.Helper()
	runtimeCtx, cancel := context.WithCancel(node.runtimeCtx)
	c := make(chan time.Time)
	ticker := &Ticker{
		C: c, c: c, name: node.PoolName + ":" + name,
		tickerMap: node.tickerMap, next: serialize(time.Now().Add(time.Hour), time.Hour),
		mapch: node.tickerMap.Subscribe(), wg: &sync.WaitGroup{}, logger: pulse.NoopLogger(),
		runtimeCtx: runtimeCtx, runtimeCancel: cancel,
	}
	_, err := node.tickerMap.SetAndWait(t.Context(), ticker.name, ticker.next)
	require.NoError(t, err)
	ticker.initTimer()
	t.Cleanup(ticker.Close)
	return ticker
}

func tickerSubscriberCount(mapped *rmap.Map) int {
	// Only the test goroutine adds/removes subscriptions in these direct
	// probes; the map loop only reads the slice until registered cleanup.
	return reflect.ValueOf(mapped).Elem().FieldByName("chans").Len()
}

func (c *tickerParentContext) Done() <-chan struct{} {
	return c.done
}

func (c *tickerParentContext) AfterFunc(func()) func() bool {
	c.active.Add(1)
	var stopped atomic.Bool
	return func() bool {
		if !stopped.CompareAndSwap(false, true) {
			return false
		}
		c.active.Add(-1)
		return true
	}
}

func (h *tickerWriteFailure) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *tickerWriteFailure) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *tickerWriteFailure) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, command redis.Cmder) error {
		if command.Name() != "evalsha" || !h.used.CompareAndSwap(false, true) {
			return next(ctx, command)
		}
		if h.after {
			if err := next(ctx, command); err != nil {
				return err
			}
		}
		return errTickerWrite
	}
}
