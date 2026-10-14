package rmap

import (
	"context"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type observationReplyGate struct {
	m *Map
}

func TestReplicaAlreadyObservedWriteSkipsSnapshot(t *testing.T) {
	rdb := startTestRedis(t)
	m := joinMap(t, rdb, "observation-cancel")
	require.NoError(t, m.sub.Unsubscribe(t.Context(), m.chankey))
	gate := &observationReadGate{entered: make(chan struct{}, 1)}
	rdb.AddHook(gate)
	rdb.AddHook(&observationReplyGate{m: m})
	ctx, cancel := context.WithTimeout(context.WithValue(t.Context(), observationContextKey{}, true), 4*time.Second)
	defer cancel()
	_, err := m.SetAndWait(ctx, "committed", "value")
	require.NoError(t, err, "legitimate early replica observation is successful")
	select {
	case <-gate.entered:
		t.Fatal("already-observed write must not require another snapshot")
	default:
	}
}

func (h *observationReplyGate) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *observationReplyGate) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *observationReplyGate) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		err := next(ctx, cmd)
		if err != nil || ctx.Value(observationContextKey{}) != true || (cmd.Name() != "evalsha" && cmd.Name() != "eval") {
			return err
		}
		// Redis has committed. Hold only its caller's reply while the actual replica
		// owner independently reconciles the same committed revision.
		poll := time.NewTicker(time.Millisecond)
		defer poll.Stop()
		for {
			if value, ok := h.m.Get("committed"); ok && value == "value" {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-poll.C:
			}
		}
	}
}
