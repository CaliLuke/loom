# Value pipeline root-cause repair

Implement the direction in [the design](value-contract-design.md): resolve value
meaning, presence, provenance and union choice once, then project into each
transport. Preserve supported untagged bodies and existing runtime contracts.
The [inventory](value-contract-inventory.md) lists the production consumers.
This is a staged replacement of duplicated decisions, not a framework rewrite.

## Status

- 2026-09-28 — Design and execution plan independently reviewed; all findings resolved.
  The design and plan were approved before implementation.
- 2026-09-28 — Added the user-approved Lean proof requirement and explicit proved/tested/assumed boundaries; proof implementation was pending at plan approval.

- 2026-09-28 — User authorized execution through agents, with the parent coordinating. Milestone 1 established the baseline and proof foundation; implementation ownership and atomic ticket order are below.


- 2026-09-28 — Milestone 1 complete: #566 retains 24 probe characterizations and
  96 independent generation records with no unexpected differences; #567 audits
  22 initial Lean witnesses, including newly found byte-length/decoder-alias
  weaknesses; #568 checks bounded ordering/cache ownership with 47,712 distinct
  candidate states and ten failing negative controls. Candidate correctness and
  production Go correspondence remain pending milestone 2. The actual schema
  representation conflict and retained named-Bytes compile failure are tracked
  by #574–#576; #575 records the reviewed contract clarifications before further
  implementation. Production generator behavior is unchanged by this milestone.

Evidence lives in [the baseline record](../internal/valuecontract/BASELINE.md),
[Lean proof guidance](../expr/lean/value_projection/README.md),
[the correspondence ledger](../expr/lean/value_projection/correspondence.md), and
[the TLA+ record](../expr/tla/value_projection/README.md). Full `make lint` and
`make test` passed; final changed-package checks, proof negative controls,
independent exact-diff reviews and refreshed affected TLC checks passed after
repairs. These characterize existing defects; they do not claim those defects
are fixed.

#576 fixes the shared validator's conversion of forced named-byte pointers.
The [repair evidence](../codegen/bytes_validation.md) records the 80-layout
compiled regression, generated HTTP/JSON-RPC runtime coverage, and parent/current
comparison of 13 retained cases plus four new probes. Exactly five generated
files change across three probes; the minimal unpreserved request control and
all retained cases stay byte-identical. Full lint, tests and generated-code
quality gates pass. The separate invalid byte-default transform discovered in
verification is fixed by #577 at the shared default-transform owners. Its
[repair evidence](../codegen/bytes_defaults.md) records 448 compiled layouts,
64 presence/context cases, generated runtime coverage and a passing 12-probe
comparison (48 runs, 18 intended generated changes). The shared semantic core in #570 builds on the #569 candidate proofs;
the next consumer is #574's representation-aware schema analysis.

## #570 delivery checkpoint

The shared semantic API, immutable occurrence/source ownership, JSON projection,
service carriers and representation plans are implemented. The first service
consumers use one retained selection/resolution/synthesis result; later rendering
consumers remain assigned to M3–M6. Direct tests live under `expr/value_*_test.go`,
`codegen/service/value_data_test.go` and `http/codegen/value_plan_test.go`.
The [inventory checkpoint](value-contract-inventory.md) records the actual
ownership changes and retained compatibility paths.

The combined proof/conformance target passed: 567 audited theorems, fresh kernel
replay, real rejected-proof controls and 13 executed conformance assertion groups.
The corpus includes 402 production source comparisons, 48 projection comparisons
and 420 native numeric target cases. These counts describe tested cases, not a
universal Go refinement. The ledger lists unsupported extraction boundaries.
The checked TLA+ configuration passed with 47,712 distinct states.

The first full Go/coverage runs found two integration regressions: premature
JSON-name uniqueness in logical service capture, and stale naming provenance in
projected collections. Both shared-owner repairs passed direct, generated and
independent scoped review. Existing gRPC metadata round trips and meal-planner
JSON/YAML goldens remain unchanged. The repeated full lint, test and coverage
ratchet gates pass. Final comparison covers six cases and 24 generation records:
meal-planner and mapped-metadata artifacts are byte-identical, and the remaining
cases retain exactly the 28 previously reviewed differences. The durable
[comparison record](../internal/valuecontract/OCCURRENCES.md) preserves all
inputs and results. Atomic delivery requires final exact-diff review after this
evidence is frozen; all later milestones remain open.

## Dependency tickets and commit order

Every row is one atomic ticket commit. Preparation may run in parallel only
across explicitly assigned disjoint paths with satisfied prerequisites. Commits
land in the order below after independent review and gates; #576 and #569 may
be prepared together once M1 is complete.
Reviewers never review a slice they implemented. Shared checkout:
`/Users/luca/.codex/worktrees/body-type-names/loom`.

