package pool

import (
	"context"
	"fmt"
	"sync"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/streaming"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

// deleteWorker removes a remote worker from the pool deleting the worker stream.
func (node *Node) deleteWorker(ctx context.Context, id string) error {
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
	if node.nodeStream == nil {
		return
	}
	if err := node.nodeStream.Destroy(ctx); err != nil {
		node.logger.Error(fmt.Errorf("cleanupNode: failed to destroy node stream: %w", err))
	}
}
