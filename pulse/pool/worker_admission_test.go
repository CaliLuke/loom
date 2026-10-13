package pool

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type constructionFailureHook struct {
	match     func(redis.Cmder) bool
	after     bool
	committed func(redis.Cmder)
	succeed   bool
}

var errConstructionInjected = errors.New("injected construction failure")

func TestConstructionWorkerIsNotPublishedBeforeReady(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "construction-worker", WithWorkerTTL(5*time.Second))
	ready, err := node.AddWorker(t.Context(), newRecordingHandler())
	require.NoError(t, err)
	var observedWorkers, observedHeartbeats map[string]string
	var workersErr, heartbeatsErr error
	var snapshot sync.Mutex
	rdb.AddHook(constructionFailureHook{match: func(cmd redis.Cmder) bool {
		args := cmd.Args()
		if cmd.Name() != "xadd" || len(args) < 2 || !strings.Contains(fmt.Sprint(args[1]), ":stream:worker:") {
			return false
		}
		// Observe from Redis before the failed worker can acquire its stream or reader.
		snapshot.Lock()
		defer snapshot.Unlock()
		observedWorkers, workersErr = rdb.HGetAll(t.Context(), rmapContentKey(workerMapName(node.PoolName))).Result()
		observedHeartbeats, heartbeatsErr = rdb.HGetAll(t.Context(), rmapContentKey(workerKeepAliveMapName(node.PoolName))).Result()
		return true
	}})
	worker, err := node.AddWorker(t.Context(), newRecordingHandler())
	require.ErrorIs(t, err, errConstructionInjected)
	require.Nil(t, worker)
	snapshot.Lock()
	defer snapshot.Unlock()
	require.NoError(t, workersErr)
	require.NoError(t, heartbeatsErr)
	require.NotEmpty(t, observedWorkers)
	for key := range observedWorkers {
		if strings.HasPrefix(key, "=") {
			delete(observedWorkers, key)
		}
	}
	for key := range observedHeartbeats {
		if strings.HasPrefix(key, "=") {
			delete(observedHeartbeats, key)
		}
	}
	assert.Equal(t, map[string]string{ready.ID: observedWorkers[ready.ID]}, observedWorkers,
		"worker was published before readiness and before failure rollback could run")
	assert.Equal(t, map[string]string{ready.ID: observedHeartbeats[ready.ID]}, observedHeartbeats)
	assert.Len(t, node.Workers(), 1)
	assert.Len(t, node.PoolWorkers(), 1)
}

// A successful Redis write followed by a transport error is distinct from a
// rejected write. The caller gets no worker handle in both cases.

func TestConstructionWorkerLostHeartbeatReply(t *testing.T) {
	for _, cancelCaller := range []bool{false, true} {
		t.Run(fmt.Sprintf("caller-canceled=%t", cancelCaller), func(t *testing.T) {
			rdb := startTestRedis(t)
			node := addTestNode(t, rdb, "construction-uncertain", WithWorkerTTL(5*time.Second))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			handler := newRecordingHandler()
			committedID := ""
			rdb.AddHook(constructionFailureHook{after: true, committed: func(cmd redis.Cmder) {
				committedID = fmt.Sprint(cmd.Args()[6])
				if cancelCaller {
					cancel()
				}
			}, match: func(cmd redis.Cmder) bool {
				args := cmd.Args()
				return len(args) == 7 && ((cmd.Name() == "evalsha" && args[1] == luaOwnerHeartbeat.Hash()) ||
					(cmd.Name() == "eval" && strings.Contains(fmt.Sprint(args[1]), "worker is no longer registered")))
			}})
			worker, err := node.AddWorker(ctx, handler)
			require.ErrorIs(t, err, errConstructionInjected)
			require.Nil(t, worker)
			members, err := rdb.HGetAll(t.Context(), rmapContentKey(workerMapName(node.PoolName))).Result()
			require.NoError(t, err)
			heartbeats, err := rdb.HGetAll(t.Context(), rmapContentKey(workerKeepAliveMapName(node.PoolName))).Result()
			require.NoError(t, err)
			require.NotEmpty(t, committedID, "must inject after committed heartbeat")
			assert.NotContains(t, members, committedID, "failed admission retained membership")
			assert.NotContains(t, heartbeats, committedID, "failed admission retained heartbeat")
			assert.Empty(t, node.Workers())

			handler.lock.Lock()
			assert.Empty(t, handler.started, "failed admission invoked handler")
			handler.lock.Unlock()
		})
	}
}

