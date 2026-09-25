package pool

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

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
		// A guarded start (a rebalance or an earlier sweep) that is still in
		// flight may yet start the job, so the sweep waits for it: once it
		// is acked the job map lists the key, and once it is lost the guard
		// no longer blocks.
		queued, err := node.claimRequeue(ctx, job)
		if err != nil {
			node.logger.Error(fmt.Errorf("requeueOrphanedPayloads: failed to requeue orphaned job: %w", err), "key", key)
			continue
		}
		if !queued {
			node.logger.Debug("requeueOrphanedPayloads: start event in flight, not requeuing", "key", key)
			continue
		}

		node.orphanedPayloads.Delete(key)
		node.logger.Info("requeueOrphanedPayloads: requeued orphaned job", "key", key, "grace", grace)
	}
}

// claimRequeue adds a start event for job with a guard that names it
// (luaClaimRequeue). It returns false with no error when the key already has
// a guard whose start event is in flight: that event may still start the job,
// so a second start is not added.
func (node *Node) claimRequeue(ctx context.Context, job *Job) (bool, error) {
	now := time.Now()
	raw, err := luaClaimRequeue.Run(ctx, node.rdb, []string{
		rmapContentKey(jobPendingMapName(node.PoolName)),
		rmapUpdateChannel(jobPendingMapName(node.PoolName)),
		node.poolStream.Key(),
	}, job.Key, strconv.FormatInt(now.UnixNano(), 10), node.poolStream.MaxLen, marshalJob(job),
		poolSinkName, now.UnixMilli(), pendingEventTTL.Milliseconds()).Result()
	if err != nil {
		return false, fmt.Errorf("failed to requeue job %q: %w", job.Key, err)
	}
	status, value, err := parseDispatchClaim(raw)
	if err != nil {
		return false, fmt.Errorf("failed to parse requeue result for job %q: %w", job.Key, err)
	}
	switch status {
	case dispatchClaimed:
		return true, nil
	case dispatchAlreadyPending:
		return false, nil
	case dispatchMalformedPending:
		return false, fmt.Errorf("malformed pending guard for job %q: %q", job.Key, value)
	default:
		return false, fmt.Errorf("unexpected requeue status %d for job %q", status, job.Key)
	}
}

// requeueInFlight reports whether claimRequeue would refuse key because its
// guard names a start event that is in flight (luaRequeueInFlight).
func (node *Node) requeueInFlight(ctx context.Context, key string) (bool, error) {
	now := time.Now()
	res, err := luaRequeueInFlight.Run(ctx, node.rdb, []string{
		rmapContentKey(jobPendingMapName(node.PoolName)),
		node.poolStream.Key(),
	}, key, poolSinkName, now.UnixMilli(), pendingEventTTL.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("failed to check the guard of job %q: %w", key, err)
	}
	return res == 1, nil
}

// cleanupWorker requeues the jobs assigned to the worker and deletes it from
// the pool.
func (node *Node) cleanupWorker(ctx context.Context, workerID string) {
	// Try to acquire or clear stale cleanup lock
	lockToken, ok := node.acquireCleanupLock(ctx, workerID)
	if !ok {
		return
	}

	// Get the worker's jobs
	keys, ok := node.jobMap.GetValues(workerID)
	if !ok || len(keys) == 0 {
		// Worker has no jobs, just delete it
		if err := node.deleteWorker(ctx, workerID, lockToken); err != nil {
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
	if err := node.deleteWorker(ctx, workerID, lockToken); err != nil {
		node.logger.Error(fmt.Errorf("cleanupWorkerJobs: failed to delete worker: %w", err), "worker", workerID)
	}
}
