package pool

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/loom/pulse/streaming"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type fencingHandler struct {
	starts    atomic.Int32
	stops     atomic.Int32
	epoch     atomic.Uint64
	failStop  atomic.Bool
	failStart atomic.Bool
}

func (h *fencingHandler) Start(job *Job) error {
	if h.failStart.Load() {
		return errors.New("start failed")
	}
	h.starts.Add(1)
	h.epoch.Store(job.Epoch)
	return nil
}

func (h *fencingHandler) Stop(string) error {
	h.stops.Add(1)
	if h.failStop.Load() {
		return errors.New("stop failed")
	}
	return nil
}

func TestWorkerFenceResume(t *testing.T) {
	for _, superseded := range []bool{false, true} {
		t.Run(map[bool]string{false: "same owner", true: "cleanup transferred ownership"}[superseded], func(t *testing.T) {
			rdb := startTestRedis(t)
			n := sinklessNode(t, rdb, "fence-resume", WithWorkerTTL(time.Hour))
			w := ownershipWorker(t, n, "w1")
			h := &fencingHandler{}
			w.handler = h
			job := &Job{Key: "k", Payload: []byte("p"), CreatedAt: time.Now(), NodeID: n.ID}
			require.NoError(t, w.startJob(t.Context(), job))
			base := time.Now()
			w.recordLease(base)
			w.updateLeaseState(t.Context(), base.Add(w.workerTTL))
			require.True(t, w.leaseFenced.Load())
			require.EqualValues(t, 1, h.stops.Load())
			require.Len(t, w.Jobs(), 1, "fencing must retain ownership for possible resume")
			require.ErrorIs(t, w.startJob(t.Context(), &Job{Key: "other"}), ErrRequeue)
			if superseded {
				require.NoError(t, rdb.HSet(t.Context(), rmapContentKey(workerKeepAliveMapName(n.PoolName)), w.ID, "1").Err())
				n.cleanupWorker(t.Context(), w.ID)
				other := ownershipWorker(t, n, "w2")
				require.NoError(t, other.startJob(t.Context(), &Job{Key: "k", Payload: []byte("p"), NodeID: n.ID}))
			}
			w.recordLease(base.Add(w.workerTTL))
			w.updateLeaseState(t.Context(), base.Add(w.workerTTL))
			if superseded {
				require.EqualValues(t, 1, h.starts.Load(), "superseded work must not resume")
				require.Empty(t, w.Jobs())
			} else {
				require.EqualValues(t, 2, h.starts.Load())
				require.Equal(t, job.Epoch, h.epoch.Load())
				require.False(t, w.leaseFenced.Load())
			}
		})
	}
}

func TestWorkerFenceCallbackFailure(t *testing.T) {
	for _, failing := range []string{"stop", "resume"} {
		t.Run(failing, func(t *testing.T) {
			rdb := startTestRedis(t)
			n := sinklessNode(t, rdb, "fence-callback", WithWorkerTTL(time.Hour))
			w := ownershipWorker(t, n, "w1")
			h := &fencingHandler{}
			w.handler = h
			require.NoError(t, w.startJob(t.Context(), &Job{Key: "k", NodeID: n.ID}))
			base := time.Now()
			w.recordLease(base)
			h.failStop.Store(failing == "stop")
			w.updateLeaseState(t.Context(), base.Add(w.workerTTL))
			h.failStart.Store(failing == "resume")
			w.recordLease(base.Add(w.workerTTL))
			w.updateLeaseState(t.Context(), base.Add(w.workerTTL))
			require.True(t, w.leaseFenced.Load())
			require.EqualValues(t, 1, h.starts.Load())
			h.failStop.Store(false)
			h.failStart.Store(false)
			w.updateLeaseState(t.Context(), base.Add(w.workerTTL))
			require.False(t, w.leaseFenced.Load())
			require.EqualValues(t, 2, h.starts.Load())
		})
	}
}

