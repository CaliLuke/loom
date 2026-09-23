package pool

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestEvictionStopsJobHandlers replays the runner_false_death TLC trace
// (pulse/pool/tla/cfg/runner_false_death.cfg, 12 states, violates
// AtMostOneRunner on main at 967f4fbe) and adds the eviction that follows:
//
//  1. k1 and k2 are dispatched and routed to w1 on nodeA (states 2-3, 5),
//  2. w1's keep-alive looks stale to nodeB (state 4),
//  3. nodeB cleans up w1: it requeues both keys and deletes w1 (states 8-10),
//  4. w2 on nodeB starts both keys (states 11-12),
//  5. nodeA evicts w1 in handleWorkerMapUpdate.
//
// Eviction used to call worker.stop only, so w1's handlers kept running next
// to w2's until the process exited. It must stop every job handler of w1.
func TestEvictionStopsJobHandlers(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	const pool = "evict-stop"
	nodeA := addTestNode(t, rdb, pool)
	handlerA := newRecordingHandler()
	w1, err := nodeA.AddWorker(ctx, handlerA)
	require.NoError(t, err)

	// Pick keys that hash to w1, the older worker, once w2 joins, so adding
	// w2 does not rebalance them.
	keys := keysHashedTo(nodeA, 0, 2, 2)
	for _, key := range keys {
		require.NoError(t, nodeA.DispatchJob(ctx, key, []byte(key)))
	}
	require.Len(t, w1.Jobs(), 2)

	nodeB := addTestNode(t, rdb, pool)
	handlerB := newRecordingHandler()
	_, err = nodeB.AddWorker(ctx, handlerB)
	require.NoError(t, err)

	// Step 3: nodeB acts on a stale keep-alive snapshot of w1.
	require.Eventually(t, func() bool {
		values, ok := nodeB.jobMap.GetValues(w1.ID)
		return ok && len(values) == 2 && len(nodeB.JobKeys()) == 2
	}, 10*time.Second, 5*time.Millisecond)
	nodeB.cleanupWorker(ctx, w1.ID)

	// Step 4: w2 starts the requeued jobs.
	for _, key := range keys {
		require.Eventually(t, func() bool {
			_, ok := handlerB.startedPayload(key)
			return ok
		}, 10*time.Second, 5*time.Millisecond, "w2 did not start %s", key)
	}

	// Step 5: nodeA sees w1 gone from the worker map and evicts it.
	require.Eventually(t, func() bool {
		_, ok := nodeA.workerMap.Get(w1.ID)
		return !ok
	}, 10*time.Second, 5*time.Millisecond)
	nodeA.handleWorkerMapUpdate(ctx)

	// The node's own watchWorkers loop may run the same eviction
	// concurrently, so wait for whichever call does it.
	for _, key := range keys {
		require.Eventually(t, func() bool {
			return handlerA.wasStopped(key)
		}, 5*time.Second, 5*time.Millisecond, "evicted w1 still runs %s", key)
	}
	require.Empty(t, w1.Jobs())
	require.True(t, w1.IsStopped())
}

// keysHashedTo returns count job keys that the node's hash assigns to bucket
// out of buckets workers.
func keysHashedTo(node *Node, bucket, buckets int64, count int) []string {
	keys := make([]string, 0, count)
	for i := 0; len(keys) < count; i++ {
		key := fmt.Sprintf("k%d", i)
		if node.h.Hash(key, buckets) == bucket {
			keys = append(keys, key)
		}
	}
	return keys
}

