# Durable workflow roles (#1024)

These models are design evidence for
[the implemented workflow contract](design.md).
The role implementation lives in `execution/selvedge/workflow*.go`. The existing `ActivityBinding` and
`PolicyTest` models in the parent directory keep their historical meaning and
are unchanged.

The released native prerequisites are recorded upstream in
[CaliLuke/selvedge#2](https://github.com/CaliLuke/selvedge/issues/2).

The [v0.8.1 release check](selvedge-v081-prerequisite-check.md)
confirms both native prerequisites are released. The adapter repeats native
shutdown only for `ErrShutdownInterrupted` using a cleanup-owned background
context, and records any unmarked result as final. Interrupted observations
stutter at the `completionWait` abstraction; they never publish a result or
mark the role stopped. Preserve the no-restart rule. Model transitions are
unchanged, so this correspondence update reuses the recorded runs below.

## Models

`WorkflowLifecycle.tla` models one role, either a Producer or a Worker, as a
state machine over its lifecycle gate (`new`, `starting`, `running`, `stopping`,
`stopped`). It covers:

- the role's native handle
- a Loom-owned start context, and a watcher that cancels it only during
  initialization
- the context of the Start caller
- two concurrent Shutdown callers, whose waits can be interrupted
- two Submit/Lookup operations that hold in-flight counts
- registration sealing
- execution of user code

Two constants state the native assumptions explicitly:

- **`Upstream`** selects the native handle.
  - `"producerHandle"` assumes the producer-only handle from selvedge#2.
  - `"clientApp"` is the v0.7.0 App with nothing registered.
- **`NativeShutdown`** selects the shutdown contract.
  - `"completionWait"` assumes the supported cleanup-completion wait from
    selvedge#2: native shutdown reports a cleanup result only after cleanup
    completes.
  - `"v070Timer"` is the truthful v0.7.0 baseline. Its finite internal timer may
    return an untyped error while cleanup continues, and Loom cannot tell that
    error apart from a cleanup result. Raising the timer to a large finite value
    does not remove this case, so the design does not rely on it.

`WorkflowAdmission.tla` models two concurrent requests for one caller-chosen ID.
It covers:

- one encoding per Submit call, from a nondeterministic stateful codec
- native insert outcomes:
  - created
  - rejected before the insert, independent of whether the ID is occupied
  - duplicate
  - uncertain, with or without a commit
- reconciliation reads that may themselves be uncertain
- a pre-existing child instance
- typed lookup
- a worker that starts after admission and can crash once

`rq[r].outcome` is the caller's conclusion for request `r`. It is derived from
the result of each Submit call and from the encoded identities of the
request's earlier uncertain attempts.

## Conditional and Loom-side results

| Property | Loom-side design | Native assumption required |
|---|---|---|
| `RunningUsable`: cancelling the Start caller's context does not kill a running handle | The Loom-owned start context and its watcher. Control `callerCtx` fails. | None; v0.7.0 semantics suffice |
| `NoUseAfterClose`, `ShutdownRequestHonored`, `RegistrationSealed`, `StartTruthful`, `ShutdownTruthful` (a waiter returns a result only after Loom records cleanup) | The Loom gate | None |
| `CleanupComplete`: the recorded cleanup result exists only after native storage closes | Loom waits for the native result | `completionWait` (selvedge#2). The v0.7.0 baseline fails (`lifecycle-native-timeout.cfg`). |
| Admission identity: `ConflictMeansAbsent`, `OneJobPerRequest`, `RejectedMeansAbsent`, `LookupNeverDecodesOther` | Loom reconciliation over by-name start and the instance read | Atomic insert, a rejection inserts nothing, bytes round-trip (assumed below) |
| `InvariantP`: the producer executes no user code | No registration path on `*Producer` | `producerHandle` as a documented guarantee. On the v0.7.0 client App it holds only because nothing is registered and because of undocumented empty activity-task filters. |
| `ProducerNoEngineLoops`: no leader election, sweeps, resumption or activity loops | — | `producerHandle`. It fails on the v0.7.0 client App (`lifecycle-client-app.cfg`). |

## Commands

The checks were run with:

- Amazon Corretto 25.0.5
- `tla2tools` 1.7.4 (SHA-256
  `936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`)

Run each check from this directory, giving each run its own metadir.
`-deadlock` is required because terminal lifecycle states are expected.

```sh
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 4 -deadlock \
  -metadir "$(mktemp -d)" -config lifecycle-producer.cfg WorkflowLifecycle.tla
java -XX:+UseParallelGC -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 4 -deadlock \
  -metadir "$(mktemp -d)" -config admission-checked.cfg WorkflowAdmission.tla
```

## Observed results (2026-11-05, focused revision)

### Lifecycle model

| Configuration | Native assumption | Expected | Observed |
|---|---|---|---|
| `lifecycle-producer.cfg` | `producerHandle`, `completionWait` | All nine invariants and `CleanupTerminates` hold | Pass: 6,040 generated, 1,986 distinct, depth 15 |
| `lifecycle-worker.cfg` | `completionWait` | Same as above | Pass: 16,398 generated, 5,290 distinct, depth 17 |
| `lifecycle-client-app-p.cfg` | `clientApp`, `v070Timer` | `InvariantP` and the gate invariants hold | Pass: 7,933 generated, 2,500 distinct, depth 15 |
| `lifecycle-client-app.cfg` | `clientApp`, `v070Timer` | `ProducerNoEngineLoops` is violated | Violated: native init starts engine loops |
| `lifecycle-native-timeout.cfg` | `v070Timer`, Loom design unchanged | `CleanupComplete` is violated (v0.7.0 baseline) | Violated: the finite timer returns while storage is closing, and Loom records that error as the cleanup result |
| `lifecycle-expose-app.cfg` | `clientApp`, `v070Timer` | `InvariantP` is violated (W(a)) | Violated: registration through the exposed v0.7.0 App executes work |
| `lifecycle-timeout-stops.cfg` | `completionWait` | `ShutdownTruthful` is violated (W(d)) | Violated: an interrupted wait marks the role `stopped`, and a later Shutdown returns while cleanup is still draining |
| `lifecycle-no-drain.cfg` | `completionWait` | `NoUseAfterClose` is violated | Violated: storage closes while a Submit is in flight |
| `lifecycle-caller-ctx.cfg` | `completionWait` | `RunningUsable` is violated | Violated: cancelling after Start kills the running handle |
| `lifecycle-start-ignores-shutdown.cfg` | `completionWait` | `ShutdownRequestHonored` is violated | Violated: Start reports running after Shutdown arrived during starting |

### Admission model

| Configuration | Expected | Observed |
|---|---|---|
| `admission-checked.cfg` | All nine invariants and `Durability` hold | Pass: 406,392 generated, 73,644 distinct, depth 15 |
| `admission-reencode.cfg` | `AcceptedIdentity` is violated (W(b)) | Violated: `r2` sends `x1`, its fresh encoding `x2` matches the stored `x2`, and the reply is `existing` |
| `admission-new-id.cfg` | `OneJobPerRequest` is violated (W(c)) | Violated: an uncertain commit on `j1` followed by a retry on `j2` leaves two jobs |
| `admission-reject-final.cfg` | `RejectedMeansAbsent` is violated | Violated: an uncertain commit, then a pre-insert rejection reported as "not admitted" while the job exists |
| `admission-conflict-final.cfg` | `ConflictMeansAbsent` is violated | Violated: `r1` commits `x1` with an uncertain reply, the stateful retry encodes `x2`, and the conflict is read as "not admitted" although `j1` holds `r1`'s own `x1` |
| `admission-decode-first.cfg` | `LookupNeverDecodesOther` is violated | Violated: B's lookup decodes A's completed output |

Every violating run exits with status 12; every passing run exits with 0.

### Changes to the models

**`reject-final` control.** It first passed vacuously, because the model
allowed a native rejection only when the ID was absent. A native rejection
happens before storage is consulted, independent of whether the ID is occupied,
so the model was corrected.

**This revision:**

- **Conflict handling.** A conflict now proves only a mismatch with the current
  call. The caller compares the conflict's stored identity with its earlier
  uncertain attempts; control `conflictFinal` shows why that comparison is
  needed.
- **Shutdown.** The giant-timer workaround was removed. The shutdown property
  is now conditional on an explicit native assumption.
- **`expose-app`.** It now runs on the v0.7.0 client App, because an exposed
  App cannot assume the guarantees of the producer handle.

## Assumptions and limits

**Bounds:**

- one role per lifecycle run
- two Shutdown waiters and two operations
- two requests, two IDs, at most two attempts, and one crash
- a three-value encoding alphabet with overlapping encodings

**Assumed native behavior.** The models assume, and do not check, that:

- inserts are atomic per instance ID
- a rejection means the call inserted nothing
- stored `TEXT` bytes round-trip
- lease-fenced completion records one result (upstream `docs/formal/Selvedge.tla`)
- native Start and Shutdown serialize on the App mutex

`completionWait` and `producerHandle` correspond to the released v0.8.1
contracts through the adapter protocol above. They are not v0.7.0 behavior.
Eventual cleanup completion remains a liveness assumption; passing probes do
not prove arbitrary storage or Go execution behavior.

**Not checked:**

- SQL behavior
- JSON codec behavior. The go1.27.3 probe recorded in the design is evidence,
  not proof.
- process kills
- polling latency
- Go refinement

The model does not define any upstream API.

## Implementation and regression correspondence

- `workflow_lifecycle.go` owns the lifecycle gate, Start watcher, operation drain,
  independent waiters and sentinel-only native cleanup loop.
  `workflow_lifecycle_test.go` exercises each rejected ordering and retained
  activity registration, including final errors that wrap context errors.
- `workflow_native_contract_test.go` checks the released producer/worker
  completion contract through supported public APIs.
- `workflow_encoding.go` freezes native input bytes; `workflow_admission.go`
  reconciles identity. `workflow_admission_test.go` covers stateful codecs,
  uncertainty, rejection after uncertainty and conflict with earlier own bytes.
- `workflow_lookup.go` checks identity before typed decoding;
  `workflow_lookup_test.go` checks the wrong-name and state/error boundaries.
- `workflow_process_test.go` runs separate producer and worker binaries, admits
  before worker startup, kills at an acknowledged activity boundary, restarts
  and verifies one durable result. `workflow_postgres_test.go` repeats the
  process scenario and checks real stored bytes for both roles.

The [acceptance record](../../workflow_roles_validation.md) records commands,
versions and their scope. The tests check Go and native behavior outside the
bounded models; they do not establish exactly-once business side effects.
These models remain explicit design tools, outside builds, hooks, CI and release
eligibility.
