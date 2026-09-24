package rmap

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/internal/redistest"
)

const (
	// testTTL is the map TTL of the TTL tests.
	testTTL = time.Hour
	// testShortTTL is the remaining TTL the TTL tests give the hash before
	// a write, as if most of testTTL had elapsed.
	testShortTTL = 10 * time.Minute
)

// TestTTLAppliedByWrites checks that every write sets the TTL of the map
// hash inside its script, on Redis 6.2 and later. A fixed TTL is set on the
// first write and never extended by later writes. A sliding TTL is reset by
// every write.
func TestTTLAppliedByWrites(t *testing.T) {
	writes := []struct {
		name  string
		write func(ctx context.Context, m *Map) error
	}{
		{name: "set", write: func(ctx context.Context, m *Map) error {
			_, err := m.Set(ctx, "key", "other")
			return err
		}},
		{name: "set if not exists", write: func(ctx context.Context, m *Map) error {
			_, err := m.SetIfNotExists(ctx, "new", "value")
			return err
		}},
		{name: "test and set", write: func(ctx context.Context, m *Map) error {
			_, err := m.TestAndSet(ctx, "key", "value", "other")
			return err
		}},
		{name: "inc", write: func(ctx context.Context, m *Map) error {
			_, err := m.Inc(ctx, "counter", 1)
			return err
		}},
		{name: "append", write: func(ctx context.Context, m *Map) error {
			_, err := m.AppendValues(ctx, "list", "a")
			return err
		}},
		{name: "append unique", write: func(ctx context.Context, m *Map) error {
			_, err := m.AppendUniqueValues(ctx, "list", "a")
			return err
		}},
		{name: "remove", write: func(ctx context.Context, m *Map) error {
			_, _, err := m.RemoveValues(ctx, "items", "a")
			return err
		}},
		{name: "delete", write: func(ctx context.Context, m *Map) error {
			_, err := m.Delete(ctx, "key")
			return err
		}},
		{name: "test and delete", write: func(ctx context.Context, m *Map) error {
			_, err := m.TestAndDelete(ctx, "key", "value")
			return err
		}},
	}
	modes := []struct {
		name string
		opt  MapOption
		// extended reports whether a write extends the remaining TTL.
		extended bool
	}{
		{name: "fixed", opt: WithTTL(testTTL)},
		{name: "sliding", opt: WithSlidingTTL(testTTL), extended: true},
	}
	for _, mode := range modes {
		for _, w := range writes {
			t.Run(mode.name+"/"+w.name, func(t *testing.T) {
				srv := redistest.Start(t, redistest.DBRmap)
				rdb := srv.Client
				ctx := t.Context()
				rec := redistest.Record(rdb)
				m := joinMap(t, rdb, "ttlwrites", mode.opt)
				hash := "map:ttlwrites:content"

				// The first write sets the TTL.
				_, err := m.Set(ctx, "key", "value")
				require.NoError(t, err)
				_, err = m.AppendValues(ctx, "items", "a", "b")
				require.NoError(t, err)
				assertTTLBetween(t, rdb, hash, testShortTTL, testTTL)

				// Pretend most of the TTL elapsed, then write again.
				require.NoError(t, rdb.PExpire(ctx, hash, testShortTTL).Err())
				rec.Reset()
				require.NoError(t, w.write(ctx, m))
				assertOneScriptCall(t, rec.Names())
				if mode.extended {
					assertTTLBetween(t, rdb, hash, testShortTTL, testTTL)
				} else {
					assertTTLBetween(t, rdb, hash, 0, testShortTTL)
				}
			})
		}
	}
}

// TestFixedTTLSetOnlyWhenAbsent checks that a fixed TTL is set whenever the
// map hash has no TTL: on a hash created without a TTL, and again after
// Reset recreates the hash.
func TestFixedTTLSetOnlyWhenAbsent(t *testing.T) {
	srv := redistest.Start(t, redistest.DBRmap)
	rdb := srv.Client
	ctx := t.Context()
	hash := "map:ttlabsent:content"

	plain := joinMap(t, rdb, "ttlabsent")
	_, err := plain.Set(ctx, "key", "value")
	require.NoError(t, err)
	ttl, err := rdb.PTTL(ctx, hash).Result()
	require.NoError(t, err)
	assert.Equal(t, time.Duration(-1), ttl, "a map without TTL leaves the hash persistent")

	fixed := joinMap(t, rdb, "ttlabsent", WithTTL(testTTL))
	_, err = fixed.Set(ctx, "key", "other")
	require.NoError(t, err)
	assertTTLBetween(t, rdb, hash, testShortTTL, testTTL)

	require.NoError(t, rdb.PExpire(ctx, hash, testShortTTL).Err())
	require.NoError(t, fixed.Reset(ctx))
	assertTTLBetween(t, rdb, hash, testShortTTL, testTTL)
}

// assertTTLBetween asserts that key has a remaining TTL in (low, high].
func assertTTLBetween(t *testing.T, rdb *redis.Client, key string, low, high time.Duration) {
	t.Helper()
	ttl, err := rdb.PTTL(t.Context(), key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, low, "TTL of %s", key)
	assert.LessOrEqual(t, ttl, high, "TTL of %s", key)
}

// assertOneScriptCall asserts that names, the commands of one map write, only
// run the write script. EVALSHA may fall back to EVAL, but no separate expiry
// command may follow the script.
func assertOneScriptCall(t *testing.T, names []string) {
	t.Helper()
	require.NotEmpty(t, names)
	for _, name := range names {
		assert.Contains(t, []string{"evalsha", "eval"}, name, "commands %v", names)
	}
}
