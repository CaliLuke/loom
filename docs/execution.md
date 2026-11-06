# Typed activities and durable workflows

`github.com/CaliLuke/loom/execution` provides a small typed activity contract for
workflow adapters. Use it when applications share typed activity definitions
and calls across execution backends. It is an ordinary Go runtime
package; it requires no DSL declarations or generated code.

An activity couples a stable name with its input and output types:

```go
readMetadata := execution.Activity[MetadataInput, MetadataOutput]{
    Name: "assets.read_metadata",
}
```

Register its handler through `readMetadata.Register(ctx, registrar, options,
handler)`. Inside a workflow, `readMetadata.Execute(ctx, executor, input, options)`
returns `MetadataOutput`; `ExecuteAsync` returns `Future[MetadataOutput]`.
Check every returned error. Inputs must be non-nil, and input and output types
must be supported by the selected backend's codec. Names must remain stable
across worker restarts and replay, and names are unique within a registry.

`ActivityOptions` contains queue, retry, attempt, queue-wait, and heartbeat
settings. Zero fields leave values to registration or backend defaults; they do
not explicitly clear inherited settings. This contract does not make different
backends' retry or cancellation behavior identical.

## Selvedge

`github.com/CaliLuke/loom/execution/selvedge` adapts these activities to Selvedge
v0.8.1. Create an activity registry with `selvedge.New(app)` and register all
handlers before `app.Start`. The app remains caller-owned: the adapter never
starts or stops it. Register through the typed `selvedge.Register` function so
Selvedge retains the concrete input and output types and owns their codecs.
This registry does not implement the type-erased `ActivityRegistrar` interface.

Inside a native Selvedge workflow, pass `activities.Scope(workflowContext)` as
the executor. Scope controls cancellation and detached execution. The standard
`context.Context` parameter does not replace that native workflow scope, and a
future always waits using its original scope. Activity JSON inputs and outputs
remain plain values without an extra envelope.

