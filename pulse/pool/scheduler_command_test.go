package pool

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type intentFirstFailureHandler struct {
	starts  atomic.Int32
	entered chan struct{}
	release chan struct{}
}

// TestAuthoredStartAfterExternalStop separates a later command from supervision.
func TestAuthoredStartAfterExternalStop(t *testing.T) {
	node := addTestNode(t, startTestRedis(t), "authored-after-stop")
	worker, err := node.AddWorker(t.Context(), newRecordingHandler())
	require.NoError(t, err)
	sched := intentTestScheduler(t, node, "producer")
	plan := &JobPlan{Start: []*JobParam{{Key: "key", Payload: []byte("same")}}}
	sched.applyPlan(t.Context(), plan)
	require.Len(t, worker.Jobs(), 1)
	old := readTestIntent(t, sched)
	require.True(t, old.Complete)
	require.NoError(t, node.StopJob(t.Context(), "key"))
	require.Eventually(t, func() bool {
		return len(worker.Jobs()) == 0
	}, time.Second, time.Millisecond)
	// Reconciliation alone must not turn responsibility into supervision.
	sched.applyPlan(t.Context(), &JobPlan{})
	require.Empty(t, worker.Jobs())
	require.Equal(t, old.Revision, readTestIntent(t, sched).Revision)
	// A later authored command is actionable even with identical key/payload.
	sched.applyPlan(t.Context(), plan)
	require.Len(t, worker.Jobs(), 1, "later authored Start was discarded after external stop")
	fresh := readTestIntent(t, sched)
	require.NotEqual(t, old.Revision, fresh.Revision)
	require.True(t, tokenAfter(fresh.Token, old.Token))
}

// TestAuthoredStartAfterClaimFailure preserves a later retry after observed claim failure.
func TestAuthoredStartAfterClaimFailure(t *testing.T) {
	node := addTestNode(t, startTestRedis(t), "authored-after-failure")
	handler := &intentFirstFailureHandler{entered: make(chan struct{}), release: make(chan struct{})}
	var release sync.Once
	defer release.Do(func() {
		close(handler.release)
	})
	worker, err := node.AddWorker(t.Context(), handler)
	require.NoError(t, err)
	sched := intentTestScheduler(t, node, "producer")
	plan := &JobPlan{Start: []*JobParam{{Key: "key", Payload: []byte("same")}}}
	first := make(chan struct{})
	go func() {
		sched.applyPlan(t.Context(), plan)
		close(first)
	}()
	select {
	case <-handler.entered:
	case <-time.After(time.Second):
		t.Fatal("first handler did not enter")
	}
	// The owner/payload exist, but Handler.Start has not completed.
	sched.applyPlan(t.Context(), plan)
	observed := readTestIntent(t, sched)
	require.True(t, observed.Complete)
	release.Do(func() {
		close(handler.release)
	})
	select {
	case <-first:
	case <-time.After(time.Second):
		t.Fatal("failed first request did not return")
	}
	require.Empty(t, worker.Jobs())
	sched.applyPlan(t.Context(), &JobPlan{})
	require.EqualValues(t, 1, handler.starts.Load())
	sched.applyPlan(t.Context(), plan)
	require.EqualValues(t, 2, handler.starts.Load(), "later authored retry never reached Handler.Start")
	require.Len(t, worker.Jobs(), 1)
	require.NotEqual(t, observed.Revision, readTestIntent(t, sched).Revision)
}

// TestAuthoredStartAfterFencedPending replaces an ineligible pending predecessor.
func TestAuthoredStartAfterFencedPending(t *testing.T) {
	node := sinklessNode(t, startTestRedis(t), "authored-after-fence")
	worker, err := node.AddWorker(t.Context(), newRecordingHandler())
	require.NoError(t, err)
	sched := intentTestScheduler(t, node, "producer")
	plan := &JobPlan{Start: []*JobParam{{Key: "key", Payload: []byte("same")}}}
	require.NoError(t, sched.publishIntent(t.Context(), plan))
	_, err = sched.issueIntent(t.Context(), "key", readTestIntent(t, sched))
	require.NoError(t, err)
	pending := readTestIntent(t, sched)
	require.False(t, pending.Complete)
	// The old start remains undelivered while an external stop confirms absence.
	stop, err := node.issueStop(t.Context(), "key")
	require.NoError(t, err)
	require.NoError(t, worker.stopOperation(t.Context(), stop))
	require.Equal(t, pending, readTestIntent(t, sched))
	require.NoError(t, sched.publishIntent(t.Context(), plan))
	fresh := readTestIntent(t, sched)
	require.NotEqual(t, pending.Revision, fresh.Revision, "new command reused a fenced pending operation")
	_, err = sched.issueIntent(t.Context(), "key", fresh)
	require.NoError(t, err)
	issued := readTestIntent(t, sched)
	require.True(t, tokenAfter(issued.Token, stop.token))
	require.NoError(t, node.routeWorkerEvent(t.Context(), readTestPoolEvent(t, node, issued.EventID)))
	require.Eventually(t, func() bool {
		return len(worker.Jobs()) == 1
	}, time.Second, time.Millisecond)
	require.Equal(t, []byte("same"), worker.Jobs()[0].Payload)
}

// Start blocks and fails once, then accepts later commands.
func (h *intentFirstFailureHandler) Start(*Job) error {
	if h.starts.Add(1) == 1 {
		close(h.entered)
		<-h.release
		return errors.New("first Start failed")
	}
	return nil
}

// Stop implements JobHandler without additional side effects.
func (*intentFirstFailureHandler) Stop(string) error {
	return nil
}
