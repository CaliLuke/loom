# Typed activities

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
v0.7.0. Create an activity registry with `selvedge.New(app)` and register all
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

Keep workflows, start options, status inspection, cancellation requests, and
transaction-bound submission on the native Selvedge API. In particular, passing
the existing transaction context to native `StartWorkflow` keeps admission in
that transaction. The activity adapter neither starts another transaction nor
turns submission into a separate background call.

Selvedge v0.7.0 requires Go 1.27.3 and a fresh database. Finish runs on the old
binary, retain its database for inspection, and configure a new database before
starting the upgraded workers. There is no migration from earlier development
schemas.

## Adapter authors

Implement `ActivityExecutor` and either a typed native registration function or
`ActivityRegistrar`, according to the backend API. Generic registration receives a
concrete handler and execution receives the concrete input plus a result pointer.
Preserve those types when the native serializer needs them. The
`execution/adapter` package provides the shared handler validation and binding
used by adapters; it is not an application scheduling API.

This package intentionally has no general workflow definition, service-method
submission DSL, scheduler, persistence engine, or agent lifecycle. Applications
can use the common activity contract while retaining the native capabilities
their workflows need.
