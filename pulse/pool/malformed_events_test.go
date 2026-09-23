package pool

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// malformedPayload is a length prefix that exceeds the remaining input, so
// every pool decoder rejects it.
var malformedPayload = []byte{0xff, 0xff, 0xff, 0x7f, 'x'}

func TestMalformedEventsDoNotStopEventLoops(t *testing.T) {
	cases := []struct {
		name   string
		inject func(t *testing.T, ctx context.Context, node *Node, worker *Worker) []string
	}{
		{
			name: "worker stream malformed envelope",
			inject: func(t *testing.T, ctx context.Context, _ *Node, worker *Worker) []string {
				t.Helper()
				_, err := worker.stream.Add(ctx, evStartJob, malformedPayload)
				require.NoError(t, err)
				return nil
			},
		},
		{
			name: "worker stream malformed job payloads",
			inject: func(t *testing.T, ctx context.Context, node *Node, worker *Worker) []string {
				t.Helper()
				events := []string{evStartJob, evStopJob, evNotify}
				acked := make([]string, 0, len(events))
				for _, event := range events {
					id, err := worker.stream.Add(ctx, event, marshalEnvelope(node.ID, malformedPayload))
					require.NoError(t, err)
					acked = append(acked, id)
				}
				return acked
			},
		},
		{
			name: "pool stream malformed job",
			inject: func(t *testing.T, ctx context.Context, node *Node, _ *Worker) []string {
				t.Helper()
				_, err := node.poolStream.Add(ctx, evStartJob, malformedPayload)
				require.NoError(t, err)
				return nil
			},
		},
		{
			name: "node stream malformed acks and returns",
			inject: func(t *testing.T, ctx context.Context, node *Node, _ *Worker) []string {
				t.Helper()
				for _, payload := range [][]byte{malformedPayload, marshalEnvelope("worker", malformedPayload)} {
					_, err := node.nodeStream.Add(ctx, evAck, payload)
					require.NoError(t, err)
				}
				_, err := node.nodeStream.Add(ctx, evDispatchReturn, malformedPayload)
				require.NoError(t, err)
				return nil
			},
		},
	}
	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			ctx := t.Context()
			pool := "malformed-" + string(rune('a'+i))
			node := addTestNode(t, rdb, pool)
			handler := newRecordingHandler()
			worker, err := node.AddWorker(ctx, handler)
			require.NoError(t, err)

			acked := tc.inject(t, ctx, node, worker)

			dispatchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			require.NoError(t, node.DispatchJob(dispatchCtx, "valid", []byte("payload")))
			payload, ok := handler.startedPayload("valid")
			require.True(t, ok)
			require.Equal(t, []byte("payload"), payload)

			// A malformed worker event with a readable envelope is acked to
			// its node so the pool event it came from is not redelivered.
			nodeStreamKey := "pulse:stream:" + nodeStreamName(pool, node.ID)
			for _, id := range acked {
				require.Eventually(t, func() bool {
					return streamHasAck(rdb.XRange(ctx, nodeStreamKey, "-", "+").Val(), id)
				}, 5*time.Second, 5*time.Millisecond, "no ack for malformed worker event %s", id)
			}

			// The malformed pool event is acked in the sink group.
			poolStreamKey := "pulse:stream:" + poolStreamName(pool)
			require.Eventually(t, func() bool {
				pending, err := rdb.XPending(ctx, poolStreamKey, "events").Result()
				return err == nil && pending.Count == 0
			}, 5*time.Second, 5*time.Millisecond)
		})
	}
}

// streamHasAck reports whether messages contain an ack event whose payload
// names eventID.
func streamHasAck(messages []redis.XMessage, eventID string) bool {
	for _, msg := range messages {
		if msg.Values["n"] != evAck {
			continue
		}
		payload, ok := msg.Values["p"].(string)
		if ok && strings.Contains(payload, eventID) {
			return true
		}
	}
	return false
}