// TestSecondWorkerStopWaitsForEventLoop checks that every caller of
// Worker.stop returns only after the event loop has exited, not just the
// first one. Eviction can run while close stops the same worker, and
// stopHandlers relies on no start being handled after stop returns.
func TestSecondWorkerStopWaitsForEventLoop(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, "stop-twice")
	handler := &gatedHandler{entered: make(chan struct{}), release: make(chan struct{})}
	worker, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)

	job := &Job{Key: "k1", Payload: []byte("p"), CreatedAt: time.Now(), NodeID: node.ID}
	_, err = node.poolStream.Add(ctx, evStartJob, marshalJob(job))
	require.NoError(t, err)
	select {
	case <-handler.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("handler.Start was not called")
	}

	// The first caller blocks while the event loop is inside startJob.
	firstDone := make(chan struct{})
	go func() {
		worker.stop(ctx)
		close(firstDone)
	}()
	require.Eventually(t, worker.IsStopped, 5*time.Second, time.Millisecond)

	// The second caller must also wait for startJob to finish.
	var jobsAtReturn int
	secondDone := make(chan struct{})
	go func() {
		worker.stop(ctx)
		jobsAtReturn = len(worker.Jobs())
		close(secondDone)
	}()
	time.Sleep(200 * time.Millisecond)
	select {
	case <-secondDone:
		t.Errorf("second stop returned before startJob finished")
	default:
	}
	close(handler.release)
	<-secondDone
	<-firstDone
	require.Equal(t, 1, jobsAtReturn, "second stop returned while startJob was running")
}

// stopRaceHandler is a JobHandler whose first Stop blocks until releaseStop is
// closed. It counts Start calls that run while that Stop is in progress.
type stopRaceHandler struct {
	starts      atomic.Int32
	concurrent  atomic.Int32
	stopping    atomic.Bool
	stopEntered chan struct{}
	releaseStop chan struct{}
	once        sync.Once
}

// Start implements JobHandler.
func (h *stopRaceHandler) Start(*Job) error {
	h.starts.Add(1)
	if h.stopping.Load() {
		h.concurrent.Add(1)
	}
	return nil
}

// Stop implements JobHandler.
func (h *stopRaceHandler) Stop(string) error {
	h.once.Do(func() {
		h.stopping.Store(true)
		close(h.stopEntered)
		<-h.releaseStop
		h.stopping.Store(false)
	})
	return nil
}

// TestStartDuringShutdownStopWaitsForStop replays the shutdown race:
// stopAllJobs runs while the event loop is still live and is inside
// handler.Stop(k) when a redelivered start for k reaches the worker. The
// start must wait for the stop to finish, so Start never runs concurrently
// with Stop, and afterwards w.jobs, jobMap and the payload agree.
func TestStartDuringShutdownStopWaitsForStop(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, "stop-start-race")
	handler := &stopRaceHandler{stopEntered: make(chan struct{}), releaseStop: make(chan struct{})}
	worker, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)
	require.NoError(t, node.DispatchJob(ctx, "k1", []byte("p")))

	// Shutdown's stopAllJobs blocks inside handler.Stop(k1).
	stopDone := make(chan struct{})
	go func() {
		node.stopAllJobs(ctx)
		close(stopDone)
	}()
	select {
	case <-handler.stopEntered:
	case <-time.After(10 * time.Second):
		t.Fatal("handler.Stop was not called")
	}

	// A redelivered start for k1 reaches the worker.
	job := &Job{Key: "k1", Payload: []byte("p"), CreatedAt: time.Now(), NodeID: node.ID}
	_, err = node.poolStream.Add(ctx, evStartJob, marshalJob(job))
	require.NoError(t, err)
	deadline := time.Now().Add(300 * time.Millisecond)
	for handler.starts.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(handler.releaseStop)
	<-stopDone

	require.Eventually(t, func() bool {
		return handler.starts.Load() == 2
	}, 10*time.Second, 5*time.Millisecond, "the redelivered start was not handled")
	require.Zero(t, handler.concurrent.Load(), "handler.Start ran while handler.Stop was in progress")
	require.Len(t, worker.Jobs(), 1)
	require.Eventually(t, func() bool {
		keys, ok := node.jobMap.GetValues(worker.ID)
		_, hasPayload := node.JobPayload("k1")
		return ok && len(keys) == 1 && keys[0] == "k1" && hasPayload
	}, 10*time.Second, 5*time.Millisecond, "running job k1 has no jobMap entry or payload")
}

// failingStopHandler is a JobHandler whose first Stop blocks until release is
// closed and then fails if failFirst is set. It counts Stop calls and Stops
// that overlap. With failLaterStarts, every Start after the first fails.
type failingStopHandler struct {
	starts          atomic.Int32
	stops           atomic.Int32
	overlapping     atomic.Int32
	active          atomic.Bool
	failFirst       bool
	failLaterStarts bool
	entered         chan struct{}
	release         chan struct{}
	once            sync.Once
}

