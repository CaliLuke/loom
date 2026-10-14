// Package rmapstate owns the mutation protocol shared by replicated maps and
// atomic Pool operations that write those maps alongside other Redis state.
package rmapstate

import (
	"context"
	"fmt"
	"strings"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/oklog/ulid/v2"
)

// Script adds a generation candidate and protocol preflight to an atomic script.
// The embedded Redis script retains the standard loading and hash operations.
type Script struct {
	*redis.Script
}

// NewScript declares the one-based KEYS positions of every replicated map the
// body may mutate. Admission runs before any body command, including non-map
// writes, because Redis Lua errors do not roll earlier commands back.
// The body may call publish_map, set_map, del_map and clear_map from the shared
// protocol. Ordinary ARGV positions are unchanged; the candidate is appended.
func NewScript(mapKeys []int, body string) *Script {
	var source strings.Builder
	source.WriteString(luaProtocol)
	for _, key := range mapKeys {
		fmt.Fprintf(&source, "\nlocal admission = validate_map(KEYS[%d], true)\nif admission then return redis.error_reply(admission) end\n", key)
	}
	source.WriteString(body)
	return &Script{Script: redis.NewScript(source.String())}
}

// Run appends a fresh candidate for this atomic operation. Redis retains an
// existing incarnation or installs this candidate on the first actual mutation.
// Generation uniqueness assumes independent collision-resistant ULID candidates.
func (s *Script) Run(ctx context.Context, client redis.Scripter, keys []string, args ...any) *redis.Cmd {
	args = append(args[:len(args):len(args)], ulid.Make().String())
	return s.Script.Run(ctx, client, keys, args...)
}

const luaProtocol = `
local candidate_id = ARGV[#ARGV]
local changed_maps = {}
local function valid_revision(value)
   return value and string.match(value, "^%d+$") and
      (#value == 1 or string.sub(value, 1, 1) ~= "0") and
      (#value < 19 or (#value == 19 and value <= "9223372036854775807"))
end
local function later_than(left, right)
   return #left > #right or (#left == #right and left > right)
end
local function validate_map(content, writing)
   local protocol = redis.call("HGET", content, "=protocol")
   if not protocol then
      local allowed = { ["=rev"]=true, ["=kind"]=true }
      for _, key in ipairs(redis.call("HKEYS", content)) do
         if string.find(key, "=", 1, true) and not allowed[key] then
            return "unknown replicated-map metadata: " .. key
         end
      end
      return false
   end
   if protocol ~= "1" then return "unsupported replicated-map protocol" end
   local generation = redis.call("HGET", content, "=generation")
   if not generation or generation == "" then return "invalid replicated-map generation" end
   local revisions = redis.call("HMGET", content, "=rev", "=reset", "=delete")
   for _, revision in ipairs(revisions) do
      if not valid_revision(revision) then return "invalid replicated-map revision metadata" end
   end
   if later_than(revisions[2], revisions[1]) or later_than(revisions[3], revisions[1]) then
      return "invalid replicated-map recovery metadata"
   end
   if writing and revisions[1] == "9223372036854775807" then
      return "replicated-map revision exhausted"
   end
   return false
end
local function initialize_map(content)
   local generation = redis.call("HGET", content, "=generation")
   if not redis.call("HGET", content, "=protocol") then
      generation = candidate_id
      redis.call("HDEL", content, "=kind")
      redis.call("HSET", content, "=protocol", "1", "=generation", generation,
                 "=rev", "0", "=reset", "0", "=delete", "0")
   end
   return generation
end
local function publish_map(content, channel, kind, key, value)
   local generation = initialize_map(content)
   redis.call("HINCRBY", content, "=rev", 1)
   local revision = redis.call("HGET", content, "=rev")
   if kind == "reset" or kind == "destroy" then
      redis.call("HSET", content, "=reset", revision)
   elseif kind == "del" then
      redis.call("HSET", content, "=delete", revision)
   end
   local values = {"1", generation, revision}
   if key then table.insert(values, key) end
   if value then table.insert(values, value) end
   local parts = {}
   for _, item in ipairs(values) do
      table.insert(parts, struct.pack("ic0", #item, item))
   end
   redis.call("PUBLISH", channel, kind .. ":" .. table.concat(parts))
   changed_maps[content] = true
   return revision, generation
end
local function set_map(content, channel, key, value)
   redis.call("HSET", content, key, value)
   return publish_map(content, channel, "set", key, value)
end
local function del_map(content, channel, key)
   if redis.call("HDEL", content, key) == 0 then return end
   return publish_map(content, channel, "del", key)
end
local function clear_map(content, channel, kind)
   local generation = initialize_map(content)
   local revision = redis.call("HGET", content, "=rev")
   local deleted = redis.call("HGET", content, "=delete")
   local reset = redis.call("HGET", content, "=reset")
   redis.call("DEL", content)
   redis.call("HSET", content, "=protocol", "1", "=generation", generation,
              "=rev", revision, "=reset", reset, "=delete", deleted)
   return publish_map(content, channel, kind)
end
`
