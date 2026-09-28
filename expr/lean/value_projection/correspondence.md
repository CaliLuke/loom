# Proof and production correspondence ledger

Status: **M1 foundation only, #567.** The reviewed contract is
[value-contract-design.md](../../../roadmap/value-contract-design.md).
The source-history reference is `f5b786b39675e2b5c1466f04f3301b7b337779b6`.
The word “proved” below applies to the Lean statement, not to handwritten Go,
generated programs, all JSON strings, or any codec implementation.

## Established statements

All names below are in `ValueContract.Legacy`. `required-theorems.txt` enumerates
the individual obligations, and the gate prints their transitive axiom sets.

| Theorem names | Status and domain | Production seam / additional evidence |
| --- | --- | --- |
| `erasedBranchesIndistinguishable`, `legacyBranchLoss` | **Proved**: two concrete selected branches erase to one payload; first-branch reconstruction changes the second identity | `expr/types_union.go` source selection and `expr/example_canonicalization.go` reconstruction; #566 nested-branch probe supplies independent generated evidence |
| `legacyByteReinterpretation`, `byteStringDecodes`, `byteEncodingWitness` | **Proved** for the concrete `hi`/`aGk=` specimen codec; literal `hi` does not decode as the original bytes | Historical OpenAPI byte conversion; current authored-byte CLI bug in `codegen/cli/conversion.go`; #566 actual generated builder/decoder probe |
| `byteTextWireCollision` | **Proved**: the same `Wire.text "aGk="` decodes as Bytes and String | Untagged OpenAPI matching in `http/codegen/openapi/internal/ir/document_examples.go`; full unique-match policy and actual-decoder correspondence pending #569/#570 |
| `legacyLengthRejectsValidBytes`, `legacyLengthAcceptsShortBytes` | **Proved** concrete mismatches between decoded byte count and ASCII wire-string length under `boundaryCodec`; neither direction can use copied length bounds | `expr/json_schema_inline.go`, OpenAPI IR `analyzer.go`, and generated decoded-length validation. #574 has independent generated-server/schema reproductions; repaired arithmetic and complete lexical equivalence remain pending |
| `canonicalByteEnumAgreement`, `aliasSchemaDecoderMismatch` | **Proved**: canonical `aGk=` meets both enum predicates, while String `aGl=` fails the canonical byte-schema enum but decodes to the allowed byte value `hi` | #569 must separately require emitted-schema validity and actual decoder branch/observation preservation. #570 tests real codecs and #574 corrects the length/grammar schema; no enum broadening or String normalization follows from this witness |
| `legacyAuthoredReplacement`, `legacySourceReplacement` | **Proved**: concrete unconditional synthesis replaces value and source | `grpc/codegen/client_cli_example.go`, `service_data_analysis.go` and transport-copy source selection; #566 probes with and without unions |
| `selectedBodyObservation`, `selectedBodyRepresentable`, `observationIsNotWholeService` | **Proved**: a body selection drops its service header, retains bytes, and remains representable under the specimen codec | `expr/http_body_types.go` and HTTP target plans; actual plan extraction remains a tested implementation boundary |
| `visibleBranchRetained` | **Proved**: a visible second branch is retained by the independent observation judgment | `expr` semantic identity and transport plans; full nested preservation pending #569 |
| `nullDistinctFromAbsent`, `nullObservation`, `absentObservation`, `omissionIsFieldLocal` | **Proved**: null/absence are distinct, and only the selected field omission rule collapses empty collections | `expr` and generated Optional/Nullable/omitempty contracts; complete JSON/protobuf presence matrix pending #569/#570 |
| `duplicateEntriesRetained` | **Proved**: the example raw entry list retains both integer `1` and string `1` keys | `internal/jsonkey`, enum/default normalization and #456; spelling/collision rejection is not proved here |
| `finiteRecursiveValue` | **Proved**: a recursive declaration has a finite null-terminated structural inhabitant | `expr` recursive type/example handling; resolver termination and incomplete-cycle outcomes pending #569/#570 |

## Vocabulary and current gaps

| Model owner | Production owner | Current boundary |
| --- | --- | --- |
| `RawValue`, `Scalar`, `Role`, `Source`, `Supplied` | `expr` authored inputs, example generator, enum/default values and future semantic API | Finite vocabulary; no Go adapter yet. Ordered raw field/map entries preserve collisions; typed bytes differ from authored strings |
| `Value`, `Identity` | Future `expr` resolved values and generation-local carrier identity | No public Go API or storage implementation is certified |
| `Outcome` | Future resolver/projector outcomes and diagnostics | Constructors only; no candidate failure classification theorem yet |
| `Shape`, `Declarations`, `HasType` | Finalized effective `expr` occurrence | Structural completeness only; optional-absent fields, constraints, source matching and complete recursive resolution are pending |
| `Plan`, `Field`, `Branch` | HTTP/OpenAPI visibility, body/view selection and allocated protobuf mapping | Explicit inputs. Producing valid plans from DSL identities is not proved. `required` and protobuf styles are vocabulary awaiting full target rules |
| `Observe` | Target-observable service meaning | Independent of a projector. Expresses field/visibility loss and retained branch identities. Full numeric, object openness, nil-container and protobuf presence rules are pending |
| `Wire`, `DecodeScalar`, `WireTyped`, `Decode`, `Representable` | Structured JSON/protobuf projection and existing runtime codecs | Current wire judgments cover scalars, nullable, arrays and selected bodies only. Objects, maps, unions and protobuf constraints must be added before universal candidate claims |

