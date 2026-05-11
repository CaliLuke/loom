package pool

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"hash/crc64"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/oklog/ulid/v2"
	redis "github.com/redis/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
	"github.com/CaliLuke/loom/pulse/streaming"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type (
	// Node is a pool of workers.
	Node struct {
		ID                 string
		PoolName           string
		poolStream         *streaming.Stream // pool event stream for dispatching jobs
		poolSink           *streaming.Sink   // pool event sink
		nodeStream         *streaming.Stream // node event stream for receiving worker events
		nodeReader         *streaming.Reader // node event reader
		nodeKeepAliveMap   *rmap.Map         // node keep-alive timestamps indexed by ID
		nodeShutdownMap    *rmap.Map         // key is node ID that requested shutdown
		workerMap          *rmap.Map         // worker creation times by ID
		workerKeepAliveMap *rmap.Map         // worker keep-alive timestamps indexed by ID
		workerCleanupMap   *rmap.Map         // key is stale worker ID that needs cleanup
		jobMap             *rmap.Map         // jobs by worker ID
		jobPendingMap      *rmap.Map         // pending jobs by job key
		jobPayloadMap      *rmap.Map         // job payloads by job key
		tickerMap          *rmap.Map         // ticker next tick time indexed by name
		workerTTL          time.Duration     // Worker considered dead if keep-alive not updated after this duration
		workerShutdownTTL  time.Duration     // Worker considered dead if not shutdown after this duration
		ackGracePeriod     time.Duration     // Wait for return status up to this duration
		clientOnly         bool
		logger             pulse.Logger
		h                  hasher
		stop               chan struct{}  // closed when node is stopped
		closed             chan struct{}  // closed when node is closed
		wg                 sync.WaitGroup // allows to wait until all goroutines exit
		rdb                *redis.Client

		localWorkers       sync.Map // workers created by this node
		workerStreams      sync.Map // worker streams indexed by ID
		nodeStreams        sync.Map // streams for worker acks indexed by ID
		pendingJobChannels sync.Map // channels used to send DispatchJob results, nil if event is requeued
		pendingEvents      sync.Map // pending events indexed by sender and event IDs
		orphanedPayloads   sync.Map // job key -> first time observed orphaned payload (unix nanos)

		lock     sync.RWMutex
		closing  bool
		shutdown bool
	}

	// hasher is the interface implemented by types that can hash keys.
	hasher interface {
		Hash(key string, numBuckets int64) int64
	}

	// jumpHash implement Jump Consistent Hash.
	jumpHash struct {
		mu sync.Mutex
		h  hash.Hash64
	}
)

const (
	// evInit is the event used to initialize a node or worker stream.
	evInit string = "i"
	// evStartJob is the event used to send new job to workers.
	evStartJob string = "j"
	// evNotify is the event used to notify a worker running a specific job.
	evNotify string = "n"
	// evStopJob is the event used to stop a job.
	evStopJob string = "s"
	// evAck is the worker event used to ack a pool event.
	evAck string = "a"
	// evDispatchReturn is the event used to forward the worker start return
	// status to the node that dispatched the job.
	evDispatchReturn string = "d"
)

// pendingEventTTL is the TTL for pending events.
var pendingEventTTL = 2 * time.Minute

// ErrJobExists is returned when attempting to dispatch a job with a key that already exists.
var ErrJobExists = errors.New("job already exists")

