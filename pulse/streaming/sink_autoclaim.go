package streaming

import (
	"context"
	"fmt"

	redis "github.com/redis/go-redis/v9"
)

// autoClaim runs XAUTOCLAIM with args. It returns the claimed events, the
// number of claimed pending entries whose events were deleted from the
// stream, and the start id of the next batch ("0-0" once the scan is done).
//
// It sends the raw command instead of using go-redis XAutoClaim: for each
// pending id whose event was deleted (by XDEL or a MAXLEN trim), Redis 6.2
// replies with a null entry, which go-redis fails to parse, dropping the
// whole batch (issue #408). A null entry is skipped. Redis 6.2 keeps its
// pending entry, now owned by args.Consumer, so later scans claim it again;
// it is not acked, because the pool dispatch guard reads a pending entry as
// an unacked start event. Redis 7 and later purge such pending entries and
// list their ids in a third reply element, which is counted.
func autoClaim(ctx context.Context, rdb *redis.Client, args redis.XAutoClaimArgs) ([]redis.XMessage, int, string, error) {
	cmd := []any{"XAUTOCLAIM", args.Stream, args.Group, args.Consumer, args.MinIdle.Milliseconds(), args.Start}
	if args.Count > 0 {
		cmd = append(cmd, "COUNT", args.Count)
	}
	reply, err := rdb.Do(ctx, cmd...).Slice()
	if err != nil {
		return nil, 0, "", fmt.Errorf("XAUTOCLAIM %s: %w", args.Stream, err)
	}
	messages, deleted, start, err := parseAutoClaim(reply)
	if err != nil {
		return nil, 0, "", fmt.Errorf("XAUTOCLAIM %s: %w", args.Stream, err)
	}
	return messages, deleted, start, nil
}

// parseAutoClaim parses an XAUTOCLAIM reply: the next start id, the claimed
// entries, and on Redis 7 and later the ids of purged pending entries. A null
// entry, or an entry with null fields, is a claimed pending entry whose event
// was deleted (Redis 6.2).
func parseAutoClaim(reply []any) ([]redis.XMessage, int, string, error) {
	if len(reply) != 2 && len(reply) != 3 {
		return nil, 0, "", fmt.Errorf("got %d reply elements, want 2 or 3", len(reply))
	}
	start, ok := reply[0].(string)
	if !ok {
		return nil, 0, "", fmt.Errorf("start id is %T, want string", reply[0])
	}
	entries, ok := reply[1].([]any)
	if !ok {
		return nil, 0, "", fmt.Errorf("entries are %T, want array", reply[1])
	}
	messages := make([]redis.XMessage, 0, len(entries))
	deleted := 0
	for i, entry := range entries {
		if entry == nil {
			deleted++
			continue
		}
		msg, ok, err := parseStreamEntry(entry)
		if err != nil {
			return nil, 0, "", fmt.Errorf("entry %d: %w", i, err)
		}
		if !ok {
			deleted++
			continue
		}
		messages = append(messages, msg)
	}
	if len(reply) == 3 {
		purged, ok := reply[2].([]any)
		if !ok {
			return nil, 0, "", fmt.Errorf("deleted ids are %T, want array", reply[2])
		}
		deleted += len(purged)
	}
	return messages, deleted, start, nil
}

// parseStreamEntry parses a stream entry reply: its id and its flat list of
// field names and values. It returns false for an entry with null fields,
// whose event was deleted.
func parseStreamEntry(entry any) (redis.XMessage, bool, error) {
	parts, ok := entry.([]any)
	if !ok || len(parts) != 2 {
		return redis.XMessage{}, false, fmt.Errorf("entry is %T of length %d, want array of length 2", entry, len(parts))
	}
	id, ok := parts[0].(string)
	if !ok {
		return redis.XMessage{}, false, fmt.Errorf("id is %T, want string", parts[0])
	}
	if parts[1] == nil {
		return redis.XMessage{}, false, nil
	}
	fields, ok := parts[1].([]any)
	if !ok || len(fields)%2 != 0 {
		return redis.XMessage{}, false, fmt.Errorf("entry %s fields are %T of length %d, want array of even length", id, parts[1], len(fields))
	}
	values := make(map[string]any, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		name, ok := fields[i].(string)
		if !ok {
			return redis.XMessage{}, false, fmt.Errorf("entry %s field %d name is %T, want string", id, i/2, fields[i])
		}
		values[name] = fields[i+1]
	}
	return redis.XMessage{ID: id, Values: values}, true, nil
}
