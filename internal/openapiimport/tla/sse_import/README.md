# SSE item-schema admission and event correspondence

These finite design models separate importer admission from exporter event
selection. They do not implement parsing, schema equivalence, sampling or Go
callbacks. Proof tools are explicit opt-in design aids, never contributor gates.

## Import admission and derived snapshots: SSEImport

A fixed authored `Source` is separate from mutable `loaded`. `Admit` computes its
decision from loaded dialect, envelope, content, occurrence and reserved-extension
facts; `Allowed` is a separately listed expected support set. This is the useful
admission evidence. The first-payload, permission-leak and stale-reserved mutation
operators each remove one admission condition and violate AdmissionSound.

There are 21 scenario labels but 19 distinct Source records. `equalOwnerValues`
is an alias of `twoEqual`; `nestedContent` is an alias of `declarationContent`.
Content keywords are one abstract flag, not a modeled distinction between nested
and declaration positions. Equal root/child values with distinct ownership occur
throughout. `order` changes visited positions but not any output choice in this
model; order independence is by construction and reach-two-reverse establishes
reachability only. The separate event model below has an order-sensitive mutant.

`ExpectedPlan`, `ExpectedOutcome` and `ExpectedEdges` are assumed extraction,
projection and normalization definitions. Plan and ordinary are assigned from
those definitions, not extracted from arbitrary loaded schemas. ContractRetained
is consequently definitional and is not independent evidence. Selector/header,
framing, payload/default/assertion tokens are assumed, not proved algorithms.
The root label `occurrence1` denotes an assumed common imported representative;
it does not choose the exporter's response/event owner. That decision is modeled
separately below. Do not implement response-first sampling from this token.

The candidate exhausts 306 states. AnnotationAuthority detects derive-step
deviations from the assumed ordinary result: wrong owner, independent child value,
absent-outcome fallback, and double symbolic custom projection. TypedClosure
only detects an injected literal edge relative to assumed ExpectedEdges.
SourceUnchanged only detects a write to loaded. The model never feeds that
mutated loaded graph into ordinary projection, so these checks do not prove a
parser, immutable snapshot implementation or source-to-plan extraction.

Nine mutation operators have ten runs: the same absent-outcome fallback operator
is exercised with suppression and filter-all. Those are separate scenarios, not
two distinct algorithms. Custom behavior is a finite value transformation;
ordered example lists, callback invocation counts, actual attr/raw-value inputs,
opaque copies, schema validity and real codecs remain implementation obligations.

Current mode uses exact 201, records media-item-schema, drops item/payload edges
and rejects without unary runtime output. This reflects current strict and partial
import (the diagnostic is not lossy). Current preservation/rejection exhausts 14
states; admitted-output predicates there are vacuous. RequiredSupport fails at
done and CurrentRetention fails after normalization, demonstrating the intended
capability gap rather than falsely successful unary behavior.

The fixed typed closure and recursive cut reuse the assumptions of neighboring
source_contracts/SchemaRoles and schema_references/ReferenceOccurrences. No claim
covers arbitrary cycles, loading, pruning, initializers or arbitrary JSON Schema
equivalence. `extensionMatches` is an assumed complete comparison result. Every
candidate path reaches done in the finite graph, and selected reach predicates
produce witnesses, but there is no fairness/eventual-callback liveness theorem.

## Per-response eligibility and event documentation: EventCorrespondence

The companion has 22 labels: thirteen single exact200 result-derived scenarios
and nine typed-handshake scenarios with exact200/201, each in both visit orders.
Legacy body comparison controls have one response, matching the DSL. The
multi-success current-algorithm, sampling, wrong-capture and reverse-order controls
use typed handshakes. ModelDomain checks this finite shape; it does not validate
arbitrary DSL. The 173 states are 13 single-response paths of five states plus
9 handshake cases × 2 orders × six states. Each path reaches done.

Endpoint facts contain mixed/mapping/projection and separate result/event capture
tokens. ResponseFacts is an indexed record of actual per-success role, docs body,
body presence, selection, exclusions and comparison outcome. Choose quantifies
over every response. `secondDocs` has an independent documentation body ONLY on
201; 200 is clean. Both traversal orders are explored. The first-status-only
mutant reads the first visited record: it wrongly admits in forward order and is
caught by BranchSound. In reverse order it sees 201's docs and rejects, and the
positive reverse control exhausts with all invariants. Both correct-order reaches
show rejection with 200's event payload and 201's documentation override retained.
No endpoint aggregate is substituted for that per-response check.

`mixed` means the exact existing MethodExpr.HasMixedResults predicate:
Result != nil && StreamingResult != nil && Result != StreamingResult. It is an
identity predicate already consumed by attachSSETargets, not a structural or
semantic type-equivalence algorithm. There is no separate Bridge operator or
independent bridge theorem. Separately captured ResultValue/StreamingResultValue
remain different handles; the equal-capture mutant wrongly rejects the otherwise
eligible handshake. No capture coalescing is proposed.