// AddNode adds a new node to the pool with the given name and returns it. The
// node can be used to dispatch jobs and add new workers. A node also routes
// dispatched jobs to the proper worker and acks the corresponding events once
// the worker acks the job.
//
// The options WithClientOnly can be used to create a node that can only be used
// to dispatch jobs. Such a node does not route or process jobs in the
// background.
//
//nolint:maintidx // Copied runtime constructor remains cohesive around node setup.
func AddNode(ctx context.Context, poolName string, rdb *redis.Client, opts ...NodeOption) (*Node, error) {
	o := parseOptions(opts...)
	logger := o.logger
	nodeID := ulid.Make().String()
	if logger == nil {
		logger = pulse.NoopLogger()
	} else {
		logger = logger.WithPrefix("pool", poolName, "node", nodeID)
	}
	logger.Info("options",
		"client_only", o.clientOnly,
		"max_queued_jobs", o.maxQueuedJobs,
		"worker_ttl", o.workerTTL,
		"worker_shutdown_ttl", o.workerShutdownTTL,
		"ack_grace_period", o.ackGracePeriod)

	nsm, err := rmap.Join(ctx, nodeShutdownMapName(poolName), rdb, rmap.WithLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("AddNode: failed to join shutdown replicated map %q: %w", nodeShutdownMapName(poolName), err)
	}
	if nsm.Len() > 0 {
		return nil, fmt.Errorf("AddNode: pool %q is shutting down", poolName)
	}

	nkm, err := rmap.Join(ctx, nodeKeepAliveMapName(poolName), rdb, rmap.WithLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("AddNode: failed to join node keep-alive map %q: %w", nodeKeepAliveMapName(poolName), err)
	}
	if _, err := nkm.Set(ctx, nodeID, strconv.FormatInt(time.Now().UnixNano(), 10)); err != nil {
		return nil, fmt.Errorf("AddNode: failed to set initial node keep-alive: %w", err)
	}

	poolStream, err := streaming.NewStream(poolStreamName(poolName), rdb,
		options.WithStreamMaxLen(o.maxQueuedJobs),
		options.WithStreamLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("AddNode: failed to create pool job stream %q: %w", poolStreamName(poolName), err)
	}

	var (
		wm   *rmap.Map
		jm   *rmap.Map
		jpm  *rmap.Map
		jpem *rmap.Map
		wkm  *rmap.Map
		tm   *rmap.Map
		wcm  *rmap.Map

		poolSink   *streaming.Sink
		nodeStream *streaming.Stream
		nodeReader *streaming.Reader
		closed     chan struct{}
	)

	if !o.clientOnly {
		wm, err = rmap.Join(ctx, workerMapName(poolName), rdb, rmap.WithLogger(logger))
		if err != nil {
			return nil, fmt.Errorf("AddNode: failed to join pool workers replicated map %q: %w", workerMapName(poolName), err)
		}
		workerIDs := wm.Keys()
		logger.Info("joined", "workers", workerIDs)

		jm, err = rmap.Join(ctx, jobMapName(poolName), rdb, rmap.WithLogger(logger))
		if err != nil {
			return nil, fmt.Errorf("AddNode: failed to join pool jobs replicated map %q: %w", jobMapName(poolName), err)
		}

		jpm, err = rmap.Join(ctx, jobPayloadMapName(poolName), rdb, rmap.WithLogger(logger))
		if err != nil {
			return nil, fmt.Errorf("AddNode: failed to join pool job payloads replicated map %q: %w", jobPayloadMapName(poolName), err)
		}

		wkm, err = rmap.Join(ctx, workerKeepAliveMapName(poolName), rdb, rmap.WithLogger(logger))
		if err != nil {
			return nil, fmt.Errorf("AddNode: failed to join worker keep-alive replicated map %q: %w", workerKeepAliveMapName(poolName), err)
		}

		tm, err = rmap.Join(ctx, tickerMapName(poolName), rdb, rmap.WithLogger(logger))
		if err != nil {
			return nil, fmt.Errorf("AddNode: failed to join pool ticker replicated map %q: %w", tickerMapName(poolName), err)
		}

		wcm, err = rmap.Join(ctx, workerCleanupMapName(poolName), rdb, rmap.WithLogger(logger))
		if err != nil {
			return nil, fmt.Errorf("AddNode: failed to join pool cleanup replicated map %q: %w", workerCleanupMapName(poolName), err)
		}

		// Initialize and join pending jobs map
		jpem, err = rmap.Join(ctx, jobPendingMapName(poolName), rdb, rmap.WithLogger(logger))
		if err != nil {
			return nil, fmt.Errorf("AddNode: failed to join pending jobs replicated map %q: %w", jobPendingMapName(poolName), err)
		}

		poolSink, err = poolStream.NewSink(ctx, "events",
			options.WithSinkBlockDuration(o.jobSinkBlockDuration),
			options.WithSinkAckGracePeriod(o.ackGracePeriod))
		if err != nil {
			return nil, fmt.Errorf("AddNode: failed to create events sink for stream %q: %w", poolStreamName(poolName), err)
		}
		closed = make(chan struct{})
	}

	nodeStream, err = streaming.NewStream(nodeStreamName(poolName, nodeID), rdb, options.WithStreamLogger(logger))
	if err != nil {
		return nil, fmt.Errorf("AddNode: failed to create node event stream %q: %w", nodeStreamName(poolName, nodeID), err)
	}
	if _, err = nodeStream.Add(ctx, evInit, []byte(nodeID)); err != nil {
		return nil, fmt.Errorf("AddNode: failed to add init event to node event stream %q: %w", nodeStreamName(poolName, nodeID), err)
	}

	nodeReader, err = nodeStream.NewReader(ctx, options.WithReaderBlockDuration(o.jobSinkBlockDuration), options.WithReaderStartAtOldest())
	if err != nil {
		return nil, fmt.Errorf("AddNode: failed to create node event reader for stream %q: %w", nodeStreamName(poolName, nodeID), err)
	}

	p := &Node{
		ID:                 nodeID,
		PoolName:           poolName,
		nodeKeepAliveMap:   nkm,
		nodeShutdownMap:    nsm,
		workerMap:          wm,
		workerKeepAliveMap: wkm,
		workerCleanupMap:   wcm,
		jobMap:             jm,
		jobPayloadMap:      jpm,
		jobPendingMap:      jpem,
		tickerMap:          tm,
		workerStreams:      sync.Map{},
		nodeStreams:        sync.Map{},
		pendingJobChannels: sync.Map{},
		pendingEvents:      sync.Map{},
		poolStream:         poolStream,
		poolSink:           poolSink,
		nodeStream:         nodeStream,
		nodeReader:         nodeReader,
		clientOnly:         o.clientOnly,
		workerTTL:          o.workerTTL,
		workerShutdownTTL:  o.workerShutdownTTL,
		ackGracePeriod:     o.ackGracePeriod,
		h:                  &jumpHash{h: crc64.New(crc64.MakeTable(crc64.ECMA))},
		stop:               make(chan struct{}),
		closed:             closed,
		rdb:                rdb,
		logger:             logger,
	}

	nch := nodeReader.Subscribe()

	if o.clientOnly {
		logger.Info("client-only")
		p.wg.Add(3)
		pulse.Go(logger, func() { p.handleNodeEvents(nch) }) // to handle job acks
		pulse.Go(logger, func() { p.processInactiveNodes() })
		pulse.Go(logger, func() { p.updateNodeKeepAlive() })
		return p, nil
	}

	// create new logger context for goroutines.
	logCtx := context.Background()
	logCtx = log.WithContext(logCtx, ctx)

	p.wg.Add(8) // Increment for all background goroutines
	pulse.Go(logger, func() { p.handlePoolEvents(poolSink.Subscribe()) })
	pulse.Go(logger, func() { p.handleNodeEvents(nch) })
	pulse.Go(logger, func() { p.watchWorkers(logCtx) })
	pulse.Go(logger, func() { p.watchShutdown(logCtx) })
	pulse.Go(logger, func() { p.processInactiveNodes() })
	pulse.Go(logger, func() { p.processInactiveWorkers(logCtx) })
	pulse.Go(logger, func() { p.processInactiveJobs(logCtx) })
	pulse.Go(logger, func() { p.updateNodeKeepAlive() })

	return p, nil
}

