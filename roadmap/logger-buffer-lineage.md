# Shared logger buffer lineage — #745

Parent: `7b59759c5`. Status: implemented, final independent review pending. #744 configuration snapshots and
#746 fresh request scope are delivered. User accepted retaining each buffered entry's original destination and format.
Configuration updates affect later entries only. Independent readiness review
precedes implementation.

1. Reproduce lost child logs when a parent flushes and duplicated ancestral logs
   when several descendants flush. Cover sibling/parent directions, per-entry
   field values, ordering and entries logged concurrently after the first flush.
2. Cover normal HTTP/unary/streaming request plus Endpoint field derivation;
   Error on the request context must flush entries created inside the endpoint.
   Preserve fresh request isolation and independent configuration from744/746.
3. Model current copied buffers versus one shared synchronized scope buffer.
   Check every admitted log is flushed, at most once, and later logs write after
   the first scope flush. Keep callback/writer behavior outside the finite model.
4. Independently review the tests/model/shared owner proposal. One buffer object
   owns entries, flush state and output serialization across With descendants.
   Logger objects retain configuration/fields; fresh requestContext allocates
   a new buffer while With retains its parent's buffer. All paths must share
   this owner: Error, Debug, FlushAndDisableBuffering and buffering callbacks.
5. After the output policy is settled, add its direct regression and model facts,
   implement at the shared logger/buffer boundary, and delete copied-entry paths.
   Keep logger-to-buffer lock order consistent; never acquire a logger lock from
   the buffer owner. Entry fields are captured at logging time, as already done.
6. Run direct and full affected log/debug race suites, scoped lint and bounded
   models. Update operational/model/owner guidance. No generator changes.
7. Freeze exact diff for independent review; root integrates/pushes one atomic
   commit, closes745 and compacts temporary artifacts. Redaction/formatting and
   unrelated logging adapters remain separate tickets.

Configuration choice matters only for entries already queued: root Info then
child WithOutput(B) then child Error must emit the root entry either to its
original A or to B. A later Context(root, WithOutput(B)) likewise needs a stated
rule for earlier entries. Keeping immutable entry output snapshots composes with
744's isolated settings and avoids the flushing context selecting another
logger's destination. Application writers/callbacks remain application-owned.

Readiness evidence: parent ownership/endpoint regressions fail under race; current
model fails CompleteAfterFlush (8 states), shared model passes (84 generated,
33 distinct). Output policy and every flush trigger have direct parent regressions.

Readiness independently approved after moving test methods and adding a writer
that detects overlapping calls without serializing them. Parent overlap control
fails; candidate must serialize all 1000 calls. Implementation shares logBuffer
and captures immutable output slices. The prior derived-buffering test now checks
configuration isolation through a fresh request scope, because the accepted
shared buffer means a child flush intentionally disables parent buffering.
No callback reentrancy or writer-error-reporting contract changes are claimed.

Final scoped validation: full log/debug race suites pass (1.504s/1.546s).
Scoped golangci-lint passes after removing an unused test assignment. No production
change after the race run. Independent readiness checker confirmed lineage
current8/8 failure, shared84/33 pass, output-current failure and snapshot5/4 pass.
No generator output changes. A later configuration update cannot reroute queued
entries; callback configuration still isolates parent/child while shared flush
state intentionally does not. Temporary results remain until review/delivery.
