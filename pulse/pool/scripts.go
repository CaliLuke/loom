// Package pool keeps its cross-map coordination scripts close to the pool
// admission code. These scripts preserve the rmap notification contract while
// making singleton job admission one atomic Redis operation.
package pool

import redis "github.com/redis/go-redis/v9"

const (
	dispatchClaimed int64 = iota + 1
	dispatchAlreadyPending
	dispatchAlreadyRunning
	dispatchMalformedPending
)

// luaReleaseDispatch statuses. Status 0 means the guard changed since it was
// read and was left alone.
const (
	// dispatchReleaseDeleted means the guard was deleted.
	dispatchReleaseDeleted int64 = iota + 1
	// dispatchReleaseInFlight means the guard was kept because its start
	// event is still unacked.
	dispatchReleaseInFlight
)

// luaDispatchGuard defines the Lua helpers shared by the dispatch scripts.
//
// A pending guard is "untilNanos:eventID", or "untilNanos" as written by
// earlier releases, which carries no event id. in_flight(stream, group, id,
// now_ms, max_age_ms) is true while the start event is unacked in the sink
// group and younger than max_age_ms (pendingEventTTL):
//   - It is in the group's pending list. Routing does not ack an event, and
//     trimming (MAXLEN) or XDEL leaves its pending entry, so a pending event
//     counts even when the stream no longer holds it.
//   - It is still in the stream and not yet delivered: its id is above the
//     group's last-delivered-id, or the group does not exist.
//
// An event that is neither pending nor in the stream is gone. The age bound
// keeps a guard from outliving every recovery path: routing acks events older
// than pendingEventTTL as stale without starting them, but a pending entry
// whose id was trimmed is never redelivered on Redis 6.2 (XAUTOCLAIM keeps
// it), and an event that no sink ever reads is never delivered. Stream ids
// are compared as numbers, never as strings.
//
// Known gap (issue #385): Redis 7 and later XAUTOCLAIM purges the pending
// entry of a deleted id. If MAXLEN trimmed a routed but unacked start event
// (maxQueuedJobs or more events added while it was unacked), the next
// XAUTOCLAIM after ackGracePeriod removes its pending entry, in_flight
// returns false before pendingEventTTL, and a retry can add a second start
// while the first is still queued on a worker stream.
const luaDispatchGuard = `
local function all_digits(value)
   return string.match(value, "^%d+$") ~= nil
end

local function parse_guard(value)
   if all_digits(value) then
      return value, nil
   end
   local until_ns, id = string.match(value, "^(%d+):(%d+%-%d+)$")
   if not until_ns then
      return nil, nil
   end
   return until_ns, id
end

local function active_until(value, now)
   if string.len(value) ~= string.len(now) then
      return string.len(value) > string.len(now)
   end
   return value >= now
end

local function id_greater(a, b)
   local ams, aseq = string.match(a, "^(%d+)%-(%d+)$")
   local bms, bseq = string.match(b, "^(%d+)%-(%d+)$")
   if not ams or not bms then
      return true
   end
   ams, aseq, bms, bseq = tonumber(ams), tonumber(aseq), tonumber(bms), tonumber(bseq)
   if ams ~= bms then
      return ams > bms
   end
   return aseq > bseq
end

local function in_flight(stream, group, id, now_ms, max_age_ms)
   if not id then
      return false
   end
   local ms = tonumber(string.match(id, "^(%d+)%-"))
   if now_ms - ms > max_age_ms then
      return false
   end
   local pending = redis.pcall("XPENDING", stream, group, id, id, 1)
   local no_group = type(pending) == "table" and pending.err ~= nil
   if not no_group and #pending > 0 then
      return true
   end
   if #redis.call("XRANGE", stream, id, id) == 0 then
      return false
   end
   if no_group then
      return true
   end
   local groups = redis.call("XINFO", "GROUPS", stream)
   for _, info in ipairs(groups) do
      local name, last
      for i = 1, #info, 2 do
         if info[i] == "name" then
            name = info[i + 1]
         elseif info[i] == "last-delivered-id" then
            last = info[i + 1]
         end
      end
      if name == group then
         return id_greater(id, last)
      end
   end
   return true
end
`

var (
	// luaClaimDispatch atomically admits a new external dispatch and adds its
	// start event. A job key may be claimed only when no durable payload
	// exists and no pending guard blocks it. A guard blocks while its TTL has
	// not passed or while its start event is in flight. A malformed guard is
	// rejected because it indicates corrupted coordination state. On success
	// the script adds the start event to the pool stream, trimmed as
	// Stream.Add trims it, and writes the guard "untilNanos:eventID", so the
	// guard is never written without its event and the event is never queued
	// without its guard. The pool stream has no TTL, so none is applied.
	//
	// KEYS: payload content, pending content, pending channel, pool stream.
	// ARGV: key, nowNanos, untilNanos, maxLen, job bytes, sink group, nowMs,
	// maxAgeMs.
	luaClaimDispatch = redis.NewScript(luaDispatchGuard + `
local payload = redis.call("HGET", KEYS[1], ARGV[1])
if payload then
   return {3, ""}
end

local pending = redis.call("HGET", KEYS[2], ARGV[1])
if pending then
   local until_ns, id = parse_guard(pending)
   if not until_ns then
      return {4, pending}
   end
   if active_until(until_ns, ARGV[2]) or in_flight(KEYS[4], ARGV[6], id, tonumber(ARGV[7]), tonumber(ARGV[8])) then
      return {2, pending}
   end
end

local event_id
if tonumber(ARGV[4]) > 0 then
   event_id = redis.call("XADD", KEYS[4], "MAXLEN", "~", ARGV[4], "*", "n", "j", "p", ARGV[5])
else
   event_id = redis.call("XADD", KEYS[4], "*", "n", "j", "p", ARGV[5])
end
local guard = ARGV[3] .. ":" .. event_id

redis.call("HSET", KEYS[2], ARGV[1], guard)
local rev = tostring(redis.call("HINCRBY", KEYS[2], "=rev", 1))
redis.call("HSET", KEYS[2], "=kind", "set")
local msg = struct.pack("ic0ic0ic0", string.len(ARGV[1]), ARGV[1], string.len(guard), guard, string.len(rev), rev)
redis.call("PUBLISH", KEYS[3], "set:" .. msg)
return {1, guard}
`)

	// luaReleaseDispatch removes the pending guard only if it still has the
	// value the caller read and its start event is not in flight. It
	// publishes a delete notification so rmap replicas converge without
	// relying on a later cleanup sweep.
	//
	// KEYS: pending content, pending channel, pool stream.
	// ARGV: key, expected guard, sink group, nowMs, maxAgeMs.
	luaReleaseDispatch = redis.NewScript(luaDispatchGuard + `
local pending = redis.call("HGET", KEYS[1], ARGV[1])
if pending ~= ARGV[2] then
   return 0
end
local _, id = parse_guard(pending)
if in_flight(KEYS[3], ARGV[3], id, tonumber(ARGV[4]), tonumber(ARGV[5])) then
   return 2
end

redis.call("HDEL", KEYS[1], ARGV[1])
local rev = tostring(redis.call("HINCRBY", KEYS[1], "=rev", 1))
redis.call("HSET", KEYS[1], "=kind", "del")
local msg = struct.pack("ic0ic0", string.len(ARGV[1]), ARGV[1], string.len(rev), rev)
redis.call("PUBLISH", KEYS[2], "del:" .. msg)
return 1
`)
)
