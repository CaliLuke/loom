package pool

import (
	"fmt"
	"io"

	"github.com/CaliLuke/loom/pulse/rmap"
)

func (node *Node) maps() []*rmap.Map {
	return []*rmap.Map{
		node.nodeKeepAliveMap,
		node.nodeShutdownMap,
		node.workerMap,
		node.workerKeepAliveMap,
		node.workerCleanupMap,
		node.jobMap,
		node.jobPendingMap,
		node.jobPayloadMap,
		node.tickerMap,
	}
}

// Hash implements the Jump Consistent Hash algorithm.
// See https://arxiv.org/ftp/arxiv/papers/1406/1406.2294.pdf for details.
func (jh *jumpHash) Hash(key string, numBuckets int64) int64 {
	var b int64 = -1
	var j int64

	jh.mu.Lock()
	jh.h.Reset()
	_, err := io.WriteString(jh.h, key)
	sum := jh.h.Sum64()
	jh.mu.Unlock()
	if err != nil {
		panic(fmt.Errorf("jumpHash: write key: %w", err))
	}

	for j < numBuckets {
		b = j
		sum = sum*2862933555777941757 + 1
		j = int64(float64(b+1) * (float64(int64(1)<<31) / float64((sum>>33)+1)))
	}
	return b
}

// pendingEventKey computes the key of a pending event from a worker ID and a
// stream event ID.
func pendingEventKey(workerID, eventID string) string {
	return fmt.Sprintf("%s:%s", workerID, eventID)
}

// nodeKeepAliveMapName returns the name of the replicated map used to store the
// node keep-alive timestamps.
func nodeKeepAliveMapName(pool string) string {
	return fmt.Sprintf("%s:node-keepalive", pool)
}

// nodeShutdownMapName returns the name of the replicated map used to store the
// worker status.
func nodeShutdownMapName(pool string) string {
	return fmt.Sprintf("%s:shutdown", pool)
}

// workerMapName returns the name of the replicated map used to store the
// worker creation timestamps.
func workerMapName(pool string) string {
	return fmt.Sprintf("%s:workers", pool)
}

// workerKeepAliveMapName returns the name of the replicated map used to store the
// worker keep-alive timestamps.
func workerKeepAliveMapName(pool string) string {
	return fmt.Sprintf("%s:worker-keepalive", pool)
}

// workerCleanupMapName returns the name of the replicated map used to store the
// worker status.
func workerCleanupMapName(pool string) string {
	return fmt.Sprintf("%s:cleanup", pool)
}

// jobMapName returns the name of the replicated map used to store the
// jobs by worker ID.
func jobMapName(pool string) string {
	return fmt.Sprintf("%s:jobs", pool)
}

// jobPendingMapName returns the name of the replicated map used to store the
// pending jobs by job key.
func jobPendingMapName(poolName string) string {
	return poolName + ":pending-jobs"
}

// jobPayloadMapName returns the name of the replicated map used to store the
// job payloads by job key.
func jobPayloadMapName(pool string) string {
	return fmt.Sprintf("%s:job-payloads", pool)
}

// rmapContentKey returns the Redis hash key used by an rmap.
func rmapContentKey(name string) string {
	return fmt.Sprintf("map:%s:content", name)
}

// rmapUpdateChannel returns the Redis pubsub channel used by an rmap.
func rmapUpdateChannel(name string) string {
	return fmt.Sprintf("map:%s:updates", name)
}

// tickerMapName returns the name of the replicated map used to store ticker
// ticks.
func tickerMapName(pool string) string {
	return fmt.Sprintf("%s:tickers", pool)
}

// poolStreamName returns the name of the stream used by pool events.
func poolStreamName(pool string) string {
	return fmt.Sprintf("%s:pool", pool)
}

// nodeStreamName returns the name of the stream used by node events.
func nodeStreamName(pool, nodeID string) string {
	return fmt.Sprintf("%s:node:%s", pool, nodeID)
}
