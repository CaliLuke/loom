package streaming

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

// TestSinkActivationRetainsOriginal checks both Redis acknowledgement modes.
func TestSinkActivationRetainsOriginal(t *testing.T) {
	for _, noAck := range []bool{false, true} {
		name := "ack"
		if noAck {
			name = "no-ack"
		}
		t.Run(name, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "activation-"+name)
			id, err := stream.Add(t.Context(), "original", []byte("payload"))
			require.NoError(t, err)
			opts := []options.Sink{options.WithSinkStartAtOldest(), options.WithSinkAckGracePeriod(time.Hour)}
			if noAck {
				opts = append(opts, options.WithSinkNoAck())
			}
			sink := newTestSink(t, stream, "activation", opts...)
			require.False(t, sink.deliveryStarted)
			// Membership and keepalive exist before delivery activation.
			requireConsumerPending(t, rdb, stream.key, sink.Name, sink.consumer, 0)
			require.Positive(t, sink.lastKeepAlive)
			ch := sink.Subscribe()
			event := receiveEvent(t, ch)
			require.Equal(t, id, event.ID)
			require.Equal(t, []byte("payload"), event.Payload)
			pending, err := rdb.XPending(t.Context(), stream.key, sink.Name).Result()
			require.NoError(t, err)
			if noAck {
				require.Zero(t, pending.Count)
			} else {
				require.EqualValues(t, 1, pending.Count)
				require.NoError(t, sink.Ack(t.Context(), event))
			}
			marker, err := stream.Add(t.Context(), "marker", nil)
			require.NoError(t, err)
			require.Equal(t, marker, receiveEvent(t, ch).ID)
		})
	}
}

// TestSinkActivationClaimsPending retains a foreign PEL entry until local admission.
func TestSinkActivationClaimsPending(t *testing.T) {
	srv := startTestServer(t)
	freezeIdleClock(srv)
	rdb := srv.Client
	stream := newTestStream(t, rdb, "activation-pending")
	id, err := stream.Add(t.Context(), "original", []byte("payload"))
	require.NoError(t, err)
	require.NoError(t, rdb.XGroupCreate(t.Context(), stream.key, "activation", "0").Err())
	messages, err := rdb.XReadGroup(t.Context(), &redis.XReadGroupArgs{
		Group: "activation", Consumer: "previous", Streams: []string{stream.key, ">"}, Count: 1, Block: -1,
	}).Result()
	require.NoError(t, err)
	require.Equal(t, id, messages[0].Messages[0].ID)
	checks := make(chan time.Time, 1)
	sink := newTestSinkWithRuntime(t, stream, "activation", sinkRuntime{idleChecks: checks}, options.WithSinkAckGracePeriod(claimTestAckGracePeriod))
	ageIdleEntries(srv)
	checks <- time.Now()
	require.False(t, sink.deliveryStarted)
	pending, err := rdb.XPendingExt(t.Context(), &redis.XPendingExtArgs{
		Stream: stream.key, Group: sink.Name, Start: "-", End: "+", Count: 1,
	}).Result()
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, "previous", pending[0].Consumer)
	ch := sink.Subscribe()
	event := awaitIdleClaim(t, rdb, sink.Name, checks, ch)
	require.Equal(t, id, event.ID)
	require.NoError(t, sink.Ack(t.Context(), event))
}

// TestSinkActivationConcurrentSubscribers checks that all admitted channels see delivery.
func TestSinkActivationConcurrentSubscribers(t *testing.T) {
	stream := newTestStream(t, startTestRedis(t), "activation-concurrent")
	sink := newTestSink(t, stream, "activation")
	start := make(chan struct{})
	channels := make([]<-chan *Event, 24)
	var wait sync.WaitGroup
	for i := range channels {
		wait.Go(func() {
			<-start
			channels[i] = sink.Subscribe()
		})
	}
	close(start)
	wait.Wait()
	id, err := stream.Add(t.Context(), "shared", nil)
	require.NoError(t, err)
	for _, ch := range channels {
		require.Equal(t, id, receiveEvent(t, ch).ID)
	}
	sink.Close(context.Background())
	for _, ch := range channels {
		assertSubscriberClosed(t, ch)
	}
}

// TestSinkActivationAfterClose does not consume the delivery startup token.
func TestSinkActivationAfterClose(t *testing.T) {
	stream := newTestStream(t, startTestRedis(t), "activation-closed")
	sink := newTestSink(t, stream, "activation")
	sink.Close(context.Background())
	assertSubscriberClosed(t, sink.Subscribe())
	require.False(t, sink.deliveryStarted)
}
