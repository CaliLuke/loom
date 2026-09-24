package streaming

import (
	"testing"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestParseAutoClaim(t *testing.T) {
	entry := func(id string, fields ...any) any {
		return []any{id, fields}
	}
	cases := []struct {
		name     string
		reply    []any
		messages []redis.XMessage
		deleted  int
		start    string
		err      string
	}{
		{
			name:     "Redis 6.2 reply",
			reply:    []any{"0-0", []any{entry("1-0", "n", "a", "p", "x")}},
			messages: []redis.XMessage{{ID: "1-0", Values: map[string]any{"n": "a", "p": "x"}}},
			start:    "0-0",
		},
		{
			name:     "Redis 6.2 null entry of a deleted id",
			reply:    []any{"3-0", []any{nil, entry("2-0", "n", "b"), nil}},
			messages: []redis.XMessage{{ID: "2-0", Values: map[string]any{"n": "b"}}},
			deleted:  2,
			start:    "3-0",
		},
		{
			name:     "entry with null fields",
			reply:    []any{"0-0", []any{[]any{"1-0", nil}, entry("2-0", "n", "b")}},
			messages: []redis.XMessage{{ID: "2-0", Values: map[string]any{"n": "b"}}},
			deleted:  1,
			start:    "0-0",
		},
		{
			name:     "Redis 7 purged ids",
			reply:    []any{"0-0", []any{entry("2-0")}, []any{"1-0", "3-0"}},
			messages: []redis.XMessage{{ID: "2-0", Values: map[string]any{}}},
			deleted:  2,
			start:    "0-0",
		},
		{
			name:     "nothing claimed",
			reply:    []any{"0-0", []any{}, []any{}},
			messages: []redis.XMessage{},
			start:    "0-0",
		},
		{name: "too short", reply: []any{"0-0"}, err: "got 1 reply elements"},
		{name: "start not a string", reply: []any{int64(0), []any{}}, err: "start id is int64"},
		{name: "entries not an array", reply: []any{"0-0", "x"}, err: "entries are string"},
		{name: "purged ids not an array", reply: []any{"0-0", []any{}, "x"}, err: "deleted ids are string"},
		{name: "entry not an array", reply: []any{"0-0", []any{"1-0"}}, err: "entry 0: entry is string"},
		{name: "entry id not a string", reply: []any{"0-0", []any{[]any{int64(1), []any{}}}}, err: "entry 0: id is int64"},
		{name: "odd fields", reply: []any{"0-0", []any{entry("1-0", "n")}}, err: "entry 0: entry 1-0 fields"},
		{name: "field name not a string", reply: []any{"0-0", []any{entry("1-0", int64(1), "v")}}, err: "field 0 name is int64"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			messages, deleted, start, err := parseAutoClaim(tc.reply)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.messages, messages)
			require.Equal(t, tc.deleted, deleted)
			require.Equal(t, tc.start, start)
		})
	}
}
