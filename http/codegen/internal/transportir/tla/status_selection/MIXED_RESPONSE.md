# Mixed response metadata ownership

`MixedResponse.tla` is a bounded companion to unchanged `SSEOpen.tla` and
`SSETerminal.tla`. Root chose one shared metadata owner for mixed unary/SSE
success, with whole unary Result and separate StreamingResult event bodies.
The owning [response-selection design](../../response_selection_design.md) records the
explicit unary client pair (Result, typed metadata, error), stream Response(),
alternatives and regeneration.
Those Go surfaces and actual schema/body extraction are implementation obligations,
not modeled values.

## Owner and assumptions

The shared runtime response-metadata admission owner holds one lock, prepared
snapshot and sealed flag for both unary capture and SSE opening. The unary
adapter holds that same owner, not a separate publication state machine. `Begin`
models atomic capture/seal; `held` is its lock-protected interval through abstract
publication, and ordinary setters cannot enter it. `Publish` consumes the frozen
snapshot. A late setter returns committed. An invalid setter retains the previous
metadata and returns invalid. All discard operations (Send, Close and exposed
open/comment/completion forms) are modeled by `DiscardSend`: none publishes or
seals. Actual SSE opening operations are folded into Begin. Thus a setter after
discarding Send/Close can succeed in unary but rejects after SSE commit; that
asymmetry is explicitly accepted.

The Go owner must also seal at every actual SSE terminal admission, including
pre-open failure, suppression and Close, and on unary failed capture or handler
error. SSE uses the writer's same admission lock for metadata and terminal state;
there is no check-then-act gap or independent discard lock. Existing preparation
callbacks retain their non-reentry contract; reentrant observation is dispatched
after unlocking. These terminal details remain obligations of SSETerminal and
runtime correspondence tests, not new transitions claimed by this companion.

`required` means a metadata value is required (for status mode or required
metadata), independently of the validation abstraction. Numeric statuses are
representative owned tokens, not a literal-singleton/multistatus DSL encoding.
`Valid` abstracts captured domain plus native/mapped predicates: the `header`
Boolean denotes the remaining validation conjunction. There is a valid-owned
status with that conjunction false, as well as an invalid status. The implicit
value is valid only when absence is allowed; no status-selection equation or
integer conversion algorithm is added.

Representation is fixed for the exchange. Validation and policy callbacks return;
snapshots are immutable. The model assumes one invocation action and does not
prove invocation counts in generated handlers. Body strings represent roles,
not schemas or decoded data. I/O failures, callback reentry, cancellation,
suppression, terminal races and client body cleanup retain the separate runtime
models/tests; this model does not re-prove them. The finite step bound establishes
no unbounded liveness property.

## Checks and counterexamples

The candidate checks type domains, single publication, validated preparation,
body separation, snapshot immutability and publication provenance. `SingleInvocation`
and `SnapshotAgreement` are structural sanity checks by construction, not independent
evidence for the implementation. `PhysicalTrigger` reads a ghost reason tag; it
checks the chosen model transition, not an observed network write.

`current-split` models the current unary/handshake body-role confusion. Its
counterexample is not evidence of Go compilation. `discard-commit` now keeps the
correct unary body and valid snapshot but publishes on the discard operation;
its intended failure is PhysicalTrigger under the whole candidate invariant set.
The predecessor mutant also changed body role and was non-minimal; narrowing its
checked invariant set did not isolate the cause, and those receipts remain only
historical evidence. `missing-check`, `late-set`, and `held-set` exercise distinct
admission omissions. Held-set deliberately admits replacement during capture,
which must fail MetadataFrozen after publication.

Reachability checks intentionally negate a desired state. In addition to implicit
unary/SSE success, the typed-unary-after-discard check runs all candidate invariants
and witnesses required, supplied metadata published by unary after a discarded
operation. SSE success does **not** distinguish first-event from zero-event opening;
actual zero-event evidence belongs to unchanged SSEOpen and generated tests. Other
controls witness replacement, invalid setters and missing required preparation.

## Explicit run commands

Use TLA+ 1.7.4, jar SHA256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Create a private scratch directory outside this source and run, for example:

```sh
scratch=$(mktemp -d /tmp/loom-mixed-response.XXXXXX)
java -Xmx1536m -cp /path/to/tla2tools-1.7.4.jar tlc2.TLC -workers 1 \
  -config mixed-checked.cfg -metadir "$scratch" MixedResponse.tla
```

Use each other `mixed-*.cfg` in the same invocation with its own scratch directory.
Candidate exit0 is expected. Each negative/reachability configuration must exit12
with its named invariant; a parse error or a different failed invariant is not
credited. Deadlock checking is disabled because MaxSteps intentionally bounds
execution. Remove task checker states only after review and active-use checks;
retain sources/configurations and summarized receipts. No proof execution belongs
in default build/test/CI or release eligibility.

## Source correspondence still owed

