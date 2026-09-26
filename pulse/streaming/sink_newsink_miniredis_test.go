package streaming

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/rmap"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type (
	// failCommandHook fails every command whose leading arguments match args
	// with errInjectedCommand.
	failCommandHook struct {
		args commandArgs
	}

	// failScriptHook fails every script call (EVAL or EVALSHA) that names
	// key with errInjectedCommand.
	failScriptHook struct {
		key string
	}

	// commandArgs are the leading arguments of a command, compared ignoring
	// case. Empty commandArgs match every command.
	commandArgs []string
)

// errInjectedCommand is the error failCommandHook and failScriptHook return.
var errInjectedCommand = errors.New("injected command failure")

// TestNewSinkClosesJoinedMapsOnError checks that NewSink closes the
// replicated maps it joined when a later step fails, so a failed NewSink
// leaves no subscription to their update channels (issue #509), and that it
// leaves no consumer in the consumers map of the stream or in the consumer
// group (issue #531).
func TestNewSinkClosesJoinedMapsOnError(t *testing.T) {
	cases := []struct {
		name string
		// sink is the name of the sink to create.
		sink string
		// fail makes a step of NewSink fail.
		fail func(t *testing.T, rdb *redis.Client, stream *Stream)
		// group is true if the consumer group exists after the failure.
		group bool
		want  string
	}{
		{
			name: "keep-alive map join",
			sink: "invalid sink name",
			fail: func(*testing.T, *redis.Client, *Stream) {},
			want: "failed to join replicated map for sink keep-alives",
		},
		{
			name: "group creation",
			sink: "sink",
			fail: func(t *testing.T, rdb *redis.Client, stream *Stream) {
				t.Helper()
				require.NoError(t, rdb.Set(t.Context(), stream.key, "not a stream", 0).Err())
			},
			want: "failed to create Redis consumer group sink",
		},
		{
			name: "consumer creation",
			sink: "sink",
			fail: func(_ *testing.T, rdb *redis.Client, _ *Stream) {
				rdb.AddHook(failCommandHook{args: commandArgs{"xgroup", "createconsumer"}})
			},
			group: true,
			want:  "failed to create consumer",
		},
		{
			name: "keep-alive update",
			sink: "sink",
			fail: func(_ *testing.T, rdb *redis.Client, _ *Stream) {
				rdb.AddHook(failScriptHook{key: fmt.Sprintf("map:%s:content", sinkKeepAliveMapName("sink"))})
			},
			group: true,
			want:  "failed to set sink keep-alive",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "newsink-leak")
			tc.fail(t, rdb, stream)

			sink, err := stream.NewSink(t.Context(), tc.sink)
			require.ErrorContains(t, err, tc.want)
			assert.Nil(t, sink)
			channels := []string{
				mapUpdatesChannel(consumersMapName(stream)),
				mapUpdatesChannel(sinkKeepAliveMapName(tc.sink)),
			}
			assert.Eventually(t, func() bool {
				subs, err := rdb.PubSubNumSub(t.Context(), channels...).Result()
				if err != nil {
					t.Errorf("PUBSUB NUMSUB: %v", err)
					return true
				}
				for _, n := range subs {
					if n != 0 {
						return false
					}
				}
				return true
			}, 5*time.Second, 10*time.Millisecond, "a replicated map joined by NewSink is still subscribed")

			m, err := rmap.Join(t.Context(), consumersMapName(stream), rdb)
			require.NoError(t, err)
			defer m.Close()
			assert.Empty(t, m.Map(), "the consumers map holds a consumer of the failed NewSink")
			if tc.group {
				consumers, err := rdb.XInfoConsumers(t.Context(), stream.key, tc.sink).Result()
				require.NoError(t, err)
				assert.Empty(t, consumers, "the consumer group holds a consumer of the failed NewSink")
			}
		})
	}
}

// TestNewSinkKeepsGroupOfConcurrentRemoveStream checks that NewSink adds its
// consumer to the consumers map of the stream before it creates the consumer
// group and the consumer, so a concurrent RemoveStream of the last other
// instance of the sink keeps the group and the new sink reads the stream
// (issue #531, tla/cfg/newsink_asis.cfg).
func TestNewSinkKeepsGroupOfConcurrentRemoveStream(t *testing.T) {
	cases := []struct {
		name string
		// after is the command of NewSink that RemoveStream runs after.
		after commandArgs
	}{
		{name: "after group creation", after: commandArgs{"xgroup", "create"}},
		{name: "after consumer creation", after: commandArgs{"xgroup", "createconsumer"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			rdb.AddHook(interleaveHook{})
			ctx := t.Context()
			main := newTestStream(t, rdb, "newsink-race-main")
			stream := newTestStream(t, rdb, "newsink-race")
			remover := newTestSink(t, main, "sink")
			require.NoError(t, remover.AddStream(ctx, stream))

			removed := false
			var removeErr error
			newCtx := interleaveAfter(ctx, tc.after, func() {
				removed = true
				removeErr = remover.RemoveStream(ctx, stream)
			})
			sink, err := stream.NewSink(newCtx, "sink",
				options.WithSinkBlockDuration(50*time.Millisecond),
				options.WithSinkAckGracePeriod(time.Second))
			require.NoError(t, err)
			t.Cleanup(func() {
				sink.Close(context.Background())
			})
			require.True(t, removed, "RemoveStream did not run inside NewSink")
			require.NoError(t, removeErr)
			assert.Equal(t, []string{sinkConsumer(sink)}, streamConsumers(t, rdb, stream))
			requireGroups(t, rdb, stream, 1)

			// Stop the old reader before publishing: its in-flight read may still
			// include the removed stream until its blocking call returns.
			remover.Close(ctx)
			events := sink.Subscribe()
			_, err = stream.Add(ctx, "event", []byte("payload"))
			require.NoError(t, err)
			ev := receiveEvent(t, events)
			assert.Equal(t, stream.Name, ev.StreamName)
			require.NoError(t, sink.Ack(ctx, ev))
		})
	}
}

// DialHook implements redis.Hook.
func (failCommandHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook implements redis.Hook.
func (h failCommandHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.args.matches(cmd) {
			cmd.SetErr(errInjectedCommand)
			return errInjectedCommand
		}
		return next(ctx, cmd)
	}
}

// ProcessPipelineHook implements redis.Hook.
func (failCommandHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

// DialHook implements redis.Hook.
func (failScriptHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook implements redis.Hook.
func (h failScriptHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.matches(cmd) {
			cmd.SetErr(errInjectedCommand)
			return errInjectedCommand
		}
		return next(ctx, cmd)
	}
}

// ProcessPipelineHook implements redis.Hook.
func (failScriptHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

// mapUpdatesChannel is the pub/sub channel of the replicated map name, as
// rmap.Join names it.
func mapUpdatesChannel(name string) string {
	return fmt.Sprintf("map:%s:updates", name)
}

// matches reports whether cmd is a script call that names h.key.
func (h failScriptHook) matches(cmd redis.Cmder) bool {
	script := commandArgs{"evalsha"}.matches(cmd) || commandArgs{"eval"}.matches(cmd)
	return script && slices.ContainsFunc(cmd.Args(), func(arg any) bool {
		return fmt.Sprint(arg) == h.key
	})
}

// matches reports whether the leading arguments of cmd are a.
func (a commandArgs) matches(cmd redis.Cmder) bool {
	args := cmd.Args()
	if len(args) < len(a) {
		return false
	}
	for i, arg := range a {
		if !strings.EqualFold(fmt.Sprint(args[i]), arg) {
			return false
		}
	}
	return true
}