Depth-indexed judgments use an unbounded existential depth, not a finite TLC-like
search bound. Successful derivations are finite. This does not yet prove that
every complete finite value resolves, projects or decodes.

## Weaknesses discovered after the initial foundation

| Weakness | Model/evidence now | Remaining owner and acceptance boundary |
| --- | --- | --- |
| Copying byte-count bounds onto base64 string length rejects valid values and accepts short decoded values | Two registered concrete Lean counterexamples above; actual generated HTTP server and inline/OpenAPI discrepancy reproduced for #574 | #575 specifies the policy; #569 may prove a candidate model; #574 implements shared schema arithmetic/grammar. Production correspondence remains pending and cannot assume the correction landed |
| Schema enum membership can differ from actual decoder acceptance of a noncanonical pad-bit alias | Registered alias counterexample plus canonical control use a finite, explicit specimen codec. This is not a universal codec axiom or a proof about generated union routing | #569 must define both predicates independently, and representability/preservation must require both. #570 and consumer gates test canonical/alias/malformed values with actual schema validators and decoders; never infer runtime uniqueness from schema-only uniqueness |
| Missing required fields and ambiguous nested unions affect complete-first branch ranking differently | M1 has no resolver or optional-field typing theorem. Existing Go complete matching excludes nested ambiguity; no M1 theorem establishes a broader candidate algorithm | #575 settles complete-first ranking, zero-complete fallback and ambiguity obstruction; #569 must encode that reviewed rule and competing complete/partial cases before proving progress; #570 preserves the public compatibility adapter |
| One named schema/cache entry can be shared by incompatible JSON, multipart or location representations | M1 `Plan` is explicit vocabulary, not a derived representation identity or a schema-cache model. The shared `Blob` reference probe reproduces this integration weakness; current Lean witnesses make no naming/cache claim | #575 must approve public-name/reference compatibility; #570 supplies durable representation/occurrence plans; #574 schema analysis consumes them before component registration. Reversed traversal, recursive references and excluded-context equivalence require independent generated tests |
| Named-Bytes pointer conversion generates uncompilable Go (#576) | Concrete generated compilation failure outside the Lean semantic domain; not a defect in the statements proved here | #576 must repair and verify generated compilation separately. While open, generated-code correctness for this case is explicitly **not established**, regardless of Lean or arithmetic-gate success |

These rows invalidate any broader inference from the old foundation approval;
the original concrete lemmas remain unchanged. Counterexample/model and ledger
updates, audited gates and independent re-review precede reliance on a repaired
claim. A documented pending row is not a substitute for the later required proof
or production test.

## Trust and evidence boundaries

- **Proved:** the registered M1 witnesses, checked by Lean 4.34.1 and fresh
  kernel replay. Allowed logical axioms are exactly `propext`,
  `Classical.choice`, `Quot.sound`; the per-theorem report is authoritative.
  `sorryAx`, custom correctness axioms and native-evaluation trust are rejected.
- **Tested:** gate orchestration and actual transitive rejection of an admitted
  dependency/custom axiom by `scripts/value_contract_proof_test.go`. Enabled
  external tests fail on missing tools. #566 owns actual generator probes;
  their results must be linked to the baseline record, not inferred from Lean.
- **Bounded-checked:** the separate M1 TLA model, owned by #568, addresses
  phase/cache/source ordering only. No TLA result is asserted by this gate.
- **Assumed or external:** the Lean kernel/toolchain and operating environment;
  correct DSL-to-plan mapping; scalar runtime precision/ranges; lexical codec
  behavior; JSON/protojson libraries; Go compiler; custom/opaque Any inputs.
  `ByteCodec` has explicit functions, not an axiom claiming arbitrary codecs
  correct. A future theorem requiring `ByteCodec.RoundTrip` must list that
  hypothesis. Its actual base64 implementation needs separate tests or proof.
  Both Bytes and String use one `Wire.text` domain before matching; byte/string
  collisions remain visible.
  `boundaryCodec` intentionally recognizes only the concrete empty, one-byte,
  two-byte and alias specimens; no universal round-trip or decoder-language law
  is claimed for it. Its string lengths in the witnesses are ASCII lengths.

## Pending acceptance obligations

**#569:** candidate resolver/projection functions; independent complete target
validity and decoding; universal soundness, progress and preservation against
`Observe`; authored-source preservation; complete failure outcomes; finite
recursive values; range/name/enum constraints; non-vacuous representability
including legitimate field loss and untagged collisions. Extend the manifest,
audit all new theorem dependencies, and fresh-check the candidate proof module.

**#570 and consumer migrations:** executable Lean reference calling the proved
functions; independent Go/reference adapters retaining map duplicates, source
and presence; differential cases and negative controls; real schema/decoder
validation; generated build/vet and process-isolated exact-byte comparisons.
This will be **tested correspondence**, not a proved Go refinement. Candidate
proofs and production correspondence must never be reported as completed from
the M1 witness gate alone.
