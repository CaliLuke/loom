# Selvedge v0.8.0 prerequisite check

Historical release check. The remaining gap recorded here is resolved in the
[v0.8.1 check](selvedge-v081-prerequisite-check.md).

Checked 2026-11-05 (America/Los_Angeles) for the
[#1024 design](design.md). This is dependency
reference and isolated consumer evidence, not framework implementation acceptance.

## Release identity

- [v0.8.0 release](https://github.com/CaliLuke/selvedge/releases/tag/v0.8.0),
  published `2026-11-06T05:39:01Z` (November 5 locally).
- Commit: `cd7b5ffe75d14139e50d3c104af734131abadd6e`.
- Module sum: `h1:tyg6gD7b5i16q/rZ5fbWRvQfF0usA1uK68m2W9ImlGw=`.
- Go 1.27.3. An isolated module required this released version; no local replace,
  dependency edits, dependency contributor suites or Loom dependency-pin change.
- [Selvedge #2](https://github.com/CaliLuke/selvedge/issues/2) is closed, but its
  closure does not establish every prerequisite below.

## Contract mapping

| Prerequisite | v0.8.0 evidence | Result |
|---|---|---|
| Producer initialization and migrations | `OpenProducer(ctx, ...Option)` in `producer.go:28–55`; initialization-only context, automatic migration option, connectivity/config validation but no schema compatibility check when disabled | Available |
| No execution/election/maintenance/listener loops | `Producer` GoDoc and `OpenProducer` use storage initialization without `App.Start` | Available by public contract and source inspection |
| No registration or execution hooks | Private App; no producer registration API; hooks replaced with `NoOpHooks` | Available; client metrics/traces wrap operations, `WithHooks` does not instrument producers |
| By-name submission, identity and typed lookup | `Producer.StartWorkflowByName`, `StartWorkflowEncoded`, `Describe`; `GetProducerWorkflowResult[O]`; `WorkflowDescription.InputData`, `ParentInstanceID`, `ParentKey` | Available |
| Cleanup-completion distinction for both roles | Producer documents repeated waits and wraps timeout with `context.DeadlineExceeded`; worker `App.Shutdown` still returns an untyped internal timeout (`app.go:187–195`) | **Worker prerequisite remains open** |

Source anchors: [producer.go](https://github.com/CaliLuke/selvedge/blob/v0.8.0/producer.go),
[app.go](https://github.com/CaliLuke/selvedge/blob/v0.8.0/app.go#L165-L230),
[describe.go](https://github.com/CaliLuke/selvedge/blob/v0.8.0/describe.go).

## Consumer probe

A fresh module pinned v0.8.0 and ran with SQLite temporary files. It opened a
producer, cancelled the successful initialization context, submitted an
unregistered workflow with ID `stable` using `StartWorkflowEncoded` and the
bytes `{ "n": 1.00, "s": "<x>" }`, then read `Describe` and shut down.
The observed result was:

```text
producer: status=running exactBytes=true root=true initCancellationSafe=true
```

`running` is the persisted admitted status while no worker executes the job;
it does not mean the producer ran workflow code. No separate worker, PostgreSQL,
crash/restart or framework adapter test was run in this release check. Those
remain implementation acceptance obligations.

The same consumer created an App with `WithShutdownTimeout(time.Nanosecond)`,
started it and called `Shutdown(context.Background())`. It observed:

```text
worker first shutdown: err=shutdown timed out after 1ns deadline=false canceled=false
worker eventual shutdown: <nil>
```

`deadline` and `canceled` are `errors.Is` checks. Subsequent bounded retries
observed eventual cleanup success. The tiny timeout exposes the branch; it is
not a claim about normal cleanup duration. Retrying every error is not a valid
adapter strategy because it conflates interrupted waiting with final cleanup
failure. Matching error strings or increasing the finite timeout is also not
an accepted binding. The standalone reproduction is attached to the upstream
[issue follow-up](https://github.com/CaliLuke/selvedge/issues/2#issuecomment-6678203688).

## Readiness consequence

The producer prerequisite is released. Keep #1024 conditional only on a
supported worker cleanup-completion distinction. Bind Producer options to
`sv.Option` and defer `sv.OpenProducer` until the Loom role's `Start`; keep the
native handle private. Producer hooks are wrappers around client operations;
worker hooks remain native `WithHooks`.

Keep the design's common UTF-8/fixed-point identity rule: the producer's new
encoded-input method preserves bytes, while the Worker still uses native
by-name JSON encoding. Do not silently broaden the accepted input domain for
one role. The existing lifecycle model's `v070Timer` counterexample also applies
to the unchanged v0.8.0 worker timer; its `completionWait` assumption is still
conditional for Worker. No model source changed and no rerun was needed.
