package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type snapshotCustom struct {
	calls *int
}

func (v *snapshotCustom) MarshalJSON() ([]byte, error) {
	*v.calls++
	return []byte(`"custom"`), nil
}

func TestValueSnapshotPreservesHostTypesAndOwnsBuiltins(t *testing.T) {
	type bytes []byte
	type numbers map[int64]any
	original := numbers{1: bytes{2, 3}, 2: int32(4), 3: []string(nil)}
	snapshot := snapshotValueSource(original)
	require.NoError(t, snapshot.err)
	require.Equal(t, original, snapshot.raw)
	owned, ok := snapshot.raw.(numbers)
	require.True(t, ok)
	original[1].(bytes)[0] = 9
	original[2] = int64(8)
	require.Equal(t, bytes{2, 3}, owned[1])
	require.Equal(t, int32(4), owned[2])
	require.Nil(t, owned[3])
}

func TestValueSnapshotCyclesAndSharedChildren(t *testing.T) {
	cyclicMap := map[string]any{}
	cyclicMap["self"] = cyclicMap
	cyclicSlice := make([]any, 1)
	cyclicSlice[0] = cyclicSlice
	shared := map[string]any{"value": []byte("hi")}
	for _, tc := range []struct {
		name  string
		raw   any
		cycle bool
	}{
		{"map cycle", cyclicMap, true},
		{"slice cycle", cyclicSlice, true},
		{"shared finite child", []any{shared, shared}, false},
		{"nil pointer", (*struct{ Value string })(nil), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := snapshotValueSource(tc.raw)
			if tc.cycle {
				require.ErrorContains(t, snapshot.err, "cyclic value")
				return
			}
			require.NoError(t, snapshot.err)
			require.Equal(t, tc.raw, snapshot.raw)
		})
	}
}

func TestValueSnapshotDoesNotMaterializeCustomValues(t *testing.T) {
	calls := 0
	raw := &snapshotCustom{calls: &calls}
	snapshot := snapshotValueSource(map[string]any{"value": raw})
	require.NoError(t, snapshot.err)
	require.Zero(t, calls)
	require.Same(t, raw, snapshot.raw.(map[string]any)["value"])
}

type snapshotPointerCodec []string

type snapshotWrongCodec []string

type snapshotTextAppender string

func (v *snapshotPointerCodec) MarshalJSON() ([]byte, error) {
	return []byte(`"pointer codec"`), nil
}

func (snapshotWrongCodec) MarshalJSONTo(int) string {
	return "not a codec"
}

func (v snapshotTextAppender) AppendText(dst []byte) ([]byte, error) {
	return append(dst, string(v)...), nil
}

func TestValueSnapshotCodecProtocols(t *testing.T) {
	for _, tc := range []struct {
		name   string
		raw    any
		custom bool
	}{
		{"pointer receiver on value", snapshotPointerCodec{"original"}, true},
		{"wrong method signature", snapshotWrongCodec{"original"}, false},
		{"text appender", snapshotTextAppender("text"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.custom, valueCustomSource(tc.raw))
			snapshot := snapshotValueSource(tc.raw)
			require.NoError(t, snapshot.err)
			require.Equal(t, tc.raw, snapshot.raw)
		})
	}
	wrong := snapshotWrongCodec{"before"}
	owned := snapshotValueSource(wrong)
	wrong[0] = "after"
	require.Equal(t, snapshotWrongCodec{"before"}, owned.raw)
}