func TestStopFencedJob(t *testing.T) {
	rdb := startTestRedis(t)
	n := sinklessNode(t, rdb, "stop-fenced", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	h := &fencingHandler{}
	w.handler = h
	require.NoError(t, w.startJob(t.Context(), &Job{Key: "k", NodeID: n.ID}))
	base := time.Now()
	w.recordLease(base)
	w.updateLeaseState(t.Context(), base.Add(w.workerTTL))
	require.NoError(t, w.stopJob(t.Context(), "k"))
	require.EqualValues(t, 1, h.stops.Load(), "a fenced handler is already stopped")
	w.recordLease(base.Add(w.workerTTL))
	w.updateLeaseState(t.Context(), base.Add(w.workerTTL))
	require.EqualValues(t, 1, h.starts.Load())
	require.Empty(t, w.Jobs())
}

// TestFenceBeforeCleanup replays redesign_false_death with two live nodes.
// A failed renewal fences locally while the last Redis lease is still live.
func TestFenceBeforeCleanup(t *testing.T) {
	rdb := startTestRedis(t)
	n1 := addTestNode(t, rdb, "fence-before-cleanup", WithWorkerTTL(2*time.Second))
	n2 := addTestNode(t, rdb, "fence-before-cleanup", WithWorkerTTL(2*time.Second))
	h := &fencingHandler{}
	w, err := n1.AddWorker(t.Context(), h)
	require.NoError(t, err)
	job := &Job{Key: "k", NodeID: n1.ID, CreatedAt: time.Now()}
	require.NoError(t, w.startJob(t.Context(), job))
	hook := &fenceHeartbeatHook{worker: w.ID}
	hook.fail.Store(true)
	rdb.AddHook(hook)
	require.Eventually(t, func() bool {
		return h.stops.Load() == 1
	}, 3*time.Second, time.Millisecond)
	require.True(t, w.leaseFenced.Load())
	// Direct cleanup must still refuse the freshly observed Redis lease.
	n2.cleanupWorker(t.Context(), w.ID)
	owner, err := rdb.HGet(t.Context(), n1.ownersKey(), job.Key).Result()
	require.NoError(t, err)
	require.Equal(t, w.ID+":1", owner)
	// A restored renewal and authoritative check resume the same epoch.
	hook.fail.Store(false)
	require.NoError(t, n1.workerHeartbeat(t.Context(), w.ID))
	w.recordLease(time.Now())
	require.Eventually(t, func() bool {
		return h.starts.Load() == 2 && !w.leaseFenced.Load()
	}, time.Second, time.Millisecond)
	require.Equal(t, job.Epoch, h.epoch.Load())
}

func TestLeaseReplyUsesRequestTime(t *testing.T) {
	rdb := startTestRedis(t)
	n := sinklessNode(t, rdb, "delayed-lease-reply", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	w.workerTTL = 200 * time.Millisecond
	original := time.Now().Add(-time.Hour)
	w.recordLease(original)
	hook := &fenceHeartbeatHook{worker: w.ID, entered: make(chan struct{}), release: make(chan struct{})}
	rdb.AddHook(hook)
	w.wg.Add(1)
	go w.keepAlive(t.Context())
	select {
	case <-hook.entered:
	case <-time.After(3 * time.Second):
		close(hook.release)
		t.Fatal("heartbeat did not reach Redis")
	}
	beforeReply := time.Now()
	hook.fail.Store(true)
	close(hook.release)
	var recorded time.Time
	require.Eventually(t, func() bool {
		w.leaseLock.Lock()
		recorded = w.leaseStart
		w.leaseLock.Unlock()
		return recorded != original
	}, time.Second, time.Millisecond)
	require.True(t, recorded.Before(beforeReply), "renewal must use request start, not reply time")
	require.False(t, w.leaseExpired(recorded.Add(149*time.Millisecond)))
	require.True(t, w.leaseExpired(recorded.Add(150*time.Millisecond)))
}

type fenceHeartbeatHook struct {
	worker  string
	fail    atomic.Bool
	delayed atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (h *fenceHeartbeatHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *fenceHeartbeatHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *fenceHeartbeatHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		args := cmd.Args()
		if len(args) == 7 && args[6] == h.worker &&
			((cmd.Name() == "evalsha" && args[1] == luaOwnerHeartbeat.Hash()) ||
				(cmd.Name() == "eval" && strings.Contains(fmt.Sprint(args[1]), "worker is no longer registered"))) {
			if h.fail.Load() {
				return errors.New("injected heartbeat failure")
			}
			if h.entered != nil {
				if err := next(ctx, cmd); err != nil {
					return err
				}
				if h.delayed.CompareAndSwap(false, true) {
					close(h.entered)
					<-h.release
				}
				return nil
			}
		}
		return next(ctx, cmd)
	}
}

func TestFencedWorkerEvents(t *testing.T) {
	rdb := startTestRedis(t)
	n := sinklessNode(t, rdb, "fenced-events", WithWorkerTTL(time.Hour))
	w := ownershipWorker(t, n, "w1")
	h := &fencingHandler{}
	w.handler = h
	require.NoError(t, w.startJob(t.Context(), &Job{Key: "k", NodeID: n.ID}))
	w.updateLeaseState(t.Context(), time.Now().Add(time.Hour))
	for _, event := range []string{evStartJob, evStopJob, evNotify} {
		t.Run(event, func(t *testing.T) {
			require.ErrorIs(t, w.handleEvent(t.Context(), &streaming.Event{EventName: event}, nil), ErrRequeue)
		})
	}
	require.ErrorIs(t, w.notify(t.Context(), "k", nil), ErrRequeue)
	require.EqualValues(t, 1, h.starts.Load())
	require.EqualValues(t, 1, h.stops.Load())
	require.Len(t, w.Jobs(), 1)
}

func TestFenceDuringOwnershipIO(t *testing.T) {
	for _, operation := range []string{"claim", "release", "resume"} {
		t.Run(operation, func(t *testing.T) {
			rdb := startTestRedis(t)
			n := sinklessNode(t, rdb, "fence-blocked-"+operation, WithWorkerTTL(time.Hour))
			w := ownershipWorker(t, n, "w1")
			h := &fencingHandler{}
			w.handler = h
			require.NoError(t, w.startJob(t.Context(), &Job{Key: "running", NodeID: n.ID}))
			script := luaClaimJob
			if operation != "claim" {
				require.NoError(t, w.startJob(t.Context(), &Job{Key: "other", NodeID: n.ID}))
				script = luaReleaseJob
			}
			if operation == "resume" {
				w.handlerLock.Lock()
				require.NoError(t, h.Stop("other"))
				w.fencedJobs = map[string]bool{"other": true}
				w.handlerLock.Unlock()
				w.leaseFenced.Store(true)
				script = luaCheckOwner
			}
			hook := &fenceOwnershipHook{hash: script.Hash(), entered: make(chan struct{}), release: make(chan struct{})}
			rdb.AddHook(hook)
			ioDone := make(chan error, 1)
			go func() {
				switch operation {
				case "claim":
					ioDone <- w.startJob(t.Context(), &Job{Key: "other", NodeID: n.ID})
				case "release":
					ioDone <- w.stopJob(t.Context(), "other")
				case "resume":
					w.updateLeaseState(t.Context(), time.Now())
					ioDone <- nil
				}
			}()
			<-hook.entered
			fenceDone := make(chan struct{})
			defer func() {
				close(hook.release)
				err := <-ioDone
				if operation == "claim" {
					require.ErrorIs(t, err, ErrRequeue)
				} else {
					require.NoError(t, err)
				}
				<-fenceDone
			}()
			before := h.stops.Load()
			w.recordLease(time.Now().Add(-time.Hour))
			go func() {
				w.updateLeaseState(t.Context(), time.Now())
				close(fenceDone)
			}()
			require.Eventually(t, func() bool {
				return h.stops.Load() > before
			}, time.Second, time.Millisecond, "Redis I/O must not block stopping the running job")
			require.True(t, w.leaseFenced.Load())
			// Transfer only after the running handler has stopped, while the
			// original ownership request is still blocked.
			require.NoError(t, rdb.HSet(t.Context(), rmapContentKey(workerKeepAliveMapName(n.PoolName)), w.ID, "1").Err())
			n.cleanupWorker(t.Context(), w.ID)
			other := ownershipWorker(t, n, "w2")
			require.NoError(t, other.startJob(t.Context(), &Job{Key: "running", NodeID: n.ID}))
			require.Len(t, other.Jobs(), 1)
		})
	}
}

type fenceOwnershipHook struct {
	hash    string
	entered chan struct{}
	release chan struct{}
	fired   atomic.Bool
}

func (h *fenceOwnershipHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *fenceOwnershipHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *fenceOwnershipHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		args := cmd.Args()
		if cmd.Name() == "evalsha" && len(args) > 1 && args[1] == h.hash && h.fired.CompareAndSwap(false, true) {
			close(h.entered)
			<-h.release
		}
		return next(ctx, cmd)
	}
}
