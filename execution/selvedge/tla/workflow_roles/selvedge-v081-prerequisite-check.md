# Selvedge v0.8.1 prerequisite check

Checked 2026-11-05 (America/Los_Angeles). All five native prerequisites for
[#1024](design.md) are available in this release.
This records dependency source inspection and isolated consumer evidence;
framework implementation acceptance remains in the execution plan.

## Release identity

- [v0.8.1 release](https://github.com/CaliLuke/selvedge/releases/tag/v0.8.1),
  published `2026-11-06T07:09:56Z` (November 5 locally).
- Commit: `5adb66608169ad8f28fa88281b514ca36e501cee`.
- Module sum: `h1:LNcpZUDNAhSIPuJMxJMvR1rRT6beE/dyK2IqxA80Cdg=`.
- Go 1.27.3. Fresh consumer module, released dependency, no local replacement,
  dependency edits or dependency contributor suites.
- [Upstream release confirmation](https://github.com/CaliLuke/selvedge/issues/2#issuecomment-6679348111).

## Contract and binding

The producer APIs and migration/observability contracts mapped in the
[v0.8.0 check](selvedge-v080-prerequisite-check.md) remain available.
Both public `Shutdown` methods now use the shared wait helper in
[shutdown.go](https://github.com/CaliLuke/selvedge/blob/v0.8.1/shutdown.go).
`ErrShutdownInterrupted` marks only a wait that did not observe the cleanup
result; it also wraps the context cancellation or deadline cause. Completed
cleanup results are returned unchanged, even if an error wraps a context error.
Completion wins over an already-cancelled caller and is rechecked after an
interruption. The Worker timeout gap is resolved.

The Loom cleanup owner must drain role operations, then call native
`Shutdown(context.Background())` until the returned error no longer matches
`sv.ErrShutdownInterrupted`. Only that sentinel permits retry. Nil or any
unmarked error is the final result, including a context-wrapping cleanup error.
Caller cancellation only interrupts the caller's Loom wait; it does not cancel
this cleanup owner. Keep the existing no-restart rule and private native handle.
No error-string matching, enlarged timeout or new persistence owner is needed.

This implements the model's `completionWait` abstraction: interrupted native
observations do not publish a cleanup result or change the role to stopped.
They are stuttering observations until the final result. Native cleanup still
must eventually finish for liveness. No model transition changed, so the
reviewed model runs remain valid; the Go adapter correspondence remains an
implementation test obligation.

## Consumer evidence and limits

`go test -race -v -count=1 .` in the isolated module passed both tests below:

```text
TestWorkerCompletion: first interrupted=true deadline=true
worker: completed; cancelled repeat returned nil
TestProducerCompletion: init cancellation safe; exact bytes;
timeout/cancellation classified; drained; cancelled repeat returned nil
PASS
```

The Producer probe holds an admitted JSON encoder to force shutdown waiting,
then releases it and observes completion. The Worker probe uses a one-nanosecond
timeout to expose the previously ambiguous branch. Final storage errors that
wrap context errors are covered by the released API contract and source
inspection, not injected in this consumer probe. PostgreSQL, separate processes,
crash/restart and the Loom adapter remain implementation acceptance obligations.
Root dependency pins were not changed during this readiness check; adopt the
verified release in #1024's atomic implementation commit.

## Reproduction

In a fresh directory, save the following as `release_test.go`, then run:

```sh
go mod init example.com/selvedge-release-check
go get github.com/CaliLuke/selvedge@v0.8.1
go test -race -v -count=1 .
```

```go
package releasecheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	sv "github.com/CaliLuke/selvedge"
	"path/filepath"
	"testing"
	"time"
)

type blockingInput struct {
	entered chan struct{}
	release chan struct{}
}

func (v blockingInput) MarshalJSON() ([]byte, error) {
	close(v.entered)
	<-v.release
	return []byte(`"held"`), nil
}
func finish(t *testing.T, shutdown func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		err := shutdown(ctx)
		if !errors.Is(err, sv.ErrShutdownInterrupted) {
			if err != nil {
				t.Fatal(err)
			}
			return
		}
		if ctx.Err() != nil {
			t.Fatal("cleanup did not complete", err)
		}
	}
}
func TestWorkerCompletion(t *testing.T) {
	a := sv.NewApp(sv.WithDatabase(filepath.Join(t.TempDir(), "worker.db")), sv.WithShutdownTimeout(time.Nanosecond))
	if err := a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := a.Shutdown(context.Background())
	if first != nil && (!errors.Is(first, sv.ErrShutdownInterrupted) || !errors.Is(first, context.DeadlineExceeded)) {
		t.Fatal(first)
	}
	t.Logf("first: interrupted=%t deadline=%t", errors.Is(first, sv.ErrShutdownInterrupted), errors.Is(first, context.DeadlineExceeded))
	finish(t, a.Shutdown)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Shutdown(ctx); err != nil {
		t.Fatal("completed result lost to cancelled waiter", err)
	}
	t.Log("worker: completed; cancelled repeat returned nil")
}
func TestProducerCompletion(t *testing.T) {
	init, cancelInit := context.WithCancel(context.Background())
	p, err := sv.OpenProducer(init, sv.WithDatabase(filepath.Join(t.TempDir(), "producer.db")), sv.WithShutdownTimeout(time.Millisecond))
	cancelInit()
	if err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(`{ "n": 1.00, "s": "<x>" }`)
	id, err := p.StartWorkflowEncoded(context.Background(), "unregistered", raw, sv.WithInstanceID("stable"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := p.Describe(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != sv.StatusRunning || !bytes.Equal(d.InputData, raw) || d.ParentInstanceID != "" || d.ParentKey != "" {
		t.Fatalf("unexpected description: %+v", d)
	}
	held := blockingInput{make(chan struct{}), make(chan struct{})}
	admission := make(chan error, 1)
	go func() {
		_, err := p.StartWorkflowByName(context.Background(), "held", held, sv.WithInstanceID("held"))
		admission <- err
	}()
	<-held.entered
	first := p.Shutdown(context.Background())
	if !errors.Is(first, sv.ErrShutdownInterrupted) || !errors.Is(first, context.DeadlineExceeded) {
		t.Fatal("expected classified timeout", first)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	next := p.Shutdown(ctx)
	if !errors.Is(next, sv.ErrShutdownInterrupted) || !errors.Is(next, context.Canceled) {
		t.Fatal("expected classified cancellation", next)
	}
	close(held.release)
	if err := <-admission; err != nil {
		t.Fatal(err)
	}
	finish(t, p.Shutdown)
	if err := p.Shutdown(ctx); err != nil {
		t.Fatal("completed result lost to cancelled waiter", err)
	}
	t.Log("producer: init cancellation safe; exact bytes; timeout/cancellation classified; drained; cancelled repeat returned nil")
}
```
