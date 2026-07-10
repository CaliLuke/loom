package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
)

type (
	// JobComputeFunc is the function called by the scheduler to compute jobs.
	// It returns the list of jobs to start and job keys to stop.
	JobProducer interface {
		// Name returns the name of the producer. Schedule calls Plan on
		// only one of the producers with identical names across all
		// nodes.
		Name() string
		// Plan computes the list of jobs to start and job keys to stop.
		// Returning ErrScheduleStop indicates that the recurring
		// schedule should be stopped.
		Plan() (*JobPlan, error)
	}

	// JobPlan represents a list of jobs to start and job keys to stop.
	JobPlan struct {
		// Jobs to start.
		Start []*JobParam
		// Job keys to stop.
		Stop []string
		// StopAll indicates that all jobs not in Jobs should be
		// stopped.  Stop is ignored if StopAll is true.
		StopAll bool
	}

	// JobParam represents a job to start.
	JobParam struct {
		// Key is the job key.
		Key string
		// Payload is the job payload.
		Payload []byte
	}

	// scheduler implements a scheduler that starts and stops jobs on a
	// recurring basis.
	scheduler struct {
		// name is the name of the scheduler.
		name string
		// interval is the interval at which the scheduler runs.
		interval time.Duration
		// producer is the job producer.
		producer JobProducer
		// node is the node running the scheduler.
		node *Node
		// ticker is the ticker used to run the scheduler.
		ticker *Ticker
		// jobMap is the map of jobs keyed by job key.
		jobMap schedulerJobMap
		// jobMapLock synchronizes map ownership during Schedule and Close.
		jobMapLock sync.Mutex
		// logger is the logger used by the scheduler.
		logger pulse.Logger
		// stop is closed when the local scheduler should stop planning.
		stop chan struct{}
		// stopOnce makes scheduler stop idempotent.
		stopOnce sync.Once
		// closeOnce makes scheduler map cleanup idempotent.
		closeOnce sync.Once
	}

	schedulerJobMap interface {
		Keys() []string
		Set(ctx context.Context, key, value string) (string, error)
		Delete(ctx context.Context, key string) (string, error)
		Reset(ctx context.Context) error
		Subscribe() <-chan rmap.EventKind
		Unsubscribe(c <-chan rmap.EventKind)
		Close()
	}
)

// ErrScheduleStop is returned by JobProducer.Plan to indicate that the
// corresponding schedule should be stopped.
var ErrScheduleStop = fmt.Errorf("stop")

// Schedule calls the producer Plan method on the given interval and starts and
// stops jobs accordingly. The schedule stops when the producer Plan method
// returns ErrScheduleStop. Plan is called on only one of the nodes that
// scheduled the same producer.
func (node *Node) Schedule(ctx context.Context, producer JobProducer, interval time.Duration) error {
	name := node.PoolName + ":" + producer.Name()
	sched := &scheduler{
		name:     name,
		interval: interval,
		producer: producer,
		node:     node,
		logger:   node.logger,
		stop:     make(chan struct{}),
	}
	if err := node.reserveScheduler(sched); err != nil {
		return err
	}
	jobMap, err := rmap.Join(ctx, name, node.rdb, rmap.WithLogger(node.logger))
	if err != nil {
		sched.unregister()
		return fmt.Errorf("failed to join job map %s: %w", name, err)
	}
	sched.setJobMap(jobMap)
	ticker, err := node.NewTicker(ctx, producer.Name(), interval)
	if err != nil {
		sched.unregister()
		sched.closeJobMap()
		return fmt.Errorf("failed to create ticker %s: %w", name, err)
	}
	sched.ticker = ticker
	if err := node.startScheduler(sched); err != nil {
		ticker.Close()
		sched.closeJobMap()
		return err
	}
	pulse.Go(sched.logger, func() {
		defer node.wg.Done()
		sched.scheduleJobs(ctx, ticker, producer)
	})
	pulse.Go(sched.logger, func() {
		defer node.wg.Done()
		sched.handleStop()
	})
	return nil
}

func (node *Node) reserveScheduler(sched *scheduler) error {
	node.lock.Lock()
	defer node.lock.Unlock()
	if node.closing {
		return fmt.Errorf("Schedule: pool %q is closed", node.PoolName)
	}
	if _, loaded := node.localSchedulers.LoadOrStore(sched.name, sched); loaded {
		return fmt.Errorf("Schedule: producer %q is already scheduled on pool %q", sched.producer.Name(), node.PoolName)
	}
	return nil
}

