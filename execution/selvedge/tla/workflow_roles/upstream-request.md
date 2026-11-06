# Add a supported producer-only lifecycle for workflow submission and lookup

Selvedge v0.7.0 supports by-name workflow submission and status/result lookup,
but opening an `App` also starts execution services. A process that only admits
jobs should have a supported way to initialize storage and use those client
operations without participating in workflow execution, activity execution,
leader election or leader maintenance.

This is a feature request, not a claim that `App.Start` violates its documented
contract. Separating these roles would let API servers and command-line clients
submit work while dedicated worker deployments own execution capacity.

## Current behavior and reproduction

The latest release checked was [v0.7.0](https://github.com/CaliLuke/selvedge/releases/tag/v0.7.0).
[`App.Start`](https://github.com/CaliLuke/selvedge/blob/v0.7.0/app.go#L112)
initializes storage and unconditionally calls
[`startBackgroundTasks`](https://github.com/CaliLuke/selvedge/blob/v0.7.0/app.go#L390),
which starts leader election, workflow resumption and activity workers.
[`StartWorkflowByName`](https://github.com/CaliLuke/selvedge/blob/v0.7.0/client.go#L19)
does not require local workflow registration, but still requires an initialized
App. Empty registrations or restrictive execution filters do not provide a
supported guarantee that the process is a client only.

Run the following as `main.go` in a fresh module:

```sh
go mod init example.com/producerprobe
go get github.com/CaliLuke/selvedge@v0.7.0
go run .
```

The workflow handler is registered solely as a visible execution probe. This
shows normal `App.Start` behavior; there is currently no producer-only startup
option to test instead.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sv "github.com/CaliLuke/selvedge"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	dir, err := os.MkdirTemp("", "producer-example-")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	app := sv.NewApp(sv.WithDatabase(filepath.Join(dir, "jobs.db")))
	sv.RegisterWorkflow(app, sv.DefineWorkflow("echo", func(_ *sv.WorkflowContext, in string) (string, error) {
		return "executed locally: " + in, nil
	}))
	if err := app.Start(context.Background()); err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := app.Shutdown(ctx); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	id, err := sv.StartWorkflowByName(ctx, app, "echo", "hello", sv.WithInstanceID("stable-id"))
	if err != nil {
		return err
	}
	result, err := sv.WaitWorkflow[string](ctx, app, id)
	if err != nil {
		return err
	}
	fmt.Printf("%s: %s\n", result.Status, result.Output)
	return nil
}
```

Observed on Go 1.27.3 and SQLite: `completed: executed locally: hello`.
The requested producer handle would reject execution registration or make it
unavailable in its API. Submission and nonblocking lookup would remain usable,
with the job pending until a separate worker starts.

## Requested contract

- A supported client/producer initializer that opens storage and validates its
  configuration without starting execution, election or maintenance loops.
  Registering local workflow/activity implementations must be unavailable or
  rejected on this handle.
- Existing by-name submission with explicit stable IDs, status lookup and typed
  result lookup, with the same rejected/duplicate/uncertain admission semantics.
  A supported read should expose enough persisted identity to reconcile a
  duplicate: workflow name, root/child identity and stored encoded input. The
  existing raw JSON submission path should have a documented byte-preservation
  contract, or an explicit encoded-input entry point should provide it.
- Explicit migration ownership: permit automatic migration or externally managed
  migrations. Document what initialization validates when migration is disabled.
- A bounded initialization context, with the handle lifetime independently owned
  by shutdown. Cancelling an initialization timeout after success must not
  silently disable a long-lived producer.
- Idempotent concurrent shutdown that rejects new operations, drains admitted
  operations, and completes storage cleanup even when one caller stops waiting.
  Callers must be able to distinguish interrupted waiting from completed cleanup.
- Document supported observability hooks for client operations. Execution hooks
  must never fire in a producer-only process.

A separate producer type or an explicit App role could satisfy this contract.
A separate type makes forbidden execution registration harder to express; an
App role may reuse more existing client signatures but needs runtime admission
checks. Reuse storage, transaction, codec and notification machinery rather than
introducing a second persistence implementation. No remote service protocol is
needed.

## Acceptance examples

1. Start only a producer, submit a stable ID, and read a pending status without
   any workflow/activity execution or leader participation in that process.
2. Start a separate worker against the same database and observe completion from
   the producer using supported lookup APIs.
3. Repeat after a worker restart and after an uncertain submission response;
   retrying the same ID must not create another workflow instance.
4. Reject execution registration on a producer. Preserve existing App worker and
   single-process behavior.
5. Exercise initialization cancellation, concurrent shutdown, an interrupted
   shutdown waiter, and operations racing shutdown on SQLite and PostgreSQL.

Existing cancellation-only shutdown log noise is tracked separately in
[#1](https://github.com/CaliLuke/selvedge/issues/1); this request concerns role
separation and the lifecycle contract.
