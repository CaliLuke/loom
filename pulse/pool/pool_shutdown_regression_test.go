package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
	"github.com/CaliLuke/loom/pulse/streaming"
)

func TestReturnDispatchStatusClosedChannelDoesNotPanic(t *testing.T) {
	node := &Node{
		PoolName: "pool",
		logger:   pulse.NoopLogger(),
	}
	node.pendingJobChannels.Store("1-0", closedErrorChannel())
	ev := &streaming.Event{EventName: evDispatchReturn, ID: "2-0", Payload: marshalAck(&ack{EventID: "1-0"})}

	require.NotPanics(t, func() {
		node.returnDispatchStatus(ev)
	})
}

func TestClientOnlyNodeCloseWithInitializedClosedChannelDoesNotPanic(t *testing.T) {
	node := &Node{
		ID:         "node-1",
		PoolName:   "pool",
		stop:       make(chan struct{}),
		closed:     make(chan struct{}),
		clientOnly: true,
		logger:     pulse.NoopLogger(),
	}

	require.NotPanics(t, func() {
		require.NoError(t, node.close(context.Background(), false))
	})
}

func TestDrainRequeueResultsKeepsFailuresAfterSuccess(t *testing.T) {
	jobs := map[string]*Job{
		"success": {Key: "success"},
		"failure": {Key: "failure"},
	}
	results := make(chan requeueResult, len(jobs))
	results <- requeueResult{key: "success"}
	results <- requeueResult{key: "failure", err: errors.New("boom")}
	close(results)

	remaining := drainRequeueResults(pulse.NoopLogger(), jobs, results, len(jobs), neverDone())

	require.Equal(t, map[string]*Job{"failure": jobs["failure"]}, remaining)
}

func TestSchedulerStopSignalStopsPlanningLoop(t *testing.T) {
	producer := &countingProducer{}
	ticks := make(chan time.Time)
	sched := &scheduler{
		name:     "pool:schedule",
		producer: producer,
		ticker:   &Ticker{C: ticks, wg: &sync.WaitGroup{}},
		logger:   pulse.NoopLogger(),
		stop:     make(chan struct{}),
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sched.scheduleJobs(context.Background(), sched.ticker, producer)
	}()

	sched.stopSchedule()

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, int32(0), producer.calls.Load())
}

func TestScheduleDoesNotPlanBeforeFirstOwnedTick(t *testing.T) {
	producer := &countingProducer{}
	ticks := make(chan time.Time)
	sched := &scheduler{
		name:     "pool:schedule",
		producer: producer,
		ticker:   &Ticker{C: ticks, wg: &sync.WaitGroup{}},
		logger:   pulse.NoopLogger(),
		stop:     make(chan struct{}),
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sched.scheduleJobs(context.Background(), sched.ticker, producer)
	}()
	t.Cleanup(func() {
		sched.stopSchedule()
		<-done
	})

	require.Never(t, func() bool {
		return producer.calls.Load() != 0
	}, 50*time.Millisecond, 5*time.Millisecond)
}

func TestReserveSchedulerRejectsClosedNode(t *testing.T) {
	node := &Node{
		PoolName: "pool",
		closing:  true,
	}
	sched := &scheduler{name: "pool:schedule", producer: &countingProducer{}}

	require.Error(t, node.reserveScheduler(sched))

	_, ok := node.localSchedulers.Load(sched.name)
	require.False(t, ok)
}

func TestReserveSchedulerRejectsDuplicateLocalSchedule(t *testing.T) {
	producer := &countingProducer{}
	node := &Node{PoolName: "pool"}
	existing := &scheduler{name: "pool:schedule", producer: producer}
	duplicate := &scheduler{name: "pool:schedule", producer: producer}

	require.NoError(t, node.reserveScheduler(existing))
	require.Error(t, node.reserveScheduler(duplicate))

	stored, ok := node.localSchedulers.Load(existing.name)
	require.True(t, ok)
	require.Same(t, existing, stored)
}

func TestStartSchedulerRejectsCloseAfterReservation(t *testing.T) {
	producer := &countingProducer{}
	node := &Node{PoolName: "pool"}
	sched := &scheduler{name: "pool:schedule", producer: producer, node: node}

	require.NoError(t, node.reserveScheduler(sched))
	node.lock.Lock()
	node.closing = true
	node.lock.Unlock()

	require.Error(t, node.startScheduler(sched))

	_, ok := node.localSchedulers.Load(sched.name)
	require.False(t, ok)
}

