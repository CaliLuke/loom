package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/streaming"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

// stopAllJobs stops all jobs running on the node.
func (node *Node) stopAllJobs(ctx context.Context) {
	var wg sync.WaitGroup
	var total atomic.Int32
	node.localWorkers.Range(func(key, value any) bool {
		wg.Add(1)
		worker := value.(*Worker)
		pulse.Go(node.logger, func() {
			defer wg.Done()
			for _, job := range worker.Jobs() {
				if err := worker.stopJob(ctx, job.Key); err != nil {
					node.logger.Error(fmt.Errorf("Close: failed to stop job %q for worker %q: %w", job.Key, worker.ID, err))
				}
				total.Add(1)
			}
		})
		return true
	})
	wg.Wait()
	node.logger.Info("stopped all jobs", "total", total.Load())
}

// handlePoolEvents reads events from the pool job stream.
func (node *Node) handlePoolEvents(ctx context.Context, c <-chan *streaming.Event) {
	defer node.wg.Done()

	for {
		select {
		case ev, ok := <-c:
			if !ok {
				return
			}
			if err := node.routeWorkerEvent(ctx, ev); err != nil {
				node.logger.Error(fmt.Errorf("handlePoolEvents: failed to route event: %w", err))
			}
		case <-node.stop:
			node.poolSink.Close(ctx)
			return
		}
	}
}

// routeWorkerEvent routes a dispatched event to the proper worker.
func (node *Node) routeWorkerEvent(ctx context.Context, ev *streaming.Event) error {
	// Filter out stale events
	if time.Since(ev.CreatedAt()) > pendingEventTTL {
		node.logger.Debug("routeWorkerEvent: stale event, not routing", "event", ev.EventName, "id", ev.ID, "since", time.Since(ev.CreatedAt()), "TTL", pendingEventTTL)
		// Ack the sink event so it does not get redelivered.
		if err := node.poolSink.Ack(ctx, ev); err != nil {
			node.logger.Error(fmt.Errorf("routeWorkerEvent: failed to ack event: %w", err), "event", ev.EventName, "id", ev.ID)
		}
		return nil
	}

	// Compute the worker ID that will handle the job.
	key, err := unmarshalJobKey(ev.Payload)
	if err != nil {
		// A malformed event can never be routed. Ack it so the sink does not
		// redeliver it, and keep routing later events.
		if ackErr := node.poolSink.Ack(ctx, ev); ackErr != nil {
			node.logger.Error(fmt.Errorf("routeWorkerEvent: failed to ack malformed event: %w", ackErr), "event", ev.EventName, "id", ev.ID)
		}
		return fmt.Errorf("routeWorkerEvent: dropped malformed event %s: %w", ev.ID, err)
	}
	activeWorkers := node.activeWorkers()
	if len(activeWorkers) == 0 {
		return fmt.Errorf("routeWorkerEvent: no active worker in pool %q", node.PoolName)
	}
	wid := activeWorkers[node.h.Hash(key, int64(len(activeWorkers)))]

	// Stream the event to the worker corresponding to the key hash.
	stream, err := node.getWorkerStream(wid)
	if err != nil {
		return err
	}
	eventID, err := stream.Add(ctx, ev.EventName, marshalEnvelope(node.ID, ev.Payload), options.WithOnlyIfStreamExists())
	if err != nil {
		return fmt.Errorf("routeWorkerEvent: failed to add event %s to worker stream %q: %w", ev.EventName, workerStreamName(wid), err)
	}
	node.logger.Debug("routed", "event", ev.EventName, "id", ev.ID, "worker", wid, "worker-event-id", eventID)

	// Record the event in the pending events map for future ack.
	node.pendingEvents.Store(pendingEventKey(wid, eventID), ev)

	return nil
}

// handleNodeEvents reads events from the node event stream and acks the pending
// events that correspond to jobs that are now running or done.
func (node *Node) handleNodeEvents(ctx context.Context, c <-chan *streaming.Event) {
	defer node.wg.Done()

	for {
		select {
		case ev, ok := <-c:
			if !ok {
				return
			}
			node.processNodeEvent(ctx, ev)
		case <-node.stop:
			node.nodeReader.Close()
			return
		}
	}
}

