package pool

import (
	"context"
	"fmt"
	"time"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/streaming"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

// watchShutdown monitors the pool shutdown map and initiates node shutdown when updated.
func (node *Node) watchShutdown(ctx context.Context) {
	defer node.wg.Done()
	updates := node.nodeShutdownMap.Subscribe()
	defer node.nodeShutdownMap.Unsubscribe(updates)
	for {
		select {
		case <-node.stop:
			return
		case _, ok := <-updates:
			if !ok {
				return
			}
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
	closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), node.workerShutdownTTL)
	defer cancel()
	if err := node.close(closeCtx, true); err != nil {
		node.logger.Error(fmt.Errorf("handleShutdown: failed to close node: %w", err))
	}

	node.lock.Lock()
	node.shutdown = true
	node.lock.Unlock()
	node.logger.Info("shutdown", "requested-by", requestingNode)
}

// processInactiveNodes periodically checks for inactive nodes and destroys their streams.
func (node *Node) processInactiveNodes(ctx context.Context) {
	defer node.wg.Done()
	ticker := time.NewTicker(node.workerTTL)
	defer ticker.Stop()

	for {
		select {
		case <-node.stop:
			return
		case <-ticker.C:
			node.cleanupInactiveNodes(ctx)
		}
	}
}

// cleanupInactiveNodes checks for inactive nodes, destroys their streams and
// removes them from the keep-alive map.
func (node *Node) cleanupInactiveNodes(ctx context.Context) {
	nodeMap := node.nodeKeepAliveMap.Map()
	for nodeID, lastSeen := range nodeMap {
		if nodeID == node.ID || node.isWithinTTL(lastSeen, node.workerTTL) {
			continue
		}

		node.logger.Info("cleaning up inactive node", "node", nodeID)

		// Clean up node's stream
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