func TestSchedulerCloseBeforeJobMapAssignmentDoesNotConsumeCleanup(t *testing.T) {
	sched := &scheduler{}
	jobMap := newFakeSchedulerJobMap()

	sched.closeJobMap()
	sched.setJobMap(jobMap)
	sched.closeJobMap()

	require.True(t, jobMap.closed.Load())
}

func TestDrainRequeueResultsReturnsUnfinishedJobsOnTimeout(t *testing.T) {
	jobs := map[string]*Job{
		"success": {Key: "success"},
		"stuck":   {Key: "stuck"},
	}
	results := make(chan requeueResult, len(jobs))
	results <- requeueResult{key: "success"}
	done := make(chan struct{})
	close(done)

	remaining := drainRequeueResults(pulse.NoopLogger(), jobs, results, len(jobs), done)

	require.Equal(t, map[string]*Job{"stuck": jobs["stuck"]}, remaining)
}

func TestErrScheduleStopCleansUpLocalScheduler(t *testing.T) {
	producer := &stopProducer{}
	ticks := make(chan time.Time, 1)
	jobMap := newFakeSchedulerJobMap()
	node := &Node{PoolName: "pool"}
	sched := &scheduler{
		name:     "pool:schedule",
		producer: producer,
		node:     node,
		jobMap:   jobMap,
		logger:   pulse.NoopLogger(),
		stop:     make(chan struct{}),
	}
	node.localSchedulers.Store(sched.name, sched)
	done := make(chan struct{})
	go func() {
		defer close(done)
		sched.scheduleJobs(context.Background(), &Ticker{C: ticks}, producer)
	}()

	ticks <- time.Now()

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, int32(1), producer.calls.Load())
	require.Equal(t, int32(1), jobMap.resetCalls.Load())
	require.True(t, jobMap.closed.Load())
	_, ok := node.localSchedulers.Load(sched.name)
	require.False(t, ok)
	requireClosed(t, sched.stop)
}

func TestHandleStopUnsubscribesAfterResetEvent(t *testing.T) {
	jobMap := newFakeSchedulerJobMap()
	sched := &scheduler{
		name:   "pool:schedule",
		jobMap: jobMap,
		logger: pulse.NoopLogger(),
		stop:   make(chan struct{}),
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sched.handleStop()
	}()

	jobMap.events <- rmap.EventReset

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
	require.True(t, jobMap.unsubscribed.Load())
	require.True(t, jobMap.closed.Load())
	requireClosed(t, sched.stop)
}

type countingProducer struct {
	calls atomic.Int32
}

func (p *countingProducer) Name() string {
	return "schedule"
}

func (p *countingProducer) Plan() (*JobPlan, error) {
	p.calls.Add(1)
	return &JobPlan{}, nil
}

type stopProducer struct {
	calls atomic.Int32
}

func (p *stopProducer) Name() string {
	return "schedule"
}

func (p *stopProducer) Plan() (*JobPlan, error) {
	p.calls.Add(1)
	return nil, ErrScheduleStop
}

type fakeSchedulerJobMap struct {
	events       chan rmap.EventKind
	resetCalls   atomic.Int32
	unsubscribed atomic.Bool
	closed       atomic.Bool
}

func newFakeSchedulerJobMap() *fakeSchedulerJobMap {
	return &fakeSchedulerJobMap{events: make(chan rmap.EventKind, 1)}
}

func (m *fakeSchedulerJobMap) Keys() []string {
	return nil
}

func (m *fakeSchedulerJobMap) Set(ctx context.Context, key, value string) (string, error) {
	return "", nil
}

func (m *fakeSchedulerJobMap) Delete(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *fakeSchedulerJobMap) Reset(ctx context.Context) error {
	m.resetCalls.Add(1)
	return nil
}

func (m *fakeSchedulerJobMap) Subscribe() <-chan rmap.EventKind {
	return m.events
}

func (m *fakeSchedulerJobMap) Unsubscribe(c <-chan rmap.EventKind) {
	if c != m.events {
		panic(fmt.Sprintf("unexpected unsubscribe channel: %v", c))
	}
	m.unsubscribed.Store(true)
}

func (m *fakeSchedulerJobMap) Close() {
	m.closed.Store(true)
}

func closedErrorChannel() chan error {
	ch := make(chan error, 1)
	close(ch)
	return ch
}

func neverDone() <-chan struct{} {
	return make(chan struct{})
}

func requireClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	default:
		t.Fatal("channel is open")
	}
}
