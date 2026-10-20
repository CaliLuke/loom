package streaming

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

type membershipReplyHook struct {
	key    string
	failed atomic.Bool
}

func TestSinkFilterAddStreamConflictRetainsRegistration(t *testing.T) {
	rdb := startTestRedis(t)
	first := newTestStream(t, rdb, "filter-first")
	second := newTestStream(t, rdb, "filter-second")
	sink := newTestSink(t, first, "shared", options.WithSinkTopic("orders"))
	peer := newTestSink(t, second, "shared", options.WithSinkTopic("payments"))
	require.ErrorContains(t, sink.AddStream(t.Context(), second), "filter")
	sink.lock.Lock()
	registrations := append([]sinkRegistration(nil), sink.streams...)
	sink.lock.Unlock()
	require.Len(t, registrations, 1)
	require.Equal(t, first, registrations[0].stream)
	for _, pair := range []struct {
		stream *Stream
		sink   *Sink
	}{{first, sink}, {second, peer}} {
		record, _, err := readSinkMembership(t.Context(), pair.stream, "shared")
		require.NoError(t, err)
		require.Equal(t, []string{sinkConsumer(pair.sink)}, record.memberNames())
	}
}

func TestSinkFilterCoordinatedDestroyResetsIdentity(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "filter-reset")
	sink := newTestSink(t, stream, "shared", options.WithSinkTopic("orders"))
	sink.Close(t.Context())
	require.NoError(t, stream.Destroy(t.Context()))
	newTestSink(t, stream, "shared", options.WithSinkTopic("payments"))
}

func TestSinkFilterAdmissionLostReplyRollsBack(t *testing.T) {
	for _, add := range []bool{false, true} {
		t.Run(fmt.Sprintf("add=%t", add), func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "filter-lost-reply")
			var sink *Sink
			if add {
				first := newTestStream(t, rdb, "filter-lost-first")
				sink = newTestSink(t, first, "shared", options.WithSinkTopic("orders"))
			}
			hook := &membershipReplyHook{key: consumersMapKey(stream)}
			rdb.AddHook(hook)
			var err error
			if add {
				err = sink.AddStream(t.Context(), stream)
			} else {
				sink, err = stream.NewSink(t.Context(), "shared", options.WithSinkTopic("orders"))
				require.Nil(t, sink)
			}
			require.ErrorIs(t, err, errInjectedCommand)
			require.True(t, hook.failed.Load())
			record, _, err := readSinkMembership(t.Context(), stream, "shared")
			require.NoError(t, err)
			require.NotNil(t, record)
			require.Empty(t, record.Members, "uncertain committed admission must be retired")
			if add {
				require.NoError(t, sink.AddStream(t.Context(), stream))
			} else {
				newTestSink(t, stream, "shared", options.WithSinkTopic("orders"))
			}
		})
	}
}

func TestSinkFilterRecordAdmission(t *testing.T) {
	for _, raw := range []string{
		`[]`,
		`{"version":2,"filter":{"kind":"none","value":""},"members":null}`,
		`{"version":2,"filter":{"kind":"exact","value":""},"members":[]}`,
		`{"version":2,"filter":{"kind":"unknown","value":"x"},"members":[]}`,
		`{"version":2,"filter":{"kind":"none","value":""},"members":[{"name":"x","registeredAt":1},{"name":"x","registeredAt":1}]}`,
		`{"version":2,"filter":{"kind":"none","value":""},"members":[],"unknown":true}`,
		`{"version":2,"filter":{"kind":"none"},"members":[{"name":"x"}]}`,
		`{"version":2,"filter":{"kind":"none"},"members":[{"name":"x","registeredAt":0}]}`,
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := decodeSinkMembership(raw)
			require.ErrorContains(t, err, "migration")
		})
	}
}

func TestSinkFilterVersionOneMembershipRequiresMigration(t *testing.T) {
	_, err := decodeSinkMembership(`{"version":1,"filter":{"kind":"none","value":""},"members":[{"name":"old","registeredAt":1}]}`)
	require.ErrorContains(t, err, "migration")
}

