package pool

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming"
)

type (
	// ackOnAddHook makes a fake worker ack each event added to its stream
	// before the XADD returns to the router, the earliest an ack can reach
	// the node.
	ackOnAddHook struct {
		streamKey string
		ack       atomic.Pointer[func(ctx context.Context, eventID string)]
	}
)

func (h *ackOnAddHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *ackOnAddHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if err := next(ctx, cmd); err != nil {
			return err
		}
		fn := h.ack.Load()
		if fn == nil || cmd.Name() != "xadd" || len(cmd.Args()) < 2 || cmd.Args()[1] != h.streamKey {
			return nil
		}
		add, ok := cmd.(*redis.StringCmd)
		if ok && add.Val() != "" {
			(*fn)(ctx, add.Val())
		}
		return nil
	}
}

func (h *ackOnAddHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

// TestWorkerAckBeforeRegistration covers a worker ack that reaches the node
// before routeWorkerEvent has registered the pending event: the pool event
// must be acked at once, not redelivered after the ack grace period, and a
// later duplicate of the ack must not match again.
func TestWorkerAckBeforeRegistration(t *testing.T) {
	cases := []struct {
		name  string
		event func(t *testing.T, ctx context.Context, node *Node)
	}{
		{
			name: "dispatched start",
			event: func(t *testing.T, ctx context.Context, node *Node) {
				t.Helper()
				dispatchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				defer cancel()
				require.NoError(t, node.DispatchJob(dispatchCtx, "k1", []byte("payload")))
			},
		},
		{
			name: "stop",
			event: func(t *testing.T, ctx context.Context, node *Node) {
				t.Helper()
				require.NoError(t, node.StopJob(ctx, "k1"))
			},
		},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pool := "early-ack-" + strconv.Itoa(i)
			workerID := "fake-worker"
			rdb := startTestRedis(t)
			hook := &ackOnAddHook{streamKey: "pulse:stream:" + workerStreamName(workerID)}
			rdb.AddHook(hook)
			ctx := t.Context()
			node := addTestNode(t, rdb, pool, WithWorkerTTL(time.Minute), WithAckGracePeriod(time.Minute))
			addFakeWorker(t, node, workerID)
			var acked atomic.Value
			ackNow := func(ctx context.Context, eventID string) {
				acked.Store(eventID)
				node.processNodeEvent(ctx, workerAckEvent(workerID, eventID))
			}
			hook.ack.Store(&ackNow)

			tc.event(t, ctx, node)

			poolStreamKey := "pulse:stream:" + poolStreamName(pool)
			require.Eventually(t, func() bool {
				pending, err := rdb.XPending(ctx, poolStreamKey, poolSinkName).Result()
				return err == nil && pending.Count == 0
			}, 3*time.Second, 5*time.Millisecond, "pool event not acked")
			require.Equal(t, 0, countPendingEvents(node))
			require.Equal(t, 0, countEarlyAcks(node))

			// A duplicate ack is parked, not matched a second time.
			eventID, ok := acked.Load().(string)
			require.True(t, ok)
			node.processNodeEvent(ctx, workerAckEvent(workerID, eventID))
			require.Equal(t, 1, countEarlyAcks(node))
		})
	}
}

// TestWorkerAckParking covers acks that no registration claims: parked acks
// are dropped once older than the ack grace period, and a routing to a
// missing worker stream registers nothing.
func TestWorkerAckParking(t *testing.T) {
	t.Run("stale parked acks dropped", func(t *testing.T) {
		rdb := startTestRedis(t)
		node := addTestNode(t, rdb, "stale-ack")
		node.ackWorkerEvent(t.Context(), workerAckEvent("w1", "2-0"))
		node.pendingEventsLock.Lock()
		stale := node.earlyAcks[pendingEventKey("w1", "2-0")]
		stale.at = time.Now().Add(-2 * node.ackGracePeriod)
		node.earlyAcks[pendingEventKey("w1", "2-0")] = stale
		node.pendingEventsLock.Unlock()

		node.ackWorkerEvent(t.Context(), workerAckEvent("w1", "3-0"))
		node.pendingEventsLock.Lock()
		_, staleKept := node.earlyAcks[pendingEventKey("w1", "2-0")]
		_, freshKept := node.earlyAcks[pendingEventKey("w1", "3-0")]
		node.pendingEventsLock.Unlock()
		require.False(t, staleKept, "stale parked ack kept")
		require.True(t, freshKept, "fresh parked ack dropped")
	})
	t.Run("missing worker stream", func(t *testing.T) {
		rdb := startTestRedis(t)
		node := addTestNode(t, rdb, "missing-stream", WithWorkerTTL(time.Minute))
		registerFakeWorker(t, node, "fake-worker")
		id := strconv.FormatInt(time.Now().UnixMilli(), 10) + "-0"
		ev := &streaming.Event{ID: id, EventName: evStopJob, Payload: marshalJobKey("k1")}
		require.ErrorContains(t, node.routeWorkerEvent(t.Context(), ev), "does not exist")
		require.Equal(t, 0, countPendingEvents(node))
	})
}

// addFakeWorker registers worker id in the pool and creates its stream
// without starting a worker loop.
func addFakeWorker(t *testing.T, node *Node, id string) {
	t.Helper()
	registerFakeWorker(t, node, id)
	stream, err := node.getWorkerStream(id)
	require.NoError(t, err)
	_, err = stream.Add(t.Context(), evInit, marshalEnvelope(node.ID, []byte(id)))
	require.NoError(t, err)
}

// registerFakeWorker adds worker id to the worker and keep-alive maps.
func registerFakeWorker(t *testing.T, node *Node, id string) {
	t.Helper()
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err := node.workerMap.SetAndWait(t.Context(), id, now)
	require.NoError(t, err)
	_, err = node.workerKeepAliveMap.SetAndWait(t.Context(), id, now)
	require.NoError(t, err)
}

// workerAckEvent returns the node stream event of worker workerID acking
// worker event eventID.
func workerAckEvent(workerID, eventID string) *streaming.Event {
	return &streaming.Event{
		ID:        "0-1",
		EventName: evAck,
		Payload:   marshalEnvelope(workerID, marshalAck(&ack{EventID: eventID})),
	}
}

// countPendingEvents returns the number of pending events of node.
func countPendingEvents(node *Node) int {
	count := 0
	node.pendingEvents.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// countEarlyAcks returns the number of worker acks node parked.
func countEarlyAcks(node *Node) int {
	node.pendingEventsLock.Lock()
	defer node.pendingEventsLock.Unlock()
	return len(node.earlyAcks)
}
