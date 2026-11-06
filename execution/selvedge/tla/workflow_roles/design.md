# Typed durable workflows with separate producer and worker roles (#1024)

Status: implemented contract for #1024; retained design and historical probes.
Revision 2 was independently reviewed, and its corrections are kept. Revision 3
adds bounded TLA+ evidence. Read-only Selvedge v0.7.0 source probes narrowed
the upstream dependency.

The design covers the full ticket scope: both roles, single-process deployment,
and the two-process acceptance test. **Implemented on Selvedge v0.8.1:**
[Selvedge v0.8.1 satisfies all native prerequisites](selvedge-v081-prerequisite-check.md),
including the producer-only API and distinguishable cleanup completion for
both roles. The v0.7.0 source inventory below remains historical evidence.

The ticket must not ship partially. In particular, it must not ship:

- a Worker-only release
- reconciliation alone
- a producer built on the v0.7.0 client App and its filter behavior
- a Loom-side shutdown timer workaround

Outside those native contracts, every lifecycle, admission and lookup property
is designed and model-checked on Loom's side.

## Summary

Add a typed workflow contract, `execution.Workflow[I, O]`, and two lifecycle
roles in `execution/selvedge`:

- A **`Producer`** admits jobs and reads their status. It never registers or
  executes workflow or activity code.
- A **`Worker`** registers the application's workflow and activity
  implementations and executes them. It can also admit and look up jobs, so a
  single-process deployment needs only a Worker.

Each role owns its native handle and has one gated lifecycle.

Admission requires a caller-chosen job ID. The input is encoded exactly once
per Submit call, which yields the bytes that identify the job. Loom checks
that those bytes are valid UTF-8 and that they reach the native fixed point.
The same bytes are inserted and used for reconciliation.

Selvedge stays the persistence engine. This design adds no DSL, no code
generation and no transport changes.

## Original problem and historical source inventory

An API handler often needs to admit a job durably, return its ID, and leave
execution to a separately deployed worker. Today applications build this
lifecycle themselves.

- **Loom covers only activities.** Loom provides typed activities only.
  Workflows, submission, status and app lifecycle stay native and caller-owned
  (`docs/execution.md:30-35`, `:49-53`). The documentation also excludes a
  "general workflow definition" from scope (`docs/execution.md:69-72`;
  `.agents/skills/loom-framework/SKILL.md:142-148`).
- **Selvedge v0.7.0 already exports submission and read APIs:**
  - By-name start with an explicit ID: `StartWorkflowByName` with
    `WithInstanceID` (`client.go:19-27`, `workflow.go:145-149`).
  - An instance read that returns the workflow name, parent ID and stored input
    bytes: `App.GetInstance` (`app.go:671`, fields in
    `internal/storage/models.go:19-39`).
  - Typed result reads: `GetWorkflowResult`/`WaitWorkflow`
    (`workflow.go:160-172`, `client.go:29-31`).
  - `Describe` (`describe.go:21-46`).

  The README also documents a "client App" that registers no workflows
  (`README.md:157-163`).
- **There is no supported producer-only lifecycle.**
  - Every `App.Start` runs leader election, workflow resumption and activity
    workers (`app.go:390-410`). The leader also runs timer and activity-timeout
    sweeps (`app.go:466-525`).
  - A client App runs no user code only because of filters. The README
    documents the storage-level rule that a non-nil empty `FindDispatchable`
    name list dispatches nothing (`README.md:554`; `internal/storage/lease.go:92-95`).
  - Activity task filters add `AND FALSE` for empty names
    (`internal/storage/activity_tasks.go:220-222`). That behavior is
    undocumented, and the README says "Every App runs activity workers"
    (`README.md:271`).
  - Neither behavior is an App-level producer guarantee.
- **Duplicate IDs are only partly reconciled.**
  - Retrying a committed ID returns `ErrWorkflowAlreadyStarted`
    (`errors.go:115-116`, `README.md:170-175`).
  - The insert uses `ON CONFLICT DO NOTHING`, and it compares neither the
    workflow name nor the input (`internal/storage/sqlite.go:248-266`,
    `postgres.go:168-184`).
  - Errors that are not `ErrWorkflowStartRejected` leave the outcome uncertain
    (`client.go:9-11`).

**Consumer evidence:**

- **Object-metadata application.** The issue's object-metadata application is
  real but is not in this workspace. Its code was not inspected.
- **loom-mcp agent engine.** This is a real sibling contract with the same
  split:
  - `NewWorker` owns an unstarted App.
  - `NewClient` uses a started, caller-owned App
    (`../loom-mcp/runtime/agent/engine/selvedge/engine.go:14-71`).

  Its client mode inherits the same engine-loop residue.

There are no synthetic Loom fixtures yet.

## Goals

- Typed workflow inputs and results. The application owns the workflow and
  activity bodies.
- Independent producer and worker initialization, readiness and shutdown.
  Producer processes never execute work.
