// Package redistest provides the Redis server used by the pulse test suites.
//
// By default Start runs an in-process miniredis server, so `go test` stays
// hermetic. When the LOOM_PULSE_REDIS_ADDR environment variable holds a
// host:port, Start connects to that real Redis server instead. This is the
// opt-in real-Redis tier run by `make test-pulse-redis`. The server must be
// on a loopback address unless LOOM_PULSE_REDIS_ALLOW_REMOTE is "1".
//
// A real server is shared by every test. Each test package uses its own
// database number and Start flushes that database, so the server must be
// disposable. Tests in one package run sequentially. Pub/sub channels are
// shared by all databases, so run the packages one at a time (`go test -p 1`)
// against a real server.
package redistest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type (
	// Server is the Redis server of one test.
	Server struct {
		// Client is connected to the server. It is closed when the test
		// ends.
		Client *redis.Client
		// mini is the in-process server, nil for a real server.
		mini *miniredis.Miniredis
		// major is the major version of a real server.
		major int
	}

	// structShimHook rewrites EVAL and SCRIPT LOAD commands on their way to
	// miniredis so scripts that use the Redis struct library keep working.
	structShimHook struct{}
)

// AddrEnv is the environment variable that selects a real Redis server. Its
// value is the host:port of a disposable server.
const AddrEnv = "LOOM_PULSE_REDIS_ADDR"

// AllowRemoteEnv is the environment variable that, set to "1", lets AddrEnv
// name a server that is not on a loopback address. Without it Start refuses
// such a server, so the tests never flush a shared Redis by accident.
const AllowRemoteEnv = "LOOM_PULSE_REDIS_ALLOW_REMOTE"

// Database numbers of the test packages. Each package flushes only its own
// database on a shared real server.
const (
	// DBRmap is the database of the pulse/rmap tests.
	DBRmap = 1
	// DBStreaming is the database of the pulse/streaming tests.
	DBStreaming = 2
	// DBPool is the database of the pulse/pool tests.
	DBPool = 3
)

// luaStructShim is a pure-Lua replacement for the Redis "struct" library used
// by the rmap and pool scripts. miniredis does not ship the struct library so
// the miniredis client rewrites scripts to prepend this shim. Only the
// struct.pack "i" (4-byte little-endian length) and "c0" (raw string) format
// codes used by the production scripts are implemented.
const luaStructShim = `
local struct = { pack = function(fmt, ...)
   local args = {...}
   local out = {}
   local ai = 1
   local i = 1
   while i <= string.len(fmt) do
      local c = string.sub(fmt, i, i)
      if c == "i" then
         local n = args[ai]
         ai = ai + 1
         out[#out+1] = string.char(n % 256, math.floor(n / 256) % 256, math.floor(n / 65536) % 256, math.floor(n / 16777216) % 256)
      elseif c == "c" then
         while i < string.len(fmt) and string.match(string.sub(fmt, i + 1, i + 1), "%d") do
            i = i + 1
         end
         out[#out+1] = args[ai]
         ai = ai + 1
      end
      i = i + 1
   end
   return table.concat(out)
end }
`

// Start returns the Redis server of test t. With AddrEnv set it connects to
// that server, selects database db and flushes it. Otherwise it runs a new
// miniredis server whose client rewrites Lua scripts to work around the
// missing struct library. The client is closed when the test ends.
func Start(t testing.TB, db int) *Server {
	t.Helper()
	addr := os.Getenv(AddrEnv)
	if addr == "" {
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		rdb.AddHook(structShimHook{})
		closeOnCleanup(t, rdb)
		return &Server{Client: rdb, mini: mr}
	}
	require.NoError(t, checkAddr(addr, os.Getenv(AllowRemoteEnv) == "1"))
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: db})
	closeOnCleanup(t, rdb)
	ctx := context.Background()
	info, err := rdb.Info(ctx, "server").Result()
	require.NoError(t, err, "connect to %s=%s", AddrEnv, addr)
	major, err := majorVersion(info)
	require.NoError(t, err)
	require.NoError(t, rdb.FlushDB(ctx).Err())
	return &Server{Client: rdb, major: major}
}

// Miniredis returns the in-process server, or nil for a real server.
func (s *Server) Miniredis() *miniredis.Miniredis {
	return s.mini
}

// Real reports whether the server is a real Redis server.
func (s *Server) Real() bool {
	return s.mini == nil
}

// MajorVersion returns the major version of a real server, or 0 for
// miniredis.
func (s *Server) MajorVersion() int {
	return s.major
}

// SkipOnRedis6 skips test t on a real Redis 6 server because of the known
// product bug tracked by issue, for example "#408". The test still runs on
// miniredis and on Redis 7 and later. Remove the call when the issue is
// fixed.
func (s *Server) SkipOnRedis6(t testing.TB, issue string) {
	t.Helper()
	if s.Real() && s.major < 7 {
		t.Skipf("known Redis 6.2 bug, see %s", issue)
	}
}

// DialHook implements redis.Hook.
func (structShimHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook implements redis.Hook.
func (structShimHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		shimStructScript(cmd)
		return next(ctx, cmd)
	}
}

// ProcessPipelineHook implements redis.Hook.
func (structShimHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			shimStructScript(cmd)
		}
		return next(ctx, cmds)
	}
}

// checkAddr returns an error unless addr is a host:port on a loopback
// address, or allowRemote is set.
func checkAddr(addr string, allowRemote bool) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%s=%q is not host:port: %w", AddrEnv, addr, err)
	}
	if allowRemote || host == "localhost" {
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("%s=%q is not a loopback address and the tests flush it; set %s=1 to use a disposable remote server", AddrEnv, addr, AllowRemoteEnv)
}

// closeOnCleanup closes rdb when the test ends.
func closeOnCleanup(t testing.TB, rdb *redis.Client) {
	t.Helper()
	t.Cleanup(func() {
		require.NoError(t, rdb.Close())
	})
}

// majorVersion returns the major version from the reply of INFO server.
func majorVersion(info string) (int, error) {
	for line := range strings.Lines(info) {
		version, ok := strings.CutPrefix(strings.TrimSpace(line), "redis_version:")
		if !ok {
			continue
		}
		major, _, _ := strings.Cut(version, ".")
		return strconv.Atoi(major)
	}
	return 0, errors.New("INFO server reply has no redis_version")
}

// shimStructScript prepends the struct shim to Lua sources sent via EVAL or
// SCRIPT LOAD. EVALSHA calls issued for the original source fail with NOSCRIPT
// and go-redis falls back to EVAL, which is then rewritten here.
func shimStructScript(cmd redis.Cmder) {
	args := cmd.Args()
	switch cmd.Name() {
	case "eval":
		if len(args) > 1 {
			if src, ok := args[1].(string); ok && strings.Contains(src, "struct.pack") {
				args[1] = luaStructShim + src
			}
		}
	case "script":
		if len(args) > 2 {
			if sub, ok := args[1].(string); ok && strings.EqualFold(sub, "load") {
				if src, ok := args[2].(string); ok && strings.Contains(src, "struct.pack") {
					args[2] = luaStructShim + src
				}
			}
		}
	}
}
