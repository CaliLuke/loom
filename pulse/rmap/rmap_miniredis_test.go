package rmap

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// luaStructShim is a pure-Lua replacement for the Redis "struct" library used
// by the rmap mutation scripts. miniredis does not ship the struct library so
// the test client rewrites scripts to prepend this shim. Only the struct.pack
// "i" (4-byte little-endian length) and "c0" (raw string) format codes used by
// the production scripts are implemented.
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

// structShimHook rewrites EVAL and SCRIPT LOAD commands on their way to
// miniredis so scripts that use the Redis struct library keep working.
type structShimHook struct{}

func (structShimHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (structShimHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		shimStructScript(cmd)
		return next(ctx, cmd)
	}
}

func (structShimHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			shimStructScript(cmd)
		}
		return next(ctx, cmds)
	}
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

// startTestRedis runs an in-process miniredis server and returns a client
// whose Lua scripts are rewritten to work around the missing struct library.
func startTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rdb.AddHook(structShimHook{})
	t.Cleanup(func() {
		require.NoError(t, rdb.Close())
	})
	return rdb
}

// joinMap joins the replicated map with the given name and closes it when the
// test completes.
func joinMap(t *testing.T, rdb *redis.Client, name string, opts ...MapOption) *Map {
	t.Helper()
	m, err := Join(t.Context(), name, rdb, opts...)
	require.NoError(t, err)
	t.Cleanup(m.Close)
	return m
}

// waitForKind consumes notifications from c until kind arrives.
func waitForKind(t *testing.T, c <-chan EventKind, kind EventKind) {
	t.Helper()
	for {
		select {
		case got, ok := <-c:
			require.True(t, ok, "notification channel closed while waiting for kind %v", kind)
			if got == kind {
				return
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for event kind %v", kind)
		}
	}
}

func TestJoinValidation(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()

	tests := []struct {
		name    string
		mapName string
		rdb     *redis.Client
		ctx     context.Context
		opts    []MapOption
		errPart string
	}{
		{name: "invalid map name", mapName: "bad name", rdb: rdb, ctx: ctx, errPart: "not a valid map name"},
		{name: "nil redis client", mapName: "valid", rdb: nil, ctx: ctx, errPart: "Redis client cannot be nil"},
		{name: "negative ttl", mapName: "valid", rdb: rdb, ctx: ctx, opts: []MapOption{WithTTL(-time.Second)}, errPart: "ttl must be >= 0"},
		{name: "canceled context", mapName: "valid", rdb: rdb, ctx: canceledContext(t), errPart: "context canceled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Join(tt.ctx, tt.mapName, tt.rdb, tt.opts...)
			require.ErrorContains(t, err, tt.errPart)
		})
	}
}

func TestSetGetRoundTrip(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "roundtrip")

	prev, err := m.Set(ctx, "color", "blue")
	require.NoError(t, err)
	require.Empty(t, prev)

	prev, existed, err := m.SetEx(ctx, "color", "green")
	require.NoError(t, err)
	require.Equal(t, "blue", prev)
	require.True(t, existed)

	prev, err = m.SetAndWait(ctx, "color", "red")
	require.NoError(t, err)
	require.Equal(t, "green", prev)

	// SetAndWait guarantees the local replica observed the write.
	v, ok := m.Get("color")
	require.True(t, ok)
	require.Equal(t, "red", v)
	require.Equal(t, map[string]string{"color": "red"}, m.Map())
	require.Equal(t, []string{"color"}, m.Keys())
	require.Equal(t, 1, m.Len())

	_, ok = m.Get("missing")
	require.False(t, ok)
}

func TestSetIfNotExists(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "setnx")

	set, err := m.SetIfNotExists(ctx, "key", "first")
	require.NoError(t, err)
	require.True(t, set)

	set, err = m.SetIfNotExists(ctx, "key", "second")
	require.NoError(t, err)
	require.False(t, set)

	require.Eventually(t, func() bool {
		v, ok := m.Get("key")
		return ok && v == "first"
	}, 5*time.Second, 5*time.Millisecond)
}

func TestTestAndSet(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "tas")

	_, err := m.Set(ctx, "color", "blue")
	require.NoError(t, err)

	prev, err := m.TestAndSet(ctx, "color", "red", "yellow")
	require.NoError(t, err)
	require.Equal(t, "blue", prev, "mismatched test value must not update")

	prev, existed, updated, err := m.TestAndSetEx(ctx, "color", "blue", "yellow")
	require.NoError(t, err)
	require.Equal(t, "blue", prev)
	require.True(t, existed)
	require.True(t, updated)

	prev, existed, updated, err = m.TestAndSetEx(ctx, "missing", "x", "y")
	require.NoError(t, err)
	require.Empty(t, prev)
	require.False(t, existed)
	require.False(t, updated)

	require.Eventually(t, func() bool {
		v, ok := m.Get("color")
		return ok && v == "yellow"
	}, 5*time.Second, 5*time.Millisecond)
}

