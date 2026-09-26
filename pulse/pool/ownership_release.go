package pool

import (
	"context"
	"fmt"
	"time"
)

// pendingRelease retains a stopped job until its epoch-checked Redis release
// succeeds. A lost reply never causes a local restart or an unfenced write.
type pendingRelease struct {
	job           *Job
	deletePayload bool
}

// finishRelease runs with jobLock held. A release that no longer matches is
// also finished: it must never delete another owner's state.
func (w *Worker) finishRelease(ctx context.Context, release pendingRelease) (bool, error) {
	if w.pendingReleases == nil {
		w.pendingReleases = make(map[string]pendingRelease)
	}
	w.pendingReleases[release.job.Key] = release
	released, err := w.releaseJob(ctx, release.job, release.deletePayload)
	if err != nil {
		return false, err
	}
	delete(w.pendingReleases, release.job.Key)
	return released, nil
}

func (w *Worker) retryReleases(ctx context.Context) {
	w.jobLock.Lock()
	defer w.jobLock.Unlock()
	for _, release := range w.pendingReleases {
		if _, err := w.finishRelease(ctx, release); err != nil {
			w.logger.Error(fmt.Errorf("retry stopped job release: %w", err))
		}
	}
}

// retryPendingReleases is independent of keepAlive: application callbacks and
// Redis release retries must never hold up renewal of a healthy worker's lease.
func (w *Worker) retryPendingReleases(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(w.workerTTL / 2)
	defer ticker.Stop()
	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			retryCtx, cancel := context.WithTimeout(ctx, w.workerTTL/4)
			w.retryReleases(retryCtx)
			cancel()
		}
	}
}
