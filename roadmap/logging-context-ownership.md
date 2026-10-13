# Logging context ownership repairs

Status: #744 implemented with focused gates passing; awaiting final independent review.
Parent: `27255d025`. Worktree: `logging-context-ownership/loom`.

## Owner and sequence

The logger currently combines configuration, derived fields and buffered output
in one struct. `With` shares the mutable options pointer but copies the buffered
entries. Middleware creates a child only when an ID is enabled. Three reported
failures follow different lifetime edges of this representation; preserve atomic
issue commits while establishing explicit owners in sequence.

1. [x] #744: reproduce shared configuration mutation deterministically and under
   race detection. Establish immutable option snapshots per logger, with copy on
   configuration update. Clone slice storage and ensure reusable option closures
   cannot mutate shared slices. Preserve `Context`'s existing initialization of
   its logger, buffer state and fields; do not silently create a second buffer.
2. [x] Model current shared options versus detached snapshots before production;
   independently review the proposal and bounded HTTP/unary/streaming controls.
3. [ ] Implement/review/deliver #744 as one atomic commit. Do not claim buffer
   isolation or request lifetime defects resolved by configuration snapshots.
4. [ ] #746: reproduce root-buffer reuse with IDs disabled. Establish request
   scope independently of request ID metadata in HTTP and both gRPC interceptors.
   Model request isolation; preserve root configuration/fields but do not inherit
   prior buffered request entries. Separate fresh request scope from field
   derivation, so the later buffer fix does not regress request isolation.
5. [ ] #745: reproduce copied-buffer duplicate/lost flushes in both parent/child
   directions. Introduce a shared synchronized buffer for one scope's derived
   contexts, retaining per-entry field snapshots and once-only flush semantics.
   Establish output/option-change behavior with tests and a bounded model before
   implementation. Keep unrelated formatting/redaction fixes separate.
6. [ ] For each atomic repair, update public operational guidance and owning
   framework guidance; run direct and affected package race checks and relevant
   HTTP/gRPC consumers. Obtain independent exact-diff review, integrate, push,
   close only its proven issue, and retain compact evidence before cleanup.

## Scope boundaries

This is framework logging, not an application logging backend replacement.
No new dependency, logging service or compatibility switch is proposed. Existing
output writers/format callbacks remain application-owned. The scalar/slice-value
formatter bug #750 and secret-name/URL logging reports #718/#721 remain separate
contracts. Generated code does not change for the runtime ownership repair;
use direct middleware/interceptor tests, not the exported-design corpus.

The first repair must not fix a race by serializing all logging globally or by
creating per-request copies only inside Debug middleware. The shared configuration
owner is `clue/log.Context` and the option constructors; all consumers inherit it.
A shallow options copy is insufficient because outputs and callback slices can
share backing storage. Public configuration API semantics beyond these existing
contracts require explicit assessment, not incidental changes hidden in a race fix.

## #744 readiness evidence and acceptance

Parent regressions: five option-isolation cases and reusable/caller output-slice
cases fail deterministically; four concurrent child configurations produce Go
race-detector failures. Mounted HTTP, unary gRPC and streaming gRPC retain the
first request's context and show a later request changing it. IDs remain enabled
here; request scope without IDs belongs to #746. Existing With, no-op, debug
toggle and disabled-buffer controls pass. Model results: current 2/2 failure,
shallow 4/4 failure, detached 13/9 complete. Raw logs are temporary under
`/tmp/loom-744-*`; durable assumptions and instructions accompany the model.

Implementation acceptance: configure a detached copy under the existing logger
lock, including owned outputs and callback slices, then publish it. Reusable
WithOutputs closures copy their supplied slice on each application. Retain the
same logger identity, fields, buffered entries and flush behavior. Writers,
format callbacks and application values are referenced, not deep-copied.
Run direct new regressions, full `clue/log` and `clue/debug` race tests, scoped
lint and whitespace checks. No generator output changes, so generation and the
exported design corpus are not acceptance gates. Update public operational and
framework/model guidance before exact final-diff review.

Readiness independently approved after a low declaration-order fix. The reviewer
also requested explicit spare-capacity callback and concurrent reusable-output
controls; both reproduced parent failures, including a real closure data race.
The implementation detaches options in Context and outputs on each WithOutputs
application; all new and existing log/debug tests pass under race detection
(1.577s/1.674s). Context's buffered-entry identity control remains green. No
transport-local copying or buffer lifecycle change was introduced.
