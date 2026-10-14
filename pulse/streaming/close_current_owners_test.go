package streaming

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/CaliLuke/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/pulse/internal/redisop"
)

type membershipBudgetHook struct {
	key       string
	retry     bool
	attempts  int
	deadlines []time.Time
	sentinel  error
}

// TestSinkCloseJoinsRegistrationRecovery checks both newly owned recovery seams
// while the maintenance mutex is held. Cancellation precedes acquiring that lock;
// cleanup still joins the command rather than abandoning the worker.
func TestSinkCloseJoinsRegistrationRecovery(t *testing.T) {
	for _, command := range []string{"pttl", "createconsumer"} {
		t.Run(command, func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "recovery-close")
			sink := newTestSink(t, stream, "owner")
			gate := &shutdownCommandGate{
				match: func(cmd redis.Cmder) bool {
					if command == "pttl" {
						return cmd.Name() == "pttl" && cmd.Args()[1] == consumersMapKey(stream)
					}
					return cmd.Name() == "xgroup" && len(cmd.Args()) > 2 && cmd.Args()[1] == "createconsumer" && cmd.Args()[2] == stream.key
				},
				entered: make(chan struct{}), canceled: make(chan struct{}), release: make(chan struct{}),
			}
			rdb.AddHook(gate)
			var release sync.Once
			t.Cleanup(func() {
				release.Do(func() {
					close(gate.release)
				})
			})
			result := make(chan error, 1)
			sink.wait.Add(1)
			go func() {
				defer sink.wait.Done()
				sink.lock.Lock()
				defer sink.lock.Unlock()
				result <- sink.reconcileRegistrations(sink.runtimeCtx, true)
			}()
			select {
			case <-gate.entered:
			case <-time.After(time.Second):
				t.Fatal("recovery did not reach selected native command")
			}
			done := make(chan struct{})
			go func() {
				sink.Close(t.Context())
				close(done)
			}()
			select {
			case <-gate.canceled:
			case <-time.After(time.Second):
				t.Fatal("Close did not cancel recovery before acquiring its lock")
			}
			select {
			case <-done:
				t.Fatal("Close abandoned a live recovery operation")
			default:
			}
			release.Do(func() {
				close(gate.release)
			})
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("Close did not join released recovery")
			}
			require.ErrorIs(t, <-result, context.Canceled)
			require.NoError(t, rdb.Ping(t.Context()).Err())
		})
	}
}

// TestSinkMembershipRetryBudget bounds the logical CAS retry, including detached
// cleanup, rather than restarting its budget with every individual command.
func TestSinkMembershipRetryBudget(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(fmt.Sprintf("retry=%t", retry), func(t *testing.T) {
			rdb := startTestRedis(t)
			stream := newTestStream(t, rdb, "membership-budget")
			sink := newTestSink(t, stream, "owner")
			sentinel := errors.New("membership budget observed")
			hook := &membershipBudgetHook{key: consumersMapKey(stream), sentinel: sentinel, retry: retry}
			rdb.AddHook(hook)
			ctx := context.WithValue(context.WithoutCancel(t.Context()), hook, true)
			_, err := changeSinkMembership(ctx, sink.consumersMap[stream.Name], sink.Name, sink.filterIdentity, sinkMemberChange{remove: []string{"absent"}})
			require.ErrorIs(t, err, sentinel)
			want := 1
			if retry {
				want = 3
				require.Equal(t, want, hook.attempts, "two successful but conflicting CAS replies precede the sentinel")
			}
			require.Len(t, hook.deadlines, want)
			remaining := time.Until(hook.deadlines[0])
			require.Greater(t, remaining, time.Duration(0))
			require.LessOrEqual(t, remaining, redisop.Budget)
			for _, deadline := range hook.deadlines {
				require.Equal(t, hook.deadlines[0], deadline, "CAS retries retain the original deadline")
			}
		})
	}
}

func (h *membershipBudgetHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *membershipBudgetHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (h *membershipBudgetHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if ctx.Value(h) != true {
			return next(ctx, cmd)
		}
		if cmd.Name() == "pttl" && cmd.Args()[1] == h.key {
			deadline, _ := ctx.Deadline()
			h.deadlines = append(h.deadlines, deadline)
			if !h.retry {
				return h.sentinel
			}
		}
		if h.retry && (cmd.Name() == "evalsha" || cmd.Name() == "eval") && cmd.Args()[3] == h.key {
			h.attempts++
			if h.attempts == 3 {
				return h.sentinel
			}
			cmd.(*redis.Cmd).SetVal("competing revision")
			return nil
		}
		return next(ctx, cmd)
	}
}
