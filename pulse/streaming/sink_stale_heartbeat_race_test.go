package streaming

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type staleHeartbeatGate struct {
	streamKey string
	after     bool
	entered   chan struct{}
	release   chan struct{}
	armed     atomic.Bool
	openOnce  sync.Once
}

func (g *staleHeartbeatGate) open() {
	g.openOnce.Do(func() {
		close(g.release)
	})
}

func TestSinkStaleSnapshotCannotRetireRefreshedMember(t *testing.T) {
	for _, tc := range []struct {
		name  string
		after bool
	}{
		{name: "refresh before native deletion"},
		{name: "refresh after native deletion", after: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertStaleHeartbeatWins(t, tc.after)
		})
	}
}

func TestSinkStaleDeletePreservesPendingReadBeforeMembershipRetirement(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "stale-pending-race")
	a := newTestSink(t, stream, "shared", options.WithSinkBlockDuration(time.Hour), options.WithSinkAckGracePeriod(time.Hour))
	b := newTestSink(t, stream, "shared", options.WithSinkAckGracePeriod(time.Hour))
	staleAt := time.Now().Add(-3 * time.Hour).UnixNano()
	setSinkMemberKeepAlive(t, a.consumersMap[stream.Name], a.Name, a.filterIdentity, a.consumer, staleAt)

	readStarted := make(chan struct{})
	rdb.AddHook(&readStartedHook{entered: readStarted})
	got := a.Subscribe()
	select {
	case <-readStarted:
	case <-time.After(time.Second):
		t.Fatal("sink did not start its blocked read")
	}

	gate := &staleHeartbeatGate{
		streamKey: stream.key,
		after:     true,
		entered:   make(chan struct{}),
		release:   make(chan struct{}),
	}
	defer gate.open()
	rdb.AddHook(gate)
	cleanupDone := make(chan error, 1)
	go func() {
		b.lock.Lock()
		defer b.lock.Unlock()
		cleanupDone <- b.deleteStreamStaleConsumers(t.Context(), stream)
	}()
	select {
	case <-gate.entered:
	case <-time.After(time.Second):
		t.Fatal("stale cleanup did not delete the native consumer")
	}
	native, err := rdb.XInfoConsumers(t.Context(), stream.key, a.Name).Result()
	require.NoError(t, err)
	for _, consumer := range native {
		require.NotEqual(t, a.consumer, consumer.Name, "native deletion must precede the pending delivery")
	}

	_, err = stream.Add(t.Context(), "pending", []byte("payload"))
	require.NoError(t, err)
	event := receiveEvent(t, got)
	require.Equal(t, "pending", event.EventName)
	pending, err := rdb.XPending(t.Context(), stream.key, a.Name).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, pending.Count, "delivery after native deletion must still be pending before membership CAS")

	gate.open()
	require.NoError(t, <-cleanupDone)
	require.NoError(t, b.RemoveStream(t.Context(), stream))
	pending, err = rdb.XPending(t.Context(), stream.key, a.Name).Result()
	require.NoError(t, err, "group destruction must not drop a PEL entry created after native deletion")
	require.EqualValues(t, 1, pending.Count)
	claimed, err := rdb.XClaim(t.Context(), &redis.XClaimArgs{
		Stream:   stream.key,
		Group:    a.Name,
		Consumer: "later-claimant",
		Messages: []string{event.ID},
	}).Result()
	require.NoError(t, err)
	require.Len(t, claimed, 1, "a later consumer can claim the retained pending entry")
	require.NoError(t, a.Ack(t.Context(), event), "the retained pending entry remains claimable and acknowledgeable")
	pending, err = rdb.XPending(t.Context(), stream.key, a.Name).Result()
	require.NoError(t, err)
	require.Zero(t, pending.Count)
}

func TestSinkStaleCleanupOfNativeOnlyConsumerKeepsActiveReader(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "native-only-lost-removal-reply")
	a := newTestSink(t, stream, "shared", options.WithSinkStartAtOldest(), options.WithSinkAckGracePeriod(time.Hour))
	b := newTestSink(t, stream, "shared", options.WithSinkStartAtOldest(), options.WithSinkAckGracePeriod(time.Hour))
	readGate := &xreadCompletionGate{
		count:   2,
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	defer readGate.open()
	rdb.AddHook(readGate)
	got := a.Subscribe()

	firstID, err := stream.Add(t.Context(), "before-removal", []byte("payload"))
	require.NoError(t, err)
	first := receiveEvent(t, got)
	require.Equal(t, firstID, first.ID)
	require.NoError(t, a.Ack(t.Context(), first))
	select {
	case <-readGate.entered:
	case <-time.After(time.Second):
		t.Fatal("the second XREADGROUP did not complete before membership removal")
	}

	memberKey := consumersMapKey(stream)
	before, _, err := readSinkMembership(t.Context(), stream, a.Name)
	require.NoError(t, err)
	require.Contains(t, before.memberNames(), a.consumer)
	removeReply := &membershipReplyHook{key: memberKey}
	rdb.AddHook(removeReply)
	require.ErrorIs(t, a.RemoveStream(t.Context(), stream), errInjectedCommand)
	require.True(t, removeReply.failed.Load(), "the membership removal committed before its reply was lost")
	after, _, err := readSinkMembership(t.Context(), stream, a.Name)
	require.NoError(t, err)
	require.NotContains(t, after.memberNames(), a.consumer)

	activeNative, err := rdb.XInfoConsumers(t.Context(), stream.key, a.Name).Result()
	require.NoError(t, err)
	require.Contains(t, consumerNames(activeNative), a.consumer)
	b.lock.Lock()
	err = b.deleteStreamStaleConsumers(t.Context(), stream)
	b.lock.Unlock()
	require.NoError(t, err)
	afterCleanup, err := rdb.XInfoConsumers(t.Context(), stream.key, a.Name).Result()
	require.NoError(t, err)
	require.NotContains(t, consumerNames(afterCleanup), a.consumer,
		"a native-only consumer has no current membership freshness authority")

	readGate.open()
	secondID, err := stream.Add(t.Context(), "after-native-cleanup", []byte("payload"))
	require.NoError(t, err)
	second := receiveEvent(t, got)
	require.Equal(t, secondID, second.ID, "deleting an empty native consumer must not interrupt the live group read")
	require.NoError(t, a.Ack(t.Context(), second))
	groups, err := rdb.XInfoGroups(t.Context(), stream.key).Result()
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, secondID, groups[0].LastDeliveredID,
		"native consumer cleanup must not reset the shared group cursor")
}

