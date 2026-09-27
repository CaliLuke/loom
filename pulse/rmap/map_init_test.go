package rmap

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type (
	initialContentFailure struct {
		before func()
		err    error
	}
)

func TestInitRetainsCleanupErrors(t *testing.T) {
	for _, closed := range []bool{false, true} {
		name := "successful cleanup"
		if closed {
			name = "failed cleanup"
		}
		t.Run(name, func(t *testing.T) {
			rdb := startTestRedis(t)
			sm := &Map{
				Name: "init-failure", chankey: "map:init-failure:updates",
				hashkey: "map:init-failure:content", rdb: rdb,
				appendScript: luaAppend, appendUniqueScript: luaAppendUnique,
				delScript: luaDelete, incrScript: luaIncr, removeScript: luaRemove,
				resetScript: luaReset, destroyScript: luaDestroy, setScript: luaSet,
				testAndDelScript: luaTestAndDel, testAndResetScript: luaTestAndReset,
				testAndSetScript: luaTestAndSet, setIfNotExistsScript: luaSetIfNotExists,
			}
			readErr := errors.New("initial content unavailable")
			rdb.AddHook(initialContentFailure{
				err: readErr,
				before: func() {
					if closed {
						require.NoError(t, sm.sub.Close())
					}
				},
			})

			err := sm.init(t.Context())
			require.ErrorIs(t, err, readErr)
			require.ErrorContains(t, err, "failed to read initial content")
			if closed {
				require.ErrorIs(t, err, redis.ErrClosed)
				require.ErrorContains(t, err, "failed to unsubscribe")
				require.ErrorContains(t, err, "failed to close subscription")
			} else {
				require.NotErrorIs(t, err, redis.ErrClosed)
			}
			require.ErrorIs(t, sm.sub.Close(), redis.ErrClosed, "init must close its subscription")
		})
	}
}

func (h initialContentFailure) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h initialContentFailure) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "hgetall" {
			h.before()
			cmd.SetErr(h.err)
			return h.err
		}
		return next(ctx, cmd)
	}
}

func (h initialContentFailure) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