func TestIncAndDelete(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "incdel")

	n, err := m.Inc(ctx, "counter", 3)
	require.NoError(t, err)
	require.Equal(t, 3, n)

	n, err = m.Inc(ctx, "counter", -1)
	require.NoError(t, err)
	require.Equal(t, 2, n)

	prev, err := m.Delete(ctx, "counter")
	require.NoError(t, err)
	require.Equal(t, "2", prev)

	prev, existed, err := m.DeleteEx(ctx, "counter")
	require.NoError(t, err)
	require.Empty(t, prev)
	require.False(t, existed)

	require.Eventually(t, func() bool {
		_, ok := m.Get("counter")
		return !ok
	}, 5*time.Second, 5*time.Millisecond)
}

func TestTestAndDelete(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "tad")

	_, err := m.Set(ctx, "color", "blue")
	require.NoError(t, err)

	prev, err := m.TestAndDelete(ctx, "color", "red")
	require.NoError(t, err)
	require.Equal(t, "blue", prev, "mismatched test value must not delete")

	prev, existed, deleted, err := m.TestAndDeleteEx(ctx, "color", "blue")
	require.NoError(t, err)
	require.Equal(t, "blue", prev)
	require.True(t, existed)
	require.True(t, deleted)

	prev, existed, deleted, err = m.TestAndDeleteEx(ctx, "color", "blue")
	require.NoError(t, err)
	require.Empty(t, prev)
	require.False(t, existed)
	require.False(t, deleted)
}

func TestListValues(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "lists")

	vals, err := m.AppendValues(ctx, "fruits", "apple", "banana")
	require.NoError(t, err)
	require.Equal(t, []string{"apple", "banana"}, vals)

	vals, err = m.AppendUniqueValues(ctx, "fruits", "banana", "cherry")
	require.NoError(t, err)
	require.Equal(t, []string{"apple", "banana", "cherry"}, vals)

	require.Eventually(t, func() bool {
		vals, ok := m.GetValues("fruits")
		return ok && len(vals) == 3
	}, 5*time.Second, 5*time.Millisecond)

	remaining, removed, err := m.RemoveValues(ctx, "fruits", "apple", "cherry")
	require.NoError(t, err)
	require.True(t, removed)
	require.Equal(t, []string{"banana"}, remaining)

	remaining, removed, err = m.RemoveValues(ctx, "fruits", "missing")
	require.NoError(t, err)
	require.False(t, removed)
	require.Equal(t, []string{"banana"}, remaining)

	// Removing the last item deletes the key.
	remaining, removed, err = m.RemoveValues(ctx, "fruits", "banana")
	require.NoError(t, err)
	require.True(t, removed)
	require.Empty(t, remaining)

	remaining, removed, err = m.RemoveValues(ctx, "fruits", "banana")
	require.NoError(t, err)
	require.False(t, removed)
	require.Empty(t, remaining)

	require.Eventually(t, func() bool {
		_, ok := m.Get("fruits")
		return !ok
	}, 5*time.Second, 5*time.Millisecond)
}

func TestKeyValidation(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "keyval")

	tests := []struct {
		name string
		op   func() error
	}{
		{name: "set empty key", op: func() error {
			_, err := m.Set(ctx, "", "v")
			return err
		}},
		{name: "set key with equal sign", op: func() error {
			_, err := m.Set(ctx, "a=b", "v")
			return err
		}},
		{name: "inc key with equal sign", op: func() error {
			_, err := m.Inc(ctx, "a=b", 1)
			return err
		}},
		{name: "delete empty key", op: func() error {
			_, err := m.Delete(ctx, "")
			return err
		}},
		{name: "append key with equal sign", op: func() error {
			_, err := m.AppendValues(ctx, "a=b", "v")
			return err
		}},
		{name: "test and reset empty key", op: func() error {
			_, err := m.TestAndReset(ctx, []string{""}, []string{"v"})
			return err
		}},
		{name: "test and reset key with equal sign", op: func() error {
			_, err := m.TestAndReset(ctx, []string{"a=b"}, []string{"v"})
			return err
		}},
		{name: "test and reset length mismatch", op: func() error {
			_, err := m.TestAndReset(ctx, []string{"a", "b"}, []string{"v"})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.op())
		})
	}
}

func TestNotificationKinds(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "kinds")
	c := m.Subscribe()

	_, err := m.Set(ctx, "color", "blue")
	require.NoError(t, err)
	waitForKind(t, c, EventChange)

	_, err = m.Delete(ctx, "color")
	require.NoError(t, err)
	waitForKind(t, c, EventDelete)

	_, err = m.Set(ctx, "color", "blue")
	require.NoError(t, err)
	waitForKind(t, c, EventChange)

	require.NoError(t, m.Reset(ctx))
	waitForKind(t, c, EventReset)
	require.Equal(t, 0, m.Len())

	m.Unsubscribe(c)
	_, ok := <-c
	require.False(t, ok, "unsubscribed channel must be closed")
}

