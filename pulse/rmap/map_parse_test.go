package rmap

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/pulse"
)

func TestIsValidRedisKeyName(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{name: "ordinary", key: "pool:jobs", valid: true},
		{name: "slash", key: "stream/events", valid: true},
		{name: "empty", key: "", valid: false},
		{name: "space", key: "pool jobs", valid: false},
		{name: "nul", key: "pool\x00jobs", valid: false},
		{name: "glob", key: "pool*", valid: false},
		{name: "too-long", key: strings.Repeat("a", 513), valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.valid, isValidRedisKeyName(tt.key))
		})
	}
}

func TestParseValuesSupportsJSONAndLegacyCSV(t *testing.T) {
	require.Equal(t, []string{"a", "b,c"}, parseValues(`["a","b,c"]`))
	require.Equal(t, []string{"a", "b", "c"}, parseValues("a,b,c"))
	require.Equal(t, []string{""}, parseValues(""))
	require.Equal(t, []string{"[not-json]"}, parseValues("[not-json]"))
}

func TestExtractStateRemovesMetadata(t *testing.T) {
	content := map[string]string{
		"alpha":   "one",
		revField:  "42",
		kindField: "del",
	}

	rev, kind := extractState(content)

	require.Equal(t, uint64(42), rev)
	require.Equal(t, EventDelete, kind)
	require.Equal(t, map[string]string{"alpha": "one"}, content)
}

func TestApplyMessageLockedAppliesFreshChanges(t *testing.T) {
	sm := testMap()

	sm.lock.Lock()
	change, applied, err := sm.applyMessageLocked("set", packedStrings("alpha", "one", "1"))
	sm.lock.Unlock()
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, mapChange{kind: EventChange, rev: 1}, change)
	require.Equal(t, "one", sm.content["alpha"])

	sm.lock.Lock()
	change, applied, err = sm.applyMessageLocked("del", packedStrings("alpha", "2"))
	sm.lock.Unlock()
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, mapChange{kind: EventDelete, rev: 2}, change)
	require.NotContains(t, sm.content, "alpha")

	sm.lock.Lock()
	change, applied, err = sm.applyMessageLocked("reset", []byte("3"))
	sm.lock.Unlock()
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, mapChange{kind: EventReset, rev: 3}, change)
	require.Empty(t, sm.content)
}

func TestApplyMessageLockedIgnoresStaleAndReservedChanges(t *testing.T) {
	sm := testMap()
	sm.rev = 5
	sm.content["alpha"] = "one"

	sm.lock.Lock()
	change, applied, err := sm.applyMessageLocked("set", packedStrings("alpha", "stale", "4"))
	sm.lock.Unlock()
	require.NoError(t, err)
	require.False(t, applied)
	require.Zero(t, change)
	require.Equal(t, "one", sm.content["alpha"])

	sm.lock.Lock()
	change, applied, err = sm.applyMessageLocked("set", packedStrings(revField, "ignored", "6"))
	sm.lock.Unlock()
	require.NoError(t, err)
	require.False(t, applied)
	require.Zero(t, change)
	require.Equal(t, uint64(5), sm.rev)
}

func TestWaitersObserveCommittedRevision(t *testing.T) {
	sm := testMap()
	sm.rev = 10
	ready := &setWaiter{rev: 9, ch: make(chan struct{}, 1), ctx: t.Context()}
	require.True(t, sm.registerWaiter(ready))
	require.Empty(t, sm.waiters)

	waiter := &setWaiter{rev: 12, ch: make(chan struct{}, 1), ctx: t.Context()}
	require.False(t, sm.registerWaiter(waiter))
	require.Len(t, sm.waiters[12], 1)

	sm.notifyWaitersUpTo(12)
	require.Empty(t, sm.waiters)
	require.NotEmpty(t, waiter.ch)
}

func TestNotifySubscribersIsBestEffort(t *testing.T) {
	ch := make(chan EventKind, 1)
	ch <- EventChange
	sm := testMap()
	sm.chans = []chan EventKind{ch}

	require.NotPanics(t, func() {
		sm.notifySubscribers(EventReset)
	})
	require.Equal(t, EventChange, <-ch)
}

func testMap() *Map {
	return &Map{
		Name:    "test",
		content: make(map[string]string),
		waiters: make(map[uint64][]*setWaiter),
		done:    make(chan struct{}),
		logger:  pulse.NoopLogger(),
	}
}

func packedStrings(values ...string) []byte {
	var data []byte
	for _, value := range values {
		var length [4]byte
		binary.LittleEndian.PutUint32(length[:], uint32(len(value)))
		data = append(data, length[:]...)
		data = append(data, value...)
	}
	return data
}
