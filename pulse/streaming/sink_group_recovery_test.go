package streaming

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type missingGroupGate struct {
	limit   int32
	count   atomic.Int32
	reached chan struct{}
	release chan struct{}
	once    sync.Once
}

func TestMissingGroupRecoveryAfterDestroy(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "missing-group")
	gate := &missingGroupGate{limit: 32, reached: make(chan struct{}), release: make(chan struct{})}
	rdb.AddHook(gate)
	sink := newTestSink(t, stream, "recovery", options.WithSinkStartAtOldest())
	t.Cleanup(func() {
		close(gate.release)
	})
	ch := sink.Subscribe()
	before, err := stream.Add(t.Context(), "before", nil)
	require.NoError(t, err)
	event := receiveEvent(t, ch)
	require.Equal(t, before, event.ID)
	require.NoError(t, sink.Ack(t.Context(), event))
	require.NoError(t, stream.Destroy(t.Context()))
	after, err := stream.Add(t.Context(), "after", nil)
	require.NoError(t, err)
	select {
	case event := <-ch:
		require.NotNil(t, event)
		require.Equal(t, after, event.ID)
		require.NoError(t, sink.Ack(t.Context(), event))
	case <-gate.reached:
		t.Errorf("reached %d consecutive NOGROUP reads without repairing the active group", gate.count.Load())
	case <-time.After(3 * time.Second):
		t.Error("no post-destroy delivery within the bounded probe")
	}
}

func TestMissingGroupRemovedRegistrationStaysRemoved(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "removed-group")
	gate := &missingGroupGate{limit: 1, reached: make(chan struct{}), release: make(chan struct{})}
	rdb.AddHook(gate)
	sink := newTestSink(t, stream, "recovery", options.WithSinkStartAtOldest())
	t.Cleanup(func() {
		close(gate.release)
	})
	// Activation is explicit, but the group's deletion precedes the first read.
	require.NoError(t, rdb.XGroupDestroy(t.Context(), stream.key, sink.Name).Err())
	sink.Subscribe()
	select {
	case <-gate.reached:
	case <-time.After(time.Second):
		t.Fatal("read did not observe missing group")
	}
	require.NoError(t, sink.RemoveStream(t.Context(), stream))
	sink.Close(context.Background())
	groups, err := rdb.XInfoGroups(t.Context(), stream.key).Result()
	require.NoError(t, err)
	require.Empty(t, groups, "a late read failure must not recreate a removed registration")
}

func TestMissingGroupInitialOverrideControl(t *testing.T) {
	rdb := startTestRedis(t)
	main := newTestStream(t, rdb, "main-oldest")
	added := newTestStream(t, rdb, "added-newest")
	_, err := added.Add(t.Context(), "old", nil)
	require.NoError(t, err)
	sink := newTestSink(t, main, "recovery", options.WithSinkStartAtOldest())
	require.NoError(t, sink.AddStream(t.Context(), added, options.WithAddStreamStartAtNewest()))
	ch := sink.Subscribe()
	id, err := added.Add(t.Context(), "fresh", nil)
	require.NoError(t, err)
	require.Equal(t, id, receiveEvent(t, ch).ID, "added stream owns its selected boundary")
}

func TestMissingGroupConsumerRotationRecovers(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "rotation-group")
	sink := newTestSink(t, stream, "recovery", options.WithSinkStartAtOldest())
	require.NoError(t, rdb.XGroupDestroy(t.Context(), stream.key, sink.Name).Err())
	sink.lock.Lock()
	sink.lastKeepAlive = time.Now().Add(-3 * sink.ackGracePeriod).UnixNano()
	sink.lock.Unlock()
	require.ErrorContains(t, sink.ensureConsumer(t.Context()), "NOGROUP", "failed rotation remains observable and admits retry backoff")
	require.NoError(t, sink.ensureConsumer(t.Context()), "rotation must reach the same active-registration recovery owner before reads")
	groups, err := rdb.XInfoGroups(t.Context(), stream.key).Result()
	require.NoError(t, err)
	require.Len(t, groups, 1)
}

func TestMissingGroupMembershipResetReconcilesActive(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "membership-reset")
	sink := newTestSink(t, stream, "recovery", options.WithSinkStartAtOldest())
	// Stream.Destroy clears this map after deleting the stream. Recovery can have
	// already recreated the group when that delayed reset becomes observable.
	cm := sink.consumersMap[stream.Name]
	require.NoError(t, cm.replica.Destroy(t.Context()))
	require.Eventually(t, func() bool {
		_, ok := cm.replica.Get(sink.Name)
		return !ok
	}, time.Second, time.Millisecond)
	require.NoError(t, sink.ensureConsumer(t.Context()))
	record, _, err := readSinkMembership(t.Context(), stream, sink.Name)
	require.NoError(t, err)
	require.NotNil(t, record)
	require.Contains(t, record.memberNames(), sink.consumer, "active registration remains responsible after an observed reset")
}

func (g *missingGroupGate) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (g *missingGroupGate) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() == "xreadgroup" && err != nil && strings.HasPrefix(err.Error(), "NOGROUP") && g.count.Add(1) >= g.limit {
			g.once.Do(func() {
				close(g.reached)
			})
			select {
			case <-g.release:
			case <-ctx.Done():
			}
		}
		return err
	}
}

func (g *missingGroupGate) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
