package streaming

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
)

type (
	rotationFaultKey struct{}
	rotationFault    struct {
		streamKey string
		consumer  string
		after     bool
		cleanup   *atomic.Bool
	}
	rotationFaultHook struct{}
)

func TestSinkRotationOwnsEveryStream(t *testing.T) {
	sink, streams := newRotationSink(t)
	old := sink.consumer
	_, err := streams[1].Add(t.Context(), "pending", []byte("payload"))
	require.NoError(t, err)
	_, err = sink.rdb.XReadGroup(t.Context(), &redis.XReadGroupArgs{
		Group: sink.Name, Consumer: old, Streams: []string{streams[1].key, ">"}, Count: 1,
	}).Result()
	require.NoError(t, err)
	sink.lastKeepAlive = 0
	require.NoError(t, sink.ensureConsumer(t.Context()))
	require.NotEqual(t, old, sink.consumer)
	for _, stream := range streams {
		require.Equal(t, []string{sink.consumer}, streamConsumers(t, sink.rdb, stream))
	}
	pending, err := sink.rdb.XPending(t.Context(), streams[1].key, sink.Name).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, pending.Count, "rotation must preserve the old consumer's pending message")
	for _, stream := range streams {
		require.NoError(t, sink.RemoveStream(t.Context(), stream))
		require.Empty(t, streamConsumers(t, sink.rdb, stream))
		requireGroups(t, sink.rdb, stream, 0)
	}
}

func TestSinkRotationFailurePreservesCurrentConsumer(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(fmt.Sprintf("after=%t", after), func(t *testing.T) {
			sink, streams := newRotationSink(t)
			old := sink.consumer
			sink.lastKeepAlive = 0
			sink.rdb.AddHook(rotationFaultHook{})
			ctx := context.WithValue(t.Context(), rotationFaultKey{}, rotationFault{streamKey: streams[1].key, after: after})
			require.ErrorIs(t, sink.ensureConsumer(ctx), errInjectedCommand)
			require.Equal(t, old, sink.consumer)
			require.Zero(t, sink.lastKeepAlive, "failed preparation must not suppress rotation retries")
			for _, stream := range streams {
				require.Equal(t, []string{old}, streamConsumers(t, sink.rdb, stream))
			}
			require.NoError(t, sink.ensureConsumer(t.Context()))
			require.NotEmpty(t, sink.consumer)
			require.NotEqual(t, old, sink.consumer)
			for _, stream := range streams {
				require.Equal(t, []string{sink.consumer}, streamConsumers(t, sink.rdb, stream))
			}
		})
	}
}

func TestSinkRemoveStreamOwnsFailedRotationCleanup(t *testing.T) {
	sink, streams := newRotationSink(t)
	old := sink.consumer
	sink.lastKeepAlive = 0
	sink.rdb.AddHook(rotationFaultHook{})
	ctx := context.WithValue(t.Context(), rotationFaultKey{}, rotationFault{consumer: old})
	require.NoError(t, sink.ensureConsumer(ctx))
	require.NotEqual(t, old, sink.consumer)
	for _, stream := range streams {
		require.ElementsMatch(t, []string{old, sink.consumer}, streamConsumers(t, sink.rdb, stream))
		require.NoError(t, sink.RemoveStream(t.Context(), stream))
		require.Empty(t, streamConsumers(t, sink.rdb, stream))
		requireGroups(t, sink.rdb, stream, 0)
	}
}

func TestSinkRemoveStreamOwnsFailedRotationRollback(t *testing.T) {
	sink, streams := newRotationSink(t)
	old := sink.consumer
	sink.lastKeepAlive = 0
	sink.rdb.AddHook(rotationFaultHook{})
	ctx := context.WithValue(t.Context(), rotationFaultKey{}, rotationFault{
		streamKey: streams[1].key, cleanup: &atomic.Bool{},
	})
	require.ErrorIs(t, sink.ensureConsumer(ctx), errInjectedCommand)
	require.Equal(t, old, sink.consumer)
	require.NotEmpty(t, sink.retiredConsumers)
	for _, stream := range streams {
		require.NoError(t, sink.RemoveStream(t.Context(), stream))
		require.Empty(t, streamConsumers(t, sink.rdb, stream))
		requireGroups(t, sink.rdb, stream, 0)
	}
}

func TestSinkRotationWithNoStreams(t *testing.T) {
	sink, streams := newRotationSink(t)
	for _, stream := range streams {
		require.NoError(t, sink.RemoveStream(t.Context(), stream))
	}
	sink.lastKeepAlive = 0
	require.NotPanics(t, func() {
		require.NoError(t, sink.ensureConsumer(t.Context()))
	})
	_, _, _, armed := sink.armRead(t.Context())
	require.False(t, armed, "an empty sink must not issue XREADGROUP")
	sink.wait.Add(1)
	go sink.read(t.Context())
	require.NoError(t, sink.AddStream(t.Context(), streams[0]))
	events := sink.Subscribe()
	_, err := streams[0].Add(t.Context(), "resumed", []byte("payload"))
	require.NoError(t, err)
	event := receiveEvent(t, events)
	require.NoError(t, sink.Ack(t.Context(), event))
}

// newRotationSink has no background work so each rotation can be driven at
// its exact failure boundary without racing a keep-alive or automatic retry.
func newRotationSink(t *testing.T) (*Sink, []*Stream) {
	t.Helper()
	rdb := startTestRedis(t)
	km, err := rmap.Join(t.Context(), sinkKeepAliveMapName("sink"), rdb)
	require.NoError(t, err)
	sink := &Sink{
		Name: "sink", consumer: "initial", rdb: rdb, logger: pulse.NoopLogger(),
		consumersMap: make(map[string]*rmap.Map), consumersKeepAliveMap: km,
		ackGracePeriod: time.Second, blockDuration: 20 * time.Millisecond,
		donechan: make(chan struct{}), startID: "0-0", bufferSize: 1,
	}
	t.Cleanup(func() {
		sink.Close(context.Background())
	})
	streams := []*Stream{newTestStream(t, rdb, "rotation-first"), newTestStream(t, rdb, "rotation-second")}
	for _, stream := range streams {
		require.NoError(t, sink.AddStream(t.Context(), stream))
		require.NoError(t, sink.createConsumer(t.Context(), stream, sink.consumer))
	}
	return sink, streams
}

func (rotationFaultHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (rotationFaultHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		fault, ok := ctx.Value(rotationFaultKey{}).(rotationFault)
		if !ok {
			return next(ctx, cmd)
		}
		if fault.cleanup != nil && fault.cleanup.Load() && (cmd.Name() == "eval" || cmd.Name() == "evalsha") {
			cmd.SetErr(errInjectedCommand)
			return errInjectedCommand
		}
		match := fault.streamKey != "" && (commandArgs{"xgroup", "createconsumer", fault.streamKey}).matches(cmd)
		if fault.consumer != "" && (cmd.Name() == "eval" || cmd.Name() == "evalsha") {
			match = slices.ContainsFunc(cmd.Args()[3:], func(arg any) bool {
				return fmt.Sprint(arg) == fault.consumer
			})
		}
		if !match {
			return next(ctx, cmd)
		}
		if fault.after {
			if err := next(ctx, cmd); err != nil {
				return err
			}
		}
		if fault.cleanup != nil {
			fault.cleanup.Store(true)
		}
		cmd.SetErr(errInjectedCommand)
		return errInjectedCommand
	}
}

func (rotationFaultHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
