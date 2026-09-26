package pool

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"time"
)

func (node *Node) ownersKey() string {
	return node.PoolName + ":owners"
}

func (node *Node) claimJob(ctx context.Context, worker string, job *Job) (uint64, error) {
	raw, err := luaClaimJob.Run(ctx, node.rdb, []string{
		node.ownersKey(), node.PoolName + ":owner-epochs",
		rmapContentKey(jobMapName(node.PoolName)), rmapUpdateChannel(jobMapName(node.PoolName)),
		rmapContentKey(jobPayloadMapName(node.PoolName)), rmapUpdateChannel(jobPayloadMapName(node.PoolName)),
		rmapContentKey(workerMapName(node.PoolName)), rmapContentKey(workerKeepAliveMapName(node.PoolName)),
		rmapContentKey(nodeKeepAliveMapName(node.PoolName)), node.PoolName + ":protocol",
	}, job.Key, worker, job.Payload, node.workerTTL.Milliseconds()).Result()
	if err != nil {
		return 0, fmt.Errorf("claim job %q: %w", job.Key, err)
	}
	status, epoch, err := parseDispatchClaim(raw)
	if err != nil || status == 0 {
		return 0, err
	}
	n, err := strconv.ParseUint(epoch, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("claim job %q: invalid epoch: %w", job.Key, err)
	}
	return n, nil
}

func (w *Worker) releaseJob(ctx context.Context, job *Job, deletePayload bool) (bool, error) {
	remove := "0"
	if deletePayload {
		remove = "1"
	}
	n := w.node
	result, err := luaReleaseJob.Run(ctx, n.rdb, []string{
		n.ownersKey(), rmapContentKey(jobMapName(n.PoolName)), rmapUpdateChannel(jobMapName(n.PoolName)),
		rmapContentKey(jobPayloadMapName(n.PoolName)), rmapUpdateChannel(jobPayloadMapName(n.PoolName)),
	}, job.Key, w.ID, strconv.FormatUint(job.Epoch, 10), remove).Int64()
	if err != nil {
		return false, fmt.Errorf("release job %q: %w", job.Key, err)
	}
	return result == 1, nil
}

func (node *Node) cleanupOwnedJobs(ctx context.Context, worker, token string) (int64, error) {
	p := node.PoolName
	now := time.Now()
	created := binary.LittleEndian.AppendUint64(nil, uint64(now.UnixNano()))
	return luaCleanupOwner.Run(ctx, node.rdb, []string{
		node.ownersKey(),
		rmapContentKey(workerCleanupMapName(p)), rmapUpdateChannel(workerCleanupMapName(p)),
		rmapContentKey(workerKeepAliveMapName(p)), rmapUpdateChannel(workerKeepAliveMapName(p)),
		rmapContentKey(workerMapName(p)), rmapUpdateChannel(workerMapName(p)),
		rmapContentKey(jobMapName(p)), rmapUpdateChannel(jobMapName(p)),
		rmapContentKey(jobPayloadMapName(p)), node.poolStream.Key(),
		"pulse:stream:" + workerStreamName(worker),
		rmapContentKey(jobPendingMapName(p)), rmapUpdateChannel(jobPendingMapName(p)),
	}, worker, token, node.workerTTL.Milliseconds(), node.ID, created,
		strconv.FormatInt(now.UnixNano(), 10), node.poolStream.MaxLen).Int64()
}

func joinOwnershipProtocol(ctx context.Context, node *Node) error {
	p := node.PoolName
	return luaJoinProtocol.Run(ctx, node.rdb, []string{
		p + ":protocol", rmapContentKey(nodeKeepAliveMapName(p)), rmapUpdateChannel(nodeKeepAliveMapName(p)),
		p + ":protocol-version", node.ownersKey(), p + ":owner-epochs",
		rmapContentKey(jobMapName(p)), rmapContentKey(workerKeepAliveMapName(p)),
	}, node.ID, node.workerTTL.Milliseconds()).Err()
}

// Health reports incompatible live pool members or a failed protocol check.
// Routing and job claims fail closed until every live member uses protocol 2.
// It does not report application handler health.
func (node *Node) Health(ctx context.Context) error {
	bad, err := luaCheckProtocol.Run(ctx, node.rdb, []string{
		rmapContentKey(nodeKeepAliveMapName(node.PoolName)), node.PoolName + ":protocol",
	}, node.workerTTL.Milliseconds()).Text()
	if err != nil {
		return fmt.Errorf("pool protocol health: %w", err)
	}
	if bad != "" {
		return fmt.Errorf("incompatible pool node %q: close all older nodes before upgrading", bad)
	}
	return nil
}

func (node *Node) workerHeartbeat(ctx context.Context, id string) error {
	return luaOwnerHeartbeat.Run(ctx, node.rdb, []string{
		rmapContentKey(workerKeepAliveMapName(node.PoolName)), rmapUpdateChannel(workerKeepAliveMapName(node.PoolName)),
		rmapContentKey(workerMapName(node.PoolName)),
	}, id).Err()
}

// initialWorkerHeartbeat retains AddWorker's guarantee that its local replica
// has observed the initial lease before it returns.
func (node *Node) initialWorkerHeartbeat(ctx context.Context, id string) error {
	updates := node.workerKeepAliveMap.Subscribe()
	defer node.workerKeepAliveMap.Unsubscribe(updates)
	if err := node.workerHeartbeat(ctx, id); err != nil {
		return err
	}
	for {
		if _, ok := node.workerKeepAliveMap.Get(id); ok {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-updates:
			if !ok {
				return fmt.Errorf("worker keep-alive map closed")
			}
		}
	}
}
