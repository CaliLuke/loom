package rmapstate

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/internal/redistest"
)

func TestSharedWriterPreflightPrecedesAllEffects(t *testing.T) {
	for _, corrupt := range []map[string]string{
		{"=protocol": "future"},
		{"=generation": "unowned"},
		{"=protocol": "1", "=generation": "existing", "=rev": "9223372036854775807", "=reset": "0", "=delete": "0"},
	} {
		srv := redistest.Start(t, redistest.DBRmap)
		const first, second, sideEffect = "map:first", "map:second", "other-owner"
		require.NoError(t, srv.Client.HSet(t.Context(), second, corrupt).Err())
		script := NewScript([]int{1, 2}, `
redis.call("SET", KEYS[3], "published")
set_map(KEYS[1], "updates:first", "value", "one")
set_map(KEYS[2], "updates:second", "value", "two")
return 1
`)
		require.Error(t, script.Run(t.Context(), srv.Client, []string{first, second, sideEffect}).Err())
		exists, err := srv.Client.Exists(t.Context(), first, sideEffect).Result()
		require.NoError(t, err)
		require.Zero(t, exists, "Redis does not roll back Lua commands before a later error")
		after, err := srv.Client.HGetAll(t.Context(), second).Result()
		require.NoError(t, err)
		require.Equal(t, corrupt, after)
	}
}

func TestSharedWriterExactRevisionAndIncarnation(t *testing.T) {
	srv := redistest.Start(t, redistest.DBRmap)
	const key = "map:exact"
	require.NoError(t, srv.Client.HSet(t.Context(), key, map[string]string{
		"=protocol": "1", "=generation": "existing", "=rev": "9007199254740992", "=reset": "0", "=delete": "0",
	}).Err())
	script := NewScript([]int{1}, `local rev, generation = set_map(KEYS[1], "updates", ARGV[1], ARGV[2]); return {rev, generation}`)
	result, err := script.Run(t.Context(), srv.Client, []string{key}, "business", "value").StringSlice()
	require.NoError(t, err)
	require.Equal(t, []string{"9007199254740993", "existing"}, result)
	require.NoError(t, srv.Client.Del(t.Context(), key).Err())
	result, err = script.Run(t.Context(), srv.Client, []string{key}, "business", "new").StringSlice()
	require.NoError(t, err)
	require.Equal(t, "1", result[0])
	require.NotEmpty(t, result[1])
	require.NotEqual(t, "existing", result[1])
	generation := result[1]
	result, err = script.Run(t.Context(), srv.Client, []string{key}, "business", "again").StringSlice()
	require.NoError(t, err)
	require.Equal(t, []string{"2", generation}, result)
}
