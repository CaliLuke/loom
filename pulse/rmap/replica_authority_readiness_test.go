package rmap

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/internal/redistest"
)

func TestReplicaAuthorityRejectsIncompleteGap(t *testing.T) {
	m := testMap()
	m.lock.Lock()
	_, applied, err := m.applyMessageLocked("set", packedStrings(mapProtocol, "test", "1", "keep", "one"))
	m.lock.Unlock()
	require.NoError(t, err)
	require.True(t, applied)
	m.lock.Lock()
	_, applied, err = m.applyMessageLocked("set", packedStrings(mapProtocol, "test", "3", "later", "three"))
	m.lock.Unlock()
	require.NoError(t, err)
	require.False(t, applied, "revision 3 cannot certify completeness when revision 2 was never observed")
}

func TestReplicaAuthorityTTLObservation(t *testing.T) {
	for _, mode := range []struct {
		name   string
		option MapOption
	}{
		{name: "fixed", option: WithTTL(time.Second)},
		{name: "sliding", option: WithSlidingTTL(time.Second)},
	} {
		t.Run(mode.name, func(t *testing.T) {
			srv := redistest.Start(t, redistest.DBRmap)
			m, err := Join(t.Context(), "authority-expiry", srv.Client, mode.option)
			require.NoError(t, err)
			t.Cleanup(m.Close)
			for i := range 3 {
				_, err = m.SetAndWait(t.Context(), fmt.Sprintf("old-%d", i), "old")
				require.NoError(t, err)
			}
			if srv.Real() {
				require.Eventually(t, func() bool {
					exists, checkErr := srv.Client.Exists(t.Context(), m.hashkey).Result()
					require.NoError(t, checkErr)
					return exists == 0
				}, 3*time.Second, 10*time.Millisecond)
			} else {
				srv.Miniredis().FastForward(2 * time.Second)
			}
			exists, err := srv.Client.Exists(t.Context(), m.hashkey).Result()
			require.NoError(t, err)
			require.Zero(t, exists)
			_, err = m.SetAndWait(t.Context(), "new", "after-expiry")
			require.NoError(t, err)
			require.Equal(t, map[string]string{"new": "after-expiry"}, m.Map(), "SetAndWait returned before observing its new incarnation")
		})
	}
}
