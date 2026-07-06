package pool

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/streaming"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

// watchShutdown monitors the pool shutdown map and initiates node shutdown when updated.
func (node *Node) watchShutdown(ctx context.Context) {
	defer node.wg.Done()
	for {
		select {
		case <-node.stop:
			return
		case <-node.nodeShutdownMap.Subscribe():
			node.logger.Debug("watchShutdown: shutdown map updated")
			// Handle shutdown in a separate goroutine to allow this one to exit
			pulse.Go(node.logger, func() { node.handleShutdown(ctx) })
		}
	}
}

// handleShutdown closes the node.
func (node *Node) handleShutdown(ctx context.Context) {
	if node.IsClosed() {
		return
	}
	sm := node.nodeShutdownMap.Map()
	var requestingNode string
	for _, node := range sm {
		// There is only one value in the map
		requestingNode = node
	}
	node.logger.Debug("handleShutdown: shutting down", "requested-by", requestingNode)
	if err := node.close(ctx, true); err != nil {
		node.logger.Error(fmt.Errorf("handleShutdown: failed to close node: %w", err))
	}

	node.lock.Lock()
	node.shutdown = true
	node.lock.Unlock()
	node.logger.Info("shutdown", "requested-by", requestingNode)
}

// processInactiveNodes periodically checks for inactive nodes and destroys their streams.
func (node *Node) processInactiveNodes() {
	defer node.wg.Done()
	ticker := time.NewTicker(node.workerTTL)
	defer ticker.Stop()

	for {
		select {
		case <-node.stop:
			return
		case <-ticker.C:
			node.cleanupInactiveNodes()
		}
	}
}

// cleanupInactiveNodes checks for inactive nodes, destroys their streams and
// removes them from the keep-alive map.
func (node *Node) cleanupInactiveNodes() {
	nodeMap := node.nodeKeepAliveMap.Map()
	for nodeID, lastSeen := range nodeMap {
		if nodeID == node.ID || node.isWithinTTL(lastSeen, node.workerTTL) {
			continue
		}

		node.logger.Info("cleaning up inactive node", "node", nodeID)

		// Clean up node's stream
		ctx := context.Background()
		stream := nodeStreamName(node.PoolName, nodeID)
		if s, err := streaming.NewStream(stream, node.rdb, options.WithStreamLogger(node.logger)); err == nil {
			if err := s.Destroy(ctx); err != nil {
				node.logger.Error(fmt.Errorf("cleanupInactiveNodes: failed to destroy stream: %w", err))
			}
		}

		// Remove from keep-alive map
		if _, err := node.nodeKeepAliveMap.Delete(ctx, nodeID); err != nil {
			node.logger.Error(fmt.Errorf("cleanupInactiveNodes: failed to delete node: %w", err))
		}
	}
}

// processInactiveWorkers periodically cleans up inactive workers.
func (node *Node) processInactiveWorkers(ctx context.Context) {
	defer node.wg.Done()
	ticker := time.NewTicker(node.workerTTL)
	defer ticker.Stop()

	for {
		select {
		case <-node.stop:
			return
		case <-ticker.C:
			node.cleanupInactiveWorkers(ctx)
		}
	}
}

// cleanupInactiveWorkers ensures all jobs are assigned to active workers by performing
// two types of cleanup:
//  1. Orphaned jobs: finds and requeues jobs assigned to workers that no longer exist
//     in the keep-alive map, which can happen if a worker was improperly terminated
//  2. Inactive workers: finds workers that haven't updated their keep-alive timestamp
//     within workerTTL duration and requeues their jobs
//
// The cleanup process is distributed and idempotent - multiple nodes can attempt
// cleanup concurrently, but only one will succeed for each worker due to cleanup
// lock acquisition. Jobs are requeued and will be reassigned to active workers
// through consistent hashing.
func (node *Node) cleanupInactiveWorkers(ctx context.Context) {
	active := node.activeWorkers()
	activeMap := make(map[string]struct{})
	for _, id := range active {
		activeMap[id] = struct{}{}
	}

	// Get all workers that need cleanup (either in jobMap or workerMap)
	workersToCheck := make(map[string]struct{})
	for _, workerID := range node.jobMap.Keys() {
		workersToCheck[workerID] = struct{}{}
	}
	for _, workerID := range node.workerMap.Keys() {
		workersToCheck[workerID] = struct{}{}
	}

	// Check each worker
	for workerID := range workersToCheck {
		// Skip active workers
		if _, ok := activeMap[workerID]; ok {
			continue
		}

		// Skip workers being cleaned up
		if cleanupTS, exists := node.workerCleanupMap.Get(workerID); exists {
			if node.isWithinTTL(cleanupTS, node.workerTTL) {
				node.logger.Debug("cleanupInactiveWorkers: worker already being cleaned up", "worker", workerID)
				continue
			}
		}

		// Worker needs cleanup
		node.logger.Info("cleanupInactiveWorkers: found inactive worker", "worker", workerID)
		node.cleanupWorker(ctx, workerID)
	}

	// Also recover any jobs that still have payloads but are missing from the job map.
	// This can happen transiently during cascading failures and is preferable to leaving
	// jobs "stuck" (payload exists, but no worker owns the job).
	node.requeueOrphanedPayloads(ctx)
}

