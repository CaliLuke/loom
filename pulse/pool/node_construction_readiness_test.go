package pool

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/rmap"
)

func TestConstructionNodeFailureReleasesSubscriptions(t *testing.T) {
	cases := []struct {
		name       string
		shutdown   bool
		clientOnly bool
		failMap    bool
	}{
		{name: "shutdown rejection", shutdown: true},
		{name: "later map join", failMap: true},
		{name: "node init after sink"},
		{name: "client node init", clientOnly: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rdb := startTestRedis(t)
			const pool = "construction-node"
			if tc.shutdown {
				m, err := rmap.Join(t.Context(), nodeShutdownMapName(pool), rdb)
				require.NoError(t, err)
				_, err = m.SetAndWait(t.Context(), "shutdown", "closed")
				require.NoError(t, err)
				m.Close()
			} else {
				rdb.AddHook(constructionFailureHook{match: func(cmd redis.Cmder) bool {
					args := cmd.Args()
					if len(args) < 2 {
						return false
					}
					key := fmt.Sprint(args[1])
					if tc.failMap {
						return cmd.Name() == "hgetall" && key == rmapContentKey(jobMapName(pool))
					}
					return cmd.Name() == "xadd" && strings.Contains(key, ":node:")
				}})
			}
			opts := []NodeOption{WithJobSinkBlockDuration(10 * time.Millisecond)}
			if tc.clientOnly {
				opts = append(opts, WithClientOnly())
			}
			node, err := AddNode(t.Context(), pool, rdb, opts...)
			require.Error(t, err)
			require.Nil(t, node)
			if tc.shutdown {
				require.ErrorContains(t, err, "is shutting down")
			} else {
				require.ErrorIs(t, err, errConstructionInjected)
			}
			channels := []string{
				rmapUpdateChannel(nodeShutdownMapName(pool)),
				rmapUpdateChannel(nodeKeepAliveMapName(pool)),
				rmapUpdateChannel(workerMapName(pool)),
				rmapUpdateChannel(jobMapName(pool)),
				rmapUpdateChannel(jobPayloadMapName(pool)),
				rmapUpdateChannel(workerKeepAliveMapName(pool)),
				rmapUpdateChannel(tickerMapName(pool)),
				rmapUpdateChannel(workerCleanupMapName(pool)),
				rmapUpdateChannel(jobPendingMapName(pool)),
				rmapUpdateChannel("stream:" + poolStreamName(pool) + ":sinks"),
				rmapUpdateChannel("sink:" + poolSinkName + ":keepalive"),
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			var last map[string]int64
			for {
				last, err = rdb.PubSubNumSub(ctx, channels...).Result()
				require.NoError(t, err)
				remaining := int64(0)
				for _, n := range last {
					remaining += n
				}
				if remaining == 0 {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatalf("failed construction still owns subscriptions: %v", last)
				case <-time.After(10 * time.Millisecond):
				}
			}
		})
	}
}

// Successful writes can outlive a failed constructor even when the caller
// cancels after Redis commits. Rollback must own the attempted write first.
func TestConstructionNodeLostWriteReply(t *testing.T) {
	for _, registration := range []bool{true, false} {
		for _, cancelCaller := range []bool{false, true} {
			t.Run(fmt.Sprintf("registration=%t/canceled=%t", registration, cancelCaller), func(t *testing.T) {
				rdb := startTestRedis(t)
				const pool = "construction-node-lost-reply"
				peer := addTestNode(t, rdb, pool, WithClientOnly(), WithWorkerTTL(time.Hour))
				require.NoError(t, luaJoinProtocol.Load(t.Context(), rdb).Err())
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				var nodeID, streamKey string
				rdb.AddHook(constructionFailureHook{
					match: func(cmd redis.Cmder) bool {
						args := cmd.Args()
						if len(args) < 2 {
							return false
						}
						if registration {
							return cmd.Name() == "evalsha" && fmt.Sprint(args[1]) == luaJoinProtocol.Hash()
						}
						return cmd.Name() == "xadd" && strings.Contains(fmt.Sprint(args[1]), ":node:")
					},
					after: true,
					committed: func(cmd redis.Cmder) {
						args := cmd.Args()
						if registration {
							nodeID = fmt.Sprint(args[len(args)-2])
						} else {
							streamKey = fmt.Sprint(args[1])
							_, nodeID, _ = strings.Cut(streamKey, ":node:")
						}
						if cancelCaller {
							cancel()
						}
					},
				})
				node, err := AddNode(ctx, pool, rdb, WithClientOnly(), WithWorkerShutdownTTL(time.Second))
				require.ErrorIs(t, err, errConstructionInjected)
				require.Nil(t, node)
				require.NotEmpty(t, nodeID, "the fault must occur after the write commits")
				registered, err := rdb.HExists(t.Context(), pool+":protocol", nodeID).Result()
				require.NoError(t, err)
				assert.False(t, registered, "failed node retained its protocol field")
				heartbeat, err := rdb.HExists(t.Context(), rmapContentKey(nodeKeepAliveMapName(pool)), nodeID).Result()
				require.NoError(t, err)
				assert.False(t, heartbeat, "failed node retained its initial heartbeat")
				if streamKey != "" {
					exists, err := rdb.Exists(t.Context(), streamKey).Result()
					require.NoError(t, err)
					assert.Zero(t, exists, "failed node retained its private stream")
				}
				version, err := rdb.Get(t.Context(), pool+":protocol-version").Result()
				require.NoError(t, err)
				assert.Equal(t, "3", version, "rollback must preserve the shared protocol marker")
				require.NoError(t, peer.Health(t.Context()))
				for _, key := range []string{pool + ":protocol", rmapContentKey(nodeKeepAliveMapName(pool))} {
					exists, err := rdb.HExists(t.Context(), key, peer.ID).Result()
					require.NoError(t, err)
					assert.True(t, exists, "rollback must preserve healthy peer registration")
				}
			})
		}
	}
}