- Both negotiated representations invoke the service once and share the exact
  metadata owner, including concurrent capture/late setter rejection.
- Send→SetResponse and Close→SetResponse in both modes establish the accepted
  nonsealing discard versus committed SSE asymmetry.
- Unary returns whole Result: same-named metadata fields do not subtract members;
  primitive/collection bodies are not suppressed. Metadata never fills Result.
- Generated unary client returns Result and the existing native HandshakeRef
  separately, validates both before returning either, and errors return both zero
  values (nil metadata for object carriers). Every admitted endpoint uses the
  shared carrier/client/interceptor return owner; affected CLI adapters preserve
  it. Typed mixed currently admits HTTP only: existing admission rejects a typed
  JSON-RPC handshake, dual HTTP/JSON-RPC streaming and gRPC mixed results.
  Preserve standalone untyped JSON-RPC mixed methods and a shared service with
  distinct HTTP typed-mixed and RPC methods; compile/runtime and existing
  both-order rejection controls establish the boundary, not a new rejection.
- After pre-open failure, suppression or Close in SSE, or failed capture/handler
  error in unary, SetResponse rejects with ErrSSEResponseCommitted. Race terminal
  admission against a setter and check one atomic winner. Discard Send/Close
  remain nonterminal and nonsealing in unary mode.
- Missing/invalid/late metadata, replacement, actual zero-event SSE, cookies/policy
  callbacks and native constraints are checked before physical publication.
- M8 documents unary and event media separately with the same metadata, honoring
  independent authored documentation. No abstract body token proves graph equality.

Final successor result counts and receipt pins are recorded below after execution.

## Successor execution results

All twelve runs used the exact model/configuration postimages, one worker,
MaxSteps=6 and pinned tool above; all source guards passed. Candidate completed
1,408 generated /609 distinct states, queue0, in3.317s. Each other run failed only
the listed intended invariant. Counts below are bounded evidence, not independent
proofs of properties absent from the transition system. This results paragraph
was appended after execution; receipts pin the prior explanatory README bytes.

| Configuration | Result | Generated / distinct |
|---|---|---|
| mixed-checked | PASS | 1408 / 609 |
| mixed-current-split | BodySeparation violation | 14 / 13 |
| mixed-discard-commit | PhysicalTrigger violation | 13 / 12 |
| mixed-missing-check | PreparedValid violation | 80 / 58 |
| mixed-late-set | MetadataFrozen violation | 181 / 114 |
| mixed-held-set | MetadataFrozen violation | 209 / 126 |
| mixed-reach-unary | NoUnarySuccess violation | 55 / 42 |
| mixed-reach-sse | NoSSESuccess violation | 95 / 68 |
| mixed-reach-unary-typed | NoUnaryTypedAfterDiscard violation | 540 / 277 |
| mixed-reach-replacement | NoReplacement violation | 32 / 28 |
| mixed-reach-invalid-set | NoInvalidSet violation | 11 / 11 |
| mixed-reach-missing | NoMissingFailure violation | 20 / 18 |

The typed unary witness is Invoke→SetResponse(one)→DiscardSend→Begin→Publish
with required=TRUE and published token one. The held-set mutant is
Invoke→Begin(implicit)→SetResponse(one)→Publish, violating MetadataFrozen.
No SSEOpen/SSETerminal source or configuration was changed or rechecked.

## Documentation-only terminal/transport clarification

R3 changes no TLA or configuration bytes and runs no checker. The twelve R2
receipts above remain bound to their original inputs, including their earlier
prose. This successor adds implementation obligations for terminal closure,
actual transport admission and HandshakeRef without claiming those behaviors
are proved by MixedResponse. See the owning
[design](../../response_selection_design.md) for the full client contract.

## Implicit preparation correspondence

The implicit token above abstracts allowed metadata absence after preparation;
it is not a supplied zero-valued native object. In generated code,
`http/codegen/sse.go:implicitSSEHandshake` uses the existing native default owner
to materialize authored defaults while leaving other optional members absent.
An explicit `SetResponse` value remains supplied, including scalar zero and empty
strings; it never takes that implicit default path. Invalid custom default
parsing returns before cookie-policy callbacks or physical publication.
`TestImplicitHandshakeWire` and `TestImplicitHandshakeClient` in the maintained
generated handshake module cover implicit defaults, explicit zero values,
custom representations, rejected parsing and policy refusal for SSE and unary.

`http/sse_writer.go:initHeaders` prepares implicit metadata with the retained
request context, matching explicit setter/unary preparation. The first
`SendWithContext` argument does not choose the cookie-policy authority.
`TestSSEHandshakeResolvesWithRequestContext` and the same generated module check
this correspondence. The model abstracts prepared values and policy callbacks,
so these fixes discharge implementation assumptions without changing model
transitions, selector equations or the recorded checker results. Dynamic unary
view headers belong to the whole Result contract;
`TestMixedDynamicView` checks that independently of handshake metadata.
