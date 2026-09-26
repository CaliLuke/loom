package streaming

import (
	"context"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type (
	// interleaveHook runs the function that interleave marks a context with
	// once, right after the first command sent with that context succeeds.
	// A test uses it to run another sink instance's call between two Redis
	// steps of one call.
	interleaveHook struct{}

	// interleaveKey is the context key of the *interleaveStep to run.
	interleaveKey struct{}

	// interleaveStep is the function interleaveHook runs, and whether it ran.
	interleaveStep struct {
		once sync.Once
		run  func()
	}
)

// TestSinkRemoveStreamKeepsGroupOfConcurrentAddStream checks that
// Sink.RemoveStream does not destroy the consumer group when another instance
// of the sink adds the stream after RemoveStream removed the last consumer
// from the consumers map, so the other instance keeps reading the stream
// (issue #508, tla/cfg/fixed_remove_asis.cfg).
func TestSinkRemoveStreamKeepsGroupOfConcurrentAddStream(t *testing.T) {
	rdb := startTestRedis(t)
	rdb.AddHook(interleaveHook{})
	ctx := t.Context()
	main := newTestStream(t, rdb, "removestream-race-main")
	stream := newTestStream(t, rdb, "removestream-race")
	remover := newTestSink(t, main, "sink")
	adder := newTestSink(t, main, "sink")
	require.NoError(t, remover.AddStream(ctx, stream))

	var addErr error
	removeCtx := interleave(ctx, func() {
		addErr = adder.AddStream(ctx, stream)
	})
	require.NoError(t, remover.RemoveStream(removeCtx, stream))
	require.NoError(t, addErr)
	assert.Equal(t, []string{sinkConsumer(adder)}, streamConsumers(t, rdb, stream))
	requireGroups(t, rdb, stream, 1)

	events := adder.Subscribe()
	_, err := stream.Add(ctx, "event", []byte("payload"))
	require.NoError(t, err)
	ev := receiveEvent(t, events)
	assert.Equal(t, stream.Name, ev.StreamName)
	require.NoError(t, adder.Ack(ctx, ev))

	require.NoError(t, adder.RemoveStream(ctx, stream))
	assert.Empty(t, streamConsumers(t, rdb, stream))
	requireGroups(t, rdb, stream, 0)
}

// TestSinkRemoveStreamDestroysGroupOfLastConsumer checks that
// Sink.RemoveStream destroys the consumer group only when the consumers map
// holds no consumer of the sink.
func TestSinkRemoveStreamDestroysGroupOfLastConsumer(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	main := newTestStream(t, rdb, "removestream-last-main")
	stream := newTestStream(t, rdb, "removestream-last")
	first := newTestSink(t, main, "sink")
	second := newTestSink(t, main, "sink")
	require.NoError(t, first.AddStream(ctx, stream))
	require.NoError(t, second.AddStream(ctx, stream))

	require.NoError(t, first.RemoveStream(ctx, stream))
	assert.Equal(t, []string{sinkConsumer(second)}, streamConsumers(t, rdb, stream))
	requireGroups(t, rdb, stream, 1)
	require.NoError(t, first.RemoveStream(ctx, stream), "RemoveStream is idempotent")

	require.NoError(t, second.RemoveStream(ctx, stream))
	assert.Empty(t, streamConsumers(t, rdb, stream))
	requireGroups(t, rdb, stream, 0)
}

// DialHook implements redis.Hook.
func (interleaveHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook implements redis.Hook.
func (interleaveHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if step, ok := ctx.Value(interleaveKey{}).(*interleaveStep); ok && err == nil {
			step.once.Do(step.run)
		}
		return err
	}
}

// ProcessPipelineHook implements redis.Hook.
func (interleaveHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

// interleave returns ctx marked so interleaveHook runs fn once, right after
// the first command sent with the returned context succeeds.
func interleave(ctx context.Context, fn func()) context.Context {
	return context.WithValue(ctx, interleaveKey{}, &interleaveStep{run: fn})
}
