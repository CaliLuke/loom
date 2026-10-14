package streaming

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type (
	recoveryFaultKey struct {
	}
	recoveryFault struct {
		command string
		after   bool
	}
	recoveryHook struct {
		streamKey                string
		group                    string
		memberKey                string
		armed                    chan struct{}
		release                  chan struct{}
		creates                  atomic.Int32
		publicationSeen          atomic.Bool
		createdWithoutMembership atomic.Bool
	}
)

var errRecoveryFault = errors.New("injected recovery reply failure")

func TestMissingGroupRetainsStartAndPeerCursor(t *testing.T) {
	for _, selector := range []string{"oldest", "newest", "explicit", "timestamp"} {
		t.Run(selector, func(t *testing.T) {
			rdb := startTestRedis(t)
			main := newTestStream(t, rdb, "start-main")
			stream := newTestStream(t, rdb, "start-added")
			for _, id := range []string{"1000-0", "2000-0"} {
				require.NoError(t, rdb.XAdd(t.Context(), &redis.XAddArgs{Stream: stream.key, ID: id, Values: map[string]any{"n": id}}).Err())
			}
			sink := newTestSink(t, main, "recovery", options.WithSinkStartAtOldest())
			opt, want := recoveryStartOption(selector)
			require.NoError(t, sink.AddStream(t.Context(), stream, opt))
			require.NoError(t, rdb.XGroupDestroy(t.Context(), stream.key, sink.Name).Err())
			require.NoError(t, reconcileSink(t, sink, t.Context()))
			requireGroupCursor(t, rdb, stream, sink.Name, want)
			// A peer's progressed cursor must survive later reconciliation.
			require.NoError(t, rdb.XAdd(t.Context(), &redis.XAddArgs{Stream: stream.key, ID: "3000-0", Values: map[string]any{"n": "peer"}}).Err())
			_, err := rdb.XReadGroup(t.Context(), &redis.XReadGroupArgs{Group: sink.Name, Consumer: "peer", Streams: []string{stream.key, ">"}, Block: -1}).Result()
			require.NoError(t, err)
			require.NoError(t, reconcileSink(t, sink, t.Context()))
			requireGroupCursor(t, rdb, stream, sink.Name, "3000-0")
			// Re-add replaces the old registration's authored selector.
			require.NoError(t, sink.RemoveStream(t.Context(), stream))
			require.NoError(t, sink.AddStream(t.Context(), stream, options.WithAddStreamStartAfter("1500-0")))
			require.NoError(t, rdb.XGroupDestroy(t.Context(), stream.key, sink.Name).Err())
			require.NoError(t, reconcileSink(t, sink, t.Context()))
			requireGroupCursor(t, rdb, stream, sink.Name, "1500-0")
		})
	}
}

func TestMissingGroupPublishesMembershipBeforeGroup(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "publication-group")
	hook := &recoveryHook{streamKey: stream.key, group: "recovery", memberKey: consumersMapKey(stream)}
	rdb.AddHook(hook)
	sink := newTestSink(t, stream, "recovery", options.WithSinkStartAtOldest())
	require.NoError(t, stream.Destroy(t.Context()))
	ctx := context.WithValue(t.Context(), recoveryFaultKey{}, recoveryFault{command: "order"})
	require.NoError(t, reconcileSink(t, sink, ctx))
	require.True(t, hook.publicationSeen.Load())
	require.False(t, hook.createdWithoutMembership.Load())
}

func TestMissingGroupRetryRetainsObligation(t *testing.T) {
	for _, command := range []string{"membership", "create", "createconsumer", "keepalive"} {
		for _, after := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/after=%t", command, after), func(t *testing.T) {
				rdb := startTestRedis(t)
				stream := newTestStream(t, rdb, "retry-group")
				hook := &recoveryHook{streamKey: stream.key, group: "recovery", memberKey: consumersMapKey(stream)}
				rdb.AddHook(hook)
				sink := newTestSink(t, stream, "recovery", options.WithSinkStartAtOldest())
				require.NoError(t, rdb.XGroupDestroy(t.Context(), stream.key, sink.Name).Err())
				ctx := context.WithValue(t.Context(), recoveryFaultKey{}, recoveryFault{command: command, after: after})
				require.ErrorIs(t, reconcileSink(t, sink, ctx), errRecoveryFault)
				sink.lock.Lock()
				pending := sink.streams[0].recoveryPending
				sink.lock.Unlock()
				require.True(t, pending)
				require.NoError(t, sink.ensureConsumer(t.Context()))
				sink.lock.Lock()
				pending = sink.streams[0].recoveryPending
				sink.lock.Unlock()
				require.False(t, pending)
				requireGroupCursor(t, rdb, stream, sink.Name, "0-0")
			})
		}
	}
}

func TestMissingGroupKeepAliveFailurePreservesPending(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "pending-group")
	hook := &recoveryHook{streamKey: stream.key, group: "recovery", memberKey: consumersMapKey(stream)}
	rdb.AddHook(hook)
	sink := newTestSink(t, stream, "recovery", options.WithSinkStartAtOldest())
	_, err := stream.Add(t.Context(), "pending", nil)
	require.NoError(t, err)
	_, err = rdb.XReadGroup(t.Context(), &redis.XReadGroupArgs{Group: sink.Name, Consumer: sink.consumer, Streams: []string{stream.key, ">"}, Count: 1}).Result()
	require.NoError(t, err)
	ctx := context.WithValue(t.Context(), recoveryFaultKey{}, recoveryFault{command: "keepalive"})
	require.ErrorIs(t, reconcileSink(t, sink, ctx), errRecoveryFault)
	pending, err := rdb.XPending(t.Context(), stream.key, sink.Name).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, pending.Count)
	require.NoError(t, sink.ensureConsumer(t.Context()))
}

