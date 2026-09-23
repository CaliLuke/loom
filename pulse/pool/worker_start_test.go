package pool

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// gatedHandler is a JobHandler whose first Start blocks until release is
// closed, so the start event stays unacked past the ack grace period.
type gatedHandler struct {
	starts  atomic.Int32
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

// Start implements JobHandler.
func (h *gatedHandler) Start(*Job) error {
	h.starts.Add(1)
	h.once.Do(func() {
		close(h.entered)
		<-h.release
	})
	return nil
}

// Stop implements JobHandler.
func (h *gatedHandler) Stop(string) error {
	return nil
}

// TestRedeliveredStartEventStartsJobOnce replays the double_asis TLC trace
// (pulse/pool/tla/cfg/double_asis.cfg, 6 states, violates
// NoDoubleStartSameWorker on main at 967f4fbe):
//
//  1. a start event for k1 is added to the pool stream,
//  2. routeWorkerEvent sends it to w1,
//  3. the event stays unacked past ackGracePeriod,
//  4. XAUTOCLAIM redelivers it and routeWorkerEvent sends a copy to w1,
//  5. w1 handles both copies.
//
// startJob must call handler.Start once and ack both deliveries.
func TestRedeliveredStartEventStartsJobOnce(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	const pool = "start-once"
	node := addTestNode(t, rdb, pool, WithAckGracePeriod(300*time.Millisecond))
	handler := &gatedHandler{entered: make(chan struct{}), release: make(chan struct{})}
	worker, err := node.AddWorker(ctx, handler)
	require.NoError(t, err)

	// Steps 1-2: the first delivery reaches w1, whose handler blocks.
	job := &Job{Key: "k1", Payload: []byte("p"), CreatedAt: time.Now(), NodeID: node.ID}
	_, err = node.poolStream.Add(ctx, evStartJob, marshalJob(job))
	require.NoError(t, err)
	select {
	case <-handler.entered:
	case <-time.After(10 * time.Second):
		t.Fatal("handler.Start was not called")
	}

	// Steps 3-4: the sink redelivers the unacked event to w1.
	workerStreamKey := "pulse:stream:" + workerStreamName(worker.ID)
	var startIDs []string
	require.Eventually(t, func() bool {
		startIDs = startIDs[:0]
		for _, msg := range rdb.XRange(ctx, workerStreamKey, "-", "+").Val() {
			if msg.Values["n"] == evStartJob {
				startIDs = append(startIDs, msg.ID)
			}
		}
		return len(startIDs) >= 2
	}, 10*time.Second, 5*time.Millisecond)

	// Step 5: w1 handles both deliveries.
	close(handler.release)
	nodeStreamKey := "pulse:stream:" + nodeStreamName(pool, node.ID)
	for _, id := range startIDs {
		require.Eventually(t, func() bool {
			return streamHasAck(rdb.XRange(ctx, nodeStreamKey, "-", "+").Val(), id)
		}, 10*time.Second, 5*time.Millisecond, "delivery %s was not acked", id)
	}
	require.Equal(t, int32(1), handler.starts.Load(), "handler.Start calls")
	require.Len(t, worker.Jobs(), 1)
}
