# JSON field presence and omission

Status: #848 implemented; bounded acceptance passes; awaiting independent final review.
Issue: https://github.com/CaliLuke/loom/issues/848.
Parent: `0f1fc3dc395f408e4c02548dbe259505eb58c1ce`.
Owner: `/root/omission_readiness_finish`.
Worktree: `/Users/luca/.codex/worktrees/json-presence-carriers/loom`.

## Accepted contract

Explicitly supplied empty strings, bytes, arrays, maps and objects remain distinct
from absence under framework-owned JSON omission. Native zero values are supplied;
defaults apply only to absence. Required admission and nullable null remain separate
contracts. Authored full tags and external custom codecs retain their explicit
contracts. This does not add application PATCH semantics or change protobuf
collection presence, repeated text formats, selected views or nil-object conversion.

#835 delivered absence-only conversion defaults. #838 delivered independent CLI
JSON decode carriers. #834 delivered selected-view presence. #844 delivered shared
authored-tag normalization at this parent; see [its delivery plan](json-tag-normalization.md).
There is no longer a competing HTTP omission-option merger to preserve.

## Sequence

1. [x] Preserve prior witnesses and completed prerequisites on the pushed parent.
2. [x] Reproduce automatic omission through actual JSON-v2, body declarations,
   retained value projection and generated HTTP/JSON-RPC clients/responses.
3. [x] Inspect actual service/HTTP field layouts, custom codecs, defaults and
   inherited tag comparison; reject global omitzero and tags-only alternatives.
4. [x] Revise the model to keep arbitrary custom codecs opaque; reproduce current
   and insufficient candidate counterexamples and check the proposed contract.
5. [x] Independently review the concrete owner proposal below and bounded acceptance.
6. [x] Implement after independent readiness approval; compare parent/candidate/repeat
   output, compile selected modules and validate runtime/spec boundaries.
7. [ ] Freeze exact atomic diff for fresh final review; integrate/push/close via root.

## Pre-implementation evidence at the parent

Go 1.27.2, GOEXPERIMENT=jsonv2. Generated helper pin is pushed `063198a2f427`;
LOOM_DIR explicitly selected this worktree. These witnesses preceded production edits.

- Existing automatic policy still loses five present-empty carriers: pointer to
  string, bytes, slice, map and pointer to empty object. Both generated clients
  lose them before service invocation; independently exercised HTTP/JSON-RPC
  responses emit `{}`. Absence/populated/custom time.Time controls pass.
  `/tmp/loom-m7-0f1fc3-parent-red.log` (generated module 2.479s).
- Sixteen physical-layout cases pass in `TestJSONPhysicalLayoutReadiness`
  (`/tmp/loom-m7-layout-controls.log`, 0.630s). A defaulted string is native with
  materialization, a pointer without it, and Optional in a JSON decoder. Required
  native values differ from forced decoder pointers. Bytes, arrays, maps,
  Nullable, raw JSON and custom time.Time retain their existing layouts.
- Three prospective inherited-tag equivalence cases fail under the old omission
  rule; three rejection/requiredness controls already pass. They specify canonical
  service semantics, not a transport decoder layout.
  `/tmp/loom-m7-composition-red.log` (0.450s).
- Full authored omitzero on a native defaulted string/bool/int already projects a
  supplied zero into the example, although Go encoding omits it. This confirms
  that `ValueFieldOmitAbsent` cannot represent every `omitzero` carrier.
  `/tmp/loom-m7-native-zero-projection-red.log`.
- The revised model exhausts 58 states. Current omission fails PreserveSupplied
  after 32; global zero omission fails after 30; changing only emitted tags fails
  ProjectionAgreement after 32. Its custom boundary preserves existing codec policy
  and marks builtin projection unsupported. See `codegen/tla/json_omission/README.md`.

Native `false`/numeric zero already survive JSON-v2 omitempty; an empty native
string does not. Replacing every option with omitzero would newly discard those
working native values. A nonnil custom value can have IsZero return true; the
working time.Time zero value must keep its current wire output. Existing test-only
candidate controls demonstrate both counterexamples. No custom wrapper is proposed.

## Implemented owner and information flow

The immutable resolved JSON field plan lives under `internal/encodingmeta`, adjacent
to the delivered JSONTag owner. Separate authored intent, actual storage, and
resolved emission/projection action. The plan owns normalized tag options and
an action: Retain, OmitAbsent, OmitEncodedEmpty, OmitNativeZero, or ExternalCodec.
An authored combination of both omission options combines encoded-empty and
native-zero predicates; it does not discard either authored option.
Do not map every `omitzero` tag back to OmitAbsent by reading option text.

The field declaration owner supplies a storage record: ordinary native value,
ordinary nil-capable carrier, explicit framework presence carrier, or opaque
external codec. Required/optional emission role is separate from that storage.
The record comes from the exact branch that chooses the field's type, including
wrapper and pointer decisions; it is not parsed from rendered Go source and is
not inferred solely from semantic kind or default existence.

