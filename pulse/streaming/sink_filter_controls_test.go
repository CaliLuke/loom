package streaming

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type (
	filterAckFaultKey struct {
	}
	filterAckHook struct {
		calls atomic.Int32
	}
	filterAdmissionGate struct {
		key     string
		arrived chan struct{}
		release chan struct{}
		reads   atomic.Int32
	}
)

var errFilterAck = errors.New("injected filter acknowledgment reply failure")

func TestSinkFilterDispositionOwnership(t *testing.T) {
	cases := []struct {
		name        string
		accepted    bool
		malformed   bool
		reader      bool
		noAck       bool
		claimed     bool
		pattern     bool
		interrupted bool
	}{
		{name: "filtered"},
		{name: "pattern filtered", pattern: true},
		{name: "accepted interrupted fanout", accepted: true, interrupted: true},
		{name: "malformed", malformed: true},
		{name: "accepted without subscribers", accepted: true},
		{name: "reader rejection", reader: true},
		{name: "fresh noack", noAck: true},
		{name: "claimed noack", noAck: true, claimed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "filter-disposition")
			opts := []options.Sink{options.WithSinkStartAtOldest(), options.WithSinkTopic("orders")}
			if tc.pattern {
				opts = []options.Sink{options.WithSinkStartAtOldest(), options.WithSinkTopicPattern("^orders$")}
			}
			if tc.noAck {
				opts = append(opts, options.WithSinkNoAck())
			}
			sink := newTestSink(t, stream, "own", opts...)
			var id string
			var err error
			if tc.malformed {
				id, err = rdb.XAdd(t.Context(), &redis.XAddArgs{Stream: stream.key, Values: map[string]any{"invalid": "entry"}}).Result()
			} else {
				topic := "payments"
				if tc.accepted {
					topic = "orders"
				}
				id, err = stream.Add(t.Context(), "entry", nil, options.WithTopic(topic))
			}
			require.NoError(t, err)
			require.NoError(t, rdb.XGroupCreate(t.Context(), stream.key, "peer", "0").Err())
			require.NoError(t, rdb.XReadGroup(t.Context(), &redis.XReadGroupArgs{Group: "peer", Consumer: "peer", Streams: []string{stream.key, ">"}, Block: -1}).Err())
			messages, err := rdb.XReadGroup(t.Context(), &redis.XReadGroupArgs{
				Group: sink.Name, Consumer: "previous", Streams: []string{stream.key, ">"},
				Block: -1, NoAck: tc.noAck && !tc.claimed,
			}).Result()
			require.NoError(t, err)
			if tc.claimed {
				_, err := sink.claim(t.Context(), stream.Name, claimArgs{stream: stream.key, group: sink.Name, consumer: sinkConsumer(sink), minIdle: 0, start: "0-0"})
				require.NoError(t, err)
			} else {
				group := sink.Name
				if tc.reader {
					group = ""
				}
				stop := make(chan struct{})
				var subscribers []*eventSubscriber
				if tc.interrupted {
					close(stop)
					subscribers = []*eventSubscriber{newEventSubscriber(0)}
				}
				streamEvents(t.Context(), stream.Name, stream.key, group, messages[0].Messages, sink.eventFilter, subscribers, rdb, pulse.NoopLogger(), stop)
			}
			pending, err := rdb.XPending(t.Context(), stream.key, sink.Name).Result()
			require.NoError(t, err)
			var want int64
			if tc.accepted || tc.reader {
				want = 1
			}
			require.Equal(t, want, pending.Count)
			peer, err := rdb.XPending(t.Context(), stream.key, "peer").Result()
			require.NoError(t, err)
			require.EqualValues(t, 1, peer.Count)
			stored, err := rdb.XRange(t.Context(), stream.key, id, id).Result()
			require.NoError(t, err)
			require.Len(t, stored, 1)
		})
	}
}