func TestConstructionWorkerCannotOutliveNodeClose(t *testing.T) {
	rdb := startTestRedis(t)
	node := addTestNode(t, rdb, "construction-close", WithWorkerTTL(5*time.Second))
	entered, release := make(chan struct{}), make(chan struct{})
	var blockOnce, releaseOnce sync.Once
	unblock := func() {
		releaseOnce.Do(func() {
			close(release)
		})
	}
	defer unblock()
	rdb.AddHook(constructionFailureHook{match: func(cmd redis.Cmder) bool {
		args := cmd.Args()
		if cmd.Name() == "xadd" && len(args) > 1 && strings.Contains(fmt.Sprint(args[1]), ":stream:worker:") {
			blockOnce.Do(func() {
				close(entered)
				<-release
			})
		}
		return false
	}})
	type result struct {
		worker *Worker
		err    error
	}
	results := make(chan result, 1)
	go func() {
		worker, err := node.AddWorker(t.Context(), newRecordingHandler())
		results <- result{worker, err}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("worker construction did not reach stream preparation")
	}
	closed := make(chan error, 1)
	go func() {
		closed <- node.Close(t.Context())
	}()
	// Observe the actual admission fence before releasing the constructor.
	require.Eventually(t, node.IsClosed, 5*time.Second, time.Millisecond,
		"node close did not publish the closing fence")
	unblock()
	select {
	case err := <-closed:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("node close did not join construction")
	}

	select {
	case got := <-results:
		if got.worker != nil {
			defer got.worker.stop(context.Background())
			got.worker.lock.Lock()
			stopped := got.worker.stopped
			got.worker.lock.Unlock()
			assert.True(t, stopped, "a returned worker must be stopped when concurrent Close completes")
		} else {
			require.Error(t, got.err)
		}
		assert.Empty(t, node.Workers(), "Close must own or exclude concurrent construction")
	case <-time.After(5 * time.Second):
		t.Fatal("worker construction did not return")
	}
}

// TestWorkerAdmissionRecovery checks the existing route/recovery contract that
// permits private stream withdrawal while peer membership replicas lag.
func TestWorkerAdmissionRecovery(t *testing.T) {
	rdb := startTestRedis(t)
	node := sinklessNode(t, rdb, "admission-recovery", WithWorkerTTL(5*time.Second))
	ctx := t.Context()
	addFakeWorker(t, node, "failed-attempt")
	require.NoError(t, rdb.XGroupCreate(ctx, node.poolStream.Key(), poolSinkName, "$").Err())
	job := &Job{Key: "recover", Payload: []byte("durable"), NodeID: node.ID, CreatedAt: time.Now()}
	id, err := node.poolStream.Add(ctx, evStartJob, marshalTestOperation(t, node, job))
	require.NoError(t, err)
	batch, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{Group: poolSinkName, Consumer: "first", Streams: []string{node.poolStream.Key(), ">"}, Count: 1, Block: -1}).Result()
	require.NoError(t, err)
	require.Len(t, batch, 1)
	require.Equal(t, id, batch[0].Messages[0].ID)
	ev := &streaming.Event{ID: id, EventName: evStartJob, Payload: marshalTestOperation(t, node, job)}
	require.NoError(t, node.routeWorkerEvent(ctx, ev))
	failedStream, err := node.getWorkerStream("failed-attempt")
	require.NoError(t, err)
	require.NoError(t, failedStream.Destroy(ctx))
	require.ErrorContains(t, node.routeWorkerEvent(ctx, ev), "does not exist")
	pending, err := rdb.XPending(ctx, node.poolStream.Key(), poolSinkName).Result()
	require.NoError(t, err)
	require.EqualValues(t, 1, pending.Count)
	recovered, _, err := rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{Stream: node.poolStream.Key(), Group: poolSinkName, Consumer: "recovery", Start: "0-0", MinIdle: 0, Count: 1}).Result()
	require.NoError(t, err)
	require.Len(t, recovered, 1)
	require.Equal(t, id, recovered[0].ID)
	node.removeWorker(ctx, "failed-attempt")
	handler := newRecordingHandler()
	healthy, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		ids := node.activeWorkers()
		return len(ids) == 1 && ids[0] == healthy.ID
	}, time.Second, time.Millisecond)
	require.NoError(t, node.routeWorkerEvent(ctx, ev))
	require.Eventually(t, func() bool {
		payload, ok := handler.startedPayload(job.Key)
		return ok && string(payload) == "durable"
	}, time.Second, time.Millisecond)
	require.Eventually(t, func() bool {
		remaining, readErr := rdb.XPending(ctx, node.poolStream.Key(), poolSinkName).Result()
		return readErr == nil && remaining.Count == 0
	}, time.Second, time.Millisecond)
}

// TestWorkerAdmissionUnstartedReader verifies the constructor-abort lifetime
// premise independently of cancellation of an already blocked native read.
func TestWorkerAdmissionUnstartedReader(t *testing.T) {
	rdb := startTestRedis(t)
	stream, err := streaming.NewStream("admission-unstarted", rdb)
	require.NoError(t, err)
	reader, err := stream.NewReader(t.Context(), options.WithReaderStartAtOldest(), options.WithReaderBlockDuration(0))
	require.NoError(t, err)
	closed := make(chan struct{})
	go func() {
		reader.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("unstarted reader close blocked")
	}
	require.True(t, reader.IsClosed())
	_, open := <-reader.Subscribe()
	require.False(t, open)
}

func (h constructionFailureHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h constructionFailureHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h constructionFailureHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if h.match(cmd) {
			if h.after {
				if err := next(ctx, cmd); err != nil {
					return err
				}
				if h.committed != nil {
					h.committed(cmd)
				}
			}
			if h.succeed {
				return nil
			}
			return errConstructionInjected
		}
		return next(ctx, cmd)
	}
}