// processNodeEvent processes a node event.
func (node *Node) processNodeEvent(ctx context.Context, ev *streaming.Event) {
	switch ev.EventName {
	case evInit:
		// Event sent by pool node to initialize the node event stream.
		node.logger.Debug("handleNodeEvents: received init node", "event", ev.EventName, "id", ev.ID)
	case evAck:
		// Event sent by worker to ack a dispatched job.
		node.logger.Debug("handleNodeEvents: received ack", "event", ev.EventName, "id", ev.ID)
		node.ackWorkerEvent(ctx, ev)
	case evDispatchReturn:
		// Event sent by pool node to node that originally dispatched the job.
		node.logger.Debug("handleNodeEvents: received dispatch return", "event", ev.EventName, "id", ev.ID)
		node.returnDispatchStatus(ev)
	}
}

// ackWorkerEvent acks the pending event that corresponds to the acked job.  If
// the event was a dispatched job then it sends a dispatch return event to the
// node that dispatched the job. It XACKs the pool event first, so the
// dispatcher's guard release that follows the return finds the event acked
// and clears the guard.
func (node *Node) ackWorkerEvent(ctx context.Context, ev *streaming.Event) {
	workerID, payload, err := unmarshalEnvelope(ev.Payload)
	if err != nil {
		node.logger.Error(fmt.Errorf("ackWorkerEvent: dropping malformed ack envelope: %w", err), "id", ev.ID)
		return
	}
	ack, err := unmarshalAck(payload)
	if err != nil {
		node.logger.Error(fmt.Errorf("ackWorkerEvent: dropping malformed ack from worker %s: %w", workerID, err), "id", ev.ID)
		return
	}
	key := pendingEventKey(workerID, ack.EventID)
	val, ok := node.pendingEvents.Load(key)
	if !ok {
		node.logger.Error(fmt.Errorf("ackWorkerEvent: received unknown event %s from worker %s", ack.EventID, workerID))
		return
	}
	pending := val.(*streaming.Event)

	// Ack the sink event so it does not get redelivered.
	if err := node.poolSink.Ack(ctx, pending); err != nil {
		node.logger.Error(fmt.Errorf("ackWorkerEvent: failed to ack event: %w", err), "event", pending.EventName, "id", pending.ID)
	}
	// If a dispatched job then send a return event to the node that
	// dispatched the job.
	if pending.EventName == evStartJob {
		node.returnDispatch(ctx, pending, ack)
	}
	node.pendingEvents.Delete(key)

	// Garbage collect stale events.
	var staleKeys []string
	node.pendingEvents.Range(func(key, value any) bool {
		ev := value.(*streaming.Event)
		if time.Since(ev.CreatedAt()) > pendingEventTTL {
			staleKeys = append(staleKeys, key.(string))
			node.logger.Error(fmt.Errorf("ackWorkerEvent: stale event, removing from pending events"), "event", ev.EventName, "id", ev.ID, "since", time.Since(ev.CreatedAt()), "TTL", pendingEventTTL)
		}
		return true
	})
	for _, key := range staleKeys {
		node.pendingEvents.Delete(key)
	}
}

// returnDispatch sends the worker ack of the dispatched start event pending
// to the node that dispatched it. When the return cannot be sent, the
// dispatcher times out; its guard release then finds the event acked, and the
// payload refuses a retry of a job that started.
func (node *Node) returnDispatch(ctx context.Context, pending *streaming.Event, ack *ack) {
	_, nodeID, err := unmarshalJobKeyAndNodeID(pending.Payload)
	if err != nil {
		node.logger.Error(fmt.Errorf("ackWorkerEvent: no dispatch return for malformed job: %w", err), "id", pending.ID)
		return
	}
	stream, err := node.getNodeStream(nodeID)
	if err != nil {
		node.logger.Error(fmt.Errorf("ackWorkerEvent: failed to create node event stream %q: %w", nodeStreamName(node.PoolName, nodeID), err))
		return
	}
	ack.EventID = pending.ID
	if _, err := stream.Add(ctx, evDispatchReturn, marshalAck(ack), options.WithOnlyIfStreamExists()); err != nil {
		node.logger.Error(fmt.Errorf("ackWorkerEvent: failed to dispatch return to stream %q: %w", nodeStreamName(node.PoolName, nodeID), err))
	}
}

