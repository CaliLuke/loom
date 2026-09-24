package redistest

import (
	"context"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

type (
	// Recorder is a redis.Hook that records the names of the commands a
	// client sends, including the commands of pipelines and transactions.
	// Tests use it to assert that an operation reaches Redis as one command.
	Recorder struct {
		mu    sync.Mutex
		names []string
	}
)

// Record adds a new Recorder to rdb and returns it.
func Record(rdb *redis.Client) *Recorder {
	r := &Recorder{}
	rdb.AddHook(r)
	return r
}

// Reset forgets the recorded commands.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = nil
}

// Names returns the lower-case names of the commands recorded since the
// last Reset, in the order the client sent them.
func (r *Recorder) Names() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.names...)
}

// DialHook implements redis.Hook.
func (r *Recorder) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

// ProcessHook implements redis.Hook.
func (r *Recorder) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		r.add(cmd)
		return next(ctx, cmd)
	}
}

// ProcessPipelineHook implements redis.Hook.
func (r *Recorder) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		for _, cmd := range cmds {
			r.add(cmd)
		}
		return next(ctx, cmds)
	}
}

// add records the name of cmd.
func (r *Recorder) add(cmd redis.Cmder) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = append(r.names, strings.ToLower(cmd.Name()))
}