| Order / milestone | Ticket | Scope and owner | Prerequisites |
| --- | --- | --- | --- |
| 1 / M1 | [#566](https://github.com/CaliLuke/loom/issues/566) | `internal/valuecontract/**`: baseline agent; execution record/inventory reconciliation: coordinator | Approved design/plan |
| 2 / M1 | [#567](https://github.com/CaliLuke/loom/issues/567) | Lean foundation, audited witnesses and proof gate | Independent baseline/model evidence |
| 3 / M1 | [#568](https://github.com/CaliLuke/loom/issues/568) | Bounded pipeline ordering and cache-ownership model | Independent baseline/model evidence |
| 4 / M2 | [#575](https://github.com/CaliLuke/loom/issues/575) | Reviewed contract clarifications, proof navigation and maintenance/cleanup process | M1; explicit counterexamples for #574 and #576 |
| 5 / M2 | [#576](https://github.com/CaliLuke/loom/issues/576) | Named Bytes validation pointer conversion and generated compile regression | M1; separate atomic correctness repair; preparation may overlap #569 |
| 6 / M2 | [#577](https://github.com/CaliLuke/loom/issues/577) | Byte-default physical type/presence handling and generated compile/behavior regression | #576; independent default-transform owner |
| 7 / M2 | [#569](https://github.com/CaliLuke/loom/issues/569) | Universal candidate Lean semantics and reviewed theorem statements | M1, #575 |
| 8 / M2 | [#570](https://github.com/CaliLuke/loom/issues/570) | Go semantic API/projection, shared value and representation-plan carriers, executable Lean correspondence and CI | M1, #575, #569, #577 |
| 9 / M2 | [#574](https://github.com/CaliLuke/loom/issues/574) | Byte JSON grammar/length schemas and complete representation-specific component ownership | Reviewed #575 policy; #570 target plans; #576 named-byte runtime probe |
| 10 / M3 | [#571](https://github.com/CaliLuke/loom/issues/571) | Shared enum/default/inline-schema consumer migration | #570, #574 |
| 11 / M3 | [#456](https://github.com/CaliLuke/loom/issues/456) | Deterministic path-qualified collision rejection and regression | #571; close only after affected consumers satisfy the contract |
| 12 / M4 | [#572](https://github.com/CaliLuke/loom/issues/572) | OpenAPI shared value pipeline and independent example validation | #571, #456, #574 |
| 13 / M5 | [#565](https://github.com/CaliLuke/loom/issues/565) | HTTP/JSON-RPC body examples/defaults, CLI hints and usage migration | #572 |
| 14 / M6 | [#434](https://github.com/CaliLuke/loom/issues/434) | Protobuf projection preserving authored source and branch | #565 |
| 15 / M6 | [#573](https://github.com/CaliLuke/loom/issues/573) | Final inventory cleanup, property tests, docs and complete proof/test evidence | All preceding rows |

The initial inventory was refreshed at `ff7873df`: all 31 production matches
still reconcile to the inventory. `expr` source/canonicalization and shared
carriers map to M2; enum/default/inline-schema consumers to M3; OpenAPI to M4;
HTTP/JSON-RPC and CLI routing/usage to M5; protobuf and final bypass removal to
M6. Retained public compatibility adapters and location/custom codecs keep the
explicit dispositions in the inventory. Additional matches are reconciled
before their owning milestone changes code. The coordinator owns this execution
record; agents must not edit one another's paths or commit shared work.

## Execution rules

Run from the executing Loom checkout. Read `AGENTS.md` and
`.agents/skills/loom-framework/SKILL.md`. Use Go 1.27 and the repository's
`make depend` prerequisites. Set `LOOM_DIR` to that checkout during unpushed
development; use pinned pushed commits for subsequent remote parity checks.

Milestones are ordered, not automatic commit boundaries. Before implementation,
map their shared infrastructure work to narrow dependency tickets. Keep #456,
#565 and #434 as separate atomic ticket commits; link shared prerequisite
commits rather than splitting a ticket's own fix across commits. Do not close
the broader #374–#377 work solely on the strength of this plan.

For every commit, state scope and acceptance rules, run the relevant commands,
and obtain independent review of its exact final diff against its intended
parent. Resolve every finding under `AGENTS.md`, obtain re-review, then commit
without edits. Push after the repository gates pass. A red required gate blocks
the commit; unrelated defects need separate tickets and atomic fixes, not hidden
changes to this work. No implementation is authorized merely by publishing this
planning document.

Every generator commit needs a comparison record for its parent and candidate:
affected checked-in fixtures, affected exported testdata designs, and the same
new probe designs on both revisions. Record generated bytes, generation/build/
vet outcomes, expected old failures, and every intended difference. Unaffected
Go/proto/schema output stays identical. Changed CLI help literals and example,
enum or default values must be individually explained. Run two independent
generation processes at each revision to test determinism. Never edit `gen/`.

The comparison harness, initial Lean files, `make value-contract-proof` and
TLA+ model commands landed in milestone 1. Candidate proofs landed in #569;
#570 implements the executable reference and `make value-contract-conformance`.
The harness uses the existing source-resolution and process-ownership helpers
without duplicating the repository's lint/build gates. Keep
active review evidence outside the checkout under a unique
`/tmp/loom-value-contract/<run>/` directory. Commit durable probes, manifests,
reproduction commands and compact findings beside their tests/models. Once that
ticket's validation and independent review finish, delete its temporary result
trees, generated modules, compiler output and raw logs; do not retain them as
repository artifacts. Preserve inputs still needed by a named active task.
Installed reusable toolchains are not task output.

A newly discovered conflict between runtime behavior and the accepted design
is a **Design Blocker**. Record the counterexample and return to the design;
do not silently widen runtime acceptance or approve a changed golden.

## Architecture-first execution

The user requires execution to converge on the shared architecture rather than
expand into successive local repairs. The #577/#569 prerequisites are delivered;
finish #570's final review and proceed through the planned consumer migrations. Do not add newly discovered local defects
as prerequisite patch tickets. Record their concrete failures, assign them to
the shared semantic or representation-plan owner, and make them acceptance cases
for that architectural implementation. A check failure still needs an honest
disposition; it is not permission to hide a regression or waive a required gate.

In particular, the named nullable default in a result view that loses nullability
before HTTP/JSON-RPC transformation belongs to #570's occurrence/shape propagation
acceptance. Preserve the reproduction and test the complete declaration-to-plan
path when implementing that owner. Do not repair individual renderer call sites.

Proof work must converge on the executable resolver/projection contracts needed
by #570. Keep the approved domain and honest boundary claims, retain discovered
counterexamples, and use external-codec/compiler tests for those explicit
boundaries. Do not turn every incidental runtime detail into another prerequisite
model expansion. The next architectural delivery must establish the common owner
and remove duplicated decisions from its first consumers.

The #570 source-codec correction and shared selection/resolution/synthesis
ownership passed the registered proof/conformance and repository gates. Obtain
the final exact-diff review and deliver the dependency before beginning #574. Do not add another prerequisite Lean vocabulary expansion merely
because a production adapter does not yet translate a supported DSL shape.
Unsupported translation must fail explicitly; it must never be counted as a
passing comparison or used to skip an already registered failing obligation.

Alias-local and map-key constraints are supported by the shared Go implementation
and have direct regressions and independent review. Full independent lowering of
those constraints into the Lean reference remains pending. #574 must discharge
alias-local schema/length correspondence before using it for byte-schema claims;
#571 must discharge cumulative alias/key-constraint and key-enum correspondence
before using it for migrated enum/default consumers. Keep the existing adapter
rejection controls until those translations are implemented and reviewed. These
are existing migration obligations, not new prerequisite patch tickets. The
proof ledger must distinguish tested Go behavior from these unproved extraction
boundaries throughout the migration.

The full gRPC mapped-metadata matrix also requires logical declarations with
unique authored names but duplicate JSON aliases: transport components can make
those names independently valid. Source ambiguity and target emitted-name
validation belong to different owners. Preserve direct/generated coverage and
an explicit adapter rejection for graphs outside the model's unique-wire-alias
premise. #573 must reconcile this lowering boundary, or the first earlier
migration that needs it; existing cross-namespace authored/wire overlaps stay
inside the current correspondence domain.

## Formal assurance requirements

Candidate Lean proofs must establish universal properties of the finite semantic
domain defined in the design; the initial legacy witnesses do not discharge them. TLA+ explores bounded phase/cache ownership. V1 connects Lean to Go
through an executable reference and differential tests; it does not prove the
Go implementation or universal validity of rendered Go. Compilation, schema
validation and actual generated decoder checks remain required.

Durable Lean files belong in proposed `expr/lean/value_projection/`. Its
`correspondence.md` must list each theorem/domain, transitive axioms, Go owner,
conformance cases, excluded cases and external assumptions, with separate
**proved**, **bounded-checked**, **tested**, and **assumed** status. Pending proof
obligations are never reported as established. A passing model without the
required Go correspondence checks cannot close a migration milestone.

Every milestone that changes a modeled operation or migrates a consumer must
extend the correspondence cases and run `make value-contract-conformance` after
its targeted Go tests. That target depends on `make value-contract-proof`; the
latter builds and audits Lean, while conformance builds the executable reference
from the same proved functions and invokes the Go differential tests. Pin a Lean toolchain at least version 4.28, including its bundled
`leanchecker`; they are development/CI dependencies,
not dependencies of generated services. Explain any additional proof library
before adding it. New semantic rules require theorem/ledger review before their
implementation; counterexamples cannot be disposed of by narrowing the domain.


### Proof maintenance during execution

The user requires the proofs to stay current as weaknesses are discovered. Every
new counterexample is first recorded against the affected definition, theorem,
assumption or correspondence boundary. Update the model and executable negative
control when the modeled behavior is wrong or incomplete; add a concrete legacy
witness before the candidate repair where the domain supports it. Update
`expr/lean/value_projection/correspondence.md` in the same ticket, including the
case identifier and the honest proved/bounded-checked/tested/assumed/pending
status. A defect outside the model, such as emitted Go pointer syntax, remains an
explicit compiler-test obligation; do not invent a semantic proof claim for it.

Invalidate affected review claims when definitions, premises or modeled behavior
change. Re-review theorem statements and non-vacuous domains independently, rerun
`make value-contract-proof` (including the axiom audit and fresh kernel replay),
then run the relevant Go correspondence/negative controls before dependent code
uses the changed claim. Rerun affected TLA+ configurations for phase/cache changes.
Keep unaffected established lemmas, but never use their previous green result as
evidence for a newly expanded domain. No weakness may be disposed of by silently
narrowing the domain, strengthening a premise to assume the result, dropping a
negative control, or leaving the ledger's former claim unchanged.

Use the [formal model index](../.agents/skills/loom-framework/references/formal-models.md)
to find each owner, local run instructions and coverage limits. Retain proof
sources, configurations, theorem manifests, pinned toolchain declarations and
correspondence records. Follow the cleanup process in `AGENTS.md`: remove
compiled proofs, checker state/trace output and temporary probes after validation
and review, including ignored local build output. Cleanup must not remove another
active task's inputs; no build-script changes are required by this policy.

## Milestones

### Milestone 1: Reproducible failures and a comparison baseline

Goal: Establish evidence that can distinguish a root-cause repair from another
local patch before production behavior changes.

Acceptance Criteria

- The executing agent has recited the workflow before code edits.
- Every inventory match has a disposition and milestone owner; no discovered
  built-in consumer is silently excluded.
- Checked-in probes reproduce the authored-byte CLI failure, nested branch loss,
  authored gRPC loss with and without unions, and map-key collision behavior at
  `f5b786b39675e2b5c1466f04f3301b7b337779b6`.
- The comparison harness retains exact artifacts and build/vet results from
  both revisions, and detects a deliberate artifact-byte mismatch in its tests.
- Lean definitions and explicit legacy counterexample witnesses build without
  admitted proofs; the initial ledger labels the main preservation/progress
  theorems as pending milestone 2. No complete-proof claim is made at this stage.
- TLC produces counterexamples for the modeled old/rejected paths and passes
  the candidate within documented bounds; its README maps states to real seams.

Checklist

- [x] Read this plan end-to-end, the design, inventory, `AGENTS.md` and `.agents/skills/loom-framework/SKILL.md`; recite milestone order, exit criteria, commands in execution order, test-first work, independent review, commit/push handoff and inherited constraints before editing code.
- [x] Run both exact `rg` commands in `value-contract-inventory.md`; reconcile every match and additional boundary to milestones 2–6, including retained compatibility adapters and plain codecs. Record the dependency-ticket/atomic-commit map in this plan before implementation.
- [x] Add the new comparison driver at `internal/valuecontract/compare_test.go`, with reusable probe designs under `internal/valuecontract/testdata`; reuse source resolution and `internal/testprocess`. Its explicit inputs are `LOOM_VALUE_BASE`, `LOOM_VALUE_CANDIDATE`, and `LOOM_VALUE_RESULTS`; its opt-in test is `TestCompareRevisions`.
- [x] Make the driver generate a common probe corpus under each revision, including probes absent from the old source tree, and capture relative paths plus exact generated bytes before temporary modules are removed. Capture generator/build/vet failures separately; do not compare source-location-dependent `go.mod` replacements as generated artifacts. Record selected affected fixture/testdata IDs and all intended differences in a checked-in manifest.
- [x] Turn applicable temporary Goa-audit cases into the durable probe corpus: authored text/binary/empty bytes, nested unions with distinct names but identical payloads, authored gRPC payload/type examples with and without unions, integer limits, mixed-key maps, mapped names, null/empty/absent values, recursive types, and custom codecs. Record legacy failures as characterization results, not permanently skipped regression tests.
- [x] Extend the probe matrix with method-local requiredness, HTTP Body selections, views, documentation-only bodies, explicit transport-local examples, suppression/exclusion, designed errors and streaming-message positions. Use existing mapped-names, type-identity, protojson and collection-default fixtures where suitable.
- [x] Create `expr/lean/value_projection/lean-toolchain`, `lakefile.toml`, `ValueContract/Model.lean`, `ValueContract/Legacy.lean`, `ValueContract/AxiomAudit.lean`, `README.md` and `correspondence.md`. Define finite typed values, raw inputs, outcomes, target plans, independent typing/decoding/observation and representability. Prove concrete legacy witnesses for branch loss, byte reinterpretation and authored-value replacement before implementing candidate rules; include legitimate field/visibility loss and distinguish explicit null from absent. Pin the toolchain and record the theorem-to-Go seam mapping.
- [x] Add planned `make value-contract-proof` through `scripts/check_value_contract_proof.sh`: run `lake build`, run `lake env lean ValueContract/AxiomAudit.lean`, and recheck the initial witness module with `lake env leanchecker --fresh ValueContract.Legacy`; milestone 2 additionally runs `lake env leanchecker --fresh ValueContract.Proofs`. Use the bundled checker from the pinned Lean >=4.28 toolchain, following the [upstream checker instructions](https://github.com/leanprover/lean4checker#using-the-built-in-leanchecker). The script must fail on warnings, missing required theorem names and forbidden transitive axioms; allow only the recorded standard logical axioms (`propext`, `Classical.choice`, `Quot.sound`). Include negative controls for an admitted dependency and an unlisted custom axiom. Document setup and exact checker invocation in the Lean README. Run `make value-contract-proof` and record concise results for the initial legacy witnesses before deleting reviewed temporary logs; milestone 2 adds all candidate obligations to the required theorem manifest.
- [x] Add `expr/tla/value_projection/ValueProjection.tla`, `legacy.cfg`, `reselection.cfg`, `checked.cfg`, and `README.md`. Model source precedence, no authored-to-synthetic substitution, representation phases, branch/presence retention, and occurrence/source/cache ownership. Reproduce branch-loss/reselection before checking the proposed pipeline; keep byte encoding and arbitrary user codecs outside the abstraction.
- [x] Run `go test ./internal/valuecontract ./internal/testdatacompile`, then the new `LOOM_VALUE_BASE=f5b786b39675e2b5c1466f04f3301b7b337779b6 LOOM_VALUE_CANDIDATE="$PWD" LOOM_VALUE_RESULTS=/tmp/loom-value-contract/baseline go test ./internal/valuecontract -run '^TestCompareRevisions$' -count=1 -timeout=90m`; preserve the checked-in baseline manifest and concise result summary, then delete temporary results after review. The driver must distinguish expected legacy probe failures from infrastructure failure.
- [x] From `expr/tla/value_projection`, run `java -cp "$TLA_TOOLS_JAR" tlc2.TLC -config legacy.cfg ValueProjection.tla`, repeat with `reselection.cfg` and `checked.cfg`, and record invariant names, traces, state counts and bounds. The first two must fail the named invariants; the checked model must pass. Document installation/version of the JAR without adding a runtime dependency.
- [x] Run `make lint` and `make test`; obtain independent exact-diff review, resolve findings, and commit/push the baseline/model dependency work.

### Milestone 2: One semantic resolver and immutable resolved values

Goal: Establish the shared semantic owner without making each transport invent
its own branch, coercion or source-selection rules.

Acceptance Criteria

- Kernel-checked theorems establish candidate projection soundness, progress,
  retained-branch/presence preservation against `Observe(plan, value)`, and
  authored-source preservation for the declared structured JSON/protobuf domain.
  The axiom audit has no admitted or custom-correctness assumptions.
- Direct tests in `expr` check all design outcomes, source precedence, target
  occurrence identity, built-in immutability and presence-table cases.
- Synthesized nested unions retain each chosen branch; authored values never
  invoke synthesis. Cache keys cannot mix sources, occurrences or representations.
- Public raw-value APIs and custom callbacks keep their documented compatibility
  behavior; new semantic queries do not expose internal-package types.
- The executable Lean reference and production core agree on fixed and generated
  conformance cases; negative controls catch byte, branch, presence and omission
  mismatches. The ledger labels this agreement as tested, not proved refinement.
- The candidate models and direct tests agree on the modeled counterexamples.

Checklist

- [x] Complete #575 as a design-only commit: record complete-first partial-union rules, exact JSON Bytes grammar/length policy, complete schemas per representation, JSON-primary canonical naming and the narrow derived-reference compatibility exception. Independently review the integrated design and dependency order; run `go run ./scripts/docscheck` and `git diff --check` before commit/push. Proof navigation is linked from the framework skill, and AGENTS.md owns the manual artifact cleanup process. Candidate proofs may model this chosen contract before the production schema correction; no current-schema correctness claim follows.
- [x] Complete #576 separately: add a failing direct regression at the shared validation seam (`codegen/validation_render.go`, `codegen/validation_test.go` or a focused new test file) plus a generated HTTP named-Bytes compile regression alongside `http/codegen/optional_validation_compile_test.go`. Inspect actual pointer/value/type references, then fix the conversion owner without forcing pointer semantics. Cover required/optional aliases and all affected transport callers, compare parent/candidate generated bytes and compile results, run targeted codegen/transport tests plus lint/test, and obtain independent exact-diff review before its one commit/push.
- [x] Complete #577 separately after #576: reproduce invalid native/named Bytes default comparisons at `codegen/go_transform.go` and nullable named-byte default type inference at `codegen/go_transform_presence.go` with compiled direct and generated HTTP/JSON-RPC tests before changing the default-transform owners. Verify generated form requests apply declared defaults to omitted native/named bytes even when the source context does not apply defaults; preserve explicitly supplied empty/nonempty bytes. Preserve null and supplied values, and honor the target default/pointer policy for Nullable just as for Optional/native fields; cover comparable scalar and raw-JSON controls. Restore the full byte-default transport cases discovered during #576 verification. Compare common parent/candidate fixtures, testdata and new probes twice with exact bytes and compile/vet results, run required gates, obtain independent exact-diff review, then commit/push. A matching golden containing an invalid slice equality is not passing compilation evidence.
- [x] Add fail-first source/resolution/eligibility regressions under `expr/value_*_test.go` and retain the existing example-canonicalization tests. Pin legacy public adapter outputs for partial, ambiguous, invalid and unsupported values separately from new built-in guarantees.
- [x] Implement and prove candidate semantics in `ValueContract/Resolve.lean`, `Projection.lean` and `Proofs.lean` under the Lean project. Define target validity and decoding independently of the projector; prove soundness and progress for complete representable values, with legitimate target field loss retained in the domain. Prove failure/source-selection properties and finite recursive-value cases without a fixed maximum depth. The #569 gate checks 483 required theorems, transitive axioms, fresh kernel replay and actual admitted-proof/custom-axiom/missing-theorem rejection controls. Independent review checks theorem statements, definitions, non-vacuous premises and the axiom report. The [proof README](../expr/lean/value_projection/README.md) maps the stages; the [correspondence ledger](../expr/lean/value_projection/correspondence.md) distinguishes proved model claims from the production boundaries still owned by #570 and later migrations. Observation must finish before canonical construction, so omitted children cannot fail construction of wire objects that will never be emitted.
- [x] Introduce the opaque result/query declarations in `expr/resolved_value.go`, with fail-first table-driven resolver coverage in `expr/value_resolve_test.go` and the focused occurrence/source test files. Cover effective method copies, independent documentation bodies, explicit transport overrides, absent source versus explicit null, wrong types, cycles, collisions, recursion and custom-materialization failure.
- [x] Establish selection/resolution/synthesis separation in `expr`, using private node storage and copied read access. First service consumers must share source selection, retained branch choices, occurrence-scoped caching and publication of the same sampling result. Existing `example.go`, `example_length.go`, `types.go`, `types_union.go`, `user_type.go` and `random.go` may remain the sampling engine behind the isolated synthesis graph; rewriting these files is not itself an acceptance condition. Keep public raw-value and `example_canonicalization.go` compatibility behavior while their remaining consumers migrate in M3–M6, then remove duplicate semantics under #573. Prevent `expr` from importing generators or `internal/enumvalue`.
- [x] Add common JSON projection beside the resolver, accepting a resolved value and target-shape plan. Test bytes/String/Any distinctions, exactly-once encoding, mapped fields, immutable access, occurrence-specific memoization, custom snapshot counts, suppression before materialization and independently seeded map inputs.
- [x] Add `ValueContract/Reference.lean` and a Lake executable `value_contract_reference` that invokes the proved functions directly. Add `internal/valuecontract/conformance_test.go` with `TestLeanConformance`. Ordinary Go tests skip only this external-reference test when `LOOM_VALUE_CONFORMANCE` is not `1`; adapter unit tests and negative controls remain in the ordinary tier. Wire `make value-contract-conformance` to build the reference and run `LOOM_VALUE_CONFORMANCE=1 LOOM_LEAN_REFERENCE=/absolute/path/to/value_contract_reference LOOM_VALUE_CONFORMANCE_REPORT=/absolute/path/to/conformance-results.json go test -json ./internal/valuecontract -run '^TestLeanConformance$' -count=1 -timeout=10m`, substituting the actual built executable path. Enabled mode fails on a missing/non-executable reference. The gate must require a passing, non-skipped `TestLeanConformance` event and a nonzero executed assertion-group count for every required owner in `conformance-results.json` (groups are not reported as corpus input counts); test the gate against missing tools and a skipped/zero-case result so it cannot silently pass. Test the input/output adapter independently: use typed map-entry lists preserving collisions, explicit source/occurrence IDs, branch IDs and presence tags. Compare outcomes, target values and observations on fixed regressions and deterministically generated finite inputs; test negative controls for double-encoding, branch substitution, lost null/absence and always-omit behavior. Record coverage and trusted serializer/compiler boundaries in `correspondence.md`. Run `make value-contract-conformance`.
- [x] Carry semantic values/source identity through `codegen/service/service_data.go`, `service_data_methods.go`, and `expr/http_body_types.go` before consumer migrations; preserve distinct documentation-only/transport-override occurrences and existing exported carrier compatibility. Add direct tests for payload/result/error/stream copies before changing these carriers. Derive a durable representation/shape plan from the actual transport encoding path and make it available to schema/body analysis as well as value projection. Include codec owner, occurrence, selected fields, names, requiredness and visibility; custom runtime injection stays an explicit boundary. `http/codegen/internal/transportir` must consume this plan without inventing another classifier. Test equal logical types used across JSON, multipart/raw and parameter positions; media labels alone cannot determine the codec.
- [x] Run `go test ./expr ./internal/enumvalue ./codegen ./codegen/service ./codegen/cli`; rerun checked TLC and the revision comparison with this commit's parent as `LOOM_VALUE_BASE`. Enumerate changed examples; retain legacy adapter and custom callback outputs.
- [ ] Add a dedicated proof/conformance job to `.github/workflows/test.yml` and make `ci-local` depend on `value-contract-conformance`; use the same scripts locally and in CI, install pinned Lean/checker tools, and fail rather than skip when unavailable. Run `go fmt ./...`, `make value-contract-conformance`, `make lint`, `make test`, and `make coverage-ratchet`; obtain independent exact-diff review, resolve findings, and commit/push the semantic-core dependency work.

- [ ] After #570, implement #574 in its own atomic commit. Add failing byte-length/schema assertions in `expr/json_schema_inline_test.go`, OpenAPI IR `byte_values_test.go`, and actual generated server/independent schema tests. A dependency-free helper under proposed `internal/byteschema/` owns residue grammars and checked arithmetic; both schema owners adapt it. Extend `expr/json_schema_inline.go`, OpenAPI `schema.go`, IR `model.go`/`render.go` and relevant clone/hash/reference/filter walkers for the required standard schema vocabulary. Preserve every field in manual marshal/clone paths.
- [ ] Consume #570's representation plan in IR `analyzer.go`, `analyzer_user_type.go`, `analyzer_hash.go`, `analyzer_body_types.go` and `document.go` before component registration. Build complete request/response schemas per actual representation, including docs-only bodies and explicit custom/non-JSON boundaries. Reserve public canonical names before deterministic variant allocation; JSON keeps the primary name for genuinely conflicting mixed-use declarations, and equivalent projected schemas still share. Preserve existing authored-name collision errors. Extend existing canonical-name, nullable, raw-body/multipart and multi-media tests with recursion, reversed traversal and generated-name collisions; assert every operation points to the correct complete schema.
- [ ] Verify #574's present-string codec/length equivalence with actual generated HTTP/JSON-RPC decoders and independent schema validators, including an ECMA-262 engine. Cover unconstrained and negative/effectively empty ranges, residues 0/1/2, huge/overflow bounds, named/inherited/nullable Bytes, collections, raw/custom exclusions, pad-bit aliases and every rejected whitespace/terminator form. Assert problem details, service invocation and decoded value. Keep schema enum membership and runtime union decoding as independent obligations; include String `aGl=` versus Bytes Enum(`hi`) as a negative uniqueness control. Do not claim enum acceptance equivalence from the length rule.
- [ ] For #574, run focused expr/IR/render tests, `make value-contract-conformance`, `make openapi-contract`, `make generated-code-quality`, the affected testdata compile corpus, `make lint`, `make test` and `make coverage-ratchet`. Compare parent/candidate fixtures and new probes twice per revision in isolated processes; enumerate grammar, metadata and necessary component-reference/name differences while proving runtime code and dereferenced excluded contracts unchanged. Obtain independent exact-diff review, resolve every finding, then commit/push. The runtime/schema Design Blocker is discharged only by this production evidence, not by the candidate model alone.

### Milestone 3: Shared enum/default semantics and inline schemas

Goal: Remove another source of conflicting interpretation while preserving valid
runtime acceptance and rejecting lossy contract values deterministically.

Acceptance Criteria

- `expr` inline schema values and shared enum comparison use the same declared
  scalar semantics as examples, including Bytes/String/Any and empty bytes.
- #456 has a deterministic path-qualified map-key collision failure; no insertion
  order chooses a winner. Valid collection enums/defaults keep their runtime behavior.
- Invalid enums/defaults cannot disappear under documentation-example omission.

Checklist

- [ ] Add failing seam tests in `internal/enumvalue/value_test.go`, the inline-schema tests in `expr`, and shared validation tests in `codegen`, covering mixed keys, nested enum/default collections, typed nil versus empty values, nullable null and authored/wire-name aliases. Exercise both collision insertion orders and isolated processes.
- [ ] Migrate `internal/enumvalue/value.go`, `expr/json_schema_inline.go`, and `codegen/validation_render.go` to the semantic API; retain `internal/jsonkey/key.go` spelling ownership. Put the #456-specific failure and regression in its own atomic ticket commit after shared prerequisites.
- [ ] Run `go test ./expr ./internal/enumvalue ./codegen ./codegen/cli` and the revision comparison for every proposed commit; compare emitted validators as well as schema enums/defaults. A newly exposed runtime/schema mismatch is a Design Blocker.
- [ ] Run `go fmt ./...`, `make lint`, `make test`, `make coverage-ratchet`, and `make openapi-contract`; obtain independent exact-diff review for each atomic commit, resolve findings, and commit/push. Close #456 only when all inventoried consumers reject the collision under the contract.

### Milestone 4: OpenAPI uses the shared pipeline

Goal: Keep OpenAPI responsible for visibility and schema validity while removing
its private synthesis, coercion and branch-reselection machinery.

Acceptance Criteria

- Ordinary emitted examples validate against their actual emitted schema in
  OpenAPI 3.1 and 3.2, JSON and YAML; custom/external examples retain their boundary.
- Untagged bodies remain bare. Nested selected branches survive; authored ambiguity
  is omitted with the specified diagnostic rather than guessed.
- Suppressed/excluded examples introduce no new errors, while reachable invalid
  enums/defaults still fail. Documentation-only bodies use schema/media proofs.
- The private OpenAPI graph-copy synthesis path is removed after replacement.

Checklist

- [ ] Extend `http/codegen/openapi/internal/ir/byte_values_test.go`, `document_test.go`, `nested_union_examples_test.go`, and `untagged_byte_examples_test.go` with failing structural assertions for the design's precedence, presence, custom-codec and visibility matrix. Extend the existing mapped-names/type-identity specimen designs for rendered coverage; preserve `RawRequestBodyOpenAPIDSL` behavior.
- [ ] Add the planned `http/codegen/openapi/v3/example_contract_test.go` with `TestRenderedExamplesValidateAgainstSchemas`. Independently validate every ordinary emitted example in the probe/specimen matrix against its actual referenced schema, across OpenAPI 3.1/3.2 and JSON/YAML, including documentation-only bodies and declared media encoding. Use a standards-based instance validator supporting the emitted JSON Schema dialect; explain and pin any added test-only dependency. Keep custom/external/serialized examples in their explicit compatibility matrix, not falsely counted as ordinary JSON instances. Make validation failures fatal and report specimen, document path, schema path and example path; include a deliberately invalid instance that proves rejection. Wire this test into `make openapi-contract` rather than treating parser or warning-only lint success as instance validation.
- [ ] Migrate `analyzer.go`, `document.go`, `document_examples.go`, `document_cookies.go`, and `example_generation.go` to shared resolved values/projection. Keep actual emitted-schema validation, location codecs and target visibility in IR. Keep `http/codegen/openapi/v3/example.go` limited to rendering/metadata; remove obsolete private synthesis only after its consumers migrate.
- [ ] Run `go test ./http/codegen/openapi/v3 -run '^TestRenderedExamplesValidateAgainstSchemas$' -count=1`, then `go test ./http/codegen/openapi/internal/ir ./http/codegen/openapi/v3 ./internal/openapiimport`, then `make openapi-contract` for libopenapi, Redocly and consumer smoke checks. Assert #302's supported bare-body contract remains intact.
- [ ] Run the revision comparison for affected fixtures/testdata plus new probes, twice per revision in separate processes; enumerate every schema/example/diagnostic difference. Run `make generated-code-quality` and `make test-testdata-compile TESTDATA_RUN='TestDesigns/http/codegen/'`.
- [ ] Run `go fmt ./...`, `make lint`, `make test`, and `make coverage-ratchet`; obtain independent exact-diff review, resolve findings, and commit/push the OpenAPI migration ticket.

### Milestone 5: HTTP and JSON-RPC consume executable examples

Goal: Make advertised JSON body examples and defaults agree with the builders
that users actually execute, while preserving plain-flag codecs.

Acceptance Criteria

- #565's advertised byte example passes the generated CLI builder and yields the
  authored bytes; binary/empty/nested byte cases also pass.
- HTTP and JSON-RPC body defaults retain their existing valid behavior. Raw byte,
  path, header, cookie, form and custom text flags retain their existing syntax.
- Partial/ambiguous/unrepresentable examples produce the specified omitted hint
  and diagnostic without disabling otherwise valid commands.
- Payload/result/error/stream records preserve source and occurrence identity;
  generated service signatures and DTO presence/pointer shapes do not change.

Checklist

- [ ] Extend `http/codegen/mapped_names_openapi_test.go` and `mapped_names_test.go` with the failing advertised-example-to-generated-builder regression for #565. Extend HTTP and JSON-RPC `collection_defaults_cli_test.go` with nested bytes, mapped fields and omitted/null/empty cases; add raw flag controls to CLI tests. Extend `codegen/cli/usage_test.go` and generated CLI tests to prove unusable examples disappear from payload diagnostics, per-command help and aggregate usage while the same command still accepts valid supplied input.
- [ ] Consume the milestone-2 semantic carriers in all HTTP service-data files listed in the inventory. Reconcile `websocket.go`, routes, auth/header/cookie arguments and `misc_sections.go` explicitly; preserve location-specific formatting.
- [ ] Migrate body/default routing in `http/codegen/client_cli.go` and `codegen/cli/{cli.go,payload.go,conversion.go,command_data.go,flag_parsing.go,usage.go}`; make unusable example outcomes explicit in flag/command data and suppress empty example sections in both usage paths. Keep plain flags on their existing codecs. Separate shared-carrier prerequisite work from the complete #565 ticket commit.
- [ ] Run `go test ./codegen/service ./codegen/cli ./http/codegen ./jsonrpc/codegen`; require `TestMappedNamesGeneratedModule`, `TestCollectionBodyDefaultsGeneratedCLI` in both transports and `TestScalarMapDefaultsGeneratedCLI` to execute, not skip due to source configuration.
- [ ] Run the revision comparison including generated error/stream surfaces and both checked-in ticktock fixtures; regenerate fixtures through the CLI. Run `make generated-code-quality`, `make test-testdata-compile TESTDATA_RUN='TestDesigns/(http|jsonrpc)/codegen/'`, and `make integration-test`.
- [ ] Run `go fmt ./...`, `make lint`, `make test`, and `make coverage-ratchet`; obtain independent exact-diff review per ticket, resolve findings, and commit/push. Close #565 with its generated-builder evidence.

### Milestone 6: Protobuf projection and migration completion

Goal: Preserve authored service values through protobuf representation and remove
the remaining built-in semantic bypasses.

Acceptance Criteria

- #434 retains authored values with and without unions, including payload-local
  examples; nested oneofs and protobuf field names survive actual decoding and
  service conversion. Out-of-range values cannot become advertised valid hints.
- The inventory has a final disposition for every match; no built-in projector
  independently synthesizes a replacement or reselects a union branch.
- Lean proof/axiom gates, bounded TLA+ checks, Go/Lean conformance, property tests,
  independent schema validation, generated
  round trips, process-isolated byte comparisons and repository gates all pass.
- Public docs describe example omissions/errors and compatibility boundaries;
  framework guidance names the single semantic owner.

Checklist

- [ ] Add failing authored-value/provenance assertions to `grpc/codegen/client_cli_protojson_test.go` and `client_cli_protojson_corpus_test.go`; extend `testdata/dsls_15.go` with union-free and nested-union controls, mapped names, omitted optionals, byte cases and protobuf integer boundaries.
- [ ] Migrate `grpc/codegen/service_data_analysis.go`, `service_data_helpers.go`, `service_data_convert.go`, `client_cli_example.go`, and `client_cli.go` to shared semantic input plus existing protobuf allocation/wrapper mapping. Remove `protoJSONExample` resynthesis; retain target projection and final protojson validation. Keep the complete #434 fix in one ticket commit.
- [ ] Run `go test ./grpc/codegen -run 'TestClientCLI|TestGeneratedClientCLI' -count=1`; assert the exact authored fields and selected branches after builder decoding and service conversion, not just parse success. Run `make test-testdata-compile TESTDATA_RUN='TestDesigns/grpc/codegen/'` and the revision comparison.
- [ ] Extend `internal/valuecontract/conformance_test.go` and the Lean correspondence ledger to the actual protobuf projection, including independently specified expected field/wrapper plans, limits, oneofs, selected-body/view observations and provenance. Pair reference comparisons with actual protojson decoding and service conversion; record plan derivation and runtime codecs as tested boundaries. Run `make value-contract-conformance` and require every migrated owner to have named cases.
- [ ] Add bounded property/fuzz tests in `expr/resolved_value_test.go` and the generated probe harness for projection round trips under target-specific equivalence; keep implementation and oracle independent. Run deterministic seeds in ordinary tests and `go test ./expr -run '^$' -fuzz '^FuzzResolvedValueProjection$' -fuzztime=60s`; commit minimized failures as regressions. The fuzz target is a planned addition in this milestone.
- [ ] Re-run both inventory searches, classify every match, and remove obsolete built-in coercion/selection helpers after proving no caller remains. Retain documented public compatibility adapters and explicit custom boundaries. Record each final disposition and its regression test in the inventory.
- [ ] Update the Examples and OneOf sections in `docs/dsl-reference.md` to explain authored ambiguity, incomplete examples and path-qualified invalid-value errors; update `.agents/skills/loom-framework/SKILL.md` and its `references/repo-map.md` with ownership. Update the consumer `loom` skill only for the documented author-visible changes.
- [ ] Review the final `expr/lean/value_projection/correspondence.md` claim-by-claim against theorem names, axiom logs, conformance tests and external assumptions. Reject any claim that generated Go or handwritten Go refinement is proved. Run checked TLC, `go fmt ./...`, `make ci-local` (now includes Lean/conformance and integration tests); run the final comparison against the pre-repair baseline and enumerate the complete intended-difference manifest. Full compile corpus failures follow `internal/testdatacompile/README.md`; do not lower coverage floors or invent exclusions.
- [ ] Obtain independent exact-diff review per remaining atomic commit, resolve findings, commit/push, then run relevant temp-module probes against the pushed commit in remote mode. Close #434 and completed dependency tickets with proof links; leave broader tickets open unless their entire scope is independently satisfied. Move durable contract guidance to maintainer docs and remove completed roadmap material at completion.
