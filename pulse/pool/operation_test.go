package pool

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming"
)

// Direct worker/ownership fixtures deliberately bypass the delivery queue. Give
// them the same durable incarnation precondition as production admission; never
// accept unidentified wire jobs in production to accommodate these fixtures.
var luaIdentifyTestJob = redis.NewScript(luaOperation + `
local token = redis.call("HGET", KEYS[2], ARGV[1])
if not token_after(token, redis.call("HGET", KEYS[3], ARGV[1])) then
   token = next_token(KEYS[1], ARGV[1])
   redis.call("HSET", KEYS[2], ARGV[1], token)
end
return token
`)

func TestIntentPlanAdmission(t *testing.T) {
	cases := []struct {
		name     string
		plan     *JobPlan
		rejected bool
	}{
		{name: "ignored Stop", plan: &JobPlan{StopAll: true, Stop: []string{"="}}},
		{name: "invalid Start", plan: &JobPlan{Start: []*JobParam{{Key: "valid"}, {Key: "="}}}, rejected: true},
		{name: "nil Start", plan: &JobPlan{Start: []*JobParam{nil}}, rejected: true},
		{name: "invalid Stop", plan: &JobPlan{Stop: []string{""}}, rejected: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			node := sinklessNode(t, startTestRedis(t), "intent-plan")
			sched := intentTestScheduler(t, node, "producer")
			require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{Start: []*JobParam{{Key: "key"}}}))
			original := readTestIntent(t, sched)
			err := sched.publishIntent(t.Context(), tc.plan)
			if tc.rejected {
				require.Error(t, err)
				require.Equal(t, original, readTestIntent(t, sched))
				require.False(t, node.rdb.HExists(t.Context(), rmapContentKey(sched.name), "valid").Val())
			} else {
				require.NoError(t, err)
				require.Equal(t, "s", readTestIntent(t, sched).Kind)
			}
			require.Zero(t, node.rdb.XLen(t.Context(), node.poolStream.Key()).Val())
		})
	}
}

func TestIntentIssuanceRejectsSupersededRevision(t *testing.T) {
	for _, reset := range []bool{false, true} {
		t.Run(fmt.Sprint("reset=", reset), func(t *testing.T) {
			node := sinklessNode(t, startTestRedis(t), "intent-revision")
			sched := intentTestScheduler(t, node, "producer")
			require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{Start: []*JobParam{{Key: "key"}}}))
			old := readTestIntent(t, sched)
			if reset {
				require.NoError(t, sched.jobMap.Reset(t.Context()))
				require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{Start: []*JobParam{{Key: "key"}}}))
			} else {
				require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{StopAll: true}))
			}
			current := readTestIntent(t, sched)
			require.NotEqual(t, old.Revision, current.Revision)
			guard, err := sched.issueIntent(t.Context(), "key", old)
			require.NoError(t, err)
			require.Empty(t, guard)
			require.Zero(t, node.rdb.XLen(t.Context(), node.poolStream.Key()).Val())
			require.False(t, node.rdb.HExists(t.Context(), node.PoolName+":owner-epochs", "key").Val())
		})
	}
}

func TestIntentRetryPreservesIncarnationAndCutoff(t *testing.T) {
	for _, stop := range []bool{false, true} {
		t.Run(fmt.Sprint("stop=", stop), func(t *testing.T) {
			node := sinklessNode(t, startTestRedis(t), "intent-retry", WithAckGracePeriod(time.Millisecond))
			sched := intentTestScheduler(t, node, "producer")
			plan := &JobPlan{Start: []*JobParam{{Key: "key"}}}
			if stop {
				plan = &JobPlan{Stop: []string{"key"}}
			}
			require.NoError(t, sched.publishIntent(t.Context(), plan))
			intent := readTestIntent(t, sched)
			_, err := luaIssueIntent.Load(t.Context(), node.rdb).Result()
			require.NoError(t, err)
			var lost atomic.Bool
			lost.Store(true)
			node.rdb.AddHook(constructionFailureHook{after: true, match: func(cmd redis.Cmder) bool {
				if cmd.Name() != "evalsha" || fmt.Sprint(cmd.Args()[1]) != luaIssueIntent.Hash() {
					return false
				}
				return lost.CompareAndSwap(true, false)
			}})
			_, err = sched.issueIntent(t.Context(), "key", intent)
			require.ErrorIs(t, err, errConstructionInjected)
			issued := readTestIntent(t, sched)
			require.NotEmpty(t, issued.Token)
			require.NotEmpty(t, issued.EventID)
			counter := node.rdb.HGet(t.Context(), node.PoolName+":owner-epochs", "key").Val()
			// There is no sink group; deleting the lost delivery makes it
			// authoritatively absent. Age only its guard, never its identity.
			require.NoError(t, node.rdb.XDel(t.Context(), node.poolStream.Key(), issued.EventID).Err())
			if !stop {
				setGuard(t, node, "key", "1:"+issued.EventID)
			}
			_, err = sched.issueIntent(t.Context(), "key", issued)
			require.NoError(t, err)
			retried := readTestIntent(t, sched)
			require.Equal(t, issued.Token, retried.Token)
			require.Equal(t, counter, node.rdb.HGet(t.Context(), node.PoolName+":owner-epochs", "key").Val())
			require.NotEqual(t, issued.EventID, retried.EventID)
			wire := readTestPoolEvent(t, node, retried.EventID)
			op, err := unmarshalOperation(wire.Payload)
			require.NoError(t, err)
			require.Equal(t, issued.Token, op.token)
		})
	}
}

