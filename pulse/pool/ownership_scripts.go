package pool

import redis "github.com/redis/go-redis/v9"

// luaOwnerMaps preserves rmap's revision and binary pub/sub protocol. Pool
// maps have no TTL. Epochs are read as strings after HINCRBY so Lua's floating
// point numbers never round a fencing token.
const luaOwnerMaps = `
local function set_map(content, channel, key, value)
   redis.call("HSET", content, key, value)
   local rev = tostring(redis.call("HINCRBY", content, "=rev", 1))
   redis.call("HSET", content, "=kind", "set")
   redis.call("PUBLISH", channel, "set:" .. struct.pack("ic0ic0ic0", #key, key, #value, value, #rev, rev))
end
local function del_map(content, channel, key)
   if redis.call("HDEL", content, key) == 0 then return end
   local rev = tostring(redis.call("HINCRBY", content, "=rev", 1))
   redis.call("HSET", content, "=kind", "del")
   redis.call("PUBLISH", channel, "del:" .. struct.pack("ic0ic0", #key, key, #rev, rev))
end
local function job_keys(value)
   if not value then return {} end
   local ok, decoded = pcall(cjson.decode, value)
   if ok and type(decoded) == "table" then return decoded end
   local keys = {}
   for key in string.gmatch(value, "[^,]+") do keys[#keys + 1] = key end
   return keys
end
local function now_ms()
   local t = redis.call("TIME")
   return tonumber(t[1]) * 1000 + math.floor(tonumber(t[2]) / 1000)
end
local function live(value, ttl)
   return value and tonumber(value) and now_ms() - tonumber(value) / 1000000 <= tonumber(ttl)
end
local function incompatible(nodes, protocols, ttl)
   local entries = redis.call("HGETALL", nodes)
   for i = 1, #entries, 2 do
      if string.sub(entries[i], 1, 1) ~= "=" and live(entries[i + 1], ttl)
         and redis.call("HGET", protocols, entries[i]) ~= "2" then
         return entries[i]
      end
   end
   return ""
end
`

