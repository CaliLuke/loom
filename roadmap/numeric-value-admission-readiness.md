# Numeric value admission readiness — #685

Status: approved readiness implemented; production acceptance is complete and awaiting independent final review. No commit has been made.

Parent: `66a9dc9090a06cd4136be1e5ebdaf26883ff90fa`.

## Scope and sequence

1. Reproduce typed default and enum range failures at the exact parent; classify examples and generated artifacts independently.
2. Identify the common typed scalar admission owner, its concrete source-type rules, aliases, recursive shapes and opaque-codec boundary.
3. Inspect representative consuming designs and the existing value-contract model; model the current omission and proposed admission rule before implementation.
4. Add bounded failing seam tests and durable generation inputs; retain accepted boundary and unsupported-source controls.
5. Specify the shared repair, consumer obligations and a bounded acceptance matrix; freeze an exact readiness manifest for independent review.

The scope is authored numeric **values**, not exact numeric **bounds**. `Int32` with `Maximum(1e12)` is a valid redundant predicate: an out-of-carrier bound does not imply an unrepresentable value. #681's exact-number owner and source-versus-target contract remain in force. #692 collection literal rendering is separately owned.

## Existing contract and initial evidence

`roadmap/value-contract-design.md` requires primitive source admission before normalization, preserves the concrete Go source-type matrix, and states that integer boundaries and floating rounding follow the declared type. Declared float values retain authored-literal formatting followed by destination parsing; this differs intentionally from exact Go float **bound** meaning.

The issue's four original invalid `Int32`/`UInt` default and enum cases still pass design evaluation at this parent. `Primitive.IsCompatible` admits broad integer source kinds without checking the destination range. `valueResolver.scalar` checks that source matrix, but its integer branch retains the raw value without checking the declared carrier. Float normalization already parses at the destination width; its role coverage needs direct verification before calling it a defect.

## Completed classification at the declared parent

Readiness environment: Go `1.27.2`, Darwin/arm64; Lean `4.34.1` from the owning pinned toolchain. At this checkpoint no production files had changed.

`go test ./expr ./dsl -run 'TestNumericValue(Carrier|Design|Redundant|Partial)' -count=1 -json` produces **53 failing leaf cases and 67 passing leaf controls**. The failures are intentional readiness tests, not an implementation acceptance result:

| Boundary | Parent result | Required correction |
| --- | --- | --- |
| Int32 overflow/underflow; negative UInt/UInt32/UInt64; UInt32 overflow; UInt64-to-Int64 overflow | Primitive compatibility and all three resolved roles accept | Reject before any artifact is rendered |
| Numeric alias, array, object member, map value, map key | Recursive typed Resolve accepts overflow in all roles | Preserve recursion; reject at the scalar leaf with its path |
| Float32 `1e39` | Typed Resolve and default/enum design admission reject; primitive compatibility and direct Example admission accept | Share existing finite destination normalization with primitive compatibility |
| Float32 maximum, ordinary rounding (including just above exact Float32 maximum), underflow to zero | Typed Resolve accepts | Preserve existing float literal normalization |
| Float64 infinity | Typed Resolve rejects; primitive compatibility accepts | Preserve rejection in the shared admission owner |
| Known invalid numeric sibling beside opaque codec | Declared-shape resolution reports Unsupported without checkable failure | Retain Unsupported and known failure; never execute the codec |
| Valid sibling beside opaque codec | Unsupported without known failure; codec not invoked | Preserve |
| Exact integer limits, unsigned zero, Any wide integer | Accepted | Preserve |
| Named scalar source; int64 source for Int32 even when value is 1 | Rejected | Preserve concrete source matrix |
| Int32 `Maximum(1e12)` and `Minimum(-1e12)` with representable values | Accepted | Preserve redundant bounds |
| Partial object Example, including missing required child or out-of-range child | Design evaluation accepts | Preserve existing optional documentation contract |

The last row is intentionally separate from explicit typed `Resolve(..., ValueRoleExample)`, which requires membership. The orchestrator confirmed that #685 must not make optional object examples complete membership assertions. Primitive numeric example checks change only where the existing compatibility path already checks the primitive. No new nested-example traversal is authorized.

The four original `Default`/`Enum` repros are retained in `expr/lean/value_projection/numeric_value_admission_probe.go.txt`. Copy it to `design/design.go` in a temporary module; select `LOOM_NUMERIC_CASE` from its header. At the exact parent, all four `loom gen` commands return 0, and `go test ./gen/...` returns 1:

