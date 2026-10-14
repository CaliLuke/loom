package pool

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type stopPublicationReplyHook struct {
	wait <-chan struct{}
}

func (h *stopPublicationReplyHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *stopPublicationReplyHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *stopPublicationReplyHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if err := next(ctx, cmd); err != nil {
			return err
		}
		args := cmd.Args()
		if len(args) > 1 && cmd.Name() == "evalsha" && args[1] == luaStopJob.Hash() {
			select {
			case <-h.wait:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
}

func TestReleaseFaultAsyncConsumerFirst(t *testing.T) {
	rdb := startTestRedis(t)
	n := addTestNode(t, rdb, "release-caller-order", WithWorkerTTL(time.Hour))
	hook := &releaseReplyHook{entered: make(chan struct{}), release: make(chan struct{})}
	workerCtx := context.WithValue(t.Context(), releaseReplyOwnerKey{}, hook)
	w, err := n.AddWorker(workerCtx, newRecordingHandler())
	require.NoError(t, err)
	require.NoError(t, startIdentifiedJob(t.Context(), w, &Job{Key: "k", NodeID: n.ID, CreatedAt: time.Now()}))
	_, err = luaStopJob.Load(t.Context(), rdb).Result()
	require.NoError(t, err)
	var releaseOnce sync.Once
	unblock := func() {
		releaseOnce.Do(func() {
			close(hook.release)
		})
	}
	t.Cleanup(unblock)
	rdb.AddHook(hook)
	rdb.AddHook(&stopPublicationReplyHook{wait: hook.entered})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- w.stopJob(ctx, "k")
	}()
	select {
	case <-hook.entered:
	case <-ctx.Done():
		t.Fatal("asynchronous consumer did not enter release")
	}
	unblock()
	require.NoError(t, <-done, "the direct caller retries the asynchronous consumer's failed release")
	require.True(t, hook.fired.Load())
	require.Empty(t, w.Jobs())
	owned, err := rdb.HExists(t.Context(), n.ownersKey(), "k").Result()
	require.NoError(t, err)
	require.False(t, owned)
}

func TestReleaseFaultAdmission(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(map[bool]string{false: "before", true: "after"}[after], func(t *testing.T) {
			hook := &releaseReplyHook{after: after, entered: make(chan struct{}), release: make(chan struct{})}
			var releaseOnce sync.Once
			unblock := func() {
				releaseOnce.Do(func() {
					close(hook.release)
				})
			}
			t.Cleanup(unblock)
			owner := context.WithValue(t.Context(), releaseReplyOwnerKey{}, hook)
			var calls atomic.Int32
			process := hook.ProcessHook(func(context.Context, redis.Cmder) error {
				calls.Add(1)
				return nil
			})
			command := func(ctx context.Context) redis.Cmder {
				return redis.NewCmd(ctx, "evalsha", luaReleaseJob.Hash(), 0)
			}
			// A non-target call cannot claim the fault even when it reaches Redis first.
			require.NoError(t, process(t.Context(), command(t.Context())))
			require.False(t, hook.fired.Load())
			done := make(chan error, 1)
			go func() {
				done <- process(owner, command(owner))
			}()
			select {
			case <-hook.entered:
			case <-time.After(time.Second):
				t.Fatal("owner did not enter release gate")
			}
			// A second matching caller must neither wait at nor close the same gate.
			require.NoError(t, process(owner, command(owner)))
			require.NoError(t, process(t.Context(), command(t.Context())))
			unblock()
			require.ErrorContains(t, <-done, "connection loss")
			expectedCalls := int32(3)
			if after {
				expectedCalls++
			}
			require.Equal(t, expectedCalls, calls.Load())
		})
	}
}

func TestReleaseFaultAfterNoScript(t *testing.T) {
	hook := &releaseReplyHook{after: true, entered: make(chan struct{}), release: make(chan struct{})}
	close(hook.release)
	owner := context.WithValue(t.Context(), releaseReplyOwnerKey{}, hook)
	noScript := errors.New("NOSCRIPT No matching script")
	var calls atomic.Int32
	process := hook.ProcessHook(func(context.Context, redis.Cmder) error {
		if calls.Add(1) == 1 {
			return noScript
		}
		return nil
	})
	require.ErrorIs(t, process(owner, redis.NewCmd(owner, "evalsha", luaReleaseJob.Hash(), 0)), noScript)
	require.False(t, hook.fired.Load())
	select {
	case <-hook.entered:
		t.Fatal("NOSCRIPT consumed fault admission")
	default:
	}
	require.ErrorContains(t, process(owner, redis.NewCmd(owner, "eval", "local remaining = {}", 0)), "connection loss")
	require.True(t, hook.fired.Load())
	require.EqualValues(t, 2, calls.Load())
}