// returnDispatchStatus returns the start job result to the caller.
func (node *Node) returnDispatchStatus(ev *streaming.Event) {
	ack, err := unmarshalAck(ev.Payload)
	if err != nil {
		node.logger.Error(fmt.Errorf("returnDispatchStatus: dropping malformed dispatch return: %w", err), "id", ev.ID)
		return
	}
	var startErr error
	if ack.Error != "" {
		startErr = errors.New(ack.Error)
	}
	node.dispatchReturnsLock.Lock()
	val, ok := node.pendingJobChannels.LoadAndDelete(ack.EventID)
	if !ok {
		// The dispatcher may not have registered yet: park the return.
		node.parkDispatchReturn(ack.EventID, startErr)
		node.dispatchReturnsLock.Unlock()
		node.logger.Debug("returnDispatchStatus: dispatch return before registration", "id", ack.EventID)
		return
	}
	node.dispatchReturnsLock.Unlock()
	node.logger.Debug("dispatch return", "event", ev.EventName, "id", ev.ID, "ack-id", ack.EventID)
	if val == nil {
		return
	}
	cherr := val.(chan error)
	defer func() {
		if r := recover(); r != nil {
			node.logger.Error(fmt.Errorf("returnDispatchStatus: dispatch return channel for event %s is closed: %v", ack.EventID, r))
		}
	}()
	cherr <- startErr
}

// parkDispatchReturn records a dispatch return whose dispatcher has not
// registered. Returns for events no dispatcher waits for (requeues,
// redeliveries, dispatchers that gave up) are dropped once they are older
// than the dispatch timeout. dispatchReturnsLock must be held.
func (node *Node) parkDispatchReturn(eventID string, err error) {
	now := time.Now()
	for id, early := range node.earlyReturns {
		if now.Sub(early.at) > 2*node.ackGracePeriod {
			delete(node.earlyReturns, id)
		}
	}
	if node.earlyReturns == nil {
		node.earlyReturns = make(map[string]earlyDispatchReturn)
	}
	node.earlyReturns[eventID] = earlyDispatchReturn{err: err, at: now}
}

// watches monitors the workers replicated map and triggers job rebalancing
// when workers are added or removed from the pool.
func (node *Node) watchWorkers(ctx context.Context) {
	defer node.wg.Done()
	updates := node.workerMap.Subscribe()
	defer node.workerMap.Unsubscribe(updates)
	for {
		select {
		case <-node.stop:
			return
		case _, ok := <-updates:
			if !ok {
				return
			}
			node.logger.Debug("watchWorkers: worker map updated")
			node.handleWorkerMapUpdate(ctx)
		}
	}
}

// handleWorkerMapUpdate is called when the worker map is updated.
func (node *Node) handleWorkerMapUpdate(ctx context.Context) {
	if node.IsClosed() {
		return
	}
	// First cleanup the local workers that are no longer active.
	node.localWorkers.Range(func(key, value any) bool {
		worker := value.(*Worker)
		if _, ok := node.workerMap.Get(worker.ID); !ok {
			// If it's not in the worker map, then it's not active and its jobs
			// have already been requeued. Stop its handlers so they do not
			// keep running next to the workers that took the jobs over.
			node.logger.Info("handleWorkerMapUpdate: removing inactive local worker", "worker", worker.ID)
			// Eviction holds no cleanup lock, so it must not release one.
			if err := node.deleteWorker(ctx, worker.ID, ""); err != nil {
				node.logger.Error(fmt.Errorf("handleWorkerMapUpdate: failed to delete inactive worker %q: %w", worker.ID, err), "worker", worker.ID)
			}
			worker.stop(ctx)
			worker.stopHandlers()
			node.localWorkers.Delete(key)
			return true
		}
		return true
	})

	// Then rebalance the jobs across the remaining active workers.
	activeWorkers := node.activeWorkers()
	if len(activeWorkers) == 0 {
		return
	}
	node.localWorkers.Range(func(key, value any) bool {
		worker := value.(*Worker)
		worker.rebalance(ctx, activeWorkers)
		return true
	})
}