type xreadCompletionGate struct {
	count   int32
	seen    atomic.Int32
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (g *xreadCompletionGate) open() {
	g.once.Do(func() {
		close(g.release)
	})
}

func (g *xreadCompletionGate) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (g *xreadCompletionGate) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() == "xreadgroup" && g.seen.Add(1) == g.count {
			close(g.entered)
			<-g.release
		}
		return err
	}
}

func (g *xreadCompletionGate) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func consumerNames(consumers []redis.XInfoConsumer) []string {
	names := make([]string, len(consumers))
	for i, consumer := range consumers {
		names[i] = consumer.Name
	}
	return names
}

func assertStaleHeartbeatWins(t *testing.T, afterNativeDelete bool) {
	t.Helper()
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "stale-heartbeat-race")
	a := newTestSink(t, stream, "shared", options.WithSinkAckGracePeriod(time.Hour))
	b := newTestSink(t, stream, "shared", options.WithSinkAckGracePeriod(time.Hour))
	staleAt := time.Now().Add(-3 * time.Hour).UnixNano()
	setSinkMemberKeepAlive(t, a.consumersMap[stream.Name], a.Name, a.filterIdentity, a.consumer, staleAt)

	gate := &staleHeartbeatGate{
		streamKey: stream.key,
		after:     afterNativeDelete,
		entered:   make(chan struct{}),
		release:   make(chan struct{}),
	}
	defer gate.open()
	rdb.AddHook(gate)
	cleanupDone := make(chan error, 1)
	go func() {
		b.lock.Lock()
		defer b.lock.Unlock()
		cleanupDone <- b.deleteStreamStaleConsumers(t.Context(), stream)
	}()
	select {
	case <-gate.entered:
	case <-time.After(time.Second):
		t.Fatal("stale cleanup did not reach native consumer deletion")
	}
	native, err := rdb.XInfoConsumers(t.Context(), stream.key, a.Name).Result()
	require.NoError(t, err)
	nativeNames := make([]string, len(native))
	for i, consumer := range native {
		nativeNames[i] = consumer.Name
	}
	if afterNativeDelete {
		require.NotContains(t, nativeNames, a.consumer, "native consumer must be deleted before refresh")
	} else {
		require.Contains(t, nativeNames, a.consumer, "native consumer is still present before refresh")
	}

	a.lock.Lock()
	err = a.updateConsumerKeepAlive(t.Context(), a.consumer)
	a.lock.Unlock()
	require.NoError(t, err)
	gate.open()
	require.NoError(t, <-cleanupDone)

	record, _, err := readSinkMembership(t.Context(), stream, a.Name)
	require.NoError(t, err)
	require.Contains(t, record.memberNames(), a.consumer, "a confirmed refresh must win against the stale snapshot")
	require.NoError(t, b.RemoveStream(t.Context(), stream))

	group, err := rdb.XInfoGroups(t.Context(), stream.key).Result()
	require.NoError(t, err)
	require.Len(t, group, 1, "the refreshed registration still owns the shared group")

	got := a.Subscribe()
	_, err = stream.Add(t.Context(), "event", []byte("payload"))
	require.NoError(t, err)
	require.Equal(t, "event", receiveEvent(t, got).EventName)
}

func setSinkMemberKeepAlive(t *testing.T, cm *sinkConsumerMap, group string, filter sinkFilter, consumer string, at int64) {
	t.Helper()
	_, err := changeSinkMembership(t.Context(), cm, group, filter, sinkMemberChange{
		add:         consumer,
		remove:      []string{consumer},
		restore:     true,
		keepAliveAt: at,
	})
	require.NoError(t, err)
}

func (g *staleHeartbeatGate) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (g *staleHeartbeatGate) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		args := cmd.Args()
		matches := (cmd.Name() == "eval" || cmd.Name() == "evalsha") && len(args) > 3 && fmt.Sprint(args[3]) == g.streamKey
		if matches && !g.after && g.armed.CompareAndSwap(false, true) {
			close(g.entered)
			<-g.release
		}
		err := next(ctx, cmd)
		if matches && g.after && err == nil && g.armed.CompareAndSwap(false, true) {
			close(g.entered)
			<-g.release
		}
		return err
	}
}

func (g *staleHeartbeatGate) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

type readStartedHook struct {
	entered chan struct{}
	once    sync.Once
}

func (h *readStartedHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *readStartedHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "xreadgroup" {
			h.once.Do(func() {
				close(h.entered)
			})
		}
		return next(ctx, cmd)
	}
}

func (h *readStartedHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
