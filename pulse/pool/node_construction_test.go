package pool

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type constructionContextKey struct {
}

func TestNodeConstructionAbortOrderAndErrors(t *testing.T) {
	for _, bound := range []time.Duration{-1, 0, time.Second} {
		t.Run(bound.String(), func(t *testing.T) {
			original := errors.New("construction")
			firstCleanup := errors.New("first cleanup")
			lastCleanup := errors.New("last cleanup")
			ctx, cancel := context.WithCancel(context.WithValue(t.Context(), constructionContextKey{}, "retained"))
			cancel()
			owner := nodeConstruction{shutdownTTL: bound}
			var released []string
			for _, step := range []struct {
				name string
				err  error
			}{
				{name: "shutdown map", err: lastCleanup},
				{name: "heartbeat map"},
				{name: "heartbeat withdrawal"},
				{name: "protocol withdrawal"},
				{name: "worker map"},
				{name: "job map"},
				{name: "payload map"},
				{name: "worker heartbeat map"},
				{name: "ticker map"},
				{name: "cleanup map"},
				{name: "pending map"},
				{name: "sink"},
				{name: "private stream", err: firstCleanup},
				{name: "reader"},
				{name: "runtime cancellation"},
			} {
				owner.own(step.name, func(ctx context.Context) error {
					assert.NoError(t, ctx.Err())
					assert.Equal(t, "retained", ctx.Value(constructionContextKey{}))
					deadline, ok := ctx.Deadline()
					assert.True(t, ok)
					assert.True(t, time.Until(deadline) > 0)
					released = append(released, step.name)
					return step.err
				})
			}
			err := errors.Join(original, owner.abort(ctx))
			assert.ErrorIs(t, err, original)
			assert.ErrorIs(t, err, firstCleanup)
			assert.ErrorIs(t, err, lastCleanup)
			assert.Equal(t, []string{"runtime cancellation", "reader", "private stream", "sink",
				"pending map", "cleanup map", "ticker map", "worker heartbeat map", "payload map",
				"job map", "worker map", "protocol withdrawal", "heartbeat withdrawal", "heartbeat map",
				"shutdown map"}, released)
			require.NoError(t, owner.abort(ctx))
			assert.Len(t, released, 15, "second abort must not release anything twice")
		})
	}
}

func TestNodeConstructionTransfer(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	owner := nodeConstruction{shutdownTTL: time.Second}
	owner.own("runtime cancellation", func(context.Context) error {
		cancel()
		return nil
	})
	owner.transfer()
	require.NoError(t, owner.abort(t.Context()))
	assert.NoError(t, ctx.Err(), "constructor must not stop the transferred runtime")
}

func TestConstructionNodeCleanupFailuresReleaseLocalResources(t *testing.T) {
	rdb := startTestRedis(t)
	const pool = "construction-cleanup-errors"
	var state sync.Mutex
	var nodeID, streamKey string
	var attempts []string
	rdb.AddHook(constructionFailureHook{
		match: func(cmd redis.Cmder) bool {
			args := cmd.Args()
			return cmd.Name() == "xadd" && len(args) > 1 && strings.Contains(fmt.Sprint(args[1]), ":node:")
		},
		after: true,
		committed: func(cmd redis.Cmder) {
			state.Lock()
			defer state.Unlock()
			streamKey = fmt.Sprint(cmd.Args()[1])
			_, nodeID, _ = strings.Cut(streamKey, ":node:")
		},
	})
	rdb.AddHook(constructionFailureHook{match: func(cmd redis.Cmder) bool {
		state.Lock()
		defer state.Unlock()
		args := cmd.Args()
		if nodeID == "" || len(args) < 2 {
			return false
		}
		switch {
		case cmd.Name() == "del" && fmt.Sprint(args[1]) == streamKey:
			attempts = append(attempts, "stream")
		case cmd.Name() == "hdel" && len(args) > 2 && fmt.Sprint(args[2]) == nodeID:
			attempts = append(attempts, "protocol")
		case cmd.Name() == "evalsha" && len(args) > 5 && fmt.Sprint(args[3]) == rmapContentKey(nodeKeepAliveMapName(pool)) && fmt.Sprint(args[5]) == nodeID:
			attempts = append(attempts, "heartbeat")
		default:
			return false
		}
		return true
	}})
	node, err := AddNode(t.Context(), pool, rdb, WithClientOnly(), WithWorkerShutdownTTL(time.Second))
	state.Lock()
	observedID, observedStream := nodeID, streamKey
	observedAttempts := append([]string(nil), attempts...)
	state.Unlock()
	require.Nil(t, node)
	require.ErrorIs(t, err, errConstructionInjected)
	assert.ErrorContains(t, err, "failed to add init event")
	for _, operation := range []string{"withdraw node stream", "withdraw node protocol", "withdraw node heartbeat"} {
		assert.ErrorContains(t, err, operation)
	}
	assert.Equal(t, []string{"stream", "protocol", "heartbeat"}, observedAttempts)
	counts, err := rdb.PubSubNumSub(t.Context(), rmapUpdateChannel(nodeShutdownMapName(pool)), rmapUpdateChannel(nodeKeepAliveMapName(pool))).Result()
	require.NoError(t, err)
	for channel, count := range counts {
		assert.Zero(t, count, "local subscription %s survives failed remote cleanup", channel)
	}
	for _, key := range []string{pool + ":protocol", rmapContentKey(nodeKeepAliveMapName(pool))} {
		exists, err := rdb.HExists(t.Context(), key, observedID).Result()
		require.NoError(t, err)
		assert.True(t, exists, "the injection must actually leave uncertain state")
	}
	exists, err := rdb.Exists(t.Context(), observedStream).Result()
	require.NoError(t, err)
	assert.EqualValues(t, 1, exists)
}