- Stable job identity:
  - Admission is idempotent for the same ID, workflow name and frozen bytes.
  - Reusing an ID for a different workflow, input or parent is reported as a
    conflict.
  - After an uncertain outcome, the caller retries with the same ID.
- Typed status and result lookup, without engine-table queries.
- Single-process deployment through `Worker`. The caller-owned native path
  stays unchanged.
- A documented ownership split between Loom and Selvedge, including migrations
  and observability.
- A two-process example. Its test:
  1. Admits a job while the worker is stopped.
  2. Inspects the pending job.
  3. Starts the worker, kills it and restarts it.
  4. Verifies completion.

## Non-Goals

- **Persistence internals.** Loom adds no persistence engine, scheduler, retry
  logic or lease logic. Selvedge owns these.
- **Backend-neutral workflow bodies.** Workflow functions keep
  `*sv.WorkflowContext`.
- **Generated surfaces.** No DSL declarations, generated job endpoints or
  transport changes.
- **Native-only features.** The roles do not offer signals, queries, child
  workflows, payload versioning or cancellation. The ticket does not request
  them, and they stay on the native path.
- **Transaction-bound admission through the roles.** This is deferred, and
  the ticket does not require it:
  - The roles hide their storage and offer no transaction facility.
  - A transaction context from another App cannot enlist the role's storage,
    so passing a context never makes role admission atomic with application
    writes.
  - Transaction-bound submission remains supported, unchanged, on the
    caller-owned native path (`docs/execution.md:49-53`).

## Current State

| Concern | Today | Owner |
|---|---|---|
| Typed activity contract | `execution.Activity[I, O]{Name}` (`execution/activity.go:53-58`) | Loom |
| Selvedge activity binding | `selvedge.New(app)` and `selvedge.Register`, with a caller-owned App (`execution/selvedge/activities.go:1-3`, `:39-80`) | Loom |
| Workflow definition and start | `sv.DefineWorkflow`, `sv.StartWorkflowByName`, `sv.WithInstanceID` | Selvedge |
| Instance, status and result reads | `App.GetInstance`, `sv.Describe`, `sv.GetWorkflowResult` | Selvedge |
| Producer-only lifecycle | None. Every App runs all loops (`app.go:390-410`). | Gap (upstream) |
| Start context | The App lifetime derives from the `Start` context (`app.go:135`) | Selvedge; handled on Loom's side (below) |
| Shutdown | One shared cleanup that continues after the caller times out (`app.go:167-196`). A finite internal timer can return the untyped error `shutdown timed out after ...` while cleanup continues. That error cannot be told apart from a cleanup failure. The timer comes from `WithShutdownTimeout`, which must be positive (`options.go:173-177`, `:321-327`). Any finite value leaves this case open. | Selvedge; supported completion wait requested in selvedge#2 |
| Duplicate reconciliation | `ErrWorkflowAlreadyStarted` only | Gap (Loom) |
| Migrations | `Start` applies them by default under a lock. `WithAutoMigrate(false)` skips them and does **no** schema validation: `initStorage` only opens storage and builds the engine (`app.go:267-296`). | Selvedge |
| Cross-process wake | Local notifications are process-local (`internal/storage/local_notify.go:24`). SQLite workers in other processes rely on polling (default 1 s; `WithMaxPollInterval` caps it, `options.go:297-299`). PostgreSQL uses LISTEN/NOTIFY. | Selvedge |
| Hooks | `sv.WithHooks`, `hooks/otel`. `OnWorkflowStart` fires in the admitting process with the value passed to start (`app.go:635`). | Selvedge |
| Workflow namespace | Database-wide (`README.md:556`) | Selvedge |

## Released Native Prerequisites