func (node *Node) startScheduler(sched *scheduler) error {
	node.lock.Lock()
	defer node.lock.Unlock()
	if node.closing {
		sched.unregister()
		return fmt.Errorf("Schedule: pool %q is closed", node.PoolName)
	}
	current, ok := node.localSchedulers.Load(sched.name)
	if !ok || current != sched {
		return fmt.Errorf("Schedule: producer %q is not reserved on pool %q", sched.producer.Name(), node.PoolName)
	}
	node.wg.Add(2)
	return nil
}

// scheduleJobs calls Plan on ticks and starts and stops jobs as needed.
func (sched *scheduler) scheduleJobs(ctx context.Context, ticker *Ticker, producer JobProducer) {
	for {
		select {
		case <-sched.stop:
			return
		case _, ok := <-ticker.C:
			if !ok {
				return
			}
		}
		plan, err := producer.Plan()
		if err != nil {
			if errors.Is(err, ErrScheduleStop) {
				if err := sched.jobMap.Reset(ctx); err != nil {
					sched.logger.Error(err, "failed to reset job map", "scheduler", sched.name)
					continue
				}
				sched.stopScheduleGlobally()
				return
			}
			sched.logger.Error(err, "failed to compute schedule", "scheduler", sched.name)
			continue
		}
		sched.logger.Info("scheduling jobs", "scheduler", sched.name, "start", len(plan.Start), "stop", len(plan.Stop), "stopAll", plan.StopAll)
		sched.startJobs(ctx, plan.Start)
		sched.stopJobs(ctx, plan)
	}
}

// startJobs dispatches the given jobs.
func (sched *scheduler) startJobs(ctx context.Context, jobs []*JobParam) {
	for _, job := range jobs {
		err := sched.node.DispatchJob(ctx, job.Key, job.Payload)
		if err != nil {
			sched.logger.Error(fmt.Errorf("failed to dispatch job: %w", err), "job", job.Key)
			continue
		}
		if _, err := sched.jobMap.Set(ctx, job.Key, time.Now().String()); err != nil {
			sched.logger.Error(fmt.Errorf("failed to store job: %w", err), "job", job.Key)
			continue
		}
	}
}

// stopJobs stops jobs according to the given schedule.
func (sched *scheduler) stopJobs(ctx context.Context, plan *JobPlan) {
	var toStop []string
	if plan.StopAll {
		toStop = sched.jobMap.Keys()
		for _, j := range plan.Start {
			for i, k := range toStop {
				if k == j.Key {
					toStop = append(toStop[:i], toStop[i+1:]...)
					break
				}
			}
		}
	} else {
		toStop = plan.Stop
	}
	for _, key := range toStop {
		err := sched.node.StopJob(ctx, key)
		if err != nil {
			sched.logger.Error(fmt.Errorf("failed to stop job: %w", err), "job", key)
			continue
		}
		if _, err := sched.jobMap.Delete(ctx, key); err != nil {
			sched.logger.Error(fmt.Errorf("failed to delete job: %w", err), "job", key)
		}
	}
}

// handleStop handles the scheduler stop signal.
func (sched *scheduler) handleStop() {
	ch := sched.jobMap.Subscribe()
	if ch == nil {
		sched.stopSchedule()
		return
	}
	defer sched.jobMap.Unsubscribe(ch)
	for {
		select {
		case <-sched.stop:
			return
		case ev, ok := <-ch:
			if !ok {
				sched.stopSchedule()
				return
			}
			if ev != rmap.EventReset {
				continue
			}
			sched.logger.Info("stopping scheduler", "scheduler", sched.name)
			sched.stopScheduleGlobally()
			return
		}
	}
}

func (sched *scheduler) stopSchedule() {
	sched.signalStop()
	if sched.ticker != nil {
		sched.ticker.Close()
	}
	sched.unregister()
	sched.closeJobMap()
}

func (sched *scheduler) stopScheduleGlobally() {
	sched.signalStop()
	if sched.ticker != nil {
		sched.ticker.Stop()
	}
	sched.unregister()
	sched.closeJobMap()
}

func (sched *scheduler) signalStop() {
	sched.stopOnce.Do(func() {
		close(sched.stop)
	})
}

func (sched *scheduler) unregister() {
	if sched.node != nil {
		current, ok := sched.node.localSchedulers.Load(sched.name)
		if ok && current == sched {
			sched.node.localSchedulers.Delete(sched.name)
		}
	}
}

func (sched *scheduler) closeJobMap() {
	sched.jobMapLock.Lock()
	defer sched.jobMapLock.Unlock()
	if sched.jobMap == nil {
		return
	}
	sched.closeOnce.Do(func() {
		sched.jobMap.Close()
	})
}

func (sched *scheduler) setJobMap(jobMap schedulerJobMap) {
	sched.jobMapLock.Lock()
	defer sched.jobMapLock.Unlock()
	sched.jobMap = jobMap
}