- `int32-default`: HTTP client CLI/types and server types assign `1099511627776` to `int32` (overflow).
- `uint-default`: client/server assignments of `-5` to `uint` overflow.
- `int32-enum`: server validation compares an `int32` with overflowing `1099511627776`.
- `uint-enum`: server validation compares a `uint` with negative `-5`.
- `boundary-control`: MaxInt32 default plus redundant maximum generates and compiles successfully.

Temporary logs: `/tmp/loom-685-generation-parent/*.log`; direct test JSON: `/tmp/loom-685-readiness-tests.json`. These outputs are disposable evidence; the source specimens and this summary are durable. The temporary module declares the verified pushed requirement `github.com/CaliLuke/loom v1.10.0-alpha.9.0.20261012214204-66a9dc9090a0` and replaces it with this exact local parent for candidate comparison. Remote reproduction removes the replacement. Independent readiness review found and corrected an initial `v0.0.0` placeholder in that retained module. `go mod graph` before/after differs only in the masked Loom requirement; `go list -m -json` confirms the same local source and dependency graph. Thus the existing five generation/compile outcomes remain valid without rerunning generation. Pin evidence: `/tmp/loom-685-pin-repair.json`.

## Shared owner proposal

Keep `Primitive.IsCompatible` as the public admission surface, backed by one private primitive admission/normalization function in `expr`. Move the existing concrete source-type table into that owner, followed by destination representability; do not introduce a separate renderer policy or silently widen source eligibility through reflection. `valueResolver.scalar` consumes the normalized result from the same owner instead of reimplementing numeric conversion.

For integer destinations, after source admission, preserve the exact source integer and compare against `numericDomain(kind).Limits()` using `internal/exactnum`. `Int` and `UInt` use `strconv.IntSize`, exactly as the bound projection owner already does. No float64 conversion, text round trip, saturation, wrapping, or new tolerance is allowed. A negative signed source cannot inhabit an unsigned carrier; a wide unsigned source cannot inhabit a smaller signed carrier. Normalized integer source representation need not change merely to prove fit.

For float destinations, move the existing finite check and `strconv.ParseFloat(fmt.Sprint(raw), destinationBits)` behavior unchanged. This intentionally admits rounding and underflow and rejects overflow/nonfinite values. Comparing the exact unrounded float source to a carrier maximum would be a different policy at the rounding edge; do not substitute it. Source formatting remains after concrete admission, so named values cannot gain entry through String methods.

Existing consumers then derive the correction:

1. DSL Default/Enum early compatibility checks reject known invalid primitives; late effective-constraint validation still catches aliases, deferred types and nested authored values with declaration provenance.
2. Shared Resolve handles every typed role, aliases and recursive data; map keys use the same scalar path before key identity and serialization.
3. Optional example compatibility and legacy union selection consume `IsCompatible`; retain their existing structural/partial contracts and do not replace them with full Resolve.
4. Required default/enum projection keeps #681's failure contract and declaration ownership. It must never become the first range check, silently drop an authored value, or let an opaque sibling conceal a known builtin failure.
5. Generators receive admitted values. No transport, CLI, OpenAPI renderer or literal helper needs its own range table.

The helper remains private; this adds no DSL syntax. Keep nonnumeric compatibility and codec handling unchanged. Opaque custom codecs remain Unsupported at the established boundary, without invoking user methods; known builtin siblings retain checkable failures. `Any` has no declared numeric carrier and keeps its separate finite-value/raw-host contract.

## Adoption evidence

Read-only inventory of active `loom-mcp` at `f6aa6cfd24eb7c48ba22961125a67fe6d95f2c28` found concrete representable uses:

- `registry/design/design.go`: `RegisterResult.lease_duration_ms` is Int64 with Example(120000); `DrainProviderPayload.settlement_duration_ms` is Int64 with Example(30000) and inherited object shape. Both have application-owned positive lease bounds.
- `integration_tests/fixtures/assistant/design/design.go`: Int search limit Default(50), another Int Default(10), bounded Float64 ratio, ArrayOf(Float64), and UInt64 payload/result fields.
- No narrow Int32/Float32 default or enum violation was found in this bounded inventory. This is not a claim about every downstream design.

