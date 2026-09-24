package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming"
)

// TestDispatchReturnBeforeRegistration covers a dispatch return that reaches
// the dispatching node before dispatchJob has registered its channel for the
// start event: the result must still reach the dispatcher.
func TestDispatchReturnBeforeRegistration(t *testing.T) {
	cases := []struct {
		name    string
		errText string
	}{
		{name: "success"},
		{name: "start error", errText: "start failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			node := addTestNode(t, rdb, "early-return")
			node.returnDispatchStatus(&streaming.Event{
				ID:        "2-0",
				EventName: evDispatchReturn,
				Payload:   marshalAck(&ack{EventID: "1-0", Error: tc.errText}),
			})

			cherr := node.registerDispatch("1-0")
			select {
			case err := <-cherr:
				if tc.errText == "" {
					require.NoError(t, err)
				} else {
					require.EqualError(t, err, tc.errText)
				}
			case <-time.After(time.Second):
				t.Fatal("early dispatch return was lost")
			}
		})
	}
}

// TestRequeueRegistersNoDispatchReturn covers the dispatch return of a
// requeued start event. No dispatcher waits for it, so requeueJob must not
// leave an entry in pendingJobChannels, and the return is parked on the node
// that the job names until it expires.
func TestRequeueRegistersNoDispatchReturn(t *testing.T) {
	cases := []struct {
		name       string
		foreign    bool
		wantParked int
	}{
		{name: "return to requeuing node", wantParked: 1},
		{name: "return to other node", foreign: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			ctx := t.Context()
			node := addTestNode(t, rdb, "requeue-return")
			handler := newRecordingHandler()
			worker, err := node.AddWorker(ctx, handler)
			require.NoError(t, err)
			nodeID := node.ID
			if tc.foreign {
				nodeID = "other-node"
			}

			job := &Job{Key: "job-1", Payload: []byte("payload-1"), CreatedAt: time.Now(), NodeID: nodeID}
			require.NoError(t, worker.requeueJob(ctx, job))
			require.Eventually(t, func() bool {
				_, ok := handler.startedPayload("job-1")
				return ok
			}, 10*time.Second, 5*time.Millisecond)
			require.Eventually(t, func() bool {
				return countEarlyReturns(node) == tc.wantParked
			}, 10*time.Second, 5*time.Millisecond)

			require.Equal(t, 0, countPendingJobChannels(node))
		})
	}
}

// countPendingJobChannels returns the number of entries in the pending
// dispatch channels of node.
func countPendingJobChannels(node *Node) int {
	count := 0
	node.pendingJobChannels.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// countEarlyReturns returns the number of dispatch returns node parked.
func countEarlyReturns(node *Node) int {
	node.dispatchReturnsLock.Lock()
	defer node.dispatchReturnsLock.Unlock()
	return len(node.earlyReturns)
}
