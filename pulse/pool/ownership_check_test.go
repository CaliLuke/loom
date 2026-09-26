package pool

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/stretchr/testify/require"
)

func TestCheckOwnership(t *testing.T) {
	for _, source := range []string{"worker", "job-snapshot", "node-snapshot"} {
		t.Run(source, func(t *testing.T) {
			rdb := startTestRedis(t)
			n := sinklessNode(t, rdb, "check-"+source, WithWorkerTTL(time.Hour))
			w := ownershipWorker(t, n, "w1")
			h := &fencingHandler{}
			w.handler = h
			n.localWorkers.Store(w.ID, w)
			t.Cleanup(func() {
				n.localWorkers.Delete(w.ID)
			})
			job := &Job{Key: "k", NodeID: n.ID}
			require.NoError(t, w.startJob(t.Context(), job))
			check := w
			switch source {
			case "job-snapshot":
				check = w.Jobs()[0].Worker
			case "node-snapshot":
				check = n.Workers()[0]
			}
			require.NoError(t, check.CheckOwnership(t.Context(), job.Key, job.Epoch))
			// A bad token must not stop a legitimate newer local incarnation.
			require.ErrorIs(t, check.CheckOwnership(t.Context(), job.Key, job.Epoch+1), ErrOwnershipLost)
			require.Zero(t, h.stops.Load())
			// Simulate an unbounded pause: ownership changes before local Stop.
			released, err := w.releaseJob(t.Context(), job, false)
			require.NoError(t, err)
			require.True(t, released)
			other := ownershipWorker(t, n, "w2")
			require.NoError(t, other.startJob(t.Context(), &Job{Key: "k", NodeID: n.ID}))
			require.ErrorIs(t, check.CheckOwnership(t.Context(), job.Key, job.Epoch), ErrOwnershipLost)
			w.stopRejectedJobs()
			require.EqualValues(t, 1, h.stops.Load())
			require.Empty(t, w.Jobs())
			require.Len(t, other.Jobs(), 1)
			require.ErrorIs(t, check.CheckOwnership(t.Context(), job.Key, job.Epoch), ErrOwnershipLost)
			require.EqualValues(t, 1, h.stops.Load())
		})
	}
}

func TestCheckOwnershipFailure(t *testing.T) {
	for _, failure := range []string{"context", "fenced", "stop"} {
		t.Run(failure, func(t *testing.T) {
			rdb := startTestRedis(t)
			n := sinklessNode(t, rdb, "check-failure-"+failure, WithWorkerTTL(time.Hour))
			w := ownershipWorker(t, n, "w1")
			h := &fencingHandler{}
			w.handler = h
			job := &Job{Key: "k", NodeID: n.ID}
			require.NoError(t, w.startJob(t.Context(), job))
			switch failure {
			case "context":
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				require.ErrorIs(t, w.CheckOwnership(ctx, job.Key, job.Epoch), context.Canceled)
				require.Zero(t, h.stops.Load())
			case "fenced":
				w.recordLease(time.Now().Add(-time.Hour))
				require.ErrorIs(t, w.CheckOwnership(t.Context(), job.Key, job.Epoch), ErrOwnershipLost)
				w.updateLeaseState(t.Context(), time.Now())
				require.EqualValues(t, 1, h.stops.Load())
				require.Len(t, w.Jobs(), 1)
			case "stop":
				require.NoError(t, rdb.HSet(t.Context(), n.ownersKey(), job.Key, "w2:2").Err())
				h.failStop.Store(true)
				err := w.CheckOwnership(t.Context(), job.Key, job.Epoch)
				require.ErrorIs(t, err, ErrOwnershipLost)
				w.stopRejectedJobs()
				require.EqualValues(t, 1, h.stops.Load())
				require.Len(t, w.Jobs(), 1)
				h.failStop.Store(false)
				w.stopRejectedJobs()
				require.Empty(t, w.Jobs())
			}
		})
	}
}

func TestCheckOwnershipZeroWorker(t *testing.T) {
	w := &Worker{}
	require.ErrorIs(t, w.CheckOwnership(t.Context(), "k", 1), ErrOwnershipLost)
}

