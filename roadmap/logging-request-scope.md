# Request logger scope — #746

Parent: `09aa506b4`. Status: implementation complete; final checks and independent review pending. #744 detached configuration is delivered. #745 field-derivation buffer
lineage stays separate.

1. Reproduce HTTP, unary gRPC and streaming gRPC cross-request buffer reuse with
   IDs disabled, both with request logging enabled and disabled. Retain enabled-ID
   controls and show a buffered/flushed root cannot transfer its entries or flush
   state into a fresh request. Preserve root fields and request context values.
2. Model root-sharing, cloning the existing buffer, and fresh request scope.
   Check request buffer ownership and scope-local flush state under both ID modes.
   Independently review the tests, counterexamples and shared owner proposal.
3. Implement one private request-context constructor in `clue/log/context.go`.
   Snapshot immutable options and detach inherited field storage under the root
   lock. Start an empty, unflushed request buffer; evaluate configured buffering
   policy against the incoming request context. HTTP and both server interceptors
   call this owner with optional request/call ID metadata supplied as initial fields.
   Preserve filtered HTTP bypass and cancellation/deadline/value propagation.
4. Run all affected log/debug race suites, focused transport controls, scoped
   lint and model checks. No generator code changes; no generated corpus gate.
5. Update public and owner/model guidance, freeze exact diff for nonimplementer
   review, integrate, push one atomic commit, close746 and compact temporary
   artifacts. Then model #745 shared buffer lineage separately.

Request lifetime must not depend on ID metadata or be implemented by calling
`With` with no fields: With currently copies buffered entries and flush state,
while #745 must eventually share a buffer within one request. Request scope owns
fresh lifetime, field derivation owns fields. No new public API or dependency is
needed. Root output writers/callbacks remain application-owned as documented.

## Readiness receipt

All three transports reproduce foreign request/root logs or inherited flush state
on parent09aa506b4. The matrix has36 scope cases:6 enabled-ID/empty-root controls
pass; the other30 fail. Buffer-policy/context controls retain3 passing debug cases
and expose3 request-policy failures withoutIDs. Full selected run0.594s. Current
and clone models fail intended invariants at7/7states; fresh completes150/96.
Inputs and assumptions are durable beside the code. Temporary logs are
`/tmp/loom-746-scope-red.log` and `/tmp/loom-746-model-{current,clone,fresh}.log`.
No production edits yet. Proposed helper snapshots options/fields only, evaluates
configured buffering against request context, then transports add optionalIDs.

Independent readiness review approved all seven files without findings and
confirmed parent failures and the fresh150/96model. Implementation uses one
private requestContext owner. Transports supply optionalIDs as initial fields,
so configured buffering callbacks run once at request scope creation; With is
not used to create the new lifetime. Root configuration/fields are snapshotted
under its lock; callbacks run against the request context after releasing it.

Final full log/debug race suites pass1.540s/1.646s; scoped lint zero issues.
No generation artifacts change. Current/shared and clone/copy alternatives retain
expected model failures; fresh150/96result is unchanged. Final independent review
must include incoming context/policy, all three call sites and unchanged With.
