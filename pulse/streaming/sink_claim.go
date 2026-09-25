package streaming

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type (
	// claimArgs selects one batch of idle pending entries of a consumer
	// group for claimIdle.
	claimArgs struct {
		// stream is the stream key.
		stream string
		// group is the consumer group name.
		group string
		// consumer is the consumer that takes the claimed entries.
		consumer string
		// minIdle is the time since the last delivery after which an entry
		// is idle.
		minIdle time.Duration
		// start is the smallest pending id of the batch.
		start string
		// count is the maximum number of pending entries the batch scans.
		count int
	}
)

const (
	// claimDone is the start id claimIdle returns once no pending entry
	// follows the batch.
	claimDone = "0-0"
	// defaultClaimCount is the batch size of claimIdle, the default COUNT
	// of XAUTOCLAIM.
	defaultClaimCount = 100
)

// claimIdleScript claims the idle pending entries of a consumer group, as
// XAUTOCLAIM does, and acks those whose events were deleted from the stream
// (by XDEL or a MAXLEN trim).
//
// Redis 7 and later XAUTOCLAIM purges such entries. Redis 6.2 XAUTOCLAIM
// claims them again on every scan and replies with a null entry that carries
// no id (issues #408 and #411), and miniredis keeps them with their owner.
// The script acks them on every version, so a pending list never keeps an
// entry whose event is gone, and a consumer that owned only such entries
// becomes deletable. The pool dispatch guard does not depend on such an
// entry: it counts a delivered event that is in neither the stream nor the
// pending list as in flight (issue #385).
//
// A batch scans at most the batch size of pending entries from the start id,
// idle or not, so one call stays bounded however many entries are not idle;
// XCLAIM with the minimum idle time still guards the claim.
//
// KEYS[1] is the stream key. ARGV holds the group, the consumer, the minimum
// idle time in milliseconds, the start id and the batch size. The reply holds
// the id of the last scanned entry when the scan filled the batch ("" once no
// entry follows), the claimed entries, and the number of acked entries.
var claimIdleScript = redis.NewScript(`
local pending = redis.call("XPENDING", KEYS[1], ARGV[1], ARGV[4], "+", ARGV[5])
local min_idle = tonumber(ARGV[3])
local live, deleted = {}, {}
for _, p in ipairs(pending) do
   if p[3] >= min_idle then
      if #redis.call("XRANGE", KEYS[1], p[1], p[1]) == 0 then
         deleted[#deleted + 1] = p[1]
      else
         live[#live + 1] = p[1]
      end
   end
end
if #deleted > 0 then
   redis.call("XACK", KEYS[1], ARGV[1], unpack(deleted))
end
local entries = {}
if #live > 0 then
   entries = redis.call("XCLAIM", KEYS[1], ARGV[1], ARGV[2], ARGV[3], unpack(live))
end
local last = ""
if #pending == tonumber(ARGV[5]) then
   last = pending[#pending][1]
end
return {last, entries, #deleted}
`)

// claimIdle scans one batch of pending entries with args and claims the idle
// ones. It returns the claimed events, the number of acked entries whose
// events were deleted, and the start id of the next batch (claimDone once the
// scan is done).
func claimIdle(ctx context.Context, rdb *redis.Client, args claimArgs) ([]redis.XMessage, int, string, error) {
	count := args.count
	if count <= 0 {
		count = defaultClaimCount
	}
	reply, err := claimIdleScript.Run(ctx, rdb, []string{args.stream},
		args.group, args.consumer, minIdleMillis(args.minIdle), args.start, count).Slice()
	if err != nil {
		return nil, 0, "", fmt.Errorf("claim idle entries of %s: %w", args.stream, err)
	}
	messages, deleted, last, err := parseClaimIdle(reply)
	if err != nil {
		return nil, 0, "", fmt.Errorf("claim idle entries of %s: %w", args.stream, err)
	}
	if last == "" {
		return messages, deleted, claimDone, nil
	}
	next, err := nextStreamID(last)
	if err != nil {
		return nil, 0, "", fmt.Errorf("claim idle entries of %s: %w", args.stream, err)
	}
	return messages, deleted, next, nil
}

// minIdleMillis returns d in milliseconds for XPENDING IDLE and XCLAIM. A
// positive d under a millisecond is one millisecond, so it never becomes 0,
// which would make every pending entry idle.
func minIdleMillis(d time.Duration) int64 {
	if d > 0 && d < time.Millisecond {
		return 1
	}
	return d.Milliseconds()
}

// nextStreamID returns the smallest stream id greater than id, or claimDone
// when id is the greatest one.
func nextStreamID(id string) (string, error) {
	msText, seqText, ok := strings.Cut(id, "-")
	if !ok {
		return "", fmt.Errorf("stream id %q has no sequence number", id)
	}
	ms, err := strconv.ParseUint(msText, 10, 64)
	if err != nil {
		return "", fmt.Errorf("stream id %q: %w", id, err)
	}
	seq, err := strconv.ParseUint(seqText, 10, 64)
	if err != nil {
		return "", fmt.Errorf("stream id %q: %w", id, err)
	}
	switch {
	case seq < math.MaxUint64:
		return strconv.FormatUint(ms, 10) + "-" + strconv.FormatUint(seq+1, 10), nil
	case ms < math.MaxUint64:
		return strconv.FormatUint(ms+1, 10) + "-0", nil
	default:
		return claimDone, nil
	}
}

// parseClaimIdle parses a claimIdleScript reply: the last scanned id of a full
// batch, the claimed entries and the number of acked entries.
func parseClaimIdle(reply []any) ([]redis.XMessage, int, string, error) {
	if len(reply) != 3 {
		return nil, 0, "", fmt.Errorf("got %d reply elements, want 3", len(reply))
	}
	last, ok := reply[0].(string)
	if !ok {
		return nil, 0, "", fmt.Errorf("last id is %T, want string", reply[0])
	}
	entries, ok := reply[1].([]any)
	if !ok {
		return nil, 0, "", fmt.Errorf("entries are %T, want array", reply[1])
	}
	deleted, ok := reply[2].(int64)
	if !ok {
		return nil, 0, "", fmt.Errorf("deleted count is %T, want integer", reply[2])
	}
	messages := make([]redis.XMessage, 0, len(entries))
	for i, entry := range entries {
		msg, err := parseStreamEntry(entry)
		if err != nil {
			return nil, 0, "", fmt.Errorf("entry %d: %w", i, err)
		}
		messages = append(messages, msg)
	}
	return messages, int(deleted), last, nil
}

// parseStreamEntry parses a stream entry reply: its id and its flat list of
// field names and values.
func parseStreamEntry(entry any) (redis.XMessage, error) {
	parts, ok := entry.([]any)
	if !ok || len(parts) != 2 {
		return redis.XMessage{}, fmt.Errorf("entry is %T of length %d, want array of length 2", entry, len(parts))
	}
	id, ok := parts[0].(string)
	if !ok {
		return redis.XMessage{}, fmt.Errorf("id is %T, want string", parts[0])
	}
	fields, ok := parts[1].([]any)
	if !ok || len(fields)%2 != 0 {
		return redis.XMessage{}, fmt.Errorf("entry %s fields are %T of length %d, want array of even length", id, parts[1], len(fields))
	}
	values := make(map[string]any, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		name, ok := fields[i].(string)
		if !ok {
			return redis.XMessage{}, fmt.Errorf("entry %s field %d name is %T, want string", id, i/2, fields[i])
		}
		values[name] = fields[i+1]
	}
	return redis.XMessage{ID: id, Values: values}, nil
}
