package pool

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/rmap"
)

// luaStructShim is a pure-Lua replacement for the Redis "struct" library used
// by the rmap and pool coordination scripts. miniredis does not ship the
// struct library so the test client rewrites scripts to prepend this shim.
// Only the struct.pack "i" (4-byte little-endian length) and "c0" (raw string)
// format codes used by the production scripts are implemented.
const luaStructShim = `
local struct = { pack = function(fmt, ...)
   local args = {...}
   local out = {}
   local ai = 1
   local i = 1
   while i <= string.len(fmt) do
      local c = string.sub(fmt, i, i)
      if c == "i" then
         local n = args[ai]
         ai = ai + 1
         out[#out+1] = string.char(n % 256, math.floor(n / 256) % 256, math.floor(n / 65536) % 256, math.floor(n / 16777216) % 256)
      elseif c == "c" then
         while i < string.len(fmt) and string.match(string.sub(fmt, i + 1, i + 1), "%d") do
            i = i + 1
         end
         out[#out+1] = args[ai]
         ai = ai + 1
      end
      i = i + 1
   end
   return table.concat(out)
end }
`

type (
	// structShimHook rewrites EVAL and SCRIPT LOAD commands on their way to
	// miniredis so scripts that use the Redis struct library keep working.
	structShimHook struct{}

	// recordingHandler is a JobHandler that records starts, stops and
	// notifications.
	recordingHandler struct {
		lock     sync.Mutex
		started  map[string][]byte
		stopped  map[string]bool
		notified map[string][]byte
	}
)

func (structShimHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (structShimHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		shimStructScript(cmd)
		return next(ctx, cmd)
	}
}

func (structShimHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			shimStructScript(cmd)
		}
		return next(ctx, cmds)
	}
}

// shimStructScript prepends the struct shim to Lua sources sent via EVAL or
// SCRIPT LOAD. EVALSHA calls issued for the original source fail with NOSCRIPT
// and go-redis falls back to EVAL, which is then rewritten here.
func shimStructScript(cmd redis.Cmder) {
	args := cmd.Args()
	switch cmd.Name() {
	case "eval":
		if len(args) > 1 {
			if src, ok := args[1].(string); ok && strings.Contains(src, "struct.pack") {
				args[1] = luaStructShim + src
			}
		}
	case "script":
		if len(args) > 2 {
			if sub, ok := args[1].(string); ok && strings.EqualFold(sub, "load") {
				if src, ok := args[2].(string); ok && strings.Contains(src, "struct.pack") {
					args[2] = luaStructShim + src
				}
			}
		}
	}
}

func newRecordingHandler() *recordingHandler {
	return &recordingHandler{
		started:  make(map[string][]byte),
		stopped:  make(map[string]bool),
		notified: make(map[string][]byte),
	}
}

// Start implements JobHandler.
func (h *recordingHandler) Start(job *Job) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.started[job.Key] = job.Payload
	return nil
}

// Stop implements JobHandler.
func (h *recordingHandler) Stop(key string) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.stopped[key] = true
	return nil
}

// HandleNotification implements NotificationHandler.
func (h *recordingHandler) HandleNotification(key string, payload []byte) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.notified[key] = payload
	return nil
}

func (h *recordingHandler) startedPayload(key string) ([]byte, bool) {
	h.lock.Lock()
	defer h.lock.Unlock()
	payload, ok := h.started[key]
	return payload, ok
}

func (h *recordingHandler) wasStopped(key string) bool {
	h.lock.Lock()
	defer h.lock.Unlock()
	return h.stopped[key]
}

func (h *recordingHandler) notifiedPayload(key string) ([]byte, bool) {
	h.lock.Lock()
	defer h.lock.Unlock()
	payload, ok := h.notified[key]
	return payload, ok
}

// startTestRedis runs an in-process miniredis server and returns a client
// whose Lua scripts are rewritten to work around the missing struct library.
func startTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rdb.AddHook(structShimHook{})
	t.Cleanup(func() {
		require.NoError(t, rdb.Close())
	})
	return rdb
}

// addTestNode creates a pool node with fast timeouts and closes it on cleanup.
func addTestNode(t *testing.T, rdb *redis.Client, pool string, opts ...NodeOption) *Node {
	t.Helper()
	opts = append([]NodeOption{
		WithWorkerTTL(time.Second),
		WithWorkerShutdownTTL(5 * time.Second),
		WithJobSinkBlockDuration(50 * time.Millisecond),
		WithAckGracePeriod(5 * time.Second),
	}, opts...)
	node, err := AddNode(t.Context(), pool, rdb, opts...)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, node.Close(context.Background()))
	})
	return node
}