func TestOwnershipCheckExpiredRedisLease(t *testing.T) {
	rdb := startTestRedis(t)
	n := sinklessNode(t, rdb, "check-expired-redis", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	h := &fencingHandler{}
	w.handler = h
	job := &Job{Key: "k", NodeID: n.ID}
	require.NoError(t, w.startJob(t.Context(), job))
	w.fenceHandlers()
	// Lease expiry alone does not prove that ownership transferred. Resume
	// must retain the paused job until its owner/epoch changes or renews.
	require.NoError(t, rdb.HSet(t.Context(), rmapContentKey(workerKeepAliveMapName(n.PoolName)), w.ID, "1").Err())
	w.updateLeaseState(t.Context(), time.Now())
	require.Len(t, w.Jobs(), 1)
	require.EqualValues(t, 1, h.starts.Load())
	require.NoError(t, n.workerHeartbeat(t.Context(), w.ID))
	w.recordLease(time.Now())
	w.updateLeaseState(t.Context(), time.Now())
	require.NoError(t, w.CheckOwnership(t.Context(), job.Key, job.Epoch))
	require.EqualValues(t, 2, h.starts.Load())
}

type joiningOwnershipHandler struct {
	exited chan struct{}
}

func (h *joiningOwnershipHandler) Start(*Job) error {
	return nil
}

func (h *joiningOwnershipHandler) Stop(string) error {
	<-h.exited
	return nil
}

func TestCheckOwnershipDoesNotJoinCaller(t *testing.T) {
	rdb := startTestRedis(t)
	n := sinklessNode(t, rdb, "check-join", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	h := &joiningOwnershipHandler{exited: make(chan struct{})}
	w.handler = h
	job := &Job{Key: "k", NodeID: n.ID}
	require.NoError(t, w.startJob(t.Context(), job))
	require.NoError(t, rdb.HSet(t.Context(), n.ownersKey(), job.Key, "w2:2").Err())
	var exit sync.Once
	result := make(chan error, 1)
	go func() {
		err := w.CheckOwnership(t.Context(), job.Key, job.Epoch)
		exit.Do(func() {
			close(h.exited)
		})
		result <- err
	}()
	select {
	case err := <-result:
		require.ErrorIs(t, err, ErrOwnershipLost)
		w.stopRejectedJobs()
		require.Empty(t, w.Jobs())
	case <-time.After(time.Second):
		exit.Do(func() {
			close(h.exited)
		})
		<-result
		t.Fatal("ownership check called Stop inline and joined its own caller")
	}
}

func TestCheckOwnershipWakesStop(t *testing.T) {
	rdb := startTestRedis(t)
	n := addTestNode(t, rdb, "check-wake", WithWorkerTTL(8*time.Second))
	h := &fencingHandler{}
	w, err := n.AddWorker(t.Context(), h)
	require.NoError(t, err)
	job := &Job{Key: "k", NodeID: n.ID}
	require.NoError(t, w.startJob(t.Context(), job))
	require.NoError(t, rdb.HSet(t.Context(), n.ownersKey(), job.Key, "w2:2").Err())
	require.ErrorIs(t, w.CheckOwnership(t.Context(), job.Key, job.Epoch), ErrOwnershipLost)
	require.Eventually(t, func() bool {
		return h.stops.Load() == 1 && len(w.Jobs()) == 0
	}, time.Second, time.Millisecond)
}

func TestOwnershipCheckLateRejection(t *testing.T) {
	rdb := startTestRedis(t)
	n := sinklessNode(t, rdb, "check-late", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	h := &fencingHandler{}
	w.handler = h
	first := &Job{Key: "k", NodeID: n.ID}
	require.NoError(t, w.startJob(t.Context(), first))
	require.NoError(t, w.stopJob(t.Context(), first.Key))
	second := &Job{Key: "k", NodeID: n.ID}
	require.NoError(t, w.startJob(t.Context(), second))
	w.rejectedJobs.Store(first, struct{}{})
	w.stopRejectedJobs()
	require.Len(t, w.Jobs(), 1, "old rejection must not stop the replacement")
	require.EqualValues(t, 1, h.stops.Load())
	// A late old reply must not overwrite a newer pending rejection.
	w.rejectedJobs.Store(second, struct{}{})
	w.rejectedJobs.Store(first, struct{}{})
	w.stopRejectedJobs()
	require.Empty(t, w.Jobs())
	require.EqualValues(t, 2, h.stops.Load())
}

type startupOwnershipHandler struct {
	checked chan error
}

func (h *startupOwnershipHandler) Start(job *Job) error {
	// The work loop can run immediately, before Start returns to the pool.
	go func() {
		h.checked <- job.Worker.CheckOwnership(context.Background(), job.Key, job.Epoch)
	}()
	return nil
}

func (h *startupOwnershipHandler) Stop(string) error {
	return nil
}

func TestOwnershipPublishedBeforeStart(t *testing.T) {
	rdb := startTestRedis(t)
	n := sinklessNode(t, rdb, "check-startup", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	h := &startupOwnershipHandler{checked: make(chan error, 1)}
	w.handler = h
	// Block the post-Start log: the work loop runs after the callback has
	// returned, but before the old implementation published local ownership.
	entered, release := make(chan struct{}), make(chan struct{})
	w.logger = &startupOwnershipLogger{Logger: w.logger, entered: entered, release: release}
	done := make(chan error, 1)
	go func() {
		done <- w.startJob(t.Context(), &Job{Key: "k", NodeID: n.ID})
	}()
	<-entered
	defer func() {
		close(release)
		require.NoError(t, <-done)
	}()
	require.NoError(t, <-h.checked)
}

type startupOwnershipLogger struct {
	pulse.Logger
	entered chan struct{}
	release chan struct{}
}

func (l *startupOwnershipLogger) Info(message string, args ...any) {
	if message == "started job" {
		close(l.entered)
		<-l.release
	}
	l.Logger.Info(message, args...)
}

func TestOwnershipRemovedAfterStartFailure(t *testing.T) {
	rdb := startTestRedis(t)
	n := sinklessNode(t, rdb, "check-start-failure", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	h := &fencingHandler{}
	h.failStart.Store(true)
	w.handler = h
	job := &Job{Key: "k", NodeID: n.ID}
	require.ErrorContains(t, w.startJob(t.Context(), job), "start failed")
	require.Empty(t, w.Jobs())
	require.ErrorIs(t, w.CheckOwnership(t.Context(), job.Key, job.Epoch), ErrOwnershipLost)
	owned, err := rdb.HExists(t.Context(), n.ownersKey(), job.Key).Result()
	require.NoError(t, err)
	require.False(t, owned)
}
