package pool

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

	// A close may have removed the replica index after a failed release.
	// The authoritative owners still make every abandoned worker discoverable.
	owners, err := node.rdb.HVals(ctx, node.ownersKey()).Result()
	if err != nil {
		node.logger.Error(fmt.Errorf("read job owners for cleanup: %w", err))
		return
	}
	for _, owner := range owners {
		if i := strings.LastIndexByte(owner, ':'); i >= 0 {
			workersToCheck[owner[:i]] = struct{}{}
		}
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

// requeueOrphanedPayloads detects payloads without authoritative ownership
// and requeues them after a grace period. The script rechecks ownership and
// payload bytes atomically, so a claim or stop after this snapshot wins.
func (node *Node) requeueOrphanedPayloads(ctx context.Context) {
	owners, err := node.rdb.HKeys(ctx, node.ownersKey()).Result()
	if err != nil {
		node.logger.Error(fmt.Errorf("read owners for orphan recovery: %w", err))
		return
	}
	existingJobs := make(map[string]struct{}, len(owners))
	for _, key := range owners {
		existingJobs[key] = struct{}{}
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
		// is acked the owner record holds the key, and once it is lost the guard
		// no longer blocks.
		queued, err := node.claimRequeue(ctx, job)
		if err != nil {
			node.logger.Error(fmt.Errorf("requeueOrphanedPayloads: failed to requeue orphaned job: %w", err), "key", key)
			continue
		}
		if !queued {
			node.logger.Debug("requeueOrphanedPayloads: snapshot or pending start prevents requeue", "key", key)
			continue
		}

		node.orphanedPayloads.Delete(key)
		node.logger.Info("requeueOrphanedPayloads: requeued orphaned job", "key", key, "grace", grace)
	}
}

// claimRequeue adds a start event for job with a guard that names it
// (luaClaimRequeue). It returns false with no error when the key already has
// a guard whose start event is in flight, has an owner, or its durable payload
// no longer matches the observed job.
func (node *Node) claimRequeue(ctx context.Context, job *Job) (bool, error) {
	now := time.Now()
	raw, err := luaClaimRequeue.Run(ctx, node.rdb, []string{
		rmapContentKey(jobPendingMapName(node.PoolName)),
		rmapUpdateChannel(jobPendingMapName(node.PoolName)),
		node.poolStream.Key(),
		node.ownersKey(), rmapContentKey(jobPayloadMapName(node.PoolName)),
	}, job.Key, strconv.FormatInt(now.UnixNano(), 10), node.poolStream.MaxLen, marshalJob(job),
		poolSinkName, now.UnixMilli(), pendingEventTTL.Milliseconds(), job.Payload).Result()
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
	case dispatchAlreadyPending, requeueObsolete:
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

	status, err := node.cleanupOwnedJobs(ctx, workerID, lockToken)
	if err != nil {
		node.logger.Error(fmt.Errorf("cleanup worker %q: %w", workerID, err))
		return
	}
	node.logger.Debug("worker cleanup", "worker", workerID, "status", status)
}
