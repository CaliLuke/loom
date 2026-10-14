package streaming

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type emptyPollResult struct {
	result chan error
}

// A healthy native timeout must reach Redis's nil reply before the owned context
// deadline. Otherwise an idle stream is reported as a transport failure forever.
func TestHealthyOwnedEmptyPoll(t *testing.T) {
	for _, kind := range []string{"reader", "sink"} {
		for _, block := range []time.Duration{0, time.Hour} {
			t.Run(fmt.Sprintf("%s/block=%s", kind, block), func(t *testing.T) {
				srv := startTestServer(t)
				if !srv.Real() {
					t.Skip("requires native Redis blocking-read timing")
				}
				opts := *srv.Client.Options()
				opts.ContextTimeoutEnabled = true
				opts.ReadTimeout = -1
				opts.WriteTimeout = -1
				opts.MaxRetries = -1
				caller := redis.NewClient(&opts)
				t.Cleanup(func() {
					require.NoError(t, caller.Close())
				})
				stream := newTestStream(t, caller, "healthy-empty-poll")
				if kind == "reader" {
					reader, err := stream.NewReader(t.Context(), options.WithReaderBlockDuration(block))
					require.NoError(t, err)
					t.Cleanup(reader.Close)
					_, err = reader.xread(t.Context())
					require.ErrorIs(t, err, redis.Nil)
				} else {
					hook := &emptyPollResult{result: make(chan error, 1)}
					caller.AddHook(hook)
					sink, err := stream.NewSink(t.Context(), "healthy-empty", options.WithSinkBlockDuration(block))
					require.NoError(t, err)
					sink.Subscribe()
					t.Cleanup(func() {
						sink.Close(t.Context())
					})
					select {
					case err := <-hook.result:
						require.ErrorIs(t, err, redis.Nil)
					case <-time.After(7 * time.Second):
						t.Fatal("empty group poll did not return")
					}
				}
			})
		}
	}
}

func (h *emptyPollResult) DialHook(next redis.DialHook) redis.DialHook {
	return next
}
func (h *emptyPollResult) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h *emptyPollResult) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() == "xreadgroup" {
			select {
			case h.result <- err:
			default:
			}
		}
		return err
	}
}

func TestOwnedEmptyPollRespectsCallerDeadline(t *testing.T) {
	for _, source := range []string{"client", "context"} {
		t.Run(source, func(t *testing.T) {
			srv := startTestServer(t)
			opts := *srv.Client.Options()
			opts.ContextTimeoutEnabled = true
			opts.ReadTimeout = -1
			opts.WriteTimeout = -1
			opts.MaxRetries = -1
			if source == "client" {
				opts.ReadTimeout = 100 * time.Millisecond
			}
			caller := redis.NewClient(&opts)
			t.Cleanup(func() {
				require.NoError(t, caller.Close())
			})
			stream := newTestStream(t, caller, "empty-stricter-deadline")
			reader, err := stream.NewReader(t.Context(), options.WithReaderBlockDuration(0))
			require.NoError(t, err)
			t.Cleanup(reader.Close)
			ctx := t.Context()
			if source == "context" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			started := time.Now()
			_, err = reader.xread(ctx)
			require.Error(t, err)
			require.NotErrorIs(t, err, redis.Nil)
			require.Less(t, time.Since(started), time.Second)
		})
	}
}
