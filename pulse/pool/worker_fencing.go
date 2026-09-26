package pool

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// recordLease uses the request start, not the reply time: a delayed successful
// reply cannot extend a lease beyond the Redis expiry it acknowledges.
func (w *Worker) recordLease(start time.Time) {
	w.leaseLock.Lock()
	w.leaseStart = start
	w.leaseLock.Unlock()
}

func (w *Worker) leaseExpired(now time.Time) bool {
	w.leaseLock.Lock()
	start := w.leaseStart
	w.leaseLock.Unlock()
	return start.IsZero() || now.Sub(start) >= w.workerTTL-w.workerTTL/4
}

// monitorLease is independent of heartbeat I/O. Handler callbacks must finish
// within the fence margin; an unbounded process pause needs store-side epochs.
func (w *Worker) monitorLease() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.workerTTL / 8)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if w.leaseExpired(time.Now()) {
				w.fenceHandlers()
			}
		case <-w.done:
			return
		}
	}
}

// resumeLease performs Redis validation independently of the fencing monitor.
// Neither blocked verification nor other ownership I/O can delay local stops.
func (w *Worker) resumeLease(ctx context.Context) {
	defer w.wg.Done()
	ticker := time.NewTicker(w.workerTTL / 8)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if w.leaseFenced.Load() && !w.leaseExpired(time.Now()) {
				checkCtx, cancel := context.WithTimeout(ctx, w.workerTTL/8)
				w.updateLeaseState(checkCtx, time.Now())
				cancel()
			}
		case <-w.done:
			return
		}
	}
}

// updateLeaseState retains stopped jobs until Redis confirms whether the same
// owner and epoch can resume. fencedJobs is protected by handlerLock: false
// means Stop still needs to succeed, true means the handler has stopped.
func (w *Worker) updateLeaseState(ctx context.Context, now time.Time) {
	if w.leaseExpired(now) {
		w.fenceHandlers()
		return
	}
	if w.leaseFenced.Load() {
		w.resumeHandlers(ctx)
	}
}

// fenceHandlers has no path to Redis, including when renewal races expiry.
func (w *Worker) fenceHandlers() {
	w.handlerLock.Lock()
	defer w.handlerLock.Unlock()
	w.leaseFenced.Store(true)
	if !w.IsStopped() {
		w.pauseHandlers(true)
	}
}

// pauseHandlers stops every tracked handler when the lease first expires and
// retries callbacks that failed on an earlier pass. The caller holds handlerLock.
func (w *Worker) pauseHandlers(expired bool) {
	if w.fencedJobs == nil {
		w.fencedJobs = make(map[string]bool)
	}
	if expired {
		w.jobs.Range(func(key, _ any) bool {
			if _, ok := w.fencedJobs[key.(string)]; !ok {
				w.fencedJobs[key.(string)] = false
			}
			return true
		})
	}
	for key, stopped := range w.fencedJobs {
		if !stopped {
			if err := w.handler.Stop(key); err != nil {
				w.logger.Error(fmt.Errorf("fence job: %w", err), "job", key)
				continue
			}
			w.fencedJobs[key] = true
		}
	}
}

// resumeHandlers serializes against other ownership transitions but allows
// the fencing monitor to stop running handlers while Redis validation waits.
func (w *Worker) resumeHandlers(ctx context.Context) {
	w.jobLock.Lock()
	defer w.jobLock.Unlock()
	w.handlerLock.Lock()
	if w.IsStopped() || w.leaseExpired(time.Now()) {
		w.handlerLock.Unlock()
		return
	}
	w.pauseHandlers(false)
	var jobs []*Job
	for key, stopped := range w.fencedJobs {
		if stopped {
			if value, ok := w.jobs.Load(key); ok {
				jobs = append(jobs, value.(*Job))
			}
		}
	}
	w.handlerLock.Unlock()
	for _, job := range jobs {
		w.resumeJob(ctx, job)
	}
	w.handlerLock.Lock()
	defer w.handlerLock.Unlock()
	if len(w.fencedJobs) == 0 && !w.leaseExpired(time.Now()) && !w.IsStopped() {
		w.leaseFenced.Store(false)
		select {
		case w.leaseWake <- struct{}{}:
		default:
		}
	}
}

func (w *Worker) resumeJob(ctx context.Context, job *Job) {
	owned, err := w.ownsJob(ctx, job)
	if err != nil {
		w.logger.Error(err, "job", job.Key)
		return
	}
	w.handlerLock.Lock()
	defer w.handlerLock.Unlock()
	if !owned {
		w.jobs.Delete(job.Key)
		delete(w.fencedJobs, job.Key)
		return
	}
	if w.IsStopped() || w.leaseExpired(time.Now()) {
		return
	}
	if err := w.handler.Start(job); err != nil {
		w.logger.Error(fmt.Errorf("resume fenced job: %w", err), "job", job.Key)
		return
	}
	delete(w.fencedJobs, job.Key)
}

func (w *Worker) ownsJob(ctx context.Context, job *Job) (bool, error) {
	n := w.node
	owned, err := luaCheckOwner.Run(ctx, n.rdb, []string{
		n.ownersKey(), rmapContentKey(workerMapName(n.PoolName)),
		rmapContentKey(workerKeepAliveMapName(n.PoolName)),
		rmapContentKey(nodeKeepAliveMapName(n.PoolName)), n.PoolName + ":protocol",
	}, job.Key, w.ID, strconv.FormatUint(job.Epoch, 10), w.workerTTL.Milliseconds()).Int64()
	if err != nil {
		return false, fmt.Errorf("check job ownership: %w", err)
	}
	return owned == 1, nil
}

// stopTrackedHandler avoids a second Stop for jobs already paused by fencing.
// The caller holds handlerLock and removes fencedJobs only after forgetting a job.
func (w *Worker) stopTrackedHandler(key string) error {
	if w.fencedJobs[key] {
		return nil
	}
	return w.handler.Stop(key)
}

// takeJob stops and forgets one local job without holding the handler lock
// across the subsequent Redis release. The caller holds jobLock.
func (w *Worker) takeJob(key string) (*Job, error) {
	w.handlerLock.Lock()
	defer w.handlerLock.Unlock()
	value, ok := w.jobs.Load(key)
	if !ok {
		return nil, nil
	}
	if err := w.stopTrackedHandler(key); err != nil {
		return nil, err
	}
	w.jobs.Delete(key)
	delete(w.fencedJobs, key)
	return value.(*Job), nil
}
