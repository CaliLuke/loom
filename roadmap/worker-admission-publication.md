# Worker admission and publication — #704

Delivered as `f1d1368fb` on main, pushed; #704 confirmed closed. Independent
production and integration reviews approved the exact diff without findings.
Implementation parent `ec2536d3c`; integration parent `464368176`.
Reuse the prior construction probes and `WorkerPublication` model from
`pool-construction-ownership/loom`. At readiness, node/worker/publication/routing
owners were byte-identical to that probe's `d57f96c6c` parent. Reader lifetime
was inspected again because #759 changed that owner.

## Contract and owner

`AddWorker` must own its construction attempt before doing Redis work. Success
transfers one prepared worker to the node's lifecycle; failure releases local
resources and withdraws only that unique attempt's distributed publication.
Concurrent Close must exclude later admission and join attempts already admitted.
No failed attempt may start user handlers or renew a lease in the background.

Peer replicas are independently stale. A stale routable ID is not itself proof
of lost work: `routeWorkerEvent` writes with `XADD NOMKSTREAM`, and retains the
original pool event until a worker acknowledgement. Stream deletion rejects
later routes; already queued messages still have an unacknowledged pool event
eligible for existing recovery. The old model's `QueuedHasOwner` invariant is
therefore stronger than the actual recovery contract and must not force a new
retained-drain runtime.

Implemented boundary:

1. Register an in-flight construction under `node.lock`, rejecting closing or
   client-only nodes. Close sets closing under the same lock, joins these attempts
   before enumerating local workers, and never holds that lock while joining.
2. Prepare the unique stream and oldest-position Reader, but do not Subscribe or
   launch worker loops. The Reader starts polling only at Subscribe; failed
   preparation therefore does not depend on #800's blocked native-read shutdown.
3. Publish membership/initial heartbeat and wait for the existing local-replica
   guarantee. Keep construction ownership throughout ambiguous replies. Before
   transfer, recheck node closing and lease age under the admission boundary.
4. Transfer into localWorkers/workerStreams and start all worker loops as one
   local admission decision. Only this path invokes handlers or renews leases.
5. Abort closes the unstarted Reader/context and attempts private stream and
   membership/heartbeat withdrawal even when the caller context was canceled.
   Return cleanup failures with the original error. Do not delete shared job
   payloads, epochs, protocol data or another worker's state. Model an uncertain
   cleanup result explicitly: no renewal/handler, finite initial lease, original
   event still recoverable. Do not promise immediate global withdrawal during a
   Redis outage or bounded Redis/handler latency.

The bounded model supports retaining the existing Redis operations: preparation,
publication, local transfer and abort belong to one private worker-admission owner.
No new atomic Redis transaction is needed to protect a job event because its
original PEL entry remains independently recoverable. Abort must attempt all
private withdrawals, collect errors, and use a detached cleanup context bounded
by the existing worker shutdown budget. An uncertain cleanup is explicitly weaker
than confirmed withdrawal; neither arbitrary late Redis command execution nor
indefinite outages are covered by an immediate global-cleanup claim. Reuse existing
lease fencing and recovery instead of adding per-peer admission state.

## Sequence and bounded acceptance

- [x] Read existing family evidence and current consumers; verify unchanged
  production owners and Reader's lazy start.
- [x] Reuse early-publication, committed-heartbeat-error and concurrent-Close
  probes. Replace the old uncertain-reply assertion with actual withdrawal or
  recovery acceptance; record the commit inside the hook before injecting error.
- [x] Extend the existing publication model with stream existence, original
  unacknowledged event and local admission/Close. Reproduce the current failure
  and compare publication-only/rollback-only candidates against the complete
  owner. State expiry/recovery and network assumptions explicitly.
- [x] Check missing-stream and queued-before-delete recovery through the existing
  event-routing tests; include healthy worker/peer preservation, no failed-handler
  invocation and caller cancellation. Verify no Reader polling on abort.
- [x] Obtain independent readiness approval of the chosen owner.
- [x] Implement the owner and delete superseded construction/publication paths.
- [x] Run focused pool race controls and pinned real Redis script/routing checks
  if Redis operations change; no broad pool suite or generator matrix.
- [x] Freeze the exact final diff for independent review and atomic delivery.

Distinct owners remain out of scope: AddNode resource ledger #687, subordinate
active-reader/sink shutdown #800, scheduler desired-job ledger #737, and already
delivered ticker lifecycle #738/#707. Healthy admitted workers retain existing
shutdown and lease-fencing semantics.

Readiness evidence: current race regressions fail at all three expected seams
(final source: 6.374s). The Close test waits for the actual published closing
fence before releasing construction, rather than relying on a scheduling delay.
The routing/reclamation, unstarted Reader, missing-stream/ack and healthy dispatch
controls pass under race in 5.535s (miniredis) and 5.464s (pinned Redis 8.10.2).
The model checks 940 distinct owned states; all four rejected alternatives fail
their designated invariants. Scoped lint reports zero issues. No production edits or broad suite runs.

Implementation acceptance adds targeted fault points for stream initialization
after commit, membership/heartbeat publication, replica-wait cancellation and
cleanup failure. Preserve a healthy peer and its maps/stream. Insert a queued
event before an uncertain publication reply where the prepared stream exists;
confirm that the failed handler never runs and the original event remains
recoverable. Inspect successful admission and Close arbitration, and keep the
existing healthy lease/ownership controls. These refine the same owner, not
separate tickets, models or transport patches.

## Final acceptance receipt

`Node.AddWorker` registers construction under `node.lock`; `prepareWorker`
returns complete or partial private resources to that owner. All failure exits
use `abortAdmission`. Successful transfer installs the worker and subscribes/
starts all five loops under the node lock. `Node.close` joins attempts outside
that lock before releasing their borrowed resources. The previous `newWorker`
publication/start path is deleted, along with its unused worker-local stream
cache. The published creation time and returned Worker now use the same value.

The expanded parent fault tests fail in 5.485s, using an explicit Go test overlay
of the three exact parent production files and an empty replacement for the new
owner file. No CLI or subprocess generation is involved. Current tests cover
rejected and committed stream/membership writes, canceled publication waits,
an expired initial lease, all cleanup failures, concurrent Close, and an event
routed before a failed publication reply. The discarded copy never invokes the
failed handler; the original is reclaimed and handled by a healthy worker.

Final focused race gate passes in 20.490s, including existing acknowledgement,
dispatch, ownership publication, lease/fencing and shutdown controls. The native
Redis 8.10.2 admission/routing matrix passes in 18.192s. The final additional
uncertain-cleanup case passes in 2.101s (miniredis) and 1.841s (native): rejected
cleanup leaves the initial lease/stream present, but never renews or starts a
handler. Tests model ambiguous replies with after-commit hooks and do not claim
network-outage delivery beyond existing retention limits. The accepted model is
unchanged; its bounded results remain applicable. No generator/fixture output
or full-corpus validation applies to this runtime-only change.

Root independently reran the construction/admission/ack/dispatch race matrix on
Redis 8.10.2: 18.298s, passed. The 15 non-skill files remained byte-identical at
integration; the two skill files retained newer scope guidance. Exact integrated
patch SHA-256 `6643fbf3f2a9aa9b67b878d5c8e581b0d264e8a00302a0806606e9b912813330`.
Reviews: `/tmp/loom-704-root-final-verdict.json` and
`/tmp/loom-704-independent-integration-verdict.json`.