func TestSinkFilterExplicitNoneRecordAdmission(t *testing.T) {
	raw := `{"version":2,"filter":{"kind":"none"},"members":[]}`
	record, err := decodeSinkMembership(raw)
	require.NoError(t, err)
	require.Equal(t, &sinkFilter{Kind: "none"}, record.Filter)
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "filter-explicit-none")
	require.NoError(t, rdb.HSet(t.Context(), consumersMapKey(stream), "shared", raw).Err())
	newTestSink(t, stream, "shared")
}

func TestSinkFilterCleanupPreservesJoiningMember(t *testing.T) {
	rdb := startTestRedis(t)
	rdb.AddHook(interleaveHook{})
	stream := newTestStream(t, rdb, "filter-joining")
	live := newTestSink(t, stream, "shared", options.WithSinkAckGracePeriod(time.Hour))
	var cleanupErr error
	var before, after []string
	var native []redis.XInfoConsumer
	ran := false
	ctx := interleaveAfter(t.Context(), commandArgs{"xgroup", "create"}, func() {
		ran = true
		record, _, err := readSinkMembership(t.Context(), stream, "shared")
		if err != nil {
			cleanupErr = err
			return
		}
		before = record.memberNames()
		native, err = rdb.XInfoConsumers(t.Context(), stream.key, "shared").Result()
		if err != nil {
			cleanupErr = err
			return
		}
		live.lock.Lock()
		cleanupErr = live.deleteStreamStaleConsumers(t.Context(), stream)
		live.lock.Unlock()
		if cleanupErr != nil {
			return
		}
		record, _, cleanupErr = readSinkMembership(t.Context(), stream, "shared")
		if cleanupErr == nil {
			after = record.memberNames()
		}
	})
	joining, err := stream.NewSink(ctx, "shared", options.WithSinkAckGracePeriod(time.Hour))
	if joining != nil {
		t.Cleanup(func() {
			joining.Close(t.Context())
		})
	}
	require.NoError(t, err)
	require.True(t, ran)
	require.NoError(t, cleanupErr)
	require.Len(t, native, 1, "joining member has no native consumer at cleanup")
	require.Equal(t, sinkConsumer(live), native[0].Name)
	require.ElementsMatch(t, []string{sinkConsumer(live), sinkConsumer(joining)}, before)
	require.ElementsMatch(t, before, after, "cleanup must preserve publication before consumer/heartbeat creation")
}

func TestSinkFilterRetiresMemberWithoutKeepAlive(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "filter-orphan")
	sink := newTestSink(t, stream, "shared", options.WithSinkAckGracePeriod(time.Hour))
	cm := sink.consumersMap[stream.Name]
	record, raw, err := readSinkMembership(t.Context(), stream, sink.Name)
	require.NoError(t, err)
	record.Members = append(record.Members,
		sinkMember{Name: "old", RegisteredAt: time.Now().Add(-3 * time.Hour).UnixNano()},
		sinkMember{Name: "joining", RegisteredAt: time.Now().UnixNano()},
	)
	encoded, err := json.Marshal(record)
	require.NoError(t, err)
	_, _, updated, err := cm.replica.TestAndSetEx(t.Context(), sink.Name, raw, string(encoded))
	require.NoError(t, err)
	require.True(t, updated)
	sink.lock.Lock()
	err = sink.deleteStreamStaleConsumers(t.Context(), stream)
	sink.lock.Unlock()
	require.NoError(t, err)
	record, _, err = readSinkMembership(t.Context(), stream, sink.Name)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{sinkConsumer(sink), "joining"}, record.memberNames())
}

func (h *membershipReplyHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *membershipReplyHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if err != nil || (cmd.Name() != "eval" && cmd.Name() != "evalsha") {
			return err
		}
		for _, arg := range cmd.Args()[3:] {
			if fmt.Sprint(arg) == h.key && h.failed.CompareAndSwap(false, true) {
				return errInjectedCommand
			}
		}
		return err
	}
}

func (h *membershipReplyHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
