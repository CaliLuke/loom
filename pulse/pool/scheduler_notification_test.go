package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
)

type gatedSchedulerSubscription struct {
	*rmap.Map
	entered chan struct{}
	release chan struct{}
}

func TestSchedulerConsumesCoalescedReset(t *testing.T) {
	for _, pendingChange := range []bool{false, true} {
		name := "reset-only"
		if pendingChange {
			name = "pending-change"
		}
		t.Run(name, func(t *testing.T) {
			mapped, err := rmap.Join(t.Context(), "scheduler-"+name, startTestRedis(t))
			require.NoError(t, err)
			gated := &gatedSchedulerSubscription{Map: mapped, entered: make(chan struct{}), release: make(chan struct{})}
			sched := &scheduler{name: name, jobMap: gated, logger: pulse.NoopLogger(), stop: make(chan struct{})}
			done := make(chan struct{})
			t.Cleanup(func() {
				select {
				case <-gated.release:
				default:
					close(gated.release)
				}
				sched.signalStop()
				mapped.Close()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("scheduler stop watcher did not exit")
				}
			})
			go func() {
				defer close(done)
				sched.handleStop()
			}()
			<-gated.entered
			if pendingChange {
				_, err := mapped.SetAndWait(t.Context(), "job", "pending")
				require.NoError(t, err)
			}
			require.NoError(t, mapped.Reset(t.Context()))
			// Reset-only starts empty; an observed barrier write confirms the reset
			// notification was applied before releasing the subscriber in both cases.
			_, err = mapped.SetAndWait(t.Context(), "barrier", "applied")
			require.NoError(t, err)
			close(gated.release)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Error("coalesced Reset did not stop the scheduler")
			}
			select {
			case <-sched.stop:
			default:
				t.Error("scheduler global stop was not signaled")
			}
		})
	}
}

func (g *gatedSchedulerSubscription) Subscribe() <-chan rmap.EventKind {
	c := g.Map.Subscribe()
	close(g.entered)
	<-g.release
	return c
}