// requeueOrphanedPayloads detects payloads for job keys that are not present in
// the job map and requeues them after a short grace period.
func (node *Node) requeueOrphanedPayloads(ctx context.Context) {
	// Build a set of all job keys referenced by the job map.
	existingJobs := make(map[string]struct{})
	for workerID := range node.jobMap.Map() {
		keys, ok := node.jobMap.GetValues(workerID)
		if !ok {
			continue
		}
		for _, key := range keys {
			if key == "" {
				continue
			}
			existingJobs[key] = struct{}{}
		}
	}

	// Use a short grace period: we want recovery to be fast under churn,
	// but still avoid requeuing during brief map inconsistencies.
	grace := 2 * node.workerTTL
	if grace < node.ackGracePeriod {
		grace = node.ackGracePeriod
	}

	now := time.Now()
	for key := range node.jobPayloadMap.Map() {
		if _, ok := existingJobs[key]; ok {
			node.orphanedPayloads.Delete(key)
			continue
		}

		firstAny, ok := node.orphanedPayloads.Load(key)
		if !ok {
			node.orphanedPayloads.Store(key, now.UnixNano())
			continue
		}
		firstNS, _ := firstAny.(int64)
		if firstNS == 0 || now.Sub(time.Unix(0, firstNS)) < grace {
			continue
		}

		payload, ok := node.JobPayload(key)
		if !ok {
			node.orphanedPayloads.Delete(key)
			continue
		}
		job := &Job{Key: key, Payload: payload, CreatedAt: now, NodeID: node.ID}
		if _, err := node.poolStream.Add(ctx, evStartJob, marshalJob(job)); err != nil {
			node.logger.Error(fmt.Errorf("requeueOrphanedPayloads: failed to requeue orphaned job: %w", err), "key", key)
			continue
		}

		node.orphanedPayloads.Delete(key)
		node.logger.Info("requeueOrphanedPayloads: requeued orphaned job", "key", key, "grace", grace)
	}
}

