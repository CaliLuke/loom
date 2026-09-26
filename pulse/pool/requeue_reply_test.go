package pool

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type (
	// requeueReplyHook injects an error before or after Redis executes the
	// requeue. A competing requeue makes the original call return refused.
	requeueReplyHook struct {
		mode  string
		node  *Node
		job   *Job
		fired atomic.Bool
	}
)

// TestRebalanceRequeueReply replays RequeueReply.tla: after releasing a job,
// a failed or refused requeue must not restart it locally. A queued start
// runs on the target; a failed write is recovered by the orphan sweep.
func TestRebalanceRequeueReply(t *testing.T) {
	for _, mode := range []string{"success", "before", "after", "refused"} {
		t.Run(mode, func(t *testing.T) {
			ctx := t.Context()
			rdb := startTestRedis(t)
			node := sinklessNode(t, rdb, "requeue-reply-"+mode, WithWorkerTTL(time.Second), WithAckGracePeriod(time.Hour))
			handler := newRecordingHandler()
			worker, err := node.AddWorker(ctx, handler)
			require.NoError(t, err)
			job := &Job{Key: "k1", Payload: []byte("payload"), CreatedAt: time.Now(), NodeID: node.ID}
			require.NoError(t, worker.startJob(ctx, job))
			require.Eventually(t, func() bool {
				_, ok := node.JobPayload(job.Key)
				return ok
			}, time.Second, time.Millisecond)
			hook := &requeueReplyHook{mode: mode, node: node, job: job}
			rdb.AddHook(hook)

			worker.rebalance(ctx, []string{"target"})

			require.True(t, hook.fired.Load(), "fault seam was not exercised")
			require.Empty(t, worker.Jobs(), "requeue reply restarted the released job")
			require.True(t, handler.wasStopped(job.Key))
			require.Eventually(t, func() bool {
				keys, _ := node.jobMap.GetValues(worker.ID)
				return !slices.Contains(keys, job.Key)
			}, time.Second, time.Millisecond)
			entries, err := rdb.XRange(ctx, node.poolStream.Key(), "-", "+").Result()
			require.NoError(t, err)
			if mode == "before" {
				require.Zero(t, countStartEvents(t, entries)[job.Key])
			} else {
				require.Equal(t, 1, countStartEvents(t, entries)[job.Key])
			}
			// Advance only the orphan observation, not a wall-clock sleep. The
			// sweep must recover a failed write and preserve an existing start.
			node.orphanedPayloads.Store(job.Key, time.Now().Add(-3*time.Hour).UnixNano())
			node.requeueOrphanedPayloads(ctx)
			entries, err = rdb.XRange(ctx, node.poolStream.Key(), "-", "+").Result()
			require.NoError(t, err)
			require.Equal(t, 1, countStartEvents(t, entries)[job.Key])

			// Deliver to a second worker through the real start transition.
			// It is not registered, so no watcher can reorder the handoff.
			target := &Worker{
				ID:             "target",
				node:           node,
				handler:        newRecordingHandler(),
				jobsMap:        node.jobMap,
				jobPayloadsMap: node.jobPayloadMap,
				logger:         node.logger,
			}
			require.NoError(t, target.startJob(ctx, job))
			require.Len(t, target.Jobs(), 1)
			require.Empty(t, worker.Jobs(), "both workers run the job")
			require.NoError(t, target.stopJob(ctx, job.Key))
		})
	}
}

func (h *requeueReplyHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *requeueReplyHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		args := cmd.Args()
		if len(args) < 2 || h.fired.Load() {
			return next(ctx, cmd)
		}
		script, ok := args[1].(string)
		if !ok || !((cmd.Name() == "evalsha" && script == luaClaimRequeue.Hash()) ||
			(cmd.Name() == "eval" && strings.Contains(script, "return {1, add_start(KEYS[3], ARGV[3], ARGV[4], KEYS[1], KEYS[2], ARGV[1], ARGV[2])}"))) {
			return next(ctx, cmd)
		}
		if h.mode == "before" && h.fired.CompareAndSwap(false, true) {
			return errors.New("requeue request not delivered")
		}
		if h.mode == "refused" && h.fired.CompareAndSwap(false, true) {
			if _, err := h.node.claimRequeue(ctx, h.job); err != nil {
				return err
			}
		}
		if err := next(ctx, cmd); err != nil {
			return err
		}
		h.fired.Store(true)
		if h.mode == "after" {
			return errors.New("requeue reply lost")
		}
		return nil
	}
}

func (h *requeueReplyHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
