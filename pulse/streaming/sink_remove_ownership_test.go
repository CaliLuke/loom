package streaming

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type (
	removeFaultKey struct{}
	removeFault    struct {
		destroy bool
		after   bool
	}
	removeFaultHook struct {
		key string
	}
)

func TestSinkRemoveStreamClosesMap(t *testing.T) {
	for _, peer := range []bool{false, true} {
		t.Run(fmt.Sprintf("peer=%t", peer), func(t *testing.T) {
			rdb := startTestRedis(t)
			main := newTestStream(t, rdb, "ownership-main")
			stream := newTestStream(t, rdb, "ownership-extra")
			sink := newTestSink(t, main, "sink")
			var expected int64
			if peer {
				other := newTestSink(t, main, "sink")
				require.NoError(t, other.AddStream(t.Context(), stream))
				expected = 1
			}
			for range 3 {
				require.NoError(t, sink.AddStream(t.Context(), stream))
				requireStreamSubscriptions(t, rdb, stream, expected+1)
				require.NoError(t, sink.RemoveStream(t.Context(), stream))
				requireStreamSubscriptions(t, rdb, stream, expected)
				sink.lock.Lock()
				_, retained := sink.consumersMap[stream.Name]
				sink.lock.Unlock()
				require.False(t, retained)
			}
		})
	}
}

func TestSinkRemoveStreamRetryOwnsMap(t *testing.T) {
	for _, destroy := range []bool{false, true} {
		for _, after := range []bool{false, true} {
			for _, readd := range []bool{false, true} {
				t.Run(fmt.Sprintf("destroy=%t/after=%t/readd=%t", destroy, after, readd), func(t *testing.T) {
					rdb := startTestRedis(t)
					main := newTestStream(t, rdb, "retry-main")
					stream := newTestStream(t, rdb, "retry-extra")
					sink := newTestSink(t, main, "sink")
					require.NoError(t, sink.AddStream(t.Context(), stream))
					rdb.AddHook(removeFaultHook{key: consumersMapKey(stream)})
					rdb.AddHook(failGroupCreateHook{})
					ctx := context.WithValue(t.Context(), removeFaultKey{}, removeFault{destroy: destroy, after: after})
					require.ErrorIs(t, sink.RemoveStream(ctx, stream), errInjectedCommand)
					sink.lock.Lock()
					_, retained := sink.consumersMap[stream.Name]
					active := len(sink.streams)
					sink.lock.Unlock()
					require.True(t, retained, "failed cleanup must remain retryable")
					if destroy {
						require.Equal(t, 1, active, "confirmed unregistered stream must stop polling")
					} else {
						require.Equal(t, 2, active, "map failure must retain local stream")
					}
					if readd && destroy {
						require.ErrorIs(t, sink.AddStream(failGroupCreate(t.Context()), stream), errInjectedGroupCreate)
						requireStreamSubscriptions(t, rdb, stream, 1)
						require.NoError(t, sink.AddStream(t.Context(), stream))
						requireStreamSubscriptions(t, rdb, stream, 1)
						require.Equal(t, []string{sinkConsumer(sink)}, streamConsumers(t, rdb, stream))
					}
					require.NoError(t, sink.RemoveStream(t.Context(), stream))
					require.Empty(t, streamConsumers(t, rdb, stream))
					requireGroups(t, rdb, stream, 0)
					requireStreamSubscriptions(t, rdb, stream, 0)
				})
			}
		}
	}
}

func TestSinkCloseOwnsPendingRemoveMap(t *testing.T) {
	rdb := startTestRedis(t)
	main := newTestStream(t, rdb, "close-pending-main")
	stream := newTestStream(t, rdb, "close-pending-extra")
	sink := newTestSink(t, main, "sink")
	require.NoError(t, sink.AddStream(t.Context(), stream))
	rdb.AddHook(removeFaultHook{key: consumersMapKey(stream)})
	ctx := context.WithValue(t.Context(), removeFaultKey{}, removeFault{destroy: true})
	require.ErrorIs(t, sink.RemoveStream(ctx, stream), errInjectedCommand)
	sink.Close(t.Context())
	requireStreamSubscriptions(t, rdb, stream, 0)
}

func requireStreamSubscriptions(t *testing.T, rdb *redis.Client, stream *Stream, want int64) {
	t.Helper()
	channel := mapUpdatesChannel(consumersMapName(stream))
	require.Eventually(t, func() bool {
		counts, err := rdb.PubSubNumSub(t.Context(), channel).Result()
		return err == nil && counts[channel] == want
	}, 2*time.Second, 5*time.Millisecond)
}

func (h removeFaultHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h removeFaultHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		fault, ok := ctx.Value(removeFaultKey{}).(removeFault)
		if !ok || !(failScriptHook{key: h.key}).matches(cmd) {
			return next(ctx, cmd)
		}
		script := fmt.Sprint(cmd.Args()[1])
		destroy := script == destroyGroupScript.Hash() || strings.Contains(script, "HEXISTS")
		if fault.destroy != destroy {
			return next(ctx, cmd)
		}
		if fault.after {
			if err := next(ctx, cmd); err != nil {
				return err
			}
		}
		cmd.SetErr(errInjectedCommand)
		return errInjectedCommand
	}
}

func (h removeFaultHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
