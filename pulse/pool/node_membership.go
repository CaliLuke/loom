package pool

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"
)

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
func (node *Node) updateNodeKeepAlive(ctx context.Context) {
	defer node.wg.Done()
	ticker := time.NewTicker(node.workerTTL / 2)
	defer ticker.Stop()

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