The adapter's typed `selvedge.Register(activities, activity, handler,
nativeOptions)` function accepts the full native `selvedge.ActivityOptions`,
including maximum retry intervals and non-retryable error predicates. That path
preserves native defaults and error types. Agent-specific error conversion and
timeout categories belong in the consuming adapter.

The caller-owned path keeps workflows, start options, status inspection,
cancellation requests and transaction-bound submission on the native Selvedge API. In particular, passing
the existing transaction context to native `StartWorkflow` keeps admission in
that transaction. The activity adapter neither starts another transaction nor
turns submission into a separate background call.

Selvedge v0.8.1 requires Go 1.27.3. It uses the v0.7.0 database schema;
there is no schema migration for this upgrade. Earlier development schemas
have no supported migration path.

## Adapter authors

Implement `ActivityExecutor` and either a typed native registration function or
`ActivityRegistrar`, according to the backend API. Generic registration receives a
concrete handler and execution receives the concrete input plus a result pointer.
Preserve those types when the native serializer needs them. The
`execution/adapter` package provides the shared handler validation and binding
used by adapters; it is not an application scheduling API.

There is no service-method submission DSL, scheduler, persistence engine or
agent lifecycle here. Applications retain native capabilities outside the
owned workflow roles described below.


## Durable workflow roles

Share an `execution.Workflow[Input, Output]{Name: "assets.inspect"}` value in a
contract package imported by the submitting service and the worker. Names are
stable, database-wide identities, limited to 1–255 bytes. Payload compatibility
across deployments remains the application's responsibility.

Use `selvedge.NewProducer(sv.WithDatabase(database))` in a submitting process.
Call `selvedge.Declare(producer, contract)` before `producer.Start(ctx)`.
A Producer opens storage and can submit and inspect jobs, but has no workflow
or activity registration API and starts no execution, election or maintenance
loops. Its native handle stays private.

Use `selvedge.NewWorker(sv.WithDatabase(database))` in a worker process.
Register workflows with `selvedge.RegisterWorkflow(worker, contract, handler)`;
handlers have signature `func(*sv.WorkflowContext, Input) (Output, error)`.
Register activities through `selvedge.Register(worker.Activities(), activity,
handler, nativeOptions)`. Inside workflows, execute activities through
`worker.Activities().Scope(ctx)`. Worker registration also declares the typed
contract. A Worker can submit and inspect jobs, so it supports single-process
applications as well. Check every constructor and registration error.

Declare and register everything before Start. Repeating the same declaration
is allowed; changing its input/output types under the same name is rejected.
Duplicate workflow registration is rejected. Once Start begins, registration
is sealed, including through an Activities pointer obtained earlier.

### Lifecycle

`Start(ctx)` bounds initialization only. Cancelling that context after a
successful return does not stop the role. `Ready()` reports lifecycle state,
not database health. Submit and Lookup require a running role and an exactly
matching declared contract.

`Shutdown(ctx)` rejects new operations, drains calls already admitted and waits
for native storage cleanup. Concurrent callers observe the same eventual
result. A caller whose context expires gets an error matching
`selvedge.ErrShutdownIncomplete`; cleanup continues. Call Shutdown again to
observe its final result. Completed cleanup takes precedence over a cancelled
waiter. A role cannot restart, including after failed startup or shutdown before
startup. Existing caller-owned `selvedge.New(app)` activity registries retain
their original lifecycle contract.

### Submission and retry outcomes

`selvedge.Submit(ctx, role, contract, jobID, input)` requires a nonempty,
caller-chosen ID. It returns `Admission{ID, Created}`. The input uses Selvedge's
native JSON codec: the application marshaler runs once per call, and normalized,
valid UTF-8 bytes identify the job. The same bytes are used for insertion and
reconciliation. Equality is byte equality, not Go equality or semantic JSON
equality. Invalid or unstable encoding fails before insertion.

| Result | Meaning |
|---|---|
| `Created: true` | This call inserted the job. |
| `Created: false`, nil error | The ID already holds a root job with the same workflow name and input bytes; it may already be completed. |
| `*AdmissionConflictError` | The ID holds a different name, input or parent identity. `ExistingInput` allows comparison with earlier uncertain attempts. |
| `*AdmissionUncertainError` | This call may have committed `Input`. Retain those bytes and retry the same ID. |
| Native `ErrWorkflowStartRejected`, `ErrNotRunning`, or `ErrInputEncoding` | This call inserted nothing. An earlier uncertain attempt may still have committed. |

Every outcome describes one call. After uncertainty, keep the `Input` bytes
from all uncertain attempts. A later conflict can identify a job matching an
earlier attempt when its workflow name matches, `ChildInstance` is false and
`ExistingInput` equals one of those saved values. That establishes admission of
that identity, but does not identify which caller inserted it. A later rejection
alone never proves that earlier uncertain attempts were absent. Stopping retries
with unresolved uncertainty leaves an unknown outcome; using a new ID can create
a second job. The adapter does not automatically retry submissions.

### Typed lookup

`selvedge.Lookup(ctx, role, contract, jobID)` returns `Job[Output]`. It checks the
stored workflow name before invoking the native typed result decoder. A missing
ID returns the native `WorkflowNotFoundError`; a decode failure is a lookup
error rather than `Job.Err`.

`Running` covers admitted jobs that have not been claimed and jobs being
executed. Output is set only for `Completed`. Failed or cancelled jobs carry
the recorded failure in `Job.Err`. Timestamps come from the native identity
snapshot; the later result read determines the returned state. Readiness and
Running status do not establish that any worker has registered this name.

### Ownership and deployment

Selvedge owns persistence, dispatch, leases, retries, codecs and migrations.
The application owns handler logic, IDs, migration deployment and payload
compatibility. Automatic migration is the native default; with
`sv.WithAutoMigrate(false)`, initialization checks connectivity/configuration
but does not validate schema compatibility. Apply migrations before using that
configuration. SQLite workers in separate processes rely on polling;
PostgreSQL also supports LISTEN/NOTIFY. No cross-process wake latency is promised.

Worker options include native `sv.WithHooks` and telemetry hooks. Producers
never invoke workflow hooks, including OnWorkflowStart; instrument their client
operations with application metrics/tracing wrappers. Worker admission hooks
receive the frozen raw input, not the original typed Go value.

The roles expose no storage, transactions, cancellation, signals, queries or
child-workflow API. Passing a transaction context from another App does not
make role submission atomic with application writes. Use the caller-owned
native path when those facilities are required.

The [durable metadata example](../execution/selvedge/examples/durable/README.md)
contains separate producer and worker binaries sharing an ordinary Go contract.
