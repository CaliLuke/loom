# Draft issue: WithTimeout drops inherited command hooks

Tested with go-redis v9.23.0. The same omission is present in master
`552379bdb4293f93c1e2c5d51698e540c1efb086`.

Calling `WithTimeout` on a client with registered hooks produces a client that
no longer invokes those inherited `ProcessHook` and `ProcessPipelineHook`
callbacks. This affects ordinary commands, pipelines and transaction pipelines.
Hooks added directly to the derived client still run. The original client's
hooks continue to run.

Expected: changing the command timeout should preserve the installed command
hook chain. Losing it silently removes tracing, metrics, auditing or command
interception from the derived client.

The attached regression is server-free: a custom dialer returns a sentinel error,
while hooks record dispatch. With the released code, the inherited hook observes
only the original client's one command and two pipeline calls; it should also
observe the derived client's calls. It additionally checks that later parent and
derived-client hook additions remain isolated and parent timeouts are unchanged.

`baseClient.clone` copies pools/options and other shared state but omits
`hooksMixin`. Copying it with the existing `hooksMixin.clone()` helper preserves
the snapshot and gives the derived client independent hook state. `Client.init`
then rebuilds that chain around the derived client's command processors, as the
connection constructor already does. This does not create new pools or alter
pool ownership. It does not change the separate shared-pool DialHook behavior.

Reproduce in an empty directory with this `hooks_test.go`:

```sh
go mod init example.com/timeoutclone
go get github.com/redis/go-redis/v9@v9.23.0
go test -race -run '^TestWithTimeoutPreservesHooks$' -count=1 .
```

```go
package timeoutclone_test

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type countingHook struct {
	process  atomic.Int32
	pipeline atomic.Int32
}

func TestWithTimeoutPreservesHooks(t *testing.T) {
	reachedDial := errors.New("dial reached")
	parent := redis.NewClient(&redis.Options{
		Addr:       "unused:1",
		MaxRetries: -1,
		Dialer: func(context.Context, string, string) (net.Conn, error) {
			return nil, reachedDial
		},
	})
	t.Cleanup(func() {
		if err := parent.Close(); err != nil {
			t.Error(err)
		}
	})
	early := new(countingHook)
	parent.AddHook(early)
	clone := parent.WithTimeout(100 * time.Millisecond)
	late := new(countingHook)
	parent.AddHook(late)
	cloneOnly := new(countingHook)
	clone.AddHook(cloneOnly)
	for _, tc := range []struct {
		name   string
		client *redis.Client
	}{
		{"parent", parent}, {"clone", clone},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.client.Ping(t.Context()).Err(); !errors.Is(err, reachedDial) {
				t.Errorf("PING error = %v, want dial sentinel", err)
			}
			_, err := tc.client.Pipelined(t.Context(), func(pipe redis.Pipeliner) error {
				pipe.Ping(t.Context())
				return nil
			})
			if !errors.Is(err, reachedDial) {
				t.Errorf("pipeline error = %v, want dial sentinel", err)
			}
			_, err = tc.client.TxPipelined(t.Context(), func(pipe redis.Pipeliner) error {
				pipe.Ping(t.Context())
				return nil
			})
			if !errors.Is(err, reachedDial) {
				t.Errorf("transaction pipeline error = %v, want dial sentinel", err)
			}
		})
	}
	for _, tc := range []struct {
		name              string
		hook              *countingHook
		process, pipeline int32
	}{
		{"inherited", early, 2, 4}, {"parent-only", late, 1, 2}, {"clone-only", cloneOnly, 1, 2},
	} {
		if got := tc.hook.process.Load(); got != tc.process {
			t.Errorf("%s ProcessHook calls = %d, want %d", tc.name, got, tc.process)
		}
		if got := tc.hook.pipeline.Load(); got != tc.pipeline {
			t.Errorf("%s ProcessPipelineHook calls = %d, want %d", tc.name, got, tc.pipeline)
		}
	}
	if parent.Options().ReadTimeout == 100*time.Millisecond || parent.Options().WriteTimeout == 100*time.Millisecond {
		t.Error("clone changed parent timeouts")
	}
}

func (h *countingHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}
func (h *countingHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		h.process.Add(1)
		return next(ctx, cmd)
	}
}
func (h *countingHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		h.pipeline.Add(1)
		return next(ctx, cmds)
	}
}
```

# Draft pull request: Preserve inherited command hooks in WithTimeout clients

`WithTimeout` currently drops installed command hooks because `baseClient.clone`
omits `hooksMixin`. Copy the hook snapshot with its existing clone helper before
initializing the derived client's processors. Later hook additions remain isolated.

Add a server-free regression covering ordinary commands, pipelines, transaction
pipelines, inherited and independently added hooks, and unchanged parent timeout
options. The test fails on v9.23.0 and passes with the change under `-race`.

Validation: `go test -race -run '^TestWithTimeoutPreservesHooks$' -count=1 .`