Allowed is an independent expected eligibility list; BranchSound and
BranchComplete compare the computed decision with it. ResponseWhole facts are assumed
extracted input. Equivalent explicit Body, local examples and primitive bodies
have named reaches, but none proves the concrete callback-free comparator,
pre-finalization authored-source retention or projection extraction. Primitive
absence of projection is not a disqualifier; an Empty binding-lowered body is.

Actual event preparation and async eligibility are separate decisions. Every
handshake prepares one ResponseValue-owned event, including mapped/projected,
mixed and independently documented cases. Every implicit handshake payload uses
that completed event outcome. Its explicit docs response uses a distinct
DocsOutcome instead, preserving override precedence. Async derivation uses the
event only when eligible; otherwise ExistingMessage remains independent.
Single-response ineligible legacy paths retain ExistingResponse/ExistingMessage.
The symbolic mapped/projected outcome assumes the existing wire projection; it
does not erase mappings, replace a selected event with its whole type or prove
the projection algorithm. Distinct projection/declaration positions remain real
owners outside the model; one prepare step is not one universal Go callback.

EventAuthority checks selected capture, completed event and one modeled prepare;
PayloadAuthority independently checks each expected payload/override;
MessageAuthority independently checks the eligible-derived versus existing async
boundary. OldPaths explicitly retains the old non-handshake ineligible outputs.
They are copy/selection properties over assumed outcomes, not a graph or callback
proof. Per-status and first-visited mutations change the chosen output;
reprepare-status increments the preparation count; eligible-only-prepare omits the
mapped handshake's required event. erase-docs loses only the independent override.
Those mutants demonstrate sensitivity to the specified decisions.

All outcomes, including empty and independent async values, have the same record
shape and homogeneous corresponding field shapes. `event-current-payload` and
`event-current-message` run the same source-grounded extrapolation separately:
per-status response sampling violates PayloadAuthority, and a record-shaped
independent async result violates MessageAuthority. Both reach done and evaluate
the named predicate. The isolated async-only mutant keeps correct payloads and
still violates MessageAuthority, so no unexecuted conjunction half or heterogeneous
comparison error is credited as evidence. These controls extrapolate the old
analysis algorithm to newly admitted typed handshakes; they are not executions
of an old DSL that accepted multi-status legacy result bodies.

Other focused mutants cover invalid capture equality, Empty metadata mistaken
for no event, metadata-name collisions, omitted exclusions/comparison/mixed/docs
checks and ineligible async overwrite. The suppressed/custom/recursive results
are finite assumed outcomes; reaches are witnesses, not liveness or general
order theorems. Both models omit canonical/name allocation, arbitrary graphs,
real callback effects and import/export implementation. No new runtime enum,
registry or secondary example authority follows from model state.

## Source correspondence and implementation obligations

- `internal/openapiimport/analyze.go:449–474` rejects itemSchema and reads only
  media.Schema; render.go, diagnostics.go and partial.go preserve rejection.
  `analyze_schema_grammar.go:78–86` reports contentMediaType/contentSchema.
  Relaxation is limited to the envelope/data positions consumed by the SSE owner;
  shared ordinary schemaAdmission remains strict. No global permission cache.
- `expr/method.go` supplies the exact HasMixedResults predicate above and
  EffectiveStreamingResult. `representation/prepare.go:56–91` captures separate
  method occurrences. `representation/value_carriers.go:41–84` binds ResultValue
  and StreamingResultValue and consumes the existing mixed predicate.
- `transportir/request_response.go:131,162–181` records each response's
  IndependentDocumentBody and preserves OpenAPIBody precedence. Eligibility must
  inspect every actual response, not a representative or an aggregate assembled
  from only the first one.
- M3 `http_status_selection.go` requires StatusField for status-selected success
  and a typed handshake for SSE metadata/status bindings. Thus the multi-success
  controls are typed-handshake controls; result-derived comparator examples use
  the admitted single exact200 role. `http_response_bindings.go` captures the
  handshake's independent metadata source before finalization; its Empty body
  is not the event, and matching names do not remove event members.
- `expr/http_body_projection.go:5–18,99–136` records declaration-level Source
  and Excluded, not occurrence-local examples. Use captured Method.Result /
  ResultValue for implicit authored facts. `http_body_types.go:341–349` does not
  record projection for a whole primitive; binding-lowered Empty is ineligible.
- Explicit Body provenance must be retained at the existing pre-finalization
  response-plan boundary: `http_endpoint_validate.go:14` precedes finalization,
  while `http_response.go:426–465` clones/initializes/wraps Body before later
  cloneExplicitHTTPBody records projection. This required retained original
  occurrence/UserExamples fact is not an existing pristine projection API.
