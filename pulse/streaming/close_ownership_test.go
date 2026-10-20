package streaming

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type shutdownCommandGate struct {
	match    func(redis.Cmder) bool
	entered  chan struct{}
	canceled chan struct{}
	release  chan struct{}
	once     sync.Once
}

func TestSinkCloseCancelsBeforeMaintenanceLock(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "close-maintenance")
	sink := newTestSink(t, stream, "owner", options.WithSinkAckGracePeriod(20*time.Millisecond))
	gate := &shutdownCommandGate{
		match: func(cmd redis.Cmder) bool {
			return cmd.Name() == "evalsha" && strings.Contains(strings.Join(stringArgs(cmd.Args()), " "), consumersMapKey(stream)) &&
				strings.Contains(strings.Join(stringArgs(cmd.Args()), " "), `"lastKeepAlive":`)
		},
		entered: make(chan struct{}), canceled: make(chan struct{}), release: make(chan struct{}),
	}
	rdb.AddHook(gate)
	var release sync.Once
	done := make(chan struct{})
	t.Cleanup(func() {
		release.Do(func() {
			close(gate.release)
		})
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("Close did not join maintenance")
		}
	})
	select {
	case <-gate.entered:
	case <-time.After(time.Second):
		t.Fatal("keepalive command did not enter")
	}
	go func() {
		sink.Close(t.Context())
		close(done)
	}()
	select {
	case <-gate.canceled:
	case <-time.After(200 * time.Millisecond):
		t.Error("Close waited for maintenance mutex before canceling runtime")
	}
	select {
	case <-done:
		t.Error("Close returned before maintenance completed")
	default:
	}
	release.Do(func() {
		close(gate.release)
	})
}

func TestSinkCloseJoinsAdmittedStreamAcquisition(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "rollback"
		}
		t.Run(name, func(t *testing.T) {
			rdb := startTestRedis(t)
			main := newTestStream(t, rdb, "close-add-main")
			added := newTestStream(t, rdb, "close-add-owned")
			sink := newTestSink(t, main, "owner")
			gate := &shutdownCommandGate{
				match: func(cmd redis.Cmder) bool {
					return cmd.Name() == "xgroup" && len(cmd.Args()) > 2 && cmd.Args()[1] == "create" && cmd.Args()[2] == added.key
				},
				entered: make(chan struct{}), canceled: make(chan struct{}), release: make(chan struct{}),
			}
			rdb.AddHook(gate)
			rdb.AddHook(failGroupCreateHook{})
			ctx := t.Context()
			if fail {
				ctx = failGroupCreate(ctx)
			}
			result := make(chan error, 1)
			go func() {
				result <- sink.AddStream(ctx, added)
			}()
			select {
			case <-gate.entered:
			case <-time.After(time.Second):
				t.Fatal("group creation did not enter")
			}
			done := make(chan struct{})
			var release sync.Once
			t.Cleanup(func() {
				release.Do(func() {
					close(gate.release)
				})
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Error("Close did not join acquisition")
				}
			})
			go func() {
				sink.Close(t.Context())
				close(done)
			}()
			require.Eventually(t, sink.lifetime.Stopping, time.Second, time.Millisecond)
			select {
			case <-done:
				t.Error("Close returned before acquired map was attached or rolled back")
			default:
			}
			release.Do(func() {
				close(gate.release)
			})
			err := <-result
			if fail {
				require.ErrorIs(t, err, errInjectedGroupCreate)
			} else {
				require.NoError(t, err)
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("Close did not join map cleanup")
			}
			channel := "map:" + consumersMapName(added) + ":updates"
			require.Eventually(t, func() bool {
				counts, err := rdb.PubSubNumSub(t.Context(), channel).Result()
				require.NoError(t, err)
				return counts[channel] == 0
			}, time.Second, time.Millisecond)
			require.ErrorContains(t, sink.AddStream(t.Context(), added), "closing")
			require.NoError(t, rdb.Ping(t.Context()).Err())
		})
	}
}

func stringArgs(args []any) []string {
	result := make([]string, 0, len(args))
	for _, arg := range args {
		if s, ok := arg.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func (h *shutdownCommandGate) DialHook(next redis.DialHook) redis.DialHook {
	return next
}
func (h *shutdownCommandGate) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *shutdownCommandGate) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.match(cmd) {
			h.once.Do(func() {
				close(h.entered)
				select {
				case <-ctx.Done():
					close(h.canceled)
					<-h.release
				case <-h.release:
				}
			})
		}
		return next(ctx, cmd)
	}
}