func TestRetriedStopCannotStopFreshStart(t *testing.T) {
	node := sinklessNode(t, startTestRedis(t), "intent-fresh")
	worker, err := node.AddWorker(t.Context(), newRecordingHandler())
	require.NoError(t, err)
	sched := intentTestScheduler(t, node, "producer")
	require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{Stop: []string{"key"}}))
	_, err = sched.issueIntent(t.Context(), "key", readTestIntent(t, sched))
	require.NoError(t, err)
	original := readTestIntent(t, sched)
	old := readTestPoolEvent(t, node, original.EventID)
	op, err := unmarshalOperation(old.Payload)
	require.NoError(t, err)
	require.NoError(t, worker.stopOperation(t.Context(), op))
	// The stop completed but its acknowledgment was lost. Admit fresh work.
	job := marshalJob(&Job{Key: "key", Payload: []byte("fresh"), NodeID: node.ID, CreatedAt: time.Now()})
	_, id, err := node.claimDispatch(t.Context(), "key", job)
	require.NoError(t, err)
	require.NoError(t, node.routeWorkerEvent(t.Context(), readTestPoolEvent(t, node, id)))
	require.Eventually(t, func() bool {
		return len(worker.Jobs()) == 1
	}, time.Second, time.Millisecond)
	require.NoError(t, node.rdb.XDel(t.Context(), node.poolStream.Key(), original.EventID).Err())
	_, err = sched.issueIntent(t.Context(), "key", original)
	require.NoError(t, err)
	retry := readTestIntent(t, sched)
	require.Equal(t, original.Token, retry.Token)
	require.NoError(t, node.routeWorkerEvent(t.Context(), readTestPoolEvent(t, node, retry.EventID)))
	require.Eventually(t, func() bool {
		return !node.rdb.HExists(t.Context(), rmapContentKey(sched.name), "key").Val()
	}, time.Second, time.Millisecond, "retried stop was not acknowledged")
	jobs := worker.Jobs()
	require.Len(t, jobs, 1)
	require.Equal(t, []byte("fresh"), jobs[0].Payload)
}

// TestIntentPublicationFailure never issues work after an unknown publication.
// A committed record remains available to a successor even when its reply is lost.
func TestIntentPublicationFailure(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(fmt.Sprint("after=", after), func(t *testing.T) {
			node := sinklessNode(t, startTestRedis(t), "intent-publication")
			sched := intentTestScheduler(t, node, "producer")
			_, err := luaPublishIntent.Load(t.Context(), node.rdb).Result()
			require.NoError(t, err)
			var fail atomic.Bool
			fail.Store(true)
			node.rdb.AddHook(constructionFailureHook{after: after, match: func(cmd redis.Cmder) bool {
				return cmd.Name() == "evalsha" && fmt.Sprint(cmd.Args()[1]) == luaPublishIntent.Hash() && fail.CompareAndSwap(true, false)
			}})
			sched.applyPlan(t.Context(), &JobPlan{Start: []*JobParam{{Key: "key"}}})
			require.Zero(t, node.rdb.XLen(t.Context(), node.poolStream.Key()).Val())
			require.Equal(t, after, node.rdb.HExists(t.Context(), rmapContentKey(sched.name), "key").Val())
			if after {
				successor := intentTestScheduler(t, node, "producer")
				require.NoError(t, successor.publishIntent(t.Context(), &JobPlan{StopAll: true}))
				require.Equal(t, "s", readTestIntent(t, successor).Kind)
			}
		})
	}
}

