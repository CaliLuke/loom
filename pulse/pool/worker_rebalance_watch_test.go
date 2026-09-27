package pool

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
)

type rebalanceWatchHash struct {
	buckets chan int64
}

// Hash records the worker count but keeps the job on the first worker. This
// test isolates reconciliation scheduling from ownership and requeue scripts.
func (h *rebalanceWatchHash) Hash(_ string, buckets int64) int64 {
	select {
	case h.buckets <- buckets:
	default:
	}
	return 0
}

// TestWorkerRebalanceRevisitsMissedJoin replays the two model counterexamples
// after the membership-triggered pass has completed. No further worker-map
// mutation occurs: only a heartbeat or an already-routed job arrives.
func TestWorkerRebalanceRevisitsMissedJoin(t *testing.T) {
	for _, late := range []string{"heartbeat", "job"} {
		t.Run(late, func(t *testing.T) {
			rdb := startTestRedis(t)
			join := func(name string) *rmap.Map {
				m, err := rmap.Join(t.Context(), "rebalance-watch-"+late+"-"+name, rdb)
				require.NoError(t, err)
				t.Cleanup(m.Close)
				return m
			}
			h := &rebalanceWatchHash{buckets: make(chan int64, 16)}
			n := &Node{
				workerMap:          join("workers"),
				workerKeepAliveMap: join("heartbeats"),
				workerCleanupMap:   join("cleanup"),
				workerTTL:          2 * time.Second,
				h:                  h,
				logger:             pulse.NoopLogger(),
				stop:               make(chan struct{}),
			}
			w := &Worker{ID: "first", node: n, logger: n.logger}
			n.localWorkers.Store(w.ID, w)
			now := time.Now().UnixNano()
			for i, id := range []string{w.ID, "second"} {
				_, err := n.workerMap.SetAndWait(t.Context(), id, strconv.FormatInt(now+int64(i), 10))
				require.NoError(t, err)
				if id == "second" && late == "heartbeat" {
					continue
				}
				_, err = n.workerKeepAliveMap.SetAndWait(t.Context(), id, strconv.FormatInt(now, 10))
				require.NoError(t, err)
			}
			if late == "heartbeat" {
				w.jobs.Store("key", &Job{Key: "key"})
			}
			n.handleWorkerMapUpdate(t.Context())
			if late == "heartbeat" {
				require.EqualValues(t, 1, <-h.buckets)
			}
			// Start the actual watcher in the state just after that pass. The
			// registered membership is stable for the rest of the test.
			n.wg.Add(1)
			go n.watchWorkers(t.Context())
			t.Cleanup(func() {
				close(n.stop)
				n.wg.Wait()
			})
			if late == "heartbeat" {
				_, err := n.workerKeepAliveMap.SetAndWait(t.Context(), "second", strconv.FormatInt(time.Now().UnixNano(), 10))
				require.NoError(t, err)
			} else {
				w.jobs.Store("key", &Job{Key: "key"})
			}
			select {
			case count := <-h.buckets:
				require.EqualValues(t, 2, count)
			case <-time.After(1500 * time.Millisecond):
				t.Error("missed join was never reconciled without another membership event")
			}
		})
	}
}

// TestWorkerRebalanceWatcherStops ensures its periodic wakeups are owned by
// the node lifecycle even when no membership or heartbeat updates arrive.
func TestWorkerRebalanceWatcherStops(t *testing.T) {
	rdb := startTestRedis(t)
	m, err := rmap.Join(t.Context(), "rebalance-watch-stop", rdb)
	require.NoError(t, err)
	defer m.Close()
	n := &Node{workerMap: m, workerTTL: time.Hour, stop: make(chan struct{})}
	n.wg.Add(1)
	done := make(chan struct{})
	go func() {
		n.watchWorkers(context.Background())
		close(done)
	}()
	close(n.stop)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("watcher did not stop")
	}
}
