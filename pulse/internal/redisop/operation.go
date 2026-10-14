// Package redisop bounds Redis operations owned by Pulse runtime lifecycles.
package redisop

import (
	"context"
	"time"

	"github.com/CaliLuke/go-redis/v9"
)

type deadlineHook struct {
	duration time.Duration
}

// Budget is the existing streaming poll and map cleanup budget. Native blocking
// commands additionally use the driver's BLOCK + 10 seconds socket-read margin.
const Budget = 5 * time.Second

// Client borrows the caller's pools and current hook snapshot for ordinary
// operations with finite socket and context budgets. Never Close the returned
// facade. Native blocking reads must use Read to reserve response headroom.
func Client(caller *redis.Client) *redis.Client {
	return borrow(caller, Budget)
}

// Read plans a native blocking read and returns its bounded BLOCK duration with
// a borrowed client. The operation context allows the poll plus the ordinary I/O
// budget for its response, reduced by stricter positive caller read/write limits.
// The native driver read deadline remains BLOCK + 10 seconds when context
// deadlines are disabled. Never Close the borrowed client.
func Read(caller *redis.Client, configured time.Duration) (*redis.Client, time.Duration) {
	block := blockDuration(configured)
	budget := Budget
	if block > 0 {
		budget += block
	}
	return borrow(caller, budget), block
}

// Context bounds whole operations such as PubSub initialization and membership
// CAS retry loops whose total lifetime is not bounded by individual commands. Its caller must invoke the returned cancellation function.
func Context(ctx context.Context, caller *redis.Client) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout(caller, Budget))
}

// blockDuration bounds a native blocking poll without changing negative nonblocking
// requests. Zero still represents continuous listening through successive polls.
func blockDuration(configured time.Duration) time.Duration {
	if configured == 0 || configured > Budget {
		return Budget
	}
	return configured
}

func borrow(caller *redis.Client, budget time.Duration) *redis.Client {
	borrowed := caller.WithTimeout(timeout(caller, Budget))
	borrowed.AddHook(deadlineHook{duration: timeout(caller, budget)})
	return borrowed
}

func timeout(caller *redis.Client, duration time.Duration) time.Duration {
	opts := caller.Options()
	for _, configured := range []time.Duration{opts.ReadTimeout, opts.WriteTimeout} {
		if configured > 0 && configured < duration {
			duration = configured
		}
	}
	return duration
}

func (h deadlineHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h deadlineHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		bounded, cancel := context.WithTimeout(ctx, h.duration)
		defer cancel()
		return next(bounded, cmd)
	}
}

func (h deadlineHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, commands []redis.Cmder) error {
		bounded, cancel := context.WithTimeout(ctx, h.duration)
		defer cancel()
		return next(bounded, commands)
	}
}
