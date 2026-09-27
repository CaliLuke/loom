package pool

import (
	"context"
	"fmt"
	"time"

	"github.com/CaliLuke/loom/pulse/pulse"
)

// rebalance rebalances the jobs handled by the worker.
func (w *Worker) rebalance(ctx context.Context, activeWorkers []string) {
	w.logger.Debug("rebalance")
	rebalanced := make(map[string]*Job)
	w.jobs.Range(func(key, value any) bool {
		job := value.(*Job)
		wid := activeWorkers[w.node.h.Hash(job.Key, int64(len(activeWorkers)))]
		if wid != w.ID {
			rebalanced[job.Key] = job
		}
		return true
	})
	total := len(rebalanced)
	if total == 0 {
		w.logger.Debug("rebalance: no jobs to rebalance")
		return
	}
	retry := false
	for key, job := range rebalanced {
		// A start of the key that may still start it refuses the requeue:
		// the start that placed the job here until its ack deletes the
		// guard, or another requeue. Keep the job running and retry, rather
		// than stop it and start it again.
		inFlight, err := w.node.requeueInFlight(ctx, key)
		if err != nil {
			w.logger.Error(fmt.Errorf("rebalance: %w", err), "job", key)
			retry = true
			continue
		}
		if inFlight {
			w.logger.Debug("rebalance: a start event of the job is in flight, retrying later", "job", key)
			retry = true
			continue
		}
		// Release the key before the start event is added: a start routed
		// back to this worker then adds the key to the job map again after
		// the release, not before it.
		released, err := w.releaseTrackedJob(ctx, key)
		if err != nil {
			w.logger.Error(fmt.Errorf("rebalance: failed to release job: %w", err), "job", key)
			retry = true
			continue
		}
		if !released {
			// Another transition (stop, eviction) already took the key.
			continue
		}
		w.logger.Debug("stopped job", "job", key)
		// The start event carries a guard, so the orphan sweep does not add a
		// second start while this one may still start the job, and requeues
		// the job once this one is lost (issue #416).
		queued, err := w.node.claimRequeue(ctx, job)
		if err != nil {
			w.logger.Error(fmt.Errorf("rebalance: %w", err), "job", key)
		} else if !queued {
			w.logger.Info("rebalance: a start event of the job is in flight, keeping it", "job", key)
		}
		// A failed reply does not prove that Redis rejected the start. Never
		// restart locally after this call: the queued event may already run
		// elsewhere. If no event was added, the payload and missing job-map
		// entry let the orphan sweep recover it after its grace period.
	}
	if retry {
		w.retryRebalance()
	}
}

// retryRebalance runs rebalance again after ackGracePeriod, with the active
// workers at that time. rebalance calls it when it left a job on this worker
// because a start of the job was in flight or releasing the job failed. Rebalance
// also runs on membership changes and periodic reconciliation, but this retry
// lets a pending guard be revisited after its own grace period. At most one
// retry is pending per worker, and it ends when the worker stops.
func (w *Worker) retryRebalance() {
	if !w.rebalanceRetry.CompareAndSwap(false, true) {
		return
	}
	w.lock.Lock()
	if w.stopped {
		w.lock.Unlock()
		w.rebalanceRetry.Store(false)
		return
	}
	// stop sets stopped under lock before it waits, so this Add happens
	// before that Wait.
	w.wg.Add(1)
	w.lock.Unlock()
	pulse.Go(w.logger, func() {
		defer w.wg.Done()
		timer := time.NewTimer(w.node.ackGracePeriod)
		defer timer.Stop()
		select {
		case <-w.done:
			return
		case <-timer.C:
		}
		w.rebalanceRetry.Store(false)
		if w.node.IsClosed() {
			return
		}
		active := w.node.activeWorkers()
		if len(active) == 0 {
			return
		}
		w.rebalance(w.runtimeCtx, active)
	})
}

// releaseTrackedJob stops the handler of a tracked job, forgets it locally
// and removes it from the worker's job map entry. The payload stays, so if
// the start event that moves the job is lost (trimmed, or dropped as stale),
// the orphan sweep requeues the job (issue #416). A job map entry of a live
// worker would hide the job from the orphan sweep and from cleanup.
//
// It returns false with no error if the worker no longer tracks the key.
// When Stop fails the job stays tracked. A failed Redis release stays pending
// for retry by a dedicated loop. The stopped handler is never restarted.
func (w *Worker) releaseTrackedJob(ctx context.Context, key string) (bool, error) {
	w.jobLock.Lock()
	defer w.jobLock.Unlock()
	job, err := w.takeJob(key)
	if err != nil || job == nil {
		return false, err
	}
	// Never restart after an ambiguous release reply: another worker may
	// already own the key. The ownership check fences all durable writes.
	released, err := w.finishRelease(ctx, pendingRelease{job: job})
	if err != nil {
		return false, err
	}
	return released, nil
}
