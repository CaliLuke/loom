package rmap

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/internal/rmapstate"
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
			require.Equal(t, tt.valid, rmapstate.ValidName(tt.key))
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
	content := map[string]string{"alpha": "one", protocolField: mapProtocol, generationField: "test", revField: "42", resetField: "5", deleteField: "40"}
	state, err := extractState(content)
	require.NoError(t, err)
	require.Equal(t, uint64(42), state.rev)
	require.Equal(t, uint64(5), state.reset)
	require.Equal(t, uint64(40), state.deleted)
	require.Equal(t, "test", state.generation)
	require.Equal(t, map[string]string{"alpha": "one"}, state.content)
}

func TestApplyMessageLockedAppliesFreshChanges(t *testing.T) {
	sm := testMap()

	sm.lock.Lock()
	change, applied, err := sm.applyMessageLocked("set", packedStrings(mapProtocol, "test", "1", "alpha", "one"))
	sm.lock.Unlock()
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, mapChange{kind: EventChange, rev: 1}, change)
	require.Equal(t, "one", sm.content["alpha"])

	sm.lock.Lock()
	change, applied, err = sm.applyMessageLocked("del", packedStrings(mapProtocol, "test", "2", "alpha"))
	sm.lock.Unlock()
	require.NoError(t, err)
	require.True(t, applied)
	require.Equal(t, mapChange{kind: EventDelete, rev: 2}, change)
	require.NotContains(t, sm.content, "alpha")

	sm.lock.Lock()
	change, applied, err = sm.applyMessageLocked("reset", packedStrings(mapProtocol, "test", "3"))
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
	change, applied, err := sm.applyMessageLocked("set", packedStrings(mapProtocol, "test", "4", "alpha", "stale"))
	sm.lock.Unlock()
	require.NoError(t, err)
	require.False(t, applied)
	require.Zero(t, change)
	require.Equal(t, "one", sm.content["alpha"])

	sm.lock.Lock()
	change, applied, err = sm.applyMessageLocked("set", packedStrings(mapProtocol, "test", "6", revField, "ignored"))
	sm.lock.Unlock()
	require.Error(t, err)
	require.False(t, applied)
	require.Zero(t, change)
	require.Equal(t, uint64(5), sm.rev)
}

func testMap() *Map {
	return &Map{
		Name:       "test",
		content:    make(map[string]string),
		generation: "test",
		done:       make(chan struct{}),
		logger:     pulse.NoopLogger(),
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
