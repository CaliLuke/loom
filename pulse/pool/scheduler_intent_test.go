package pool

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/rmap"
	"github.com/CaliLuke/loom/pulse/streaming"
)

type laggedIntentMap struct {
	schedulerJobMap
}

// TestSchedulerRetainsUncertainIntent checks producer responsibility after a
// timed-out admission subsequently starts and a retry returns ErrJobExists.
func TestSchedulerRetainsUncertainIntent(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "scheduler-uncertain", WithAckGracePeriod(50*time.Millisecond))
	sched := intentTestScheduler(t, node, "first")
	sched.applyPlan(t.Context(), &JobPlan{Start: []*JobParam{{Key: "requested", Payload: []byte("value")}}})
	exists, err := rdb.HExists(t.Context(), rmapContentKey(sched.name), "requested").Result()
	require.NoError(t, err)
	if !exists {
		t.Error("timed-out admission lost producer responsibility")
	}

	handler := newRecordingHandler()
	_, err = node.AddWorker(t.Context(), handler)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		_, started := handler.startedPayload("requested")
		return started
	}, 3*time.Second, 5*time.Millisecond)
	sched.applyPlan(t.Context(), &JobPlan{Start: []*JobParam{{Key: "requested", Payload: []byte("value")}}})
	exists, err = rdb.HExists(t.Context(), rmapContentKey(sched.name), "requested").Result()
	require.NoError(t, err)
	if !exists {
		t.Error("ErrJobExists retry lost producer responsibility")
	}
	sched.applyPlan(t.Context(), &JobPlan{StopAll: true})
	require.Eventually(t, func() bool {
		return handler.wasStopped("requested")
	}, time.Second, 5*time.Millisecond, "StopAll missed the producer's running job")
}

// TestSchedulerCanceledAdmissionRetainsIntent checks cancellation after the
// Redis start/guard transaction commits, before the caller observes success.
func TestSchedulerCanceledAdmissionRetainsIntent(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "scheduler-cancel", WithAckGracePeriod(50*time.Millisecond))
	_, err := luaIssueIntent.Load(t.Context(), rdb).Result()
	require.NoError(t, err)
	sched := intentTestScheduler(t, node, "first")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	rdb.AddHook(constructionFailureHook{after: true, succeed: true, match: func(cmd redis.Cmder) bool {
		args := cmd.Args()
		return cmd.Name() == "evalsha" && len(args) > 1 && fmt.Sprint(args[1]) == luaIssueIntent.Hash()
	}, committed: func(redis.Cmder) {
		cancel()
	}})
	sched.applyPlan(ctx, &JobPlan{Start: []*JobParam{{Key: "requested"}}})
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	exists, err := rdb.HExists(t.Context(), rmapContentKey(jobPendingMapName(node.PoolName)), "requested").Result()
	require.NoError(t, err)
	require.True(t, exists, "control: start admission committed despite canceled reply")
	exists, err = rdb.HExists(t.Context(), rmapContentKey(sched.name), "requested").Result()
	require.NoError(t, err)
	require.True(t, exists, "committed canceled admission lost producer responsibility")
}

// TestSchedulerHandoffIgnoresReplicaLag checks a successor whose independent
// producer-map replica has not caught up to the already committed intent.
func TestSchedulerHandoffIgnoresReplicaLag(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "scheduler-handoff")
	handler := newRecordingHandler()
	_, err := node.AddWorker(t.Context(), handler)
	require.NoError(t, err)
	sched := intentTestScheduler(t, node, "first")
	sched.applyPlan(t.Context(), &JobPlan{Start: []*JobParam{{Key: "requested"}}})
	sched.jobMap = &laggedIntentMap{schedulerJobMap: sched.jobMap}
	sched.applyPlan(t.Context(), &JobPlan{StopAll: true})
	require.Eventually(t, func() bool {
		return handler.wasStopped("requested")
	}, time.Second, 5*time.Millisecond, "StopAll used stale replica instead of authoritative intent")
}

// TestSchedulerStopEnqueueRetainsIntent checks that an enqueued stop without a
// worker acknowledgment does not erase the producer's remaining responsibility.
func TestSchedulerStopEnqueueRetainsIntent(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "scheduler-stop-enqueue")
	sched := intentTestScheduler(t, node, "first")
	require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{Start: []*JobParam{{Key: "requested"}}}))
	sched.applyPlan(t.Context(), &JobPlan{Stop: []string{"requested"}})
	exists, err := rdb.HExists(t.Context(), rmapContentKey(sched.name), "requested").Result()
	require.NoError(t, err)
	require.True(t, exists, "queued stop is not confirmed completion")
}

