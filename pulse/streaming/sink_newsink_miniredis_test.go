package streaming

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// failCommandHook fails every command whose leading arguments match args,
// ignoring case, with errInjectedCommand.
type failCommandHook struct {
	args []string
}

// errInjectedCommand is the error failCommandHook returns.
var errInjectedCommand = errors.New("injected command failure")

// TestNewSinkClosesJoinedMapsOnError checks that NewSink closes the
// replicated maps it joined when a later step fails, so a failed NewSink
// leaves no subscription to their update channels (issue #509).
func TestNewSinkClosesJoinedMapsOnError(t *testing.T) {
	cases := []struct {
		name string
		// sink is the name of the sink to create.
		sink string
		// fail makes a step of NewSink fail.
		fail func(t *testing.T, rdb *redis.Client, stream *Stream)
		want string
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
				rdb.AddHook(failCommandHook{args: []string{"xgroup", "createconsumer"}})
			},
			want: "failed to create consumer",
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
		if h.matches(cmd) {
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

// mapUpdatesChannel is the pub/sub channel of the replicated map name, as
// rmap.Join names it.
func mapUpdatesChannel(name string) string {
	return fmt.Sprintf("map:%s:updates", name)
}

// matches reports whether the leading arguments of cmd are h.args.
func (h failCommandHook) matches(cmd redis.Cmder) bool {
	args := cmd.Args()
	if len(args) < len(h.args) {
		return false
	}
	for i, arg := range h.args {
		if !strings.EqualFold(fmt.Sprint(args[i]), arg) {
			return false
		}
	}
	return true
}