// AddWorker adds a new worker to the pool and returns it. The worker starts
// processing jobs immediately. handler can optionally implement the
// NotificationHandler interface to handle notifications.
func (node *Node) AddWorker(ctx context.Context, handler JobHandler) (*Worker, error) {
	if node.IsClosed() {
		return nil, fmt.Errorf("AddWorker: pool %q is closed", node.PoolName)
	}
	if node.clientOnly {
		return nil, fmt.Errorf("AddWorker: pool %q is client-only", node.PoolName)
	}
	w, err := newWorker(ctx, node, handler)
	if err != nil {
		return nil, err
	}
	node.localWorkers.Store(w.ID, w)
	node.workerStreams.Store(w.ID, w.stream)
	return w, nil
}

// RemoveWorker stops the worker, removes it from the pool and requeues all its
// jobs.
func (node *Node) RemoveWorker(ctx context.Context, w *Worker) error {
	w.stop(ctx)
	if err := w.requeueJobs(ctx); err != nil {
		node.logger.Error(fmt.Errorf("RemoveWorker: failed to requeue jobs for worker %q: %w", w.ID, err))
	}
	node.removeWorker(ctx, w.ID)
	node.localWorkers.Delete(w.ID)
	node.logger.Info("removed worker", "worker", w.ID)
	return nil
}