// TestIntentStopFailureRetainsResponsibility checks both callbacks and release
// writes, including a release which committed before its reply was lost.
func TestIntentStopFailureRetainsResponsibility(t *testing.T) {
	for _, failure := range []string{"callback", "release-before", "release-after"} {
		t.Run(failure, func(t *testing.T) {
			node := sinklessNode(t, startTestRedis(t), "intent-stop-failure", WithWorkerTTL(time.Hour))
			worker := ownershipWorker(t, node, "worker")
			if failure == "callback" {
				release := make(chan struct{})
				close(release)
				worker.handler = &failingStopHandler{failFirst: true, entered: make(chan struct{}), release: release}
			}
			require.NoError(t, startIdentifiedJob(t.Context(), worker, &Job{Key: "key", Payload: []byte("value")}))
			sched := intentTestScheduler(t, node, "producer")
			require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{Stop: []string{"key"}}))
			_, err := sched.issueIntent(t.Context(), "key", readTestIntent(t, sched))
			require.NoError(t, err)
			issued := readTestIntent(t, sched)
			op, err := unmarshalOperation(readTestPoolEvent(t, node, issued.EventID).Payload)
			require.NoError(t, err)
			if failure != "callback" {
				node.rdb.AddHook(&releaseReplyHook{after: failure == "release-after"})
			}
			require.Error(t, worker.stopOperation(t.Context(), op))
			require.Equal(t, issued, readTestIntent(t, sched))
			require.NoError(t, worker.stopOperation(t.Context(), op))
			require.NoError(t, node.completeIntent(t.Context(), op, evStopJob))
			require.False(t, node.rdb.HExists(t.Context(), rmapContentKey(sched.name), "key").Val())
			require.Empty(t, worker.Jobs())
			require.False(t, node.rdb.HExists(t.Context(), node.ownersKey(), "key").Val())
		})
	}
}

// TestIntentStopCallbackRetryAcknowledgment follows failed and successful stop
// acknowledgments through the real worker/router before removing responsibility.
func TestIntentStopCallbackRetryAcknowledgment(t *testing.T) {
	node := addTestNode(t, startTestRedis(t), "intent-stop-ack")
	release := make(chan struct{})
	close(release)
	handler := &failingStopHandler{failFirst: true, entered: make(chan struct{}), release: release}
	_, err := node.AddWorker(t.Context(), handler)
	require.NoError(t, err)
	sched := intentTestScheduler(t, node, "producer")
	sched.applyPlan(t.Context(), &JobPlan{Start: []*JobParam{{Key: "key"}}})
	sched.applyPlan(t.Context(), &JobPlan{StopAll: true})
	require.Eventually(t, func() bool {
		return handler.stops.Load() == 1
	}, time.Second, time.Millisecond)
	pending := readTestIntent(t, sched)
	require.False(t, pending.Complete)
	require.Equal(t, "s", pending.Kind)
	require.Eventually(t, func() bool {
		require.NoError(t, sched.reconcileIntent(t.Context()))
		return !node.rdb.HExists(t.Context(), rmapContentKey(sched.name), "key").Val()
	}, 3*time.Second, 5*time.Millisecond)
	require.EqualValues(t, 2, handler.stops.Load())
}

func TestIntentNewStartSurvivesOldCompletion(t *testing.T) {
	node := sinklessNode(t, startTestRedis(t), "intent-new-plan")
	worker, err := node.AddWorker(t.Context(), newRecordingHandler())
	require.NoError(t, err)
	require.NoError(t, startIdentifiedJob(t.Context(), worker, &Job{Key: "key", Payload: []byte("old")}))
	sched := intentTestScheduler(t, node, "producer")
	require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{Stop: []string{"key"}}))
	_, err = sched.issueIntent(t.Context(), "key", readTestIntent(t, sched))
	require.NoError(t, err)
	stop := readTestIntent(t, sched)
	require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{Start: []*JobParam{{Key: "key", Payload: []byte("new")}}}))
	newer := readTestIntent(t, sched)
	require.NotEqual(t, stop.Revision, newer.Revision)
	_, err = sched.issueIntent(t.Context(), "key", newer)
	require.NoError(t, err)
	require.Empty(t, readTestIntent(t, sched).Token, "fenced old owner cannot satisfy the new Start")
	op, err := unmarshalOperation(readTestPoolEvent(t, node, stop.EventID).Payload)
	require.NoError(t, err)
	require.NoError(t, worker.stopOperation(t.Context(), op))
	require.NoError(t, node.completeIntent(t.Context(), op, evStopJob))
	require.Equal(t, newer.Revision, readTestIntent(t, sched).Revision)
	_, err = sched.issueIntent(t.Context(), "key", newer)
	require.NoError(t, err)
	admitted := readTestIntent(t, sched)
	require.True(t, tokenAfter(admitted.Token, stop.Token))
	require.NoError(t, node.routeWorkerEvent(t.Context(), readTestPoolEvent(t, node, admitted.EventID)))
	require.Eventually(t, func() bool {
		return readTestIntent(t, sched).Complete
	}, time.Second, time.Millisecond)
	require.Len(t, worker.Jobs(), 1)
	require.Equal(t, []byte("new"), worker.Jobs()[0].Payload)
}

