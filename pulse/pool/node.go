package pool

import (
	"context"
	"errors"
	"fmt"
	"hash"
	"hash/crc64"
	"strconv"
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
		runtimeCtx         context.Context
		runtimeCancel      context.CancelFunc
		stop               chan struct{}  // closed when node is stopped
		closed             chan struct{}  // closed when node is closed
		wg                 sync.WaitGroup // allows to wait until all goroutines exit
		rdb                *redis.Client

		localWorkers       sync.Map // workers created by this node
		localSchedulers    sync.Map // schedulers created by this node
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
		closed     = make(chan struct{})
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

	runtimeBase := log.WithContext(context.WithoutCancel(ctx), ctx)
	runtimeCtx, runtimeCancel := context.WithCancel(runtimeBase)
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
		runtimeCtx:         runtimeCtx,
		runtimeCancel:      runtimeCancel,
		stop:               make(chan struct{}),
		closed:             closed,
		rdb:                rdb,
		logger:             logger,
	}

	nch := nodeReader.Subscribe()

	if o.clientOnly {
		logger.Info("client-only")
		p.wg.Add(3)
		pulse.Go(logger, func() { p.handleNodeEvents(runtimeCtx, nch) }) // to handle job acks
		pulse.Go(logger, func() { p.processInactiveNodes(runtimeCtx) })
		pulse.Go(logger, func() { p.updateNodeKeepAlive(runtimeCtx) })
		return p, nil
	}

	p.wg.Add(8) // Increment for all background goroutines
	pulse.Go(logger, func() { p.handlePoolEvents(runtimeCtx, poolSink.Subscribe()) })
	pulse.Go(logger, func() { p.handleNodeEvents(runtimeCtx, nch) })
	pulse.Go(logger, func() { p.watchWorkers(runtimeCtx) })
	pulse.Go(logger, func() { p.watchShutdown(runtimeCtx) })
	pulse.Go(logger, func() { p.processInactiveNodes(runtimeCtx) })
	pulse.Go(logger, func() { p.processInactiveWorkers(runtimeCtx) })
	pulse.Go(logger, func() { p.processInactiveJobs(runtimeCtx) })
	pulse.Go(logger, func() { p.updateNodeKeepAlive(runtimeCtx) })

	return p, nil
}