func TestDispatchJobLifecycle(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, "lifecycle")
	handler := newRecordingHandler()
	worker, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)

	require.NoError(t, node.DispatchJob(ctx, "job-1", []byte("payload-1")))
	require.Eventually(t, func() bool {
		payload, ok := handler.startedPayload("job-1")
		return ok && string(payload) == "payload-1"
	}, 10*time.Second, 5*time.Millisecond)

	require.Equal(t, []string{"job-1"}, node.JobKeys())
	payload, ok := node.JobPayload("job-1")
	require.True(t, ok)
	require.Equal(t, []byte("payload-1"), payload)
	_, ok = node.JobPayload("missing")
	require.False(t, ok)

	// A second dispatch of the same key is rejected.
	require.ErrorIs(t, node.DispatchJob(ctx, "job-1", []byte("payload-1")), ErrJobExists)

	// The worker reports the running job.
	jobs := worker.Jobs()
	require.Len(t, jobs, 1)
	require.Equal(t, "job-1", jobs[0].Key)
	require.Equal(t, node.ID, jobs[0].NodeID)

	require.Len(t, node.Workers(), 1)
	require.Len(t, node.PoolWorkers(), 1)

	require.NoError(t, node.StopJob(ctx, "job-1"))
	require.Eventually(t, func() bool {
		return handler.wasStopped("job-1")
	}, 10*time.Second, 5*time.Millisecond)
	require.Eventually(t, func() bool {
		return len(node.JobKeys()) == 0
	}, 10*time.Second, 5*time.Millisecond)
}

func TestDispatchJobValidation(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, "validation")

	require.ErrorContains(t, node.DispatchJob(ctx, "", []byte("p")), "job key cannot be empty")
	require.ErrorContains(t, node.DispatchJob(ctx, "a=b", []byte("p")), "cannot contain '='")

	require.NoError(t, node.Close(ctx))
	require.ErrorContains(t, node.DispatchJob(ctx, "job", []byte("p")), "is closed")
	require.ErrorContains(t, node.StopJob(ctx, "job"), "is closed")
	require.ErrorContains(t, node.NotifyWorker(ctx, "job", []byte("p")), "is closed")
	_, err := node.AddWorker(ctx, newRecordingHandler())
	require.ErrorContains(t, err, "is closed")
}

func TestClientOnlyNodeDispatches(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	workerNode := addTestNode(t, rdb, "clientonly")
	handler := newRecordingHandler()
	_, err := workerNode.AddWorker(ctx, handler)
	require.NoError(t, err)

	clientNode := addTestNode(t, rdb, "clientonly", WithClientOnly())

	_, err = clientNode.AddWorker(ctx, newRecordingHandler())
	require.ErrorContains(t, err, "client-only")
	_, err = clientNode.NewTicker(ctx, "tick", time.Second)
	require.ErrorContains(t, err, "client-only")
	require.ErrorContains(t, clientNode.Shutdown(ctx), "client-only")

	require.NoError(t, clientNode.DispatchJob(ctx, "remote-job", []byte("remote")))
	require.Eventually(t, func() bool {
		payload, ok := handler.startedPayload("remote-job")
		return ok && string(payload) == "remote"
	}, 10*time.Second, 5*time.Millisecond)
}

func TestNotifyWorker(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, "notify")
	handler := newRecordingHandler()
	_, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)

	require.NoError(t, node.DispatchJob(ctx, "job-n", []byte("payload")))
	require.Eventually(t, func() bool {
		_, ok := handler.startedPayload("job-n")
		return ok
	}, 10*time.Second, 5*time.Millisecond)

	require.NoError(t, node.NotifyWorker(ctx, "job-n", []byte("ping")))
	require.Eventually(t, func() bool {
		payload, ok := handler.notifiedPayload("job-n")
		return ok && string(payload) == "ping"
	}, 10*time.Second, 5*time.Millisecond)
}

func TestRemoveWorkerRequeuesJobs(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, "requeue-worker")
	first := newRecordingHandler()
	worker, err := node.AddWorker(ctx, first)
	require.NoError(t, err)

	require.NoError(t, node.DispatchJob(ctx, "sticky-job", []byte("payload")))
	require.Eventually(t, func() bool {
		_, ok := first.startedPayload("sticky-job")
		return ok
	}, 10*time.Second, 5*time.Millisecond)

	// Add a second worker to pick up the requeued job, then remove the first.
	second := newRecordingHandler()
	_, err = node.AddWorker(ctx, second)
	require.NoError(t, err)
	require.NoError(t, node.RemoveWorker(ctx, worker))

	require.Eventually(t, func() bool {
		payload, ok := second.startedPayload("sticky-job")
		return ok && string(payload) == "payload"
	}, 10*time.Second, 5*time.Millisecond)
	require.True(t, first.wasStopped("sticky-job"))
}