These existing uses should remain byte-identical after regeneration. The architectural risk is general: invalid design literals currently reach emitted Go across multiple consumers, as the durable HTTP probe proves; no application-specific exception is proposed.

## Model, counterexample and correspondence

The approved readiness model reproduced the former integer omission and modeled
source admission conjoined with carrier membership. Implementation migrated that
rule into the existing `ScalarSemantics.coerceSourceScalar` / `SourceCoerces`
owner and reused `ScalarRules.integerFormat`; the temporary readiness resolver
was removed. `ValueContract/NumericValueAdmission.lean` retains the old source-only
counterexample plus six corrected-owner claims. All seven are registered in the
existing theorem manifest, Proofs entry point and axiom audit.

Run `make value-contract-conformance` from the repository root. The proof script
also requires `NegativeNumericValueAdmission.lean` to fail at its false assertion
that source eligibility implies carrier membership. The independent production
graph extractor supplies integer formats; 15 additional cases across three roles
compare actual resolution with the proved reference. The full existing 18-group
conformance gate passes.

Existing `NumericCoercionControls` covers destination literal parsing under codec
assumptions; `NumericTargetControls` covers decoder width and schema/runtime
separation. JSON Schema raw integer membership remains independent of finite
native width. None of these proofs establishes reflection, native host-width
extraction, exactnum's Go arithmetic, float parsing/formatting, diagnostics,
generated Go or custom codecs. Tests independently extract source/destination
facts; added expected results do not call production compatibility. Oversized
machine-source tests skip on 32-bit hosts, while fixed-width UInt32-to-Int32
checks remain portable. The generation witness retains the issue's original
64-bit `int(1<<40)` source.

## Bounded production acceptance plan

1. Keep all readiness failures meaningful and turn them green. Add direct source-attributed effective/default/enum tests for named aliases, deferred type resolution, recursive object/array/map values and keys; include inherited enum predicates and replacement-default controls from the existing owner.
2. Extend independent numeric-source conformance with signed/unsigned exact limits, one-step overflow, source-type rejection, host32/host64 extraction controls, Float32 overflow/rounding/underflow, and opaque sibling failure. Update the existing source proof/reference owner and theorem manifest, run affected proof/conformance gates and deliberate negative controls.
3. Regenerate the five durable HTTP cases at parent and candidate. Invalid cases must stop with design diagnostics, before files are installed. The valid boundary design must be byte-identical and compile; repeat its generation in a separate process for determinism. Keep emitted OpenAPI parse/lint checks on this specimen.
4. Use one existing valid numeric fixture per relevant transport family (HTTP, JSON-RPC, gRPC), including default/enum/Float32 paths. Compare affected output bytes and compile generated Go. Add one rejected numeric design per distinct generator entry path to prove none bypasses evaluated admission. Keep this bounded; no full exported-design audit.
5. Run direct affected expr/dsl and shared codegen tests, source-admission/legacy-selection controls and existing optional-example checks. Exercise the representative consuming Int/Int64 values above without changing their application policy.
6. Intended output differences: invalid numeric declarations now fail early with source/role/path context; primitive-invalid examples reject wherever primitive compatibility was already required. Valid normalized values and unrelated generated files stay unchanged. Any additional output drift needs classification before review.
7. Freeze the exact final production diff and evidence for an independent reviewer. No commit is authorized by this readiness record.

## Readiness completion and cleanup

- [x] Direct role/carrier classification and controls.
- [x] Exact-parent generated failure/control outcomes.
- [x] Bounded actual consuming-design inventory.
- [x] Current model counterexample and proposed invariant, checked by Lean.
- [x] Concrete owner and bounded acceptance plan.
- [x] Independent readiness approval: manifest `d940f6ce541322f6c62dda81df15cbe52fc0e2b78da0b1baeec37f642f74dcbe`, no unresolved findings.

Old #681 ignored `.lake` output was removed after checking that no process referenced either old numeric worktree. Managed archival was attempted: `exact-numeric-bounds` refused because another task owns it; `numeric-comparison` refused because a pinned task/workspace protects it. Both checkouts were left intact. Root removed the 147 delivered #681 temporary files and six temporary directories after confirming no active #685 dependency; cleanup manifest `/tmp/loom-correctness-cleanup-numeric.json`. Committed #798/#799 durable probes remain untouched.


## Production acceptance evidence