// TestSchedulerStopAllProducerIsolation preserves successful dispatch, Start
// exclusion and isolation from another producer's disjoint responsibility.
func TestSchedulerStopAllProducerIsolation(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "scheduler-isolation")
	handler := newRecordingHandler()
	_, err := node.AddWorker(t.Context(), handler)
	require.NoError(t, err)
	first := intentTestScheduler(t, node, "first")
	second := intentTestScheduler(t, node, "second")
	first.applyPlan(t.Context(), &JobPlan{Start: []*JobParam{{Key: "stop"}, {Key: "keep"}}})
	second.applyPlan(t.Context(), &JobPlan{Start: []*JobParam{{Key: "other"}}})
	require.True(t, rdb.HExists(t.Context(), rmapContentKey(first.name), "stop").Val())
	require.True(t, rdb.HExists(t.Context(), rmapContentKey(first.name), "keep").Val())
	require.True(t, rdb.HExists(t.Context(), rmapContentKey(second.name), "other").Val())
	first.applyPlan(t.Context(), &JobPlan{StopAll: true, Start: []*JobParam{{Key: "keep"}}})
	require.Eventually(t, func() bool {
		return handler.wasStopped("stop")
	}, time.Second, 5*time.Millisecond)
	require.False(t, handler.wasStopped("keep"))
	require.False(t, handler.wasStopped("other"))
}

// TestSchedulerFreshStartAfterStop retains the valid new-incarnation case.
func TestSchedulerFreshStartAfterStop(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "scheduler-fresh-start")
	worker, err := node.AddWorker(t.Context(), newRecordingHandler())
	require.NoError(t, err)
	require.NoError(t, node.DispatchJob(t.Context(), "requested", []byte("first")))
	require.NoError(t, worker.stopJob(t.Context(), "requested"))
	require.NoError(t, node.DispatchJob(t.Context(), "requested", []byte("second")))
	jobs := worker.Jobs()
	require.Len(t, jobs, 1)
	require.Equal(t, []byte("second"), jobs[0].Payload)
}

// TestSchedulerLateRoutedStartCannotReviveStoppedJob follows the original
// admitted pool event through the real router, worker reader and ack loop.
func TestSchedulerLateRoutedStartCannotReviveStoppedJob(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "scheduler-late-route")
	handler := newRecordingHandler()
	worker, err := node.AddWorker(t.Context(), handler)
	require.NoError(t, err)
	require.NoError(t, node.DispatchJob(t.Context(), "requested", []byte("value")))
	messages, err := rdb.XRange(t.Context(), node.poolStream.Key(), "-", "+").Result()
	require.NoError(t, err)
	var original *streaming.Event
	for _, message := range messages {
		if message.Values["n"] == evStartJob {
			original = &streaming.Event{ID: message.ID, EventName: evStartJob, Payload: []byte(message.Values["p"].(string))}
		}
	}
	require.NotNil(t, original)
	require.NoError(t, node.StopJob(t.Context(), "requested"))
	require.Eventually(t, func() bool {
		return handler.wasStopped("requested") && len(worker.Jobs()) == 0
	}, time.Second, 5*time.Millisecond)
	// A second router had already acquired this original event before its ack,
	// and resumes delivery after the stop. Routing has no PEL or stop check.
	require.NoError(t, node.routeWorkerEvent(t.Context(), original))
	delivered, err := rdb.XRevRangeN(t.Context(), worker.stream.Key(), "+", "-", 1).Result()
	require.NoError(t, err)
	require.Len(t, delivered, 1)
	copyID := delivered[0].ID
	require.Eventually(t, func() bool {
		acks, readErr := rdb.XRange(t.Context(), "pulse:stream:"+nodeStreamName(node.PoolName, node.ID), "-", "+").Result()
		return readErr == nil && streamHasAck(acks, copyID)
	}, time.Second, 5*time.Millisecond, "late routed copy did not finish")
	require.Empty(t, worker.Jobs(), "late copy revived the confirmed stopped job")
}

func intentTestScheduler(t *testing.T, node *Node, producer string) *scheduler {
	t.Helper()
	name := node.PoolName + ":" + producer
	jobMap, err := rmap.Join(context.Background(), name, node.rdb)
	require.NoError(t, err)
	t.Cleanup(jobMap.Close)
	return &scheduler{name: name, node: node, jobMap: jobMap, logger: node.logger}
}

func (m *laggedIntentMap) Keys() []string {
	return nil
}