// cleanupWorker requeues the jobs assigned to the worker and deletes it from
// the pool.
func (node *Node) cleanupWorker(ctx context.Context, workerID string) {
	// Try to acquire or clear stale cleanup lock
	if !node.acquireCleanupLock(ctx, workerID) {
		return
	}

	// Get the worker's jobs
	keys, ok := node.jobMap.GetValues(workerID)
	if !ok || len(keys) == 0 {
		// Worker has no jobs, just delete it
		if err := node.deleteWorker(workerID); err != nil {
			node.logger.Error(fmt.Errorf("cleanupWorkerJobs: failed to delete worker: %w", err), "worker", workerID)
		}
		node.logger.Info("cleaned up worker with no jobs", "worker", workerID)
		return
	}

	// Requeue jobs and process them
	var (
		requeued  int // jobs successfully requeued
		processed int // jobs that were either requeued or cleaned up as stale
	)
	for _, key := range keys {
		payload, ok := node.JobPayload(key)
		if !ok {
			// The job key can remain in the jobs map even if the payload has already
			// been removed (e.g. the job was stopped, or another node already handled
			// the requeue). Treat it as a stale entry and remove it so future cleanup
			// attempts don't keep looping on it.
			if _, _, err := node.jobMap.RemoveValues(ctx, workerID, key); err != nil {
				node.logger.Error(fmt.Errorf("cleanupWorker: failed to remove stale job from jobs map: %w", err), "job", key, "worker", workerID)
				continue
			}
			node.logger.Info("cleanupWorker: removed stale job key with missing payload", "job", key, "worker", workerID)
			processed++
			continue
		}
		job := &Job{Key: key, Payload: payload, CreatedAt: time.Now(), NodeID: node.ID}
		// Requeue by adding an event back to the pool stream.
		// We intentionally do not wait for the job to start (which can time out
		// under heavy churn) - the pool sink will retry routing until it is acked.
		if _, err := node.poolStream.Add(ctx, evStartJob, marshalJob(job)); err != nil {
			node.logger.Error(fmt.Errorf("requeueWorkerJobs: failed to requeue job: %w", err), "job", job.Key, "worker", workerID)
			continue
		}
		requeued++
		processed++
	}
	if len(keys) != processed {
		node.logger.Info("partially processed stale worker jobs", "requeued", requeued, "processed", processed, "jobs", len(keys), "worker", workerID)
		return
	}

	// Delete worker
	node.logger.Info("cleaned up worker", "worker", workerID, "requeued", requeued)
	if err := node.deleteWorker(workerID); err != nil {
		node.logger.Error(fmt.Errorf("cleanupWorkerJobs: failed to delete worker: %w", err), "worker", workerID)
	}
}

// processInactiveJobs periodically checks for and removes stale entries in the pending jobs map.
func (node *Node) processInactiveJobs(ctx context.Context) {
	defer node.wg.Done()
	ticker := time.NewTicker(node.ackGracePeriod) // Run at ackGracePeriod frequency since pending jobs expire after 2*ackGracePeriod
	defer ticker.Stop()

	for {
		select {
		case <-node.stop:
			return
		case <-ticker.C:
			node.cleanupStalePendingJobs(ctx)
		}
	}
}

// cleanupStalePendingJobs checks for and removes stale entries in the pending jobs map.
// An entry is considered stale if its timestamp has expired.
func (node *Node) cleanupStalePendingJobs(ctx context.Context) {
	for key, pendingTS := range node.jobPendingMap.Map() {
		if _, err := strconv.ParseInt(pendingTS, 10, 64); err != nil {
			node.logger.Error(fmt.Errorf("cleanupStalePendingJobs: malformed pending timestamp for job %q: %w", key, err))
			continue
		}
		if node.isWithinTTL(pendingTS, 0) {
			continue
		}
		prev, err := node.jobPendingMap.TestAndDelete(ctx, key, pendingTS)
		if err != nil {
			node.logger.Error(fmt.Errorf("cleanupStalePendingJobs: failed to delete stale pending entry: %w", err))
		}
		if prev == pendingTS {
			node.logger.Info("cleanupStalePendingJobs: removed stale pending entry", "key", key)
		}
	}
}

// acquireCleanupLock tries to acquire the cleanup lock for a worker.
// It returns true if the lock was acquired, false if another node holds the lock.
// It will clear any stale or invalid locks it finds.
func (node *Node) acquireCleanupLock(ctx context.Context, workerID string) bool {
	// Check for existing lock
	if existingTS, exists := node.workerCleanupMap.Get(workerID); exists {
		if !node.isWithinTTL(existingTS, node.workerTTL) {
			// Invalid or stale lock, delete it
			if _, err := node.workerCleanupMap.Delete(ctx, workerID); err != nil {
				node.logger.Error(fmt.Errorf("cleanupWorkerJobs: failed to delete stale cleanup timestamp: %w", err), "worker", workerID)
				return false
			}
			node.logger.Info("cleanupWorkerJobs: cleared stale cleanup lock", "worker", workerID, "ts", existingTS, "ttl", node.workerTTL)
		} else {
			// Lock is still valid
			node.logger.Debug("cleanupWorkerJobs: cleanup already in progress", "worker", workerID)
			return false
		}
	}

	// Try to acquire lock
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	ok, err := node.workerCleanupMap.SetIfNotExists(ctx, workerID, now)
	if err != nil {
		node.logger.Error(fmt.Errorf("cleanupWorkerJobs: failed to set cleanup timestamp: %w", err), "worker", workerID)
		return false
	}
	if !ok {
		node.logger.Debug("cleanupWorkerJobs: cleanup already in progress", "worker", workerID)
		return false
	}

	return true
}

