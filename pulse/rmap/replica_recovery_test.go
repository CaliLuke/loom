package rmap

import (
	"fmt"
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/internal/redistest"
)

func TestReplicaPeriodicRepairWithoutPublication(t *testing.T) {
	for _, mode := range []string{"lost last write", "expiry without recreation"} {
		t.Run(mode, func(t *testing.T) {
			srv := redistest.Start(t, redistest.DBRmap)
			reader := joinMap(t, srv.Client, "periodic-repair")
			_, err := reader.SetAndWait(t.Context(), "old", "value")
			require.NoError(t, err)
			require.NoError(t, reader.sub.Unsubscribe(t.Context(), reader.chankey))
			if mode == "lost last write" {
				writer := joinMap(t, srv.Client, "periodic-repair")
				_, err = writer.Set(t.Context(), "last", "unpublished")
				require.NoError(t, err)
				require.Eventually(t, func() bool {
					return maps.Equal(reader.Map(), map[string]string{"old": "value", "last": "unpublished"})
				}, 3*time.Second, 10*time.Millisecond)
			} else {
				require.NoError(t, srv.Client.PExpire(t.Context(), reader.hashkey, time.Millisecond).Err())
				if !srv.Real() {
					srv.Miniredis().FastForward(time.Second)
				}
				require.Eventually(t, func() bool { return reader.Len() == 0 }, 3*time.Second, 10*time.Millisecond)
			}
		})
	}
}

func TestReplicaRecoveryPreservesUnseenKinds(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind EventKind
	}{
		{name: "set", kind: EventChange},
		{name: "delete then set", kind: EventDelete},
		{name: "reset then set", kind: EventReset},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := redistest.Start(t, redistest.DBRmap)
			writer := joinMap(t, srv.Client, "recovery-kind")
			_, err := writer.Set(t.Context(), "before", "value")
			require.NoError(t, err)
			// A disconnected replica has a complete baseline but receives no deltas.
			reader := testMap()
			reader.rdb, reader.hashkey = srv.Client, writer.hashkey
			state, err := reader.readSnapshot(t.Context())
			require.NoError(t, err)
			reader.content, reader.generation, reader.rev = state.content, state.generation, state.rev
			notifications := reader.Subscribe()
			switch tc.name {
			case "delete then set":
				_, err = writer.Delete(t.Context(), "before")
			case "reset then set":
				err = writer.Reset(t.Context())
			}
			require.NoError(t, err)
			_, err = writer.Set(t.Context(), "after", "value")
			require.NoError(t, err)
			require.NoError(t, reader.reconcile(t.Context()))
			require.Equal(t, tc.kind, <-notifications)
			// A delayed message from another generation only requests a read. An
			// unchanged complete snapshot must not invent another Reset event.
			reader.closectx = t.Context()
			require.NoError(t, reader.receiveMessage("set:"+string(packedStrings(mapProtocol, "retired", "1", "before", "old"))))
			select {
			case kind := <-notifications:
				t.Errorf("unchanged recovery emitted %v", kind)
			default:
			}
		})
	}
}

func TestReplicaCoordinatedMetadataAdoption(t *testing.T) {
	for _, metadata := range []string{"old", "unknown protocol", "reserved collision"} {
		t.Run(metadata, func(t *testing.T) {
			srv := redistest.Start(t, redistest.DBRmap)
			key := "map:adoption:content"
			fields := map[string]string{"business": "preserved", revField: "82", kindField: "reset"}
			switch metadata {
			case "unknown protocol":
				fields[protocolField] = "future"
			case "reserved collision":
				fields[generationField] = "unowned"
			}
			require.NoError(t, srv.Client.HSet(t.Context(), key, fields).Err())
			require.NoError(t, srv.Client.PExpire(t.Context(), key, time.Hour).Err())
			beforeTTL, err := srv.Client.PTTL(t.Context(), key).Result()
			require.NoError(t, err)
			m, err := Join(t.Context(), "adoption", srv.Client)
			if metadata != "old" {
				require.Error(t, err)
				require.Nil(t, m)
				after, readErr := srv.Client.HGetAll(t.Context(), key).Result()
				require.NoError(t, readErr)
				require.Equal(t, fields, after)
				return
			}
			require.NoError(t, err)
			t.Cleanup(m.Close)
			require.Equal(t, map[string]string{"business": "preserved"}, m.Map())
			afterTTL, err := srv.Client.PTTL(t.Context(), key).Result()
			require.NoError(t, err)
			require.Positive(t, afterTTL)
			require.LessOrEqual(t, afterTTL, beforeTTL)
			require.Greater(t, afterTTL, beforeTTL-time.Second)
			_, err = m.SetAndWait(t.Context(), "added", "value")
			require.NoError(t, err)
			require.Equal(t, map[string]string{"business": "preserved", "added": "value"}, m.Map())
		})
	}
}

func TestReplicaNoOpPreservesStateAndTTL(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			srv := redistest.Start(t, redistest.DBRmap)
			m := joinMap(t, srv.Client, "noop-state", WithSlidingTTL(time.Hour))
			if existing {
				_, err := m.SetAndWait(t.Context(), "key", "value")
				require.NoError(t, err)
				require.NoError(t, srv.Client.PExpire(t.Context(), m.hashkey, time.Minute).Err())
			}
			before, err := srv.Client.HGetAll(t.Context(), m.hashkey).Result()
			require.NoError(t, err)
			beforeTTL, err := srv.Client.PTTL(t.Context(), m.hashkey).Result()
			require.NoError(t, err)
			_, err = m.TestAndSet(t.Context(), "key", "does not match", "unused")
			require.NoError(t, err)
			after, err := srv.Client.HGetAll(t.Context(), m.hashkey).Result()
			require.NoError(t, err)
			require.Equal(t, before, after)
			afterTTL, err := srv.Client.PTTL(t.Context(), m.hashkey).Result()
			require.NoError(t, err)
			require.LessOrEqual(t, afterTTL, beforeTTL)
			require.Greater(t, afterTTL, beforeTTL-time.Second)
			for _, key := range []string{protocolField, generationField, resetField, deleteField, revField, kindField} {
				_, err = m.Set(t.Context(), key, "cannot override")
				require.Error(t, err)
			}
		})
	}
}

func TestReplicaWriteObservationRejectsRetiredGeneration(t *testing.T) {
	srv := redistest.Start(t, redistest.DBRmap)
	writer := joinMap(t, srv.Client, "retire-before-observe")
	committed, err := writer.runSetScript(t.Context(), "lost", "value")
	require.NoError(t, err)
	// Isolate this observer from Pub/Sub and periodic repair: either could
	// legitimately observe the committed revision before it is retired.
	m := newPendingObservationMap(t, writer)
	require.NoError(t, srv.Client.Del(t.Context(), m.hashkey).Err())
	err = m.observe(observationRequest{ctx: t.Context(), generation: committed.generation, rev: committed.rev})
	require.ErrorContains(t, err, "retired before observation")
	require.Empty(t, m.Map())
}
