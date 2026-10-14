package streaming

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type lostGroupReply struct {
	once sync.Once
	lost chan struct{}
}

func TestClosePreservesLostReplyRecovery(t *testing.T) {
	for _, noAck := range []bool{false, true} {
		name := "pending"
		if noAck {
			name = "no-ack"
		}
		t.Run(name, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "lost-reply")
			hook := &lostGroupReply{lost: make(chan struct{})}
			rdb.AddHook(hook)
			opts := []options.Sink{options.WithSinkStartAtOldest(), options.WithSinkAckGracePeriod(time.Hour)}
			if noAck {
				opts = append(opts, options.WithSinkNoAck())
			}
			sink := newTestSink(t, stream, "owner", opts...)
			ch := sink.Subscribe()
			id, err := stream.Add(t.Context(), "event", []byte("payload"))
			require.NoError(t, err)
			select {
			case <-hook.lost:
			case <-time.After(time.Second):
				t.Fatal("server reply not admitted")
			}
			sink.Close(t.Context())
			for event := range ch {
				t.Errorf("lost reply unexpectedly delivered %s", event.ID)
			}
			pending, err := rdb.XPendingExt(t.Context(), &redis.XPendingExtArgs{Stream: stream.key, Group: sink.Name, Start: "-", End: "+", Count: 10}).Result()
			require.NoError(t, err)
			if noAck {
				require.Empty(t, pending)
			} else {
				require.Len(t, pending, 1)
				require.Equal(t, id, pending[0].ID)
			}
			require.NoError(t, rdb.Ping(t.Context()).Err())
		})
	}
}

func (h *lostGroupReply) DialHook(next redis.DialHook) redis.DialHook {
	return next
}
func (h *lostGroupReply) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *lostGroupReply) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() == "xreadgroup" && err == nil {
			h.once.Do(func() {
				err = errors.New("simulated lost server reply")
				cmd.SetErr(err)
				close(h.lost)
			})
		}
		return err
	}
}