The final owner uses exact integer comparisons against `numericDomain.Limits`,
with no renderer checks. Primitive compatibility and typed resolution share the
same finite float normalization. The expected behavioral difference is early
rejection of invalid numeric values; exact bounds and representable values retain
their contracts. The former projection-overflow test now asserts invalid source
resolution followed by an unrepresentable projection.

Verified on Go 1.27.2 Darwin/arm64 and Lean 4.34.1:

| Check | Result and scope |
| --- | --- |
| expr, dsl, internal/valuecontract package suites | Pass; new direct regressions, source matrix, role, nesting, deferred default and partial-example controls |
| Shared codegen package | Pass |
| Generated wire tests | HTTP selected-body defaults and collection enums; JSON-RPC/gRPC effective constraints modules pass, including their compile/runtime assertions |
| Proof gate | 672 required claims plus 29 existing collision claims pass audit and fresh kernel replay; deliberate negative controls fail as expected |
| Independent conformance | All 18 registered groups pass; 45 added integer/float source carrier role comparisons |
| Bounded generated comparison | Existing `ExactNumericPayload` across HTTP, JSON-RPC and gRPC: 16/14/15 files byte-identical at parent/candidate and a second isolated candidate generation; all parent/candidate generated Go compiles |
| Rejected transport entry points | Each of HTTP, JSON-RPC and gRPC rejects invalid Int32 default during DSL evaluation, before creating gen/ |
| Original issue cases | All four default/enum cases now fail design evaluation with zero generated files; parent generated code fails compilation |
| Redundant-bound boundary control | MaxInt32 default with Maximum(1e12) compiles and all 16 files are byte-identical to retained parent output |
| OpenAPI | Numeric HTTP specimen and redundant-bound control JSON/YAML parse with pinned libopenapi and pass repository-pinned Vacuum rules |
| Focused vet | expr, dsl, internal/valuecontract pass |

The durable transport driver is `numeric_value_transport_probe.go.txt`, which
reuses `codegen/testdata.ExactNumericPayload` and takes `LOOM_NUMERIC_TRANSPORT`
(http/jsonrpc/grpc) and optional `LOOM_NUMERIC_INVALID=1`. Temporary modules use the
verified pushed pseudo-version above with explicit parent/candidate replacements.
The read-only exact parent is a Git archive at `/tmp/loom-685-parent-source`.
The specimen has defaults/enums, Float32/Float64, Int32/Int64/UInt64, named aliases,
Extend and exact/exclusive/extreme bounds. The HTTP wire fixture adds selected
body defaults; the collection fixture adds nested enum normalization.

Temporary evidence is under `/tmp/loom-685-*`: `transport-matrix-v2/results.json`
contains file hashes and process statuses; `original-candidate/results.json`
contains the original rejection/control results. Logs are `direct.log`,
`final-direct.log`, `generated-wire.log`, `proof-gate.log`, `conformance.log`,
`openapi.log` and `vet.log`. An initial transport harness put Type declarations
inside Method and failed DSL scope validation on both revisions; the corrected
durable driver declares the shared specimen at package scope. Only v2 results
are acceptance evidence.

`codegen/service` has one unrelated pre-existing fixture failure:
`TestSecureEndpointInit/endpoints-with-requirements` binds Basic and JWT to the
same Authorization header. The exact parent reproduces the same diagnostic
(`/tmp/loom-685-parent-security.log`). The orchestrator owns its separate fixture
repair; no #685 source change compensates for it.

- [x] Implementation and bounded acceptance.
- [x] Independent isolated final review: `/root/review_value_admission_readiness`, manifest `9679bc649d67e6ce9fccff0af47c8ba2c243ba09dd0b24b5764d58c99da0fe8f`, no findings.
- [ ] Integrated final review, commit, push and issue reconciliation (orchestrator-owned).

Required guide integration: public `docs/dsl-reference.md` documents this change.
Root integration adds the shared-owner and authored-value admission guidance
to the framework/consumer skills and formal-model index. No reviewed production
file needed a merge adjustment. Main also contains the separately reviewed
security fixture correction `8aec9136a`; its endpoint golden remains unchanged.

Integrated on main parent `8aec9136a`: full expr, DSL, value-contract and service
package suites pass, including the separately corrected security fixture.
Evidence: `/tmp/loom-685-integrated-packages.log`. Shared guidance and tracking
are the only integration additions; production source hashes match the reviewed
22-file manifest.
