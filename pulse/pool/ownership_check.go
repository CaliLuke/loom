package pool

import (
	"context"
	"fmt"
	"time"
)

// CheckOwnership verifies that this local worker still runs key at epoch and
// that Redis confirms the owner, epoch, and active lease. ErrOwnershipLost
// means the supplied run cannot proceed, including while the worker is fenced.
// A confirmed mismatch queues a stop of the matching local run. The check
// returns without waiting for Stop, so Stop may safely join the calling work
// loop. Failed stops are logged and retried by the pool. A wrong epoch never
// stops a different local incarnation. Redis errors are returned without
// treating them as proof of an ownership transfer.
//
// This is a point-in-time check, not an atomic guard for external writes. The
// destination store must enforce epochs in the same transaction as each write.
// The job is available to its work loop as soon as Start receives it. Abort
// the planned operation on any error. Workers returned by AddWorker, Jobs, and Workers
// support the check. A remote PoolWorkers entry or a worker that is no longer
// local returns ErrOwnershipLost.
func (w *Worker) CheckOwnership(ctx context.Context, key string, epoch uint64) error {
	if w.handler == nil {
		if w.node == nil {
			return ErrOwnershipLost
		}
		value, ok := w.node.localWorkers.Load(w.ID)
		if !ok {
			return ErrOwnershipLost
		}
		w = value.(*Worker)
	}
	value, ok := w.jobs.Load(key)
	if !ok || value.(*Job).Epoch != epoch || !w.canCheckOwnership() {
		return ErrOwnershipLost
	}
	job := value.(*Job)
	if _, ok := w.rejectedJobs.Load(job); ok {
		return ErrOwnershipLost
	}
	owned, err := w.ownsJob(ctx, job)
	if err != nil {
		return err
	}
	if owned {
		current, ok := w.jobs.Load(key)
		if !ok || current != job || !w.canCheckOwnership() {
			return ErrOwnershipLost
		}
		return nil
	}
	w.rejectedJobs.Store(job, struct{}{})
	select {
	case w.ownershipWake <- struct{}{}:
	default:
	}
	return ErrOwnershipLost
}

func (w *Worker) canCheckOwnership() bool {
	return !w.IsStopped() && !w.leaseFenced.Load() && !w.leaseExpired(time.Now())
}

// stopRejectedJobs runs on a pool goroutine, never on the checking work loop.
// Recheck the local incarnation under both transition locks: a late reply for
// an older run must not stop a replacement that now uses the same key.
func (w *Worker) stopRejectedJobs() {
	w.jobLock.Lock()
	defer w.jobLock.Unlock()
	w.handlerLock.Lock()
	defer w.handlerLock.Unlock()
	w.rejectedJobs.Range(func(rejected, _ any) bool {
		key := rejected.(*Job).Key
		current, ok := w.jobs.Load(key)
		if !ok || current != rejected {
			w.rejectedJobs.Delete(rejected)
			return true
		}
		if err := w.stopTrackedHandler(key); err != nil {
			w.logger.Error(fmt.Errorf("stop superseded job: %w", err), "job", key)
			return true
		}
		w.jobs.Delete(key)
		delete(w.fencedJobs, key)
		w.rejectedJobs.Delete(rejected)
		return true
	})
}