- In service codegen, retain the storage decision in
  `NameScope.objectFieldTypeDef` and hand it to tag lowering. Native materialized
  defaults retain zeros; forced pointer layouts retain their actual absence.
  Direct public tag helpers need an explicit/canonical context rather than
  guessing the caller's layout from the attribute alone.
- In HTTP codegen, prepare the field layout once at the representation boundary.
  `goObjectFieldDef` uses it for wrapper/pointer declaration and tags;
  `emittedHTTPFieldPolicy` consumes the same plan for projection. Use the registered
  named layout from `representation.ApplyLayout`, not the current occurrence's
  incidental context. JSON-RPC continues to consume this owner.
- Do not replace complete type reference/name allocation: NameScope still renders
  every type. The new record captures presence properties, not Go source text.
- Existing external replacement selection (`encodingmeta.GoType` and
  `representation.CustomCodec`) retains precedence. Opaque custom types keep their
  current tag/codec policy; value projection remains explicitly unsupported where
  it already lacks a builtin representation. Known framework Optional/Nullable/
  JSONValue carriers keep their own absence contract, not an application's IsZero.

### Automatic lowering

| Physical field | Framework-owned automatic action |
| --- | --- |
| Optional ordinary pointer/slice/map/raw JSON | OmitAbsent, rendered as omitzero |
| Optional framework Optional/Nullable | OmitAbsent, rendered as omitzero |
| Native value, including materialized scalar defaults | Retain; no omission option |
| Required encoder member | Retain; existing validation owns required admission |
| External custom codec | Preserve its established codec/tag contract |

Forced decoder storage and explicit-presence normalization remain role-owned;
required semantic membership does not justify pretending a physical pointer is a
native value. Full authored tags retain #844 behavior. When an authored omitzero
actually targets a native scalar, its projection must use native zero semantics;
when it targets a known nil-capable field, it means absence. Authored omitempty
uses encoded emptiness. These are distinct decisions of the same resolved owner.

## Inherited metadata identity

Extend compares authored metadata under canonical service absence semantics,
including the finalized inherited default and requiredness. It must not use a
request decoder's pointer/Optional carrier, nor preserve the old automatic
omitempty comparison after automatic omission changes. Canonical service storage
facts and declaration lowering must agree through the same representation record;
avoid a second transport-specific tag-equivalence algorithm.

Concrete acceptance cases in `TestJSONOmissionCompositionReadiness`:

| Canonical optional string | Full tag vs name-only tag |
| --- | --- |
| No default, pointer storage | full `value,omitzero` and name-only `value` agree |
| No default, pointer storage | full `value,omitempty` and name-only `value` conflict |
| No default, pointer storage | full `value` and name-only `value` conflict |
| Inherited materialized default | full `value` and name-only `value` agree |
| Inherited materialized default | full `value,omitzero` and name-only `value` conflict |
| Required native value | full `value` and name-only `value` agree |

This follows accepted absence-only automatic intent and explicit full-tag intent.
Update the existing documented Extend examples to those semantics; preserve
opaque codec metadata identity and full-tag precedence. This is implemented through the canonical service layout owner. The composition
pass retains original authored metadata and checks identity after effective
constraints/defaults are finalized.

## Correspondence and bounded acceptance

| Obligation | Owner / selected evidence |
| --- | --- |
| Actual storage | NameScope object fields and HTTP field layout record; direct 16-case ledger plus named/recursive/custom controls |
| Automatic omission | Shared resolved JSON field plan; nil/empty/populated serialization and native zero tests |
| Explicit tags | Delivered JSONTag normalization; full/no-option/empty/zero and exact skip controls |
| Projection | HTTP value plan consumes resolved action; direct empty and native-zero witnesses, rendered examples |
| Composition | Canonical finalized service facts; both traversal orders and inherited defaults/requiredness |
| Runtime preservation | Generated HTTP/RPC clients and independent response encoders; exact key presence, service calls and payload |
| Decoder admission | Existing missing required/null/malformed/truncated/whitespace matrix; no weakening or new parser |
| Custom boundary | Native and generated time.Time zero plus custom IsZero control; preserve output and opaque projection |

Implementation acceptance uses the existing ordinary/full-tag/custom
specimen from #844, extended with native defaults and named/recursive containers,
and one existing selected-view fixture. Compare every emitted file at the parent,
candidate and independent repeat; build/vet every distinct generated module and
list each intended difference. Add only cases that cover a distinct changed
representation. Reuse the nontrivial scalar-map OpenAPI specimen for rendered
empty/default examples, libopenapi parsing and pinned Vacuum JSON/YAML linting.
Regenerate affected checked-in fixtures and update only intended goldens.

Do not run the exported-design corpus as a ticket gate. Do not duplicate
omission decisions in JSON-RPC, change default admission, alter custom public Go
field types, or conflate semantic absence with arbitrary application IsZero.
The model assumes the field classifier is correct; direct/generated tests must
establish that correspondence. Full authored exclusion, arbitrary custom codec
output, incomplete example materialization and protocol envelopes remain separate
contracts; do not claim a generic Go-JSON refinement proof.

## Recovery and cleanup