- `ir/analyzer_body_types.go:182–235` has status-dependent ordinary response
  contexts and source-bound SSE projection preparation. New typed handshakes
  need one status-independent event outcome, preserving those projection owners.
  Explicit OpenAPIBody retains its separate projection/annotation authority;
  only implicit payload documentation consumes the completed event outcome.
- `ir/schema_examples.go`, component_annotations.go and async_schema_inline.go
  own custom projection, acquired annotations, snapshots and expansion/cuts.
  Copy completed owned outcomes without a second async baseline on the eligible
  branch. Keep all existing owners on ineligible async and independent docs.

Named implementation acceptance obligations (not executed by these models):

1. A typed handshake has an OpenAPIBody on 201 only. In both declaration/visit
   orders derived eligibility must be false; 200 retains event documentation,
   201 retains its actual authored override, and async remains independently
   owned. Include event-document members with the same logical names as handshake
   StatusField/header/cookie members. Source-identity-scoped binding validation
   must admit that independent override while genuine same-carrier collisions
   and ordinary non-handshake checks still reject. Current M3 name-only
   OpenAPIBody status competition still needs correction before exporter work. This is an intended contract and pending
   Go obligation, not a claim that current validation admits every such case.
   Add the mirror with OpenAPIBody on 200 only, in both declaration/visit
   orders: eligibility is false, 200 retains its authored override, 201 retains
   event documentation, and async remains independently owned. First-only and
   last-only implementations must each fail this complete test matrix.
2. Single exact200 with occurrence-local Example/Description and a separate
   inline primitive event must be eligible and survive first/second ordinary
   import-export. Compare actual finalized targets; primitive bindings that
   lower Empty remain ineligible. Equal explicit Body retains local examples and
   canonical claims; differing Body remains ineligible.
3. Mapped typed-handshake200/201 in both orders retains actual mappings/projection
   and one status-independent event payload outcome, even though async stays
   independent. Preserve an explicit-document typed-handshake override as above.
4. Establish actual callback owner/attr/raw-value inputs and counts, suppression,
   source isolation, selected/declaration annotations, nullable/recursive closure,
   removed async-only component inventory and exact rendered first/second
   roundtrips. No sample/name equality, stripped examples or a generic graph
   theorem can replace those Go/parser/Vacuum checks.

## Run explicitly

Use Java Corretto 25.0.5 and TLA+ tools 1.7.4 (TLC 2.19), JAR SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
From this directory, select a configuration below. Use SSEImport.tla for the first table and
EventCorrespondence.tla for event-*.cfg:

```sh
TLA_TOOLS_JAR=/Users/luca/.cache/loom-tools/tla2tools-1.7.4.jar
states_dir=$(mktemp -d /tmp/loom-sse-import-states.XXXXXX)
java -Xmx768m -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC \
  -workers 1 -config candidate.cfg -metadir "$states_dir" SSEImport.tla
# Or use a fresh states directory for the companion:
# java -Xmx768m -XX:+UseParallelGC -cp "$TLA_TOOLS_JAR" tlc2.TLC \
#   -workers 1 -config event-candidate.cfg -metadir "$states_dir" EventCorrespondence.tla
```

Run configurations sequentially. A successful candidate exits 0; invariant
counterexamples and reach witnesses exit 12. Any other exit is an authoring/tool
failure, not semantic evidence. Delete task-owned checker states after independent
review and when no task uses them; retain sources/configurations and concise
findings. No tool installation or execution is added to ordinary contributor gates.

## Executed bounded outcomes

The table records generated/distinct states and queue length with one TLC worker.
The mutation/reach rows name the exact falsified predicate; positive rows exhaust
their state space. Counts describe this finite abstraction only.

