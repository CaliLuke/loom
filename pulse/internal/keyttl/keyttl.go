// Package keyttl applies the TTL of a pulse Redis key inside the Lua script
// that writes the key, so the write and its expiry are one atomic step.
//
// A fixed TTL is set only when the key has no TTL, the semantics of
// `EXPIRE key seconds NX`. The NX flag needs Redis 7.0, so the script checks
// `TTL key` instead, which works on Redis 6.2 and later and on miniredis. A
// sliding TTL is reset by every write.
package keyttl

import "time"

// LuaApply is Lua source that defines the local function
// apply_ttl(key, seconds, sliding). Prepend it to a script. The function sets
// the TTL of key to seconds when seconds is positive and either sliding is
// "1" or key has no TTL. It does nothing when key does not exist. Pass
// seconds and sliding as script arguments built by Args.
const LuaApply = `
local function apply_ttl(key, seconds, sliding)
   seconds = tonumber(seconds)
   if seconds <= 0 then
      return
   end
   if sliding == "1" or redis.call("TTL", key) == -1 then
      redis.call("EXPIRE", key, seconds)
   end
end
`

// Args returns the seconds and sliding arguments of apply_ttl for ttl. A TTL
// below one second rounds up to one second, as the go-redis Expire command
// does. A zero or negative ttl disables the expiry.
func Args(ttl time.Duration, sliding bool) (seconds int64, slidingArg string) {
	slidingArg = "0"
	if sliding {
		slidingArg = "1"
	}
	switch {
	case ttl <= 0:
		return 0, slidingArg
	case ttl < time.Second:
		return 1, slidingArg
	default:
		return int64(ttl / time.Second), slidingArg
	}
}
