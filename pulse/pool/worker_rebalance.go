package pool

import (
	"context"
	"errors"
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
		if err != nil || !queued {
			// Restart the job locally and track it again so future
			// close/shutdown can still requeue it.
			if err := w.restartJob(ctx, job); err != nil {
				w.logger.Error(fmt.Errorf("rebalance: failed to restart job: %w", err), "job", key)
			}
			retry = true
		}
	}
	if retry {
		w.retryRebalance()
	}
}

// retryRebalance runs rebalance again after ackGracePeriod, with the active
// workers at that time. rebalance calls it when it left a job on this worker
// because a start of the job was in flight or the requeue failed. Rebalance
// otherwise runs only when the worker map changes, so without the retry the
// job would stay on a worker that does not own it, where StopJob and
// NotifyWorker do not reach it. At most one retry is pending per worker, and
// it ends when the worker stops.
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
// When Stop fails the job stays tracked. When the job map update fails, the
// handler is started again and the job stays tracked; if that start fails
// too, the job is forgotten and the error says so.
func (w *Worker) releaseTrackedJob(ctx context.Context, key string) (bool, error) {
	w.jobLock.Lock()
	defer w.jobLock.Unlock()
	job, ok := w.jobs.Load(key)
	if !ok {
		return false, nil
	}
	if err := w.handler.Stop(key); err != nil {
		return false, err
	}
	if _, _, err := w.jobsMap.RemoveValues(ctx, w.ID, key); err != nil {
		err = fmt.Errorf("failed to remove job %q from jobs map: %w", key, err)
		if startErr := w.handler.Start(job.(*Job)); startErr != nil {
			w.jobs.Delete(key)
			return false, errors.Join(err, fmt.Errorf("failed to restart job %q: %w", key, startErr))
		}
		return false, err
	}
	w.jobs.Delete(key)
	return true, nil
}

// restartJob restarts a job locally after its requeue failed. It adds the
// job back to the worker's job map entry first, which releaseTrackedJob
// removed. It does nothing if the worker is stopped or already tracks the
// key again. When it fails, the job runs nowhere and has no job map entry,
// so the orphan sweep requeues it from its payload.
func (w *Worker) restartJob(ctx context.Context, job *Job) error {
	w.jobLock.Lock()
	defer w.jobLock.Unlock()
	if w.IsStopped() {
		return fmt.Errorf("worker %q stopped", w.ID)
	}
	if _, ok := w.jobs.Load(job.Key); ok {
		return nil
	}
	if _, err := w.jobsMap.AppendUniqueValues(ctx, w.ID, job.Key); err != nil {
		return fmt.Errorf("failed to add job %q to jobs map: %w", job.Key, err)
	}
	if err := w.handler.Start(job); err != nil {
		if _, _, rerr := w.jobsMap.RemoveValues(ctx, w.ID, job.Key); rerr != nil {
			return errors.Join(err, fmt.Errorf("failed to remove job %q from jobs map: %w", job.Key, rerr))
		}
		return err
	}
	w.jobs.Store(job.Key, job)
	return nil
}