// isWithinTTL checks if a timestamp is within a TTL. If lastSeen is not a valid
// timestamp, false is returned. lastSeen is a string representation of a unix
// timestamp in nanoseconds.
func (node *Node) isWithinTTL(lastSeen string, ttl time.Duration) bool {
	lsi, err := strconv.ParseInt(lastSeen, 10, 64)
	if err != nil {
		node.logger.Error(fmt.Errorf("isWithinTTL: failed to parse last seen timestamp: %w", err))
		return false
	}
	return time.Since(time.Unix(0, lsi)) <= ttl
}

// Keep node alive
func (node *Node) updateNodeKeepAlive() {
	defer node.wg.Done()
	ticker := time.NewTicker(node.workerTTL / 2)
	defer ticker.Stop()

	ctx := context.Background()
	for {
		select {
		case <-node.stop:
			return
		case <-ticker.C:
			if _, err := node.nodeKeepAliveMap.Set(ctx, node.ID,
				strconv.FormatInt(time.Now().UnixNano(), 10)); err != nil {
				node.logger.Error(fmt.Errorf("updateNodeKeepAlive: failed to update timestamp: %w", err))
			}
		}
	}
}

// activeWorkers returns the IDs of the active workers in the pool.
func (node *Node) activeWorkers() []string {
	workers := node.workerMap.Map()
	workerCreatedAtByID := make(map[string]int64)
	var sortedIDs []string
	for id, createdAt := range workers {
		if createdAt == "-" {
			continue // worker is in the process of being removed
		}

		// Skip workers that are being cleaned up
		if cleanupTS, exists := node.workerCleanupMap.Get(id); exists {
			if node.isWithinTTL(cleanupTS, node.workerTTL) {
				continue // Skip workers being actively cleaned up
			}
		}
		cai, err := strconv.ParseInt(createdAt, 10, 64)
		if err != nil {
			node.logger.Error(fmt.Errorf("activeWorkers: failed to parse created at timestamp: %w", err), "worker", id)
			continue
		}
		workerCreatedAtByID[id] = cai
		sortedIDs = append(sortedIDs, id)
	}
	sort.Slice(sortedIDs, func(i, j int) bool {
		return workerCreatedAtByID[sortedIDs[i]] < workerCreatedAtByID[sortedIDs[j]]
	})

	// Then filter out workers that have not been seen for more than workerTTL.
	alive := node.workerKeepAliveMap.Map()
	var activeIDs []string
	for _, id := range sortedIDs {
		ls, ok := alive[id]
		if !ok {
			// This could happen if a worker is removed from the
			// pool and the last seen map deletion replicates before
			// the workers map deletion.
			continue
		}
		if !node.isWithinTTL(ls, node.workerTTL) {
			continue
		}
		activeIDs = append(activeIDs, id)
	}

	return activeIDs
}

// deleteWorker removes a remote worker from the pool deleting the worker stream.
func (node *Node) deleteWorker(id string) error {
	ctx := context.Background()
	node.logger.Debug("deleteWorker: deleting worker", "worker", id)

	// Remove from all maps including cleanup map
	node.removeWorkerFromMaps(ctx, id)

	// Destroy the worker's stream
	stream, err := node.getWorkerStream(id)
	if err != nil {
		return fmt.Errorf("deleteWorker: failed to retrieve worker stream for %q: %w", id, err)
	}
	if err := stream.Destroy(ctx); err != nil {
		node.logger.Error(fmt.Errorf("deleteWorker: failed to delete worker stream: %w", err))
	}
	return nil
}

// removeWorker removes a worker that was created by this node.
// This is used during graceful shutdown or explicit worker removal.
func (node *Node) removeWorker(ctx context.Context, id string) {
	node.removeWorkerFromMaps(ctx, id)
	node.workerStreams.Delete(id)
}