// Workers returns the list of workers running in the local node.
func (node *Node) Workers() []*Worker {
	var workers []*Worker
	node.localWorkers.Range(func(key, value any) bool {
		w := value.(*Worker)
		workers = append(workers, &Worker{
			ID:        w.ID,
			CreatedAt: w.CreatedAt,
		})
		return true
	})
	return workers
}

// PoolWorkers returns the list of workers running in the entire pool.
func (node *Node) PoolWorkers() []*Worker {
	workers := node.workerMap.Map()
	poolWorkers := make([]*Worker, 0, len(workers))
	for id, createdAt := range workers {
		cat, err := strconv.ParseInt(createdAt, 10, 64)
		if err != nil {
			node.logger.Error(fmt.Errorf("PoolWorkers: failed to parse createdAt %q for worker %q: %w", createdAt, id, err))
			continue
		}
		poolWorkers = append(poolWorkers, &Worker{ID: id, CreatedAt: time.Unix(0, cat)})
	}
	return poolWorkers
}

// DispatchJob dispatches a job to the worker in the pool that is assigned to
// the job key using consistent hashing.
// It returns:
// - nil if the job is successfully dispatched and started by a worker
// - ErrJobExists if a job with the same key already exists in the pool
// - an error returned by the worker's start handler if the job fails to start
// - an error if the pool is closed or if there's a failure in adding the job
//
// The method blocks until one of the above conditions is met.
func (node *Node) DispatchJob(ctx context.Context, key string, payload []byte) error {
	job := marshalJob(&Job{Key: key, Payload: payload, CreatedAt: time.Now(), NodeID: node.ID})
	return node.dispatchJob(ctx, key, job)
}

func (node *Node) dispatchJob(ctx context.Context, key string, job []byte) error {
	if node.IsClosed() {
		return fmt.Errorf("DispatchJob: pool %q is closed", node.PoolName)
	}

	pendingTS, err := node.claimDispatch(ctx, key)
	if err != nil {
		return err
	}

	eventID, err := node.poolStream.Add(ctx, evStartJob, job)
	if err != nil {
		// Clean up pending entry on failure
		node.releaseDispatchPending(key, pendingTS)
		return fmt.Errorf("DispatchJob: failed to add job to stream %q: %w", node.poolStream.Name, err)
	}

	cherr := make(chan error, 1)
	node.pendingJobChannels.Store(eventID, cherr)

	timer := time.NewTimer(2 * node.ackGracePeriod)
	defer timer.Stop()

	select {
	case err = <-cherr:
	case <-timer.C:
		err = fmt.Errorf("DispatchJob: job %q timed out, TTL: %v", key, 2*node.ackGracePeriod)
	case <-ctx.Done():
		err = ctx.Err()
	}

	node.pendingJobChannels.Delete(eventID)
	close(cherr)

	// Clean up pending entry
	node.releaseDispatchPending(key, pendingTS)

	if err != nil {
		node.logger.Error(fmt.Errorf("DispatchJob: failed to dispatch job: %w", err), "key", key)
		return err
	}

	node.logger.Info("dispatched", "key", key)
	return nil
}

