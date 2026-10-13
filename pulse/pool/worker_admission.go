package pool

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/CaliLuke/loom/clue/log"
	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/streaming"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

// AddWorker adds a new worker to the pool and returns it. Processing may start
// before the method returns successfully. A failed attempt never starts handlers
// or renews its initial lease. Close excludes new attempts and joins admitted
// constructors before stopping their workers. handler can optionally implement
// NotificationHandler to handle notifications.
func (node *Node) AddWorker(ctx context.Context, handler JobHandler) (_ *Worker, err error) {
	node.lock.Lock()
	if node.closing {
		node.lock.Unlock()
		return nil, fmt.Errorf("AddWorker: pool %q is closed", node.PoolName)
	}
	if node.clientOnly {
		node.lock.Unlock()
		return nil, fmt.Errorf("AddWorker: pool %q is client-only", node.PoolName)
	}
	node.workerAdmissions.Add(1)
	node.lock.Unlock()
	defer node.workerAdmissions.Done()

	w, err := node.prepareWorker(ctx, handler)
	defer func() {
		if err != nil {
			err = errors.Join(err, w.abortAdmission(ctx))
		}
	}()

	if err != nil {
		return nil, err
	}
	if _, err := node.workerMap.SetAndWait(ctx, w.ID, strconv.FormatInt(w.CreatedAt.UnixNano(), 10)); err != nil {
		return nil, fmt.Errorf("publish worker membership: %w", err)
	}
	leaseStart := time.Now()
	if err := node.initialWorkerHeartbeat(ctx, w.ID); err != nil {
		return nil, fmt.Errorf("publish worker heartbeat: %w", err)
	}
	w.recordLease(leaseStart)

	node.lock.Lock()
	switch {
	case node.closing:
		err = fmt.Errorf("AddWorker: pool %q is closing", node.PoolName)
	case ctx.Err() != nil:
		err = ctx.Err()
	case w.leaseExpired(time.Now()):
		err = fmt.Errorf("AddWorker: initial worker lease expired before admission")
	}
	if err != nil {
		node.lock.Unlock()
		return nil, err
	}
	node.localWorkers.Store(w.ID, w)
	node.workerStreams.Store(w.ID, w.stream)
	w.start()
	node.lock.Unlock()
	w.logger.Info("created", "worker_ttl", w.workerTTL, "worker_shutdown_ttl", w.workerShutdownTTL)
	return w, nil
}

// prepareWorker allocates the attempt without subscribing or starting loops.
// It returns the partial worker on failure so the admission owner can release it.
func (node *Node) prepareWorker(ctx context.Context, handler JobHandler) (*Worker, error) {
	wid := ulid.Make().String()
	runtimeCtx, runtimeCancel := context.WithCancel(log.WithContext(context.WithoutCancel(ctx), ctx))
	w := &Worker{
		ID:                wid,
		CreatedAt:         time.Now(),
		node:              node,
		handler:           handler,
		done:              make(chan struct{}),
		leaseWake:         make(chan struct{}, 1),
		ownershipWake:     make(chan struct{}, 1),
		jobsMap:           node.jobMap,
		jobPayloadsMap:    node.jobPayloadMap,
		keepAliveMap:      node.workerKeepAliveMap,
		shutdownMap:       node.nodeShutdownMap,
		workerTTL:         node.workerTTL,
		workerShutdownTTL: node.workerShutdownTTL,
		logger:            node.logger.WithPrefix("worker", wid),
		runtimeCtx:        runtimeCtx,
		runtimeCancel:     runtimeCancel,
	}
	var err error
	w.stream, err = streaming.NewStream(workerStreamName(wid), node.rdb, options.WithStreamLogger(node.logger))
	if err != nil {
		return w, fmt.Errorf("create worker stream: %w", err)
	}
	if _, err := w.stream.Add(ctx, evInit, marshalEnvelope(node.ID, []byte(wid))); err != nil {
		return w, fmt.Errorf("initialize worker stream: %w", err)
	}
	w.reader, err = w.stream.NewReader(ctx, options.WithReaderBlockDuration(node.workerTTL/2), options.WithReaderStartAtOldest())
	if err != nil {
		return w, fmt.Errorf("create worker reader: %w", err)
	}
	return w, nil
}

// abortAdmission releases only this attempt's private state. Its reader has
// never subscribed, so local release cannot wait for a blocked Redis read.
// Uncertain withdrawal may leave stale publication until the initial lease
// expires; no runtime can renew it or acknowledge a routed pool event.
func (w *Worker) abortAdmission(ctx context.Context) error {
	w.runtimeCancel()
	if w.reader != nil {
		w.reader.Close()
	}
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), w.workerShutdownTTL)
	defer cancel()
	var errs []error
	if w.stream != nil {
		if err := w.stream.Destroy(cleanupCtx); err != nil {
			errs = append(errs, fmt.Errorf("withdraw worker stream: %w", err))
		}
	}
	if _, err := w.node.workerMap.Delete(cleanupCtx, w.ID); err != nil {
		errs = append(errs, fmt.Errorf("withdraw worker membership: %w", err))
	}
	if _, err := w.node.workerKeepAliveMap.Delete(cleanupCtx, w.ID); err != nil {
		errs = append(errs, fmt.Errorf("withdraw worker heartbeat: %w", err))
	}
	w.node.workerStreams.Delete(w.ID)
	return errors.Join(errs...)
}

// start transfers prepared resources to the runtime under node.lock. Register
// all goroutines and subscribe before Close can take its worker snapshot.
func (w *Worker) start() {
	w.wg.Add(5)
	events := w.reader.Subscribe()
	pulse.Go(w.logger, func() {
		w.handleEvents(w.runtimeCtx, events)
	})
	pulse.Go(w.logger, func() {
		w.keepAlive(w.runtimeCtx)
	})
	pulse.Go(w.logger, func() {
		w.monitorLease()
	})
	pulse.Go(w.logger, func() {
		w.resumeLease(w.runtimeCtx)
	})
	pulse.Go(w.logger, func() {
		w.retryPendingReleases(w.runtimeCtx)
	})
}