// Start implements JobHandler.
func (h *failingStopHandler) Start(*Job) error {
	if h.starts.Add(1) > 1 && h.failLaterStarts {
		return fmt.Errorf("start refused")
	}
	return nil
}

// Stop implements JobHandler.
func (h *failingStopHandler) Stop(string) error {
	h.stops.Add(1)
	if h.active.Swap(true) {
		h.overlapping.Add(1)
		return nil
	}
	defer h.active.Store(false)
	var err error
	h.once.Do(func() {
		close(h.entered)
		<-h.release
		if h.failFirst {
			err = fmt.Errorf("stop failed")
		}
	})
	return err
}

// startBlockingStopWorker returns a node and a worker running key.
func startBlockingStopWorker(t *testing.T, pool, key string, handler *failingStopHandler) (*Node, *Worker) {
	t.Helper()
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, pool)
	worker, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)
	require.NoError(t, node.DispatchJob(ctx, key, []byte("p")))
	return node, worker
}

// waitEntered waits until the handler's first Stop is in progress.
func waitEntered(t *testing.T, handler *failingStopHandler) {
	t.Helper()
	select {
	case <-handler.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("handler.Stop was not called")
	}
}

// waitBriefly waits until done closes or 200ms pass.
func waitBriefly(done <-chan struct{}) {
	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
	}
}

// TestEvictionStopsJobRestoredByFailedStop covers an eviction that overlaps a
// failing stopJob, as in shutdown's stopAllJobs: stopJob takes k, eviction
// starts, then Stop fails and stopJob tracks k again. Eviction must still
// stop k and leave the worker with no tracked job.
func TestEvictionStopsJobRestoredByFailedStop(t *testing.T) {
	handler := &failingStopHandler{failFirst: true, entered: make(chan struct{}), release: make(chan struct{})}
	_, worker := startBlockingStopWorker(t, "evict-failed-stop", "k1", handler)
	stopDone := make(chan struct{})
	go func() {
		defer close(stopDone)
		if err := worker.stopJob(t.Context(), "k1"); err == nil {
			t.Errorf("stopJob succeeded, want the handler error")
		}
	}()
	waitEntered(t, handler)

	worker.stop(t.Context())
	evictDone := make(chan struct{})
	go func() {
		worker.stopHandlers()
		close(evictDone)
	}()
	waitBriefly(evictDone)
	close(handler.release)
	<-stopDone
	<-evictDone

	require.Empty(t, worker.Jobs(), "evicted worker still tracks a job")
	require.Equal(t, int32(2), handler.stops.Load(), "handler.Stop calls")
}

// TestStopJobWaitsForRebalanceStop covers a stopJob (shutdown's stopAllJobs)
// that overlaps rebalance of the same key: while rebalance is inside
// handler.Stop(k), stopJob must not stop k again, and must find k gone once
// rebalance has moved it.
//
// "other-worker" is not registered, so the node routes rebalance's requeued
// start back to this worker. failLaterStarts makes that start fail, so it
// never tracks k1 again, whether it runs before or after stopJob.
func TestStopJobWaitsForRebalanceStop(t *testing.T) {
	handler := &failingStopHandler{
		failLaterStarts: true,
		entered:         make(chan struct{}),
		release:         make(chan struct{}),
	}
	node, worker := startBlockingStopWorker(t, "rebalance-stop", "k1", handler)
	// Hash k1 to the other worker so rebalance moves it.
	active := []string{worker.ID, "other-worker"}
	if node.h.Hash("k1", 2) == 0 {
		active = []string{"other-worker", worker.ID}
	}
	rebalanceDone := make(chan struct{})
	go func() {
		worker.rebalance(t.Context(), active)
		close(rebalanceDone)
	}()
	waitEntered(t, handler)

	stopDone := make(chan struct{})
	var stopErr error
	go func() {
		stopErr = worker.stopJob(t.Context(), "k1")
		close(stopDone)
	}()
	waitBriefly(stopDone)
	close(handler.release)
	<-rebalanceDone
	<-stopDone

	require.Zero(t, handler.overlapping.Load(), "stopJob called handler.Stop during rebalance's Stop")
	require.Equal(t, int32(1), handler.stops.Load(), "handler.Stop calls")
	require.ErrorContains(t, stopErr, "not found")
}