// claimDispatch atomically decides whether a job key may be dispatched. Redis is
// the source of truth for both states that matter to singleton admission: a
// durable payload means the job is already running, and a pending guard means a
// worker is currently starting it.
func (node *Node) claimDispatch(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("DispatchJob: job key cannot be empty")
	}
	if strings.Contains(key, "=") {
		return "", fmt.Errorf("DispatchJob: job key %q cannot contain '='", key)
	}
	now := time.Now()
	pendingUntil := strconv.FormatInt(now.Add(2*node.ackGracePeriod).UnixNano(), 10)
	raw, err := luaClaimDispatch.Run(ctx, node.rdb, []string{
		rmapContentKey(jobPayloadMapName(node.PoolName)),
		rmapContentKey(jobPendingMapName(node.PoolName)),
		rmapUpdateChannel(jobPendingMapName(node.PoolName)),
	}, key, strconv.FormatInt(now.UnixNano(), 10), pendingUntil).Result()
	if err != nil {
		return "", fmt.Errorf("DispatchJob: failed to claim job %q: %w", key, err)
	}
	status, value, err := parseDispatchClaim(raw)
	if err != nil {
		return "", fmt.Errorf("DispatchJob: failed to parse claim result for job %q: %w", key, err)
	}
	switch status {
	case dispatchClaimed:
		return value, nil
	case dispatchAlreadyPending:
		node.logger.Info("DispatchJob: job already dispatched", "key", key)
		return "", fmt.Errorf("%w: job %q is already dispatched", ErrJobExists, key)
	case dispatchAlreadyRunning:
		node.logger.Info("DispatchJob: job already exists", "key", key)
		return "", fmt.Errorf("%w: job %q", ErrJobExists, key)
	case dispatchMalformedPending:
		return "", fmt.Errorf("DispatchJob: malformed pending guard for job %q: %q", key, value)
	default:
		return "", fmt.Errorf("DispatchJob: unexpected claim status %d for job %q", status, key)
	}
}

// releaseDispatchPending clears the pending guard only if this dispatch still
// owns it. Dispatch callers can time out while another node later claims a
// stale pending key, so unconditional deletion would erase a newer guard.
func (node *Node) releaseDispatchPending(key, pendingTS string) {
	if _, err := luaReleaseDispatch.Run(context.Background(), node.rdb, []string{
		rmapContentKey(jobPendingMapName(node.PoolName)),
		rmapUpdateChannel(jobPendingMapName(node.PoolName)),
	}, key, pendingTS).Result(); err != nil {
		node.logger.Error(fmt.Errorf("DispatchJob: failed to clean up pending entry for job %q: %w", key, err))
	}
}

// parseDispatchClaim decodes the Lua admission result into its status code and
// payload. The script owns the result schema so malformed data is a programming
// error, not a recoverable distributed state.
func parseDispatchClaim(raw any) (int64, string, error) {
	values, ok := raw.([]any)
	if !ok || len(values) != 2 {
		return 0, "", fmt.Errorf("invalid claim result %T", raw)
	}
	status, ok := values[0].(int64)
	if !ok {
		return 0, "", fmt.Errorf("invalid claim status %T", values[0])
	}
	value, ok := values[1].(string)
	if !ok {
		return 0, "", fmt.Errorf("invalid claim value %T", values[1])
	}
	return status, value, nil
}

// StopJob stops the job with the given key.
func (node *Node) StopJob(ctx context.Context, key string) error {
	if node.IsClosed() {
		return fmt.Errorf("StopJob: pool %q is closed", node.PoolName)
	}
	if _, err := node.poolStream.Add(ctx, evStopJob, marshalJobKey(key)); err != nil {
		return fmt.Errorf("StopJob: failed to add stop job to stream %q: %w", node.poolStream.Name, err)
	}
	node.logger.Info("stop requested", "key", key)
	return nil
}

// JobKeys returns the list of keys of the jobs running in the pool.
func (node *Node) JobKeys() []string {
	var jobKeys []string
	for workerID := range node.jobMap.Map() {
		keys, ok := node.jobMap.GetValues(workerID)
		if !ok {
			continue
		}
		jobKeys = append(jobKeys, keys...)
	}
	return jobKeys
}

// JobPayload returns the payload of the job with the given key.
// It returns:
// - (payload, true) if the job exists and has a payload
// - (nil, true) if the job exists but has an empty payload
// - (nil, false) if the job does not exist
func (node *Node) JobPayload(key string) ([]byte, bool) {
	payload, ok := node.jobPayloadMap.Get(key)
	if !ok {
		return nil, false
	}
	if payload == "" {
		return nil, true
	}
	return []byte(payload), true
}