func TestReplicasConverge(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	a := joinMap(t, rdb, "converge")
	b := joinMap(t, rdb, "converge")

	_, err := a.Set(ctx, "shared", "from-a")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		v, ok := b.Get("shared")
		return ok && v == "from-a"
	}, 5*time.Second, 5*time.Millisecond)

	_, err = b.Delete(ctx, "shared")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		_, ok := a.Get("shared")
		return !ok
	}, 5*time.Second, 5*time.Millisecond)

	// A replica joining later must read the existing content.
	_, err = a.SetAndWait(ctx, "late", "value")
	require.NoError(t, err)
	late := joinMap(t, rdb, "converge")
	v, ok := late.Get("late")
	require.True(t, ok)
	require.Equal(t, "value", v)
}

func TestDestroyClearsReplicas(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	a := joinMap(t, rdb, "destroy")
	b := joinMap(t, rdb, "destroy")

	_, err := a.SetAndWait(ctx, "key", "value")
	require.NoError(t, err)
	require.NoError(t, a.Destroy(ctx))

	require.Eventually(t, func() bool {
		return b.Len() == 0
	}, 5*time.Second, 5*time.Millisecond)
}

func TestCloseSemantics(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "closesem")

	_, err := m.SetAndWait(ctx, "key", "value")
	require.NoError(t, err)

	m.Close()
	m.Close() // Close is idempotent.

	// The local replica remains a read-only point-in-time copy.
	v, ok := m.Get("key")
	require.True(t, ok)
	require.Equal(t, "value", v)

	_, err = m.Set(ctx, "key", "other")
	require.ErrorContains(t, err, "is stopped")
	_, err = m.SetAndWait(ctx, "key", "other")
	require.ErrorContains(t, err, "is stopped")
	_, err = m.Delete(ctx, "key")
	require.ErrorContains(t, err, "is stopped")
	require.Nil(t, m.Subscribe())

	// Reset and Destroy still mutate Redis state after Close.
	require.NoError(t, m.Reset(ctx))
	require.NoError(t, m.Destroy(ctx))

	// The Redis hash only holds the internal revision bookkeeping now.
	fresh := joinMap(t, rdb, "closesem")
	require.Equal(t, 0, fresh.Len())
}

func TestTTLApplied(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()

	fixed := joinMap(t, rdb, "ttlfixed", WithTTL(time.Minute))
	_, err := fixed.Set(ctx, "key", "value")
	require.NoError(t, err)
	ttl, err := rdb.TTL(ctx, "map:ttlfixed:content").Result()
	require.NoError(t, err)
	require.Positive(t, ttl)

	sliding := joinMap(t, rdb, "ttlsliding", WithSlidingTTL(time.Minute))
	_, err = sliding.Set(ctx, "key", "value")
	require.NoError(t, err)
	ttl, err = rdb.TTL(ctx, "map:ttlsliding:content").Result()
	require.NoError(t, err)
	require.Positive(t, ttl)
}

func TestTestAndReset(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	m := joinMap(t, rdb, "tar")

	_, err := m.SetAndWait(ctx, "color", "blue")
	require.NoError(t, err)
	_, err = m.SetAndWait(ctx, "size", "large")
	require.NoError(t, err)

	reset, err := m.TestAndReset(ctx, []string{"color", "size"}, []string{"blue", "small"})
	require.NoError(t, err)
	require.False(t, reset, "mismatched test values must not reset")
	require.Equal(t, 2, m.Len())

	reset, err = m.TestAndReset(ctx, []string{"color", "size"}, []string{"blue", "large"})
	require.NoError(t, err)
	require.True(t, reset)
	require.Eventually(t, func() bool {
		return m.Len() == 0
	}, 5*time.Second, 5*time.Millisecond)
}

func TestReconnectResyncsContent(t *testing.T) {
	rdb := startTestRedis(t)
	ctx := t.Context()
	a := joinMap(t, rdb, "reconnect")
	b := joinMap(t, rdb, "reconnect")

	_, err := a.SetAndWait(ctx, "before", "value")
	require.NoError(t, err)

	// Drop the pubsub connection of replica b to force the reconnect path.
	b.lock.RLock()
	sub := b.sub
	b.lock.RUnlock()
	require.NoError(t, sub.Close())

	// Writes made after the drop must eventually reach the reconnected replica.
	_, err = a.Set(ctx, "after", "value")
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		v, ok := b.Get("after")
		return ok && v == "value"
	}, 10*time.Second, 5*time.Millisecond)
}

// canceledContext returns a context that is already canceled.
func canceledContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	return ctx
}