The #844 cleanup receipt is `/tmp/loom-844-cleanup-receipt.json` (25 inactive paths;
source, exact patch, final manifest and compact comparisons retained). The prior
readiness plan is preserved at `/tmp/loom-m7-prior-readiness-plan.md`.
The new managed checkout exists at the path above; creation and attachment retry
hit the app's 100-attachment limit. Do not create another checkout to retry that.
The old `json-presence-omission` source worktree remains untouched.

Final readiness controls: `/tmp/loom-m7-final-controls.log` reruns the expanded
16-layout ledger and eight direct candidate/carrier examples, plus the nine
existing required/Optional/Nullable controls. The native authored-zero projection
witness fails all three scalar cases at 0.633s. Frozen source/input/evidence hashes
are recorded in `/tmp/loom-m7-presence-readiness-manifest.json`.

## Final acceptance and exact scope

Readiness was approved without findings by `review_checkpoint95`; the exact
report was approved by root and filed as #848 before production changes. The
final owner is `ResolveJSONField`; service declaration/composition use
`expr.ServiceFieldLayout`, while HTTP declarations and value projection use
`representation.ResolveField`. NameScope still renders type names. The old
HTTP `FieldOmission` and omission-option parsing policy have been removed.

Selected acceptance, Go 1.27.2 with JSON v2:

- `internal/encodingmeta`, `expr`, `codegen`, `codegen/service` full package tests
  pass (0.424s, 0.804s, 23.183s, 35.970s). Direct tests cover nil/empty/populated,
  native false/zero/empty, custom IsZero, full tags and both omission options.
- Focused HTTP tests pass (8.092s): all changed declaration/initializer goldens,
  mapped fields, selected-view requiredness, body/value-plan controls, real
  inline/union round trips and #849's anonymous type identity controls.
- Generated HTTP/RPC cases pass (15.458s), including native default zeros,
  recursive named collections, full/automatic tags, direct server responses,
  default admission, CLI null/required/malformed-body contracts and required views.
- Canonical Extend cases pass in both peer/local and forward/reverse order through
  an inherited intermediate type. Focused protobuf transforms/type files pass
  (0.586s); protobuf presence behavior is not changed.
- All three checked-in SSE fixtures were regenerated, tested and vetted. Only
  seven fixture files have #848-specific tag changes. Independent parent
  regeneration exposes one earlier stale CLI file, separately owned by #851.
- File-size, gofmt, JSON-v2 and NameScope lints pass. Existing file-size warnings
  remain below the hard cap. `git diff --check` passes.

The six bounded specimens produce 124 artifacts per generation: ordinary and
full authored tags, native/recursive storage, selected views, skip control and
nontrivial scalar-map OpenAPI design. All six parent and candidate modules build
and vet. Independent candidate output is byte-identical. Exactly 30 files change:

| Specimen | Files and intended differences |
| --- | --- |
| ordinary | `plain/service.go`, `rpc/service.go`, and HTTP plain / JSON-RPC rpc client+server `types.go`: ordinary nil-capable fields use absence omission |
| full | Same six files: the nested Child.note automatic field changes; full authored tags and custom time.Time stay intact |
| storage | Same six files: service native defaults retain zeros; recursive/named collection fields omit absence |
| views | `presence/service.go`, `presence/views/view.go`, HTTP presence client+server `types.go`: ordinary optional fields retain empties; view selection/requiredness are unchanged |
| spec | `flags/service.go`, `tagged/service.go`, HTTP flags server `types.go`, tagged client+server `types.go`, HTTP flags+tagged client `cli.go`, and `http/cli/scalar_map_keys/cli.go`: omission tags and retained empty CLI examples follow the shared owner |

The other 94 generated artifacts are unchanged. The independent JSON/YAML specs
are byte-identical and pass libopenapi parsing/building plus pinned Vacuum.
Their only parent/candidate differences are the retained `empty`, `labels` and
`names` members at both response-media and response-schema examples for
`POST /tagged`. Declared schema examples and request examples remain unchanged.

Compact evidence: `/tmp/loom-848-artifact-comparison.json`,
`/tmp/loom-848-generated.diff`, `/tmp/loom-848-spec-comparison.json`,
`/tmp/loom-848-fixture-checks.json` and the frozen source/evidence manifest
`/tmp/loom-848-final-manifest.json` with exact patch
`/tmp/loom-848-final.patch`. Model
sources are unchanged from independently reviewed readiness; reuse its 58-state
positive run and three negative counterexamples.

A broader HTTP package audit reached the aggregate ten-minute timeout and is
incomplete evidence; it is not claimed green. It also reproduced pre-existing
#850 diagnostic fixture assembly failures in HTTP/JSON-RPC. Those changes are
owned separately. #849's two pushed test files are imported unchanged as a local
validation prerequisite and excluded from #848's patch. #851's parent-generated
CLI alignment is likewise excluded; no generated output was hand-edited or
spliced. Fixture module/checksum files retain their original contents: tidy
proved generation-only dependencies were unrelated, recorded in
`/tmp/loom-848-fixture-manifest-receipt.json`.
