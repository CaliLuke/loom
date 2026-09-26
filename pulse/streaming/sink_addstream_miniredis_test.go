package streaming

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/rmap"
)

type (
	// failGroupCreateHook fails the XGROUP CREATE commands sent with a
	// context that failGroupCreate or cancelGroupCreate marks, so a test can
	// fail the group creation of one sink instance and not of another one
	// that shares the client.
	failGroupCreateHook struct{}

	// failGroupCreateKey is the context key of the failure: a
	// context.CancelFunc that cancels the context of the command, or nil to
	// fail it with errInjectedGroupCreate.
	failGroupCreateKey struct{}
)

// errInjectedGroupCreate is the error failGroupCreateHook returns.
var errInjectedGroupCreate = errors.New("injected group creation failure")

// TestSinkAddStreamRollsBackFailedGroupCreate checks that Sink.AddStream
// removes the consumer it added to the consumers map of the stream when the
// consumer group creation fails, so the map holds no consumer for a stream
// that was not added (issue #481), and that a later AddStream adds the
// consumer once.
func TestSinkAddStreamRollsBackFailedGroupCreate(t *testing.T) {
	cases := []struct {
		name string
		// fail makes the group creation of stream fail and returns the
		// context AddStream uses and a function that removes the failure.
		fail func(t *testing.T, rdb *redis.Client, stream *Stream) (context.Context, func())
		want string
	}{
		{
			name: "group creation error",
			fail: func(t *testing.T, _ *redis.Client, _ *Stream) (context.Context, func()) {
				t.Helper()
				return failGroupCreate(t.Context()), func() {}
			},
			want: errInjectedGroupCreate.Error(),
		},
		{
			name: "context canceled during group creation",
			fail: func(t *testing.T, _ *redis.Client, _ *Stream) (context.Context, func()) {
				t.Helper()
				return cancelGroupCreate(t.Context()), func() {}
			},
			want: context.Canceled.Error(),
		},
		{
			name: "stream key of another type",
			fail: func(t *testing.T, rdb *redis.Client, stream *Stream) (context.Context, func()) {
				t.Helper()
				require.NoError(t, rdb.Set(t.Context(), stream.key, "not a stream", 0).Err())
				return t.Context(), func() {
					require.NoError(t, rdb.Del(t.Context(), stream.key).Err())
				}
			},
			want: "WRONGTYPE",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			rdb.AddHook(failGroupCreateHook{})
			main := newTestStream(t, rdb, "addstream-rollback-main")
			stream := newTestStream(t, rdb, "addstream-rollback")
			sink := newTestSink(t, main, "sink")

			ctx, restore := tc.fail(t, rdb, stream)
			err := sink.AddStream(ctx, stream)
			require.ErrorContains(t, err, "failed to create Redis consumer group sink")
			require.ErrorContains(t, err, tc.want)
			sink.lock.Lock()
			assert.Len(t, sink.streams, 1)
			assert.NotContains(t, sink.consumersMap, stream.Name)
			sink.lock.Unlock()
			assert.Empty(t, streamConsumers(t, rdb, stream))

			restore()
			require.NoError(t, sink.AddStream(t.Context(), stream))
			sink.lock.Lock()
			assert.Len(t, sink.streams, 2)
			sink.lock.Unlock()
			assert.Equal(t, []string{sinkConsumer(sink)}, streamConsumers(t, rdb, stream))
		})
	}
}

// TestSinkAddStreamConcurrentFailedGroupCreate checks that a failed
// AddStream of one sink instance removes only its own consumer from the
// consumers map while another instance adds the same stream, so the other
// instance's RemoveStream still destroys the group once it is the last
// consumer.
func TestSinkAddStreamConcurrentFailedGroupCreate(t *testing.T) {
	rdb := startTestRedis(t)
	rdb.AddHook(failGroupCreateHook{})
	ctx := t.Context()
	main := newTestStream(t, rdb, "addstream-concurrent-main")
	stream := newTestStream(t, rdb, "addstream-concurrent")
	ok := newTestSink(t, main, "sink")
	failing := newTestSink(t, main, "sink")

	var wg sync.WaitGroup
	var okErr, failErr error
	wg.Go(func() {
		okErr = ok.AddStream(ctx, stream)
	})
	wg.Go(func() {
		failErr = failing.AddStream(failGroupCreate(ctx), stream)
	})
	wg.Wait()
	require.NoError(t, okErr)
	require.ErrorIs(t, failErr, errInjectedGroupCreate)
	assert.Equal(t, []string{sinkConsumer(ok)}, streamConsumers(t, rdb, stream))
	requireGroups(t, rdb, stream, 1)

	require.NoError(t, ok.RemoveStream(ctx, stream))
	assert.Empty(t, streamConsumers(t, rdb, stream))
	requireGroups(t, rdb, stream, 0)
}

// DialHook implements redis.Hook.
func (failGroupCreateHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook implements redis.Hook.
func (failGroupCreateHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		failure, marked := ctx.Value(failGroupCreateKey{}).(context.CancelFunc)
		if marked && isGroupCreate(cmd) {
			err := errInjectedGroupCreate
			if failure != nil {
				failure()
				err = ctx.Err()
			}
			cmd.SetErr(err)
			return err
		}
		return next(ctx, cmd)
	}
}

// ProcessPipelineHook implements redis.Hook.
func (failGroupCreateHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

// failGroupCreate returns ctx marked so failGroupCreateHook fails the group
// creations sent with it.
func failGroupCreate(ctx context.Context) context.Context {
	return context.WithValue(ctx, failGroupCreateKey{}, context.CancelFunc(nil))
}

// cancelGroupCreate returns a child of ctx that failGroupCreateHook cancels
// when a group creation is sent with it, failing that creation with the
// cancellation error.
func cancelGroupCreate(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	return context.WithValue(ctx, failGroupCreateKey{}, cancel)
}

// isGroupCreate reports whether cmd is an XGROUP CREATE command.
func isGroupCreate(cmd redis.Cmder) bool {
	args := cmd.Args()
	return len(args) > 1 &&
		strings.EqualFold(fmt.Sprint(args[0]), "xgroup") &&
		strings.EqualFold(fmt.Sprint(args[1]), "create")
}

// sinkConsumer returns the current consumer of sink.
func sinkConsumer(sink *Sink) string {
	sink.lock.Lock()
	defer sink.lock.Unlock()
	return sink.consumer
}

// streamConsumers returns the consumers that the consumers map of stream
// holds for the sink named "sink", as a new replica reads them from Redis.
func streamConsumers(t *testing.T, rdb *redis.Client, stream *Stream) []string {
	t.Helper()
	m, err := rmap.Join(t.Context(), consumersMapName(stream), rdb)
	require.NoError(t, err)
	defer m.Close()
	consumers, _ := m.GetValues("sink")
	return consumers
}

// requireGroups asserts the number of consumer groups of stream.
func requireGroups(t *testing.T, rdb *redis.Client, stream *Stream, want int) {
	t.Helper()
	groups, err := rdb.XInfoGroups(t.Context(), stream.key).Result()
	require.NoError(t, err)
	require.Len(t, groups, want)
}