func TestSinkFilterAcknowledgmentFailureAndReclaim(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(fmt.Sprintf("after=%t", after), func(t *testing.T) {
			rdb := startTestRedis(t)
			hook := &filterAckHook{}
			rdb.AddHook(hook)
			stream := newTestStream(t, rdb, "filter-ack-fault")
			sink := newTestSink(t, stream, "own", options.WithSinkStartAtOldest(), options.WithSinkTopic("orders"))
			_, err := stream.Add(t.Context(), "entry", nil, options.WithTopic("payments"))
			require.NoError(t, err)
			messages, err := rdb.XReadGroup(t.Context(), &redis.XReadGroupArgs{Group: sink.Name, Consumer: "previous", Streams: []string{stream.key, ">"}, Block: -1}).Result()
			require.NoError(t, err)
			ctx := context.WithValue(t.Context(), filterAckFaultKey{}, after)
			streamEvents(ctx, stream.Name, stream.key, sink.Name, messages[0].Messages, sink.eventFilter, nil, rdb, pulse.NoopLogger(), make(chan struct{}))
			pending, err := rdb.XPending(t.Context(), stream.key, sink.Name).Result()
			require.NoError(t, err)
			if after {
				require.Zero(t, pending.Count)
			} else {
				require.EqualValues(t, 1, pending.Count)
			}
			before := hook.calls.Load()
			_, err = sink.claim(t.Context(), stream.Name, claimArgs{stream: stream.key, group: sink.Name, consumer: sinkConsumer(sink), minIdle: 0, start: "0-0"})
			require.NoError(t, err)
			pending, err = rdb.XPending(t.Context(), stream.key, sink.Name).Result()
			require.NoError(t, err)
			require.Zero(t, pending.Count)
			if after {
				require.Equal(t, before, hook.calls.Load(), "already acknowledged entry cannot be reclaimed")
			} else {
				require.Greater(t, hook.calls.Load(), before)
			}
		})
	}
}

func TestSinkFilterConcurrentFirstJoins(t *testing.T) {
	for _, different := range []bool{false, true} {
		t.Run(fmt.Sprintf("different=%t", different), func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "filter-concurrent")
			gate := &filterAdmissionGate{key: consumersMapKey(stream), arrived: make(chan struct{}, 2), release: make(chan struct{})}
			rdb.AddHook(gate)
			type result struct {
				sink *Sink
				err  error
			}
			results := make(chan result, 2)
			for i := range 2 {
				topic := "orders"
				if different && i == 1 {
					topic = "payments"
				}
				go func() {
					sink, err := stream.NewSink(t.Context(), "shared", options.WithSinkTopic(topic))
					results <- result{sink: sink, err: err}
				}()
			}
			for range 2 {
				select {
				case <-gate.arrived:
				case <-time.After(5 * time.Second):
					close(gate.release)
					t.Fatal("first joins did not reach admission")
				}
			}
			close(gate.release)
			var admitted int
			for range 2 {
				got := <-results
				if got.sink != nil {
					admitted++
					t.Cleanup(func() {
						got.sink.Close(context.Background())
					})
				}
				if got.err != nil {
					require.True(t, different)
					require.ErrorContains(t, got.err, "filter")
				}
			}
			want := 2
			if different {
				want = 1
			}
			require.Equal(t, want, admitted)
			record, _, err := readSinkMembership(t.Context(), stream, "shared")
			require.NoError(t, err)
			require.Len(t, record.Members, want)
		})
	}
}

func (h *filterAckHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *filterAckHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() != "xack" {
			return next(ctx, cmd)
		}
		h.calls.Add(1)
		after, fault := ctx.Value(filterAckFaultKey{}).(bool)
		if !fault {
			return next(ctx, cmd)
		}
		if after {
			if err := next(ctx, cmd); err != nil {
				return err
			}
		}
		return errFilterAck
	}
}

func (h *filterAckHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *filterAdmissionGate) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *filterAdmissionGate) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if cmd.Name() == "hget" && fmt.Sprint(cmd.Args()[1]) == h.key && h.reads.Add(1) <= 2 {
			h.arrived <- struct{}{}
			select {
			case <-h.release:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return err
	}
}

func (h *filterAdmissionGate) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
