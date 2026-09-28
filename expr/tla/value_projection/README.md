# Value selection, phase and cache ownership

This is the bounded pipeline model for [#568](https://github.com/CaliLuke/loom/issues/568),
part of milestone 1 in [the value contract plan](../../../roadmap/value-contract-plan.md).
It checks the proposed ownership and ordering rules before migrating production
consumers. It is not a proof of the Go implementation or of arbitrary values.
The separate Lean work owns the unbounded structural semantic obligations.

## Run

Use Java and the TLA+ tools JAR. No generated service dependency is added.
The recorded runs used Corretto Java 25.0.4.1 and TLC 2.19, revision `5a47802`
(8 August 2024). The existing local JAR was `/tmp/loom-tla2tools.jar`, SHA-256
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Obtain `tla2tools.jar` from the [official TLA+ releases](https://github.com/tlaplus/tlaplus/releases)
if it is not installed, and check the version printed by TLC. Set the variable
to the actual local path; it is not a repository-managed runtime dependency.

From this directory:

```sh
export TLA_TOOLS_JAR=/absolute/path/to/tla2tools.jar
java -XX:+UseParallelGC -Xmx2g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-value-contract-tlc/legacy -config legacy.cfg ValueProjection.tla
java -XX:+UseParallelGC -Xmx2g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-value-contract-tlc/reselection -config reselection.cfg ValueProjection.tla
java -XX:+UseParallelGC -Xmx2g -cp "$TLA_TOOLS_JAR" tlc2.TLC -workers 1 \
  -metadir /tmp/loom-value-contract-tlc/checked -config checked.cfg ValueProjection.tla
```

The first two commands must exit 12 with the invariant named below. The third
must exit 0 with no invariant, temporal, evaluation or parse error. Repeat the
same command for every negative-control configuration in the table, using a
separate `-metadir`. Those must also exit 12 for their named invariant. A tool
failure is not a successful counterexample. Keep logs and TLC state directories
outside the checkout. The checked configuration enables deadlock detection and
weak-fairness termination checking; no state constraint truncates exploration.

## Model and bounds

There are two consumer jobs and ten scenario classes. Both jobs can advance in
either order, or interleave any of their five processing steps. A job first
checks effective target eligibility, selects a source, resolves it, projects a
wire snapshot, validates target representability, and emits or records an
explicit outcome. An excluded/suppressed example and an absent contract value
stop before selection. The two jobs share one immutable-record cache.

The finite domains are:

- Two effective occurrence identities, representing copies that may share a
  nominal leaf/type identity but have different semantic constraints.
- Local authored, inherited authored, absent and synthetic source identities.
  Local wins over inherited when both exist. The inherited tier abstracts the
  existing ordered reference/base/type traversal; it does not change that order.
- Example, enum and default roles. Only a missing example permits synthesis.
- Four two-level union paths: `a/x`, `a/y`, `b/x`, `b/y`. Every path has the
  same abstract leaf shape, so shape cannot recover its selected branch.
- An independent field with three presence states: absent, null and present.
  This field is separate from the unions; a null field does not erase their
  branch identity. The model does not globally equate null and absence.
- Full and selected-body observation plans. The body plan legitimately removes
  the inner union field and the independent field. Its observation retains the
  outer branch and reports the removed field as absent.
- Raw, resolved and wire cache representations. Semantic keys use occurrence,
  source and role; wire keys also use the target plan. Raw and resolved entries
  have no target because inherited consumers intentionally share resolution.

The ten scenarios exercise ordinary reuse, distinct occurrences, distinct
sources, distinct roles, distinct targets, synthesis, missing enum/default,
effective example suppression, target exclusion, and unrepresentability.
The suppression and unrepresentability scenarios pair an example job with
an enum/default job, so documentation policy cannot suppress contract checking.
The `suppressed` input means the *result* of current visibility/suppression rules;
it does not model metadata parsing or override authored-example exceptions.

An initial-state condition makes the branch/presence inputs identical when the
complete semantic owner is identical. This states that an immutable selected
source resolved against the same effective occurrence and role has one meaning.
It does not merge different sources, method copies, or roles, and does not
assume any cache access is correct. Each omitted key dimension has its own
failing configuration. The condition still allows independent choices whenever
an owner differs. There are 1,560 initial states in the checked configuration.

## Invariants and progress

`SourcePrecedence` checks the chosen source against the local/inherited rule.
`NoAuthoredResynthesis` checks the snapshot's origin at every active phase, and
`ContractNeverSynthesized` checks enum/default source selection independently.
`BranchRetention` compares resolved selections with the input choices;
`PresenceRetention` compares raw/resolved presence with the independent field.
`ProjectionRetention` compares wire selections and presence with the separate
`ObservedPath`/`ObservedPresence` target observation, including legitimate loss.

`CacheOwnership` checks each consumed entry against the requesting occurrence,
source, role, representation and target. The check happens before the final
output, even when two entries happen to contain equal payloads. `PhaseOrder`
checks each job's history; `SuppressionBeforeSelection` prevents materializing
an effectively suppressed/excluded example. `EmissionAfterValidation` requires
all five steps before emission.

`TerminalOutcome` independently specifies each final result: emitted for a
representable supplied/synthesized value, omitted for an unrepresentable
example, error for an unrepresentable enum/default, suppressed for an excluded
input, and no-value for an absent enum/default. `Termination` requires every job
to reach its terminal state under weak fairness. These checks are not satisfied
by always omitting examples: `always-omit.cfg` demonstrates that failure.
All scenario classes are reachable; none is hidden behind a state constraint.

## Recorded results and minimal traces

Recorded on 28 September 2026 with one TLC worker. Counts include all initial
states explored before a failure, not just its displayed trace. Depth includes
the initial state. Failed runs stop at the first violation, while the checked
run explores its complete bounded graph and checks termination.

| Configuration | Expected invariant/result | Generated / distinct states | Depth | Counterexample transition sequence |
| --- | --- | ---: | ---: | --- |
| `legacy.cfg` | `BranchRetention` fails | 73 / 73 | 3 | Select synthetic source; resolve `a/x` as inner payload `x`, losing the outer selection. |
| `reselection.cfg` | `ProjectionRetention` fails | 361 / 193 | 4 | Select local source; resolve `b/x`; projection guesses `a/x` from the common payload shape. |
| `authored-replacement.cfg` | `NoAuthoredResynthesis` fails | 121 / 73 | 3 | Select local authored source; resolution replaces its origin with synthetic. |
| `source-precedence.cfg` | `SourcePrecedence` fails | 25 / 25 | 2 | Both local and inherited sources exist; selection incorrectly chooses inherited. |
| `cache-occurrence.cfg` | `CacheOwnership` fails | 1,442 / 866 | 3 | Job 1 caches raw occurrence 1; job 2 asks for occurrence 2 and gets occurrence 1. |
| `cache-source.cfg` | `CacheOwnership` fails | 1,442 / 866 | 3 | Job 1 caches inherited source; job 2 asks for local override and gets inherited. |
| `cache-role.cfg` | `CacheOwnership` fails | 1,154 / 866 | 3 | Job 1 caches an example; job 2 asks for enum and gets the example entry. |
| `cache-phase.cfg` | `CacheOwnership` fails | 121 / 73 | 3 | Select caches raw input; resolve reads that raw entry as a semantic result. |
| `cache-target.cfg` | `CacheOwnership` fails | 990 / 507 | 7 | Job 1 selects, resolves and projects full `a/x`; job 2 selects/resolves and reuses that full wire entry for its body plan, which requires `a`. |
| `always-omit.cfg` | `TerminalOutcome` fails | 697 / 361 | 6 | Select, resolve, project, validate, finish a representable value with omitted instead of emitted. |
| `checked.cfg` | All invariants and termination pass | 92,304 / 47,712 | 11 | All ten scenarios and their permitted branch/presence choices, roles and processing orders. |

## Correspondence to inspected production seams

The code mapping was inspected at `ff7873dfff1ad3803c421e1e64978906ee1139ce`.
Model variable names are architectural concepts, not claims that equivalent
production types already exist.

| Model operation/property | Inspected Go seam | Status and limit |
| --- | --- | --- |
| Source selection and precedence | `expr/attribute.go`: `ExtractUserExamples`/`extractUserExamples`; `expr/example.go`: `AttributeExpr.Example` | Current code checks local examples, ordered references/bases/type, then the last selected example; synthesis follows only when no authored example exists. The model groups inherited tiers and tests preservation of their selected identity. |
| Legacy outer-branch loss | `expr/types_union.go`: `Union.Example` | Still present in the shared raw expression path at the inspected revision: it returns the chosen branch payload. The model abstracts a nested payload as `x`; it does not claim the real raw value retains even the inner tag. |
| Payload inference / rejected reselection | `expr/example_canonicalization.go`: `canonicalizeUnionExample`, `pickUnionVariantForExample` | Current code honors `examplevalue.Union`, otherwise infers from raw shape and returns no selected branch for ambiguity. The model's deterministic first-branch guess is a **rejected repair**, not a claim that current code picks the first ambiguous branch. Both illustrate why a lost choice cannot be reconstructed from identical shapes. |
| Authored replacement | `grpc/codegen/client_cli_example.go`: `protoJSONExampleR`, `protoJSONObjectExample`, `protoJSONOneofExample` | Current union-containing path walks fields and synthesizes union choices rather than projecting a selected authored root example. The fault mode abstracts that loss of origin; exact emitted value behavior belongs to the generated gRPC probes. |
| Occurrence/source/role cache ownership | `expr/random.go`: `ExampleGenerator.exampleSlot`; `expr/user_type.go`: `recExample` | Current raw-example memo keys type ID plus the private HTTP-wrapper bit. It is not a semantic/wire cache. The candidate model requires richer keys for the planned shared resolved pipeline; the isolated faults are negative controls, **not claims of five independently reproduced production bugs**. |
| Separate representation caches | `http/codegen/openapi/internal/ir/example_generation.go`: `synthesizedOpenAPIExample`, `selectedExampleUnion.Example` | Already repaired privately for OpenAPI: copied expression graph, retained branch wrapper, separate generator memo. `legacy.cfg` does **not** describe this current OpenAPI path. The model generalizes its ownership principle to the shared pipeline. |
| Schema/body reuse across representations | `http/codegen/openapi/internal/ir/analyzer.go`: `analyzeUserType`; `analyzer_body_types.go`: `analyzeRequestBody`; `document.go`: `buildRequestBody` | Confirmed generated sharing at the inspected revision: one canonical Bytes declaration is reused by JSON, multipart and query occurrences. The current schema fingerprint and canonical type-ID reuse do not include a representation plan. `cache-target.cfg` reproduces this ownership class abstractly; it does not execute schema construction or component naming. |
| Selected target and final validation | Design's effective occurrence, projection plan, and target-validation contract | Proposed shared operations, not an implemented Go API. The model's body observation and representability input abstract their responsibilities; actual decoder/schema validation remains mandatory. |

The schema-sharing specimen declares `Blob` as Bytes with `MinLength(2)`,
`MaxLength(2)`, `Example([]byte("hi"))`, and canonical OpenAPI name `Blob`.
Three methods reuse it as required `data`: an ordinary JSON POST, a POST with
`MultipartRequest()`, and a GET with `Param("data")`. Generation at the inspected
pushed revision gives both POST bodies the same `JSONRequestBody` reference;
its `data` property and the query parameter all reference the same `Blob`
component. That component has `format: binary`, `minLength: 2` and
`maxLength: 2`. Changing it to the corrected JSON base64 contract would also
change the other occurrences unless schema ownership follows representation.
This is observed schema sharing, not a claim that every currently shared schema
is already wrong: the conflict becomes explicit when applying the byte fix.

In `cache-target.cfg`, a full-plan result reused for a body-plan request is the
finite analogue of reusing one schema result for incompatible codec plans.
Its `CacheOwnership` failure rejects the missing target dimension before final
emission. The checked model retains that dimension and passes, but it does not
prove a particular schema key includes it or that the plan was correctly derived.
Complete recursive schema graphs, JSON-primary canonical names, deterministic
representation variants and explicit-name collision handling remain obligations
of the approved design and its Go tests. They are not encoded by the model's two
observation plans, so no new model invariant or larger proof claim follows from
this correspondence.

The same specimen exposed a separate named-Bytes validator compilation failure
(`[]byte(body.Data)` with `body.Data` of type `*BlobRequestBody`). The unnamed
Bytes runtime specimen independently demonstrates the decoded-byte/schema-length
mismatch. Named alias correctness, byte-length arithmetic, lexical base64
acceptance (including accepted noncanonical pad bits and rejected CR/LF), and
schema/runtime union ambiguity belong to the Lean correspondence ledger and
generated Go/schema probes. This TLA+ model proves none of those codec facts.

## Evidence limits

This establishes bounded ordering and ownership checks only. It does not prove
Go refinement, arbitrary nesting, recursive graph termination, map-key collision
handling, scalar coercion, byte/base64 correctness, numeric ranges, discriminator
allocation, target-plan derivation, JSON/protobuf serialization or decoding,
custom marshaler purity, cross-process determinism, or generated compilation.
It assumes immutable built-in input observations and the supplied target
eligibility/representability classifications. It does not prove those inputs
were correctly derived from the DSL. Presence equality is checked only for the
explicit field/visibility abstraction; it does not replace codec-specific
`omitempty`, defaults, nullable or protobuf equivalence rules.

The Go characterization/comparison corpus and later resolver/conformance tests
must map the same witnesses to production behavior. Lean must separately prove
its declared structured-value properties, and the correspondence ledger must
keep proved, bounded-checked, tested and assumed claims separate. A passing TLC
run alone cannot close a consumer migration milestone.