func TestMissingGroupRecoveryOrdersRemovalAndClose(t *testing.T) {
	for _, closeSink := range []bool{false, true} {
		t.Run(fmt.Sprintf("close=%t", closeSink), func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "ordered-group")
			hook := &recoveryHook{streamKey: stream.key, group: "recovery", memberKey: consumersMapKey(stream), armed: make(chan struct{}), release: make(chan struct{})}
			rdb.AddHook(hook)
			sink := newTestSink(t, stream, "recovery", options.WithSinkStartAtOldest())
			require.NoError(t, rdb.XGroupDestroy(t.Context(), stream.key, sink.Name).Err())
			ctx := context.WithValue(t.Context(), recoveryFaultKey{}, recoveryFault{command: "hold"})
			recovered := make(chan error, 1)
			go func() {
				recovered <- reconcileSink(t, sink, ctx)
			}()
			select {
			case <-hook.armed:
			case <-time.After(time.Second):
				t.Fatal("recovery did not reach group publication")
			}
			removed := make(chan error, 1)
			go func() {
				if closeSink {
					sink.Close(context.Background())
					removed <- nil
					return
				}
				removed <- sink.RemoveStream(t.Context(), stream)
			}()
			close(hook.release)
			require.NoError(t, awaitRecoveryResult(t, recovered))
			require.NoError(t, awaitRecoveryResult(t, removed))
			before := hook.creates.Load()
			require.NoError(t, reconcileSink(t, sink, t.Context()))
			require.Equal(t, before, hook.creates.Load(), "terminal local state cannot publish a late repair")
			if !closeSink {
				groups, err := rdb.XInfoGroups(t.Context(), stream.key).Result()
				require.NoError(t, err)
				require.Empty(t, groups)
			}
		})
	}
}

func TestMissingGroupRetryWaitIsPositiveAndInterruptible(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "backoff-group")
	sink := newTestSink(t, stream, "recovery", options.WithSinkBlockDuration(time.Nanosecond))
	start := time.Now()
	require.True(t, sink.waitBeforeEnsureConsumerRetry())
	require.GreaterOrEqual(t, time.Since(start), ensureConsumerRetryDelay)
	sink.Close(t.Context())
	require.False(t, sink.waitBeforeEnsureConsumerRetry())
}

func TestMissingGroupKeepAliveReconcilesResetWithoutReads(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "keepalive-group")
	sink := newTestSink(t, stream, "recovery", options.WithSinkAckGracePeriod(30*time.Millisecond))
	cm := sink.consumersMap[stream.Name]
	require.NoError(t, cm.replica.Destroy(t.Context()))
	require.Eventually(t, func() bool {
		raw, ok := cm.replica.Get(sink.Name)
		if !ok {
			return false
		}
		record, err := decodeSinkMembership(raw)
		return err == nil && len(record.Members) == 1 && record.Members[0].Name == sinkConsumer(sink)
	}, time.Second, time.Millisecond)
	sink.lock.Lock()
	started := sink.deliveryStarted
	sink.lock.Unlock()
	require.False(t, started, "registration repair must not activate delivery")
}

func awaitRecoveryResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("lifecycle operation did not complete")
		return nil
	}
}

func reconcileSink(t *testing.T, sink *Sink, ctx context.Context) error {
	t.Helper()
	sink.lock.Lock()
	defer sink.lock.Unlock()
	return sink.reconcileRegistrations(ctx, true)
}

func recoveryStartOption(selector string) (options.AddStream, string) {
	switch selector {
	case "oldest":
		return options.WithAddStreamStartAtOldest(), "0-0"
	case "newest":
		return options.WithAddStreamStartAtNewest(), "2000-0"
	case "explicit":
		return options.WithAddStreamStartAfter("1000-0"), "1000-0"
	default:
		return options.WithAddStreamStartAt(time.UnixMilli(1500)), "1500-0"
	}
}

func requireGroupCursor(t *testing.T, rdb *redis.Client, stream *Stream, group, want string) {
	t.Helper()
	groups, err := rdb.XInfoGroups(t.Context(), stream.key).Result()
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Equal(t, group, groups[0].Name)
	actual := groups[0].LastDeliveredID
	// Miniredis preserves the short zero supplied to XGROUP; native Redis
	// reports the equivalent full ID. Compare the same Redis position.
	if !strings.Contains(actual, "-") {
		actual += "-0"
	}
	require.Equal(t, want, actual)
}

func (h *recoveryHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *recoveryHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		args := cmd.Args()
		op := ""
		if cmd.Name() == "xgroup" && len(args) > 2 && fmt.Sprint(args[2]) == h.streamKey {
			op = strings.ToLower(fmt.Sprint(args[1]))
			if op == "create" {
				h.creates.Add(1)
			}
		}
		if (cmd.Name() == "evalsha" || cmd.Name() == "eval") && len(args) > 3 {
			key := fmt.Sprint(args[3])
			if key == h.memberKey {
				op = "membership"
			} else if strings.Contains(key, "sink:"+h.group+":keepalive") {
				op = "keepalive"
			}
		}
		fault, ok := ctx.Value(recoveryFaultKey{}).(recoveryFault)
		if ok && fault.command == "order" {
			if op == "membership" {
				h.publicationSeen.Store(true)
			}
			if op == "create" && !h.publicationSeen.Load() {
				h.createdWithoutMembership.Store(true)
			}
		}
		if ok && fault.command == "hold" && op == "create" {
			close(h.armed)
			<-h.release
		}
		if !ok || op != fault.command {
			return next(ctx, cmd)
		}
		if fault.after {
			if err := next(ctx, cmd); err != nil {
				return err
			}
		}
		return errRecoveryFault
	}
}

func (h *recoveryHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