| Configuration | Exit | Predicate/result | Generated / distinct / queue |
| --- | ---: | --- | --- |
| candidate.cfg | 0 | Exhausted; configured invariants pass | 306 / 306 / 0 |
| current-preservation.cfg | 0 | Exhausted; configured invariants pass | 14 / 14 / 0 |
| current-required.cfg | 12 | RequiredSupport | 13 / 13 / 1 |
| current-loss.cfg | 12 | CurrentRetention | 3 / 3 / 1 |
| first-payload.cfg | 12 | AdmissionSound | 15 / 15 / 1 |
| permission-leak.cfg | 12 | AdmissionSound | 13 / 13 / 1 |
| stale-reserved.cfg | 12 | AdmissionSound | 13 / 13 / 1 |
| wrong-owner.cfg | 12 | AnnotationAuthority | 15 / 15 / 1 |
| independent-example.cfg | 12 | AnnotationAuthority | 15 / 15 / 1 |
| suppression-fallback.cfg | 12 | AnnotationAuthority | 13 / 13 / 1 |
| double-custom.cfg | 12 | AnnotationAuthority | 13 / 13 / 1 |
| filter-fallback.cfg | 12 | AnnotationAuthority | 13 / 13 / 1 |
| source-mutation.cfg | 12 | SourceUnchanged | 11 / 11 / 1 |
| literal-edge.cfg | 12 | TypedClosure | 13 / 13 / 1 |
| reach-done.cfg | 12 | NoDone | 13 / 13 / 1 |
| reach-exact201.cfg | 12 | NoAdmitted | 13 / 13 / 1 |
| reach-class.cfg | 12 | NoAdmitted | 13 / 13 / 1 |
| reach-two-reverse.cfg | 12 | NoTwoReverse | 16 / 16 / 1 |
| reach-suppressed.cfg | 12 | NoSuppressed | 13 / 13 / 1 |
| reach-custom.cfg | 12 | NoCustom | 13 / 13 / 1 |
| reach-recursive.cfg | 12 | NoRecursive | 13 / 13 / 1 |
| reach-rejected.cfg | 12 | NoRejected | 13 / 13 / 1 |

The refined companion's candidate exhausts 173 states with seven configured
invariants. Thirty-two invocations below comprise two current-algorithm predicate
checks, sixteen unsafe operators (the first-status operator also has a positive
reverse-order control), the candidate, and twelve reach witnesses. Thus eighteen
negative safety checks, two positive exhaustive checks and twelve reaches ran.
All use one worker and a 60-second guard; none timed out or failed parsing/types.
The original 22 results above are reused unchanged, not rerun for this refinement.

| Configuration | Exit | Predicate/result | Generated / distinct / queue |
| --- | ---: | --- | --- |
| event-current-payload.cfg | 12 | PayloadAuthority | 11 / 11 / 1 |
| event-current-message.cfg | 12 | MessageAuthority | 11 / 11 / 1 |
| event-candidate.cfg | 0 | Exhausted; configured invariants pass | 173 / 173 / 0 |
| event-equal-capture.cfg | 12 | BranchComplete | 11 / 11 / 1 |
| event-empty-handshake.cfg | 12 | BranchComplete | 11 / 11 / 1 |
| event-name-collision.cfg | 12 | BranchComplete | 11 / 11 / 1 |
| event-ignore-exclusions.cfg | 12 | BranchSound | 5 / 5 / 0 |
| event-ignore-body-contract.cfg | 12 | BranchSound | 5 / 5 / 0 |
| event-ignore-mixed.cfg | 12 | BranchSound | 5 / 5 / 0 |
| event-ignore-docs.cfg | 12 | BranchSound | 11 / 11 / 1 |
| event-wrong-capture.cfg | 12 | EventAuthority | 11 / 11 / 1 |
| event-per-status.cfg | 12 | PayloadAuthority | 11 / 11 / 1 |
| event-first-visited.cfg | 12 | EventAuthority | 11 / 11 / 1 |
| event-overwrite-ineligible.cfg | 12 | OldPaths | 5 / 5 / 0 |
| event-async-only.cfg | 12 | MessageAuthority | 11 / 11 / 1 |
| event-eligible-only-prepare.cfg | 12 | EventAuthority | 11 / 11 / 1 |
| event-erase-docs.cfg | 12 | PayloadAuthority | 11 / 11 / 1 |
| event-reprepare-status.cfg | 12 | EventAuthority | 11 / 11 / 1 |
| event-first-status-forward.cfg | 12 | BranchSound | 6 / 6 / 0 |
| event-first-status-reverse.cfg | 0 | Exhausted; configured invariants pass | 6 / 6 / 0 |
| event-reach-eligible.cfg | 12 | NoEligible | 5 / 5 / 0 |
| event-reach-local-example.cfg | 12 | NoEligible | 5 / 5 / 0 |
| event-reach-primitive.cfg | 12 | NoEligible | 5 / 5 / 0 |
| event-reach-handshake.cfg | 12 | NoHandshake | 11 / 11 / 1 |
| event-reach-mapped.cfg | 12 | NoMapped | 11 / 11 / 1 |
| event-reach-mixed.cfg | 12 | NoMixed | 5 / 5 / 0 |
| event-reach-docs.cfg | 12 | NoDocs | 5 / 5 / 0 |
| event-reach-reverse.cfg | 12 | NoReverse | 12 / 12 / 1 |
| event-reach-suppressed.cfg | 12 | NoSuppressed | 11 / 11 / 1 |
| event-reach-recursive.cfg | 12 | NoRecursive | 11 / 11 / 1 |
| event-reach-second-docs-forward.cfg | 12 | NoSecondDocs | 6 / 6 / 0 |
| event-reach-second-docs-reverse.cfg | 12 | NoSecondDocs | 6 / 6 / 0 |