// removeWorkerFromMaps removes the worker from all tracking maps.
// This is the common cleanup needed for both local and remote worker removal.
func (node *Node) removeWorkerFromMaps(ctx context.Context, id string) {
	if _, err := node.workerMap.Delete(ctx, id); err != nil {
		node.logger.Error(fmt.Errorf("removeWorkerFromMaps: failed to remove worker %s from worker map: %w", id, err))
	}
	if _, err := node.workerKeepAliveMap.Delete(ctx, id); err != nil {
		node.logger.Error(fmt.Errorf("removeWorkerFromMaps: failed to remove worker %s from keep-alive map: %w", id, err))
	}
	if _, err := node.workerCleanupMap.Delete(ctx, id); err != nil {
		node.logger.Error(fmt.Errorf("removeWorkerFromMaps: failed to remove cleanup timestamp: %w", err), "worker", id)
	}
	// NOTE: Do not delete job payloads here.
	//
	// Payload entries are job-scoped (not worker-scoped) and are required to
	// safely requeue jobs from a stale worker during distributed cleanup. Deleting
	// payloads during worker removal can race with another node performing
	// cleanup/requeue and lead to permanent job loss.
	//
	// Payloads are deleted when jobs stop (see Worker.stopJob) and any remaining
	// orphaned payloads are eventually collected by cleanupOrphanedJobPayloads.
	if _, err := node.jobMap.Delete(ctx, id); err != nil {
		node.logger.Error(fmt.Errorf("removeWorkerFromMaps: failed to remove worker %s from jobs map: %w", id, err))
	}
}

// getWorkerStream retrieves the stream for a worker. It caches the result in the
// workerStreams map.
func (node *Node) getWorkerStream(id string) (*streaming.Stream, error) {
	val, ok := node.workerStreams.Load(id)
	if !ok {
		s, err := streaming.NewStream(workerStreamName(id), node.rdb, options.WithStreamLogger(node.logger))
		if err != nil {
			return nil, fmt.Errorf("workerStream: failed to retrieve stream for worker %q: %w", id, err)
		}
		node.workerStreams.Store(id, s)
		return s, nil
	}
	return val.(*streaming.Stream), nil
}

// getNodeStream retrieves the given node stream.
func (node *Node) getNodeStream(nodeID string) (*streaming.Stream, error) {
	if nodeID == node.ID {
		return node.nodeStream, nil
	}
	val, ok := node.nodeStreams.Load(nodeID)
	if !ok {
		s, err := streaming.NewStream(nodeStreamName(node.PoolName, nodeID), node.rdb, options.WithStreamLogger(node.logger))
		if err != nil {
			return nil, fmt.Errorf("getNodeStream: failed to create node stream %q: %w", nodeStreamName(node.PoolName, nodeID), err)
		}
		node.nodeStreams.Store(nodeID, s)
		return s, nil
	}
	return val.(*streaming.Stream), nil
}

// requeueAllJobs requeues all jobs from all local workers in parallel. It waits for all
// requeue operations to complete before returning. If any requeue operations fail, it
// collects all errors and returns them as a single error. This is typically called
// during node close to ensure no jobs are lost.
func (node *Node) requeueAllJobs(ctx context.Context) error {
	var wg sync.WaitGroup
	var errs []error
	var errLock sync.Mutex

	node.localWorkers.Range(func(key, value any) bool {
		wg.Add(1)
		pulse.Go(node.logger, func() {
			defer wg.Done()
			if err := value.(*Worker).requeueJobs(ctx); err != nil {
				errLock.Lock()
				errs = append(errs, err)
				errLock.Unlock()
			}
		})
		return true
	})
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("failed to requeue %d jobs: %v", len(errs), errs)
	}
	return nil
}

// cleanupPool removes the pool resources from Redis.
func (node *Node) cleanupPool(ctx context.Context) {
	for _, m := range node.maps() {
		if m != nil {
			if err := m.Destroy(ctx); err != nil {
				node.logger.Error(fmt.Errorf("cleanupPool: failed to destroy map: %w", err))
			}
		}
	}
	if err := node.poolStream.Destroy(ctx); err != nil {
		node.logger.Error(fmt.Errorf("cleanupPool: failed to destroy pool stream: %w", err))
	}
}

// cleanupNode closes the node resources.
func (node *Node) cleanupNode(ctx context.Context) {
	for _, m := range node.maps() {
		if m != nil {
			m.Close()
		}
	}
	if err := node.nodeStream.Destroy(ctx); err != nil {
		node.logger.Error(fmt.Errorf("cleanupNode: failed to destroy node stream: %w", err))
	}
}
