package pool

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/CaliLuke/loom/pulse/internal/redistest"
	"github.com/stretchr/testify/require"
)

func TestProtocolAdmissionMigration(t *testing.T) {
	for _, version := range []string{"", "3", "4", "2", "future", "04"} {
		t.Run("marker-"+version, func(t *testing.T) {
			rdb := startTestRedis(t)
			ctx := t.Context()
			keys := []string{"protocols", "nodes", "channel", "marker"}
			if version != "" {
				require.NoError(t, rdb.Set(ctx, keys[3], version, 0).Err())
			}
			require.NoError(t, rdb.HSet(ctx, keys[1], "old", "1", "=rev", "7", "=kind", "set").Err())
			require.NoError(t, rdb.PExpire(ctx, keys[1], time.Hour).Err())
			require.NoError(t, rdb.HSet(ctx, "business", "pending", "original-operation", "owner", "worker:42").Err())
			before, err := rdb.HGetAll(ctx, keys[1]).Result()
			require.NoError(t, err)
			ttl, err := rdb.PTTL(ctx, keys[1]).Result()
			require.NoError(t, err)
			_, err = luaJoinProtocol.Run(ctx, rdb, keys, "new", 60000).Result()
			if version == "" || version == "3" || version == "4" {
				require.NoError(t, err)
				marker, err := rdb.Get(ctx, keys[3]).Result()
				require.NoError(t, err)
				require.Equal(t, "4", marker)
				old, err := rdb.HGet(ctx, keys[1], "old").Result()
				require.NoError(t, err)
				require.Equal(t, "1", old)
				remaining, err := rdb.PTTL(ctx, keys[1]).Result()
				require.NoError(t, err)
				require.Positive(t, remaining)
				require.LessOrEqual(t, remaining, ttl)
			} else {
				require.ErrorContains(t, err, "unsupported pool protocol")
				after, err := rdb.HGetAll(ctx, keys[1]).Result()
				require.NoError(t, err)
				require.Equal(t, before, after)
				marker, err := rdb.Get(ctx, keys[3]).Result()
				require.NoError(t, err)
				require.Equal(t, version, marker)
			}
			business, err := rdb.HGetAll(ctx, "business").Result()
			require.NoError(t, err)
			require.Equal(t, map[string]string{"pending": "original-operation", "owner": "worker:42"}, business)
		})
	}
}

// The old admission is historical test input, not a supported runtime writer.
var oldProtocolAdmission = redis.NewScript(`
local version = redis.call("GET", KEYS[4])
if version and version ~= "3" then return redis.error_reply("unsupported pool protocol " .. version) end
redis.call("SET", KEYS[4], "3")
redis.call("HSET", KEYS[1], "old", "3")
redis.call("HSET", KEYS[2], "old", ARGV[1])
return 1
`)

func TestProtocolAdmissionConcurrentOldJoin(t *testing.T) {
	rdb := startTestRedis(t)
	for iteration := range 12 {
		keys := []string{fmt.Sprintf("protocols-%d", iteration), fmt.Sprintf("nodes-%d", iteration), "channel", fmt.Sprintf("marker-%d", iteration)}
		require.NoError(t, rdb.Set(t.Context(), keys[3], "3", 0).Err())
		start := make(chan struct{})
		results := make(chan error, 2)
		go func() {
			<-start
			results <- oldProtocolAdmission.Run(t.Context(), rdb, keys, strconv.FormatInt(time.Now().UnixNano(), 10)).Err()
		}()
		go func() {
			<-start
			results <- luaJoinProtocol.Run(t.Context(), rdb, keys, "new", 60000).Err()
		}()
		close(start)
		first, second := <-results, <-results
		require.NotEqual(t, first == nil, second == nil, "exactly one protocol can join: %v, %v", first, second)
		members, err := rdb.HGetAll(t.Context(), keys[0]).Result()
		require.NoError(t, err)
		require.Len(t, members, 1)
		marker, err := rdb.Get(t.Context(), keys[3]).Result()
		require.NoError(t, err)
		if members["old"] == "3" {
			require.Equal(t, "3", marker)
		} else {
			require.Equal(t, "4", members["new"])
			require.Equal(t, "4", marker)
		}
	}
}

func TestProtocolRejectedConstructorPreservesOldMaps(t *testing.T) {
	for _, marker := range []string{"3", "future", "04", "2"} {
		for _, clientOnly := range []bool{false, true} {
			t.Run(fmt.Sprintf("marker=%s/client=%t", marker, clientOnly), func(t *testing.T) {
				srv := redistest.Start(t, redistest.DBPool)
				rdb := srv.Client
				expiry := rdb.PTTL
				if srv.Real() {
					expiry = rdb.PExpireTime
				}
				const pool = "admission-before-migration"
				ctx := t.Context()
				require.NoError(t, rdb.Set(ctx, pool+":protocol-version", marker, 0).Err())
				require.NoError(t, rdb.HSet(ctx, pool+":protocol", "old", "3").Err())
				heartbeat := "1"
				if marker == "3" {
					heartbeat = strconv.FormatInt(time.Now().UnixNano(), 10)
				}
				hashes := map[string]map[string]string{
					rmapContentKey(nodeKeepAliveMapName(pool)): {"old": heartbeat, "=rev": "82", "=kind": "set"},
					rmapContentKey(nodeShutdownMapName(pool)):  {"=rev": "41", "=kind": "reset"},
					rmapContentKey(jobMapName(pool)):           {"worker": "original-job", "=rev": "25", "=kind": "set"},
				}
				expiries := map[string]int64{}
				for key, values := range hashes {
					require.NoError(t, rdb.HSet(ctx, key, values).Err())
					require.NoError(t, rdb.PExpire(ctx, key, time.Hour).Err())
					expires, err := expiry(ctx, key).Result()
					require.NoError(t, err)
					expiries[key] = expires.Milliseconds()
				}
				opts := []NodeOption{WithWorkerTTL(time.Hour)}
				if clientOnly {
					opts = append(opts, WithClientOnly())
				}
				node, err := AddNode(ctx, pool, rdb, opts...)
				require.Error(t, err)
				require.Nil(t, node)
				for key, values := range hashes {
					actual, err := rdb.HGetAll(ctx, key).Result()
					require.NoError(t, err)
					require.Equal(t, values, actual, "rejected constructor migrated %s", key)
					expires, err := expiry(ctx, key).Result()
					require.NoError(t, err)
					require.Equal(t, expiries[key], expires.Milliseconds(), "rejection changed expiry")
				}
				version, err := rdb.Get(ctx, pool+":protocol-version").Result()
				require.NoError(t, err)
				require.Equal(t, marker, version)
				members, err := rdb.HGetAll(ctx, pool+":protocol").Result()
				require.NoError(t, err)
				require.Equal(t, map[string]string{"old": "3"}, members)
			})
		}
	}
}