var (
	// KEYS: owners, epochs, jobs content/channel, payload content/channel,
	// workers, worker keep-alive, node keep-alive, protocols.
	// ARGV: job key, worker ID, payload, worker TTL milliseconds.
	luaClaimJob = redis.NewScript(luaOwnerMaps + `
local bad = incompatible(KEYS[9], KEYS[10], ARGV[4])
if bad ~= "" then return redis.error_reply("incompatible pool node " .. bad) end
local worker = redis.call("HGET", KEYS[7], ARGV[2])
if not worker or worker == "-" or not live(redis.call("HGET", KEYS[8], ARGV[2]), ARGV[4]) then
   return redis.error_reply("worker is not active")
end
local current = redis.call("HGET", KEYS[1], ARGV[1])
if current then
   local owner, epoch = string.match(current, "^(.*):(%d+)$")
   if owner ~= ARGV[2] then return {0, ""} end
   return {2, epoch}
end
local keys = job_keys(redis.call("HGET", KEYS[3], ARGV[2]))
local found = false
for _, key in ipairs(keys) do if key == ARGV[1] then found = true end end
if not found then keys[#keys + 1] = ARGV[1] end
local encoded = cjson.encode(keys)
redis.call("HINCRBY", KEYS[2], ARGV[1], 1)
local epoch = redis.call("HGET", KEYS[2], ARGV[1])
redis.call("HSET", KEYS[1], ARGV[1], ARGV[2] .. ":" .. epoch)
set_map(KEYS[3], KEYS[4], ARGV[2], encoded)
set_map(KEYS[5], KEYS[6], ARGV[1], ARGV[3])
return {1, epoch}
`)

	// KEYS: owners, jobs content/channel, payload content/channel.
	// ARGV: key, worker ID, epoch, delete payload (1 or 0).
	luaReleaseJob = redis.NewScript(luaOwnerMaps + `
if redis.call("HGET", KEYS[1], ARGV[1]) ~= ARGV[2] .. ":" .. ARGV[3] then return 0 end
local remaining = {}
for _, key in ipairs(job_keys(redis.call("HGET", KEYS[2], ARGV[2]))) do
   if key ~= ARGV[1] then remaining[#remaining + 1] = key end
end
redis.call("HDEL", KEYS[1], ARGV[1])
if #remaining == 0 then del_map(KEYS[2], KEYS[3], ARGV[2])
else set_map(KEYS[2], KEYS[3], ARGV[2], cjson.encode(remaining)) end
if ARGV[4] == "1" then del_map(KEYS[4], KEYS[5], ARGV[1]) end
return 1
`)

	// KEYS: owners, cleanup content/channel, keepalive content/channel,
	// workers content/channel, jobs content/channel, payloads, pool stream,
	// worker stream, pending content/channel.
	// ARGV: worker ID, lock token, TTL ms, node ID, createdAt (8 wire bytes),
	// now nanoseconds, stream maximum length.
	luaCleanupOwner = redis.NewScript(luaOwnerMaps + luaDispatchGuard + `
if redis.call("HGET", KEYS[2], ARGV[1]) ~= ARGV[2] then return 0 end
if live(redis.call("HGET", KEYS[4], ARGV[1]), ARGV[3]) then
   del_map(KEYS[2], KEYS[3], ARGV[1])
   return 1
end
local entries = redis.call("HGETALL", KEYS[1])
for i = 1, #entries, 2 do
   local owner = string.match(entries[i + 1], "^(.*):%d+$")
   if owner == ARGV[1] then
      local key = entries[i]
      local payload = redis.call("HGET", KEYS[10], key)
      if payload then
         local job = struct.pack("ic0ic0ic0", #key, key, #ARGV[4], ARGV[4], #payload, payload) .. ARGV[5]
         add_start(KEYS[11], ARGV[7], job, KEYS[13], KEYS[14], key, ARGV[6])
      end
      redis.call("HDEL", KEYS[1], key)
   end
end
del_map(KEYS[6], KEYS[7], ARGV[1])
del_map(KEYS[4], KEYS[5], ARGV[1])
del_map(KEYS[8], KEYS[9], ARGV[1])
del_map(KEYS[2], KEYS[3], ARGV[1])
redis.call("DEL", KEYS[12])
return 2
`)

	// Registration and backfill are atomic with the initial node heartbeat.
	// KEYS: protocols, node keepalive content/channel, marker, owners,
	// epochs, jobs, worker keepalive. ARGV: node ID, TTL ms.
	luaJoinProtocol = redis.NewScript(luaOwnerMaps + `
local bad = incompatible(KEYS[2], KEYS[1], ARGV[2])
if bad ~= "" then return redis.error_reply("incompatible pool node " .. bad) end
local version = redis.call("GET", KEYS[4])
if version and version ~= "2" then return redis.error_reply("unsupported pool protocol " .. version) end
if not version then
   local entries = redis.call("HGETALL", KEYS[7])
   for i = 1, #entries, 2 do
      local worker = entries[i]
      if string.sub(worker, 1, 1) ~= "=" and live(redis.call("HGET", KEYS[8], worker), ARGV[2]) then
         for _, key in ipairs(job_keys(entries[i + 1])) do
            if redis.call("HEXISTS", KEYS[5], key) == 0 then
               redis.call("HINCRBY", KEYS[6], key, 1)
               redis.call("HSET", KEYS[5], key, worker .. ":" .. redis.call("HGET", KEYS[6], key))
            end
         end
      end
   end
   redis.call("SET", KEYS[4], "2")
end
redis.call("HSET", KEYS[1], ARGV[1], "2")
set_map(KEYS[2], KEYS[3], ARGV[1], string.format("%.0f", now_ms() * 1000000))
return 1
`)

	// KEYS: node keepalive, protocols. ARGV: TTL ms.
	luaCheckProtocol = redis.NewScript(luaOwnerMaps + `return incompatible(KEYS[1], KEYS[2], ARGV[1])`)

	// Use Redis time for leases so cleanup does not depend on node clock skew.
	// KEYS: keepalive content/channel, worker map. ARGV: worker ID.
	luaOwnerHeartbeat = redis.NewScript(luaOwnerMaps + `
local worker = redis.call("HGET", KEYS[3], ARGV[1])
if not worker or worker == "-" then return redis.error_reply("worker is no longer registered") end
set_map(KEYS[1], KEYS[2], ARGV[1], string.format("%.0f", now_ms() * 1000000))
return 1
`)
)