func TestIntentResetPreservesAdmittedStart(t *testing.T) {
	node := sinklessNode(t, startTestRedis(t), "intent-reset-admitted")
	worker, err := node.AddWorker(t.Context(), newRecordingHandler())
	require.NoError(t, err)
	sched := intentTestScheduler(t, node, "producer")
	require.NoError(t, sched.publishIntent(t.Context(), &JobPlan{Start: []*JobParam{{Key: "key"}}}))
	_, err = sched.issueIntent(t.Context(), "key", readTestIntent(t, sched))
	require.NoError(t, err)
	admitted := readTestIntent(t, sched)
	require.NoError(t, sched.jobMap.Reset(t.Context()))
	require.NoError(t, node.routeWorkerEvent(t.Context(), readTestPoolEvent(t, node, admitted.EventID)))
	require.Eventually(t, func() bool {
		return len(worker.Jobs()) == 1
	}, time.Second, time.Millisecond)
	require.False(t, node.rdb.HExists(t.Context(), rmapContentKey(sched.name), "key").Val())
}

func TestUnidentifiedStartDoesNotClaimOwnership(t *testing.T) {
	node := sinklessNode(t, startTestRedis(t), "unidentified-start")
	worker := ownershipWorker(t, node, "worker")
	job := &Job{Key: "key", Payload: []byte("value")}
	require.NoError(t, worker.startJob(t.Context(), job))
	require.Empty(t, worker.Jobs())
	require.False(t, node.rdb.HExists(t.Context(), node.ownersKey(), job.Key).Val())
	_, err := unmarshalOperation(marshalJob(job))
	require.ErrorIs(t, err, errMalformedMessage)
}

func identifyTestJob(ctx context.Context, node *Node, job *Job) error {
	if job.incarnation != "" {
		return nil
	}
	token, err := luaIdentifyTestJob.Run(ctx, node.rdb, []string{
		node.PoolName + ":owner-epochs", node.PoolName + ":job-incarnations", node.PoolName + ":job-cutoffs",
	}, job.Key).Text()
	job.incarnation = token
	return err
}

func startIdentifiedJob(ctx context.Context, worker *Worker, job *Job) error {
	if err := identifyTestJob(ctx, worker.node, job); err != nil {
		return err
	}
	return worker.startJob(ctx, job)
}

func readTestIntent(t *testing.T, sched *scheduler) *schedulerIntent {
	t.Helper()
	raw, err := sched.node.rdb.HGet(t.Context(), rmapContentKey(sched.name), "key").Result()
	require.NoError(t, err)
	var intent schedulerIntent
	require.NoError(t, json.Unmarshal([]byte(raw), &intent))
	return &intent
}

func readTestPoolEvent(t *testing.T, node *Node, id string) *streaming.Event {
	t.Helper()
	messages, err := node.rdb.XRange(t.Context(), node.poolStream.Key(), id, id).Result()
	require.NoError(t, err)
	require.Len(t, messages, 1)
	return &streaming.Event{ID: id, EventName: messages[0].Values["n"].(string), Payload: []byte(messages[0].Values["p"].(string))}
}

func marshalTestOperation(t *testing.T, node *Node, job *Job) []byte {
	t.Helper()
	require.NoError(t, identifyTestJob(t.Context(), node, job))
	payload := marshalJob(job)
	data := make([]byte, 0, 20+len(job.Key)+len(job.incarnation)+len(payload))
	for _, field := range []string{job.Key, job.incarnation, "", "", string(payload)} {
		data = append(data, marshalJobKey(field)...)
	}
	return data
}
