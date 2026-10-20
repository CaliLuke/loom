package streaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/streaming/options"
)

func TestSinkGroupFilterAdmission(t *testing.T) {
	cases := []struct {
		name     string
		first    []options.Sink
		second   []options.Sink
		conflict bool
	}{
		{name: "none"},
		{
			name:   "same exact",
			first:  []options.Sink{options.WithSinkTopic("orders")},
			second: []options.Sink{options.WithSinkTopic("orders")},
		},
		{
			name:   "same pattern",
			first:  []options.Sink{options.WithSinkTopicPattern("^orders$")},
			second: []options.Sink{options.WithSinkTopicPattern("^orders$")},
		},
		{
			name:     "different exact",
			first:    []options.Sink{options.WithSinkTopic("orders")},
			second:   []options.Sink{options.WithSinkTopic("payments")},
			conflict: true,
		},
		{
			name:     "none versus exact",
			second:   []options.Sink{options.WithSinkTopic("orders")},
			conflict: true,
		},
		{
			name:     "exact versus pattern",
			first:    []options.Sink{options.WithSinkTopic("orders")},
			second:   []options.Sink{options.WithSinkTopicPattern("^orders$")},
			conflict: true,
		},
		{
			name:     "pattern spelling",
			first:    []options.Sink{options.WithSinkTopicPattern("orders")},
			second:   []options.Sink{options.WithSinkTopicPattern("(?:orders)")},
			conflict: true,
		},
		{
			name:   "effective exact precedence",
			first:  []options.Sink{options.WithSinkTopic("orders"), options.WithSinkTopicPattern("ignored")},
			second: []options.Sink{options.WithSinkTopic("orders")},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "filter-admission")
			newTestSink(t, stream, "shared", tc.first...)
			second, err := stream.NewSink(t.Context(), "shared", tc.second...)
			if second != nil {
				t.Cleanup(func() {
					second.Close(t.Context())
				})
			}
			if tc.conflict {
				require.ErrorContains(t, err, "filter")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSinkGroupFilterSurvivesEmptyMembership(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "filter-empty")
	first := newTestSink(t, stream, "shared", options.WithSinkTopic("orders"))
	require.NoError(t, first.RemoveStream(t.Context(), stream))
	second, err := stream.NewSink(t.Context(), "shared", options.WithSinkTopic("payments"))
	if second != nil {
		t.Cleanup(func() {
			second.Close(t.Context())
		})
	}
	require.ErrorContains(t, err, "filter")
}

func TestSinkGroupFilterHasNoIndependentExpiry(t *testing.T) {
	for _, mode := range ttlModes {
		t.Run(mode.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream, err := NewStream("filter-ttl", rdb, mode.opt)
			require.NoError(t, err)
			newTestSink(t, stream, "shared", options.WithSinkTopic("orders"))
			ttl, err := rdb.TTL(t.Context(), consumersMapKey(stream)).Result()
			require.NoError(t, err)
			require.Equal(t, -time.Nanosecond, ttl, "group identity must survive independent stream/map activity")
			assertTTLBetween(t, rdb, stream.key, 0, testTTL)
		})
	}
}

func TestSinkGroupFilterIndependentGroups(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "filter-independent")
	newTestSink(t, stream, "orders", options.WithSinkTopic("orders"))
	newTestSink(t, stream, "payments", options.WithSinkTopic("payments"))
}

func TestSinkGroupFilterRejectsExpiringMapForNewGroup(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "filter-expiring-map")
	key := consumersMapKey(stream)
	require.NoError(t, rdb.HSet(t.Context(), key, "old-group", `[]`).Err())
	require.NoError(t, rdb.Expire(t.Context(), key, time.Hour).Err())
	sink, err := stream.NewSink(t.Context(), "new-group", options.WithSinkTopic("orders"))
	if sink != nil {
		t.Cleanup(func() {
			sink.Close(t.Context())
		})
	}
	require.ErrorContains(t, err, "migration")
	exists, err := rdb.HExists(t.Context(), key, "new-group").Result()
	require.NoError(t, err)
	require.False(t, exists, "rejected admission must not publish a member")
}