The native contracts are requested in
[CaliLuke/selvedge#2](https://github.com/CaliLuke/selvedge/issues/2), "Add a
supported producer-only lifecycle for workflow submission and lookup". This
design binds both roles to v0.8.1; see the
[release mapping and consumer evidence](selvedge-v081-prerequisite-check.md).
All five prerequisites below are satisfied by the released public contract.

Loom requires the following from a released Selvedge version:

1. **Producer-only startup.** The handle opens storage and validates
   configuration. Migrations are an explicit option, and the handle documents
   what it validates when migrations are disabled.
2. **No engine loops.** The producer-only handle starts none of these:
   - leader election
   - workflow resumption
   - activity workers
   - leader sweeps
   - dispatch notification handling
3. **No execution path.** The producer-only handle has no registration API,
   and it documents that no workflow or activity code, and no execution hook,
   runs through it.
4. **Existing operations on the producer-only handle.** It exposes the
   operations Loom already uses on v0.7.0:
   - By-name start with an explicit ID, keeping today's error classes: rejected,
     duplicate and uncertain.
   - An instance read that returns the workflow name, parent ID and stored
     input bytes.
   - A typed result read and `Describe`.
5. **Supported cleanup-completion wait, on both handles.** Shutdown runs one
   shared cleanup. A caller must be able to wait for that cleanup to complete
   and receive its result. The caller must also be able to tell an interrupted
   wait apart from completed cleanup.

   v0.7.0 does not provide this. Its finite internal timer can return an untyped
   error while cleanup continues (`app.go:186-195`), and raising the timer to a
   large finite value does not remove that case. Both roles therefore depend on
   this item. A Worker on v0.7.0 would also violate the lifecycle contract.

In v0.8.1, both handles mark interrupted waits with
`sv.ErrShutdownInterrupted`. The cleanup owner repeats native shutdown only
for that sentinel and records every unmarked result as final. This supplies
item 5 without an unbounded native wait API or a timer override.

**Not a prerequisite: the start context.** v0.7.0 derives the App lifetime from
the `Start` context (`app.go:135`). Loom's own lifetime context and watcher
(below) bound only initialization, so v0.7.0 semantics are enough here.
selvedge#2 also asks for native initialization-only start contexts. If that
ships, the watcher stays correct.

**Byte preservation is checked at runtime.** selvedge#2 also asks for a
documented byte-preservation contract for raw JSON submission. Loom's
fixed-point check before I/O (below) is correct with or without that contract.

**What the model shows (`execution/selvedge/tla/workflow_roles`).**

- **v0.7.0 client App:**
  - `ProducerNoEngineLoops` fails.
  - Invariant P holds only because nothing is registered and because of the
    empty-filter behavior. That is implementation behavior, not a contract.
- **v0.7.0 finite shutdown timer:** `CleanupComplete` fails
  (`lifecycle-native-timeout.cfg`).
- **Remaining invariants:** the other lifecycle, admission and lookup
  invariants hold with the Loom-side design. The `completionWait` and
  `producerHandle` assumptions are stated explicitly in the model.

The README of that model separates the conditional properties from the
Loom-side ones.

## Proposed Design

**Source of truth.** The application writes one `execution.Workflow[I, O]`
value, in a package that both binaries import. The workflow name is the
durable identity. A job's input identity is its stored bytes, not the Go value.

**Owners.**

- **`execution`** owns the contract value. It sits next to `Activity`.
- **`execution/selvedge`** owns:
  - the roles
  - the lifecycle gate
  - the Loom-side lifecycle adapters
  - input identity
  - reconciliation
  - lookup
- **Selvedge** owns:
  - storage
  - dispatch
  - leases
  - retries
  - output codecs
  - migrations
  - hooks

**Ownership policy.** Each role constructs and owns its native handle and never
exposes it. This is the only way to enforce Invariant P and the registration
gate. Control `exposeApp` shows that registration on an exposed v0.7.0 App
executes work in the producer process.

This changes the documented boundary in `docs/execution.md:32-34`, but only for
the new roles. `selvedge.New(app)` keeps its caller-owned contract.

## API / Contract Changes

```go
// package execution
type Workflow[I, O any] struct{ Name string } // stable across deploys and replay

// package execution/selvedge
func NewProducer(opts ...ProducerOption) (*Producer, error) // built on the upstream producer handle
func NewWorker(opts ...sv.Option) (*Worker, error)            // builds and owns an App
func (w *Worker) Activities() *Activities                     // registry gated by the worker lifecycle
func Declare[I, O any](r Role, wf execution.Workflow[I, O]) error
func RegisterWorkflow[I, O any](w *Worker, wf execution.Workflow[I, O], fn func(*sv.WorkflowContext, I) (O, error)) error
// Start(ctx) error; Shutdown(ctx) error; Ready() error on both *Producer and *Worker
func Submit[I, O any](ctx context.Context, r Role, wf execution.Workflow[I, O], id string, in I) (Admission, error)
func Lookup[I, O any](ctx context.Context, r Role, wf execution.Workflow[I, O], id string) (Job[O], error)

type Role interface{ role() *core } // sealed: *Producer, *Worker
type Admission struct{ ID string; Created bool }
type Job[O any] struct {
	ID        string
	State     sv.WorkflowStatus // Running, Completed, Failed, Cancelled
	Output    O                 // only when Completed
	Err       error             // recorded failure when Failed or Cancelled
	StartedAt, UpdatedAt time.Time
}
type AdmissionConflictError struct { // Unwrap: sv.ErrWorkflowAlreadyStarted
	ID, Workflow, ExistingWorkflow string
	InputDiffers, ChildInstance    bool
	ExistingInput                  []byte // stored input bytes, for the caller's earlier-attempt check
}
type AdmissionUncertainError struct { // Unwrap: the native or read error
	ID    string
	Input []byte // this call's frozen bytes; it may have committed them
	Err   error
}
var ErrNotRunning, ErrRegistrationSealed, ErrShutdownIncomplete, ErrInputEncoding error
```

`ProducerOption` uses `sv.Option`. `NewProducer` stores these options; its
`Start` opens the private native handle with `sv.OpenProducer`. The returned
producer lifetime is independent of the initialization context. Worker-only
options are ignored by the native producer. Producer operation observability
uses wrappers; `sv.WithHooks` applies to the Worker, not the Producer.

`NewWorker` passes the caller's native options through unchanged. Its native
shutdown wait uses the released completion-wait contract (prerequisite item 5).
Callers bound their own wait with the role's `Shutdown(ctx)`.

```go
// API process
p, err := selvedge.NewProducer(...)                                       // check err
err = selvedge.Declare(p, jobs.Export)
err = p.Start(ctx)
adm, err := selvedge.Submit(ctx, p, jobs.Export, req.JobID, input)
job, err := selvedge.Lookup(ctx, p, jobs.Export, req.JobID)
// worker process
w, err := selvedge.NewWorker(sv.WithDatabase(url))
err = selvedge.RegisterWorkflow(w, jobs.Export, exportWorkflow)          // application body
err = selvedge.Register(w.Activities(), jobs.ReadMetadata, readMetadata, sv.ActivityOptions{})
err = w.Start(ctx)
```

## Detailed Behavior

### Invariant P

**Invariant P.** A process whose only roles are Producers never runs workflow
functions, activity handlers or inline activity attempts. It also runs no
engine loops. Two parts enforce this:

- The upstream handle guarantees it (prerequisite items 1–3).
- Loom provides no registration path on `*Producer`.

Loom adds no queue or filter workaround.

### Lifecycle gate

One mutex guards the state, the declared contracts and the registration gate.
The Worker's `Activities` registry takes this gate before its own lock, so the
lock order is gate first, then registry.

| State | Declare / Register (incl. `Activities()`) | Start | Submit / Lookup | Ready | Shutdown(ctx) |
|---|---|---|---|---|---|
| new | allowed | → starting | `ErrNotRunning` | `ErrNotRunning` | → stopped; returns nil |
| starting | `ErrRegistrationSealed` | error (already starting) | `ErrNotRunning` | `ErrNotRunning` | records the request; waits as in stopping |
| running | `ErrRegistrationSealed` | error | admitted | nil | → stopping; launches cleanup; waits |
| stopping | `ErrRegistrationSealed` | error | `ErrNotRunning` | `ErrNotRunning` | waits for the same cleanup |
| stopped | `ErrRegistrationSealed` | error (no restart, although native Apps allow one) | `ErrNotRunning` | `ErrNotRunning` | returns the recorded cleanup result |

```go
func Start(ctx) error {
	lock; if state != new { unlock; return stateError }; state = starting; unlock
	life, cancelLife := context.WithCancel(context.Background()) // Loom-owned native lifetime
	watch := startWatcher(ctx, cancelLife)     // cancels life only while initialization runs
	err := native.start(life)
	watch.stop()                               // after stop returns, the watcher can no longer cancel
	cancelled := life.Err() != nil             // the caller gave up during initialization
	lock; defer unlock
	switch {
	case shutdownRequested:                    // Shutdown arrived during starting
		state = stopping; launchCleanup(nativeStarted: err == nil)
		return errors.Join(ErrNotRunning, err) // never reports success
	case err == nil && cancelled:              // started, but its lifetime is already cancelled
		state = stopping; launchCleanup(nativeStarted: true)
		return errors.Join(ErrNotRunning, ctx.Err())
	case err != nil:
		state = stopped; result = nil; return err // native start already released its resources
	default:
		state = running; return nil
	}
}
func Shutdown(ctx) error {
	lock; switch state { case new: state = stopped; unlock; return nil
	case starting: shutdownRequested = true
	case running: state = stopping; launchCleanup(nativeStarted: true) }
	done := cleanupDone; unlock
	select {
	case <-done: return result                 // the eventual cleanup result, identical for every caller
	case <-ctx.Done(): return fmt.Errorf("%w: %w", ErrShutdownIncomplete, ctx.Err()) // state stays stopping
	}
}
// launchCleanup runs once. It waits for in-flight role operations to drain. If the
// native handle started, it then waits, without a deadline of its own, for native
// cleanup to complete, using the released completion-wait contract (prerequisite 5).
// It records that cleanup result, sets state = stopped and closes cleanupDone. If
// Shutdown arrives during starting, cleanupDone is created at that point.
```

**Lifecycle adapters.**

- **Start context (Loom-side, v0.7.0 semantics suffice).**
  - v0.7.0 derives the App's lifetime from the context passed to `Start`
    (`app.go:135`).
  - Passing the caller's context directly would let a later cancellation kill
    a running role (control `callerCtx`).
  - The Loom-owned `life` context and its watcher bound initialization,
    including migration lock waits, by the caller's context. After that, the
    handle lives until `Shutdown`.
- **Native cleanup wait (native prerequisite 5).**
  - Loom records a cleanup result only when the native contract reports that
    cleanup has completed.
  - On v0.7.0, a finite internal timer can return an untyped error while
    cleanup continues (`app.go:186-195`). Recording that error would report
    completion before storage closes. The model's v0.7.0 baseline
    (`lifecycle-native-timeout.cfg`) shows this violates `CleanupComplete`.
  - With v0.8.1, the cleanup owner loops over
    `native.Shutdown(context.Background())` only while
    `errors.Is(err, sv.ErrShutdownInterrupted)`. Any unmarked result is final,
    even when it wraps `context.Canceled` or `context.DeadlineExceeded`.
    This is the supported native wait protocol, not a timeout override. Keep
    the handle private and the role non-restartable until and after completion.
  - The caller's own bound always comes from the role's `Shutdown(ctx)`.

**Operations and readiness.**

- **`ErrNotRunning` from `Submit`.** No insert was attempted, so this call
  definitely admitted nothing.
- **In-flight operations.** A `Submit` or `Lookup` admitted while `running`
  holds an in-flight count. It completes against live storage, and cleanup
  waits for it before calling native shutdown (control `noDrain`).
- **`Ready`.** It reports lifecycle state only. Selvedge has no health API.
  Deployments that want a database check add their own.
- **Worker shutdown.** It keeps the native semantics: claims are released, and
  interrupted work resumes elsewhere.

### Contract validation

Validation happens at declaration time.

```go
func declare(c *core, name string, in, out reflect.Type) error {
	if len(name) == 0 || len(name) > 255 { return errInvalidName }   // Selvedge limit (app.go:600-602)
	if prior, ok := c.contracts[name]; ok {
		if prior.in != in || prior.out != out { return errTypeCollision }
		return nil                                                    // idempotent redeclaration
	}
	c.contracts[name] = contract{in, out}; return nil
}
// Submit and Lookup on an undeclared name or with mismatched types fail without I/O.
```

Names share one database-wide namespace (`README.md:556`). Loom adds no
prefixes and no aliases.

If no worker registers a declared name, jobs for that name stay `Running`, and
nothing detects it.

Payload compatibility across builds is the application's responsibility.

### Admission with frozen input bytes

```go
func Submit(ctx, r, wf, id, in) (Admission, error) {
	if err := r.admit(wf); err != nil { return Admission{}, err }   // running and declared; holds in-flight count
	defer r.release()
	if id == "" { return Admission{}, errEmptyID }
	encoded, err := json.Marshal(in)                // the user's MarshalJSON runs exactly once per call
	if err != nil { return Admission{}, fmt.Errorf("%w: %w", ErrInputEncoding, err) }       // not admitted
	frozen, err := json.Marshal(json.RawMessage(encoded))                                     // native normalization
	if err != nil { return Admission{}, fmt.Errorf("%w: %w", ErrInputEncoding, err) }       // not admitted
	again, err := json.Marshal(json.RawMessage(frozen))  // what native start will store (app.go:611)
	if err != nil || !bytes.Equal(again, frozen) || !utf8.Valid(frozen) {
		return Admission{}, fmt.Errorf("%w: unstable or non-UTF-8 encoding", ErrInputEncoding) // not admitted
	}
	err = native.start(ctx, wf.Name, json.RawMessage(frozen), id)
	switch {
	case err == nil:
		return Admission{ID: id, Created: true}, nil
	case errors.Is(err, sv.ErrWorkflowStartRejected):
		return Admission{}, err                     // this call inserted nothing
	case errors.Is(err, sv.ErrWorkflowAlreadyStarted):
		inst, rerr := native.instance(ctx, id)
		if rerr != nil { return Admission{}, &AdmissionUncertainError{ID: id, Input: frozen, Err: rerr} }
		if inst.WorkflowName != wf.Name || inst.ParentInstanceID != "" || !bytes.Equal(inst.InputData, frozen) {
			return Admission{}, &AdmissionConflictError{..., ExistingInput: inst.InputData}
		}
		return Admission{ID: id, Created: false}, nil // may already be terminal
	default:
		return Admission{}, &AdmissionUncertainError{ID: id, Input: frozen, Err: err} // retry with the same ID
	}
}
```

**One encoding per call.** The same `frozen` bytes are inserted and compared,
so a stateful or nondeterministic `MarshalJSON` cannot produce a false match.
Control `reencode` shows the false match that a fresh encoding produces. Two
attempts match only when their bytes are equal, and that equality is the
definition of job identity.

**Stored bytes.**

- **Fixed point.** The fixed-point property is now checked at runtime, not
  assumed. Native start re-marshals the `json.RawMessage` (`app.go:611`).
  Loom computes that result and rejects the call before any I/O unless it
  equals `frozen`.
- **UTF-8.** `input_data` is a `TEXT` column (SQLite and PostgreSQL
  `20251217000000_initial_schema.sql:35`). PostgreSQL rejects invalid UTF-8 in
  `TEXT`. A raw message passes invalid UTF-8 through unchanged. Without the
  UTF-8 check, the insert would fail after it was attempted, and every retry
  of that ID would be classified as uncertain.

**Probe evidence.** A scratch program under go1.27.3, the version pinned in
`go.mod`, produced these results:

- **Idempotence.** `json.Marshal(json.RawMessage(x))` was idempotent for:
  - compact and whitespace-padded objects
  - HTML-escaped characters
  - U+2028
  - duplicate keys
  - large integers
  - `-0`
  - `null`
- **Invalid UTF-8.** A raw message containing `"\xff"` passed through unchanged
  and failed `utf8.Valid`.
- **Stateful encoders.** A stateful `MarshalJSON` produced different bytes on
  each call.

This is evidence, not proof. The runtime check keeps Loom correct even when
the property fails.

**Interop boundary.** `execution/selvedge/workflow_encoding.go` alone uses
Selvedge's native JSON implementation for the accepted identity contract. It
owns the one application encoding, fixed-point normalization and raw-input
conversion; all other files pass frozen bytes. The exact-path lint exception
preserves the repository's JSON v2 rule everywhere else. Native field, nil and
marshaler semantics cannot be replaced with JSON v2 defaults without changing
this contract.

**Codec and version.** The supported input codec is the native JSON codec, through
`json.Marshal` on the declared `I`. If the codec or Go version changes between
builds, a retry can become a false conflict. That is the safe direction.

**Hooks and decoding.** Native `OnWorkflowStart` hooks receive the frozen
`json.RawMessage`, not the typed value. Workers decode the stored bytes into
`I`, as they do today.

**Caller retry contract.** Every `Submit` result describes **that call**. A
caller that retries one logical request under one ID keeps the `Input` bytes of
every `AdmissionUncertainError` it has received for that ID. Call that set
`U`. The caller then draws exactly these conclusions:

| Result of this call | What it establishes | Caller conclusion |
|---|---|---|
| `Created` | This call inserted a job holding this call's name and bytes | Admitted |
| `Created=false` | A job exists with this name, no parent, and bytes equal to this call's bytes | Admitted, with an identity equal to this call's. It may be an earlier attempt's commit or another request with the same identity. |
| `AdmissionConflictError` | The stored job differs from **this call**: a different name, a parent, or different bytes. Nothing more. | If `ExistingWorkflow` is this workflow, `ChildInstance` is false, and `ExistingInput` equals some element of `U`, a job exists matching an earlier attempt's identity; another caller may have inserted the same identity. The request is **admitted** with those bytes. Otherwise, the ID is held by a job whose identity matches none of this request's possibly committed attempts, and the request is **not admitted under this ID**. |
| `AdmissionUncertainError` | Unknown; this call may have committed `Input` | Add `Input` to `U` and retry the same ID |
| `ErrWorkflowStartRejected`, `ErrNotRunning`, `ErrInputEncoding` | This call inserted nothing | If `U` is empty, **not admitted**. Otherwise **unknown**, because an earlier uncertain attempt may have committed. Retry the same ID. |

**Why a conflict needs the earlier-attempt check.** With a stateful or
nondeterministic `MarshalJSON`, a retry can encode the same value to different
bytes. The job found at the ID can then be the request's own earlier commit,
so reading a conflict as "not admitted" would be wrong. Control `conflictFinal`
reproduces this case.

**Bounded retries.** A caller that stops retrying while `U` is non-empty has an
unknown outcome. It must not report "not admitted".

The model checks this contract with these invariants:

- `RejectedMeansAbsent`, with control `rejectFinal`. A pre-insert rejection is
  modeled independently of whether the ID is occupied.
- `ConflictMeansAbsent` and `ConflictSound`.
- `AcceptedIdentity`, which covers both the earlier-attempt conclusion and
  `Created=false`.
- `OneJobPerRequest`. Control `newIdOnUncertain` shows the duplicate job that a
  new ID creates.
- `CreatedOnce`.

### Lookup

1. Lookup reads the instance.
2. It rejects a name mismatch without decoding into `O` (control
   `decodeFirst`).
3. It then reads the typed result. Terminal states never change, so the later
   read is authoritative.

Error cases:

- An unknown ID returns `*sv.WorkflowNotFoundError`.
- A decode failure is a lookup error, not `Job.Err` (`workflow.go:160-172`).

**Pending jobs.** A job that was admitted but not yet claimed reports
`Running`, with zero `Output` and nil `Err`. Loom does not expose `LeaseEpoch`,
because it is a lease internal. For the acceptance test, "inspect the pending
job" means `Lookup` returns `Running` and the worker's application-owned
execution markers are absent. Before any worker starts, every admitted job is
`Running` (model `PendingUntilWorker`). With the worker eventually up, every
admitted job completes exactly once, across one crash (`Durability`,
`CompletedOnce`).

**Projection surfaces.** The Go runtime APIs, `docs/execution.md`, the consumer
`loom` skill and the framework skill's boundary note. No generated artifacts
change, and HTTP, JSON-RPC and gRPC are unaffected. Applications map the
outcomes themselves, for example:

- conflict → 409
- uncertain → 503
- `ErrNotRunning` → 503
- `ErrInputEncoding` → 400

### Responsibility split

| Responsibility | Loom | Selvedge / deployment |
|---|---|---|
| Contract typing; name and type collisions | Yes | — |
| Lifecycle gate, readiness, shutdown wait semantics | Yes, including the start-context watcher, draining, and one recorded cleanup result | Handle start and cleanup; supported cleanup-completion wait (selvedge#2) |
| No local execution in producers | No registration path | Producer-only handle (released in v0.8.0) |
| Admission identity and reconciliation | Single encoding, fixed-point and UTF-8 checks, comparison, ID policy | Insert, uniqueness, error classes |
| Transactions | None on the roles (deferred) | Native caller-owned path |
| Leases, retries, takeover | — | Selvedge |
| Migrations | Passes native options through; defaults unchanged | Applied under a lock; the deployment chooses the migrating process (recommended: a deploy step or the worker). With migrations disabled, no schema check runs at Start, so the schema must exist before the role starts. |
| Observability | No new hooks | Worker: `sv.WithHooks` / `hooks/otel`. Producer: wrap client operations; native workflow hooks never run, including `OnWorkflowStart`. |
| Cross-process latency | None promised | Polling intervals and options (SQLite); LISTEN/NOTIFY (PostgreSQL) |

## Contract Matrix

| Capability | Supported | Rejected | Deferred | Default | Source of truth | Placement | Routing | Naming | Equivalence | Proof |
|---|---|---|---|---|---|---|---|---|---|---|
| Workflow contract | Shared `Workflow[I,O]` | Empty or >255-byte name; same name with other types | Versioning | Must declare | App package | `execution` | Declare / Register | Database-wide, no aliases | Worker and producer types match through the shared value | Collision tests |
| Producer | Admit, lookup | Registration (no API); restart | Cancel, transactions | Upstream defaults | Role options | `execution/selvedge` | Upstream producer handle | — | — | Obl. 1 (released native guarantee) |
| Worker | Register, execute, admit, lookup | Registration after `new`; restart | Dynamic registration | Native defaults | Role options | `execution/selvedge` | Selvedge dispatch | — | Single-process results equal two-process results | Obl. 3, 5 |
| Lifecycle | Five-state gate; Loom-owned start lifetime | Operations outside running; second Start | Database readiness probe | `new` | Role | core | — | — | Identical cleanup result for every waiter | Obl. 4; lifecycle model |
| Admission | Idempotent for the same bytes | Empty ID; undeclared; conflict; encode, unstable or non-UTF-8 input | Semantic JSON, role transactions | `Created` reported | Stored name, parent and input bytes | `Submit` | Insert → reconcile | Caller ID | A retry equals the first admission | Obl. 2; admission model |
| Lookup | Typed status and result | Name mismatch; undeclared | Pending vs claimed | — | Stored instance | `Lookup` | Native reads | — | Same `Job` from either role | Obl. 6 |
| Native path | Unchanged, including transactions | — | — | Caller-owned | Docs | Unchanged | Unchanged | — | Unchanged behavior | Existing tests unchanged |

## Accepted and Rejected Examples

**Admission**

- Submit(`exports.run`, "j1", {a}) → Created. A retry after a timeout returns
  `Created=false` with a nil error.
- Submit(`exports.run`, "j1", {b}) → `AdmissionConflictError{InputDiffers}`.
- A retry where a stateful `MarshalJSON` now encodes {a} differently → that
  call returns `AdmissionConflictError`; it is never a silent match.
  - If no earlier attempt was uncertain, the request is not admitted under "j1".
  - If the first attempt returned `AdmissionUncertainError{Input: b1}` and
    `ExistingInput == b1`, the request is admitted: the first attempt committed.
- An input whose `MarshalJSON` emits a string containing byte `0xff` →
  `ErrInputEncoding`, before any I/O.
- The first attempt is uncertain, and the second returns
  `ErrWorkflowStartRejected` because the context was cancelled. The caller must
  not report "not admitted". It retries "j1", gets `Created=false`, and
  proceeds.

**Identity conflicts**

- Submit(`webhooks.deliver`, "j1") when "j1" belongs to `exports.run` → a
  conflict that names `exports.run`.
- A child instance ID → a conflict with `ChildInstance`.

**Rejections and errors**

- Submit("") or an undeclared workflow → rejected before I/O.
- Submit during stopping → `ErrNotRunning`.
- Declare `[A,B]` and then `[A,C]` under one name → type collision.
- Registration through `w.Activities()` after `w.Start` →
  `ErrRegistrationSealed`.

**Lifecycle**

- Shutdown with a context that has a 1 ms deadline → `ErrShutdownIncomplete`,
  and the state stays `stopping`. A later Shutdown with
  `context.Background()` returns the cleanup result.
- Start racing Shutdown during starting → Start returns an error matching
  `ErrNotRunning`, and the handle is cleaned up.
- The Start caller cancels its context after `Start` returns nil → the role
  keeps running.

## Alternatives Considered

- **Use the v0.7.0 client App with empty queues and no registrations.**
  Rejected. It relies on filter behavior, and its leader and sweep loops keep
  running (model `ProducerNoEngineLoops`).
- **Caller-owned App plus documentation.** Rejected. It cannot enforce
  Invariant P or the registration gate (control `exposeApp`).
- **Pass the caller's Start context to native start.** Rejected (control
  `callerCtx`).
- **Treat a v0.7.0 native shutdown return as completed cleanup.** Rejected. The
  finite timer can fire first (`lifecycle-native-timeout.cfg`).
- **Override the native shutdown timeout with a huge finite duration.**
  Rejected. A finite deadline cannot establish that cleanup completed, and the
  workaround hides a native contract gap. The supported completion wait is a
  native prerequisite (selvedge#2).
- **Compare a re-encoded value during reconciliation.** Rejected (control
  `reencode`).
- **Assume the encoding fixed point.** Superseded. Loom checks the fixed point
  before I/O.
- **Semantic JSON comparison.** Deferred. It raises questions about number and
  duplicate-key normalization.
- **Exposing transactions on the roles.** Deferred. It needs a native facility
  bound to the role's storage, and the ticket does not require it.
- **DSL job declarations.** Deferred. There is no consumer evidence for a
  generated shape.
- **Partial delivery before the producer handle exists.** Rejected. The ticket
  requires both roles, and a Worker-only release does not complete it.

## Proof Obligations

**Model status.** The lifecycle and admission models have been run (see the
model README). Every obligation below is still a Go implementation obligation;
the models do not prove arbitrary Go correctness.

1. **Invariant P (released native guarantee).**
   - A producer-only process records no workflow execution, activity attempt,
     execution hook, leader lease or sweep. This holds whether workers are
     stopped, running or newly registered.
   - This requires the upstream handle.
2. **Admission.**
   - For any interleaving of concurrent and retried Submits with the same ID,
     name and bytes, exactly one instance exists, and at most one response
     reports Created.
   - A differing name, parent or bytes always produces a conflict.
   - Inputs that fail the fixed-point check or UTF-8 validation are rejected
     before I/O.
   - The stored bytes round-trip unchanged on SQLite and PostgreSQL.
   - The caller's conclusions follow the retry-contract table:
     - A rejection after an uncertain attempt never yields "not admitted".
     - A conflict yields "not admitted" only when the stored identity matches
       none of the request's uncertain-attempt bytes.
     - Uncertain errors and conflicts carry the bytes needed to decide.
3. **Durability.**
   - A job admitted with no worker stays `Running`.
   - After a SIGKILL during an activity, a restarted worker completes the job,
     with one recorded result.
   - The test's wait bounds follow the configured polling intervals; no wake
     latency is promised.
4. **Lifecycle.**
   - No registration succeeds after the role leaves `new`.
   - Start reports success only when the role reaches `running`. A later
     cancellation of the caller's context never stops a running role.
   - Every Shutdown waiter receives either `ErrShutdownIncomplete` or the
     single recorded cleanup result.
   - The cleanup result is recorded only after the native completion-wait
     contract reports completed cleanup. v0.8.1 supplies this distinction;
     the adapter must retry only `sv.ErrShutdownInterrupted`.
   - Storage is never closed while an admitted operation is in flight.
5. **Single process.** Worker results match the two-process results for the
   same contract and inputs.
6. **Lookup.** Lookup never decodes the output of another workflow name.

## Acceptance Coverage (#1024)

| Ticket acceptance item | Design coverage | Status |
|---|---|---|
| Typed workflow/activity inputs and results; application-owned implementation | `Workflow[I,O]`, `RegisterWorkflow`, existing typed activities | Implemented |
| Independent producer and worker initialization, shutdown, readiness; producer never executes work | Roles, gate, start-context watcher, Invariant P | Implemented on released v0.8.1 contracts |
| Supported status/result lookup and stable identity, incl. retry after an uncertain response | `Submit`, `Lookup`, single-encoding identity, caller retry contract | Implemented, race-tested and bounded-model checked |
| Two-process example; test submits with the worker stopped, inspects, starts, completes | Obligation 3 and the Goals sequence, plus kill and restart | Implemented; SQLite and PostgreSQL process-restart checks pass |
| Single-process deployment remains supported | `Worker` admits and looks up; the native path is unchanged | Implemented |
| Document Loom versus runtime responsibilities, including migrations and observability | Responsibility split | Implemented |

## Delivery evidence and limits

**No remaining external gate.** [v0.8.1](selvedge-v081-prerequisite-check.md)
satisfies all five prerequisites. The [acceptance record](../../workflow_roles_validation.md)
records the implemented contract, real-store and two-process checks. The
independent exact-diff review remains the pre-commit gate.

**Routine decisions made in this design:**

- readiness reports lifecycle state only
- each role owns its handle
- the roles offer no cancellation and no transactions
- a pending job reports `Running`
- input identity is the native-normalized JSON codec bytes, and the
  fixed point and UTF-8 validity are checked before I/O
- uncertain and conflict errors carry input bytes for the caller's retry
  conclusions

**Evidence resolved in this revision:**

- `WithAutoMigrate(false)` performs no schema validation at Start.
- Cross-process SQLite wake relies on polling.
- `App.GetInstance` exposes the stored input bytes.
- The README documents the storage-level dispatch filter. The activity filter
  is not documented.

**Still missing:**

- The object-metadata consumer's code, which is not in this workspace.
