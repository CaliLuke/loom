# Exact numeric bounds readiness

## Implementation checkpoint

Final isolated review approved all 168 files against `27da0247ecbb766e0dfef8a6de33f54193368837`.
Manifest SHA-256: `c79fc05f1298ef6503a8e8d54c14abdfbde877fc9e7e6d69f19a5f3451489c42`.
Reviewer `/root/review_numeric_final` confirmed all four medium findings fixed,
including alias-to-sequence merge values. Integration onto `ef0205d71` merges
`docs/dsl-reference.md`, `docs/openapi-import-coverage.md` and
`internal/openapiimport/render_schema.go`; all other candidate bytes are unchanged.
Shared skill/index guidance and live trackers are root-owned additions. Combined
boundary checks pass; exact integrated-diff review remains required before delivery.

Integrated acceptance: root-module compile-only check passes. Focused numeric,
projected validation, wire-binding, custom multipart and recursive-import checks
pass across expr, service, HTTP, gRPC, JSON-RPC and importer packages (45.374s,
57.669s, 4.975s, 4.538s and 76.155s for the substantive package runs).
The ordinary and exact HTTP generation inputs were reused against main's prior
production source at `d57f96c6` (the later `ef0205d71` changes only trackers), the
integrated source and a second independent process. Ordinary generation/build/vet
passes at both versions; exactly six artifacts differ as recorded above. The
exact input still fails at the parent and builds/vets at the candidate. All 32
candidate artifacts match both their repeated output and the previously reviewed
isolated output byte for byte. Evidence: `/tmp/loom-numeric-integrated-generation/report.json`,
`/tmp/loom-numeric-integrated-boundaries.log` and `-compile.log`. Prior proof,
conformance and other transport evidence covers unchanged implementation bytes;
no exhaustive corpus audit was added.

The nineteen-file v3 readiness manifest
`e369c61af21823c15d07f7570cfe55fa18246f61fc906dad44ee2e8ad9e58f3b`
passed independent review. Root authorized implementation under these obligations.
The isolated worktree advanced without rewriting files to committed parent
`27da0247e`; all nineteen readiness hashes were verified unchanged afterward.

- [x] Add the immutable common numeric owner with direct admission, comparison,
  canonical output, integrality and ownership tests, reusing the modeled algorithm.
- [x] Migrate DSL and expression bounds, effective composition and value admission.
- [x] Migrate runtime/prepared-schema membership and fatal evaluator errors.
- [x] Derive generated typed thresholds, example sampling and schema serialization.
- [x] Repair source-specific OpenAPI import and unsupported-DSL diagnostics.
- [x] Run the bounded generated/runtime/proof acceptance matrix and freeze the
  final exact diff for independent review. Commit and push remain root-owned.

The historical readiness status and evidence below refer to the preimplementation
baseline. Implementation and bounded acceptance are complete; independent implementation
review remains required. The required-annotation bypass discovered during
generated-artifact acceptance is included in this diff.

### Independent review repair sequence

The first implementation review found three medium issues. Commit remains gated.

1. Resolve effective YAML mapping keys before flattening, retaining whole-subtree
   replacement and first-wins merge-sequence precedence; add source and importer regressions.
2. Normalize admitted YAML float spellings before the strict JSON-number owner;
   retain quoted/string rejection and exact exponent meaning.
3. Retain inherited enum declaration identity through nested refinement and
   shared value-plan preparation; preserve malformed/opaque rejection controls.
4. Rerun affected tests, model correspondence and bounded generated comparisons;
   freeze the revised manifest for the same reviewer's bounded re-review.

The three findings are repaired and the revised diff awaits bounded re-review.
Effective YAML mappings now select whole values before flattening. YAML-tagged
floats normalize leading zeroes before strict exact JSON parsing; JSON and quoted
string admission remain separate. Enum clauses retain the declaring occurrence
and bind through the existing plan builder to target representation policies.
Later predicates do not erase constants, and native enum comparison translates
projected constants to target identities before comparing. Tests cover renamed
fields as well as unchanged field names, preserve `{n:1},{n:2}`, and enforce the
final conjunction (reject 1 and 3, accept 2).

A subsequent bounded re-review confirmed those fixes and found one remaining
YAML consequence: aliases to merge sequences were dispatched as mappings.
The narrow repair adds source/import regressions and resolves aliases before
merge-value shape dispatch. Cycle detection, malformed nested-sequence rejection,
first-wins precedence and explicit subtree replacement remain covered. The
reviewer's original overlay now passes; focused source/admission and exact
import/generate/export checks pass, as does importer lint. Final narrow logs:
`/tmp/loom-numeric-alias-sequence-tests.log`, `-reviewer.log` and `-lint.log`.
No numeric arithmetic, value-plan, generated transport or proof source changed;
their prior green evidence remains applicable. The revised three-file delta is
frozen for bounded re-review.

Rerun evidence: `review-import`, `review-packages3`, `review-transports2`,
`review-enum-target3`, `review-enum-openapi4`, `review-conformance`, `review-proof`
and `review-lint2` under `/tmp/loom-numeric-` pass. The source-normalization
extraction was subsequently checked in `review-yaml2`; it changes no output.
All five previously compiled generation specimens remain byte-identical after
repair and repeat identically in independent processes (`repair-report.json`).
The nested-enum OpenAPI specimen fails at the exact parent before emission and
now renders checked JSON/YAML retaining both constants; its independent-process
comparison is under `/tmp/loom-numeric-nested-spec/`.

#### Explicitly excluded, parent-proven generator defects

Both probes are durable under `expr/lean/value_projection/`. To reproduce, copy
one `.go.txt` input to `design/design.go` in a temporary module, replace Loom with
the committed parent `27da0247ecbb766e0dfef8a6de33f54193368837`, run `loom gen`
for that design import, then `go vet -mod=mod ./...` when generation succeeds.