func TestSinkGroupFilterRejectsLegacyRecord(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "filter-legacy-map")
	key := consumersMapKey(stream)
	const legacy = `["old-consumer"]`
	require.NoError(t, rdb.HSet(t.Context(), key, "shared", legacy).Err())
	sink, err := stream.NewSink(t.Context(), "shared", options.WithSinkTopic("orders"))
	if sink != nil {
		t.Cleanup(func() {
			sink.Close(t.Context())
		})
	}
	require.ErrorContains(t, err, "migration")
	value, err := rdb.HGet(t.Context(), key, "shared").Result()
	require.NoError(t, err)
	require.Equal(t, legacy, value, "rejected admission must preserve the legacy value")
}

func TestSinkGroupFilterRejectsUnknownExistingGroup(t *testing.T) {
	rdb := startTestRedis(t)
	stream := newTestStream(t, rdb, "filter-unknown-group")
	require.NoError(t, rdb.XGroupCreateMkStream(t.Context(), stream.key, "shared", "0").Err())
	key := consumersMapKey(stream)
	exists, err := rdb.HExists(t.Context(), key, "shared").Result()
	require.NoError(t, err)
	require.False(t, exists)
	sink, err := stream.NewSink(t.Context(), "shared", options.WithSinkTopic("orders"))
	if sink != nil {
		t.Cleanup(func() {
			sink.Close(t.Context())
		})
	}
	require.ErrorContains(t, err, "migration")
	exists, err = rdb.HExists(t.Context(), key, "shared").Result()
	require.NoError(t, err)
	require.False(t, exists, "unknown existing group must not acquire a guessed identity")
}

func TestSinkFilterReviewExpiredStreamRetiresClosedMember(t *testing.T) {
	srv := startTestServer(t)
	stream, err := NewStream("review-expired-member", srv.Client, options.WithStreamTTL(time.Hour))
	require.NoError(t, err)
	old := newTestSink(t, stream, "shared", options.WithSinkAckGracePeriod(time.Hour))
	oldName := sinkConsumer(old)
	old.Close(t.Context())
	if mr := srv.Miniredis(); mr != nil {
		mr.FastForward(2 * time.Hour)
	} else {
		require.NoError(t, srv.Client.PExpire(t.Context(), stream.key, 0).Err())
	}
	require.EqualValues(t, 0, srv.Client.Exists(t.Context(), stream.key).Val())
	live := newTestSink(t, stream, "shared", options.WithSinkAckGracePeriod(time.Hour))
	setSinkMemberKeepAlive(t, live.consumersMap[stream.Name], live.Name, live.filterIdentity, oldName, time.Now().Add(-3*time.Hour).UnixNano())
	native, err := srv.Client.XInfoConsumers(t.Context(), stream.key, live.Name).Result()
	require.NoError(t, err)
	require.Len(t, native, 1)
	require.Equal(t, sinkConsumer(live), native[0].Name)
	live.lock.Lock()
	err = live.deleteStreamStaleConsumers(t.Context(), stream)
	live.lock.Unlock()
	require.NoError(t, err)
	record, _, err := readSinkMembership(t.Context(), stream, live.Name)
	require.NoError(t, err)
	if len(record.Members) != 1 || record.Members[0].Name != sinkConsumer(live) {
		t.Errorf("closed stale consumer survived stream expiry: members=%v old=%s live=%s", record.Members, oldName, sinkConsumer(live))
	}
	require.NoError(t, live.RemoveStream(t.Context(), stream))
	groups, err := srv.Client.XInfoGroups(t.Context(), stream.key).Result()
	require.NoError(t, err)
	require.Empty(t, groups, "final removal must destroy group after stale closed member is retired")
}

func TestSinkFilterReviewMissingFilterIdentityRejects(t *testing.T) {
	for _, raw := range []string{
		`{"version":2,"members":[]}`,
		`{"version":2,"filter":null,"members":[]}`,
		`{"version":2,"filter":{},"members":[]}`,
		`{"version":2,"filter":{"kind":null},"members":[]}`,
	} {
		t.Run(raw, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "review-missing-filter")
			require.NoError(t, rdb.HSet(t.Context(), consumersMapKey(stream), "shared", raw).Err())
			sink, err := stream.NewSink(t.Context(), "shared", options.WithSinkAckGracePeriod(time.Hour))
			if sink != nil {
				sink.Close(t.Context())
			}
			require.ErrorContains(t, err, "migration", "absent identity must not be inferred as unfiltered")
		})
	}
}