// NotifyWorker notifies the worker that handles the job with the given key.
func (node *Node) NotifyWorker(ctx context.Context, key string, payload []byte) error {
	if node.IsClosed() {
		return fmt.Errorf("NotifyWorker: pool %q is closed", node.PoolName)
	}
	if _, err := node.poolStream.Add(ctx, evNotify, marshalNotification(key, payload)); err != nil {
		return fmt.Errorf("NotifyWorker: failed to add notification to stream %q: %w", node.poolStream.Name, err)
	}
	node.logger.Info("notification sent", "key", key)
	return nil
}

// Shutdown stops the pool workers gracefully across all nodes. It notifies all
// workers and waits until they are completed. Shutdown prevents the pool nodes
// from creating new workers and the pool workers from accepting new jobs. After
// Shutdown returns, the node object cannot be used anymore and should be
// discarded. One of Shutdown or Close should be called before the node is
// garbage collected unless it is client-only.
func (node *Node) Shutdown(ctx context.Context) error {
	if node.IsClosed() {
		return nil
	}
	if node.clientOnly {
		return fmt.Errorf("Shutdown: client-only node cannot shutdown worker pool")
	}

	// Signal all nodes to shutdown.
	if _, err := node.nodeShutdownMap.Set(ctx, "shutdown", node.ID); err != nil {
		node.logger.Error(fmt.Errorf("Shutdown: failed to set shutdown status in shutdown map: %w", err))
	}
	<-node.closed // Wait for this node to be closed
	node.cleanupPool(ctx)

	node.logger.Info("shutdown")
	return nil
}

// Close stops the node workers and closes the Redis connection but does
// not stop workers running in other nodes. It requeues all the jobs run by
// workers of the node. One of Shutdown or Close should be called before the
// node is garbage collected unless it is client-only.
func (node *Node) Close(ctx context.Context) error {
	return node.close(ctx, false)
}

// IsShutdown returns true if the pool is shutdown.
func (node *Node) IsShutdown() bool {
	node.lock.RLock()
	defer node.lock.RUnlock()
	return node.shutdown
}

// IsClosed returns true if the node is closed.
func (node *Node) IsClosed() bool {
	node.lock.RLock()
	defer node.lock.RUnlock()
	return node.closing
}

// close stops the node and its workers, optionally requeuing jobs. If shutdown
// is true, jobs are not requeued as the pool is being shutdown. Otherwise, jobs
// are requeued to be picked up by other nodes. The method stops all workers,
// waits for background goroutines to complete, cleans up resources and closes
// connections. It is idempotent and can be called multiple times safely.
func (node *Node) close(ctx context.Context, shutdown bool) error {
	node.lock.Lock()
	if node.closing {
		node.lock.Unlock()
		return nil
	}
	node.closing = true
	node.lock.Unlock()

	// If we're shutting down then stop all the jobs.
	if shutdown {
		node.stopAllJobs(ctx)
	}

	// Stop all workers before waiting for goroutines.
	//
	// IMPORTANT: do NOT remove workers from the replicated maps here.
	// Removing the worker deletes the worker->jobs mapping which is what other
	// nodes use to recover/requeue jobs if this node dies mid-close. We only
	// remove workers from maps after we've attempted to requeue.
	var wg sync.WaitGroup
	node.localWorkers.Range(func(key, value any) bool {
		worker := value.(*Worker)
		wg.Add(1)
		pulse.Go(node.logger, func() {
			defer wg.Done()
			worker.stop(ctx)
		})
		return true
	})
	wg.Wait()

	// Stop all goroutines
	close(node.stop)
	node.wg.Wait()

	// Requeue jobs if not shutting down.
	//
	// This is done after stopping node goroutines so we don't route any new pool
	// events to workers that have already been stopped.
	if !shutdown {
		if err := node.requeueAllJobs(ctx); err != nil {
			node.logger.Error(fmt.Errorf("close: failed to requeue jobs: %w", err))
		}
	}

	// Now that we attempted requeue, remove all local workers from pool maps.
	node.localWorkers.Range(func(key, value any) bool {
		worker := value.(*Worker)
		node.removeWorker(ctx, worker.ID)
		node.localWorkers.Delete(key)
		return true
	})

	// Cleanup resources
	node.cleanupNode(ctx)

	// Signal that the node is closed
	close(node.closed)

	node.logger.Info("closed")
	return nil
}