- [#799](https://github.com/CaliLuke/loom/issues/799), `numeric_nested_enum_http_probe.go.txt`: a direct HTTP payload with inherited
  object enum and child `Minimum(2)` fails at the parent during `eval.RunDSL`:
  `enum member map[string]interface {}{"n":1} declared by "ReadRequestBody" violates its declared type`.
  Candidate fails at the same pre-OpenAPI body-preparation boundary. The nested
  component specimen and shared/inline/runtime tests do not claim to repair it.
  Parent log: `/tmp/loom-numeric-nested-http-parent/generate.log`.
- [#798](https://github.com/CaliLuke/loom/issues/798), `numeric_object_enum_generator_probe.go.txt`: an admissible child `Minimum(0)`
  isolates a second parent defect. Generation succeeds, then vet reports
  `gen/jsonrpc/numbers/server/types.go:41:16: invalid operation: actual != nil (mismatched types EnumRefinedRequestBody and untyped nil)`.
  Parent log: `/tmp/loom-numeric-object-enum-parent/vet.log`.

Root filed both residuals separately with their exact-parent reproductions. They are not hidden
by the successful original numeric transport matrix and are not part of #681.

### Current acceptance evidence

Source is the isolated worktree at `27da0247e`, using Go 1.27.2. Direct DSL,
effective composition, numeric owner, runtime, codegen, importer and OpenAPI
checks pass, including the annotation extension. The full root module passes
a compile-only check, and affected-package golangci-lint reports zero issues.
The independent reference adapter passes all eighteen registered conformance
groups, including four new exact-bound alias controls. The proof gate passes
665 registered owner claims and 29 historical issue-456 claims, negative controls
and fresh kernel replay. The five annotation/identity controls are registered in
that same gate.

The bounded generated matrix exercised eighteen neighboring-boundary cases in
each HTTP, gRPC and JSON-RPC transport, asserting status/problem code and service
invocation. Generated untagged array unions cover tiny positive bounds in both
branch orders. CLI and service code compile for all three transports. JSON and
YAML OpenAPI outputs pass libopenapi and pinned Vacuum. Exact import-to-design-
to-generated-export round trips cover all four bounds, YAML radix integers and
3.0 boolean exclusives.

Independent-process generation compared the parent with this worktree. The
ordinary numeric HTTP design compiles and vets at both versions; ten files are
byte-identical and six intentionally differ: two CLI/example-help files, server
validators, JSON/YAML schemas and the manifest fingerprint. Differences are exact
bound spelling, directed exclusive thresholds, string-valued range diagnostics
and valid samples drawn from the corrected domain. In particular binary `100.1`
exports as `100.099999999999994315658113919198513031005859375`. Exact HTTP,
gRPC and JSON-RPC designs fail at the parent with the false rounded UInt64
contradiction and compile/vet at the candidate. Each candidate repeats byte for
byte in a second process. The exact designs include the decimal default/enum
positive control. A fifth selected design, the existing
`NotGenerateAttributeDSL`, compiles and vets at both versions and repeats
deterministically. Only its JSON/YAML examples and manifest fingerprint change;
all generated Go is byte-identical. Temporary evidence lives under
`/tmp/loom-numeric-generation/`; the managed comparison worktree is
`/Users/luca/.codex/worktrees/numeric-comparison/loom` and must be archived through
the managed lifecycle after review, not manually removed.

### Required annotation owner correction

The mandatory artifact check exposed an existing bypass: OpenAPI defaults and
enums used recursive renderer projection rather than the shared value result.
`Minimum(0.1); Default(0.1)` passes native numeric validation but its shortest
JSON token violates the emitted exact bound. The default must fail generation;
it cannot disappear or substitute another value. OpenAPI examples have their separate
existing suppression contract. Optional inline examples retain their established
partial/ambiguous declared-value behavior; #681 does not turn them into required
full-membership annotations. Their retained numeric values still serialize with
the declared precision. Inline schema defaults/enums had the same bypass.

`ValueContext.PrepareJSONSchemaAnnotations` now extends captured value preparation.
A record retains the declaring occurrence, exact checked projection outcome and
declared host value. Defaults use the final target; authored enum members are
checked at their captured declaring occurrence. All inherited clauses remain
predicates and project through declared shape, so a base enum `{1, 2}` remains
intact under a derived minimum `1.5`. Clause constants do not become candidate
defaults. OpenAPI consumes the checked wire using its existing exact numeric
materializer; inline schema emits the same checked JSON bytes. Neither consumer
reselects an enum union branch.

Opaque custom encoders retain an explicit `ProjectionUnsupported` boundary and
borrowed declared values, preserving the established external-codec contract.
This is not a builtin schema-membership claim. The owner checks every known
builtin subtree before retaining that boundary: an opaque sibling cannot hide
`Minimum(0.1)`'s known raw-JSON failure. Native known failures remain admission
errors; malformed schema evaluator errors remain fatal. Direct regressions cover
both binary/decimal controls and the opaque sibling counterexample.

The shared `internal/jsonkey.Value` owns the preexisting documentary map-key
canonicalization formerly embedded in the renderer. Documentary Any projection
uses it; runtime Any projection keeps actual codec behavior. OpenAPI-only hidden
fields are captured in documentation plan policies, rather than filtered after
value projection. One existing not-generate specimen consequently gains valid
examples; its checked-in goldens are regenerated by their owning test. IR numeric
annotations now retain raw numeric tokens and byte values retain their checked
base64 spelling; these internal representation changes preserve serialized bytes.

All selected gates passed. Raw logs remain task-local under `/tmp/loom-numeric-*`:
`final-packages`, `final-transports`, `annotation-vacuum2`, `annotation-proof`,
`conformance-final`, `compile-final2` and `lint-final3` identify the final runs.
The generated comparison report records generation, compilation, vet and
independent-process byte comparison for all five selected designs.

Independent implementation review is next; integration must retain the separate
#792 importer reference changes delivered after this worktree parent. #685 source
value admission and #692 array literal rendering remain separate issues. No
commit or push is authorized from this subagent. Compiled Lean output is retained only while active checks and
review need it; delete task-owned temporary output after final review.


Status: revised language-neutral architectural candidate; no production edits or
implementation authorization. Owner #681. The user did not accept decimal-only
string syntax. Root directs preserving existing finite hexadecimal DSL strings,
source-specific JSON/YAML admission, and exact shared normalization with no new
arbitrary numeric complexity limit. Independent readiness review remains required.
No further user question is pending. Parent
`651372ce6ff05ed1db495369fd51d48df0c0ec75`; worktree
`/Users/luca/.codex/worktrees/exact-numeric-bounds/loom`. Root reports #755 pushed
as `4f7892a2`; production will start from a current stable parent, while this
worktree preserves the original red baseline.

The first eleven-file readiness snapshot is preserved under
`/tmp/loom-exact-numeric-readiness-baseline` with manifest SHA-256
`065c1ac3379e2a1e35d063be4b2a8aa0bbd919d0fec12a7faf32bfe8f202d769`.
The language-neutral update below adds prototype/proof and actual protocol evidence;
it does not retroactively claim that the first candidate solved those questions.

## Observed ownership failure

The four numeric DSL functions already accept `any`, but
`dsl.parseNumericValidationValue` converts integers and numeric strings to
`float64`. `expr.ValidationExpr` and `EffectiveNumericBound` then retain only
that rounded value. Tight-bound selection, default/example admission, sampling,
Go validation literals, schema hashing and OpenAPI export all consume it.
Changing a Go formatter cannot recover the authored number.

`dsl/numeric_bound_precision_test.go` checks all four operations with integers
above 2^53, signed and unsigned extrema, negative values, decimal strings, and a
small integer control. At the parent, twenty exact-number cases fail and four
small-value controls pass. Run:

```sh
go test ./dsl -run '^TestNumericBoundPreservesAuthoredInteger$' -count=1
```

The test inspects the stored bound through JSON number serialization, before
attribute validation or code generation. It does not yet prove runtime admission,
generated compilation, or kind-range validation. Raw red output is temporarily
at `/tmp/loom-numeric-bounds-red.log`.

## Proposed architectural direction

1. Introduce one immutable exact number in a small internal numeric package below
   both `expr` and runtime `pkg`. Preserve accepted source numeric meaning
   before normalization. Compare endpoints exactly, including inclusive versus
   exclusive ties; do not retain parallel rounded and exact authorities.
2. Make effective constraint selection, copying, schema hashing, example/default
   admission and sampling derive from that representation. Update the existing
   numeric proof correspondence: the abstract exact-number model currently does
   not prove that Go DSL extraction preserves exact integers.
3. Give Go validation and schema renderers explicit target projections from the
   shared number. Integer validators must compile at Int64/UInt64 extrema and
   enforce the authored boundary. JSON/YAML must emit numeric values without
   converting back through float64. Retain ordinary float behavior unless a
   separately recorded contract correction is necessary.
4. Inventory OpenAPI import as a source boundary: the imported schema model also
   uses float64. Determine whether raw numeric nodes remain available before
   normalization, and record a bounded exact import/export case. Do not claim an
   exact round trip if the import boundary already discarded precision.
5. Validate aliases and Extend through the existing effective owner. No
   transport-specific bounds, compensating tolerances, or per-renderer parsing.

Before design approval, settle the exact representation and its finite-number,
string parsing, exponent, negative-zero and float-source semantics against
existing documentation/tests. Integer bounds already require whole numbers;
do not accidentally broaden that contract. A float supplied by application Go
code has already undergone Go rounding; no implementation can reconstruct an
unretained source literal. State that boundary explicitly. Preserve the concrete
source-admission matrix rather than treating every reflected numeric kind as
implicitly supported.

## Acceptance obligations

- Direct DSL/effective tests: all four bounds, 2^53 neighbors, signed/unsigned
  extrema, negative values, decimal-text inputs, inherited tighter bounds,
  inclusive/exclusive equal endpoints, and ordinary integer/float controls.
- Generated HTTP, gRPC and JSON-RPC validators plus CLI controls must compile and
  accept/reject adjacent values correctly. Reuse meaningful constrained numeric
  fixtures; explicitly list which shared and distinct branches they exercise.
- Rendered OpenAPI JSON/YAML must retain exact numeric tokens and pass the pinned
  parser/linter gates. Compare parser raw nodes as needed so a lossy test reader
  cannot conceal the bug.
- Compare a bounded parent/candidate fixture selection byte for byte; list all
  intended literal/schema/example/fingerprint differences. Generate the candidate
  in independent processes, compile generated modules, and reuse unchanged
  evidence. Do not run the full exported-design corpus as a default gate.
- Explore current precision loss and corrected exact endpoint selection using
  the existing `expr/lean/value_projection/ValueContract/NumericBounds.lean`
  owner or a bounded adjacent model before production. Keep the proof's source
  admission and concrete Go/codec limits explicit.

## Related reports and exclusions

#685 concerns kind-range admission of defaults/enums/bounds. Its bound cases
intersect this owner, but source-value admission is a distinct task; neither is
closed solely because the other passes. #692 is a target-literal rendering
defect: array defaults fall back to the supplied Go slice type while map defaults
use the declared type. It needs a separate shared literal-rendering repair.
Presence/default substitution policy remains Milestone 7's separate task.
No public policy, dependency change or repair has shipped from this draft.


## Concrete owner proposal (not yet approved)

Use an immutable `expr.NumericBound` value with private exact state, represented
as a sign, a nonnegative integer coefficient, and arbitrary signed exponents.
Store large integers in immutable canonical text or privately owned `big.Int`
values never mutated after publication. Absence remains a nil pointer in
`ValidationExpr`; the zero value is exact zero. Source eligibility precedes
normalization. `Cmp`, `IsInteger`, exact numeric serialization and explicit
target-domain projection belong to this owner. Any exposed rational or integer
is a fresh copy. Do not retain a float64 cache as an alternate authority.

**Compact decimal core:** an input with decimal coefficient C and exponent E
keeps C and E; it does not allocate 10^abs(E). Remove coefficient trailing zeroes
and add them to E. Compare sign, then `len(abs(C)) + E` using arbitrary-precision
integer arithmetic. If magnitudes tie, compare coefficient digit sequences with
virtual trailing zeroes, without constructing either power. Emission is the exact
numeric token `C e E`, using a deterministic canonical formatting rule. Ordinary
small integers can keep ordinary decimal formatting. A billion-digit exponent's
*value* never asks comparison to allocate a billion zeroes.

**Exact binary inputs and semantic identity:** finite builtin IEEE floats have
bounded dyadic expansion (at most 1074 fractional binary places), so conversion
to the decimal core is bounded. Accepted hexadecimal *strings* have unbounded
underflow exponents, requiring the extension described below. If preserving that
entire grammar, use a common factorized form `sign * C * 2^E2 * 5^E5`, stripping
all factors of 2 and 5 from C. Decimal input contributes its exponent to both E2
and E5; hexadecimal input contributes to E2. Equivalent decimal/hexadecimal
values then share exact identity and schema hashes without expansion. This is
one authority, not separate rounded and exact interpretations.

The factorized representation is the current preservation candidate. Decimal-vs-decimal order
uses the simple compact comparison above. Comparing arbitrarily large opposing
binary/decimal exponents needs an adaptive **outward-rounded interval** algorithm:
compute leading fixed-precision mantissa intervals for powers by squaring, retain
arbitrary-size scale exponents separately, and decide order only when intervals
are disjoint. Increase precision when they overlap; detect exact equality using
the normalized factor tuple first. Never decide from an unproved logarithm or
rounded Float64 estimate. This avoids unconditional huge power allocation, but
has no claimed universal fixed precision/time bound. The readiness prototype and new interval model below establish concrete
feasibility and abstract containment/termination obligations. Actual big.Int and
scaled-interval correspondence remain implementation acceptance obligations.

This proposal adds no dependency. `math/big` provides arithmetic on the compact
coefficients/scales, not an implicit source policy via `big.Rat.SetString`.
Snapshots can copy the immutable bound while preserving occurrence provenance.

Replace all four `*float64` fields in `expr.ValidationExpr`, and the public and
private effective endpoint values, with this value. This is a source API change
for code constructing expressions directly, with a constructor as the migration;
there is no parallel legacy field. The ordinary `Minimum(any)` DSL signatures
remain unchanged. Schema public/IR models use the same exact owner. The runtime
`pkg.JSONShapeRules` holds bound text, but its current `big.Rat.SetString` parser
rejects admitted compact bounds such as `1e-1000001`. Merely changing that text is
insufficient. Descriptor admission, numeric wire matching, integrality and bounds
comparison must use the common internal number. Neither runtime `pkg` nor the
numeric package imports `expr`; `expr` and `pkg` are consumers of one owner.

Specific consumers that must derive from it:

1. DSL extraction; integer whole-number validation; `EffectiveConstraintsFor`;
   schema hashes and semantic equality; default, enum and example checks.
2. #755's retained composition layers and contradiction check (current main
   `expr/attribute_composition.go` compares `lower.Value > upper.Value`). Replace
   those operators with the same exact comparison after #755 is stable. Do not
   change composition ordering, provenance, or its rejection policy.
3. Shared Go validation rendering. Produce a **target-domain bound plan** once,
   with comparison direction, exact typed literal, and an always/never outcome
   when the bound lies outside the representable domain. This prevents an
   untyped UInt64 maximum passed to an `any` error argument from overflowing Go
   `int`. Diagnostic bounds must preserve the authored exact number and must not
   pretend a directed rounded comparison threshold was the authored number.
4. Sampling. Intersect the effective interval with the declared numeric domain;
   sample integer endpoints/deltas using integer arithmetic and float endpoints
   using representable neighbors, then pass through existing exact admission.
   Do not cast an out-of-range float64 into an int64/uint64 to invent an example.
5. Inline schema, JSON shape matching, OpenAPI IR/public schema and JSON/YAML
   emission. Emit numeric tokens, not quoted strings or float64 conversions.
   Semantic hashing uses canonical exact meaning, so equivalent `1`, `1.0`,
   `1e0` declarations do not acquire distinct identities.
   Both `pkg/json_shape.go` and `expr/value_projection_schema.go` must replace
   rational-capacity-dependent numeric decisions. Schema evaluation must expose
   errors separately from ordinary nonmembership through its recursive callers.
6. OpenAPI import. `source.GoLow().Minimum.ValueNode.Value` and equivalent
   exclusive nodes retain original JSON/YAML text despite high-level float64
   loss. Parse these nodes into the same owner, preserving 3.0 boolean-exclusive
   normalization. Render exact DSL strings where untyped Go constants would be
   rounded at the `any` call boundary. Import example admission must compare
   exact source numbers against this owner; do not retain its float64-only
   `normalizedExampleNumber` comparison for numeric-bound checks. General
   imported enum equality/default normalization remains #685's separate scope.
   Source numeric recognition is distinct from renderability in the retained DSL:
   `minimum: 1e309` is a valid numeric source token, but `Minimum("1e309")` is
   rejected. Import must report the schema path, keyword and exact source value
   with a diagnostic explaining the unsupported DSL range, rather than dropping
   the bound or emitting a known-invalid design. Apply this to all four endpoints
   and JSON/YAML. An alternative exact construction may be used only if it is
   already part of the accepted public contract; none is assumed here.

For finite float targets, an exact endpoint not representable by the target
needs **directed** projection: round to the nearest target float, compare that
float's exact value with the endpoint, and use `Nextafter`/`Nextafter32` when the
rounding direction and inclusive/exclusive endpoint require it. Then emit the
closed accepted-domain boundary. Overflow/underflow and empty domains must be
explicit outcomes. Generated typed validators use this generation-time lowering.
Runtime JSON schema evaluators use the shared exact arithmetic owner; transports
do not gain independent arbitrary-precision comparisons or tolerance policies.

## Source admission and finite-number decisions

The current builtin source matrix is unusual but explicit. Preserve it for this
repair: accept `int`, `int8`, `int16`, `int32`, `int64`, `uint8`, `uint16`,
`uint32`, `uint64`, `float32`, `float64`, and builtin `string`. Do not admit
`uint`, defined numeric/string types, pointers, or nil. The 30-case
`TestNumericBoundSourceAdmission` is a green parent control of that boundary;
accepting `uint` can be a separate documented policy correction if desired.

Candidate semantics:

| Input | Proposed exact meaning | Parent behavior / change |
| --- | --- | --- |
| `int64(9007199254740993)` | exact integer | Parent loses one before any consumer runs |
| `uint64(18446744073709551615)` | exact integer | Parent rounds to 2^64 |
| `float64(0.1)` | the supplied finite IEEE value, exact binary rational | Retains existing source-admission meaning; its exact schema token becomes longer |
| `float32(0.1)` | the supplied 32-bit value, without another normalization | Retains its distinct exact value; does not substitute short spelling `0.1` |
| string `"0.1"` | proposed exact decimal 1/10 | Parent parses it as Float64; this interpretation remains a policy proposal |
| string `"125e-2"` | proposed exact decimal 1.25 | Preserve exponent meaning, normalize semantic identity |
| string `"0x1.4p0"` | exact binary-radix 1.25 | Preserve existing DSL admission; emit language-neutral decimal externally |
| `-0`, string `"-0"` | numeric bound zero | Bound comparisons have no signed-zero distinction; source-value representation policy is unchanged |
| string `"1e-400"` | proposed nonzero positive rational | Parent underflows silently to zero; retaining it changes admission of zero |
| NaN or infinity, numeric or string | proposed design diagnostic | Parent accepts these even though JSON numeric output cannot express them |
| string `"1e309"` | keep current rejection | Parent reports ParseFloat range error; exact text is not permission to silently widen admission |

The existing parser also accepts underscores, leading plus, `.5`, and `1.`;
those green controls must remain unless a reviewed public syntax correction says
otherwise. It rejects ratios and surrounding whitespace. String grammar checking
must be separate from mathematical conversion: ParseFloat may be useful as an
admission oracle, but its returned float is never the retained value.

**Resource boundary and hexadecimal syntax:** ParseFloat accepts
`"1e-1000001"` as zero without an error, while `big.Rat.SetString` rejects that
decimal exponent. The compact decimal core supports it exactly and emits `1e-1000001`
without a large power allocation; no new public complexity limit is recommended.
Do not use the standard rational parser's limit as an accidental new contract.

The same parser accepts `"0x1p-1000001"`. Its exact decimal coefficient is
5^1000001, which needs about 699,000 digits. For an arbitrarily larger binary
exponent, any exact JSON decimal token has the corresponding unavoidable output
size. A compact factor tuple avoids this cost during storage, hashing and target
range checks. Exact decimal emission cannot avoid work proportional to its output.
To emit, let K=min(E2,E5); the decimal coefficient is
`C * 2^(E2-K) * 5^(E5-K)` and the exponent is K. One residual power is zero.
For ordinary decimal inputs and bounded IEEE floats, this is small; for exotic
hexadecimal underflow strings, it may be intrinsically enormous. There is no
claim that exact arbitrary-hex serialization has bounded constant resources.

Target range checks do not require that expansion. Compare against the finite
kind endpoints first using compact order. Integer whole-number admission rejects
a negative denominator exponent after coefficient normalization. Float projection
can classify values below the least subnormal, between finite endpoints, or beyond
the greatest finite value. For in-range candidates use a bounded-significant-digit
rounded approximation, then compare the proposed target float exactly and select
its neighbor if needed. The rejected interval-logarithm shortcut must remain
explicit. Decimal tiny inputs below the least subnormal need only zero and the
least subnormal as candidates, independent of exponent magnitude.

The user did not accept the decimal-only proposal. The current recommendation is
to preserve admitted finite hexadecimal DSL syntax, use the shared factorized
number, and acknowledge output-proportional cost. Do not broaden the DSL to YAML
integer prefixes just because the importer supports them, or narrow the importer
to Go's parser grammar. The prototype below demonstrates feasibility; it is not
production authorization. No arbitrary input/output limit has been introduced.

## Float policy comparison and real adoption evidence

The accepted value-contract design says:

> Numeric meaning and representation identity are separate.

It also says:

> encoding a Float32 as its short JSON decimal must not replace its exact numeric meaning.

(`roadmap/value-contract-design.md`, numeric rules around lines 418–434.) The
public DSL guide only states `Minimum(n)` is “Minimum value (inclusive)” and the
corresponding exclusive/maximum descriptions (`docs/dsl-reference.md:412`). The
DSL GoDoc supplies numeric examples but does not define string grammar or whether
string decimals mean exact mathematical decimals. Thus exact source-float meaning
is grounded in the accepted design; the string interpretation and finite-only
rejection need explicit readiness approval, not an inference from ParseFloat.

Candidate A (recommended): preserve exact source numerical meaning. For
`Maximum(float64(0.1))`, `float64(0.1)` passes and its next greater Float64 fails;
for proposed `Maximum("0.1")`, that same Float64 fails because it is slightly
above 1/10, and its previous Float64 passes. Both decisions derive from exact
comparison. `Minimum("0.1")` accepts Float64(0.1), while the previous float fails.

Candidate B: treat shortest decimal spelling as the meaning of every bound.
With exact source value admission, `Maximum(float64(0.1))` now rejects the very
same Float64 value. For the existing `ExclusiveMaximum(100.1)` fixture, the
Float64 value is slightly **below** decimal 100.1 and would start passing, where
current source meaning rejects equality. Making B preserve those results would
require a second context-dependent comparison rule. Reject B as the architecture
recommendation; it conflicts with the quoted accepted distinction.

Candidate C: retain rounded Float64 meaning for strings but exact builtin
integers. This is narrower compatibility, yet `Minimum("9007199254740993")`
still admits 9007199254740992. It fails the exact text/import contract and leaves
a source-dependent lossy authority. Reject C as the #681 recommendation.

Actual inspected consumers (read-only, no changes):

- `loom-mcp/registry/design/design.go` lease bounds are 1 through 24 hours in
  milliseconds (86,400,000); its other bound is zero. Exact representation
  preserves these literals and admissible values.
- `loom-mcp/integration_tests/fixtures/assistant/design/design.go` uses integer
  1/200 limits and a floating score strictly between 0 and 1. Those endpoints
  are exactly representable under every candidate; preserve behavior.
- The framework's existing `codegen/testdata/validation_types_dsl.go` has Float64
  `Maximum(100.1)` and `ExclusiveMaximum(100.1)`. It is a required changed-output
  control, not a hypothetical decimal case. The current binary value is
  100.099999999999994315658113919198513031005859375.

The repository/these available consumer designs had no authored fractional
string bounds. This is a bounded adoption inspection, not an ecosystem claim.

**Schema/runtime separation remains explicit.** A raw JSON token `0.1` is exact
1/10. Its Float64 decoder produces
0.1000000000000000055511151231257827021181583404541015625. An exact schema minimum
at that binary bound rejects the raw token, while a generated decoded-value
validator accepts it. Choosing shortest decimal bounds does not generally remove
rounding-cell differences. Do not claim exact JSON schema membership equals
postdecode acceptance for floats, and do not introduce a decoder-preimage schema
policy merely to force that equality. Existing `NumericCodec` and value-plan
schema/decoder separation already model those as different judgments.

## Reproducible readiness evidence

Parent: `651372ce6ff05ed1db495369fd51d48df0c0ec75`; Go 1.27.2;
Lean 4.34.1, no new dependencies or production edits.

- DSL extraction: 20 failing large-integer/text cases, 4 passing small controls.
- Import extraction: 12 failing cases (3 large signed values × 4 keywords),
  4 passing small controls. `TestAnalyzePreservesExactNumericBounds` inspects the
  imported owner before DSL rendering, so this is independent of the DSL bug.
- `TestNumericBoundRawNodesPreservePrecision` checks actual libopenapi low nodes
  for both JSON and YAML, establishing that precision remains recoverable.
- `TestNumericBoundSourceAdmission`: all 30 compatibility cases pass at parent.
- `ValueContract.NumericExtraction`: nine claims build and pass the allowed-axiom
  audit and fresh kernel replay. Eight controls prove the concrete loss, dual exclusive failure,
  nonzero-underflow value, binary/decimal and Float32/Float64 distinctions, and
  the existing 100.1 fixture’s exclusive-admission change, a 36-pair compact
  comparison corpus, and huge-exponent algorithm controls; the ninth reuses the
  existing conjunction theorem under an explicit extraction-equality premise.
  It does not prove IEEE conversion, any Go parser, exact serialization, target
  projection, or the planned representation implementation.
- `NegativeNumericExtraction.lean` fails at its false assertion, as required:
  the rounded minimum does admit the below-authored-bound integer.
- The durable `numeric_bound_probe.go.txt` independently prints exact binary
  rationals, adjacent Float64 values, actual JSON decoding, and parser resource
  edge controls. Copy it to a temporary `.go` file and run with root Go tooling.

Run the focused tests above plus:

```sh
cd expr/lean/value_projection
lake build ValueContract.NumericExtraction
lake env lean -DwarningAsError=true ValueContract/AxiomAudit.lean
lake env leanchecker --fresh ValueContract.NumericExtraction
# Expected nonzero, specifically the false rounded-minimum assertion:
lake env lean NegativeNumericExtraction.lean
```

Those nine extraction claims are part of the sixteen-theorem readiness manifest
described below. Before a production
commit, integrate accepted claims into the existing proof gate and update its
correspondence ledger; do not leave a second unregistered proof authority.

## Bounded implementation acceptance sequence

1. Confirm the retained source-admission matrix and language-neutral owner below
   during readiness review. Confirm #755's stable parent, then preserve and adapt
   its detached composition tests.
2. Add owner tests for builtin admission, immutability, semantic equality,
   serialization, approved decimal/hexadecimal admission, all four endpoints, finite
   floats, smallest/largest floats, negative zero, and selected rejected syntax.
   Pair exact extraction with independent rational expectations, not the new
   owner's own formatter/parser.
3. Extend existing `ValidationTypesDSL` with a grouped exact Int64/UInt64 boundary
   case and decimal-source controls. Keep its 100.1 fixture. Add alias and Extend
   cases using existing effective-constraints fixtures; test default/enum/example
   selection near a large bound without widening #685's source-type/range scope.
   Include authored float defaults/enums/examples exactly on each boundary:
   their shortest JSON encoding can fall outside an exact binary schema endpoint
   even when the decoded carrier passes. Record whether the existing projection
   owner rejects or suppresses that artifact; do not silently substitute a new
   example representation or broaden schema bounds to conceal the discrepancy.
4. Extend the existing HTTP, gRPC and JSON-RPC effective-constraints generated
   module tests rather than introducing another temp-module harness. Cover
   2^53 neighbors, signed and unsigned extrema, exclusive equality, float32 and
   float64 directed boundary cases, and ordinary small controls. Assert actual
   transport error/invocation outcomes, plus CLI compilation/literal typing.
5. Render an OpenAPI 3.1 exact-bound specimen through JSON and YAML; inspect raw
   tokens and compare semantic exact values, run the pinned libopenapi/Vacuum
   gates. Import JSON and YAML through all four keywords and 3.0 boolean
   exclusives, render/evaluate DSL, then export. Large exact bound text must not
   become an untyped Go float argument during that round trip.
6. Parent/candidate byte comparisons: the existing numeric validation fixture,
   one composed/aliased numeric design, the three transport fixtures above, and
   one schema import/export specimen. Parent failures are retained as evidence;
   candidate generations in separate processes must match exactly, build and
   vet. List every literal, threshold, schema, example and fingerprint change.
   No exported-design corpus sweep is needed for this ticket acceptance.
7. Independent exact-diff review, required re-review, atomic #681 commit, push
   and ticket reconciliation are root-owned. #685/#692 remain open unless their
   distinct later acceptance rules are independently completed.

## Readiness review repair: runtime numeric ownership

The first independent review found a medium blocker: the candidate assumed exact
descriptor text alone repaired runtime matching. At this parent,
`JSONShapeRules{Minimum: "1e-1000001"}` fails descriptor validation even for wire
`1`. `pkg/json_shape.go` also uses `big.Rat.SetString` for wire integrality and
bound comparisons. `expr/value_projection_schema.go` turns failed rational
extraction into `false`, and `ValuePlan.MatchJSONSchema` returns that as a normal
nonmatch. This is a second manifestation of the same incomplete numeric owner.

Readiness repair sequence:

1. Record the common owner and error semantics here before adding evidence.
2. Add direct failing runtime and expression tests, plus a two-order runtime
   union control; keep production and the existing capacity regression unchanged.
3. Run only these new seams, preserve their exact outcomes, and freeze a new
   manifest for independent re-review. Production migration remains gated.

The internal owner must expose exact source-independent value, equality/order,
integrality and canonical numeric emission. Admission adapters are explicit:
JSON accepts only JSON numeric grammar (no leading plus, hexadecimal spelling,
underscores or ratios), while DSL strings and YAML numeric fields retain their
own previously documented source matrices. Runtime descriptors emitted by Loom
contain canonical JSON numeric tokens and use that same JSON adapter. Valid
JSON numbers such as `1e1000001` are not rejected merely because they exceed the
DSL's source-admission range or a runtime Float64 carrier. In a canonical factor
tuple, a nonzero value is integral exactly when both E2 and E5 are nonnegative;
zero is integral. This query requires no large power allocation.

Replace the four rational-parser seams in `pkg/json_shape.go`: rules validation,
wire numeric-kind/integrality recognition, wire extraction for bound comparison,
and descriptor endpoint extraction. Replace the corresponding numeric seams in
`expr/value_projection_schema.go`. Do not add another float or rational fallback.
The generated descriptor lowering in `expr/value_plan_json_shape.go` emits the
common canonical number, so generated codecs inherit the runtime correction.
No general enum-equality rewrite follows from these observations; that path
remains outside this repair unless independent evidence establishes a related
failure that requires expanding the scope.

Schema membership becomes `(matched, error)` throughout the expression recursion
(aliases, members, elements and union branches), using the existing public
`ValuePlan.MatchJSONSchema` error result. A valid incompatible kind or a violated
constraint is `(false, nil)`; a broken descriptor, unsupported evaluator state or
other fatal evaluation failure is an error. An error cannot become a branch
nonmatch or be hidden by a later matching branch. Untagged union evaluation must
retain the existing whole-candidate check: even a previously established match
or ambiguity does not justify skipping a candidate that could fail fatally.
`ProjectJSON` and `projectCustomJSON` must retain the diagnostic in their existing
projection result error channel and classify it as evaluator/plan failure, not
as an ordinary value outside the schema. This is an error-routing correction;
it does not change schema-versus-decoder uniqueness or target float domains.

The existing `TestJSONShapeNumericCapacityFailureIsFatal` currently fixes the
old `big.Rat` ceiling into a test. At production migration, replace its numeric
capacity expectation with successful huge-exponent number/integer membership
and fractional nonmembership. Preserve the intended fatal-error invariant with
malformed numeric descriptors and unsupported/cyclic shape controls. Place a
broken candidate before and after a valid matching candidate, and after an
already ambiguous pair; require an error and no selected branch in every order.
For expression evaluation, cover an invalid evaluator/plan descendant before and
after a valid union branch; require the error from `MatchJSONSchema` and the
projection result. Do not fabricate a new numeric capacity error to keep the old
test passing.

Bounded production acceptance extends the existing sequence with:

- Direct common-owner cases for JSON grammar, tiny/huge exponent integrality and
  all four endpoints. Runtime descriptor and prepared-schema membership must
  agree for `1`, `0`, the exact tiny boundary and a smaller nonzero decimal.
- Extend existing `TestJSONShapeLoweringMatchesPreparedSchema` with a constrained
  number, alias and one nested object member using the tiny bound; compare both
  paths and preserve fatal errors separately from expected mismatches.
- Reuse one existing generated untagged union fixture with a number branch
  constrained by `Minimum("1e-1000001")` and a string branch. Decode `1` and a
  string, reject `0`, and prove receiver state is unchanged on failure. Include
  the tiny wire token as a schema-match/float-underflow disagreement control:
  schema membership is true, while its decoded zero violates the minimum, so
  the existing schema/decoder identity rule rejects it. Regenerate in both branch
  orders, compile, compare exact output in independent processes, and verify no
  fallback branch is selected. No additional full-corpus sweep is required.
- Import JSON/YAML `1e309` at each endpoint and require a precise unsupported-DSL
  diagnostic before rendering; ordinary finite `1e308` and supported negative
  exponent controls remain importable. Keep quoted `"1e309"` a source-kind error,
  distinct from a numeric token that cannot be represented by the retained DSL.

Readiness evidence at the unchanged parent: the new runtime/prepared-schema
tests have fourteen failing cases and three ordinary/nonintegral controls pass.
The new JSON/YAML positive-overflow importer test fails all eight cases because
analysis returns no diagnostic. Diagnostic classification must remain fatal even
with allow-lossy enabled. The source-specific admission controls already confirm
that the corresponding DSL string is rejected. Run:

```sh
go test ./pkg ./expr -run 'TestJSONShapeCompactExactNumericBounds|TestJSONShapeNumericDescriptorUsesJSONGrammar|TestUntaggedJSONCompactNumericBound|TestPreparedSchemaCompactNumericMembership' -count=1
go test ./internal/openapiimport -run '^TestAnalyzeDiagnosesNumericBoundOutsideDSLAdmission$' -count=1
```

The scoped raw results are `/tmp/loom-numeric-runtime-red.json` and
`/tmp/loom-numeric-import-overflow-red.json`. These tests prove the newly observed
losses, not the future evaluator error-routing implementation. No production
code, existing capacity expectation, Lean claim or arithmetic prototype changed
during this repair; previous proof/prototype evidence remains applicable.


## Language-neutral update: concrete algorithm and evidence

The user explicitly emphasized interoperability with other languages. Therefore
a Go parser's convenience cannot define the shared numeric contract. The
recommended owner retains mathematical value independently from authored syntax,
source language and a target carrier's precision. A C/Python/Java-derived binary
hexadecimal spelling and an equivalent JSON/YAML decimal normalize to one number;
Go target comparison is a later projection, not the number's definition.

Relevant protocol contracts support this separation:

- JSON numbers have decimal wire syntax; parsers may impose range/precision
  limits, and binary64 interoperability does not preserve every larger integer.
  [RFC 8259 section 6](https://www.rfc-editor.org/rfc/rfc8259#section-6).
- JSON Schema does not add a finite numeric precision/range bound to its semantic
  model. [JSON Schema 2020-12 section 4.2](https://json-schema.org/draft/2020-12/json-schema-validation).
- Python's JSON decoder accepts an explicit `parse_float` adapter such as Decimal;
  preserving exact numbers is therefore a real consumer path, not a Go-specific
  assumption. [Python JSON decoding](https://docs.python.org/3/library/json.html#json.load).

### Prototype algorithm

`numeric_factor_probe.go.txt` is a standalone readiness prototype, not framework
code. It has no production callers or new dependencies. Its exact key is
`(sign, C, E2, E5)` for `sign * C * 2^E2 * 5^E5`, with C stripped of factors 2 and
5 and zero canonicalized separately. Arbitrary integer exponents are compact;
normalization, equality and hashing do not allocate powers determined by exponent
magnitude. Decimal scientific input contributes E to both exponents; binary/hex
input contributes only to E2. For example `0x1.4p0` and `1.25` have identical keys.

For ordering, subtract exponent pairs and move negative factors to the opposite
side. The remaining positive integer products are evaluated with outward-rounded
binary mantissa intervals and arbitrary-size scale exponents. Repeated squaring
uses O(bit length of exponent) multiplications per trial precision. Each
normalization rounds the lower endpoint down and the upper endpoint up. Compare
scales plus mantissa bit lengths first; equal-magnitude alignment shifts only by
the bounded mantissa lengths. A decision is made only when the two intervals are
strictly separated. On overlap, double the working precision. Canonical equal
keys bypass this loop. No tolerance or approximate-logarithm decision is used.

The implementation prototype passed 10,000 deterministic comparisons against an
independently materialized `big.Rat` oracle (signed coefficients and both exponents
in [-400,400]). Selected huge exponent cases, ordinary aliases and zero pass.
The 1001-digit pair `10^1000+1` versus `10^1000` requires seven refinement rounds
and 4736-bit mantissas, proving the overlap/refinement path is actually exercised.
A mixed decimal/binary pair near exponent -1,000,001 also matches a separately
materialized rational oracle. For exponents of magnitude 10^30, the selected
cross-radix comparisons finish in one round at 164-bit precision, around 75 μs
in the observed run. These are bounded feasibility measurements, not worst-case
performance promises.

A separate compiled-binary run of the complete factor probe, including its 10,000
oracle checks and all six selected emissions, took 0.54 seconds wall time with
17,252,352 bytes maximum resident memory. This is one local feasibility observation,
not a benchmark gate or a bound on arbitrary admitted inputs.

The prototype parser is deliberately *not* a replacement source-admission parser.
Its pure-arithmetic controls include `1e1000`, which current DSL admission rejects;
that control does not authorize widening source admission. Production must retain
its separately tested concrete source matrix and finite overflow decisions.
Likewise its repeated division for removing factors of 5 is a simple prototype;
a production implementation should remove coefficient trailing zeros first and
use bounded-by-input-size factor extraction rather than repeating expensive work
for very long coefficients. This is an implementation choice, not a numeric cap.

### Model: safety and termination, with limits

`ValueContract.NumericIntervals` adds seven audited claims to the existing nine
extraction claims. It models finite multiplication expressions, outward enclosures,
and exact arithmetic below a trial capacity. It proves:

- every enclosure contains the true result;
- an order inferred from disjoint enclosures is sound;
- a sufficiently large finite capacity makes every intermediate exact;
- the doubling-precision schedule eventually separates every unequal finite pair;
- overlapping rounded lower endpoints do not establish equality.

The last two facts address the earlier speculative algorithm's missing termination
argument. The existence bound can be as large as the expanded products, so it is
**not** a polynomial-time or practical universal resource bound. No arbitrary
precision ceiling is needed for mathematical termination, and none is proposed.
The model's normalization is an abstract floor/ceiling operator; Go `big.Int`,
BitLen/scale bookkeeping, normalized factor-key uniqueness, exponentiation-by-
squaring extraction and actual serialization are still correspondence obligations.
The 10,000 oracle comparisons exercise the actual scaled prototype but do not
turn those obligations into universal implementation proofs. Both negative Lean
controls must fail. Sixteen claims pass the scoped axiom audit and fresh kernel
replay; the candidate is not yet registered in the production proof gate.

### Exact output and actual language consumers

Emission takes K=min(E2,E5), coefficient `C*2^(E2-K)*5^(E5-K)`, and decimal
exponent K. It emits an unquoted exact numeric token. Selected observed outputs:

| Authored source | Exact emitted token size | Observation |
| --- | --- | --- |
| `9007199254740993` | 16 bytes | Python integer/Decimal preserves it; JavaScript Number rounds it |
| `0x1p-2` | 5 bytes (`25e-2`) | Go, Python and JavaScript preserve 0.25 |
| `1e-1000001` | 10 bytes | Exact compact decimal; native Float64/Number decoders round to zero |
| `0x1p-1000001` | 698,980 bytes | Exact output inherently needs a large coefficient; generation around 0.25 s in the observed run |
| actual Float64 0.1 | 59 bytes | Exact binary value, distinct from decimal 0.1 |
| actual Float32 0.1 | 31 bytes | Exact binary value, distinct from Float64 0.1 |

The large hexadecimal token passes Go JSON lexical validity. Python 3.14.8
`json.loads(..., parse_float=Decimal, parse_int=Decimal)` preserves every exact
coefficient/exponent in the six selected outputs. A separate Python 3.9.6
Fraction oracle confirmed exact mathematical values for all six, including
2^-1000001, but that optional large-rational conversion took about 109 seconds;
it is not part of the fast reproducible acceptance recipe. The retained Python
probe checks exact lexical coefficient/exponent preservation in under a second
for that specimen. These are observed interpreter versions, not a new supported-
runtime matrix or framework dependency.

Node 26.11.1's ordinary JSON.parse rounds the large integer and tiny numbers, while
its source-aware reviver preserves every numeric token exactly for later BigInt
or decimal processing. Go's JSON-v2 Float64 decoder accepts the compact decimal
underflow as zero but reports a range error for the large-coefficient hexadecimal
expansion. The latter is a concrete decoder limitation, not invalid JSON or an
excuse to lose the number in Loom's shared contract. No claim of identical
native-number behavior across languages is made. The runtime Float64 target
continues to have its own representable domain and directed bound projection.

### Source-specific OpenAPI normalization

`numeric_schema_probe.go.txt` checks the pinned libopenapi adapter and original
source token kinds. Its high model cannot be the exact source owner:

- YAML `0x20000000000001`, `0o377`, and `0b101` retain raw !!int nodes but become
  high-model Minimum=0 without an error. Twelve additional importer regressions
  now fail across these three values and all four bound keywords. YAML `077`
  currently means decimal 77 in this boundary; `1_000` means 1000. Eight controls
  preserve those meanings. These syntax forms do not broaden the Go DSL grammar.
- The 698,980-byte exact token is a JSON number in an independent JSON-v2 raw-token
  parse. libopenapi/go-yaml retags it as !!str, leaves the low Minimum reference
  absent and silently omits the high bound. The original schema-proxy source node
  still contains its full text. A low-field-reference-only fix would lose it.
- Independent raw JSON parsing preserves a quoted `"42"` as a string, so numeric
  recovery must reject it rather than feeding its contents to a number parser.
- For YAML, retain quote/explicit-tag style and protocol lexical classification.
  Plain decimal numeric syntax can remain recoverable when a finite parser
  retags it; quoted `"42"`, quoted `"0x10"`, and explicit `!!str 42` stay strings.
  Explicit `!!float "42"` remains numeric as it does at the parent. Plain YAML
  `0x1.4p0` remains a string, distinct from the admitted DSL hexadecimal-float
  string syntax. The probe asserts these distinctions, including huge emitted
  decimal text in both JSON and YAML.

The implementation proposal is a source-boundary adapter that retains raw numeric
meaning and source kind/style *before* libopenapi's high floating conversion.
JSON extraction uses the original JSON token kind; YAML uses its source schema
and retained explicit tags/quoting plus exact lexical normalization. The adapter
returns the common exact number or an existing-style precise design diagnostic.
It must not infer numeric admission solely from parseable node text, silently
accept quoted numeric fields, or trust a downstream retagged node as the original
kind. Full YAML grammar/alias/tag correspondence requires tests beyond this
bounded probe before production acceptance; no general parser has been written.

### Reproduce the update

From the repository root, copy the `.go.txt` specimens to task-owned temporary
`.go` files and run them there. The factor probe takes an output directory; the
schema, Python and JavaScript probes take that same directory. Preserve a bounded
specimen selection; do not request expansion of the 10^30-exponent controls.

```sh
go test ./internal/openapiimport -run '^TestAnalyzePreservesYAMLIntegerNumericBounds$' -count=1
# Expected twelve failing preservation cases and eight passing controls.
go run /tmp/loom-numeric-factor-probe.go /tmp/loom-numeric-factor-output
go run /tmp/loom-numeric-schema-probe.go /tmp/loom-numeric-factor-output
python3.14 expr/lean/value_projection/numeric_cross_language_probe.py /tmp/loom-numeric-factor-output
node expr/lean/value_projection/numeric_cross_language_probe.js /tmp/loom-numeric-factor-output
cd expr/lean/value_projection
lake build ValueContract.NumericIntervals
lake env lean -DwarningAsError=true ValueContract/AxiomAudit.lean
lake env leanchecker --fresh ValueContract.NumericIntervals
# Each command must fail at its documented false assertion:
lake env lean NegativeNumericExtraction.lean
lake env lean NegativeNumericIntervalGuess.lean
```

Current total direct reds: 20 DSL + 12 decimal-import + 12 YAML-radix-import +
14 runtime/prepared-schema + 8 import-range diagnostic cases = 66.
The earlier source-admission and raw-node controls remain valid, as production has
not changed. The bounded generator/transport/OpenAPI acceptance plan above still
applies, extended with original-source grammar and cross-language exact-token
controls. Main/shared skills, commits, public policy and production implementation
remain root-owned; independent readiness review is the next gate.