func TestNodeCloseRequeuesJobsToOtherNode(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	nodeA := addTestNode(t, rdb, "requeue-node")
	handlerA := newRecordingHandler()
	_, err := nodeA.AddWorker(ctx, handlerA)
	require.NoError(t, err)

	require.NoError(t, nodeA.DispatchJob(ctx, "movable-job", []byte("payload")))
	require.Eventually(t, func() bool {
		_, ok := handlerA.startedPayload("movable-job")
		return ok
	}, 10*time.Second, 5*time.Millisecond)

	nodeB := addTestNode(t, rdb, "requeue-node")
	handlerB := newRecordingHandler()
	_, err = nodeB.AddWorker(ctx, handlerB)
	require.NoError(t, err)

	require.NoError(t, nodeA.Close(ctx))
	require.True(t, nodeA.IsClosed())

	require.Eventually(t, func() bool {
		payload, ok := handlerB.startedPayload("movable-job")
		return ok && string(payload) == "payload"
	}, 15*time.Second, 5*time.Millisecond)
}

func TestTickerDeliversTicks(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, "ticker")

	ticker, err := node.NewTicker(ctx, "beat", 30*time.Millisecond)
	require.NoError(t, err)

	var (
		lock  sync.Mutex
		count int
	)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				lock.Lock()
				count++
				lock.Unlock()
			case <-done:
				return
			}
		}
	}()
	require.Eventually(t, func() bool {
		lock.Lock()
		defer lock.Unlock()
		return count >= 2
	}, 10*time.Second, 5*time.Millisecond)
	close(done)
	ticker.Close()

	// Stop deletes the shared ticker map entry.
	stopper, err := node.NewTicker(ctx, "beat2", 30*time.Millisecond)
	require.NoError(t, err)
	stopper.Stop()
	require.Eventually(t, func() bool {
		_, ok := node.tickerMap.Get("ticker:beat2")
		return !ok
	}, 10*time.Second, 5*time.Millisecond)
}

func TestScheduleRunsAndStops(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	node := addTestNode(t, rdb, "schedule")
	handler := newRecordingHandler()
	_, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)

	producer := &scheduleOnceProducer{}
	require.NoError(t, node.Schedule(ctx, producer, 30*time.Millisecond))

	// A second schedule of the same producer on the same node is rejected.
	require.ErrorContains(t, node.Schedule(ctx, &scheduleOnceProducer{}, 30*time.Millisecond), "already scheduled")

	require.Eventually(t, func() bool {
		payload, ok := handler.startedPayload("scheduled-job")
		return ok && string(payload) == "scheduled"
	}, 10*time.Second, 5*time.Millisecond)

	// The producer returns ErrScheduleStop on the next plan which stops the
	// schedule globally and unregisters the local scheduler.
	require.Eventually(t, func() bool {
		_, ok := node.localSchedulers.Load("schedule:once")
		return !ok
	}, 10*time.Second, 5*time.Millisecond)
}

func TestShutdownPropagatesAcrossNodes(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	nodeA := addTestNode(t, rdb, "shutdown")
	nodeB := addTestNode(t, rdb, "shutdown")
	handler := newRecordingHandler()
	_, err := nodeB.AddWorker(ctx, handler)
	require.NoError(t, err)

	require.NoError(t, nodeB.Shutdown(ctx))
	require.True(t, nodeB.IsClosed())
	require.True(t, nodeB.IsShutdown())

	// The other node observes the shutdown map update and closes itself.
	require.Eventually(t, nodeA.IsClosed, 15*time.Second, 5*time.Millisecond)

	// Shutdown is idempotent once the node is closed.
	require.NoError(t, nodeB.Shutdown(ctx))
}

func TestAddNodeRejectsShuttingDownPool(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()

	// Mark the pool as shutting down before any node joins.
	shutdownMap, err := rmap.Join(ctx, nodeShutdownMapName("rejected"), rdb)
	require.NoError(t, err)
	t.Cleanup(shutdownMap.Close)
	_, err = shutdownMap.SetAndWait(ctx, "shutdown", "some-node")
	require.NoError(t, err)

	_, err = AddNode(ctx, "rejected", rdb)
	require.ErrorContains(t, err, "is shutting down")
}

type scheduleOnceProducer struct {
	lock  sync.Mutex
	plans int
}

// Name implements JobProducer.
func (p *scheduleOnceProducer) Name() string {
	return "once"
}

// Plan starts one job on the first tick and stops the schedule on the next.
func (p *scheduleOnceProducer) Plan() (*JobPlan, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.plans++
	if p.plans == 1 {
		return &JobPlan{Start: []*JobParam{{Key: "scheduled-job", Payload: []byte("scheduled")}}}, nil
	}
	return nil, ErrScheduleStop
}
